package api

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"iac-tool/internal/build"
	"iac-tool/internal/crypto"
	"iac-tool/internal/database"
	"iac-tool/internal/git"
	"iac-tool/internal/models"

	"github.com/gin-gonic/gin"
)

// ListDeployments lists all deployments
// GET /api/deployments
func ListDeployments(c *gin.Context) {
	rows, err := database.DB.Query(`
		SELECT d.id, d.namespace_id, d.name, d.description, d.git_url, d.created_at, d.updated_at, n.name as namespace
		FROM deployments d
		JOIN namespaces n ON d.namespace_id = n.id
		ORDER BY d.created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	deployments := make([]models.DeploymentWithNamespace, 0)
	for rows.Next() {
		var d models.DeploymentWithNamespace
		err := rows.Scan(&d.ID, &d.NamespaceID, &d.Name, &d.Description, &d.GitURL, &d.CreatedAt, &d.UpdatedAt, &d.Namespace)
		if err != nil {
			continue
		}
		deployments = append(deployments, d)
	}

	c.JSON(http.StatusOK, deployments)
}

// GetDeployment gets a deployment by ID
// GET /api/deployments/:id
func GetDeployment(c *gin.Context) {
	id := c.Param("id")

	var d models.DeploymentWithNamespace
	err := database.DB.QueryRow(`
		SELECT d.id, d.namespace_id, d.name, d.description, d.git_url, d.created_at, d.updated_at, n.name as namespace
		FROM deployments d
		JOIN namespaces n ON d.namespace_id = n.id
		WHERE d.id = ?
	`, id).Scan(&d.ID, &d.NamespaceID, &d.Name, &d.Description, &d.GitURL, &d.CreatedAt, &d.UpdatedAt, &d.Namespace)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Deployment not found"})
		return
	}

	c.JSON(http.StatusOK, d)
}

// CreateDeployment creates a new deployment
// POST /api/deployments
func CreateDeployment(c *gin.Context) {
	var input models.DeploymentCreate

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate Git URL format
	if !isValidGitURL(input.GitURL) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Git URL. Must be a valid git repository URL (https:// or git@)"})
		return
	}

	// Prepare auth config and encrypted data
	var authType sql.NullString
	var authData sql.NullString

	if input.IsPrivate && input.GitUsername != "" {
		// Encrypt authentication data
		authJSON := map[string]string{
			"username": input.GitUsername,
			"password": input.GitPassword,
		}
		authDataBytes, _ := json.Marshal(authJSON)
		encrypted, err := crypto.EncryptJSON(string(authDataBytes))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt authentication data"})
			return
		}

		authType = sql.NullString{String: "http", Valid: true}
		authData = sql.NullString{String: encrypted, Valid: true}
	}

	deploymentID := generateID()
	now := time.Now()

	_, err := database.DB.Exec(`
		INSERT INTO deployments (id, namespace_id, name, description, git_url, git_auth_type, git_auth_data, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, deploymentID, input.NamespaceID, input.Name, input.Description, input.GitURL, authType, authData, now, now)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			c.JSON(http.StatusConflict, gin.H{"error": "A deployment with this name already exists in this namespace"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// Get the created deployment with namespace
	var deployment models.DeploymentWithNamespace
	database.DB.QueryRow(`
		SELECT d.id, d.namespace_id, d.name, d.description, d.git_url, d.created_at, d.updated_at, n.name as namespace
		FROM deployments d
		JOIN namespaces n ON d.namespace_id = n.id
		WHERE d.id = ?
	`, deploymentID).Scan(&deployment.ID, &deployment.NamespaceID, &deployment.Name, &deployment.Description, &deployment.GitURL, &deployment.CreatedAt, &deployment.UpdatedAt, &deployment.Namespace)

	c.JSON(http.StatusCreated, deployment)
}

// DeleteDeployment deletes a deployment
// DELETE /api/deployments/:id
func DeleteDeployment(c *gin.Context) {
	id := c.Param("id")

	result, err := database.DB.Exec("DELETE FROM deployments WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if affected, _ := result.RowsAffected(); affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Deployment not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Deployment deleted"})
}

// GetDeploymentReferences gets branches and tags for a deployment repository
// GET /api/deployments/:id/references
func GetDeploymentReferences(c *gin.Context) {
	id := c.Param("id")

	// Get deployment and its git URL
	var gitURL string
	err := database.DB.QueryRow("SELECT git_url FROM deployments WHERE id = ?", id).Scan(&gitURL)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Deployment not found"})
		return
	}

	// Load auth config from database
	var auth *git.AuthConfig
	var authType sql.NullString
	var authDataStr sql.NullString
	err = database.DB.QueryRow("SELECT git_auth_type, git_auth_data FROM deployments WHERE id = ?", id).Scan(&authType, &authDataStr)
	if err == nil && authType.Valid && authDataStr.Valid {
		// Decrypt auth data
		decryptedData, err := crypto.DecryptJSON(authDataStr.String)
		if err == nil {
			// Parse auth data
			var authJSON map[string]string
			if err := json.Unmarshal([]byte(decryptedData), &authJSON); err == nil {
				auth = &git.AuthConfig{
					Type:     authType.String,
					Username: authJSON["username"],
					Password: authJSON["password"],
				}
			}
		}
	}

	// Get branches
	branches, err := git.ListBranches(gitURL, auth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list branches: " + err.Error()})
		return
	}

	// Get tags
	tags, err := git.ListTagNames(gitURL, auth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tags: " + err.Error()})
		return
	}

	// Build response
	references := gin.H{
		"branches": branches,
		"tags":     tags,
	}

	c.JSON(http.StatusOK, references)
}

// GetDeploymentDirectory lists directory contents
// GET /api/deployments/:id/browse?ref=main&path=/
func GetDeploymentDirectory(c *gin.Context) {
	id := c.Param("id")
	ref := c.Query("ref")
	path := c.Query("path")

	if ref == "" {
		ref = "HEAD"
	}
	if path == "" {
		path = ""
	}

	// Get deployment
	var gitURL string
	err := database.DB.QueryRow("SELECT git_url FROM deployments WHERE id = ?", id).Scan(&gitURL)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Deployment not found"})
		return
	}

	// Load auth config
	var auth *git.AuthConfig
	var authType sql.NullString
	var authDataStr sql.NullString
	err = database.DB.QueryRow("SELECT git_auth_type, git_auth_data FROM deployments WHERE id = ?", id).Scan(&authType, &authDataStr)
	if err == nil && authType.Valid && authDataStr.Valid {
		decryptedData, err := crypto.DecryptJSON(authDataStr.String)
		if err == nil {
			var authJSON map[string]string
			if err := json.Unmarshal([]byte(decryptedData), &authJSON); err == nil {
				auth = &git.AuthConfig{
					Type:     authType.String,
					Username: authJSON["username"],
					Password: authJSON["password"],
				}
			}
		}
	}

	// List directory
	files, readme, err := git.ListDirectory(gitURL, ref, path, auth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list directory: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"path":   path,
		"files":  files,
		"readme": readme,
	})
}

// CreateDeploymentRun creates a new deployment run
// POST /api/deployments/:id/runs
func CreateDeploymentRun(c *gin.Context) {
	id := c.Param("id")
	var input models.DeploymentRunCreate

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input.DeploymentID = id

	// Validate tool
	if input.Tool != "terraform" && input.Tool != "tofu" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tool must be 'terraform' or 'tofu'"})
		return
	}

	// Verify deployment exists
	var gitURL string
	err := database.DB.QueryRow("SELECT git_url FROM deployments WHERE id = ?", id).Scan(&gitURL)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Deployment not found"})
		return
	}

	runID := generateID()
	now := time.Now()

	// Serialize env vars to JSON
	envVarsJSON, _ := json.Marshal(input.EnvVars)

	_, err = database.DB.Exec(`
		INSERT INTO deployment_runs (id, deployment_id, path, ref, tool, env_vars, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, 'pending', ?)
	`, runID, input.DeploymentID, input.Path, input.Ref, input.Tool, string(envVarsJSON), now)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get the created run
	run, err := getDeploymentRun(runID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Start the deployment asynchronously
	go build.ExecuteDeploymentRun(runID, id, input.Path, input.Ref, input.Tool, input.EnvVars)

	c.JSON(http.StatusCreated, run)
}

// ListDeploymentRuns lists all runs for a deployment
// GET /api/deployments/:id/runs?path=/optional/path
func ListDeploymentRuns(c *gin.Context) {
	id := c.Param("id")
	path := c.Query("path")

	var query string
	var args []interface{}

	if path != "" {
		query = `
			SELECT id FROM deployment_runs
			WHERE deployment_id = ? AND path = ?
			ORDER BY created_at DESC
		`
		args = []interface{}{id, path}
	} else {
		query = `
			SELECT id FROM deployment_runs
			WHERE deployment_id = ?
			ORDER BY created_at DESC
		`
		args = []interface{}{id}
	}

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	runs := make([]models.DeploymentRun, 0)
	for rows.Next() {
		var runID string
		if err := rows.Scan(&runID); err != nil {
			continue
		}
		run, err := getDeploymentRun(runID)
		if err != nil {
			continue
		}
		runs = append(runs, *run)
	}

	c.JSON(http.StatusOK, runs)
}

// GetDirectoryStatus gets the status of the last deployment run for a directory
// GET /api/deployments/:id/status?path=/optional/path
func GetDirectoryStatus(c *gin.Context) {
	id := c.Param("id")
	path := c.Query("path")

	if path == "" {
		path = ""
	}

	var status models.DirectoryStatus
	status.Path = path
	status.Status = "none"
	status.StatusColor = "blue"

	// Get last run for this path
	var run models.DeploymentRun
	err := database.DB.QueryRow(`
		SELECT id, deployment_id, path, ref, status, error_message, created_at, started_at, completed_at
		FROM deployment_runs
		WHERE deployment_id = ? AND path = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, id, path).Scan(&run.ID, &run.DeploymentID, &run.Path, &run.Ref, &run.Status, &run.ErrorMessage, &run.CreatedAt, &run.StartedAt, &run.CompletedAt)

	if err == nil {
		status.LastRun = &run
		status.Status = run.Status

		switch run.Status {
		case "success":
			status.StatusColor = "green"
		case "running":
			status.StatusColor = "yellow"
		case "failed":
			status.StatusColor = "red"
		default:
			status.StatusColor = "blue"
		}
	}

	c.JSON(http.StatusOK, status)
}

// GetDeploymentRun gets a single deployment run by ID
// GET /api/deployments/:id/runs/:runId
func GetDeploymentRun(c *gin.Context) {
	runID := c.Param("runId")

	run, err := getDeploymentRun(runID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Run not found"})
		return
	}

	c.JSON(http.StatusOK, run)
}

// ApproveDeploymentRun approves or rejects a deployment run
// POST /api/deployments/:id/runs/:runId/approve
func ApproveDeploymentRun(c *gin.Context) {
	runID := c.Param("runId")
	var input models.DeploymentRunApproval

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check that run is in awaiting_approval state
	var status string
	err := database.DB.QueryRow(`SELECT status FROM deployment_runs WHERE id = ?`, runID).Scan(&status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Run not found"})
		return
	}

	if status != "awaiting_approval" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Run is not awaiting approval"})
		return
	}

	// Update approval status
	approvedBy := input.ApprovedBy
	if !input.Approved {
		approvedBy = "REJECTED"
	}

	now := time.Now()
	_, err = database.DB.Exec(`
		UPDATE deployment_runs 
		SET approved_by = ?, approved_at = ?
		WHERE id = ?
	`, approvedBy, now, runID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	run, _ := getDeploymentRun(runID)
	c.JSON(http.StatusOK, run)
}

// CancelDeploymentRun cancels a running deployment
// POST /api/deployments/:id/runs/:runId/cancel
func CancelDeploymentRun(c *gin.Context) {
	runID := c.Param("runId")

	// Get the runner deployment ID
	var workDir sql.NullString
	var status string
	err := database.DB.QueryRow(`SELECT work_dir, status FROM deployment_runs WHERE id = ?`, runID).Scan(&workDir, &status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Run not found"})
		return
	}

	// Check if run can be cancelled
	cancellableStatuses := []string{"pending", "initializing", "planning", "awaiting_approval", "applying"}
	canCancel := false
	for _, s := range cancellableStatuses {
		if status == s {
			canCancel = true
			break
		}
	}

	if !canCancel {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Run cannot be cancelled in current state: " + status})
		return
	}

	// Send cancel request to runner if it has started
	if workDir.Valid && workDir.String != "" {
		runnerURL := os.Getenv("RUNNER_URL")
		if runnerURL == "" {
			runnerURL = "http://runner:8080"
		}

		resp, err := http.Post(runnerURL+"/deploy/"+workDir.String+"/cancel", "application/json", nil)
		if err == nil {
			defer resp.Body.Close()
		}
	}

	// Update database
	now := time.Now()
	result, err := database.DB.Exec(`
		UPDATE deployment_runs 
		SET status = 'cancelled', error_message = 'Cancelled by user', completed_at = ?
		WHERE id = ?
	`, now, runID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update run status"})
		return
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Run not updated - may already be completed"})
		return
	}

	run, _ := getDeploymentRun(runID)
	c.JSON(http.StatusOK, run)
}

// getDeploymentRun is a helper to fetch a deployment run with all fields
func getDeploymentRun(runID string) (*models.DeploymentRun, error) {
	var run models.DeploymentRun
	var envVarsJSON, initLog, planLog, applyLog, workDir, approvedBy sql.NullString

	err := database.DB.QueryRow(`
		SELECT id, deployment_id, path, ref, tool, env_vars, status, 
		       init_log, plan_log, apply_log, error_message, work_dir,
		       approved_by, approved_at, created_at, started_at, completed_at
		FROM deployment_runs
		WHERE id = ?
	`, runID).Scan(
		&run.ID, &run.DeploymentID, &run.Path, &run.Ref, &run.Tool,
		&envVarsJSON, &run.Status, &initLog, &planLog, &applyLog,
		&run.ErrorMessage, &workDir, &approvedBy, &run.ApprovedAt,
		&run.CreatedAt, &run.StartedAt, &run.CompletedAt,
	)

	if err != nil {
		return nil, err
	}

	// Parse env vars
	if envVarsJSON.Valid && envVarsJSON.String != "" {
		json.Unmarshal([]byte(envVarsJSON.String), &run.EnvVars)
	} else {
		run.EnvVars = make(map[string]string)
	}

	// Set nullable strings
	if initLog.Valid {
		run.InitLog = initLog.String
	}
	if planLog.Valid {
		run.PlanLog = planLog.String
	}
	if applyLog.Valid {
		run.ApplyLog = applyLog.String
	}
	if workDir.Valid {
		run.WorkDir = workDir.String
	}
	if approvedBy.Valid {
		run.ApprovedBy = &approvedBy.String
	}

	return &run, nil
}

// StreamDeploymentRunLogs streams real-time logs from the runner
// GET /api/deployments/:id/runs/:runId/stream
func StreamDeploymentRunLogs(c *gin.Context) {
	runID := c.Param("runId")

	// Get the runner deployment ID (stored in work_dir)
	var workDir sql.NullString
	err := database.DB.QueryRow(`SELECT work_dir FROM deployment_runs WHERE id = ?`, runID).Scan(&workDir)
	if err != nil || !workDir.Valid {
		c.JSON(http.StatusNotFound, gin.H{"error": "Run not found or not started"})
		return
	}

	runnerDeploymentID := workDir.String

	// Get runner URL
	runnerURL := os.Getenv("RUNNER_URL")
	if runnerURL == "" {
		runnerURL = "http://runner:8080"
	}

	// Proxy the SSE stream from runner
	resp, err := http.Get(runnerURL + "/deploy/" + runnerDeploymentID + "/logs")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to runner"})
		return
	}
	defer resp.Body.Close()

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Get the response writer
	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	// Proxy the stream line by line
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			w.Write(line)
			flusher.Flush()
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("Stream error: %v", err)
			}
			break
		}
	}
}

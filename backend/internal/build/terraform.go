package build

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"iac-tool/internal/crypto"
	"iac-tool/internal/database"
	"io"
	"net/http"
	"os"
	"time"
)

// RunnerDeploymentRequest matches the runner's DeploymentRequest
type RunnerDeploymentRequest struct {
	Tool        string            `json:"tool"`
	GitURL      string            `json:"git_url"`
	GitRef      string            `json:"git_ref"`
	Path        string            `json:"path"`
	EnvVars     map[string]string `json:"env_vars"`
	Timeout     int               `json:"timeout"`
	GitAuth     *RunnerGitAuth    `json:"git_auth,omitempty"`
	AutoApprove bool              `json:"auto_approve"`
}

type RunnerGitAuth struct {
	Type     string `json:"type"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type RunnerDeploymentResponse struct {
	DeploymentID string `json:"deployment_id"`
	Status       string `json:"status"`
	Message      string `json:"message,omitempty"`
}

type RunnerDeploymentStatus struct {
	DeploymentID string     `json:"deployment_id"`
	Status       string     `json:"status"`
	Phase        string     `json:"phase"`
	StartedAt    time.Time  `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
	Error        string     `json:"error,omitempty"`
	InitLog      string     `json:"init_log,omitempty"`
	PlanLog      string     `json:"plan_log,omitempty"`
	ApplyLog     string     `json:"apply_log,omitempty"`
}

// ExecuteDeploymentRun executes a deployment run via the runner HTTP API
func ExecuteDeploymentRun(runID, deploymentID, path, ref, tool string, envVars map[string]string) {
	// Mark as initializing
	now := time.Now()
	database.DB.Exec(`
UPDATE deployment_runs
SET status = 'initializing', started_at = ?
WHERE id = ?
`, now, runID)

	// Get deployment info
	var gitURL string
	var authType, authDataStr sql.NullString
	err := database.DB.QueryRow(`
SELECT git_url, git_auth_type, git_auth_data 
FROM deployments 
WHERE id = ?
`, deploymentID).Scan(&gitURL, &authType, &authDataStr)
	if err != nil {
		failRun(runID, "Failed to get deployment info: "+err.Error())
		return
	}

	// Prepare git auth
	var gitAuth *RunnerGitAuth
	if authType.Valid && authDataStr.Valid {
		decryptedData, err := crypto.DecryptJSON(authDataStr.String)
		if err == nil {
			var authJSON map[string]string
			if err := json.Unmarshal([]byte(decryptedData), &authJSON); err == nil {
				gitAuth = &RunnerGitAuth{
					Type:     authType.String,
					Username: authJSON["username"],
					Password: authJSON["password"],
				}
			}
		}
	}

	// Create deployment request
	runnerReq := RunnerDeploymentRequest{
		Tool:        tool,
		GitURL:      gitURL,
		GitRef:      ref,
		Path:        path,
		EnvVars:     envVars,
		Timeout:     60,
		GitAuth:     gitAuth,
		AutoApprove: false, // Manual approval required
	}

	// Get runner URL from environment
	runnerURL := os.Getenv("RUNNER_URL")
	if runnerURL == "" {
		runnerURL = "http://runner:8080"
	}

	// Start deployment on runner
	reqBody, _ := json.Marshal(runnerReq)
	resp, err := http.Post(runnerURL+"/deploy", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		failRun(runID, "Failed to contact runner: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 202 {
		body, _ := io.ReadAll(resp.Body)
		failRun(runID, fmt.Sprintf("Runner returned error: %s", string(body)))
		return
	}

	var deployResp RunnerDeploymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&deployResp); err != nil {
		failRun(runID, "Failed to parse runner response: "+err.Error())
		return
	}

	runnerDeploymentID := deployResp.DeploymentID

	// Store runner deployment ID
	database.DB.Exec(`UPDATE deployment_runs SET work_dir = ? WHERE id = ?`, runnerDeploymentID, runID)

	// Poll runner for status updates
	pollRunnerStatus(runID, runnerDeploymentID, runnerURL)
}

func pollRunnerStatus(runID, runnerDeploymentID, runnerURL string) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(2 * time.Hour)
	firstUpdate := true

	for {
		select {
		case <-ticker.C:
			// Get status from runner
			resp, err := http.Get(fmt.Sprintf("%s/deploy/%s/status", runnerURL, runnerDeploymentID))
			if err != nil {
				continue
			}

			var status RunnerDeploymentStatus
			if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
				resp.Body.Close()
				continue
			}
			resp.Body.Close()

			// Update started_at on first update
			if firstUpdate && status.StartedAt.Unix() > 0 {
				database.DB.Exec(`UPDATE deployment_runs SET started_at = ? WHERE id = ?`, status.StartedAt, runID)
				firstUpdate = false
			}

			// Update database
			database.DB.Exec(`
			UPDATE deployment_runs 
			SET status = ?, init_log = ?, plan_log = ?, apply_log = ?
			WHERE id = ?
		`, status.Phase, status.InitLog, status.PlanLog, status.ApplyLog, runID)

			// Check if waiting for approval
			if status.Status == "awaiting_approval" {
				database.DB.Exec(`UPDATE deployment_runs SET status = 'awaiting_approval' WHERE id = ?`, runID)

				// Wait for approval in database
				if !waitForApproval(runID, runnerDeploymentID, runnerURL, 24*time.Hour) {
					// Rejected - continue polling to get the final status from runner
					time.Sleep(1 * time.Second)
					continue
				}
				// Approved - continue polling for apply phase
				continue
			}

			// Check if deployment finished
			if status.Status == "success" {
				database.DB.Exec(`
				UPDATE deployment_runs 
				SET status = 'success', completed_at = ? 
				WHERE id = ?
			`, time.Now(), runID)
				return
			}

			if status.Status == "failed" || status.Status == "cancelled" {
				database.DB.Exec(`
				UPDATE deployment_runs 
				SET status = ?, error_message = ?, completed_at = ? 
				WHERE id = ?
			`, status.Status, status.Error, time.Now(), runID)
				return
			}

		case <-timeout:
			failRun(runID, "Deployment timeout")
			return
		}
	}
}

func waitForApproval(runID, runnerDeploymentID, runnerURL string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		var approvedBy sql.NullString
		err := database.DB.QueryRow(`
SELECT approved_by FROM deployment_runs WHERE id = ?
`, runID).Scan(&approvedBy)

		if err == nil && approvedBy.Valid {
			if approvedBy.String == "REJECTED" {
				// Send rejection to runner (runner will update status to cancelled)
				http.Post(fmt.Sprintf("%s/deploy/%s/reject", runnerURL, runnerDeploymentID), "application/json", nil)
				return false
			} else {
				// Send approval to runner
				http.Post(fmt.Sprintf("%s/deploy/%s/approve", runnerURL, runnerDeploymentID), "application/json", nil)
				database.DB.Exec(`UPDATE deployment_runs SET status = 'applying' WHERE id = ?`, runID)
				return true
			}
		}

		time.Sleep(2 * time.Second)
	}

	return false
}

func failRun(runID, errorMsg string) {
	database.DB.Exec(`
UPDATE deployment_runs 
SET status = 'failed', error_message = ?, completed_at = ? 
WHERE id = ?
`, errorMsg, time.Now(), runID)
}

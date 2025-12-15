package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

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

		authType = sql.NullString{String: "https", Valid: true}
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

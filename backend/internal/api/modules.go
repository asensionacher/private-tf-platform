package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"iac-tool/internal/crypto"
	"iac-tool/internal/database"
	"iac-tool/internal/git"
	"iac-tool/internal/models"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// Terraform Module Registry Protocol Endpoints
// Docs: https://www.terraform.io/internals/module-registry-protocol
// ============================================================================

// TFListModuleVersions lists available versions for a specific module
// GET /v1/modules/:namespace/:name/:provider/versions
func TFListModuleVersions(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	provider := c.Param("provider")

	// Get module
	var moduleID string
	err := database.DB.QueryRow(`
		SELECT m.id FROM modules m
		JOIN namespaces n ON m.namespace_id = n.id
		WHERE n.name = $1 AND m.name = $2 AND m.provider = $3
	`, namespace, name, provider).Scan(&moduleID)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"errors": "no module found with given arguments (source " + namespace + "/" + provider + "/" + name + ")",
		})
		return
	}

	// Get versions (only enabled ones for Terraform)
	rows, err := database.DB.Query(`
		SELECT version FROM module_versions
		WHERE module_id = $1 AND enabled = TRUE
		ORDER BY COALESCE(tag_date, created_at) DESC
	`, moduleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"errors": []string{err.Error()}})
		return
	}
	defer rows.Close()

	versions := make([]models.ModuleVersionDTO, 0)
	for rows.Next() {
		var v models.ModuleVersionDTO
		if err := rows.Scan(&v.Version); err != nil {
			continue
		}
		versions = append(versions, v)
	}

	// Return in Terraform protocol format
	response := models.ModuleVersionsResponse{
		Modules: []models.ModuleVersionsDTO{
			{Versions: versions},
		},
	}

	c.JSON(http.StatusOK, response)
}

// TFDownloadModule returns the download URL for a specific module version
// GET /v1/modules/:namespace/:name/:provider/:version/download
func TFDownloadModule(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	provider := c.Param("provider")
	version := c.Param("version")

	// Get download URL (only if version is enabled)
	var downloadURL string
	var enabled bool
	err := database.DB.QueryRow(`
		SELECT mv.download_url, mv.enabled FROM module_versions mv
		JOIN modules m ON mv.module_id = m.id
		JOIN namespaces n ON m.namespace_id = n.id
		WHERE n.name = $1 AND m.name = $2 AND m.provider = $3 AND mv.version = $4
	`, namespace, name, provider, version).Scan(&downloadURL, &enabled)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"errors": []string{"Module version not found"},
		})
		return
	}

	if !enabled {
		c.JSON(http.StatusNotFound, gin.H{
			"errors": []string{"Module version is not available"},
		})
		return
	}

	// Return download URL in X-Terraform-Get header
	c.Header("X-Terraform-Get", downloadURL)
	c.Status(http.StatusNoContent)
}

// ============================================================================
// REST API Endpoints for Module Management
// ============================================================================

// GetModules returns all modules (admin API)
func GetModules(c *gin.Context) {
	// Get optional namespace filter
	namespaceFilter := c.Query("namespace")

	query := `
		SELECT m.id, m.namespace_id, m.name, m.provider, m.description, m.source_url, 
			   m.synced, m.sync_error, m.created_at, m.updated_at, n.name as namespace
		FROM modules m
		JOIN namespaces n ON m.namespace_id = n.id
	`
	args := []interface{}{}

	if namespaceFilter != "" {
		query += " WHERE n.name = $1"
		args = append(args, namespaceFilter)
	}

	query += " ORDER BY n.name, m.name, m.provider"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"errors": []string{err.Error()}})
		return
	}
	defer rows.Close()

	modules := make([]models.ModuleWithNamespace, 0)
	for rows.Next() {
		var mod models.ModuleWithNamespace
		if err := rows.Scan(&mod.ID, &mod.NamespaceID, &mod.Name, &mod.Provider, &mod.Description,
			&mod.SourceURL, &mod.Synced, &mod.SyncError, &mod.CreatedAt, &mod.UpdatedAt, &mod.Namespace); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"errors": []string{err.Error()}})
			return
		}
		modules = append(modules, mod)
	}

	c.JSON(http.StatusOK, modules)
}

// GetModule returns a single module with its versions
func GetModule(c *gin.Context) {
	id := c.Param("id")

	var mod models.ModuleWithNamespace
	err := database.DB.QueryRow(`
		SELECT m.id, m.namespace_id, m.name, m.provider, m.description, m.source_url, 
			   m.synced, m.sync_error, m.created_at, m.updated_at, n.name as namespace
		FROM modules m
		JOIN namespaces n ON m.namespace_id = n.id
		WHERE m.id = $1
	`, id).Scan(&mod.ID, &mod.NamespaceID, &mod.Name, &mod.Provider, &mod.Description,
		&mod.SourceURL, &mod.Synced, &mod.SyncError, &mod.CreatedAt, &mod.UpdatedAt, &mod.Namespace)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"errors": []string{"Module not found"}})
		return
	}

	c.JSON(http.StatusOK, mod)
}

// CreateModule creates a new module (via API)
// POST /api/modules/:namespace/:name/:provider
func CreateModule(c *gin.Context) {
	// Get namespace from path or from authenticated context
	namespace := c.Param("namespace")
	name := c.Param("name")
	provider := c.Param("provider")

	// If no namespace in path, use from auth context
	if namespace == "" {
		if nsName, exists := c.Get("namespace_name"); exists {
			namespace = nsName.(string)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"errors": []string{"Namespace required"}})
			return
		}
	}

	var input struct {
		Description *string `json:"description,omitempty"`
		SourceURL   *string `json:"source_url,omitempty"`
	}
	c.ShouldBindJSON(&input)

	// Get namespace ID
	var namespaceID string
	err := database.DB.QueryRow("SELECT id FROM namespaces WHERE name = $1", namespace).Scan(&namespaceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"errors": []string{"Namespace not found"}})
		return
	}

	id := generateID()
	now := time.Now()

	_, err = database.DB.Exec(`
		INSERT INTO modules (id, namespace_id, name, provider, description, source_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, id, namespaceID, name, provider, input.Description, input.SourceURL, now, now)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			c.JSON(http.StatusConflict, gin.H{"errors": []string{"Module already exists"}})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"errors": []string{err.Error()}})
		}
		return
	}

	mod := models.ModuleWithNamespace{
		Module: models.Module{
			ID:          id,
			NamespaceID: namespaceID,
			Name:        name,
			Provider:    provider,
			Description: input.Description,
			SourceURL:   input.SourceURL,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		Namespace: namespace,
	}

	c.JSON(http.StatusCreated, mod)
}

// UploadModuleVersion uploads a new version of a module
// POST /api/modules/:name/:provider/:version/upload
func UploadModuleVersion(c *gin.Context) {
	name := c.Param("name")
	provider := c.Param("provider")
	version := c.Param("version")

	// Get namespace from auth context
	namespace, exists := c.Get("namespace_name")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"errors": []string{"API key required"}})
		return
	}

	var input struct {
		DownloadURL   string            `json:"download_url" binding:"required"`
		Documentation *string           `json:"documentation,omitempty"`
		Headers       map[string]string `json:"headers,omitempty"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"errors": []string{err.Error()}})
		return
	}

	// Get or create module
	var moduleID string
	err := database.DB.QueryRow(`
		SELECT m.id FROM modules m
		JOIN namespaces n ON m.namespace_id = n.id
		WHERE n.name = $1 AND m.name = $2 AND m.provider = $3
	`, namespace, name, provider).Scan(&moduleID)

	if err != nil {
		// Create module first
		var namespaceID string
		err := database.DB.QueryRow("SELECT id FROM namespaces WHERE name = $1", namespace).Scan(&namespaceID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"errors": []string{"Namespace not found"}})
			return
		}

		moduleID = generateID()
		now := time.Now()
		_, err = database.DB.Exec(`
			INSERT INTO modules (id, namespace_id, name, provider, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, moduleID, namespaceID, name, provider, now, now)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"errors": []string{err.Error()}})
			return
		}
	}

	// Create version
	versionID := generateID()
	now := time.Now()

	_, err = database.DB.Exec(`
		INSERT INTO module_versions (id, module_id, version, download_url, documentation, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, versionID, moduleID, version, input.DownloadURL, input.Documentation, now)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			c.JSON(http.StatusConflict, gin.H{"errors": []string{"Version already exists"}})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"errors": []string{err.Error()}})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"errors": []string{}})
}

// GetModuleVersions returns all versions of a module
func GetModuleVersions(c *gin.Context) {
	id := c.Param("id")

	rows, err := database.DB.Query(`
		SELECT id, version, download_url, documentation, enabled, tag_date, created_at
		FROM module_versions
		WHERE module_id = $1
		ORDER BY COALESCE(tag_date, created_at) DESC
	`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"errors": []string{err.Error()}})
		return
	}
	defer rows.Close()

	versions := make([]models.ModuleVersion, 0)
	for rows.Next() {
		var v models.ModuleVersion
		var tagDateStr sql.NullString
		if err := rows.Scan(&v.ID, &v.Version, &v.DownloadURL, &v.Documentation, &v.Enabled, &tagDateStr, &v.CreatedAt); err != nil {
			log.Printf("Error scanning module version: %v", err)
			continue
		}
		if tagDateStr.Valid && tagDateStr.String != "" {
			if t, err := time.Parse(time.RFC3339, tagDateStr.String); err == nil {
				v.TagDate = &t
			}
		}
		v.ModuleID = id
		versions = append(versions, v)
	}

	c.JSON(http.StatusOK, versions)
}

// DeleteModule deletes a module and all its versions
// DELETE /api/modules/:name/:provider/remove
func DeleteModule(c *gin.Context) {
	name := c.Param("name")
	provider := c.Param("provider")

	// Get namespace from auth context
	namespace, exists := c.Get("namespace_name")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"errors": []string{"API key required"}})
		return
	}

	result, err := database.DB.Exec(`
		DELETE FROM modules
		WHERE id IN (
			SELECT m.id FROM modules m
			JOIN namespaces n ON m.namespace_id = n.id
			WHERE n.name = $1 AND m.name = $2 AND m.provider = $3
		)
	`, namespace, name, provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"errors": []string{err.Error()}})
		return
	}

	if affected, _ := result.RowsAffected(); affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"errors": []string{"Module not found"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"errors": []string{}})
}

// DeleteModuleVersion deletes a specific version of a module
// DELETE /api/modules/:name/:provider/:version/remove
func DeleteModuleVersion(c *gin.Context) {
	name := c.Param("name")
	provider := c.Param("provider")
	version := c.Param("version")

	// Get namespace from auth context
	namespace, exists := c.Get("namespace_name")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"errors": []string{"API key required"}})
		return
	}

	result, err := database.DB.Exec(`
		DELETE FROM module_versions
		WHERE id IN (
			SELECT mv.id FROM module_versions mv
			JOIN modules m ON mv.module_id = m.id
			JOIN namespaces n ON m.namespace_id = n.id
			WHERE n.name = $1 AND m.name = $2 AND m.provider = $3 AND mv.version = $4
		)
	`, namespace, name, provider, version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"errors": []string{err.Error()}})
		return
	}

	if affected, _ := result.RowsAffected(); affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"errors": []string{"Module version not found"}})
		return
	}

	// Check if module has no more versions and delete it
	var count int
	database.DB.QueryRow(`
		SELECT COUNT(*) FROM module_versions mv
		JOIN modules m ON mv.module_id = m.id
		JOIN namespaces n ON m.namespace_id = n.id
		WHERE n.name = $1 AND m.name = $2 AND m.provider = $3
	`, namespace, name, provider).Scan(&count)

	if count == 0 {
		database.DB.Exec(`
			DELETE FROM modules
			WHERE id IN (
				SELECT m.id FROM modules m
				JOIN namespaces n ON m.namespace_id = n.id
				WHERE n.name = $1 AND m.name = $2 AND m.provider = $3
			)
		`, namespace, name, provider)
	}

	c.JSON(http.StatusOK, gin.H{"errors": []string{}})
}

// UpdateModule updates a module
func UpdateModule(c *gin.Context) {
	id := c.Param("id")

	var input models.ModuleUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"errors": []string{err.Error()}})
		return
	}

	// Build update query dynamically
	query := "UPDATE modules SET updated_at = $1"
	args := []interface{}{time.Now()}

	if input.Name != nil {
		query += ", name = ?"
		args = append(args, *input.Name)
	}
	if input.Provider != nil {
		query += ", provider = ?"
		args = append(args, *input.Provider)
	}
	if input.Description != nil {
		query += ", description = ?"
		args = append(args, *input.Description)
	}
	if input.SourceURL != nil {
		query += ", source_url = ?"
		args = append(args, *input.SourceURL)
	}

	query += " WHERE id = $1"
	args = append(args, id)

	result, err := database.DB.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"errors": []string{err.Error()}})
		return
	}

	if affected, _ := result.RowsAffected(); affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"errors": []string{"Module not found"}})
		return
	}

	GetModule(c)
}

// ============================================================================
// Git-based Module Creation
// ============================================================================

// CreateModuleFromGit creates a module from a Git repository URL
// POST /api/modules
func CreateModuleFromGit(c *gin.Context) {
	var input struct {
		NamespaceID string  `json:"namespace_id" binding:"required"`
		Name        string  `json:"name" binding:"required"`
		Provider    string  `json:"provider" binding:"required"`
		GitURL      string  `json:"git_url" binding:"required"`
		Description *string `json:"description,omitempty"`
		Subdir      *string `json:"subdir,omitempty"` // Subdirectory in repo containing the module
		IsPrivate   bool    `json:"is_private,omitempty"`
		GitUsername string  `json:"git_username,omitempty"` // For HTTPS authentication
		GitPassword string  `json:"git_password,omitempty"` // Personal Access Token
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate Git URL format
	if !isValidGitURL(input.GitURL) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Git URL. Must be a valid HTTPS git repository URL (e.g., https://github.com/org/repo.git)"})
		return
	}

	// Prepare auth config if repository is private (HTTPS only)
	var authConfig *git.AuthConfig
	var authData string
	if input.IsPrivate && input.GitUsername != "" {
		authConfig = &git.AuthConfig{
			Type:     "https",
			Username: input.GitUsername,
			Password: input.GitPassword,
		}
		// Store auth data as encrypted JSON
		authJSON := map[string]string{
			"username": input.GitUsername,
			"password": input.GitPassword,
		}
		authDataBytes, _ := json.Marshal(authJSON)

		// Encrypt the auth data before storing
		encrypted, err := crypto.EncryptJSON(string(authDataBytes))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt authentication data: " + err.Error()})
			return
		}
		authData = encrypted
	}

	// Verify the repository exists and is accessible (skip validation for now if private with auth)
	if !input.IsPrivate {
		if err := git.ValidateGitRepository(input.GitURL); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Git repository validation failed: " + err.Error()})
			return
		}
	}

	// Verify namespace exists
	var namespaceName string
	err := database.DB.QueryRow("SELECT name FROM namespaces WHERE id = $1", input.NamespaceID).Scan(&namespaceName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Namespace not found"})
		return
	}

	// Check if module already exists
	var existingID string
	err = database.DB.QueryRow(`
		SELECT id FROM modules 
		WHERE namespace_id = $1 AND name = $2 AND provider = $3
	`, input.NamespaceID, input.Name, input.Provider).Scan(&existingID)

	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Module already exists"})
		return
	}

	now := time.Now()
	moduleID := generateID()

	// Store subdir in a metadata field or as part of source_url
	sourceURL := input.GitURL
	if input.Subdir != nil && *input.Subdir != "" {
		sourceURL = input.GitURL + "//" + *input.Subdir
	}

	_, err = database.DB.Exec(`
		INSERT INTO modules (id, namespace_id, name, provider, description, source_url, synced, git_auth_type, git_auth_data, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, FALSE, $7, $8, $9, $10)
	`, moduleID, input.NamespaceID, input.Name, input.Provider, input.Description, sourceURL, sql.NullString{String: "https", Valid: input.IsPrivate && input.GitUsername != ""}, authData, now, now)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Automatically sync tags in background
	go func() {
		syncModuleTagsBackgroundWithAuth(moduleID, input.GitURL, input.Subdir, authConfig)
	}()

	response := models.ModuleWithNamespace{
		Module: models.Module{
			ID:          moduleID,
			NamespaceID: input.NamespaceID,
			Name:        input.Name,
			Provider:    input.Provider,
			Description: input.Description,
			SourceURL:   &sourceURL,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		Namespace: namespaceName,
	}

	c.JSON(http.StatusCreated, response)
}

// SyncModuleTags fetches tags from the Git repository and syncs them with module versions
// POST /api/modules/:id/sync-tags
func SyncModuleTags(c *gin.Context) {
	moduleID := c.Param("id")

	// Get module and its source URL
	var sourceURL string
	err := database.DB.QueryRow("SELECT source_url FROM modules WHERE id = $1", moduleID).Scan(&sourceURL)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Module not found"})
		return
	}

	// Parse source URL to get git URL
	gitURL, subdir := parseSourceURL(sourceURL)

	// Load auth config from database
	var auth *git.AuthConfig
	var authType sql.NullString
	var authData sql.NullString
	err = database.DB.QueryRow("SELECT git_auth_type, git_auth_data FROM modules WHERE id = $1", moduleID).Scan(&authType, &authData)
	if err == nil && authType.Valid && authData.Valid {
		// Decrypt auth data
		decryptedData, err := crypto.DecryptJSON(authData.String)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decrypt authentication data"})
			return
		}

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

	// Fetch tags from Git with authentication
	tags, err := git.GetTagsWithAuth(gitURL, auth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tags: " + err.Error()})
		return
	}

	now := time.Now()
	addedCount := 0

	// Add each tag as a version (if not exists)
	for _, tag := range tags {
		var existingID string
		err := database.DB.QueryRow(`
			SELECT id FROM module_versions WHERE module_id = $1 AND version = $2
		`, moduleID, tag.Version).Scan(&existingID)

		if err != nil { // Version doesn't exist, create it
			versionID := generateID()
			downloadURL := buildGitDownloadURL(gitURL, tag.Name, subdir)

			var tagDate interface{}
			if !tag.TagDate.IsZero() {
				tagDate = tag.TagDate
			}

			_, err = database.DB.Exec(`
				INSERT INTO module_versions (id, module_id, version, download_url, enabled, tag_date, created_at)
				VALUES ($1, $2, $3, $4, FALSE, $5, $6)
			`, versionID, moduleID, tag.Version, downloadURL, tagDate, now)

			if err == nil {
				addedCount++
			}
		}
	}

	// Update module: mark as synced and clear sync errors
	database.DB.Exec("UPDATE modules SET updated_at = $1, synced = TRUE, sync_error = NULL WHERE id = $2", now, moduleID)

	c.JSON(http.StatusOK, gin.H{
		"message":    "Tags synced successfully",
		"tags_found": len(tags),
		"tags_added": addedCount,
	})
}

// syncModuleTagsBackground syncs tags in the background (wrapper for backward compatibility)
func syncModuleTagsBackground(moduleID string, gitURL string, subdir *string) {
	syncModuleTagsBackgroundWithAuth(moduleID, gitURL, subdir, nil)
}

// syncModuleTagsBackgroundWithAuth syncs tags in the background with authentication
func syncModuleTagsBackgroundWithAuth(moduleID string, gitURL string, subdir *string, auth *git.AuthConfig) {
	log.Printf("Starting background tag sync for module %s", moduleID)

	// Defer recovery to handle panics
	defer func() {
		if r := recover(); r != nil {
			errorMsg := fmt.Sprintf("Panic during sync: %v", r)
			log.Printf("Module %s sync panic: %v", moduleID, r)
			database.DB.Exec("UPDATE modules SET synced = TRUE, sync_error = $1, updated_at = $2 WHERE id = $3",
				errorMsg, time.Now(), moduleID)
		}
	}()

	// Load auth config from database if not provided
	if auth == nil {
		var authType sql.NullString
		var authData sql.NullString
		err := database.DB.QueryRow("SELECT git_auth_type, git_auth_data FROM modules WHERE id = $1", moduleID).Scan(&authType, &authData)
		if err == nil && authType.Valid && authData.Valid {
			// Decrypt auth data
			decryptedData, err := crypto.DecryptJSON(authData.String)
			if err != nil {
				errorMsg := fmt.Sprintf("Failed to decrypt authentication data: %v", err)
				log.Printf("Module %s decrypt error: %v", moduleID, err)
				database.DB.Exec("UPDATE modules SET synced = TRUE, sync_error = $1, updated_at = $2 WHERE id = $3",
					errorMsg, time.Now(), moduleID)
				return
			}

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

	tags, err := git.GetTagsWithAuth(gitURL, auth)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to fetch tags: %v", err)
		log.Printf("Failed to fetch tags for module %s: %v", moduleID, err)
		// Mark as synced but with error so it stops trying automatically
		database.DB.Exec("UPDATE modules SET synced = TRUE, sync_error = $1, updated_at = $2 WHERE id = $3",
			errorMsg, time.Now(), moduleID)
		return
	}

	if len(tags) == 0 {
		errorMsg := "No valid version tags found in repository"
		log.Printf("No tags found for module %s", moduleID)
		database.DB.Exec("UPDATE modules SET synced = TRUE, sync_error = $1, updated_at = $2 WHERE id = $3",
			errorMsg, time.Now(), moduleID)
		return
	}

	now := time.Now()
	addedCount := 0

	for _, tag := range tags {
		var existingID string
		err := database.DB.QueryRow(`
			SELECT id FROM module_versions WHERE module_id = $1 AND version = $2
		`, moduleID, tag.Version).Scan(&existingID)

		if err != nil {
			versionID := generateID()
			downloadURL := buildGitDownloadURL(gitURL, tag.Name, subdir)

			var tagDate interface{}
			if !tag.TagDate.IsZero() {
				tagDate = tag.TagDate
			}

			_, err = database.DB.Exec(`
				INSERT INTO module_versions (id, module_id, version, download_url, enabled, tag_date, created_at)
				VALUES ($1, $2, $3, $4, FALSE, $5, $6)
			`, versionID, moduleID, tag.Version, downloadURL, tagDate, now)

			if err == nil {
				addedCount++
			}
		}
	}

	// Update module: mark as synced and clear any previous errors
	database.DB.Exec("UPDATE modules SET updated_at = $1, synced = TRUE, sync_error = NULL WHERE id = $2", now, moduleID)

	log.Printf("Background tag sync completed for module %s: %d tags found, %d added", moduleID, len(tags), addedCount)
}

// GetModuleGitTags fetches available tags from the Git repository
// GET /api/modules/:id/git-tags
func GetModuleGitTags(c *gin.Context) {
	moduleID := c.Param("id")

	// Get module and its source URL
	var sourceURL string
	err := database.DB.QueryRow("SELECT source_url FROM modules WHERE id = $1", moduleID).Scan(&sourceURL)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Module not found"})
		return
	}

	// Parse source URL to get git URL
	gitURL, _ := parseSourceURL(sourceURL)

	// Fetch tags from Git
	tags, err := git.GetTags(gitURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tags: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, tags)
}

// ToggleModuleVersion enables or disables a module version
// PATCH /api/modules/:id/versions/:versionId
func ToggleModuleVersion(c *gin.Context) {
	moduleID := c.Param("id")
	versionID := c.Param("versionId")

	var input struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := database.DB.Exec(`
		UPDATE module_versions SET enabled = $1 WHERE id = $2 AND module_id = $3
	`, input.Enabled, versionID, moduleID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if affected, _ := result.RowsAffected(); affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}

	// Update module updated_at
	database.DB.Exec("UPDATE modules SET updated_at = $1 WHERE id = $2", time.Now(), moduleID)

	c.JSON(http.StatusOK, gin.H{"message": "Version updated", "enabled": input.Enabled})
}

// GetModuleReadme fetches the README.md from the module's Git repository
// GET /api/modules/:id/readme
func GetModuleReadme(c *gin.Context) {
	moduleID := c.Param("id")
	ref := c.Query("ref") // Optional: specific version/tag

	// Get module source URL
	var sourceURL string
	err := database.DB.QueryRow("SELECT source_url FROM modules WHERE id = $1", moduleID).Scan(&sourceURL)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Module not found"})
		return
	}

	// Parse source URL to get git URL
	gitURL, _ := parseSourceURL(sourceURL)

	// Load auth config from database
	var auth *git.AuthConfig
	var authType sql.NullString
	var authData sql.NullString
	err = database.DB.QueryRow("SELECT git_auth_type, git_auth_data FROM modules WHERE id = $1", moduleID).Scan(&authType, &authData)
	if err == nil && authType.Valid && authData.Valid {
		// Decrypt auth data
		decryptedData, err := crypto.DecryptJSON(authData.String)
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

	// Fetch README with authentication
	readme, err := git.GetReadmeWithAuth(gitURL, ref, auth)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "README not found: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"content": readme})
}

// AddModuleVersion adds a new version to an existing module (kept for manual addition)
// POST /api/modules/:id/versions
func AddModuleVersion(c *gin.Context) {
	moduleID := c.Param("id")

	var input struct {
		Version string  `json:"version" binding:"required"`
		Enabled bool    `json:"enabled"`
		Subdir  *string `json:"subdir,omitempty"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate version format
	if !isValidVersion(input.Version) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version format. Use semantic versioning (e.g., 1.0.0)"})
		return
	}

	// Get module and its source URL
	var sourceURL string
	err := database.DB.QueryRow("SELECT source_url FROM modules WHERE id = $1", moduleID).Scan(&sourceURL)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Module not found"})
		return
	}

	// Check if version already exists
	var existingVersionID string
	err = database.DB.QueryRow(`
		SELECT id FROM module_versions WHERE module_id = $1 AND version = $2
	`, moduleID, input.Version).Scan(&existingVersionID)

	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Version " + input.Version + " already exists"})
		return
	}

	// Parse source URL to get git URL and subdir
	gitURL, defaultSubdir := parseSourceURL(sourceURL)
	subdir := defaultSubdir
	if input.Subdir != nil {
		subdir = input.Subdir
	}

	// Build download URL
	downloadURL := buildGitDownloadURL(gitURL, input.Version, subdir)

	// Create version
	versionID := generateID()
	now := time.Now()

	_, err = database.DB.Exec(`
		INSERT INTO module_versions (id, module_id, version, download_url, enabled, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, versionID, moduleID, input.Version, downloadURL, input.Enabled, now)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update module updated_at
	database.DB.Exec("UPDATE modules SET updated_at = $1 WHERE id = $2", now, moduleID)

	version := models.ModuleVersion{
		ID:          versionID,
		ModuleID:    moduleID,
		Version:     input.Version,
		DownloadURL: downloadURL,
		Enabled:     input.Enabled,
		CreatedAt:   now,
	}

	c.JSON(http.StatusCreated, version)
}

// DeleteModuleVersionByID deletes a specific version of a module by version ID
// DELETE /api/modules/:id/versions/:versionId
func DeleteModuleVersionByID(c *gin.Context) {
	moduleID := c.Param("id")
	versionID := c.Param("versionId")

	result, err := database.DB.Exec(`
		DELETE FROM module_versions WHERE id = $1 AND module_id = $2
	`, versionID, moduleID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if affected, _ := result.RowsAffected(); affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Version deleted"})
}

// DeleteModuleByID deletes a module and all its versions
// DELETE /api/modules/:id
func DeleteModuleByID(c *gin.Context) {
	moduleID := c.Param("id")

	// First delete all versions
	_, err := database.DB.Exec("DELETE FROM module_versions WHERE module_id = $1", moduleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Then delete the module
	result, err := database.DB.Exec("DELETE FROM modules WHERE id = $1", moduleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if affected, _ := result.RowsAffected(); affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Module not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Module deleted"})
}

// parseSourceURL extracts git URL and subdir from source_url
func parseSourceURL(sourceURL string) (string, *string) {
	// Format: https://github.com/org/repo.git//subdir
	// We need to find "//" that comes after the domain (not in https://)
	// Look for "//" that is not part of the protocol

	// Skip the protocol (https://, git://, etc.)
	protocolEnd := strings.Index(sourceURL, "://")
	if protocolEnd == -1 {
		return sourceURL, nil
	}

	// Look for "//" after the protocol
	searchStart := protocolEnd + 3 // skip "://"
	restOfURL := sourceURL[searchStart:]
	if idx := strings.Index(restOfURL, "//"); idx != -1 {
		gitURL := sourceURL[:searchStart+idx]
		subdir := restOfURL[idx+2:]
		return gitURL, &subdir
	}
	return sourceURL, nil
}

// isValidVersion validates that the version follows semver-like format
func isValidVersion(version string) bool {
	// Simple validation: must contain at least one dot and numbers
	if len(version) == 0 {
		return false
	}
	// Allow versions like 1.0.0, v1.0.0, 1.0, 0.1.0-beta, etc.
	for _, c := range version {
		if c != '.' && c != '-' && c != '+' && c != 'v' && (c < '0' || c > '9') && (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
			return false
		}
	}
	return strings.Contains(version, ".")
}

// buildGitDownloadURL builds a Terraform-compatible Git download URL
func buildGitDownloadURL(gitURL, tagName string, subdir *string) string {
	// Terraform expects: git::https://example.com/repo.git?ref=<tag>
	// or with subdir: git::https://example.com/repo.git//subdir?ref=<tag>
	// Use the original tag name (e.g., "v1.0.0" or "0.4.2") as-is

	result := "git::" + gitURL

	// Add subdir if specified
	if subdir != nil && *subdir != "" {
		result += "//" + *subdir
	}

	result += "?ref=" + tagName

	return result
}

package api

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"iac-tool/internal/crypto"
	"iac-tool/internal/database"
	"iac-tool/internal/git"
	"iac-tool/internal/gpg"
	"iac-tool/internal/models"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// Terraform Provider Registry Protocol Endpoints
// Docs: https://www.terraform.io/internals/provider-registry-protocol
// ============================================================================

// TFListProviderVersions lists available versions for a specific provider
// GET /v1/providers/:namespace/:name/versions
func TFListProviderVersions(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	// Get provider
	var providerID string
	err := database.DB.QueryRow(`
		SELECT p.id FROM providers p
		JOIN namespaces n ON p.namespace_id = n.id
		WHERE n.name = $1 AND p.name = $2
	`, namespace, name).Scan(&providerID)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"errors": []string{"Provider not found"},
		})
		return
	}

	// Get versions with platforms
	rows, err := database.DB.Query(`
		SELECT pv.id, pv.version, pv.protocols
		FROM provider_versions pv
		WHERE pv.provider_id = $1
		ORDER BY COALESCE(pv.tag_date, pv.created_at) DESC
	`, providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"errors": []string{err.Error()}})
		return
	}
	defer rows.Close()

	versions := make([]models.ProviderVersionDTO, 0)
	for rows.Next() {
		var v models.ProviderVersionDTO
		var versionID string
		var protocolsJSON string
		if err := rows.Scan(&versionID, &v.Version, &protocolsJSON); err != nil {
			continue
		}

		// Parse protocols
		if protocolsJSON != "" {
			json.Unmarshal([]byte(protocolsJSON), &v.Protocols)
		}
		if v.Protocols == nil {
			v.Protocols = []string{"5.0"}
		}

		// Get platforms
		platformRows, _ := database.DB.Query(`
			SELECT os, arch FROM provider_platforms WHERE version_id = $1
		`, versionID)
		v.Platforms = make([]models.ProviderPlatformDTO, 0)
		for platformRows.Next() {
			var p models.ProviderPlatformDTO
			if err := platformRows.Scan(&p.OS, &p.Arch); err == nil {
				v.Platforms = append(v.Platforms, p)
			}
		}
		platformRows.Close()

		versions = append(versions, v)
	}

	c.JSON(http.StatusOK, models.ProviderVersionsResponse{Versions: versions})
}

// TFDownloadProvider returns download info for a specific provider version and platform
// GET /v1/providers/:namespace/:name/:version/download/:os/:arch
func TFDownloadProvider(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	version := c.Param("version")
	osParam := c.Param("os")
	arch := c.Param("arch")

	var pp models.ProviderPlatform
	var protocolsJSON string
	var shasumURL, shasumSigURL, dbSigningKeys sql.NullString
	err := database.DB.QueryRow(`
		SELECT pp.filename, pp.download_url, pp.shasums_url, pp.shasums_signature_url, 
			   pp.shasum, pp.signing_keys, pv.protocols
		FROM provider_platforms pp
		JOIN provider_versions pv ON pp.version_id = pv.id
		JOIN providers p ON pv.provider_id = p.id
		JOIN namespaces n ON p.namespace_id = n.id
		WHERE n.name = $1 AND p.name = $2 AND pv.version = $3 AND pp.os = $4 AND pp.arch = $5
		  AND pv.enabled = true
	`, namespace, name, version, osParam, arch).Scan(
		&pp.Filename, &pp.DownloadURL, &shasumURL, &shasumSigURL,
		&pp.SHASum, &dbSigningKeys, &protocolsJSON)

	if err != nil {
		log.Printf("TFDownloadProvider error: namespace=%s name=%s version=%s os=%s arch=%s err=%v",
			namespace, name, version, osParam, arch, err)
		c.JSON(http.StatusNotFound, gin.H{
			"errors": []string{"Provider version not found for this platform"},
		})
		return
	}

	if shasumURL.Valid {
		pp.SHASumsURL = shasumURL.String
	}
	if shasumSigURL.Valid {
		pp.SHASumsSignature = shasumSigURL.String
	}
	if dbSigningKeys.Valid {
		pp.SigningKeys = dbSigningKeys.String
	}

	var protocols []string
	if protocolsJSON != "" {
		json.Unmarshal([]byte(protocolsJSON), &protocols)
	}
	if protocols == nil {
		protocols = []string{"5.0"}
	}

	// Build download URL dynamically from request headers
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		host := c.GetHeader("X-Forwarded-Host")
		if host == "" {
			host = c.GetHeader("Host")
		}
		if host == "" {
			backendHost := os.Getenv("BACKEND_HOST")
			if backendHost == "" {
				backendHost = "localhost"
			}
			backendPort := os.Getenv("PORT")
			if backendPort == "" {
				backendPort = "9080"
			}
			host = backendHost + ":" + backendPort
		}
		scheme := c.GetHeader("X-Forwarded-Proto")
		if scheme == "" {
			scheme = "http"
		}
		baseURL = scheme + "://" + host
	}
	downloadURL := baseURL + "/downloads/providers/" + namespace + "/" + name + "/" + version + "/" + pp.Filename

	// Build signing keys from GPG
	var signingKeys *models.SigningKeys
	if gpg.GetKeyID() != "" {
		signingKeys = &models.SigningKeys{
			GPGPublicKeys: []models.GPGPublicKey{
				{
					KeyID:      gpg.GetKeyID(),
					ASCIIArmor: gpg.GetPublicKey(),
				},
			},
		}
	} else {
		signingKeys = &models.SigningKeys{GPGPublicKeys: []models.GPGPublicKey{}}
	}

	response := models.ProviderDownloadResponse{
		Protocols:           protocols,
		OS:                  osParam,
		Arch:                arch,
		Filename:            pp.Filename,
		DownloadURL:         downloadURL,
		SHASumsURL:          baseURL + "/shasums/providers/" + namespace + "/" + name + "/" + version,
		SHASumsSignatureURL: baseURL + "/shasums/providers/" + namespace + "/" + name + "/" + version + "/sig",
		SHASum:              pp.SHASum,
		SigningKeys:         signingKeys,
	}

	c.JSON(http.StatusOK, response)
}

// GetProviderSHASums returns SHA256SUMS file for a provider version
// GET /shasums/providers/:namespace/:name/:version
func GetProviderSHASums(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	version := c.Param("version")

	// Get all platforms for this version
	rows, err := database.DB.Query(`
		SELECT pp.filename, pp.shasum
		FROM provider_platforms pp
		JOIN provider_versions pv ON pp.version_id = pv.id
		JOIN providers p ON pv.provider_id = p.id
		JOIN namespaces n ON p.namespace_id = n.id
		WHERE n.name = $1 AND p.name = $2 AND pv.version = $3 AND pv.enabled = true
	`, namespace, name, version)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}
	defer rows.Close()

	var shasums strings.Builder
	for rows.Next() {
		var filename, shasum string
		if err := rows.Scan(&filename, &shasum); err == nil {
			shasums.WriteString(shasum + "  " + filename + "\n")
		}
	}

	if shasums.Len() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No platforms found"})
		return
	}

	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, shasums.String())
}

// GetProviderSHASumsSig returns GPG signature for SHA256SUMS file
// GET /shasums/providers/:namespace/:name/:version/sig
func GetProviderSHASumsSig(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	version := c.Param("version")

	// Get all platforms for this version to build shasums
	rows, err := database.DB.Query(`
		SELECT pp.filename, pp.shasum
		FROM provider_platforms pp
		JOIN provider_versions pv ON pp.version_id = pv.id
		JOIN providers p ON pv.provider_id = p.id
		JOIN namespaces n ON p.namespace_id = n.id
		WHERE n.name = $1 AND p.name = $2 AND pv.version = $3 AND pv.enabled = true
	`, namespace, name, version)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}
	defer rows.Close()

	var shasums strings.Builder
	for rows.Next() {
		var filename, shasum string
		if err := rows.Scan(&filename, &shasum); err == nil {
			shasums.WriteString(shasum + "  " + filename + "\n")
		}
	}

	if shasums.Len() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No platforms found"})
		return
	}

	// Sign the shasums content
	signature, err := gpg.Sign(shasums.String())
	if err != nil {
		log.Printf("Failed to sign SHA256SUMS: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sign"})
		return
	}

	c.Data(http.StatusOK, "application/octet-stream", signature)
}

// ============================================================================
// REST API Endpoints for Provider Management
// ============================================================================

// GetProviders returns all providers
func GetProviders(c *gin.Context) {
	namespaceFilter := c.Query("namespace")

	query := `
		SELECT p.id, p.namespace_id, p.name, p.description, p.synced, p.created_at, p.updated_at, 
			   n.name as namespace
		FROM providers p
		JOIN namespaces n ON p.namespace_id = n.id
	`
	args := []interface{}{}

	if namespaceFilter != "" {
		query += " WHERE n.name = $1"
		args = append(args, namespaceFilter)
	}

	query += " ORDER BY n.name, p.name"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"errors": []string{err.Error()}})
		return
	}
	defer rows.Close()

	providers := make([]models.ProviderWithNamespace, 0)
	for rows.Next() {
		var p models.ProviderWithNamespace
		if err := rows.Scan(&p.ID, &p.NamespaceID, &p.Name, &p.Description, &p.Synced,
			&p.CreatedAt, &p.UpdatedAt, &p.Namespace); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"errors": []string{err.Error()}})
			return
		}
		providers = append(providers, p)
	}

	c.JSON(http.StatusOK, providers)
}

// GetProvider returns a single provider
func GetProvider(c *gin.Context) {
	id := c.Param("id")

	var p models.ProviderWithNamespace
	err := database.DB.QueryRow(`
		SELECT p.id, p.namespace_id, p.name, p.description, p.synced, p.created_at, p.updated_at,
			   n.name as namespace
		FROM providers p
		JOIN namespaces n ON p.namespace_id = n.id
		WHERE p.id = $1
	`, id).Scan(&p.ID, &p.NamespaceID, &p.Name, &p.Description, &p.Synced,
		&p.CreatedAt, &p.UpdatedAt, &p.Namespace)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"errors": []string{"Provider not found"}})
		return
	}

	c.JSON(http.StatusOK, p)
}

// UploadProviderVersion uploads a new version of a provider
// POST /api/providers/:name/:version/upload
func UploadProviderVersion(c *gin.Context) {
	name := c.Param("name")
	version := c.Param("version")

	// Get namespace from auth context
	namespace, exists := c.Get("namespace_name")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"errors": []string{"API key required"}})
		return
	}

	var input struct {
		Protocols []string                        `json:"protocols"`
		Platforms []models.ProviderPlatformCreate `json:"platforms" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"errors": []string{err.Error()}})
		return
	}

	if input.Protocols == nil {
		input.Protocols = []string{"5.0"}
	}

	// Get or create provider
	var providerID string
	err := database.DB.QueryRow(`
		SELECT p.id FROM providers p
		JOIN namespaces n ON p.namespace_id = n.id
		WHERE n.name = $1 AND p.name = $2
	`, namespace, name).Scan(&providerID)

	if err != nil {
		// Create provider first
		var namespaceID string
		err := database.DB.QueryRow("SELECT id FROM namespaces WHERE name = $1", namespace).Scan(&namespaceID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"errors": []string{"Namespace not found"}})
			return
		}

		providerID = generateID()
		now := time.Now()
		_, err = database.DB.Exec(`
			INSERT INTO providers (id, namespace_id, name, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5)
		`, providerID, namespaceID, name, now, now)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"errors": []string{err.Error()}})
			return
		}
	}

	// Create version
	versionID := generateID()
	protocolsJSON, _ := json.Marshal(input.Protocols)
	now := time.Now()

	_, err = database.DB.Exec(`
		INSERT INTO provider_versions (id, provider_id, version, protocols, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, versionID, providerID, version, string(protocolsJSON), now)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			c.JSON(http.StatusConflict, gin.H{"errors": []string{"Version already exists"}})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"errors": []string{err.Error()}})
		}
		return
	}

	// Create platforms
	for _, platform := range input.Platforms {
		platformID := generateID()
		signingKeysJSON := ""
		if platform.SigningKeys != "" {
			signingKeysJSON = platform.SigningKeys
		}

		_, err = database.DB.Exec(`
			INSERT INTO provider_platforms (id, version_id, os, arch, filename, download_url, 
				shasums_url, shasums_signature_url, shasum, signing_keys)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, platformID, versionID, platform.OS, platform.Arch, platform.Filename,
			platform.DownloadURL, platform.SHASumsURL, platform.SHASumsSignature,
			platform.SHASum, signingKeysJSON)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"errors": []string{err.Error()}})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"errors": []string{}})
}

// GetProviderVersions returns all versions of a provider with their platforms
func GetProviderVersions(c *gin.Context) {
	id := c.Param("id")

	rows, err := database.DB.Query(`
		SELECT id, version, protocols, COALESCE(enabled, TRUE) as enabled, tag_date, created_at
		FROM provider_versions
		WHERE provider_id = $1
		ORDER BY COALESCE(tag_date, created_at) DESC
	`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"errors": []string{err.Error()}})
		return
	}
	defer rows.Close()

	versions := make([]models.ProviderVersion, 0)
	for rows.Next() {
		var v models.ProviderVersion
		var protocolsJSON string
		var tagDateStr sql.NullString
		if err := rows.Scan(&v.ID, &v.Version, &protocolsJSON, &v.Enabled, &tagDateStr, &v.CreatedAt); err != nil {
			log.Printf("Error scanning provider version: %v", err)
			continue
		}
		if tagDateStr.Valid && tagDateStr.String != "" {
			if t, err := time.Parse(time.RFC3339, tagDateStr.String); err == nil {
				v.TagDate = &t
			}
		}
		if protocolsJSON != "" {
			json.Unmarshal([]byte(protocolsJSON), &v.Protocols)
		}
		v.ProviderID = id

		// Get platforms for this version
		platformRows, err := database.DB.Query(`
			SELECT id, os, arch, filename, shasum, download_url
			FROM provider_platforms
			WHERE version_id = $1
		`, v.ID)
		if err == nil {
			v.Platforms = make([]models.ProviderPlatform, 0)
			for platformRows.Next() {
				var p models.ProviderPlatform
				if err := platformRows.Scan(&p.ID, &p.OS, &p.Arch, &p.Filename, &p.SHASum, &p.DownloadURL); err == nil {
					p.VersionID = v.ID
					v.Platforms = append(v.Platforms, p)
				}
			}
			platformRows.Close()
		}

		versions = append(versions, v)
	}

	c.JSON(http.StatusOK, versions)
}

// DeleteProvider deletes a provider and all its versions
// DELETE /api/providers/:name/remove
func DeleteProvider(c *gin.Context) {
	name := c.Param("name")

	// Get namespace from auth context
	namespace, exists := c.Get("namespace_name")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"errors": []string{"API key required"}})
		return
	}

	result, err := database.DB.Exec(`
		DELETE FROM providers
		WHERE id IN (
			SELECT p.id FROM providers p
			JOIN namespaces n ON p.namespace_id = n.id
			WHERE n.name = $1 AND p.name = $2
		)
	`, namespace, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"errors": []string{err.Error()}})
		return
	}

	if affected, _ := result.RowsAffected(); affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"errors": []string{"Provider not found"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"errors": []string{}})
}

// DeleteProviderVersion deletes a specific version of a provider
// DELETE /api/providers/:name/:version/remove
func DeleteProviderVersion(c *gin.Context) {
	name := c.Param("name")
	version := c.Param("version")

	// Get namespace from auth context
	namespace, exists := c.Get("namespace_name")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"errors": []string{"API key required"}})
		return
	}

	result, err := database.DB.Exec(`
		DELETE FROM provider_versions
		WHERE id IN (
			SELECT pv.id FROM provider_versions pv
			JOIN providers p ON pv.provider_id = p.id
			JOIN namespaces n ON p.namespace_id = n.id
			WHERE n.name = $1 AND p.name = $2 AND pv.version = $3
		)
	`, namespace, name, version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"errors": []string{err.Error()}})
		return
	}

	if affected, _ := result.RowsAffected(); affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"errors": []string{"Provider version not found"}})
		return
	}

	// Check if provider has no more versions and delete it
	var count int
	database.DB.QueryRow(`
		SELECT COUNT(*) FROM provider_versions pv
		JOIN providers p ON pv.provider_id = p.id
		JOIN namespaces n ON p.namespace_id = n.id
		WHERE n.name = $1 AND p.name = $2
	`, namespace, name).Scan(&count)

	if count == 0 {
		database.DB.Exec(`
			DELETE FROM providers
			WHERE id IN (
				SELECT p.id FROM providers p
				JOIN namespaces n ON p.namespace_id = n.id
				WHERE n.name = $1 AND p.name = $2
			)
		`, namespace, name)
	}

	c.JSON(http.StatusOK, gin.H{"errors": []string{}})
}

// ============================================================================
// Git-based Provider Creation (similar to modules)
// ============================================================================

// CreateProviderFromGit creates a provider from a Git repository URL
// POST /api/providers
func CreateProviderFromGit(c *gin.Context) {
	var input struct {
		NamespaceID string  `json:"namespace_id" binding:"required"`
		Name        string  `json:"name" binding:"required"`
		GitURL      string  `json:"git_url" binding:"required"`
		Description *string `json:"description,omitempty"`
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

	// Verify the repository exists and is accessible (skip validation for private repos with auth)
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

	// Check if provider already exists
	var existingID string
	err = database.DB.QueryRow(`
		SELECT id FROM providers 
		WHERE namespace_id = $1 AND name = $2
	`, input.NamespaceID, input.Name).Scan(&existingID)

	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Provider already exists"})
		return
	}

	now := time.Now()
	providerID := generateID()

	_, err = database.DB.Exec(`
		INSERT INTO providers (id, namespace_id, name, description, source_url, synced, git_auth_type, git_auth_data, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, FALSE, $6, $7, $8, $9)
	`, providerID, input.NamespaceID, input.Name, input.Description, input.GitURL, sql.NullString{String: "https", Valid: input.IsPrivate && input.GitUsername != ""}, authData, now, now)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Automatically sync tags and generate documentation
	go func() {
		syncProviderTagsBackgroundWithAuth(providerID, input.GitURL, authConfig)
	}()

	response := models.ProviderWithNamespace{
		Provider: models.Provider{
			ID:          providerID,
			NamespaceID: input.NamespaceID,
			Name:        input.Name,
			Description: input.Description,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		Namespace: namespaceName,
	}

	c.JSON(http.StatusCreated, response)
}

// DeleteProviderByID deletes a provider and all its versions
// DELETE /api/providers/:id
func DeleteProviderByID(c *gin.Context) {
	providerID := c.Param("id")

	// First delete all platforms for all versions
	_, err := database.DB.Exec(`
		DELETE FROM provider_platforms 
		WHERE version_id IN (SELECT id FROM provider_versions WHERE provider_id = $1)
	`, providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Then delete all versions
	_, err = database.DB.Exec("DELETE FROM provider_versions WHERE provider_id = $1", providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Then delete the provider
	result, err := database.DB.Exec("DELETE FROM providers WHERE id = $1", providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if affected, _ := result.RowsAffected(); affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Provider deleted"})
}

// AddProviderVersion adds a new version to an existing provider
// POST /api/providers/:id/versions
func AddProviderVersion(c *gin.Context) {
	providerID := c.Param("id")

	var input struct {
		Version   string   `json:"version" binding:"required"`
		Protocols []string `json:"protocols,omitempty"`
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

	// Verify provider exists
	var exists int
	err := database.DB.QueryRow("SELECT 1 FROM providers WHERE id = $1", providerID).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	// Check if version already exists
	var existingVersionID string
	err = database.DB.QueryRow(`
		SELECT id FROM provider_versions WHERE provider_id = $1 AND version = $2
	`, providerID, input.Version).Scan(&existingVersionID)

	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Version " + input.Version + " already exists"})
		return
	}

	if input.Protocols == nil {
		input.Protocols = []string{"5.0"}
	}

	// Create version
	versionID := generateID()
	protocolsJSON, _ := json.Marshal(input.Protocols)
	now := time.Now()

	_, err = database.DB.Exec(`
		INSERT INTO provider_versions (id, provider_id, version, protocols, enabled, created_at)
		VALUES ($1, $2, $3, $4, TRUE, $5)
	`, versionID, providerID, input.Version, string(protocolsJSON), now)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update provider updated_at
	database.DB.Exec("UPDATE providers SET updated_at = $1 WHERE id = $2", now, providerID)

	version := models.ProviderVersion{
		ID:         versionID,
		ProviderID: providerID,
		Version:    input.Version,
		Protocols:  input.Protocols,
		Enabled:    true,
		CreatedAt:  now,
	}

	c.JSON(http.StatusCreated, version)
}

// DeleteProviderVersionByID deletes a specific version of a provider by version ID
// DELETE /api/providers/:id/versions/:versionId
func DeleteProviderVersionByID(c *gin.Context) {
	providerID := c.Param("id")
	versionID := c.Param("versionId")

	// First delete all platforms for this version
	_, err := database.DB.Exec("DELETE FROM provider_platforms WHERE version_id = $1", versionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result, err := database.DB.Exec(`
		DELETE FROM provider_versions WHERE id = $1 AND provider_id = $2
	`, versionID, providerID)

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

// ============================================================================
// Git-based Provider Management (similar to modules)
// ============================================================================

// syncProviderTagsBackground syncs tags in the background
// syncProviderTagsBackground syncs tags in the background (wrapper for backward compatibility)
func syncProviderTagsBackground(providerID string, sourceURL string) {
	syncProviderTagsBackgroundWithAuth(providerID, sourceURL, nil)
}

// syncProviderTagsBackgroundWithAuth syncs tags in the background with authentication
func syncProviderTagsBackgroundWithAuth(providerID string, sourceURL string, auth *git.AuthConfig) {
	log.Printf("Starting background tag sync for provider %s", providerID)

	// Defer recovery to handle panics
	defer func() {
		if r := recover(); r != nil {
			errorMsg := fmt.Sprintf("Panic during sync: %v", r)
			log.Printf("Provider %s sync panic: %v", providerID, r)
			database.DB.Exec("UPDATE providers SET synced = TRUE, sync_error = $1, updated_at = $2 WHERE id = $3",
				errorMsg, time.Now(), providerID)
		}
	}()

	// Load auth config from database if not provided
	if auth == nil {
		var authType sql.NullString
		var authData sql.NullString
		err := database.DB.QueryRow("SELECT git_auth_type, git_auth_data FROM providers WHERE id = $1", providerID).Scan(&authType, &authData)
		if err == nil && authType.Valid && authData.Valid {
			// Decrypt auth data
			decryptedData, err := crypto.DecryptJSON(authData.String)
			if err != nil {
				errorMsg := fmt.Sprintf("Failed to decrypt authentication data: %v", err)
				log.Printf("Provider %s decrypt error: %v", providerID, err)
				database.DB.Exec("UPDATE providers SET synced = TRUE, sync_error = $1, updated_at = $2 WHERE id = $3",
					errorMsg, time.Now(), providerID)
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

	// Fetch tags from Git
	tags, err := git.GetTagsWithAuth(sourceURL, auth)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to fetch tags: %v", err)
		log.Printf("Failed to fetch tags for provider %s: %v", providerID, err)
		database.DB.Exec("UPDATE providers SET synced = TRUE, sync_error = $1, updated_at = $2 WHERE id = $3",
			errorMsg, time.Now(), providerID)
		return
	}

	if len(tags) == 0 {
		errorMsg := "No valid version tags found in repository"
		log.Printf("No tags found for provider %s", providerID)
		database.DB.Exec("UPDATE providers SET synced = TRUE, sync_error = $1, updated_at = $2 WHERE id = $3",
			errorMsg, time.Now(), providerID)
		return
	}

	now := time.Now()
	addedCount := 0

	// Add each tag as a version (if not exists)
	for _, tag := range tags {
		var existingID string
		err := database.DB.QueryRow(`
			SELECT id FROM provider_versions WHERE provider_id = $1 AND version = $2
		`, providerID, tag.Version).Scan(&existingID)

		if err != nil { // Version doesn't exist, create it
			versionID := generateID()
			protocolsJSON, _ := json.Marshal([]string{"5.0"})

			var tagDate interface{}
			if !tag.TagDate.IsZero() {
				tagDate = tag.TagDate
			}

			_, err = database.DB.Exec(`
				INSERT INTO provider_versions (id, provider_id, version, protocols, enabled, tag_date, created_at)
				VALUES ($1, $2, $3, $4, FALSE, $5, $6)
			`, versionID, providerID, tag.Version, string(protocolsJSON), tagDate, now)

			if err == nil {
				addedCount++
			}
		}
	}

	// Update provider updated_at, mark as synced and clear errors
	database.DB.Exec("UPDATE providers SET updated_at = $1, synced = TRUE, sync_error = NULL WHERE id = $2", now, providerID)

	log.Printf("Background tag sync completed for provider %s: %d tags found, %d added", providerID, len(tags), addedCount)
}

// SyncProviderTags fetches tags from the Git repository and syncs them with provider versions
// POST /api/providers/:id/sync-tags
func SyncProviderTags(c *gin.Context) {
	providerID := c.Param("id")

	// Get provider and its source URL
	var sourceURL *string
	err := database.DB.QueryRow("SELECT source_url FROM providers WHERE id = $1", providerID).Scan(&sourceURL)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	if sourceURL == nil || *sourceURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provider has no Git source URL"})
		return
	}

	// Load auth config from database
	var auth *git.AuthConfig
	var authType sql.NullString
	var authData sql.NullString
	err = database.DB.QueryRow("SELECT git_auth_type, git_auth_data FROM providers WHERE id = $1", providerID).Scan(&authType, &authData)
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
	tags, err := git.GetTagsWithAuth(*sourceURL, auth)
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
			SELECT id FROM provider_versions WHERE provider_id = $1 AND version = $2
		`, providerID, tag.Version).Scan(&existingID)

		if err != nil { // Version doesn't exist, create it
			versionID := generateID()
			protocolsJSON, _ := json.Marshal([]string{"5.0"})

			var tagDate interface{}
			if !tag.TagDate.IsZero() {
				tagDate = tag.TagDate
			}

			_, err = database.DB.Exec(`
				INSERT INTO provider_versions (id, provider_id, version, protocols, enabled, tag_date, created_at)
				VALUES ($1, $2, $3, $4, FALSE, $5, $6)
			`, versionID, providerID, tag.Version, string(protocolsJSON), tagDate, now)

			if err == nil {
				addedCount++
			}
		}
	}

	// Update provider updated_at
	database.DB.Exec("UPDATE providers SET updated_at = $1 WHERE id = $2", now, providerID)

	c.JSON(http.StatusOK, gin.H{
		"message":    "Tags synced successfully",
		"tags_found": len(tags),
		"tags_added": addedCount,
	})
}

// GetProviderGitTags fetches available tags from the Git repository
// GET /api/providers/:id/git-tags
func GetProviderGitTags(c *gin.Context) {
	providerID := c.Param("id")

	// Get provider and its source URL
	var sourceURL *string
	err := database.DB.QueryRow("SELECT source_url FROM providers WHERE id = $1", providerID).Scan(&sourceURL)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	if sourceURL == nil || *sourceURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provider has no Git source URL"})
		return
	}

	// Fetch tags from Git
	tags, err := git.GetTags(*sourceURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tags: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, tags)
}

// GetProviderReadme fetches the README from the provider's Git repository
// GET /api/providers/:id/readme
func GetProviderReadme(c *gin.Context) {
	providerID := c.Param("id")
	ref := c.Query("ref") // Optional: specific version/tag

	// Get provider source URL
	var sourceURL *string
	err := database.DB.QueryRow("SELECT source_url FROM providers WHERE id = $1", providerID).Scan(&sourceURL)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	if sourceURL == nil || *sourceURL == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider has no Git source URL"})
		return
	}

	// Load auth config from database
	var auth *git.AuthConfig
	var authType sql.NullString
	var authData sql.NullString
	err = database.DB.QueryRow("SELECT git_auth_type, git_auth_data FROM providers WHERE id = $1", providerID).Scan(&authType, &authData)
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

	// Fetch README from Git with authentication
	readme, err := git.GetReadmeWithAuth(*sourceURL, ref, auth)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "README not found: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"content": readme})
}

// ToggleProviderVersion enables or disables a provider version
// PATCH /api/providers/:id/versions/:versionId
func ToggleProviderVersion(c *gin.Context) {
	providerID := c.Param("id")
	versionID := c.Param("versionId")

	var input struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := database.DB.Exec(`
		UPDATE provider_versions SET enabled = $1 WHERE id = $2 AND provider_id = $3
	`, input.Enabled, versionID, providerID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if affected, _ := result.RowsAffected(); affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}

	// Update provider updated_at
	database.DB.Exec("UPDATE providers SET updated_at = $1 WHERE id = $2", time.Now(), providerID)

	c.JSON(http.StatusOK, gin.H{"message": "Version updated", "enabled": input.Enabled})
}

// GetProviderPlatforms returns all platforms for a provider version
// GET /api/providers/:id/versions/:versionId/platforms
func GetProviderPlatforms(c *gin.Context) {
	versionID := c.Param("versionId")

	rows, err := database.DB.Query(`
		SELECT id, version_id, os, arch, filename, download_url, 
		       COALESCE(shasums_url, '') as shasums_url,
		       COALESCE(shasums_signature_url, '') as shasums_signature_url,
		       shasum, COALESCE(signing_keys, '') as signing_keys
		FROM provider_platforms
		WHERE version_id = $1
		ORDER BY os, arch
	`, versionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	platforms := make([]models.ProviderPlatform, 0)
	for rows.Next() {
		var p models.ProviderPlatform
		if err := rows.Scan(&p.ID, &p.VersionID, &p.OS, &p.Arch, &p.Filename,
			&p.DownloadURL, &p.SHASumsURL, &p.SHASumsSignature, &p.SHASum, &p.SigningKeys); err != nil {
			continue
		}
		platforms = append(platforms, p)
	}

	c.JSON(http.StatusOK, platforms)
}

// AddProviderPlatform adds a platform binary to a provider version
// POST /api/providers/:id/versions/:versionId/platforms
func AddProviderPlatform(c *gin.Context) {
	providerID := c.Param("id")
	versionID := c.Param("versionId")

	var input models.ProviderPlatformCreate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify version exists and belongs to provider
	var exists int
	err := database.DB.QueryRow(`
		SELECT 1 FROM provider_versions WHERE id = $1 AND provider_id = $2
	`, versionID, providerID).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}

	// Check if platform already exists
	var existingID string
	err = database.DB.QueryRow(`
		SELECT id FROM provider_platforms WHERE version_id = $1 AND os = $2 AND arch = $3
	`, versionID, input.OS, input.Arch).Scan(&existingID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Platform already exists for this OS/arch"})
		return
	}

	// Create platform
	platformID := generateID()
	_, err = database.DB.Exec(`
		INSERT INTO provider_platforms (id, version_id, os, arch, filename, download_url, shasums_url, shasums_signature_url, shasum, signing_keys)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, platformID, versionID, input.OS, input.Arch, input.Filename, input.DownloadURL,
		input.SHASumsURL, input.SHASumsSignature, input.SHASum, input.SigningKeys)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update provider updated_at
	database.DB.Exec("UPDATE providers SET updated_at = $1 WHERE id = $2", time.Now(), providerID)

	platform := models.ProviderPlatform{
		ID:               platformID,
		VersionID:        versionID,
		OS:               input.OS,
		Arch:             input.Arch,
		Filename:         input.Filename,
		DownloadURL:      input.DownloadURL,
		SHASumsURL:       input.SHASumsURL,
		SHASumsSignature: input.SHASumsSignature,
		SHASum:           input.SHASum,
		SigningKeys:      input.SigningKeys,
	}

	c.JSON(http.StatusCreated, platform)
}

// DeleteProviderPlatform removes a platform from a provider version and deletes the file
// DELETE /api/providers/:id/versions/:versionId/platforms/:platformId
func DeleteProviderPlatform(c *gin.Context) {
	providerID := c.Param("id")
	versionID := c.Param("versionId")
	platformID := c.Param("platformId")

	// Get platform info before deleting (to delete the file)
	var filename, namespace, providerName, version string
	err := database.DB.QueryRow(`
		SELECT pp.filename, n.name, p.name, pv.version
		FROM provider_platforms pp
		JOIN provider_versions pv ON pp.version_id = pv.id
		JOIN providers p ON pv.provider_id = p.id
		JOIN namespaces n ON p.namespace_id = n.id
		WHERE pp.id = $1 AND pp.version_id = $2 AND pv.provider_id = $3
	`, platformID, versionID, providerID).Scan(&filename, &namespace, &providerName, &version)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Platform not found"})
		return
	}

	// Delete from database
	result, err := database.DB.Exec(`
		DELETE FROM provider_platforms 
		WHERE id = $1 AND version_id = $2 
		AND version_id IN (SELECT id FROM provider_versions WHERE provider_id = $3)
	`, platformID, versionID, providerID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if affected, _ := result.RowsAffected(); affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Platform not found"})
		return
	}

	// Delete file from disk
	buildDir := os.Getenv("BUILD_DIR")
	if buildDir == "" {
		buildDir = "/app/data/builds"
	}
	filePath := filepath.Join(buildDir, "providers", namespace, providerName, version, filename)
	if err := os.Remove(filePath); err != nil {
		log.Printf("Warning: Could not delete file %s: %v", filePath, err)
	}

	// Update provider updated_at
	database.DB.Exec("UPDATE providers SET updated_at = $1 WHERE id = $2", time.Now(), providerID)

	c.JSON(http.StatusOK, gin.H{"message": "Platform deleted"})
}

// UploadProviderPlatform uploads a zip file for a specific platform
// POST /api/providers/:id/versions/:versionId/platforms/upload
func UploadProviderPlatform(c *gin.Context) {
	providerID := c.Param("id")
	versionID := c.Param("versionId")

	// Get form values
	osParam := c.PostForm("os")
	arch := c.PostForm("arch")

	if osParam == "" || arch == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "os and arch are required"})
		return
	}

	// Get provider info
	var providerName, namespace string
	err := database.DB.QueryRow(`
		SELECT p.name, n.name
		FROM providers p
		JOIN namespaces n ON p.namespace_id = n.id
		WHERE p.id = $1
	`, providerID).Scan(&providerName, &namespace)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	// Get version info
	var version string
	err = database.DB.QueryRow(`
		SELECT version FROM provider_versions WHERE id = $1 AND provider_id = $2
	`, versionID, providerID).Scan(&version)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}

	// Get the uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}
	defer file.Close()

	// Validate file extension
	if !strings.HasSuffix(header.Filename, ".zip") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File must be a .zip file"})
		return
	}

	// Create directory structure
	buildDir := os.Getenv("BUILD_DIR")
	if buildDir == "" {
		buildDir = "/app/data/builds"
	}
	outputDir := filepath.Join(buildDir, "providers", namespace, providerName, version)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create directory"})
		return
	}

	// Generate filename
	filename := "terraform-provider-" + providerName + "_" + version + "_" + osParam + "_" + arch + ".zip"
	filePath := filepath.Join(outputDir, filename)

	// Save file
	out, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}
	defer out.Close()

	// Calculate SHA256 while copying
	hash := sha256.New()
	writer := io.MultiWriter(out, hash)
	if _, err := io.Copy(writer, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}
	shasum := hex.EncodeToString(hash.Sum(nil))

	// Generate download URL - use X-Forwarded-Host/Host header or BASE_URL
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		// Try to get from request headers (set by nginx proxy)
		host := c.GetHeader("X-Forwarded-Host")
		if host == "" {
			host = c.GetHeader("Host")
		}
		if host == "" {
			backendHost := os.Getenv("BACKEND_HOST")
			if backendHost == "" {
				backendHost = "localhost"
			}
			backendPort := os.Getenv("PORT")
			if backendPort == "" {
				backendPort = "9080"
			}
			host = backendHost + ":" + backendPort
		}
		scheme := c.GetHeader("X-Forwarded-Proto")
		if scheme == "" {
			scheme = "http"
		}
		baseURL = scheme + "://" + host
	}
	downloadURL := baseURL + "/downloads/providers/" + namespace + "/" + providerName + "/" + version + "/" + filename

	// Check if platform already exists
	var existingID string
	err = database.DB.QueryRow(`
		SELECT id FROM provider_platforms WHERE version_id = $1 AND os = $2 AND arch = $3
	`, versionID, osParam, arch).Scan(&existingID)

	if err == nil {
		// Update existing platform
		_, err = database.DB.Exec(`
			UPDATE provider_platforms SET filename = $1, download_url = $2, shasum = $3
			WHERE id = $1
		`, filename, downloadURL, shasum, existingID)
	} else {
		// Create new platform
		platformID := generateID()
		_, err = database.DB.Exec(`
			INSERT INTO provider_platforms (id, version_id, os, arch, filename, download_url, shasum)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, platformID, versionID, osParam, arch, filename, downloadURL, shasum)
		existingID = platformID
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save platform: " + err.Error()})
		return
	}

	// Update provider updated_at
	database.DB.Exec("UPDATE providers SET updated_at = $1 WHERE id = $2", time.Now(), providerID)

	c.JSON(http.StatusOK, gin.H{
		"message":      "Platform uploaded successfully",
		"platform_id":  existingID,
		"os":           osParam,
		"arch":         arch,
		"filename":     filename,
		"shasum":       shasum,
		"download_url": downloadURL,
	})
}

package api

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"iac-tool/internal/database"
	"iac-tool/internal/models"
)

// generateAPIKey creates a cryptographically secure API key
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "tfr_" + hex.EncodeToString(bytes), nil
}

// hashAPIKey creates a SHA256 hash of the API key
func hashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// TerraformAuthMiddleware validates API keys for Terraform CLI access
// This is used for /v1/modules and /v1/providers endpoints
func TerraformAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if namespace requires authentication
		namespace := c.Param("namespace")
		if namespace == "" {
			c.Next()
			return
		}

		// Check if namespace is public
		var isPublic bool
		err := database.DB.QueryRow("SELECT is_public FROM namespaces WHERE name = ?", namespace).Scan(&isPublic)
		if err != nil {
			// Namespace not found - let the handler deal with it
			c.Next()
			return
		}

		// If namespace is public, allow access
		if isPublic {
			c.Next()
			return
		}

		// Private namespace - require authentication
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"errors": []string{"Authentication required for private namespace"},
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>" format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"errors": []string{"Invalid authorization header format"},
			})
			c.Abort()
			return
		}

		token := parts[1]
		keyHash := hashAPIKey(token)

		// Look up the API key
		var apiKey models.APIKey
		var expiresAt sql.NullTime
		err = database.DB.QueryRow(`
			SELECT ak.id, ak.namespace_id, ak.name, ak.permissions, ak.expires_at
			FROM api_keys ak
			JOIN namespaces n ON ak.namespace_id = n.id
			WHERE ak.key_hash = ? AND n.name = ?
		`, keyHash, namespace).Scan(&apiKey.ID, &apiKey.NamespaceID, &apiKey.Name, &apiKey.Permissions, &expiresAt)

		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{
				"errors": []string{"Invalid API key for this namespace"},
			})
			c.Abort()
			return
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"errors": []string{"Authentication error"},
			})
			c.Abort()
			return
		}

		// Check expiration
		if expiresAt.Valid && expiresAt.Time.Before(time.Now()) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"errors": []string{"API key has expired"},
			})
			c.Abort()
			return
		}

		// Update last used timestamp
		database.DB.Exec("UPDATE api_keys SET last_used_at = ? WHERE id = ?", time.Now(), apiKey.ID)

		c.Next()
	}
}

// ============================================================================
// Namespace CRUD (no authentication required)
// ============================================================================

// GetNamespaces returns all namespaces with stats
func GetNamespaces(c *gin.Context) {
	rows, err := database.DB.Query(`
		SELECT n.id, n.name, n.description, n.is_public, n.created_at, n.updated_at,
			   (SELECT COUNT(*) FROM modules WHERE namespace_id = n.id) as module_count,
			   (SELECT COUNT(*) FROM providers WHERE namespace_id = n.id) as provider_count
		FROM namespaces n
		ORDER BY n.name
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	namespaces := []models.NamespaceWithStats{}
	for rows.Next() {
		var ns models.NamespaceWithStats
		var description sql.NullString
		if err := rows.Scan(&ns.ID, &ns.Name, &description, &ns.IsPublic, &ns.CreatedAt, &ns.UpdatedAt, &ns.ModuleCount, &ns.ProviderCount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if description.Valid {
			ns.Description = &description.String
		}
		namespaces = append(namespaces, ns)
	}

	c.JSON(http.StatusOK, namespaces)
}

// GetNamespace returns a single namespace by ID
func GetNamespace(c *gin.Context) {
	id := c.Param("id")

	var ns models.NamespaceWithStats
	var description sql.NullString
	err := database.DB.QueryRow(`
		SELECT n.id, n.name, n.description, n.is_public, n.created_at, n.updated_at,
			   (SELECT COUNT(*) FROM modules WHERE namespace_id = n.id) as module_count,
			   (SELECT COUNT(*) FROM providers WHERE namespace_id = n.id) as provider_count
		FROM namespaces n WHERE n.id = ?
	`, id).Scan(&ns.ID, &ns.Name, &description, &ns.IsPublic, &ns.CreatedAt, &ns.UpdatedAt, &ns.ModuleCount, &ns.ProviderCount)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Namespace not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if description.Valid {
		ns.Description = &description.String
	}

	c.JSON(http.StatusOK, ns)
}

// CreateNamespace creates a new namespace
func CreateNamespace(c *gin.Context) {
	var input models.NamespaceCreate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := uuid.New().String()
	now := time.Now()

	_, err := database.DB.Exec(`
		INSERT INTO namespaces (id, name, description, is_public, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, id, input.Name, input.Description, input.IsPublic, now, now)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			c.JSON(http.StatusConflict, gin.H{"error": "Namespace already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ns := models.Namespace{
		ID:          id,
		Name:        input.Name,
		Description: input.Description,
		IsPublic:    input.IsPublic,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	c.JSON(http.StatusCreated, ns)
}

// UpdateNamespace updates an existing namespace
func UpdateNamespace(c *gin.Context) {
	id := c.Param("id")

	var input models.NamespaceUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}

	if input.Name != nil {
		updates = append(updates, "name = ?")
		args = append(args, *input.Name)
	}
	if input.Description != nil {
		updates = append(updates, "description = ?")
		args = append(args, *input.Description)
	}
	if input.IsPublic != nil {
		updates = append(updates, "is_public = ?")
		args = append(args, *input.IsPublic)
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	updates = append(updates, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, id)

	query := "UPDATE namespaces SET " + strings.Join(updates, ", ") + " WHERE id = ?"
	result, err := database.DB.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Namespace not found"})
		return
	}

	// Fetch updated namespace
	var ns models.Namespace
	var description sql.NullString
	database.DB.QueryRow("SELECT id, name, description, is_public, created_at, updated_at FROM namespaces WHERE id = ?", id).Scan(
		&ns.ID, &ns.Name, &description, &ns.IsPublic, &ns.CreatedAt, &ns.UpdatedAt,
	)
	if description.Valid {
		ns.Description = &description.String
	}

	c.JSON(http.StatusOK, ns)
}

// DeleteNamespace deletes a namespace
func DeleteNamespace(c *gin.Context) {
	id := c.Param("id")

	// Check if namespace has modules or providers
	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM modules WHERE namespace_id = ?", id).Scan(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Namespace has modules. Delete modules first."})
		return
	}
	database.DB.QueryRow("SELECT COUNT(*) FROM providers WHERE namespace_id = ?", id).Scan(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Namespace has providers. Delete providers first."})
		return
	}

	// Delete API keys first
	database.DB.Exec("DELETE FROM api_keys WHERE namespace_id = ?", id)

	result, err := database.DB.Exec("DELETE FROM namespaces WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Namespace not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Namespace deleted"})
}

// ============================================================================
// API Key Management (for Terraform CLI access to private namespaces)
// ============================================================================

// GetAPIKeys returns API keys for a namespace
func GetAPIKeys(c *gin.Context) {
	namespaceID := c.Param("id")

	rows, err := database.DB.Query(`
		SELECT id, namespace_id, name, permissions, expires_at, created_at, last_used_at
		FROM api_keys WHERE namespace_id = ?
		ORDER BY created_at DESC
	`, namespaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	keys := []models.APIKey{}
	for rows.Next() {
		var key models.APIKey
		var expiresAt, lastUsedAt sql.NullTime
		if err := rows.Scan(&key.ID, &key.NamespaceID, &key.Name, &key.Permissions, &expiresAt, &key.CreatedAt, &lastUsedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if expiresAt.Valid {
			key.ExpiresAt = &expiresAt.Time
		}
		if lastUsedAt.Valid {
			key.LastUsedAt = &lastUsedAt.Time
		}
		keys = append(keys, key)
	}

	c.JSON(http.StatusOK, keys)
}

// CreateAPIKey creates a new API key for a namespace
func CreateAPIKey(c *gin.Context) {
	namespaceID := c.Param("id")

	// Verify namespace exists
	var exists bool
	database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM namespaces WHERE id = ?)", namespaceID).Scan(&exists)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Namespace not found"})
		return
	}

	var input models.APIKeyCreate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate permissions
	if input.Permissions != "read" && input.Permissions != "write" && input.Permissions != "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid permissions. Must be read, write, or admin"})
		return
	}

	// Generate API key
	key, err := generateAPIKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate API key"})
		return
	}
	keyHash := hashAPIKey(key)

	id := uuid.New().String()
	now := time.Now()

	_, err = database.DB.Exec(`
		INSERT INTO api_keys (id, namespace_id, name, key_hash, permissions, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, namespaceID, input.Name, keyHash, input.Permissions, input.ExpiresAt, now)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	apiKey := models.APIKey{
		ID:          id,
		NamespaceID: namespaceID,
		Name:        input.Name,
		Key:         key, // Only returned on creation
		Permissions: input.Permissions,
		ExpiresAt:   input.ExpiresAt,
		CreatedAt:   now,
	}

	c.JSON(http.StatusCreated, apiKey)
}

// DeleteAPIKey deletes an API key
func DeleteAPIKey(c *gin.Context) {
	keyID := c.Param("keyId")

	result, err := database.DB.Exec("DELETE FROM api_keys WHERE id = ?", keyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key deleted"})
}

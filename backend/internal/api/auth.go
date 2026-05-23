package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"iac-tool/internal/database"
	"iac-tool/internal/models"
)

// jwtSecret is read from JWT_SECRET env var, falls back to a generated default
func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "change-me-in-production-jwt-secret-32chars"
	}
	return []byte(secret)
}

// JWTClaims holds the JWT payload
type JWTClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// JWTAuthMiddleware validates JWT tokens for management API
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenStr := parts[1]
		claims := &JWTClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return getJWTSecret(), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// AdminOnlyMiddleware restricts access to admin users
func AdminOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// TFStateBasicAuthMiddleware validates HTTP Basic Auth credentials against the
// users table for the Terraform/OpenTofu HTTP state backend endpoints.
// Any valid user (admin or reader) is allowed access.
func TFStateBasicAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, password, ok := c.Request.BasicAuth()
		if !ok {
			c.Header("WWW-Authenticate", `Basic realm="Terraform State Backend"`)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Basic authentication required"})
			c.Abort()
			return
		}

		db := database.DB
		var userID, hashedPassword, role string
		err := db.QueryRow(
			"SELECT id, password_hash, role FROM users WHERE username = $1",
			username,
		).Scan(&userID, &hashedPassword, &role)
		if err != nil {
			c.Header("WWW-Authenticate", `Basic realm="Terraform State Backend"`)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			c.Abort()
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
			c.Header("WWW-Authenticate", `Basic realm="Terraform State Backend"`)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Set("username", username)
		c.Set("role", role)
		c.Next()
	}
}

// ============================================================================
// Auth endpoints
// ============================================================================

// Login handles username/password login and returns a JWT
func Login(c *gin.Context) {
	var input models.LoginRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	err := database.DB.QueryRow(
		"SELECT id, username, password_hash, role, created_at, updated_at FROM users WHERE username = $1",
		input.Username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication error"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// Generate JWT
	claims := &JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(getJWTSecret())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, models.LoginResponse{
		Token: tokenStr,
		User:  user,
	})
}

// GetCurrentUser returns the authenticated user's info
func GetCurrentUser(c *gin.Context) {
	userID, _ := c.Get("user_id")
	var user models.User
	err := database.DB.QueryRow(
		"SELECT id, username, role, created_at, updated_at FROM users WHERE id = $1",
		userID,
	).Scan(&user.ID, &user.Username, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// ============================================================================
// User management (admin only)
// ============================================================================

// GetUsers returns all users (admin only)
func GetUsers(c *gin.Context) {
	rows, err := database.DB.Query(
		"SELECT id, username, role, created_at, updated_at FROM users ORDER BY username",
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	users := []models.User{}
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		users = append(users, u)
	}
	c.JSON(http.StatusOK, users)
}

// CreateUser creates a new user (admin only)
func CreateUser(c *gin.Context) {
	var input models.UserCreate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Role != "admin" && input.Role != "reader" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role must be 'admin' or 'reader'"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	id := uuid.New().String()
	now := time.Now()
	_, err = database.DB.Exec(
		"INSERT INTO users (id, username, password_hash, role, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)",
		id, input.Username, string(hash), input.Role, now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "UNIQUE") {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user := models.User{
		ID:        id,
		Username:  input.Username,
		Role:      input.Role,
		CreatedAt: now,
		UpdatedAt: now,
	}
	c.JSON(http.StatusCreated, user)
}

// UpdateUser updates a user's username or role (admin only)
func UpdateUser(c *gin.Context) {
	id := c.Param("id")

	var input models.UserUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Role != nil && *input.Role != "admin" && *input.Role != "reader" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role must be 'admin' or 'reader'"})
		return
	}

	now := time.Now()
	updates := []string{"updated_at = $1"}
	args := []interface{}{now}

	if input.Username != nil {
		updates = append(updates, "username = $"+itoa(len(args)+1))
		args = append(args, *input.Username)
	}
	if input.Role != nil {
		updates = append(updates, "role = $"+itoa(len(args)+1))
		args = append(args, *input.Role)
	}
	args = append(args, id)

	query := "UPDATE users SET " + strings.Join(updates, ", ") + " WHERE id = $" + itoa(len(args))
	result, err := database.DB.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var user models.User
	database.DB.QueryRow(
		"SELECT id, username, role, created_at, updated_at FROM users WHERE id = $1", id,
	).Scan(&user.ID, &user.Username, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	c.JSON(http.StatusOK, user)
}

// ChangeUserPassword updates a user's password (admin only)
func ChangeUserPassword(c *gin.Context) {
	id := c.Param("id")

	var input models.UserChangePassword
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	result, err := database.DB.Exec(
		"UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3",
		string(hash), time.Now(), id,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Password updated"})
}

// DeleteUser deletes a user (admin only)
func DeleteUser(c *gin.Context) {
	id := c.Param("id")

	// Prevent deleting yourself
	currentUserID, _ := c.Get("user_id")
	if currentUserID == id {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete your own account"})
		return
	}

	result, err := database.DB.Exec("DELETE FROM users WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

// itoa converts int to string for query parameter placeholders
func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

package registry

import (
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	registryToken string
	tokenMu       sync.RWMutex
)

// InitToken initializes or loads the registry authentication token
func InitToken() error {
	tokenMu.Lock()
	defer tokenMu.Unlock()

	// Check if token exists in environment
	if token := os.Getenv("REGISTRY_AUTH_TOKEN"); token != "" {
		registryToken = token
		return nil
	}

	// Try to load from file
	tokenFile := filepath.Join("/app/data", ".registry-token")
	if data, err := os.ReadFile(tokenFile); err == nil {
		registryToken = strings.TrimSpace(string(data))
		return nil
	}

	// Generate new token
	token, err := generateToken()
	if err != nil {
		return err
	}

	registryToken = token

	// Save to file
	if err := os.WriteFile(tokenFile, []byte(token), 0600); err != nil {
		return err
	}

	return nil
}

// GetToken returns the current registry token
func GetToken() string {
	tokenMu.RLock()
	defer tokenMu.RUnlock()
	return registryToken
}

// generateToken generates a cryptographically secure random token
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

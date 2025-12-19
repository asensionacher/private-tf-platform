package api

import (
	"strings"

	"github.com/google/uuid"
)

// generateID generates a new UUID for use as an identifier
func generateID() string {
	return uuid.New().String()
}

// isValidGitURL validates that the URL is a valid Git repository URL (HTTPS only)
func isValidGitURL(url string) bool {
	// Only accept https:// URLs
	if strings.HasPrefix(url, "https://") {
		return true
	}
	return false
}

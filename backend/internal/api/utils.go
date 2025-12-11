package api

import (
	"strings"

	"github.com/google/uuid"
)

// generateID generates a new UUID for use as an identifier
func generateID() string {
	return uuid.New().String()
}

// isValidGitURL validates that the URL is a valid Git repository URL
func isValidGitURL(url string) bool {
	// Accept https:// URLs
	if strings.HasPrefix(url, "https://") {
		return true
	}
	// Accept git@ SSH URLs
	if strings.HasPrefix(url, "git@") {
		return true
	}
	// Accept git:// URLs
	if strings.HasPrefix(url, "git://") {
		return true
	}
	return false
}

package api

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// providerNameRe matches valid Terraform provider names:
// lowercase letters, digits, hyphens; no leading/trailing/consecutive hyphens.
var providerNameRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// isValidProviderName checks that a provider name (the part after the namespace slash,
// e.g. "random" in "hashicorp/random") conforms to Terraform registry naming rules.
func isValidProviderName(name string) bool {
	// Allow optional "namespace/name" form — validate only the local name part.
	local := name
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		local = name[idx+1:]
	}
	if local == "" {
		return false
	}
	return providerNameRe.MatchString(local)
}

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

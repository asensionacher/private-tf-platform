package git

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Tag represents a Git tag
type Tag struct {
	Name    string    `json:"name"`
	Version string    `json:"version"` // Normalized version (without 'v' prefix)
	TagDate time.Time `json:"tag_date"`
}

// GetTags fetches all tags from a Git repository URL with their dates
func GetTags(repoURL string) ([]Tag, error) {
	// Ensure URL ends with .git
	url := repoURL
	if !strings.HasSuffix(url, ".git") {
		url = url + ".git"
	}

	// Clone the repo and get all tags with their dates
	tags, err := getTagsViaGitClone(url)
	if err != nil {
		return nil, err
	}

	// Sort by tag date (newest first), falling back to version comparison
	sort.Slice(tags, func(i, j int) bool {
		if !tags[i].TagDate.IsZero() && !tags[j].TagDate.IsZero() {
			return tags[i].TagDate.After(tags[j].TagDate)
		}
		return compareVersions(tags[i].Version, tags[j].Version) > 0
	})

	return tags, nil
}

// getTagsViaGitClone clones the repository and gets all tags with their commit dates
func getTagsViaGitClone(repoURL string) ([]Tag, error) {
	// Create a temporary directory for the clone
	tmpDir, err := os.MkdirTemp("", "git-tags-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Do a bare clone with minimal data
	cmd := exec.Command("git", "clone", "--bare", "--filter=blob:none", repoURL, tmpDir)
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_SSH_COMMAND=ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git clone failed: %v: %s", err, string(output))
	}

	// Use git for-each-ref to get all tags with their dates
	// %(creatordate) gives the tag date for annotated tags, or commit date for lightweight tags
	forEachRefCmd := exec.Command("git", "-C", tmpDir, "for-each-ref",
		"--format=%(refname:short)|%(creatordate:iso8601-strict)",
		"refs/tags")

	refOutput, err := forEachRefCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git for-each-ref failed: %w", err)
	}

	// Regex to match version-like tags
	versionRegex := regexp.MustCompile(`^v?(\d+\.\d+(\.\d+)?(-[\w.]+)?)$`)

	tags := make([]Tag, 0)
	lines := strings.Split(strings.TrimSpace(string(refOutput)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}

		tagName := parts[0]
		dateStr := strings.TrimSpace(parts[1])

		// Check if it looks like a version
		if !versionRegex.MatchString(tagName) {
			continue
		}

		version := tagName
		if strings.HasPrefix(version, "v") {
			version = version[1:]
		}

		tag := Tag{
			Name:    tagName,
			Version: version,
		}

		// Parse the date
		if dateStr != "" {
			if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
				tag.TagDate = t
			} else if t, err := time.Parse("2006-01-02 15:04:05 -0700", dateStr); err == nil {
				tag.TagDate = t
			}
		}

		tags = append(tags, tag)
	}

	return tags, nil
}

// compareVersions compares two version strings
// Returns: 1 if v1 > v2, -1 if v1 < v2, 0 if equal
func compareVersions(v1, v2 string) int {
	// Split by dots and compare each part
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var p1, p2 string
		if i < len(parts1) {
			p1 = parts1[i]
		} else {
			p1 = "0"
		}
		if i < len(parts2) {
			p2 = parts2[i]
		} else {
			p2 = "0"
		}

		// Extract numeric part (handle pre-release suffixes)
		n1 := extractNumber(p1)
		n2 := extractNumber(p2)

		if n1 > n2 {
			return 1
		} else if n1 < n2 {
			return -1
		}
	}

	return 0
}

// extractNumber extracts the leading number from a string
func extractNumber(s string) int {
	num := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			num = num*10 + int(c-'0')
		} else {
			break
		}
	}
	return num
}

// ValidateGitRepository checks if a URL points to a valid, accessible Git repository
func ValidateGitRepository(repoURL string) error {
	// Ensure URL ends with .git
	url := repoURL
	if !strings.HasSuffix(url, ".git") {
		url = url + ".git"
	}

	// Try to do a minimal ls-remote to verify the repository exists and is accessible
	cmd := exec.Command("git", "ls-remote", "--heads", url)
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_SSH_COMMAND=ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// For URLs with embedded credentials (e.g., Azure DevOps), ls-remote might fail
		// but the actual clone with credentials might work. Return a warning but don't fail.
		if strings.Contains(repoURL, "@") && strings.Contains(repoURL, "://") {
			// This looks like a URL with embedded credentials, allow it
			return nil
		}
		return fmt.Errorf("repository validation failed: %v: %s", err, string(output))
	}

	// If we get here, the repository is valid and accessible
	return nil
}

// GetReadme fetches the README.md content from a Git repository
// Works with any Git repository by cloning and reading the file
func GetReadme(repoURL string, ref string) (string, error) {
	// Create a temporary directory for the clone
	tmpDir, err := os.MkdirTemp("", "git-readme-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Ensure URL ends with .git
	url := repoURL
	if !strings.HasSuffix(url, ".git") {
		url = url + ".git"
	}

	// If no ref specified, use HEAD
	if ref == "" {
		ref = "HEAD"
	}

	// Do a shallow clone with depth 1 for the specific ref
	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", ref, url, tmpDir)
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_SSH_COMMAND=ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// If branch doesn't exist, try without --branch flag (uses default branch)
		if ref == "HEAD" || strings.Contains(string(output), "Remote branch") {
			cmd = exec.Command("git", "clone", "--depth", "1", url, tmpDir)
			cmd.Env = append(os.Environ(),
				"GIT_TERMINAL_PROMPT=0",
				"GIT_SSH_COMMAND=ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null",
			)
			output, err = cmd.CombinedOutput()
			if err != nil {
				return "", fmt.Errorf("git clone failed: %v: %s", err, string(output))
			}
		} else {
			return "", fmt.Errorf("git clone failed: %v: %s", err, string(output))
		}
	}

	// Try to read README.md (case variations)
	readmeNames := []string{"README.md", "readme.md", "Readme.md", "README.MD", "README"}
	for _, name := range readmeNames {
		readmePath := tmpDir + "/" + name
		if content, err := os.ReadFile(readmePath); err == nil {
			return string(content), nil
		}
	}

	return "", fmt.Errorf("README not found in repository")
}


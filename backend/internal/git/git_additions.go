package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ListBranches lists all branches in a repository
func ListBranches(repoURL string, auth *AuthConfig) ([]string, error) {
	return listReferences(repoURL, auth, "refs/heads/")
}

// ListTagNames lists all tags in a repository
func ListTagNames(repoURL string, auth *AuthConfig) ([]string, error) {
	return listReferences(repoURL, auth, "refs/tags/")
}

// listReferences lists git references (branches or tags)
func listReferences(repoURL string, auth *AuthConfig, refPrefix string) ([]string, error) {
	// Ensure URL format
	url := repoURL
	if !strings.HasSuffix(url, ".git") && !strings.Contains(url, "dev.azure.com") && !strings.Contains(url, "/_git/") {
		url = url + ".git"
	}

	// Inject HTTPS credentials if provided
	if auth != nil && auth.Username != "" {
		url = injectHTTPSCredentials(url, auth.Username, auth.Password)
	}

	// Prepare environment
	env := os.Environ()
	env = append(env, "GIT_TERMINAL_PROMPT=0")
	env = append(env, "GIT_SSH_COMMAND=ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null")

	// Run git ls-remote
	cmd := exec.Command("git", "ls-remote", "--heads", "--tags", url)
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git ls-remote failed: %v: %s", err, string(output))
	}

	// Parse output
	var refs []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		ref := parts[1]
		if strings.HasPrefix(ref, refPrefix) {
			refName := strings.TrimPrefix(ref, refPrefix)
			refs = append(refs, refName)
		}
	}

	return refs, nil
}

// ListDirectory lists files and directories at a specific path in a repository
// Returns files and an optional README content if found
func ListDirectory(repoURL string, ref string, path string, auth *AuthConfig) ([]map[string]interface{}, *string, error) {
	// Create a temporary directory for the clone
	tmpDir, err := os.MkdirTemp("", "git-ls-*")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Ensure URL format
	url := repoURL
	if !strings.HasSuffix(url, ".git") && !strings.Contains(url, "dev.azure.com") && !strings.Contains(url, "/_git/") {
		url = url + ".git"
	}

	// Inject HTTPS credentials if provided
	if auth != nil && auth.Username != "" {
		url = injectHTTPSCredentials(url, auth.Username, auth.Password)
	}

	// If no ref specified, use HEAD
	if ref == "" {
		ref = "HEAD"
	}

	// Prepare environment
	env := os.Environ()
	env = append(env, "GIT_TERMINAL_PROMPT=0")
	env = append(env, "GIT_SSH_COMMAND=ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null")

	// Clone repository
	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", ref, url, tmpDir)
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("git clone failed: %v: %s", err, string(output))
	}

	// List directory contents
	fullPath := filepath.Join(tmpDir, path)
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []map[string]interface{}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		entryPath := path
		if entryPath != "" && !strings.HasSuffix(entryPath, "/") {
			entryPath += "/"
		}
		entryPath += entry.Name()

		fileType := "file"
		if entry.IsDir() {
			fileType = "dir"
		}

		files = append(files, map[string]interface{}{
			"name":   entry.Name(),
			"path":   entryPath,
			"type":   fileType,
			"is_dir": entry.IsDir(),
			"size":   info.Size(),
		})
	}

	// Try to read README from the same clone (case-insensitive)
	var readme *string
	for _, entry := range entries {
		// Check if the filename matches "readme.md" (case-insensitive)
		entryNameLower := strings.ToLower(entry.Name())
		if entryNameLower == "readme.md" || entryNameLower == "readme" {
			readmeFullPath := filepath.Join(fullPath, entry.Name())
			content, err := os.ReadFile(readmeFullPath)
			if err == nil {
				contentStr := string(content)
				readme = &contentStr
				break
			}
		}
	}

	return files, readme, nil
}

// GetFileContent reads the content of a specific file from a repository
func GetFileContent(repoURL string, ref string, filePath string, auth *AuthConfig) (string, error) {
	// Create a temporary directory for the clone
	tmpDir, err := os.MkdirTemp("", "git-file-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Ensure URL format
	url := repoURL
	if !strings.HasSuffix(url, ".git") && !strings.Contains(url, "dev.azure.com") && !strings.Contains(url, "/_git/") {
		url = url + ".git"
	}

	// Inject HTTPS credentials if provided
	if auth != nil && auth.Username != "" {
		url = injectHTTPSCredentials(url, auth.Username, auth.Password)
	}

	// If no ref specified, use HEAD
	if ref == "" {
		ref = "HEAD"
	}

	// Prepare environment
	env := os.Environ()
	env = append(env, "GIT_TERMINAL_PROMPT=0")
	env = append(env, "GIT_SSH_COMMAND=ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null")

	// Clone the repository
	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", ref, url, tmpDir)
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git clone failed: %v: %s", err, string(output))
	}

	// Read the file
	fullPath := filepath.Join(tmpDir, filePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(content), nil
}

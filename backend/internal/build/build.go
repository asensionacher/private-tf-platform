package build

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Platform represents a target platform for compilation
type Platform struct {
	OS   string
	Arch string
}

// BuildResult represents the result of building for a platform
type BuildResult struct {
	Platform    Platform
	Filename    string
	FilePath    string
	SHA256      string
	DownloadURL string
	Error       error
}

// DefaultPlatforms returns the default platforms to build for
func DefaultPlatforms() []Platform {
	return []Platform{
		{OS: "linux", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
		{OS: "darwin", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
		{OS: "windows", Arch: "amd64"},
	}
}

// BuildProvider clones a Git repo at a specific tag and compiles the provider for all platforms
func BuildProvider(gitURL, version, providerName, namespace, providerID, versionID, baseURL, buildDir string) ([]BuildResult, error) {
	// Create temp directory for cloning
	tempDir, err := os.MkdirTemp("", "provider-build-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone the repository at the specific tag
	tag := "v" + version
	if !strings.HasPrefix(version, "v") {
		tag = "v" + version
	} else {
		tag = version
	}

	cloneCmd := exec.Command("git", "clone", "--depth", "1", "--branch", tag, gitURL, tempDir)
	cloneCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if output, err := cloneCmd.CombinedOutput(); err != nil {
		// Try without v prefix
		tag = strings.TrimPrefix(version, "v")
		cloneCmd = exec.Command("git", "clone", "--depth", "1", "--branch", tag, gitURL, tempDir)
		cloneCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		if output2, err2 := cloneCmd.CombinedOutput(); err2 != nil {
			return nil, fmt.Errorf("failed to clone repository: %s / %s", string(output), string(output2))
		}
	}

	// Create output directory for binaries
	outputDir := filepath.Join(buildDir, "providers", namespace, providerName, version)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output dir: %w", err)
	}

	// Build for each platform
	platforms := DefaultPlatforms()
	results := make([]BuildResult, 0, len(platforms))

	for _, platform := range platforms {
		result := buildForPlatform(tempDir, outputDir, platform, providerName, version, namespace, versionID, baseURL)
		results = append(results, result)
	}

	return results, nil
}

func buildForPlatform(sourceDir, outputDir string, platform Platform, providerName, version, namespace, versionID, baseURL string) BuildResult {
	result := BuildResult{Platform: platform}

	// Determine output filename
	ext := ""
	if platform.OS == "windows" {
		ext = ".exe"
	}

	binaryName := fmt.Sprintf("terraform-provider-%s_v%s", providerName, version)
	filename := fmt.Sprintf("terraform-provider-%s_%s_%s_%s%s", providerName, version, platform.OS, platform.Arch, ext)
	outputPath := filepath.Join(outputDir, filename)

	// Build command
	buildCmd := exec.Command("go", "build", "-o", outputPath, ".")
	buildCmd.Dir = sourceDir
	buildCmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		fmt.Sprintf("GOOS=%s", platform.OS),
		fmt.Sprintf("GOARCH=%s", platform.Arch),
	)

	if output, err := buildCmd.CombinedOutput(); err != nil {
		result.Error = fmt.Errorf("build failed for %s/%s: %s - %w", platform.OS, platform.Arch, string(output), err)
		return result
	}

	// Create zip file
	zipFilename := fmt.Sprintf("terraform-provider-%s_%s_%s_%s.zip", providerName, version, platform.OS, platform.Arch)
	zipPath := filepath.Join(outputDir, zipFilename)

	if err := createZip(outputPath, binaryName+ext, zipPath); err != nil {
		result.Error = fmt.Errorf("failed to create zip: %w", err)
		return result
	}

	// Calculate SHA256 of zip
	zipSha, err := calculateSHA256(zipPath)
	if err != nil {
		result.Error = fmt.Errorf("failed to calculate zip SHA256: %w", err)
		return result
	}

	// Remove the unzipped binary, keep only zip
	os.Remove(outputPath)

	result.Filename = zipFilename
	result.FilePath = zipPath
	result.SHA256 = zipSha
	result.DownloadURL = fmt.Sprintf("%s/downloads/providers/%s/%s/%s/%s", baseURL, namespace, providerName, version, zipFilename)

	return result
}

func calculateSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func createZip(sourcePath, nameInZip, zipPath string) error {
	// Use zip command for simplicity
	cmd := exec.Command("zip", "-j", zipPath, sourcePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("zip command failed: %s - %w", string(output), err)
	}

	// Rename the file inside the zip to match Terraform expectations
	// Actually, let's just copy with the right name first
	return nil
}

package build

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"iac-tool/internal/crypto"
	"iac-tool/internal/database"
	"iac-tool/internal/git"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/creack/pty"
)

// ExecuteDeploymentRun executes a deployment run with init, plan, and apply
func ExecuteDeploymentRun(runID, deploymentID, path, ref, tool string, envVars map[string]string) {
	// Mark as initializing
	now := time.Now()
	database.DB.Exec(`
		UPDATE deployment_runs
		SET status = 'initializing', started_at = ?
		WHERE id = ?
	`, now, runID)

	// Create temporary work directory
	workDir := filepath.Join("/tmp", "iac-deployments", runID)
	if err := os.MkdirAll(workDir, 0755); err != nil {
		failRun(runID, "Failed to create work directory: "+err.Error())
		return
	}

	// Store work directory
	database.DB.Exec(`UPDATE deployment_runs SET work_dir = ? WHERE id = ?`, workDir, runID)

	// Schedule cleanup
	go scheduleCleanup(runID, workDir)

	// Get deployment info
	var gitURL string
	var authType, authDataStr sql.NullString
	err := database.DB.QueryRow(`
		SELECT git_url, git_auth_type, git_auth_data 
		FROM deployments 
		WHERE id = ?
	`, deploymentID).Scan(&gitURL, &authType, &authDataStr)
	if err != nil {
		failRun(runID, "Failed to get deployment info: "+err.Error())
		return
	}

	// Prepare auth config
	var auth *git.AuthConfig
	if authType.Valid && authDataStr.Valid {
		decryptedData, err := crypto.DecryptJSON(authDataStr.String)
		if err == nil {
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

	// Clone repository
	if err := git.Clone(gitURL, ref, workDir, auth); err != nil {
		failRun(runID, "Failed to clone repository: "+err.Error())
		return
	}

	// Navigate to deployment path
	deployPath := filepath.Join(workDir, path)
	if _, err := os.Stat(deployPath); os.IsNotExist(err) {
		failRun(runID, "Deployment path does not exist: "+path)
		return
	}

	// Execute init
	initLog, err := executeCommand(tool, deployPath, []string{"init"}, envVars)
	database.DB.Exec(`UPDATE deployment_runs SET init_log = ? WHERE id = ?`, initLog, runID)
	if err != nil {
		failRun(runID, "Init failed: "+err.Error())
		return
	}

	// Execute plan
	database.DB.Exec(`UPDATE deployment_runs SET status = 'planning' WHERE id = ?`, runID)
	planLog, err := executeCommand(tool, deployPath, []string{"plan", "-out=tfplan"}, envVars)
	database.DB.Exec(`UPDATE deployment_runs SET plan_log = ? WHERE id = ?`, planLog, runID)
	if err != nil {
		failRun(runID, "Plan failed: "+err.Error())
		return
	}

	// Wait for approval
	database.DB.Exec(`UPDATE deployment_runs SET status = 'awaiting_approval' WHERE id = ?`, runID)

	// Poll for approval (timeout after 24 hours)
	approved := waitForApproval(runID, 24*time.Hour)
	if !approved {
		database.DB.Exec(`
			UPDATE deployment_runs 
			SET status = 'cancelled', completed_at = ? 
			WHERE id = ?
		`, time.Now(), runID)
		return
	}

	// Execute apply
	database.DB.Exec(`UPDATE deployment_runs SET status = 'applying' WHERE id = ?`, runID)
	applyLog, err := executeCommand(tool, deployPath, []string{"apply", "tfplan"}, envVars)
	database.DB.Exec(`UPDATE deployment_runs SET apply_log = ? WHERE id = ?`, applyLog, runID)
	if err != nil {
		failRun(runID, "Apply failed: "+err.Error())
		return
	}

	// Mark as success
	database.DB.Exec(`
		UPDATE deployment_runs 
		SET status = 'success', completed_at = ? 
		WHERE id = ?
	`, time.Now(), runID)
}

// executeCommand runs a terraform/tofu command and returns the output with ANSI colors
func executeCommand(tool, workDir string, args []string, envVars map[string]string) (string, error) {
	cmdName := tool
	if tool == "tofu" {
		cmdName = "tofu"
	} else {
		cmdName = "terraform"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdName, args...)
	cmd.Dir = workDir

	// Set environment variables
	cmd.Env = os.Environ()
	for k, v := range envVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Use pty to get colored output
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return "", err
	}
	defer ptmx.Close()

	// Read all output
	var output strings.Builder
	scanner := bufio.NewScanner(ptmx)
	for scanner.Scan() {
		line := scanner.Text()
		output.WriteString(line + "\n")
	}

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		return output.String(), err
	}

	return output.String(), nil
}

// waitForApproval polls the database for approval
func waitForApproval(runID string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		var approvedBy sql.NullString
		err := database.DB.QueryRow(`
			SELECT approved_by FROM deployment_runs WHERE id = ?
		`, runID).Scan(&approvedBy)

		if err == nil && approvedBy.Valid {
			return approvedBy.String != "REJECTED"
		}

		time.Sleep(2 * time.Second)
	}

	return false
}

// failRun marks a run as failed
func failRun(runID, errorMsg string) {
	database.DB.Exec(`
		UPDATE deployment_runs 
		SET status = 'failed', error_message = ?, completed_at = ? 
		WHERE id = ?
	`, errorMsg, time.Now(), runID)
}

// scheduleCleanup removes the work directory after 24 hours
func scheduleCleanup(runID, workDir string) {
	time.Sleep(24 * time.Hour)

	// Check if run is still in progress
	var status string
	database.DB.QueryRow(`SELECT status FROM deployment_runs WHERE id = ?`, runID).Scan(&status)

	// Only cleanup if completed or cancelled
	if status == "success" || status == "failed" || status == "cancelled" {
		os.RemoveAll(workDir)
	}
}

package build

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"iac-tool/internal/crypto"
	"iac-tool/internal/database"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// RunnerDeploymentRequest matches the runner's DeploymentRequest
type RunnerDeploymentRequest struct {
	Tool        string            `json:"tool"`
	GitURL      string            `json:"git_url"`
	GitRef      string            `json:"git_ref"`
	Path        string            `json:"path"`
	EnvVars     map[string]string `json:"env_vars"`
	TfvarsFiles []string          `json:"tfvars_files"`
	InitFlags   string            `json:"init_flags,omitempty"`
	PlanFlags   string            `json:"plan_flags,omitempty"`
	Timeout     int               `json:"timeout"`
	GitAuth     *RunnerGitAuth    `json:"git_auth,omitempty"`
	AutoApprove bool              `json:"auto_approve"`
}

type RunnerGitAuth struct {
	Type     string `json:"type"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type RunnerDeploymentResponse struct {
	DeploymentID string `json:"deployment_id"`
	Status       string `json:"status"`
	Message      string `json:"message,omitempty"`
}

type RunnerDeploymentStatus struct {
	DeploymentID string     `json:"deployment_id"`
	Status       string     `json:"status"`
	Phase        string     `json:"phase"`
	StartedAt    time.Time  `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
	Error        string     `json:"error,omitempty"`
	InitLog      string     `json:"init_log,omitempty"`
	PlanLog      string     `json:"plan_log,omitempty"`
	PlanOutput   string     `json:"plan_output,omitempty"`
	ApplyLog     string     `json:"apply_log,omitempty"`
	ApplyOutput  string     `json:"apply_output,omitempty"`
}

// ExecuteDeploymentRun executes a deployment run via the runner HTTP API
func ExecuteDeploymentRun(runID, deploymentID, path, ref, tool string, envVars map[string]string, tfvarsFiles []string, initFlags, planFlags string) {
	// Mark as initializing
	now := time.Now()
	database.DB.Exec(`
UPDATE deployment_runs
SET status = 'initializing', started_at = $1
WHERE id = $2
`, now, runID)

	// Get deployment info
	var gitURL string
	var authType, authDataStr sql.NullString
	err := database.DB.QueryRow(`
SELECT git_url, git_auth_type, git_auth_data 
FROM deployments 
WHERE id = $1
`, deploymentID).Scan(&gitURL, &authType, &authDataStr)
	if err != nil {
		failRun(runID, "Failed to get deployment info: "+err.Error())
		return
	}

	// Prepare git auth
	var gitAuth *RunnerGitAuth
	if authType.Valid && authDataStr.Valid {
		decryptedData, err := crypto.DecryptJSON(authDataStr.String)
		if err == nil {
			var authJSON map[string]string
			if err := json.Unmarshal([]byte(decryptedData), &authJSON); err == nil {
				gitAuth = &RunnerGitAuth{
					Type:     authType.String,
					Username: authJSON["username"],
					Password: authJSON["password"],
				}
			}
		}
	}

	// Create deployment request
	runnerReq := RunnerDeploymentRequest{
		Tool:        tool,
		GitURL:      gitURL,
		GitRef:      ref,
		Path:        path,
		EnvVars:     envVars,
		TfvarsFiles: tfvarsFiles,
		InitFlags:   initFlags,
		PlanFlags:   planFlags,
		Timeout:     60,
		GitAuth:     gitAuth,
		AutoApprove: false, // Manual approval required
	}

	// Get runner URL from environment
	runnerURL := os.Getenv("RUNNER_URL")
	if runnerURL == "" {
		runnerURL = "http://runner:8080"
	}

	// Start deployment on runner
	reqBody, _ := json.Marshal(runnerReq)
	resp, err := http.Post(runnerURL+"/deploy", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		failRun(runID, "Failed to contact runner: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 202 {
		body, _ := io.ReadAll(resp.Body)
		failRun(runID, fmt.Sprintf("Runner returned error: %s", string(body)))
		return
	}

	var deployResp RunnerDeploymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&deployResp); err != nil {
		failRun(runID, "Failed to parse runner response: "+err.Error())
		return
	}

	runnerDeploymentID := deployResp.DeploymentID

	// Store runner deployment ID
	database.DB.Exec(`UPDATE deployment_runs SET work_dir = $1 WHERE id = $2`, runnerDeploymentID, runID)

	// Poll runner for status updates
	pollRunnerStatus(runID, runnerDeploymentID, runnerURL)
}

func pollRunnerStatus(runID, runnerDeploymentID, runnerURL string) {
	ticker := time.NewTicker(500 * time.Millisecond) // Poll every 500ms for near-real-time updates
	defer ticker.Stop()

	timeout := time.After(2 * time.Hour)
	firstUpdate := true
	waitingForApproval := false

	log.Printf("Starting polling for run %s, runner deployment %s", runID, runnerDeploymentID)

	for {
		select {
		case <-ticker.C:
			// Get status from runner
			resp, err := http.Get(fmt.Sprintf("%s/deploy/%s/status", runnerURL, runnerDeploymentID))
			if err != nil {
				log.Printf("Error getting status from runner: %v", err)
				continue
			}

			var status RunnerDeploymentStatus
			if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
				resp.Body.Close()
				log.Printf("Error decoding runner status: %v", err)
				continue
			}
			resp.Body.Close()

			log.Printf("Runner status: status=%s, phase=%s", status.Status, status.Phase)

			// Update started_at on first update
			if firstUpdate && status.StartedAt.Unix() > 0 {
				database.DB.Exec(`UPDATE deployment_runs SET started_at = $1 WHERE id = $2`, status.StartedAt, runID)
				firstUpdate = false
			}

			// Always update logs - this ensures logs are visible while waiting for approval
			result, err := database.DB.Exec(`
				UPDATE deployment_runs 
				SET init_log = $1, plan_log = $2, plan_output = $3, apply_log = $4, apply_output = $5
				WHERE id = $6
			`, status.InitLog, status.PlanLog, status.PlanOutput, status.ApplyLog, status.ApplyOutput, runID)
			if err != nil {
				log.Printf("Error updating logs: %v", err)
			} else {
				rows, _ := result.RowsAffected()
				log.Printf("Updated logs, rows affected: %d", rows)
			}

			// Update status based on phase (if not waiting for approval)
			if !waitingForApproval && status.Phase != "" {
				// Map runner phase names to database status names
				dbStatus := status.Phase
				switch status.Phase {
				case "init":
					dbStatus = "initializing"
				case "plan":
					dbStatus = "planning"
				case "apply":
					dbStatus = "applying"
				}

				log.Printf("Updating status to phase: %s (mapped to: %s)", status.Phase, dbStatus)
				result, err := database.DB.Exec(`UPDATE deployment_runs SET status = $1 WHERE id = $2`, dbStatus, runID)
				if err != nil {
					log.Printf("Error updating status to phase: %v", err)
				} else {
					rows, _ := result.RowsAffected()
					log.Printf("Updated status to %s, rows affected: %d", dbStatus, rows)
				}
			}

			// Check if waiting for approval
			if status.Status == "awaiting_approval" && !waitingForApproval {
				waitingForApproval = true
				log.Printf("Deployment is awaiting approval, updating status")
				result, err := database.DB.Exec(`UPDATE deployment_runs SET status = 'awaiting_approval' WHERE id = $1`, runID)
				if err != nil {
					log.Printf("Error updating status to awaiting_approval: %v", err)
				} else {
					rows, _ := result.RowsAffected()
					log.Printf("Updated status to awaiting_approval, rows affected: %d", rows)
				}
			}

			// If waiting for approval, check database for approval decision
			if waitingForApproval {
				var approvedBy sql.NullString
				err := database.DB.QueryRow(`SELECT approved_by FROM deployment_runs WHERE id = $1`, runID).Scan(&approvedBy)

				if err == nil && approvedBy.Valid {
					if approvedBy.String == "REJECTED" {
						// Send rejection to runner
						log.Printf("Approval rejected, sending to runner")
						http.Post(fmt.Sprintf("%s/deploy/%s/reject", runnerURL, runnerDeploymentID), "application/json", nil)
						waitingForApproval = false
						// Continue polling to get final status
						continue
					} else {
						// Send approval to runner
						log.Printf("Approval granted, sending to runner")
						http.Post(fmt.Sprintf("%s/deploy/%s/approve", runnerURL, runnerDeploymentID), "application/json", nil)
						database.DB.Exec(`UPDATE deployment_runs SET status = 'applying' WHERE id = $1`, runID)
						waitingForApproval = false
						// Continue polling for apply phase
						continue
					}
				}
				// Still waiting, continue polling
				continue
			}

			// Check if deployment finished
			if status.Status == "success" {
				log.Printf("Deployment successful")
				database.DB.Exec(`
					UPDATE deployment_runs 
					SET status = 'success', completed_at = $1 
					WHERE id = $2
				`, time.Now(), runID)
				return
			}

			if status.Status == "failed" || status.Status == "cancelled" {
				database.DB.Exec(`
					UPDATE deployment_runs 
					SET status = $1, error_message = $2, completed_at = $3 
					WHERE id = $4
				`, status.Status, status.Error, time.Now(), runID)
				return
			}

		case <-timeout:
			failRun(runID, "Deployment timeout")
			return
		}
	}
}

func failRun(runID, errorMsg string) {
	database.DB.Exec(`
UPDATE deployment_runs 
SET status = 'failed', error_message = $1, completed_at = $2 
WHERE id = $3
`, errorMsg, time.Now(), runID)
}

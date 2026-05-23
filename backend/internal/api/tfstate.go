package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"iac-tool/internal/models"

	"github.com/gin-gonic/gin"
)

// stateDir returns the root directory for all tfstate files.
// Defaults to /app/data/tfstates; override with STATE_DIR env var.
func stateDir() string {
	if d := os.Getenv("STATE_DIR"); d != "" {
		return d
	}
	return "/app/data/tfstates"
}

// statePath returns the path to the .tfstate file for a deployment+workspace.
func statePath(deploymentID, workspace string) string {
	return filepath.Join(stateDir(), deploymentID, workspace+".tfstate")
}

// lockPath returns the path to the .tfstate.lock file for a deployment+workspace.
func lockPath(deploymentID, workspace string) string {
	return filepath.Join(stateDir(), deploymentID, workspace+".tfstate.lock")
}

// ensureStateDir creates the per-deployment directory if it does not exist.
func ensureStateDir(deploymentID string) error {
	return os.MkdirAll(filepath.Join(stateDir(), deploymentID), 0o755)
}

// writeFileAtomic writes data to path atomically using a temp file + rename.
// This prevents partial reads if the process is killed mid-write.
func writeFileAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// readSerial extracts the "serial" field from a Terraform state JSON blob.
// Returns 0 if the file does not exist or the field is absent.
func readSerial(path string) (int, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var state map[string]interface{}
	if err := json.Unmarshal(data, &state); err != nil {
		return 0, nil
	}
	if s, ok := state["serial"].(float64); ok {
		return int(s), nil
	}
	return 0, nil
}

// parseDeploymentAndWorkspace extracts the deployment ID and workspace from the
// request. OpenTofu appends ":<workspace>" to the address for named workspaces
// and ":" (with nothing after) for the default workspace, so the colon and any
// suffix end up as part of the :deploymentId path param rather than a query
// string. This helper normalises both cases.
func parseDeploymentAndWorkspace(c *gin.Context) (deploymentID, workspace string) {
	raw := c.Param("deploymentId")
	if idx := strings.Index(raw, ":"); idx != -1 {
		ws := raw[idx+1:]
		deploymentID = raw[:idx]
		if ws == "" {
			workspace = "default"
		} else {
			workspace = ws
		}
	} else {
		deploymentID = raw
		workspace = c.DefaultQuery("workspace", "default")
	}
	return
}

// GetTFState handles GET /api/tfstate/:deploymentId
// Returns the current state JSON, or empty 200 if no state exists yet.
func GetTFState(c *gin.Context) {
	deploymentID, workspace := parseDeploymentAndWorkspace(c)

	data, err := os.ReadFile(statePath(deploymentID, workspace))
	if os.IsNotExist(err) || len(data) == 0 {
		c.Status(http.StatusOK)
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Data(http.StatusOK, "application/json", data)
}

// UpdateTFState handles POST /api/tfstate/:deploymentId
// Stores the new state JSON. Per the Terraform HTTP backend protocol, serial
// conflict detection is handled client-side by Terraform; the backend just
// persists whatever state it receives while the caller holds the lock.
func UpdateTFState(c *gin.Context) {
	deploymentID, workspace := parseDeploymentAndWorkspace(c)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read request body"})
		return
	}

	if err := ensureStateDir(deploymentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := writeFileAtomic(statePath(deploymentID, workspace), body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// DeleteTFState handles DELETE /api/tfstate/:deploymentId
func DeleteTFState(c *gin.Context) {
	deploymentID, workspace := parseDeploymentAndWorkspace(c)

	// Remove state file and lock file; ignore not-found errors.
	os.Remove(statePath(deploymentID, workspace))
	os.Remove(lockPath(deploymentID, workspace))

	c.Status(http.StatusOK)
}

// LockTFState handles LOCK /api/tfstate/:deploymentId
// Returns 200 if the lock is acquired, 423 if already held by a different ID.
func LockTFState(c *gin.Context) {
	deploymentID, workspace := parseDeploymentAndWorkspace(c)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read request body"})
		return
	}

	var incoming models.TFStateLockInfo
	if err := json.Unmarshal(body, &incoming); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lock info"})
		return
	}

	lp := lockPath(deploymentID, workspace)

	// Check for an existing lock.
	existing, err := os.ReadFile(lp)
	if err != nil && !os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err == nil && len(existing) > 0 {
		var existingLock models.TFStateLockInfo
		if json.Unmarshal(existing, &existingLock) == nil && existingLock.ID != incoming.ID {
			// Locked by someone else — return 423 with existing lock info.
			c.Data(http.StatusLocked, "application/json", existing)
			return
		}
	}

	if err := ensureStateDir(deploymentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Stamp the lock with the current time if not already set.
	if incoming.Created.IsZero() {
		incoming.Created = time.Now()
	}
	lockData, err := json.Marshal(incoming)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := writeFileAtomic(lp, lockData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Data(http.StatusOK, "application/json", lockData)
}

// UnlockTFState handles UNLOCK /api/tfstate/:deploymentId
// Validates the lock ID in the request body before removing the lock file.
func UnlockTFState(c *gin.Context) {
	deploymentID, workspace := parseDeploymentAndWorkspace(c)

	lp := lockPath(deploymentID, workspace)

	// Read the incoming lock info (best-effort; empty body = unconditional unlock).
	body, _ := io.ReadAll(c.Request.Body)
	var incoming models.TFStateLockInfo
	hasIncoming := len(body) > 0 && json.Unmarshal(body, &incoming) == nil && incoming.ID != ""

	if hasIncoming {
		// If a lock file exists and belongs to a different ID, reject.
		existing, err := os.ReadFile(lp)
		if err == nil && len(existing) > 0 {
			var existingLock models.TFStateLockInfo
			if json.Unmarshal(existing, &existingLock) == nil && existingLock.ID != incoming.ID {
				c.Data(http.StatusConflict, "application/json", existing)
				return
			}
		}
	}

	os.Remove(lp)
	c.Status(http.StatusOK)
}

// ListAllTFStateDeployments handles GET /api/tfstates
// Returns all deployment IDs that have state files on disk (regardless of DB).
func ListAllTFStateDeployments(c *gin.Context) {
	root := stateDir()
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		c.JSON(http.StatusOK, []string{})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ids := []string{}
	for _, e := range entries {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}
	c.JSON(http.StatusOK, ids)
}

// ListTFStates handles GET /api/deployments/:id/tfstates
// Returns all workspaces for a deployment (used by the UI).
func ListTFStates(c *gin.Context) {
	deploymentID := c.Param("id")
	dir := filepath.Join(stateDir(), deploymentID)

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		c.JSON(http.StatusOK, []models.TFStateSummary{})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	states := []models.TFStateSummary{}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".tfstate") {
			continue
		}
		workspace := strings.TrimSuffix(name, ".tfstate")
		fp := filepath.Join(dir, name)

		serial, _ := readSerial(fp)

		info, _ := entry.Info()
		createdAt := info.ModTime() // best approximation; no ctime on Linux via os.FileInfo
		updatedAt := info.ModTime()

		s := models.TFStateSummary{
			ID:           deploymentID + "/" + workspace,
			DeploymentID: deploymentID,
			Workspace:    workspace,
			StateSerial:  serial,
			CreatedAt:    createdAt,
			UpdatedAt:    updatedAt,
		}

		// Check for a lock file.
		lp := lockPath(deploymentID, workspace)
		if lockData, err := os.ReadFile(lp); err == nil {
			var li models.TFStateLockInfo
			if json.Unmarshal(lockData, &li) == nil {
				s.LockID = &li.ID
				s.LockedAt = &li.Created
			}
		}

		states = append(states, s)
	}

	c.JSON(http.StatusOK, states)
}

// GetTFStateRaw handles GET /api/deployments/:id/tfstates/:workspace/raw
// Returns the raw state JSON for download/inspection in the UI.
func GetTFStateRaw(c *gin.Context) {
	deploymentID := c.Param("id")
	workspace := c.Param("workspace")

	data, err := os.ReadFile(statePath(deploymentID, workspace))
	if os.IsNotExist(err) || len(data) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "no state found for this workspace"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Data(http.StatusOK, "application/json", data)
}

// ForceUnlockTFState handles DELETE /api/deployments/:id/tfstates/:workspace/lock
// Force-removes the lock file (admin action from UI).
func ForceUnlockTFState(c *gin.Context) {
	deploymentID := c.Param("id")
	workspace := c.Param("workspace")

	os.Remove(lockPath(deploymentID, workspace))
	c.JSON(http.StatusOK, gin.H{"message": "lock released"})
}

// DeleteTFStateWorkspace handles DELETE /api/deployments/:id/tfstates/:workspace
// Deletes the state and lock files for a single workspace (admin action from UI).
func DeleteTFStateWorkspace(c *gin.Context) {
	deploymentID := c.Param("id")
	workspace := c.Param("workspace")

	sp := statePath(deploymentID, workspace)
	if _, err := os.Stat(sp); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
		return
	}

	os.Remove(sp)
	os.Remove(lockPath(deploymentID, workspace))

	c.JSON(http.StatusOK, gin.H{"message": "workspace deleted"})
}



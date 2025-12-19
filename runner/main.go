package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DeploymentRequest represents a deployment request
type DeploymentRequest struct {
	Tool        string            `json:"tool" binding:"required"`    // "terraform" or "tofu"
	GitURL      string            `json:"git_url" binding:"required"` // Git repository URL
	GitRef      string            `json:"git_ref" binding:"required"` // Branch, tag, or commit
	Path        string            `json:"path"`                       // Path within repo (default: root)
	EnvVars     map[string]string `json:"env_vars"`                   // Environment variables
	TfvarsFiles []string          `json:"tfvars_files"`               // List of .tfvars files to use
	InitFlags   string            `json:"init_flags"`                 // Custom flags for terraform init
	PlanFlags   string            `json:"plan_flags"`                 // Custom flags for terraform plan
	Timeout     int               `json:"timeout"`                    // Timeout in minutes (default: 60)
	GitAuth     *GitAuth          `json:"git_auth,omitempty"`         // Git authentication
	AutoApprove bool              `json:"auto_approve"`               // Auto-approve terraform apply
}

// GitAuth represents git authentication credentials (HTTPS only)
type GitAuth struct {
	Type     string `json:"type"`               // "https" (SSH not supported)
	Username string `json:"username,omitempty"` // For HTTPS auth
	Password string `json:"password,omitempty"` // For HTTPS auth (token or password)
}

// DeploymentResponse represents the deployment response
type DeploymentResponse struct {
	DeploymentID string `json:"deployment_id"`
	Status       string `json:"status"`
	Message      string `json:"message,omitempty"`
}

// DeploymentStatus represents the current status of a deployment
type DeploymentStatus struct {
	DeploymentID string     `json:"deployment_id"`
	Status       string     `json:"status"` // "running", "success", "failed", "awaiting_approval"
	Phase        string     `json:"phase"`  // "cloning", "init", "plan", "apply"
	StartedAt    time.Time  `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
	Error        string     `json:"error,omitempty"`
	InitLog      string     `json:"init_log,omitempty"`
	PlanLog      string     `json:"plan_log,omitempty"`
	PlanOutput   string     `json:"plan_output,omitempty"`
	ApplyLog     string     `json:"apply_log,omitempty"`
	ApplyOutput  string     `json:"apply_output,omitempty"`
}

// Deployment represents an active deployment
type Deployment struct {
	ID          string
	Request     DeploymentRequest
	Status      DeploymentStatus
	WorkDir     string
	LogChan     chan string
	LogBuffer   []string // Store all logs for late subscribers
	ApproveChan chan bool
	CancelChan  chan bool
	mu          sync.RWMutex
}

var (
	deployments = make(map[string]*Deployment)
	deployMu    sync.RWMutex
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS middleware with configurable origins
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "*"
	}

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigins)
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// Start a new deployment
	r.POST("/deploy", handleDeploy)

	// Get deployment status
	r.GET("/deploy/:id/status", handleStatus)

	// Stream deployment logs
	r.GET("/deploy/:id/logs", handleLogs)

	// Approve deployment (for manual approval before apply)
	r.POST("/deploy/:id/approve", handleApprove)

	// Reject deployment
	r.POST("/deploy/:id/reject", handleReject)

	// Cancel/stop deployment
	r.POST("/deploy/:id/cancel", handleCancel)

	log.Println("Runner HTTP server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func handleDeploy(c *gin.Context) {
	var req DeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	if req.Timeout <= 0 {
		req.Timeout = 60
	}
	if req.Path == "" {
		req.Path = "."
	}

	// Create deployment
	deploymentID := uuid.New().String()
	deployment := &Deployment{
		ID:          deploymentID,
		Request:     req,
		LogChan:     make(chan string, 100),
		LogBuffer:   make([]string, 0),
		ApproveChan: make(chan bool, 1),
		CancelChan:  make(chan bool, 1),
		Status: DeploymentStatus{
			DeploymentID: deploymentID,
			Status:       "running",
			Phase:        "initializing",
			StartedAt:    time.Now(),
		},
	}

	deployMu.Lock()
	deployments[deploymentID] = deployment
	deployMu.Unlock()

	// Start deployment in goroutine
	go executeDeployment(deployment)

	c.JSON(202, DeploymentResponse{
		DeploymentID: deploymentID,
		Status:       "running",
		Message:      "Deployment started",
	})
}

func handleStatus(c *gin.Context) {
	deploymentID := c.Param("id")

	deployMu.RLock()
	deployment, exists := deployments[deploymentID]
	deployMu.RUnlock()

	if !exists {
		c.JSON(404, gin.H{"error": "Deployment not found"})
		return
	}

	deployment.mu.RLock()
	status := deployment.Status
	deployment.mu.RUnlock()

	c.JSON(200, status)
}

func handleLogs(c *gin.Context) {
	deploymentID := c.Param("id")

	deployMu.RLock()
	deployment, exists := deployments[deploymentID]
	deployMu.RUnlock()

	if !exists {
		c.JSON(404, gin.H{"error": "Deployment not found"})
		return
	}

	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	flusher, _ := c.Writer.(http.Flusher)

	// First, send all buffered logs
	deployment.mu.RLock()
	bufferedLogs := make([]string, len(deployment.LogBuffer))
	copy(bufferedLogs, deployment.LogBuffer)
	deployment.mu.RUnlock()

	for _, log := range bufferedLogs {
		c.SSEvent("log", log)
		flusher.Flush()
	}

	// Stream logs
	for {
		select {
		case log, ok := <-deployment.LogChan:
			if !ok {
				return
			}
			c.SSEvent("log", log)
			flusher.Flush()
		case <-c.Request.Context().Done():
			return
		}
	}
}

func handleApprove(c *gin.Context) {
	deploymentID := c.Param("id")

	deployMu.RLock()
	deployment, exists := deployments[deploymentID]
	deployMu.RUnlock()

	if !exists {
		c.JSON(404, gin.H{"error": "Deployment not found"})
		return
	}

	deployment.ApproveChan <- true
	c.JSON(200, gin.H{"message": "Deployment approved"})
}

func handleReject(c *gin.Context) {
	deploymentID := c.Param("id")

	deployMu.RLock()
	deployment, exists := deployments[deploymentID]
	deployMu.RUnlock()

	if !exists {
		c.JSON(404, gin.H{"error": "Deployment not found"})
		return
	}

	deployment.ApproveChan <- false
	c.JSON(200, gin.H{"message": "Deployment rejected"})
}

func handleCancel(c *gin.Context) {
	deploymentID := c.Param("id")

	deployMu.RLock()
	deployment, exists := deployments[deploymentID]
	deployMu.RUnlock()

	if !exists {
		c.JSON(404, gin.H{"error": "Deployment not found"})
		return
	}

	// Send cancel signal
	select {
	case deployment.CancelChan <- true:
		c.JSON(200, gin.H{"message": "Cancellation requested"})
	default:
		c.JSON(400, gin.H{"error": "Deployment already completed or cancelled"})
	}
}

func executeDeployment(deployment *Deployment) {
	defer close(deployment.LogChan)

	// Helper to check for cancellation
	checkCancel := func() bool {
		select {
		case <-deployment.CancelChan:
			deployment.updateStatus("cancelled", "", "Deployment cancelled by user")
			return true
		default:
			return false
		}
	}

	// Create working directory
	workDir := filepath.Join("/tmp/iac-deployments", deployment.ID)
	if err := os.MkdirAll(workDir, 0755); err != nil {
		deployment.updateStatus("failed", "", fmt.Sprintf("Failed to create work directory: %v", err))
		return
	}
	deployment.WorkDir = workDir

	// Schedule cleanup
	defer func() {
		time.AfterFunc(24*time.Hour, func() {
			os.RemoveAll(workDir)
		})
	}()

	// Clone repository
	if checkCancel() {
		return
	}
	deployment.updateStatus("running", "cloning", "")
	deployment.log("Cloning repository...")
	if err := gitClone(deployment); err != nil {
		deployment.updateStatus("failed", "cloning", fmt.Sprintf("Git clone failed: %v", err))
		return
	}

	// Navigate to deployment path
	deployPath := filepath.Join(workDir, deployment.Request.Path)
	if _, err := os.Stat(deployPath); os.IsNotExist(err) {
		deployment.updateStatus("failed", "cloning", fmt.Sprintf("Path does not exist: %s", deployment.Request.Path))
		return
	}

	// Terraform init
	if checkCancel() {
		return
	}
	deployment.updateStatus("running", "init", "")
	deployment.log("Running terraform init...")

	// Parse custom init flags
	var initArgs []string
	if deployment.Request.InitFlags != "" {
		initArgs = parseShellArgs(deployment.Request.InitFlags)
		deployment.log(fmt.Sprintf("Using custom init flags: %s", deployment.Request.InitFlags))
	}

	initLog, err := runTerraformCommand(deployment, deployPath, "init", initArgs)
	deployment.Status.InitLog = initLog
	if err != nil {
		deployment.updateStatus("failed", "init", fmt.Sprintf("Init failed: %v", err))
		return
	}

	// Terraform plan
	if checkCancel() {
		return
	}
	deployment.updateStatus("running", "plan", "")
	deployment.log("Running terraform plan...")
	planArgs := []string{"-out=tfplan"}

	// Add custom plan flags
	if deployment.Request.PlanFlags != "" {
		customFlags := parseShellArgs(deployment.Request.PlanFlags)
		planArgs = append(planArgs, customFlags...)
		deployment.log(fmt.Sprintf("Using custom plan flags: %s", deployment.Request.PlanFlags))
	}

	// Add tfvars files
	for _, tfvarsFile := range deployment.Request.TfvarsFiles {
		planArgs = append(planArgs, "-var-file="+tfvarsFile)
		deployment.log(fmt.Sprintf("Using tfvars file: %s", tfvarsFile))
	}
	planLog, err := runTerraformCommand(deployment, deployPath, "plan", planArgs)
	deployment.Status.PlanLog = planLog
	if err != nil {
		deployment.updateStatus("failed", "plan", fmt.Sprintf("Plan failed: %v", err))
		return
	}

	// Wait for approval if not auto-approved
	if !deployment.Request.AutoApprove {
		deployment.updateStatus("awaiting_approval", "plan", "")
		deployment.log("Waiting for approval...")

		select {
		case approved := <-deployment.ApproveChan:
			if !approved {
				deployment.log("❌ Deployment rejected by user")
				deployment.updateStatus("cancelled", "plan", "")
				return
			}
			deployment.log("✅ Deployment approved, continuing with apply...")
		case <-deployment.CancelChan:
			deployment.log("🛑 Deployment cancelled by user")
			deployment.updateStatus("cancelled", "plan", "")
			return
		case <-time.After(24 * time.Hour):
			deployment.log("⏱️ Approval timeout exceeded")
			deployment.updateStatus("cancelled", "plan", "")
			return
		}
	}

	// Terraform apply
	if checkCancel() {
		return
	}
	deployment.updateStatus("running", "apply", "")
	deployment.log("Running terraform apply...")
	applyLog, err := runTerraformCommand(deployment, deployPath, "apply", []string{"tfplan"})
	deployment.Status.ApplyLog = applyLog
	if err != nil {
		deployment.updateStatus("failed", "apply", fmt.Sprintf("Apply failed: %v", err))
		return
	}

	// Get terraform outputs
	deployment.log("Retrieving outputs...")
	outputLog, err := runTerraformCommand(deployment, deployPath, "output", []string{"-json"})
	if err != nil {
		// Non-fatal if there are no outputs
		deployment.log("No outputs available")
	} else {
		deployment.Status.ApplyOutput = outputLog
	}

	// Success
	deployment.updateStatus("success", "completed", "")
	deployment.log("Deployment completed successfully!")
}

func gitClone(deployment *Deployment) error {
	req := deployment.Request
	args := []string{"clone", "--depth", "1", "--branch", req.GitRef}

	// Add auth if provided
	gitURL := req.GitURL
	if req.GitAuth != nil && req.GitAuth.Type == "http" && req.GitAuth.Username != "" {
		// Inject credentials into URL
		parts := strings.SplitN(gitURL, "://", 2)
		if len(parts) == 2 {
			gitURL = fmt.Sprintf("%s://%s:%s@%s", parts[0], req.GitAuth.Username, req.GitAuth.Password, parts[1])
		}
	}

	args = append(args, gitURL, deployment.WorkDir)

	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		deployment.log(string(output))
		return err
	}

	deployment.log(string(output))
	return nil
}

func runTerraformCommand(deployment *Deployment, workDir, command string, args []string) (string, error) {
	cmdName := deployment.Request.Tool
	if cmdName != "tofu" {
		cmdName = "terraform"
	}

	cmdArgs := []string{command}
	cmdArgs = append(cmdArgs, args...)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(deployment.Request.Timeout)*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)
	cmd.Dir = workDir

	// Set environment variables
	cmd.Env = os.Environ()
	for k, v := range deployment.Request.EnvVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Configure private registry if REGISTRY_HOST is set
	if registryURL := os.Getenv("REGISTRY_HOST"); registryURL != "" {
		terraformrcPath := filepath.Join(workDir, ".terraformrc")
		if err := configureTerraformRegistry(workDir, registryURL); err != nil {
			deployment.log(fmt.Sprintf("Warning: Failed to configure private registry: %v", err))
		} else {
			// Set TF_CLI_CONFIG_FILE environment variable
			cmd.Env = append(cmd.Env, fmt.Sprintf("TF_CLI_CONFIG_FILE=%s", terraformrcPath))
			deployment.log(fmt.Sprintf("✓ Configured private registry: %s", registryURL))
		}
	}

	// Use PTY for colored output
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return "", err
	}
	defer ptmx.Close()

	// Read output and stream to logs
	var output strings.Builder
	scanner := bufio.NewScanner(ptmx)
	for scanner.Scan() {
		line := scanner.Text()
		output.WriteString(line + "\n")
		deployment.log(line)
	}

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		return output.String(), err
	}

	return output.String(), nil
}

func (d *Deployment) updateStatus(status, phase, errorMsg string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.Status.Status = status
	d.Status.Phase = phase
	d.Status.Error = errorMsg

	if status == "success" || status == "failed" || status == "cancelled" {
		now := time.Now()
		d.Status.EndedAt = &now
	}
}

func (d *Deployment) log(message string) {
	// Store in buffer
	d.mu.Lock()
	d.LogBuffer = append(d.LogBuffer, message)
	d.mu.Unlock()

	// Try to send to active listeners
	select {
	case d.LogChan <- message:
	default:
		// Channel full or no listeners, log already in buffer
	}
}

func configureTerraformRegistry(workDir, registryURL string) error {
	// Create .terraformrc in the work directory
	terraformrcPath := filepath.Join(workDir, ".terraformrc")

	// Parse registry URL to get host
	registryHost := strings.TrimPrefix(registryURL, "http://")
	registryHost = strings.TrimPrefix(registryHost, "https://")

	// Fetch token from backend (this token will work for all private namespaces from runner)
	token, err := fetchRegistryToken(registryURL)
	if err != nil {
		// If we can't get the token, continue without auth (for development)
		log.Printf("Warning: Could not fetch registry token: %v", err)
		token = "no-auth"
	}

	// Configure .terraformrc with single credential for the entire registry
	content := fmt.Sprintf(`# Terraform Registry Configuration
# Single credential for both public namespaces and runner access to private namespaces

credentials "%s" {
  token = "%s"
}

# Private Registry Configuration
host "%s" {
  services = {
    "modules.v1"   = "%s/v1/modules/",
    "providers.v1" = "%s/v1/providers/"
  }
}
`, registryHost, token, registryHost, registryURL, registryURL)

	return os.WriteFile(terraformrcPath, []byte(content), 0644)
}

// fetchRegistryToken gets the authentication token from the backend
func fetchRegistryToken(registryURL string) (string, error) {
	resp, err := http.Get(registryURL + "/api/internal/registry-token")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("backend returned status %d", resp.StatusCode)
	}

	var result struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Token, nil
}

// parseShellArgs parses a command-line string into arguments, respecting quotes
func parseShellArgs(s string) []string {
	var args []string
	var current strings.Builder
	var inQuote rune

	for i := 0; i < len(s); i++ {
		c := rune(s[i])

		switch {
		case inQuote != 0:
			// Inside quotes
			if c == inQuote {
				inQuote = 0
			} else if c == '\\' && i+1 < len(s) {
				// Handle escape sequences
				i++
				current.WriteRune(rune(s[i]))
			} else {
				current.WriteRune(c)
			}
		case c == '"' || c == '\'':
			// Start quote
			inQuote = c
		case c == ' ' || c == '\t' || c == '\n':
			// Whitespace - end current arg
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		case c == '\\' && i+1 < len(s):
			// Escape sequence outside quotes
			i++
			current.WriteRune(rune(s[i]))
		default:
			current.WriteRune(c)
		}
	}

	// Add last argument
	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

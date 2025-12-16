package models

import "time"

// Deployment represents an IaC deployment repository
type Deployment struct {
	ID          string    `json:"id"`
	NamespaceID string    `json:"namespace_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	GitURL      string    `json:"git_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DeploymentWithNamespace includes namespace information
type DeploymentWithNamespace struct {
	Deployment
	Namespace string `json:"namespace"`
}

// DeploymentCreate is used for creating a new deployment
type DeploymentCreate struct {
	NamespaceID string  `json:"namespace_id" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description,omitempty"`
	GitURL      string  `json:"git_url" binding:"required"`
	IsPrivate   bool    `json:"is_private,omitempty"`
	GitUsername string  `json:"git_username,omitempty"`
	GitPassword string  `json:"git_password,omitempty"`
}

// GitReference represents a branch or tag
type GitReference struct {
	Name string `json:"name"`
	Type string `json:"type"` // "branch" or "tag"
	SHA  string `json:"sha,omitempty"`
}

// FileNode represents a file or directory in the repository
type FileNode struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Type  string `json:"type"` // "file" or "dir"
	Size  int64  `json:"size,omitempty"`
	IsDir bool   `json:"is_dir"`
}

// DirectoryListing represents the contents of a directory
type DirectoryListing struct {
	Path      string     `json:"path"`
	Files     []FileNode `json:"files"`
	Readme    *string    `json:"readme,omitempty"`
	HasGitOps bool       `json:"has_gitops"` // If directory contains IaC files
}

// DeploymentRun represents an execution of a deployment
type DeploymentRun struct {
	ID           string            `json:"id"`
	DeploymentID string            `json:"deployment_id"`
	Path         string            `json:"path"`
	Ref          string            `json:"ref"`
	Tool         string            `json:"tool"`      // "tofu" or "terraform"
	EnvVars      map[string]string `json:"env_vars"`  // Environment variables
	Status       string            `json:"status"`    // "pending", "initializing", "planning", "awaiting_approval", "applying", "success", "failed", "cancelled"
	InitLog      string            `json:"init_log"`  // Init command output
	PlanLog      string            `json:"plan_log"`  // Plan command output
	ApplyLog     string            `json:"apply_log"` // Apply command output
	ErrorMessage *string           `json:"error_message,omitempty"`
	WorkDir      string            `json:"work_dir"` // Temporary work directory
	ApprovedBy   *string           `json:"approved_by,omitempty"`
	ApprovedAt   *time.Time        `json:"approved_at,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	StartedAt    *time.Time        `json:"started_at,omitempty"`
	CompletedAt  *time.Time        `json:"completed_at,omitempty"`
}

// DeploymentRunCreate is used for creating a new deployment run
type DeploymentRunCreate struct {
	DeploymentID string            `json:"deployment_id" binding:"required"`
	Path         string            `json:"path" binding:"required"`
	Ref          string            `json:"ref" binding:"required"`
	Tool         string            `json:"tool" binding:"required"` // "tofu" or "terraform"
	EnvVars      map[string]string `json:"env_vars,omitempty"`      // Environment variables
}

// DeploymentRunApproval is used for approving/rejecting a plan
type DeploymentRunApproval struct {
	Approved   bool   `json:"approved"`
	ApprovedBy string `json:"approved_by,omitempty"`
}

// DirectoryStatus represents the deployment status for a directory
type DirectoryStatus struct {
	Path        string         `json:"path"`
	LastRun     *DeploymentRun `json:"last_run,omitempty"`
	Status      string         `json:"status"`       // "none", "success", "running", "failed"
	StatusColor string         `json:"status_color"` // "blue", "green", "yellow", "red"
}

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

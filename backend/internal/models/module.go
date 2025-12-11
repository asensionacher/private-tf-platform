package models

import "time"

// Module represents a Terraform module in the registry
type Module struct {
	ID          string    `json:"id"`
	NamespaceID string    `json:"namespace_id"`
	Name        string    `json:"name"`
	Provider    string    `json:"provider"` // e.g., "aws", "azure", "gcp"
	Description *string   `json:"description,omitempty"`
	SourceURL   *string   `json:"source_url,omitempty"` // Optional source repository
	Synced      bool      `json:"synced"`
	SyncError   *string   `json:"sync_error,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ModuleVersion represents a version of a module
type ModuleVersion struct {
	ID            string     `json:"id"`
	ModuleID      string     `json:"module_id"`
	Version       string     `json:"version"`
	DownloadURL   string     `json:"download_url"`
	Documentation *string    `json:"documentation,omitempty"`
	Enabled       bool       `json:"enabled"`
	TagDate       *time.Time `json:"tag_date,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// ModuleCreate is used for creating a new module
type ModuleCreate struct {
	Name        string  `json:"name" binding:"required"`
	Provider    string  `json:"provider" binding:"required"`
	Description *string `json:"description,omitempty"`
	SourceURL   *string `json:"source_url,omitempty"`
}

// ModuleUpdate is used for updating a module
type ModuleUpdate struct {
	Name        *string `json:"name,omitempty"`
	Provider    *string `json:"provider,omitempty"`
	Description *string `json:"description,omitempty"`
	SourceURL   *string `json:"source_url,omitempty"`
}

// ModuleVersionCreate is used for uploading a new module version
type ModuleVersionCreate struct {
	Version       string            `json:"version" binding:"required"`
	DownloadURL   string            `json:"download_url" binding:"required"`
	Documentation *string           `json:"documentation,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"` // Optional headers for download
}

// ModuleVersionUpload is used for uploading module files directly
type ModuleVersionUpload struct {
	Version       string  `json:"version" binding:"required"`
	Documentation *string `json:"documentation,omitempty"`
}

// ModuleWithNamespace includes namespace information
type ModuleWithNamespace struct {
	Module
	Namespace string `json:"namespace"`
}

// Terraform Protocol DTOs

// ModuleVersionsResponse is the response for listing module versions (Terraform protocol)
type ModuleVersionsResponse struct {
	Modules []ModuleVersionsDTO `json:"modules"`
}

type ModuleVersionsDTO struct {
	Versions []ModuleVersionDTO `json:"versions"`
}

type ModuleVersionDTO struct {
	Version string `json:"version"`
}

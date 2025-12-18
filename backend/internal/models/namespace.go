package models

import "time"

// Namespace (Authority) represents an organization/user that owns modules and providers
type Namespace struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	IsPublic    bool      `json:"is_public"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NamespaceCreate is used for creating a new namespace
type NamespaceCreate struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description,omitempty"`
	IsPublic    bool    `json:"is_public"`
}

// NamespaceUpdate is used for updating a namespace
type NamespaceUpdate struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	IsPublic    *bool   `json:"is_public,omitempty"`
}

// APIKey represents an API key for authenticating with the registry
type APIKey struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Key         string     `json:"key,omitempty"` // Only shown on creation
	KeyHash     string     `json:"-"`             // Stored in DB
	Permissions string     `json:"permissions"`   // "read", "write", "admin"
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
}

// APIKeyCreate is used for creating a new API key
type APIKeyCreate struct {
	Name        string     `json:"name" binding:"required"`
	Permissions string     `json:"permissions" binding:"required"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// NamespaceWithStats includes module and provider counts
type NamespaceWithStats struct {
	Namespace
	ModuleCount   int `json:"module_count"`
	ProviderCount int `json:"provider_count"`
}

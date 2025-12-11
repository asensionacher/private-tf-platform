package models

import "time"

// Provider represents a Terraform provider in the registry
type Provider struct {
	ID          string    `json:"id"`
	NamespaceID string    `json:"namespace_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	SourceURL   *string   `json:"source_url,omitempty"`
	Synced      bool      `json:"synced"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProviderVersion represents a version of a provider
type ProviderVersion struct {
	ID         string             `json:"id"`
	ProviderID string             `json:"provider_id"`
	Version    string             `json:"version"`
	Protocols  []string           `json:"protocols"`
	Enabled    bool               `json:"enabled"`
	Platforms  []ProviderPlatform `json:"platforms,omitempty"`
	TagDate    *time.Time         `json:"tag_date,omitempty"`
	CreatedAt  time.Time          `json:"created_at"`
}

// ProviderPlatform represents a platform-specific binary for a provider version
type ProviderPlatform struct {
	ID               string `json:"id"`
	VersionID        string `json:"version_id"`
	OS               string `json:"os"`
	Arch             string `json:"arch"`
	Filename         string `json:"filename"`
	DownloadURL      string `json:"download_url"`
	SHASumsURL       string `json:"shasums_url,omitempty"`
	SHASumsSignature string `json:"shasums_signature_url,omitempty"`
	SHASum           string `json:"shasum"`
	SigningKeys      string `json:"signing_keys,omitempty"`
}

// ProviderCreate is used for creating a new provider
type ProviderCreate struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description,omitempty"`
}

// ProviderVersionCreate is used for creating a new provider version
type ProviderVersionCreate struct {
	Version   string   `json:"version" binding:"required"`
	Protocols []string `json:"protocols"`
}

// ProviderPlatformCreate is used for adding a platform to a provider version
type ProviderPlatformCreate struct {
	OS               string `json:"os" binding:"required"`
	Arch             string `json:"arch" binding:"required"`
	Filename         string `json:"filename" binding:"required"`
	DownloadURL      string `json:"download_url" binding:"required"`
	SHASumsURL       string `json:"shasums_url,omitempty"`
	SHASumsSignature string `json:"shasums_signature_url,omitempty"`
	SHASum           string `json:"shasum" binding:"required"`
	SigningKeys      string `json:"signing_keys,omitempty"`
}

// ProviderWithNamespace includes namespace information
type ProviderWithNamespace struct {
	Provider
	Namespace string `json:"namespace"`
}

// Terraform Protocol DTOs

// ProviderVersionsResponse is the response for listing provider versions (Terraform protocol)
type ProviderVersionsResponse struct {
	Versions []ProviderVersionDTO `json:"versions"`
}

type ProviderVersionDTO struct {
	Version   string                `json:"version"`
	Protocols []string              `json:"protocols"`
	Platforms []ProviderPlatformDTO `json:"platforms"`
}

type ProviderPlatformDTO struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// ProviderDownloadResponse is the response for downloading a provider (Terraform protocol)
type ProviderDownloadResponse struct {
	Protocols           []string     `json:"protocols"`
	OS                  string       `json:"os"`
	Arch                string       `json:"arch"`
	Filename            string       `json:"filename"`
	DownloadURL         string       `json:"download_url"`
	SHASumsURL          string       `json:"shasums_url,omitempty"`
	SHASumsSignatureURL string       `json:"shasums_signature_url,omitempty"`
	SHASum              string       `json:"shasum"`
	SigningKeys         *SigningKeys `json:"signing_keys,omitempty"`
}

type SigningKeys struct {
	GPGPublicKeys []GPGPublicKey `json:"gpg_public_keys"`
}

type GPGPublicKey struct {
	KeyID      string `json:"key_id"`
	ASCIIArmor string `json:"ascii_armor"`
}

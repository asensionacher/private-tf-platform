// Namespace (Authority) - organization that owns modules and providers
export interface Namespace {
  id: string;
  name: string;
  description?: string;
  is_public: boolean;
  module_count?: number;
  provider_count?: number;
  created_at: string;
  updated_at: string;
}

export interface NamespaceCreate {
  name: string;
  description?: string;
  is_public: boolean;
}

// API Key for authentication
export interface APIKey {
  id: string;
  namespace_id: string;
  name: string;
  key?: string; // Only shown on creation
  permissions: 'read' | 'write' | 'admin';
  expires_at?: string;
  created_at: string;
  last_used_at?: string;
}

export interface APIKeyCreate {
  name: string;
  permissions: 'read' | 'write' | 'admin';
  expires_at?: string;
}

// Module in the registry
export interface Module {
  id: string;
  namespace_id: string;
  namespace: string;
  name: string;
  provider: string; // e.g., "aws", "azure", "gcp"
  description?: string;
  source_url?: string;
  synced: boolean;
  sync_error?: string;
  created_at: string;
  updated_at: string;
}

export interface ModuleCreate {
  name: string;
  provider: string;
  description?: string;
  source_url?: string;
}

// Create module from Git repository
export interface ModuleFromGitCreate {
  namespace_id: string;
  name: string;
  provider: string;
  git_url: string;
  description?: string;
  subdir?: string;
}

// Add version to existing module
export interface ModuleVersionAdd {
  version: string;
  enabled?: boolean;
  subdir?: string;
}

export interface ModuleVersion {
  id: string;
  module_id: string;
  version: string;
  download_url: string;
  documentation?: string;
  enabled: boolean;
  created_at: string;
}

// Git tag from repository
export interface GitTag {
  name: string;
  version: string;
}

export interface ModuleVersionCreate {
  version: string;
  download_url: string;
  documentation?: string;
}

// Provider in the registry
export interface Provider {
  id: string;
  namespace_id: string;
  namespace: string;
  name: string;
  description?: string;
  source_url?: string;
  synced: boolean;
  created_at: string;
  updated_at: string;
}

export interface ProviderCreate {
  name: string;
  description?: string;
}

// Create provider from Git repository
export interface ProviderFromGitCreate {
  namespace_id: string;
  name: string;
  git_url: string;
  description?: string;
}

export interface ProviderVersion {
  id: string;
  provider_id: string;
  version: string;
  protocols: string[];
  enabled: boolean;
  platforms?: ProviderPlatform[];
  created_at: string;
}

export interface ProviderPlatform {
  id: string;
  version_id: string;
  os: string;
  arch: string;
  filename: string;
  download_url: string;
  shasums_url?: string;
  shasums_signature_url?: string;
  shasum: string;
  signing_keys?: string;
}

export interface ProviderPlatformCreate {
  os: string;
  arch: string;
  filename: string;
  download_url: string;
  shasums_url?: string;
  shasums_signature_url?: string;
  shasum: string;
  signing_keys?: string;
}

export interface ProviderVersionCreate {
  version: string;
  protocols?: string[];
  platforms: ProviderPlatformCreate[];
}

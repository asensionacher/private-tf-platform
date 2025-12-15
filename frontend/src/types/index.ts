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
  is_private?: boolean;
  git_auth_type?: 'ssh' | 'https';
  git_username?: string;
  git_password?: string;
  git_ssh_key?: string;
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
  is_private?: boolean;
  git_auth_type?: 'ssh' | 'https';
  git_username?: string;
  git_password?: string;
  git_ssh_key?: string;
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
}

// Deployment for IaC management
export interface Deployment {
  id: string;
  namespace_id: string;
  namespace?: string;
  name: string;
  description?: string;
  git_url: string;
  is_private: boolean;
  created_at: string;
  updated_at: string;
}

export interface DeploymentCreate {
  namespace_id: string;
  name: string;
  description?: string;
  git_url: string;
  is_private?: boolean;
  git_username?: string;
  git_password?: string;
}

export interface GitReference {
  name: string;
  type: 'branch' | 'tag';
  sha: string;
}

export interface FileNode {
  name: string;
  path: string;
  type: string;
  size: number;
  is_dir: boolean;
}

export interface DirectoryListing {
  path: string;
  files: FileNode[];
  readme?: string;
  has_gitops: boolean;
  shasums_url?: string;
  shasums_signature_url?: string;
  shasum: string;
  signing_keys?: string;
}

export interface DeploymentRun {
  id: string;
  deployment_id: string;
  path: string;
  ref: string;
  tool: 'terraform' | 'tofu';
  env_vars: Record<string, string>;
  status: 'pending' | 'initializing' | 'planning' | 'awaiting_approval' | 'applying' | 'success' | 'failed' | 'cancelled';
  init_log: string;
  plan_log: string;
  apply_log: string;
  error_message?: string;
  work_dir: string;
  approved_by?: string;
  approved_at?: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
}

export interface DeploymentRunCreate {
  deployment_id: string;
  path: string;
  ref: string;
  tool: 'terraform' | 'tofu';
  env_vars?: Record<string, string>;
}

export interface DeploymentRunApproval {
  approved: boolean;
  approved_by?: string;
}

export interface DirectoryStatus {
  path: string;
  last_run?: DeploymentRun;
  status: 'none' | 'success' | 'running' | 'failed' | 'initializing' | 'planning' | 'awaiting_approval' | 'applying' | 'cancelled';
  status_color: 'blue' | 'green' | 'yellow' | 'red';
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

import axios from 'axios';
import type {
  Namespace,
  NamespaceCreate,
  APIKey,
  APIKeyCreate,
  Module,
  ModuleCreate,
  ModuleFromGitCreate,
  ModuleVersion,
  GitTag,
  Provider,
  ProviderFromGitCreate,
  ProviderVersion,
  ProviderPlatform,
  ProviderPlatformCreate,
  TFStateSummary
} from '../types';

// Use environment variable for API base URL, fallback to relative path
const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || '/api';

const api = axios.create({
  baseURL: apiBaseUrl,
});

// No authentication needed for management API
// API keys are only required for Terraform protocol endpoints

// Namespaces API
export const namespacesApi = {
  getAll: () => api.get<Namespace[]>('/namespaces').then(res => res.data || []),
  getById: (id: string) => api.get<Namespace>(`/namespaces/${id}`).then(res => res.data),
  create: (data: NamespaceCreate) => api.post<Namespace>('/namespaces', data).then(res => res.data),
  update: (id: string, data: Partial<NamespaceCreate>) => api.patch<Namespace>(`/namespaces/${id}`, data).then(res => res.data),
  delete: (id: string) => api.delete(`/namespaces/${id}`).then(res => res.data),

  // API Keys (for Terraform CLI access)
  getAPIKeys: () => api.get<APIKey[]>('/api-keys').then(res => res.data || []),
  createAPIKey: (_namespaceId: string, data: APIKeyCreate) => api.post<APIKey>('/api-keys', data).then(res => res.data),
  deleteAPIKey: (_namespaceId: string, keyId: string) => api.delete(`/api-keys/${keyId}`).then(res => res.data),
};

// Modules API
export const modulesApi = {
  getAll: (namespace?: string) => {
    const params = namespace ? { namespace } : {};
    return api.get<Module[]>('/modules', { params }).then(res => res.data || []);
  },
  getById: (id: string) => api.get<Module>(`/modules/${id}`).then(res => res.data),
  create: (data: ModuleFromGitCreate) => api.post<Module>('/modules', data).then(res => res.data),
  update: (id: string, data: Partial<ModuleCreate>) => api.put<Module>(`/modules/${id}`, data).then(res => res.data),
  delete: (id: string) => api.delete(`/modules/${id}`).then(res => res.data),
  getVersions: (id: string) => api.get<ModuleVersion[]>(`/modules/${id}/versions`).then(res => res.data || []),
  getGitTags: (id: string) => api.get<GitTag[]>(`/modules/${id}/git-tags`).then(res => res.data || []),
  getReadme: (id: string, ref?: string) => {
    const params = ref ? { ref } : {};
    return api.get<{ content: string }>(`/modules/${id}/readme`, { params }).then(res => res.data);
  },
  syncTags: (id: string) => api.post<{ message: string; tags_found: number; tags_added: number }>(`/modules/${id}/sync-tags`).then(res => res.data),
  addVersion: (id: string, data: { version: string; enabled?: boolean; subdir?: string }) =>
    api.post<ModuleVersion>(`/modules/${id}/versions`, data).then(res => res.data),
  toggleVersion: (id: string, versionId: string, enabled: boolean) =>
    api.patch<{ message: string; enabled: boolean }>(`/modules/${id}/versions/${versionId}`, { enabled }).then(res => res.data),
  deleteVersion: (id: string, versionId: string) =>
    api.delete(`/modules/${id}/versions/${versionId}`).then(res => res.data),
};

// Providers API
export const providersApi = {
  getAll: (namespace?: string) => {
    const params = namespace ? { namespace } : {};
    return api.get<Provider[]>('/providers', { params }).then(res => res.data || []);
  },
  getById: (id: string) => api.get<Provider>(`/providers/${id}`).then(res => res.data),
  create: (data: ProviderFromGitCreate) => api.post<Provider>('/providers', data).then(res => res.data),
  delete: (id: string) => api.delete(`/providers/${id}`).then(res => res.data),
  getVersions: (id: string) => api.get<ProviderVersion[]>(`/providers/${id}/versions`).then(res => res.data || []),
  getGitTags: (id: string) => api.get<GitTag[]>(`/providers/${id}/git-tags`).then(res => res.data || []),
  getReadme: (id: string, ref?: string) => {
    const params = ref ? { ref } : {};
    return api.get<{ content: string }>(`/providers/${id}/readme`, { params }).then(res => res.data);
  },
  syncTags: (id: string) => api.post<{ message: string; tags_found: number; tags_added: number }>(`/providers/${id}/sync-tags`).then(res => res.data),
  addVersion: (id: string, data: { version: string; protocols?: string[] }) =>
    api.post<ProviderVersion>(`/providers/${id}/versions`, data).then(res => res.data),
  toggleVersion: (id: string, versionId: string, enabled: boolean) =>
    api.patch<{ message: string; enabled: boolean }>(`/providers/${id}/versions/${versionId}`, { enabled }).then(res => res.data),
  deleteVersion: (id: string, versionId: string) =>
    api.delete(`/providers/${id}/versions/${versionId}`).then(res => res.data),

  // Platforms (binaries per OS/arch)
  getPlatforms: (id: string, versionId: string) =>
    api.get<ProviderPlatform[]>(`/providers/${id}/versions/${versionId}/platforms`).then(res => res.data || []),
  addPlatform: (id: string, versionId: string, data: ProviderPlatformCreate) =>
    api.post<ProviderPlatform>(`/providers/${id}/versions/${versionId}/platforms`, data).then(res => res.data),
  deletePlatform: (id: string, versionId: string, platformId: string) =>
    api.delete(`/providers/${id}/versions/${versionId}/platforms/${platformId}`).then(res => res.data),

  // Upload platform binary (zip file)
  uploadPlatform: (id: string, versionId: string, os: string, arch: string, file: File) => {
    const formData = new FormData();
    formData.append('os', os);
    formData.append('arch', arch);
    formData.append('file', file);
    return api.post<{ message: string; platform_id: string; os: string; arch: string; filename: string; shasum: string; download_url: string }>(
      `/providers/${id}/versions/${versionId}/platforms/upload`,
      formData,
      { headers: { 'Content-Type': 'multipart/form-data' } }
    ).then(res => res.data);
  },
};

// TF State management (for UI inspection)
export const tfStateApi = {
  listDeployments: () =>
    api.get<string[]>('/tfstates').then(res => res.data || []),
  getWorkspaces: (id: string) =>
    api.get<TFStateSummary[]>(`/deployments/${id}/tfstates`).then(res => res.data || []),
  getRaw: (id: string, workspace: string) =>
    api.get<unknown>(`/deployments/${id}/tfstates/${workspace}/raw`).then(res => res.data),
  forceUnlock: (id: string, workspace: string) =>
    api.delete(`/deployments/${id}/tfstates/${workspace}/lock`).then(res => res.data),
  deleteWorkspace: (id: string, workspace: string) =>
    api.delete(`/deployments/${id}/tfstates/${workspace}`).then(res => res.data),
};

export default api;

import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { Package, Box, ChevronRight, Plus, X, GitBranch, AlertCircle, RefreshCw, Trash2 } from 'lucide-react';
import { modulesApi, namespacesApi } from '../api';
import type { Module, ModuleFromGitCreate, Namespace } from '../types';

export default function ModulesPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [formData, setFormData] = useState<Partial<ModuleFromGitCreate>>({
    name: '',
    provider: '',
    git_url: '',
    description: '',
    subdir: '',
  });
  const [error, setError] = useState<string | null>(null);
  const [notification, setNotification] = useState<{ message: string; type: 'success' | 'error' } | null>(null);

  const { data: modulesData, isLoading } = useQuery({
    queryKey: ['modules'],
    queryFn: () => modulesApi.getAll(),
  });

  const modules = Array.isArray(modulesData) ? modulesData : [];

  // Show notification with auto-dismiss
  const showNotification = (message: string, type: 'success' | 'error' = 'success') => {
    setNotification({ message, type });
    setTimeout(() => setNotification(null), 4000);
  };

  // Check if there are any modules still syncing
  const hasSyncingModules = modules.some(m => !m.synced);

  // Auto-refresh when there are syncing modules
  useEffect(() => {
    if (hasSyncingModules) {
      const interval = setInterval(() => {
        queryClient.invalidateQueries({ queryKey: ['modules'] });
      }, 3000); // Check every 3 seconds

      return () => clearInterval(interval);
    }
  }, [hasSyncingModules, queryClient]);

  const { data: namespacesData } = useQuery({
    queryKey: ['namespaces'],
    queryFn: () => namespacesApi.getAll(),
  });

  const createMutation = useMutation({
    mutationFn: (data: ModuleFromGitCreate) => modulesApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['modules'] });
      setShowCreateModal(false);
      setFormData({
        name: '',
        provider: '',
        git_url: '',
        description: '',
        subdir: '',
      });
      setError(null);
    },
    onError: (err: any) => {
      setError(err.response?.data?.error || 'Failed to create module');
    },
  });

  const syncMutation = useMutation({
    mutationFn: (moduleId: string) => modulesApi.syncTags(moduleId),
    onSuccess: (_, moduleId) => {
      queryClient.invalidateQueries({ queryKey: ['modules'] });
      const module = modules.find(m => m.id === moduleId);
      showNotification(`Module "${module?.name || 'Unknown'}" sync started successfully`, 'success');
    },
    onError: (err: any, moduleId) => {
      const module = modules.find(m => m.id === moduleId);
      showNotification(`Failed to sync module "${module?.name || 'Unknown'}": ${err.response?.data?.error || err.message}`, 'error');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (moduleId: string) => modulesApi.delete(moduleId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['modules'] });
    },
  });

  const handleRetrySync = (e: React.MouseEvent, moduleId: string) => {
    e.stopPropagation();
    syncMutation.mutate(moduleId);
  };

  const handleDelete = (e: React.MouseEvent, moduleId: string) => {
    e.stopPropagation();
    if (confirm('Are you sure you want to delete this module? This action cannot be undone.')) {
      deleteMutation.mutate(moduleId);
    }
  };

  const namespaces = Array.isArray(namespacesData) ? namespacesData : [];

  // Group modules by namespace
  const modulesByNamespace = modules.reduce((acc, mod) => {
    const ns = mod.namespace || 'default';
    if (!acc[ns]) acc[ns] = [];
    acc[ns].push(mod);
    return acc;
  }, {} as Record<string, Module[]>);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.namespace_id || !formData.name || !formData.provider || !formData.git_url) {
      setError('Please fill in all required fields');
      return;
    }
    createMutation.mutate(formData as ModuleFromGitCreate);
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600"></div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Notification Toast */}
      {notification && (
        <div
          className={`fixed top-4 right-4 z-50 px-6 py-4 rounded-lg shadow-lg border transition-all duration-300 ${notification.type === 'success'
            ? 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800 text-green-800 dark:text-green-200'
            : 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800 text-red-800 dark:text-red-200'
            }`}
        >
          <div className="flex items-center gap-3">
            {notification.type === 'success' ? (
              <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
              </svg>
            ) : (
              <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
              </svg>
            )}
            <p className="font-medium">{notification.message}</p>
            <button
              onClick={() => setNotification(null)}
              className="ml-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
            >
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
              </svg>
            </button>
          </div>
        </div>
      )}

      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Modules</h1>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Terraform modules available in the registry
          </p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700"
        >
          <Plus className="h-4 w-4 mr-2" />
          Add Module
        </button>
      </div>

      {modules.length === 0 ? (
        <div className="text-center py-12 bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
          <Package className="mx-auto h-12 w-12 text-gray-400" />
          <h3 className="mt-2 text-sm font-semibold text-gray-900 dark:text-white">No modules</h3>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Click "Add Module" to add your first Terraform module from a Git repository
          </p>
        </div>
      ) : (
        <div className="space-y-8">
          {Object.entries(modulesByNamespace).map(([namespace, mods]) => (
            <div key={namespace}>
              <h2 className="text-lg font-semibold text-gray-700 dark:text-gray-300 mb-4">
                {namespace}
              </h2>
              <div className="bg-white dark:bg-gray-800 shadow rounded-lg overflow-hidden">
                <ul className="divide-y divide-gray-200 dark:divide-gray-700">
                  {mods.map((mod) => (
                    <li
                      key={mod.id}
                      className={`transition-colors ${mod.synced && !mod.sync_error
                        ? 'hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer'
                        : ''
                        }`}
                      onClick={() => (mod.synced && !mod.sync_error) && navigate(`/modules/${mod.id}`)}
                    >
                      <div className="px-4 py-4 sm:px-6">
                        <div className="flex items-start justify-between">
                          <div className="flex items-start flex-1">
                            <Box className={`h-8 w-8 mt-0.5 ${mod.sync_error ? 'text-red-500' :
                              mod.synced ? 'text-indigo-500' : 'text-gray-400'
                              }`} />
                            <div className="ml-4 flex-1">
                              <div className="flex items-center gap-2 flex-wrap">
                                <p className={`text-sm font-medium ${mod.sync_error ? 'text-red-600 dark:text-red-400' :
                                  mod.synced ? 'text-indigo-600 dark:text-indigo-400'
                                    : 'text-gray-400 dark:text-gray-500'
                                  }`}>
                                  {mod.namespace}/{mod.name}/{mod.provider}
                                </p>
                                {!mod.synced && !mod.sync_error && (
                                  <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-400">
                                    <svg className="animate-spin -ml-0.5 mr-1.5 h-3 w-3" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                                    </svg>
                                    Syncing tags...
                                  </span>
                                )}
                                {mod.sync_error && (
                                  <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-400">
                                    <AlertCircle className="h-3 w-3 mr-1" />
                                    Sync failed
                                  </span>
                                )}
                              </div>
                              <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                                {mod.description || 'No description'}
                              </p>
                              {mod.source_url && (
                                <p className="text-xs text-gray-400 dark:text-gray-500 flex items-center mt-1">
                                  <GitBranch className="h-3 w-3 mr-1" />
                                  {mod.source_url}
                                </p>
                              )}
                              {mod.sync_error && (
                                <div className="mt-2 p-2 bg-red-50 dark:bg-red-900/10 border border-red-200 dark:border-red-800 rounded text-xs">
                                  <p className="text-red-600 dark:text-red-400 font-medium mb-1">Error details:</p>
                                  <p className="text-red-600 dark:text-red-400">{mod.sync_error}</p>
                                  <div className="mt-2 flex gap-2">
                                    <button
                                      onClick={(e) => handleRetrySync(e, mod.id)}
                                      disabled={syncMutation.isPending}
                                      className="inline-flex items-center px-2 py-1 text-xs font-medium text-red-700 dark:text-red-300 bg-red-100 dark:bg-red-900/20 border border-red-300 dark:border-red-700 rounded hover:bg-red-200 dark:hover:bg-red-900/30 disabled:opacity-50"
                                    >
                                      <RefreshCw className={`h-3 w-3 mr-1 ${syncMutation.isPending ? 'animate-spin' : ''}`} />
                                      {syncMutation.isPending ? 'Retrying...' : 'Retry Sync'}
                                    </button>
                                    <button
                                      onClick={(e) => handleDelete(e, mod.id)}
                                      disabled={deleteMutation.isPending}
                                      className="inline-flex items-center px-2 py-1 text-xs font-medium text-red-700 dark:text-red-300 bg-red-100 dark:bg-red-900/20 border border-red-300 dark:border-red-700 rounded hover:bg-red-200 dark:hover:bg-red-900/30 disabled:opacity-50"
                                    >
                                      <Trash2 className="h-3 w-3 mr-1" />
                                      {deleteMutation.isPending ? 'Deleting...' : 'Delete'}
                                    </button>
                                  </div>
                                </div>
                              )}
                              {!mod.synced && !mod.sync_error && (
                                <div className="mt-2">
                                  <button
                                    onClick={(e) => handleDelete(e, mod.id)}
                                    disabled={deleteMutation.isPending}
                                    className="inline-flex items-center px-2 py-1 text-xs font-medium text-gray-600 dark:text-gray-400 bg-gray-100 dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded hover:bg-gray-200 dark:hover:bg-gray-600 disabled:opacity-50"
                                  >
                                    <Trash2 className="h-3 w-3 mr-1" />
                                    {deleteMutation.isPending ? 'Deleting...' : 'Cancel & Delete'}
                                  </button>
                                </div>
                              )}
                            </div>
                          </div>
                          {mod.synced && !mod.sync_error && <ChevronRight className="h-5 w-5 text-gray-400" />}
                        </div>
                      </div>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Create Module Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 overflow-y-auto">
          <div className="flex min-h-screen items-center justify-center p-4">
            <div className="fixed inset-0 bg-black/50" onClick={() => setShowCreateModal(false)} />
            <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-lg w-full p-6">
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-lg font-semibold text-gray-900 dark:text-white flex items-center">
                  <GitBranch className="h-5 w-5 mr-2 text-indigo-500" />
                  Add Module from Git Repository
                </h2>
                <button
                  onClick={() => setShowCreateModal(false)}
                  className="text-gray-400 hover:text-gray-500"
                >
                  <X className="h-5 w-5" />
                </button>
              </div>

              {error && (
                <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
                  <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
                </div>
              )}

              <form onSubmit={handleSubmit} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Namespace *
                  </label>
                  <select
                    value={formData.namespace_id || ''}
                    onChange={(e) => setFormData({ ...formData, namespace_id: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-indigo-500 focus:border-indigo-500"
                    required
                  >
                    <option value="">Select a namespace</option>
                    {namespaces.map((ns: Namespace) => (
                      <option key={ns.id} value={ns.id}>
                        {ns.name}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                      Module Name *
                    </label>
                    <input
                      type="text"
                      value={formData.name || ''}
                      onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                      placeholder="e.g., vpc, eks, rds"
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-indigo-500 focus:border-indigo-500"
                      required
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                      Provider *
                    </label>
                    <input
                      type="text"
                      value={formData.provider || ''}
                      onChange={(e) => setFormData({ ...formData, provider: e.target.value })}
                      placeholder="e.g., aws, azure, gcp"
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-indigo-500 focus:border-indigo-500"
                      required
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Git Repository URL *
                  </label>
                  <input
                    type="text"
                    value={formData.git_url || ''}
                    onChange={(e) => setFormData({ ...formData, git_url: e.target.value })}
                    placeholder="https://github.com/org/terraform-module.git"
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-indigo-500 focus:border-indigo-500"
                    required
                  />
                  <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    HTTPS or SSH Git URL (e.g., https://github.com/org/repo.git or git@github.com:org/repo.git)
                  </p>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Subdirectory
                  </label>
                  <input
                    type="text"
                    value={formData.subdir || ''}
                    onChange={(e) => setFormData({ ...formData, subdir: e.target.value })}
                    placeholder="modules/vpc"
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-indigo-500 focus:border-indigo-500"
                  />
                  <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    Optional: path within repo if module is not at root
                  </p>
                </div>

                <div>
                  <label className="flex items-center space-x-2">
                    <input
                      type="checkbox"
                      checked={formData.is_private || false}
                      onChange={(e) => setFormData({ ...formData, is_private: e.target.checked })}
                      className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
                    />
                    <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                      Private Repository (requires HTTPS authentication)
                    </span>
                  </label>
                </div>

                {formData.is_private && (
                  <>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                        Username
                      </label>
                      <input
                        type="text"
                        value={formData.git_username || ''}
                        onChange={(e) => setFormData({ ...formData, git_username: e.target.value })}
                        placeholder="your-username or token"
                        className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-indigo-500 focus:border-indigo-500"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                        Password / Personal Access Token
                      </label>
                      <input
                        type="password"
                        value={formData.git_password || ''}
                        onChange={(e) => setFormData({ ...formData, git_password: e.target.value })}
                        placeholder="•••••••••••••"
                        className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-indigo-500 focus:border-indigo-500"
                        required={formData.is_private}
                      />
                      <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                        For Azure DevOps/GitHub/GitLab, use a Personal Access Token
                      </p>
                    </div>
                  </>
                )}

                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Description
                  </label>
                  <textarea
                    value={formData.description || ''}
                    onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                    placeholder="A brief description of what this module does"
                    rows={2}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-indigo-500 focus:border-indigo-500"
                  />
                </div>

                <div className="flex justify-end gap-3 pt-4">
                  <button
                    type="button"
                    onClick={() => setShowCreateModal(false)}
                    className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-600"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={createMutation.isPending}
                    className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 border border-transparent rounded-md hover:bg-indigo-700 disabled:opacity-50"
                  >
                    {createMutation.isPending ? 'Creating...' : 'Create Module'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

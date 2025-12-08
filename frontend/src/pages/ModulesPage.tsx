import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { Package, Box, ChevronRight, Plus, X, GitBranch } from 'lucide-react';
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

  const { data: modulesData, isLoading } = useQuery({
    queryKey: ['modules'],
    queryFn: () => modulesApi.getAll(),
  });

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

  const modules = Array.isArray(modulesData) ? modulesData : [];
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
                      className="hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer transition-colors"
                      onClick={() => navigate(`/modules/${mod.id}`)}
                    >
                      <div className="px-4 py-4 sm:px-6 flex items-center justify-between">
                        <div className="flex items-center">
                          <Box className="h-8 w-8 text-indigo-500" />
                          <div className="ml-4">
                            <p className="text-sm font-medium text-indigo-600 dark:text-indigo-400">
                              {mod.namespace}/{mod.provider}/{mod.name}
                            </p>
                            <p className="text-sm text-gray-500 dark:text-gray-400">
                              {mod.description || 'No description'}
                            </p>
                            {mod.source_url && (
                              <p className="text-xs text-gray-400 dark:text-gray-500 flex items-center mt-1">
                                <GitBranch className="h-3 w-3 mr-1" />
                                {mod.source_url}
                              </p>
                            )}
                          </div>
                        </div>
                        <ChevronRight className="h-5 w-5 text-gray-400" />
                      </div>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Usage instructions */}
      <div className="bg-blue-50 dark:bg-blue-900/20 rounded-lg p-4 mt-8">
        <h3 className="text-sm font-medium text-blue-800 dark:text-blue-200 mb-2">
          Usage in Terraform
        </h3>
        <code className="text-xs text-blue-700 dark:text-blue-300 block">
          module "example" {'{'}
          <br />
          &nbsp;&nbsp;source  = "localhost:9080/NAMESPACE/NAME/PROVIDER"
          <br />
          &nbsp;&nbsp;version = "1.0.0"
          <br />
          {'}'}
        </code>
      </div>

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

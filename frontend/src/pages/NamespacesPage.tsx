import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Building2, Plus, Trash2 } from 'lucide-react';
import { namespacesApi } from '../api';
import type { NamespaceCreate } from '../types';

export default function NamespacesPage() {
  const queryClient = useQueryClient();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [formData, setFormData] = useState<NamespaceCreate>({
    name: '',
    description: '',
    is_public: false,
  });

  const { data: namespacesData, isLoading } = useQuery({
    queryKey: ['namespaces'],
    queryFn: () => namespacesApi.getAll(),
  });

  const namespaces = Array.isArray(namespacesData) ? namespacesData : [];

  const createMutation = useMutation({
    mutationFn: namespacesApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['namespaces'] });
      setIsModalOpen(false);
      setFormData({ name: '', description: '', is_public: false });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: namespacesApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['namespaces'] });
    },
  });

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
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Namespaces</h1>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Manage namespaces (organizations) for modules and providers
          </p>
        </div>
        <button
          onClick={() => setIsModalOpen(true)}
          className="flex items-center gap-2 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700"
        >
          <Plus className="h-4 w-4" />
          New Namespace
        </button>
      </div>

      <div className="bg-white dark:bg-gray-800 shadow rounded-lg overflow-hidden">
        {namespaces.length === 0 ? (
          <div className="text-center py-12">
            <Building2 className="mx-auto h-12 w-12 text-gray-400" />
            <h3 className="mt-2 text-sm font-semibold text-gray-900 dark:text-white">No namespaces</h3>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              Create a namespace to organize your modules and providers
            </p>
          </div>
        ) : (
          <ul className="divide-y divide-gray-200 dark:divide-gray-700">
            {namespaces.map((ns) => (
              <li
                key={ns.id}
                className="hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
              >
                <div className="px-4 py-4 sm:px-6 flex items-center justify-between">
                  <div className="flex items-center flex-1">
                    <Building2 className="h-8 w-8 text-gray-400" />
                    <div className="ml-4">
                      <p className="text-sm font-medium text-gray-900 dark:text-white">
                        {ns.name}
                        {ns.is_public && (
                          <span className="ml-2 px-2 py-0.5 text-xs bg-green-100 text-green-800 rounded-full">
                            Public
                          </span>
                        )}
                      </p>
                      <p className="text-sm text-gray-500 dark:text-gray-400">
                        {ns.description || 'No description'}
                      </p>
                      <div className="mt-1 flex gap-4 text-xs text-gray-500">
                        <span>{ns.module_count || 0} modules</span>
                        <span>{ns.provider_count || 0} providers</span>
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    {ns.name !== 'default' && (
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          if (confirm(`Delete namespace "${ns.name}"?`)) {
                            deleteMutation.mutate(ns.id);
                          }
                        }}
                        className="p-2 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 rounded"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    )}
                  </div>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>

      {/* Create Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-md">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
              Create Namespace
            </h2>
            <form
              onSubmit={(e) => {
                e.preventDefault();
                createMutation.mutate(formData);
              }}
              className="space-y-4"
            >
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Name
                </label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                  required
                  pattern="[a-z0-9-]+"
                  placeholder="my-organization"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Description
                </label>
                <textarea
                  value={formData.description || ''}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                  rows={2}
                />
              </div>
              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  id="is_public"
                  checked={formData.is_public}
                  onChange={(e) => setFormData({ ...formData, is_public: e.target.checked })}
                  className="rounded"
                />
                <label htmlFor="is_public" className="text-sm text-gray-700 dark:text-gray-300">
                  Public (allow anonymous downloads)
                </label>
              </div>
              <div className="flex justify-end gap-2 pt-4">
                <button
                  type="button"
                  onClick={() => setIsModalOpen(false)}
                  className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700"
                >
                  Create
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}

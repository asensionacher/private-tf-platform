import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Building2, Key, Plus, Trash2, Copy, Eye, EyeOff } from 'lucide-react';
import { namespacesApi } from '../api';
import type { APIKeyCreate } from '../types';

export default function NamespaceDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [isKeyModalOpen, setIsKeyModalOpen] = useState(false);
  const [newKeyData, setNewKeyData] = useState<APIKeyCreate>({
    name: '',
    permissions: 'read',
  });
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const [showKey, setShowKey] = useState(false);

  const { data: namespace, isLoading } = useQuery({
    queryKey: ['namespace', id],
    queryFn: () => namespacesApi.getById(id!),
    enabled: !!id,
  });

  const { data: apiKeysData } = useQuery({
    queryKey: ['api-keys', id],
    queryFn: () => namespacesApi.getAPIKeys(id!),
    enabled: !!id,
  });

  const apiKeys = Array.isArray(apiKeysData) ? apiKeysData : [];

  const createKeyMutation = useMutation({
    mutationFn: (data: APIKeyCreate) => namespacesApi.createAPIKey(id!, data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['api-keys', id] });
      setCreatedKey(data.key || null);
      setNewKeyData({ name: '', permissions: 'read' });
    },
  });

  const deleteKeyMutation = useMutation({
    mutationFn: (keyId: string) => namespacesApi.deleteAPIKey(id!, keyId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['api-keys', id] });
    },
  });

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600"></div>
      </div>
    );
  }

  if (!namespace) {
    return (
      <div className="text-center py-12">
        <p className="text-gray-500">Namespace not found</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center space-x-4">
        <button
          onClick={() => navigate('/namespaces')}
          className="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700"
        >
          <ArrowLeft className="h-5 w-5 text-gray-600 dark:text-gray-400" />
        </button>
        <div className="flex items-center">
          <Building2 className="h-10 w-10 text-gray-400" />
          <div className="ml-4">
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
              {namespace.name}
              {namespace.is_public && (
                <span className="ml-2 px-2 py-0.5 text-xs bg-green-100 text-green-800 rounded-full">
                  Public
                </span>
              )}
            </h1>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              {namespace.description || 'No description'}
            </p>
          </div>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 gap-4">
        <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-4">
          <p className="text-3xl font-bold text-indigo-600">{namespace.module_count || 0}</p>
          <p className="text-sm text-gray-500 dark:text-gray-400">Modules</p>
        </div>
        <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-4">
          <p className="text-3xl font-bold text-purple-600">{namespace.provider_count || 0}</p>
          <p className="text-sm text-gray-500 dark:text-gray-400">Providers</p>
        </div>
      </div>

      {/* API Keys */}
      <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-6">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white">API Keys</h2>
          <button
            onClick={() => setIsKeyModalOpen(true)}
            className="flex items-center gap-2 px-3 py-1.5 bg-indigo-600 text-white text-sm rounded-lg hover:bg-indigo-700"
          >
            <Plus className="h-4 w-4" />
            New Key
          </button>
        </div>

        {apiKeys.length === 0 ? (
          <p className="text-sm text-gray-500 dark:text-gray-400">No API keys created yet</p>
        ) : (
          <ul className="space-y-2">
            {apiKeys.map((key) => (
              <li
                key={key.id}
                className="flex items-center justify-between py-3 px-4 bg-gray-50 dark:bg-gray-900 rounded-lg"
              >
                <div className="flex items-center">
                  <Key className="h-5 w-5 text-gray-400 mr-3" />
                  <div>
                    <p className="text-sm font-medium text-gray-900 dark:text-white">{key.name}</p>
                    <div className="flex gap-2 text-xs text-gray-500">
                      <span className={`px-1.5 py-0.5 rounded ${
                        key.permissions === 'admin' ? 'bg-red-100 text-red-800' :
                        key.permissions === 'write' ? 'bg-yellow-100 text-yellow-800' :
                        'bg-green-100 text-green-800'
                      }`}>
                        {key.permissions}
                      </span>
                      {key.last_used_at && (
                        <span>Last used: {new Date(key.last_used_at).toLocaleDateString()}</span>
                      )}
                    </div>
                  </div>
                </div>
                <button
                  onClick={() => {
                    if (confirm('Delete this API key?')) {
                      deleteKeyMutation.mutate(key.id);
                    }
                  }}
                  className="p-2 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 rounded"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      {/* Create Key Modal */}
      {isKeyModalOpen && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-md">
            {createdKey ? (
              <>
                <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
                  API Key Created
                </h2>
                <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4 mb-4">
                  <p className="text-sm text-yellow-800 dark:text-yellow-200 mb-2">
                    ⚠️ Copy this key now. You won't be able to see it again!
                  </p>
                  <div className="flex items-center gap-2 bg-white dark:bg-gray-900 rounded p-2">
                    <code className="flex-1 text-sm font-mono break-all">
                      {showKey ? createdKey : '•'.repeat(40)}
                    </code>
                    <button
                      onClick={() => setShowKey(!showKey)}
                      className="p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
                    >
                      {showKey ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                    </button>
                    <button
                      onClick={() => copyToClipboard(createdKey)}
                      className="p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
                    >
                      <Copy className="h-4 w-4" />
                    </button>
                  </div>
                </div>
                <button
                  onClick={() => {
                    setIsKeyModalOpen(false);
                    setCreatedKey(null);
                    setShowKey(false);
                  }}
                  className="w-full px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700"
                >
                  Done
                </button>
              </>
            ) : (
              <>
                <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
                  Create API Key
                </h2>
                <form
                  onSubmit={(e) => {
                    e.preventDefault();
                    createKeyMutation.mutate(newKeyData);
                  }}
                  className="space-y-4"
                >
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                      Name
                    </label>
                    <input
                      type="text"
                      value={newKeyData.name}
                      onChange={(e) => setNewKeyData({ ...newKeyData, name: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                      required
                      placeholder="CI/CD Pipeline"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                      Permissions
                    </label>
                    <select
                      value={newKeyData.permissions}
                      onChange={(e) => setNewKeyData({ ...newKeyData, permissions: e.target.value as 'read' | 'write' | 'admin' })}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                    >
                      <option value="read">Read - Download modules/providers</option>
                      <option value="write">Write - Upload and delete</option>
                      <option value="admin">Admin - Full access</option>
                    </select>
                  </div>
                  <div className="flex justify-end gap-2 pt-4">
                    <button
                      type="button"
                      onClick={() => setIsKeyModalOpen(false)}
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
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

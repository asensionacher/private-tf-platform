import { useState, useEffect } from 'react';
import { Key, Plus, Trash2, Copy, Check } from 'lucide-react';
import { namespacesApi } from '@/api';
import { useAuth } from '../context/AuthContext';
import type { APIKey } from '../types';

export default function ApiKeysPage() {
  const { user, isAdmin } = useAuth();
  const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [newKeyName, setNewKeyName] = useState('');
  const [createdKey, setCreatedKey] = useState<APIKey | null>(null);
  const [copiedKey, setCopiedKey] = useState(false);
  const [loadingKeys, setLoadingKeys] = useState(false);

  useEffect(() => {
    fetchApiKeys();
  }, []);

  const fetchApiKeys = async () => {
    setLoadingKeys(true);
    try {
      const data = await namespacesApi.getAPIKeys();
      setApiKeys(data);
    } catch (error) {
      console.error('Failed to fetch API keys:', error);
      setApiKeys([]);
    } finally {
      setLoadingKeys(false);
    }
  };

  const handleCreateKey = async () => {
    if (!newKeyName) return;
    try {
      const response = await namespacesApi.createAPIKey('', { name: newKeyName });
      setCreatedKey(response);
      setNewKeyName('');
      setShowCreateForm(false);
      fetchApiKeys();
    } catch (error) {
      console.error('Failed to create API key:', error);
      alert('Failed to create API key');
    }
  };

  const handleDeleteKey = async (keyId: string) => {
    if (!confirm('Are you sure you want to revoke this API key? This action cannot be undone.')) {
      return;
    }
    try {
      await namespacesApi.deleteAPIKey('', keyId);
      fetchApiKeys();
    } catch (error) {
      console.error('Failed to delete API key:', error);
      alert('Failed to delete API key');
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(true);
    setTimeout(() => setCopiedKey(false), 2000);
  };

  const canDelete = (key: APIKey) => isAdmin || key.user_id === user?.id;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">API Keys</h1>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {isAdmin
            ? 'Manage all API keys across all users.'
            : 'Manage your API keys for Terraform and OpenTofu CLI authentication.'}
        </p>
      </div>

      <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center">
            <Key className="h-5 w-5 text-gray-400 mr-2" />
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
              {isAdmin ? 'All API Keys' : 'My API Keys'}
            </h2>
          </div>
          {!showCreateForm && (
            <button
              onClick={() => setShowCreateForm(true)}
              className="flex items-center gap-2 px-4 py-2 bg-indigo-600 text-white rounded hover:bg-indigo-700"
            >
              <Plus className="h-4 w-4" />
              Create API Key
            </button>
          )}
        </div>

        <div className="space-y-4">
          {/* Created Key Alert */}
          {createdKey && createdKey.key && (
            <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4">
              <h3 className="text-sm font-medium text-green-800 dark:text-green-200 mb-2">
                API Key Created Successfully
              </h3>
              <p className="text-xs text-green-700 dark:text-green-300 mb-2">
                Make sure to copy your API key now. You won't be able to see it again!
              </p>
              <div className="flex items-center gap-2">
                <code className="flex-1 text-xs bg-green-100 dark:bg-green-900/40 p-2 rounded font-mono break-all">
                  {createdKey.key}
                </code>
                <button
                  onClick={() => copyToClipboard(createdKey.key!)}
                  className="px-3 py-2 bg-green-600 text-white rounded hover:bg-green-700 flex items-center gap-1"
                >
                  {copiedKey ? (
                    <>
                      <Check className="h-4 w-4" />
                      Copied
                    </>
                  ) : (
                    <>
                      <Copy className="h-4 w-4" />
                      Copy
                    </>
                  )}
                </button>
              </div>
            </div>
          )}

          {/* Create Form */}
          {showCreateForm && (
            <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
              <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-3">
                Create New API Key
              </h3>
              <div className="space-y-3">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Name
                  </label>
                  <input
                    type="text"
                    value={newKeyName}
                    onChange={(e) => setNewKeyName(e.target.value)}
                    placeholder="e.g., CI/CD Pipeline, Development"
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-700 dark:text-white"
                  />
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={handleCreateKey}
                    disabled={!newKeyName}
                    className="px-4 py-2 bg-indigo-600 text-white rounded hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Create Key
                  </button>
                  <button
                    onClick={() => { setShowCreateForm(false); setNewKeyName(''); }}
                    className="px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded hover:bg-gray-300 dark:hover:bg-gray-600"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* API Keys List */}
          {loadingKeys ? (
            <div className="flex items-center justify-center py-8">
              <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-indigo-600"></div>
            </div>
          ) : apiKeys.length > 0 ? (
            <div className="space-y-2">
              <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">
                {apiKeys.length} key{apiKeys.length !== 1 ? 's' : ''}
              </h3>
              {apiKeys.map((key) => (
                <div
                  key={key.id}
                  className="flex items-center justify-between p-3 border border-gray-200 dark:border-gray-700 rounded-lg"
                >
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="font-medium text-gray-900 dark:text-white">
                        {key.name}
                      </span>
                      {isAdmin && key.username && (
                        <span className="px-2 py-0.5 text-xs rounded bg-indigo-100 dark:bg-indigo-900/40 text-indigo-700 dark:text-indigo-300">
                          {key.username}
                        </span>
                      )}
                    </div>
                    <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                      Created: {new Date(key.created_at).toLocaleDateString()}
                      {key.last_used_at && (
                        <> · Last used: {new Date(key.last_used_at).toLocaleDateString()}</>
                      )}
                    </div>
                  </div>
                  {canDelete(key) && (
                    <button
                      onClick={() => handleDeleteKey(key.id)}
                      className="ml-3 p-2 text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 rounded flex-shrink-0"
                      title="Revoke API Key"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-gray-500 dark:text-gray-400 py-4">
              No API keys found. Create one to use with Terraform or OpenTofu CLI.
            </p>
          )}
        </div>
      </div>
    </div>
  );
}

import { useState, useEffect } from 'react';
import { Settings, Terminal, Key, Plus, Trash2, Copy, Check } from 'lucide-react';
import axios from 'axios';
import type { APIKey } from '../types';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '';

export default function SettingsPage() {
  const [registryUrl, setRegistryUrl] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [newKeyName, setNewKeyName] = useState('');
  const [createdKey, setCreatedKey] = useState<APIKey | null>(null);
  const [copiedKey, setCopiedKey] = useState(false);
  const [loadingKeys, setLoadingKeys] = useState(false);

  useEffect(() => {
    const fetchData = async () => {
      try {
        // Fetch registry URL
        const registryResponse = await axios.get(`${API_BASE_URL}/.well-known/terraform.json`);
        const modulesUrl = registryResponse.data['modules.v1'];
        if (modulesUrl) {
          const url = new URL(modulesUrl);
          setRegistryUrl(`${url.protocol}//${url.host}`);
        }
      } catch (error) {
        console.error('Failed to fetch data:', error);
        setRegistryUrl(window.location.origin);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
    fetchApiKeys();
  }, []);

  const fetchApiKeys = async () => {
    setLoadingKeys(true);
    try {
      const response = await axios.get(`${API_BASE_URL}/api/api-keys`);
      setApiKeys(response.data || []);
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
      const payload = {
        name: newKeyName,
      };

      const response = await axios.post(
        `${API_BASE_URL}/api/api-keys`,
        payload
      );

      setCreatedKey(response.data);
      setNewKeyName('');
      setShowCreateForm(false);
      fetchApiKeys();
    } catch (error) {
      console.error('Failed to create API key:', error);
      alert('Failed to create API key');
    }
  };

  const handleDeleteKey = async (keyId: string) => {
    if (!confirm('Are you sure you want to delete this API key? This action cannot be undone.')) {
      return;
    }

    try {
      await axios.delete(`${API_BASE_URL}/api/api-keys/${keyId}`);
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

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600"></div>
      </div>
    );
  }

  const registryHost = registryUrl ? new URL(registryUrl).host : 'registry.local:9080';

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Settings</h1>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Registry information and Terraform CLI configuration
        </p>
      </div>

      <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-6">
        <div className="flex items-center mb-4">
          <Settings className="h-5 w-5 text-gray-400 mr-2" />
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Registry Information</h2>
        </div>
        <dl className="space-y-3">
          <div>
            <dt className="text-sm text-gray-500 dark:text-gray-400">Service Discovery</dt>
            <dd className="text-sm font-mono text-gray-900 dark:text-white">
              {registryUrl}/.well-known/terraform.json
            </dd>
          </div>
          <div>
            <dt className="text-sm text-gray-500 dark:text-gray-400">Modules API</dt>
            <dd className="text-sm font-mono text-gray-900 dark:text-white">
              {registryUrl}/v1/modules/
            </dd>
          </div>
          <div>
            <dt className="text-sm text-gray-500 dark:text-gray-400">Providers API</dt>
            <dd className="text-sm font-mono text-gray-900 dark:text-white">
              {registryUrl}/v1/providers/
            </dd>
          </div>
        </dl>
      </div>

      <div className="bg-blue-50 dark:bg-blue-900/20 rounded-lg p-4">
        <div className="flex items-center mb-2">
          <Terminal className="h-4 w-4 text-blue-600 dark:text-blue-400 mr-2" />
          <h3 className="text-sm font-medium text-blue-800 dark:text-blue-200">
            Terraform CLI Configuration
          </h3>
        </div>
        <p className="text-xs text-blue-700 dark:text-blue-300 mb-2">
          To use private namespaces with Terraform CLI, create an API key from the namespace detail page,
          then add this to your ~/.terraformrc:
        </p>
        <pre className="text-xs text-blue-700 dark:text-blue-300 bg-blue-100 dark:bg-blue-900/40 p-3 rounded">
          {`credentials "${registryHost}" {
  token = "YOUR_API_KEY_HERE"
}

# For local development, add to /etc/hosts:
# 127.0.0.1 registry.local`}
        </pre>
      </div>

      <div className="bg-amber-50 dark:bg-amber-900/20 rounded-lg p-4">
        <h3 className="text-sm font-medium text-amber-800 dark:text-amber-200 mb-2">
          About API Keys
        </h3>
        <p className="text-xs text-amber-700 dark:text-amber-300">
          API keys are only required for Terraform CLI access to private namespaces.
          Public namespaces can be accessed without authentication.
          The web interface does not require authentication - you can create and manage
          namespaces, modules, and providers directly from this UI.
        </p>
      </div>

      {/* API Keys Management */}
      <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center">
            <Key className="h-5 w-5 text-gray-400 mr-2" />
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Global API Keys</h2>
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

        <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
          API keys provide global access to all namespaces in the registry. Use them with Terraform CLI for authentication.
        </p>

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
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  All API keys have full admin permissions.
                </p>
                <div className="flex gap-2">
                  <button
                    onClick={handleCreateKey}
                    disabled={!newKeyName}
                    className="px-4 py-2 bg-indigo-600 text-white rounded hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Create Key
                  </button>
                  <button
                    onClick={() => {
                      setShowCreateForm(false);
                      setNewKeyName('');
                    }}
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
                Existing Keys ({apiKeys.length})
              </h3>
              {apiKeys.map((key) => (
                <div
                  key={key.id}
                  className="flex items-center justify-between p-3 border border-gray-200 dark:border-gray-700 rounded-lg"
                >
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-gray-900 dark:text-white">
                        {key.name === '__runner__' ? 'Runner (Internal)' : key.name}
                      </span>
                      <span className="px-2 py-0.5 text-xs rounded bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400">
                        {key.permissions}
                      </span>
                      {key.name === '__runner__' && (
                        <span className="px-2 py-0.5 text-xs rounded bg-blue-100 dark:bg-blue-900 text-blue-600 dark:text-blue-400">
                          Protected
                        </span>
                      )}
                    </div>
                    <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                      Created: {new Date(key.created_at).toLocaleDateString()}
                      {key.last_used_at && (
                        <> • Last used: {new Date(key.last_used_at).toLocaleDateString()}</>
                      )}
                    </div>
                  </div>
                  {key.name !== '__runner__' && (
                    <button
                      onClick={() => handleDeleteKey(key.id)}
                      className="p-2 text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 rounded"
                      title="Delete API Key"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-gray-500 dark:text-gray-400 py-4">
              No API keys found. Create one to use with Terraform CLI.
            </p>
          )}
        </div>
      </div>
    </div>
  );
}

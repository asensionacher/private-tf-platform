import { useState, useRef, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Puzzle, Tag, ExternalLink, Copy, RefreshCw, Check, Eye, EyeOff, Trash2, FileText, Upload } from 'lucide-react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';
import rehypeSanitize, { defaultSchema } from 'rehype-sanitize';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { oneDark, oneLight } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { useTheme } from '../context/ThemeContext';
import { providersApi } from '../api';

// Custom schema to allow anchor names and common HTML elements
const sanitizeSchema = {
  ...defaultSchema,
  attributes: {
    ...defaultSchema.attributes,
    a: [...(defaultSchema.attributes?.a || []), 'name'],
    '*': ['className', 'id'],
  },
  // Remove HTML comments
  strip: ['comment'],
};

export default function ProviderDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { theme } = useTheme();

  const [syncing, setSyncing] = useState(false);
  const [syncMessage, setSyncMessage] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [selectedVersion, setSelectedVersion] = useState<string | undefined>(undefined);

  const { data: provider, isLoading: providerLoading } = useQuery({
    queryKey: ['provider', id],
    queryFn: () => providersApi.getById(id!),
    enabled: !!id,
  });

  // Auto-refresh when provider is not synced yet
  useEffect(() => {
    if (provider && !provider.synced) {
      const interval = setInterval(() => {
        queryClient.invalidateQueries({ queryKey: ['provider', id] });
      }, 3000); // Check every 3 seconds

      return () => clearInterval(interval);
    }
  }, [provider, id, queryClient]);

  const { data: versionsData, isLoading: versionsLoading } = useQuery({
    queryKey: ['provider-versions', id],
    queryFn: () => providersApi.getVersions(id!),
    enabled: !!id,
  });

  const { data: readmeData, isLoading: readmeLoading } = useQuery({
    queryKey: ['provider-readme', id, selectedVersion],
    queryFn: () => providersApi.getReadme(id!, selectedVersion),
    enabled: !!id,
  });

  const syncTagsMutation = useMutation({
    mutationFn: () => providersApi.syncTags(id!),
    onMutate: () => {
      setSyncing(true);
      setSyncMessage(null);
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['provider-versions', id] });
      setSyncMessage(`Found ${data.tags_found} tags, added ${data.tags_added} new versions`);
      setSyncing(false);
    },
    onError: (err: any) => {
      setSyncMessage(err.response?.data?.error || 'Failed to sync tags');
      setSyncing(false);
    },
  });

  const toggleVersionMutation = useMutation({
    mutationFn: ({ versionId, enabled }: { versionId: string; enabled: boolean }) =>
      providersApi.toggleVersion(id!, versionId, enabled),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['provider-versions', id] });
    },
  });

  const deleteVersionMutation = useMutation({
    mutationFn: (versionId: string) => providersApi.deleteVersion(id!, versionId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['provider-versions', id] });
    },
  });

  const deleteProviderMutation = useMutation({
    mutationFn: () => providersApi.delete(id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['providers'] });
      navigate('/providers');
    },
  });

  // Upload state
  const [expandedVersion, setExpandedVersion] = useState<string | null>(null);
  const [uploadingPlatform, setUploadingPlatform] = useState<string | null>(null);
  const [uploadMessage, setUploadMessage] = useState<{ versionId: string; message: string; success: boolean } | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [selectedUpload, setSelectedUpload] = useState<{ versionId: string; os: string; arch: string } | null>(null);

  const PLATFORMS = [
    { os: 'linux', arch: 'amd64', label: 'Linux AMD64' },
    { os: 'linux', arch: 'arm64', label: 'Linux ARM64' },
    { os: 'darwin', arch: 'amd64', label: 'macOS AMD64' },
    { os: 'darwin', arch: 'arm64', label: 'macOS ARM64 (Apple Silicon)' },
    { os: 'windows', arch: 'amd64', label: 'Windows AMD64' },
  ];

  const uploadPlatformMutation = useMutation({
    mutationFn: ({ versionId, os, arch, file }: { versionId: string; os: string; arch: string; file: File }) =>
      providersApi.uploadPlatform(id!, versionId, os, arch, file),
    onMutate: ({ os, arch }) => {
      setUploadingPlatform(`${os}_${arch}`);
      setUploadMessage(null);
    },
    onSuccess: (data, { versionId }) => {
      queryClient.invalidateQueries({ queryKey: ['provider-platforms', id, versionId] });
      queryClient.invalidateQueries({ queryKey: ['provider-versions', id] });
      setUploadMessage({
        versionId,
        message: `Uploaded ${data.os}/${data.arch} successfully`,
        success: true,
      });
      setUploadingPlatform(null);
      setSelectedUpload(null);
    },
    onError: (err: any, { versionId }) => {
      setUploadMessage({
        versionId,
        message: err.response?.data?.error || 'Upload failed',
        success: false,
      });
      setUploadingPlatform(null);
      setSelectedUpload(null);
    },
  });

  const [deletingPlatform, setDeletingPlatform] = useState<string | null>(null);

  const deletePlatformMutation = useMutation({
    mutationFn: ({ versionId, platformId }: { versionId: string; platformId: string }) =>
      providersApi.deletePlatform(id!, versionId, platformId),
    onMutate: ({ platformId }) => {
      setDeletingPlatform(platformId);
    },
    onSuccess: (_, { versionId }) => {
      queryClient.invalidateQueries({ queryKey: ['provider-platforms', id, versionId] });
      queryClient.invalidateQueries({ queryKey: ['provider-versions', id] });
      setDeletingPlatform(null);
    },
    onError: (err: any, { versionId }) => {
      setUploadMessage({
        versionId,
        message: err.response?.data?.error || 'Delete failed',
        success: false,
      });
      setDeletingPlatform(null);
    },
  });

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file && selectedUpload) {
      uploadPlatformMutation.mutate({
        versionId: selectedUpload.versionId,
        os: selectedUpload.os,
        arch: selectedUpload.arch,
        file,
      });
    }
    // Reset input
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const triggerUpload = (versionId: string, os: string, arch: string) => {
    setSelectedUpload({ versionId, os, arch });
    fileInputRef.current?.click();
  };

  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  const versions = Array.isArray(versionsData) ? versionsData : [];
  const enabledVersions = versions.filter(v => v.enabled);

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  if (providerLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-purple-600"></div>
      </div>
    );
  }

  if (!provider) {
    return (
      <div className="text-center py-12">
        <p className="text-gray-500">Provider not found</p>
      </div>
    );
  }

  // Show syncing message if provider is not yet synced
  if (!provider.synced) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <button
              onClick={() => navigate('/providers')}
              className="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700"
            >
              <ArrowLeft className="h-5 w-5 text-gray-600 dark:text-gray-400" />
            </button>
            <div className="flex items-center">
              <Puzzle className="h-10 w-10 text-gray-400" />
              <div className="ml-4">
                <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
                  {provider.namespace}/{provider.name}
                </h1>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  {provider.description || 'No description'}
                </p>
              </div>
            </div>
          </div>
        </div>

        <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-8 text-center">
          <div className="flex justify-center mb-4">
            <svg className="animate-spin h-12 w-12 text-yellow-600 dark:text-yellow-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
          </div>
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
            Synchronizing Tags
          </h2>
          <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
            Please wait while we synchronize the tags from the Git repository. This may take a few moments.
          </p>
          <p className="text-xs text-gray-500 dark:text-gray-500">
            The provider will be available once the synchronization is complete.
          </p>
        </div>
      </div>
    );
  }

  const host = window.location.hostname === 'localhost' ? `localhost.localdomain${window.location.port ? ':' + window.location.port : ''}` : window.location.host;
  const providerSource = `${host}/${provider.namespace}/${provider.name}`;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <button
            onClick={() => navigate('/providers')}
            className="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700"
          >
            <ArrowLeft className="h-5 w-5 text-gray-600 dark:text-gray-400" />
          </button>
          <div className="flex items-center">
            <Puzzle className="h-10 w-10 text-purple-500" />
            <div className="ml-4">
              <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
                {provider.namespace}/{provider.name}
              </h1>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                {provider.description || 'No description'}
              </p>
            </div>
          </div>
        </div>
        <button
          onClick={() => setShowDeleteConfirm(true)}
          className="inline-flex items-center px-3 py-2 border border-red-300 dark:border-red-700 rounded-md text-sm font-medium text-red-700 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"
        >
          <Trash2 className="h-4 w-4 mr-2" />
          Delete Provider
        </button>
      </div>

      {/* Usage */}
      <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-6">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Usage</h2>
        {enabledVersions.length === 0 ? (
          <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4">
            <p className="text-sm text-yellow-700 dark:text-yellow-300">
              No versions enabled yet. Sync tags from the repository and enable the versions you want to publish.
            </p>
          </div>
        ) : (
          <div className="bg-gray-50 dark:bg-gray-900 rounded-lg p-4 relative">
            <button
              onClick={() => copyToClipboard(`terraform {\n  required_providers {\n    ${provider.name} = {\n      source  = "${providerSource}"\n      version = "${enabledVersions[0]?.version || '1.0.0'}"\n    }\n  }\n}`)}
              className="absolute top-2 right-2 p-2 hover:bg-gray-200 dark:hover:bg-gray-700 rounded"
              title="Copy to clipboard"
            >
              {copied ? (
                <Check className="h-4 w-4 text-green-500" />
              ) : (
                <Copy className="h-4 w-4 text-gray-500" />
              )}
            </button>
            <pre className="text-sm text-gray-700 dark:text-gray-300">
              {`terraform {
  required_providers {
    ${provider.name} = {
      source  = "${providerSource}"
      version = "${enabledVersions[0]?.version || '1.0.0'}"
    }
  }
}`}
            </pre>
          </div>
        )}
      </div>

      {/* Provider Info */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-6">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Details</h2>
          <dl className="space-y-3">
            <div>
              <dt className="text-sm text-gray-500 dark:text-gray-400">Namespace</dt>
              <dd className="text-sm font-medium text-gray-900 dark:text-white">{provider.namespace}</dd>
            </div>
            <div>
              <dt className="text-sm text-gray-500 dark:text-gray-400">Name</dt>
              <dd className="text-sm font-medium text-gray-900 dark:text-white">{provider.name}</dd>
            </div>
            {provider.source_url && (
              <div>
                <dt className="text-sm text-gray-500 dark:text-gray-400">Git Repository</dt>
                <dd>
                  <a
                    href={provider.source_url.replace(/\.git$/, '')}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-sm text-purple-600 dark:text-purple-400 hover:underline flex items-center"
                  >
                    {provider.source_url.replace(/\.git$/, '')}
                    <ExternalLink className="h-3 w-3 ml-1" />
                  </a>
                </dd>
              </div>
            )}
            <div>
              <dt className="text-sm text-gray-500 dark:text-gray-400">Created</dt>
              <dd className="text-sm font-medium text-gray-900 dark:text-white">
                {new Date(provider.created_at).toLocaleDateString()}
              </dd>
            </div>
          </dl>
        </div>

        {/* Versions */}
        <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-6">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
                Versions
              </h2>
              <p className="text-xs text-gray-500 dark:text-gray-400">
                {enabledVersions.length} of {versions.length} enabled
              </p>
            </div>
            <button
              onClick={() => syncTagsMutation.mutate()}
              disabled={syncing}
              className="inline-flex items-center px-3 py-1.5 border border-transparent text-sm font-medium rounded-md text-white bg-purple-600 hover:bg-purple-700 disabled:opacity-50"
            >
              <RefreshCw className={`h-4 w-4 mr-1 ${syncing ? 'animate-spin' : ''}`} />
              Sync Tags
            </button>
          </div>

          {syncMessage && (
            <div className={`mb-4 p-3 rounded-lg text-sm ${syncMessage.includes('Failed')
              ? 'bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-300'
              : 'bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-300'
              }`}>
              {syncMessage}
            </div>
          )}

          {versionsLoading ? (
            <div className="flex justify-center py-8">
              <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-purple-600"></div>
            </div>
          ) : versions.length === 0 ? (
            <div className="text-center py-8">
              <Tag className="mx-auto h-8 w-8 text-gray-400" />
              <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
                No versions found
              </p>
              <p className="text-xs text-gray-400 dark:text-gray-500 mt-1">
                Click "Sync Tags" to fetch tags from the Git repository
              </p>
            </div>
          ) : (
            <>
              {/* Hidden file input */}
              <input
                ref={fileInputRef}
                type="file"
                accept=".zip"
                onChange={handleFileSelect}
                className="hidden"
              />

              <ul className="space-y-2 max-h-[32rem] overflow-y-auto">
                {versions.map((version) => (
                  <li
                    key={version.id}
                    className={`rounded group transition-colors ${version.enabled
                      ? 'bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800'
                      : 'bg-gray-50 dark:bg-gray-900 border border-transparent'
                      }`}
                  >
                    <div className="py-2 px-3 flex items-center justify-between">
                      <button
                        onClick={() => setExpandedVersion(expandedVersion === version.id ? null : version.id)}
                        className="flex items-center flex-1 text-left"
                      >
                        <Tag className={`h-4 w-4 mr-2 ${version.enabled ? 'text-green-500' : 'text-gray-400'}`} />
                        <span className={`text-sm font-medium ${version.enabled
                          ? 'text-green-700 dark:text-green-300'
                          : 'text-gray-600 dark:text-gray-400'
                          }`}>
                          {version.version}
                        </span>
                        {version.enabled && (
                          <span className="ml-2 text-xs text-green-600 dark:text-green-400 bg-green-100 dark:bg-green-900/40 px-1.5 py-0.5 rounded">
                            Published
                          </span>
                        )}
                        {version.platforms && version.platforms.length > 0 && (
                          <span className="ml-2 text-xs text-purple-600 dark:text-purple-400 bg-purple-100 dark:bg-purple-900/40 px-1.5 py-0.5 rounded">
                            {version.platforms.length} platform{version.platforms.length !== 1 ? 's' : ''}
                          </span>
                        )}
                      </button>
                      <div className="flex items-center gap-1">
                        <button
                          onClick={() => toggleVersionMutation.mutate({
                            versionId: version.id,
                            enabled: !version.enabled
                          })}
                          className={`p-1.5 rounded transition-colors ${version.enabled
                            ? 'text-green-600 hover:bg-green-100 dark:hover:bg-green-900/40'
                            : 'text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700'
                            }`}
                          title={version.enabled ? 'Disable version' : 'Enable version'}
                        >
                          {version.enabled ? (
                            <Eye className="h-4 w-4" />
                          ) : (
                            <EyeOff className="h-4 w-4" />
                          )}
                        </button>
                        <button
                          onClick={() => {
                            if (confirm(`Delete version ${version.version}? This cannot be undone.`)) {
                              deleteVersionMutation.mutate(version.id);
                            }
                          }}
                          className="p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 rounded opacity-0 group-hover:opacity-100 transition-opacity"
                          title="Delete version"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>
                    </div>

                    {/* Expanded platforms section */}
                    {expandedVersion === version.id && (
                      <div className="px-3 pb-3 border-t border-gray-200 dark:border-gray-700 mt-2 pt-2">
                        <div className="text-xs font-medium text-gray-500 dark:text-gray-400 mb-2">
                          Platforms
                        </div>
                        <div className="space-y-1">
                          {PLATFORMS.map((platform) => {
                            const uploaded = version.platforms?.find(
                              p => p.os === platform.os && p.arch === platform.arch
                            );
                            const isUploading = uploadingPlatform === `${platform.os}_${platform.arch}` &&
                              selectedUpload?.versionId === version.id;
                            const isDeleting = uploaded && deletingPlatform === uploaded.id;

                            return (
                              <div
                                key={`${platform.os}_${platform.arch}`}
                                className="flex items-center justify-between py-1 px-2 rounded bg-white dark:bg-gray-800 group"
                              >
                                <span className="text-xs text-gray-600 dark:text-gray-400">
                                  {platform.label}
                                </span>
                                {uploaded ? (
                                  <div className="inline-flex items-center gap-2">
                                    <span className="inline-flex items-center text-xs text-green-600 dark:text-green-400">
                                      <Check className="h-3 w-3 mr-1" />
                                      Uploaded
                                    </span>
                                    <button
                                      onClick={() => {
                                        if (confirm(`Delete ${platform.label} binary?`)) {
                                          deletePlatformMutation.mutate({
                                            versionId: version.id,
                                            platformId: uploaded.id
                                          });
                                        }
                                      }}
                                      disabled={isDeleting}
                                      className="p-0.5 text-gray-400 hover:text-red-500 rounded opacity-0 group-hover:opacity-100 transition-opacity disabled:opacity-50"
                                      title="Delete binary"
                                    >
                                      {isDeleting ? (
                                        <RefreshCw className="h-3 w-3 animate-spin" />
                                      ) : (
                                        <Trash2 className="h-3 w-3" />
                                      )}
                                    </button>
                                  </div>
                                ) : (
                                  <button
                                    onClick={() => triggerUpload(version.id, platform.os, platform.arch)}
                                    disabled={isUploading}
                                    className="inline-flex items-center px-2 py-0.5 text-xs font-medium rounded text-purple-600 hover:bg-purple-100 dark:text-purple-400 dark:hover:bg-purple-900/40 disabled:opacity-50"
                                  >
                                    {isUploading ? (
                                      <>
                                        <RefreshCw className="h-3 w-3 mr-1 animate-spin" />
                                        Uploading...
                                      </>
                                    ) : (
                                      <>
                                        <Upload className="h-3 w-3 mr-1" />
                                        Upload
                                      </>
                                    )}
                                  </button>
                                )}
                              </div>
                            );
                          })}
                        </div>
                        {uploadMessage?.versionId === version.id && (
                          <div className={`mt-2 text-xs px-2 py-1 rounded ${uploadMessage.success
                            ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300'
                            : 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300'
                            }`}>
                            {uploadMessage.message}
                          </div>
                        )}
                      </div>
                    )}
                  </li>
                ))}
              </ul>
            </>
          )}

          <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
            <p className="text-xs text-gray-500 dark:text-gray-400">
              <strong>Tip:</strong> Click on a version to expand and upload platform binaries (.zip files).
              Only enabled versions with uploaded binaries are visible to Terraform.
            </p>
          </div>
        </div>
      </div>

      {/* Documentation */}
      <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center">
            <FileText className="h-5 w-5 text-purple-500 mr-2" />
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">README</h2>
          </div>
          <div className="flex items-center gap-3">
            {versions.length > 0 && (
              <>
                <label htmlFor="version-select" className="text-sm text-gray-600 dark:text-gray-400">
                  Version:
                </label>
                <select
                  id="version-select"
                  value={selectedVersion || ''}
                  onChange={(e) => setSelectedVersion(e.target.value || undefined)}
                  className="px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                >
                  <option value="">Latest (main branch)</option>
                  {versions.map((version) => (
                    <option key={version.id} value={version.version}>
                      {version.version}
                    </option>
                  ))}
                </select>
              </>
            )}
          </div>
        </div>

        {readmeLoading ? (
          <div className="flex justify-center py-8">
            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-purple-600"></div>
          </div>
        ) : readmeData?.content ? (
          <div className="prose prose-sm dark:prose-invert max-w-none 
            prose-headings:text-gray-900 dark:prose-headings:text-white prose-headings:border-b prose-headings:border-gray-200 dark:prose-headings:border-gray-700 prose-headings:pb-2
            prose-h1:text-2xl prose-h2:text-xl prose-h3:text-lg
            prose-p:text-gray-700 dark:prose-p:text-gray-300
            prose-a:text-blue-600 dark:prose-a:text-blue-400 prose-a:no-underline hover:prose-a:underline
            prose-code:bg-gray-100 dark:prose-code:bg-gray-800 prose-code:px-1.5 prose-code:py-0.5 prose-code:rounded-md prose-code:text-sm prose-code:font-mono prose-code:text-pink-600 dark:prose-code:text-pink-400 prose-code:before:content-none prose-code:after:content-none
            prose-pre:bg-transparent prose-pre:p-0
            prose-th:text-gray-900 dark:prose-th:text-white prose-th:bg-gray-50 dark:prose-th:bg-gray-800
            prose-td:text-gray-700 dark:prose-td:text-gray-300
            prose-li:text-gray-700 dark:prose-li:text-gray-300 prose-li:marker:text-gray-400
            prose-blockquote:border-l-4 prose-blockquote:border-blue-500 prose-blockquote:bg-blue-50 dark:prose-blockquote:bg-blue-900/20 prose-blockquote:pl-4 prose-blockquote:py-1
            prose-table:border prose-table:border-gray-200 dark:prose-table:border-gray-700
            prose-th:border prose-th:border-gray-200 dark:prose-th:border-gray-700 prose-th:px-4 prose-th:py-2
            prose-td:border prose-td:border-gray-200 dark:prose-td:border-gray-700 prose-td:px-4 prose-td:py-2
            prose-hr:border-gray-200 dark:prose-hr:border-gray-700
            prose-img:rounded-lg prose-img:shadow-md
          ">
            <ReactMarkdown
              remarkPlugins={[remarkGfm]}
              rehypePlugins={[rehypeRaw, [rehypeSanitize, sanitizeSchema]]}
              components={{
                code({ node, className, children, ...props }) {
                  const match = /language-(\w+)/.exec(className || '');
                  const isInline = !match && !className;

                  // Map terraform/tf to hcl for proper syntax highlighting
                  let language = match?.[1] || '';
                  if (language === 'tf' || language === 'terraform') {
                    language = 'hcl';
                  }

                  return !isInline && match ? (
                    <SyntaxHighlighter
                      style={theme === 'dark' ? oneDark : oneLight as { [key: string]: React.CSSProperties }}
                      language={language}
                      PreTag="div"
                      className="rounded-lg !my-4"
                      showLineNumbers={true}
                    >
                      {String(children).replace(/\n$/, '')}
                    </SyntaxHighlighter>
                  ) : (
                    <code className={className} {...props}>
                      {children}
                    </code>
                  );
                },
              }}
            >
              {readmeData.content.replace(/<!--[\s\S]*?-->/g, '')}
            </ReactMarkdown>
          </div>
        ) : (
          <div className="text-center py-8">
            <FileText className="mx-auto h-8 w-8 text-gray-400" />
            <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
              No README available. Sync tags to generate documentation.
            </p>
          </div>
        )}
      </div>

      {/* Delete Confirmation Modal */}
      {
        showDeleteConfirm && (
          <div className="fixed inset-0 z-50 overflow-y-auto">
            <div className="flex min-h-screen items-center justify-center p-4">
              <div className="fixed inset-0 bg-black/50" onClick={() => setShowDeleteConfirm(false)} />
              <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full p-6">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
                  Delete Provider
                </h3>
                <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">
                  Are you sure you want to delete <strong>{provider.namespace}/{provider.name}</strong>?
                  This will also delete all versions and cannot be undone.
                </p>
                <div className="flex justify-end space-x-3">
                  <button
                    onClick={() => setShowDeleteConfirm(false)}
                    className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-md"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={() => deleteProviderMutation.mutate()}
                    disabled={deleteProviderMutation.isPending}
                    className="px-4 py-2 text-sm font-medium text-white bg-red-600 hover:bg-red-700 rounded-md disabled:opacity-50"
                  >
                    {deleteProviderMutation.isPending ? 'Deleting...' : 'Delete'}
                  </button>
                </div>
              </div>
            </div>
          </div>
        )
      }
    </div >
  );
}

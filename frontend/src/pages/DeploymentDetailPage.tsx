import { useState, useEffect, useMemo, useRef } from 'react';
import { useParams, Link, useSearchParams } from 'react-router-dom';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';
import rehypeSanitize, { defaultSchema } from 'rehype-sanitize';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { oneDark, oneLight } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { useTheme } from '../context/ThemeContext';
import { deploymentsApi } from '../api';
import DeploymentModal from '../components/DeploymentModal';
import type { Deployment, GitReference, DirectoryListing, FileNode, DirectoryStatus } from '../types';

// Custom schema to allow anchor names and common HTML elements
const sanitizeSchema = {
    ...defaultSchema,
    attributes: {
        ...defaultSchema.attributes,
        a: [...(defaultSchema.attributes?.a || []), 'name'],
        '*': ['className', 'id'],
    },
    strip: ['comment'],
};

export default function DeploymentDetailPage() {
    const { id } = useParams<{ id: string }>();
    const { theme } = useTheme();
    const [searchParams, setSearchParams] = useSearchParams();
    const [deployment, setDeployment] = useState<Deployment | null>(null);
    const [references, setReferences] = useState<GitReference[]>([]);
    const [listing, setListing] = useState<DirectoryListing | null>(null);
    const [loading, setLoading] = useState(true);
    const [loadingDirectory, setLoadingDirectory] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [searchTerm, setSearchTerm] = useState<string>('');
    const [isDropdownOpen, setIsDropdownOpen] = useState(false);
    const [isSyncing, setIsSyncing] = useState(false);
    const [notification, setNotification] = useState<{ message: string; type: 'success' | 'error' } | null>(null);
    const [currentDirStatus, setCurrentDirStatus] = useState<DirectoryStatus | null>(null);
    const [showDeployModal, setShowDeployModal] = useState(false);
    const [deployModalPath, setDeployModalPath] = useState<string>('');
    const dropdownRef = useRef<HTMLDivElement>(null);

    // Show notification with auto-dismiss
    const showNotification = (message: string, type: 'success' | 'error' = 'success') => {
        setNotification({ message, type });
        setTimeout(() => setNotification(null), 4000);
    };

    // Get current path and ref from URL params
    const currentPath = searchParams.get('path') || '';
    const urlBranch = searchParams.get('branch');
    const urlTag = searchParams.get('tag');
    const selectedRef = urlBranch || urlTag || '';

    // Close dropdown when clicking outside
    useEffect(() => {
        const handleClickOutside = (event: MouseEvent) => {
            if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
                setIsDropdownOpen(false);
            }
        };

        document.addEventListener('mousedown', handleClickOutside);
        return () => document.removeEventListener('mousedown', handleClickOutside);
    }, []);

    const loadDeployment = async () => {
        try {
            setLoading(true);
            const [deploymentData, referencesData] = await Promise.all([
                deploymentsApi.getById(id!),
                deploymentsApi.getReferences(id!),
            ]);
            setDeployment(deploymentData);
            setReferences(referencesData);

            // Auto-select first branch or tag if not in URL
            if (!selectedRef && referencesData.length > 0) {
                const mainBranch = referencesData.find((r: GitReference) => r.type === 'branch' && (r.name === 'main' || r.name === 'master'));
                const defaultRef = mainBranch || referencesData[0];

                // Update URL with default ref
                const params: Record<string, string> = {};
                if (defaultRef.type === 'branch') {
                    params.branch = defaultRef.name;
                } else {
                    params.tag = defaultRef.name;
                }
                setSearchParams(params);
            }

            setError(null);
        } catch (err: any) {
            setError(err.response?.data?.error || 'Failed to load deployment');
        } finally {
            setLoading(false);
        }
    };

    const syncReferences = async () => {
        if (!id) return;
        try {
            setIsSyncing(true);
            const referencesData = await deploymentsApi.getReferences(id);
            setReferences(referencesData);

            // Also refresh current directory if we have a selected ref
            if (selectedRef) {
                await loadDirectory(currentPath);
            }

            // Count branches and tags
            const branches = referencesData.filter(r => r.type === 'branch').length;
            const tags = referencesData.filter(r => r.type === 'tag').length;

            showNotification(
                `Synchronized: ${branches} branch${branches !== 1 ? 'es' : ''}, ${tags} tag${tags !== 1 ? 's' : ''}, and current directory refreshed`,
                'success'
            );
        } catch (err: any) {
            const errorMsg = err.response?.data?.error || 'Failed to sync references';
            setError(errorMsg);
            showNotification(errorMsg, 'error');
        } finally {
            setIsSyncing(false);
        }
    };

    const handleDeploy = (dirPath: string) => {
        setDeployModalPath(dirPath);
        setShowDeployModal(true);
    };

    const loadDirectory = async (path: string) => {
        if (!selectedRef) return;
        try {
            setLoadingDirectory(true);
            const listingData = await deploymentsApi.getDirectory(id!, selectedRef, path || undefined);
            setListing(listingData);

            // Load status for current directory
            try {
                const status = await deploymentsApi.getStatus(id!, path || '');
                setCurrentDirStatus(status);
            } catch {
                setCurrentDirStatus(null);
            }

            setError(null);
        } catch (err: any) {
            setError(err.response?.data?.error || 'Failed to load directory');
        } finally {
            setLoadingDirectory(false);
        }
    };

    useEffect(() => {
        if (id) loadDeployment();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [id]);

    useEffect(() => {
        if (id && selectedRef) {
            loadDirectory(currentPath);
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [id, selectedRef, currentPath]);

    // Auto-refresh status when there's a running deployment
    useEffect(() => {
        const hasRunning = currentDirStatus?.status === 'running';
        if (hasRunning) {
            const interval = setInterval(() => {
                loadDirectory(currentPath);
            }, 3000);
            return () => clearInterval(interval);
        }
    }, [currentDirStatus, currentPath]);

    const navigateTo = (file: FileNode) => {
        if (file.is_dir) {
            const params: Record<string, string> = { path: file.path };
            if (urlBranch) params.branch = urlBranch;
            if (urlTag) params.tag = urlTag;
            setSearchParams(params);
        }
    };

    const navigateUp = () => {
        if (currentPath === '') return;
        const parts = currentPath.split('/');
        parts.pop();
        const newPath = parts.join('/');

        const params: Record<string, string> = {};
        if (urlBranch) params.branch = urlBranch;
        if (urlTag) params.tag = urlTag;
        if (newPath !== '') params.path = newPath;

        setSearchParams(params);
    };

    const branches = useMemo(() => references.filter((r: GitReference) => r.type === 'branch'), [references]);
    const tags = useMemo(() => references.filter((r: GitReference) => r.type === 'tag'), [references]);

    // Filtrar ramas y tags basándose en el término de búsqueda
    const filteredBranches = useMemo(() =>
        branches.filter(b => b.name.toLowerCase().includes(searchTerm.toLowerCase())),
        [branches, searchTerm]
    );
    const filteredTags = useMemo(() =>
        tags.filter(t => t.name.toLowerCase().includes(searchTerm.toLowerCase())),
        [tags, searchTerm]
    );

    if (loading) {
        return (
            <div className="flex items-center justify-center h-64">
                <div className="text-gray-500 dark:text-gray-400">Loading deployment...</div>
            </div>
        );
    }

    if (!deployment) {
        return (
            <div className="text-center py-12">
                <p className="text-red-600 dark:text-red-400">Deployment not found</p>
                {error && (
                    <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">{error}</p>
                )}
                <Link
                    to="/deployments"
                    className="mt-4 inline-block text-blue-600 dark:text-blue-400 hover:underline"
                >
                    ← Back to Deployments
                </Link>
            </div>
        );
    }

    if (loading) {
        return (
            <div className="flex items-center justify-center h-64">
                <div className="text-gray-500 dark:text-gray-400">Loading deployment...</div>
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

            <div>
                <Link to="/deployments" className="text-blue-600 dark:text-blue-400 hover:underline text-sm">
                    ← Back to Deployments
                </Link>
                <h1 className="text-3xl font-bold text-gray-900 dark:text-white mt-2">{deployment.name}</h1>
                {deployment.description && (
                    <p className="mt-2 text-gray-600 dark:text-gray-400">{deployment.description}</p>
                )}
                <div className="mt-3 flex items-center gap-4 text-sm text-gray-500 dark:text-gray-400">
                    <span className="flex items-center gap-1">
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
                        </svg>
                        {deployment.git_url}
                    </span>
                </div>
            </div>

            {error && (
                <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 text-red-700 dark:text-red-400 px-4 py-3 rounded">
                    {error}
                </div>
            )}

            <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                <div className="mb-4">
                    <div className="flex items-center justify-between mb-2">
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                            Select Branch or Tag
                        </label>
                        <button
                            onClick={syncReferences}
                            disabled={isSyncing}
                            className="flex items-center gap-1 px-3 py-1 text-sm text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                            title="Sync branches and tags"
                        >
                            <svg
                                className={`w-4 h-4 ${isSyncing ? 'animate-spin' : ''}`}
                                fill="none"
                                stroke="currentColor"
                                viewBox="0 0 24 24"
                            >
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                            </svg>
                            {isSyncing ? 'Syncing...' : 'Sync'}
                        </button>
                    </div>
                    {references.length === 0 ? (
                        <div className="text-gray-500 dark:text-gray-400 py-4">
                            No branches or tags found in this repository.
                        </div>
                    ) : (
                        <div className="relative w-full max-w-md" ref={dropdownRef}>
                            {/* Selector Button */}
                            <button
                                type="button"
                                onClick={() => setIsDropdownOpen(!isDropdownOpen)}
                                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-left flex items-center justify-between hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
                            >
                                <span className="flex items-center gap-2">
                                    {selectedRef ? (
                                        <>
                                            <span>{urlBranch ? '⎇' : '◉'}</span>
                                            <span>{selectedRef}</span>
                                        </>
                                    ) : (
                                        <span className="text-gray-400">Select branch or tag...</span>
                                    )}
                                </span>
                                <svg
                                    className={`w-4 h-4 transition-transform ${isDropdownOpen ? 'transform rotate-180' : ''}`}
                                    fill="none"
                                    stroke="currentColor"
                                    viewBox="0 0 24 24"
                                >
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                                </svg>
                            </button>

                            {/* Dropdown */}
                            {isDropdownOpen && (
                                <div className="absolute z-10 w-full mt-1 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg shadow-lg max-h-96 overflow-hidden">
                                    {/* Buscador dentro del dropdown */}
                                    <div className="p-2 border-b border-gray-200 dark:border-gray-600">
                                        <div className="relative">
                                            <input
                                                type="text"
                                                placeholder="Search branches and tags..."
                                                value={searchTerm}
                                                onChange={(e) => setSearchTerm(e.target.value)}
                                                className="w-full px-3 py-2 pl-9 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
                                                autoFocus
                                            />
                                            <svg
                                                className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400"
                                                fill="none"
                                                stroke="currentColor"
                                                viewBox="0 0 24 24"
                                            >
                                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                                            </svg>
                                        </div>
                                    </div>

                                    {/* Lista de opciones */}
                                    <div className="overflow-y-auto max-h-80">
                                        {filteredBranches.length === 0 && filteredTags.length === 0 ? (
                                            <div className="px-3 py-4 text-center text-sm text-gray-500 dark:text-gray-400">
                                                No results found
                                            </div>
                                        ) : (
                                            <>
                                                {filteredBranches.length > 0 && (
                                                    <div>
                                                        <div className="px-3 py-2 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase bg-gray-50 dark:bg-gray-800">
                                                            Branches
                                                        </div>
                                                        {filteredBranches.map((ref: GitReference) => (
                                                            <button
                                                                key={ref.name}
                                                                onClick={() => {
                                                                    const params: Record<string, string> = { branch: ref.name };
                                                                    if (currentPath) params.path = currentPath;
                                                                    setSearchParams(params);
                                                                    setIsDropdownOpen(false);
                                                                    setSearchTerm('');
                                                                }}
                                                                className={`w-full px-3 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-gray-600 flex items-center gap-2 ${selectedRef === ref.name ? 'bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400' : 'text-gray-900 dark:text-white'}`}
                                                            >
                                                                <span>⎇</span>
                                                                <span>{ref.name}</span>
                                                            </button>
                                                        ))}
                                                    </div>
                                                )}
                                                {filteredTags.length > 0 && (
                                                    <div>
                                                        <div className="px-3 py-2 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase bg-gray-50 dark:bg-gray-800">
                                                            Tags
                                                        </div>
                                                        {filteredTags.map((ref: GitReference) => (
                                                            <button
                                                                key={ref.name}
                                                                onClick={() => {
                                                                    const params: Record<string, string> = { tag: ref.name };
                                                                    if (currentPath) params.path = currentPath;
                                                                    setSearchParams(params);
                                                                    setIsDropdownOpen(false);
                                                                    setSearchTerm('');
                                                                }}
                                                                className={`w-full px-3 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-gray-600 flex items-center gap-2 ${selectedRef === ref.name ? 'bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400' : 'text-gray-900 dark:text-white'}`}
                                                            >
                                                                <span>◉</span>
                                                                <span>{ref.name}</span>
                                                            </button>
                                                        ))}
                                                    </div>
                                                )}
                                            </>
                                        )}
                                    </div>

                                    {/* Info de resultados */}
                                    {searchTerm && (filteredBranches.length > 0 || filteredTags.length > 0) && (
                                        <div className="px-3 py-2 text-xs text-gray-500 dark:text-gray-400 border-t border-gray-200 dark:border-gray-600 bg-gray-50 dark:bg-gray-800">
                                            Found {filteredBranches.length} branch{filteredBranches.length !== 1 ? 'es' : ''} and {filteredTags.length} tag{filteredTags.length !== 1 ? 's' : ''}
                                        </div>
                                    )}
                                </div>
                            )}
                        </div>
                    )}
                </div>

                {selectedRef && (
                    <>
                        <div className="mb-4 flex items-center justify-between">
                            <div className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
                                <span className="font-medium">Current path:</span>
                                <span className="font-mono bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded">
                                    /{currentPath || ''}
                                </span>
                                {currentPath && (
                                    <button
                                        onClick={navigateUp}
                                        className="ml-2 text-blue-600 dark:text-blue-400 hover:underline"
                                    >
                                        ← Go up
                                    </button>
                                )}
                            </div>
                            <div className="flex items-center gap-3">
                                {currentDirStatus && (
                                    <div className="flex items-center gap-2 px-3 py-2 bg-gray-50 dark:bg-gray-800 rounded-lg">
                                        <span className={`inline-flex h-3 w-3 rounded-full ${currentDirStatus.status_color === 'green' ? 'bg-green-500' :
                                            currentDirStatus.status_color === 'yellow' ? 'bg-yellow-500 animate-pulse' :
                                                currentDirStatus.status_color === 'red' ? 'bg-red-500' :
                                                    'bg-blue-500'
                                            }`}></span>
                                        <span className={`text-sm font-medium ${currentDirStatus.status_color === 'green' ? 'text-green-600 dark:text-green-400' :
                                            currentDirStatus.status_color === 'yellow' ? 'text-yellow-600 dark:text-yellow-400' :
                                                currentDirStatus.status_color === 'red' ? 'text-red-600 dark:text-red-400' :
                                                    'text-blue-600 dark:text-blue-400'
                                            }`}>
                                            {currentDirStatus.status === 'none' ? 'No deployments' :
                                                currentDirStatus.status === 'running' ? 'Deploying...' :
                                                    currentDirStatus.status === 'success' ? 'Last deploy succeeded' :
                                                        'Last deploy failed'
                                            }
                                        </span>
                                    </div>
                                )}
                                <button
                                    onClick={() => handleDeploy(currentPath)}
                                    disabled={!selectedRef}
                                    className="px-4 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors flex items-center gap-2"
                                >
                                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
                                    </svg>
                                    Deploy
                                </button>
                                <Link
                                    to={`/deployments/${id}/runs`}
                                    className="px-4 py-2 text-sm bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors flex items-center gap-2"
                                >
                                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
                                    </svg>
                                    Deployment Runs
                                </Link>
                            </div>
                        </div>

                        {listing && (
                            <div className="space-y-4">
                                {loadingDirectory ? (
                                    <div className="flex justify-center py-8">
                                        <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600"></div>
                                    </div>
                                ) : (
                                    <>
                                        <div className="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
                                            <div className="bg-gray-50 dark:bg-gray-700 px-4 py-2 border-b border-gray-200 dark:border-gray-600">
                                                <h3 className="font-semibold text-gray-900 dark:text-white">Files & Directories</h3>
                                            </div>
                                            <div className="divide-y divide-gray-200 dark:divide-gray-700">
                                                {listing.files.length === 0 ? (
                                                    <div className="px-4 py-8 text-center text-gray-500 dark:text-gray-400">
                                                        Empty directory
                                                    </div>
                                                ) : (
                                                    listing.files.map((file: FileNode) => {
                                                        return (
                                                            <div
                                                                key={file.path}
                                                                className="px-4 py-3 hover:bg-gray-50 dark:hover:bg-gray-700 flex items-center gap-3"
                                                            >
                                                                {file.is_dir ? (
                                                                    <button
                                                                        onClick={() => navigateTo(file)}
                                                                        className="flex items-center gap-2 text-blue-600 dark:text-blue-400 hover:underline"
                                                                    >
                                                                        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                                                                        </svg>
                                                                        <span className="font-medium">{file.name}</span>
                                                                    </button>
                                                                ) : (
                                                                    <div className="flex items-center gap-2 text-gray-700 dark:text-gray-300">
                                                                        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                                                                        </svg>
                                                                        <span>{file.name}</span>
                                                                    </div>
                                                                )}
                                                            </div>
                                                        );
                                                    })
                                                )}
                                            </div>
                                        </div>

                                        {listing.readme && (
                                            <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-6">
                                                <div className="flex items-center mb-4">
                                                    <svg className="h-5 w-5 text-blue-500 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                                                    </svg>
                                                    <h2 className="text-lg font-semibold text-gray-900 dark:text-white">README.md</h2>
                                                </div>
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
                                                        {listing.readme.replace(/<!--[\s\S]*?-->/g, '')}
                                                    </ReactMarkdown>
                                                </div>
                                            </div>
                                        )}

                                        {listing.has_gitops && (
                                            <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 text-green-700 dark:text-green-400 px-4 py-3 rounded flex items-center gap-2">
                                                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                                                </svg>
                                                <span>GitOps configuration detected in this directory</span>
                                            </div>
                                        )}
                                    </>
                                )}
                            </div>
                        )}
                    </>
                )}
            </div>

            {/* Deployment Modal */}
            {showDeployModal && (
                <DeploymentModal
                    deploymentId={id!}
                    path={deployModalPath}
                    gitRef={selectedRef}
                    onClose={() => setShowDeployModal(false)}
                    onSuccess={() => {
                        showNotification(`Deployment started for ${deployModalPath || '/'} on ${selectedRef}`, 'success');
                        setTimeout(() => loadDirectory(currentPath), 1000);
                    }}
                />
            )}
        </div >
    );
}

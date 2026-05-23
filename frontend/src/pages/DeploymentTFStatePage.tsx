import { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { tfStateApi } from '../api';
import type { TFStateSummary } from '../types';

export default function DeploymentTFStatePage() {
    const { id } = useParams<{ id: string }>();

    const [states, setStates] = useState<TFStateSummary[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [notification, setNotification] = useState<{ message: string; type: 'success' | 'error' } | null>(null);
    const [inspecting, setInspecting] = useState<string | null>(null);
    const [rawState, setRawState] = useState<string | null>(null);
    const [rawWorkspace, setRawWorkspace] = useState<string | null>(null);
    const [actionLoading, setActionLoading] = useState<string | null>(null);

    const showNotification = (message: string, type: 'success' | 'error' = 'success') => {
        setNotification({ message, type });
        setTimeout(() => setNotification(null), 4000);
    };

    const loadStates = async () => {
        if (!id) return;
        try {
            setLoading(true);
            const data = await tfStateApi.getWorkspaces(id);
            setStates(data);
            setError(null);
        } catch (err: unknown) {
            const e = err as { response?: { data?: { error?: string } } };
            setError(e.response?.data?.error || 'Failed to load tfstates');
        } finally {
            setLoading(false);
        }
    };

    const handleInspect = async (workspace: string) => {
        if (!id) return;
        try {
            setInspecting(workspace);
            const data = await tfStateApi.getRaw(id, workspace);
            setRawState(JSON.stringify(data, null, 2));
            setRawWorkspace(workspace);
        } catch (err: unknown) {
            const e = err as { response?: { data?: { error?: string } } };
            showNotification(e.response?.data?.error || 'Failed to fetch state', 'error');
        } finally {
            setInspecting(null);
        }
    };

    const handleForceUnlock = async (workspace: string) => {
        if (!id) return;
        if (!confirm(`Force-unlock state for workspace "${workspace}"? Only do this if you are sure no Terraform process is running.`)) return;
        try {
            setActionLoading(`unlock-${workspace}`);
            await tfStateApi.forceUnlock(id, workspace);
            showNotification(`Workspace "${workspace}" unlocked successfully`, 'success');
            await loadStates();
        } catch (err: unknown) {
            const e = err as { response?: { data?: { error?: string } } };
            showNotification(e.response?.data?.error || 'Failed to unlock', 'error');
        } finally {
            setActionLoading(null);
        }
    };

    const handleDeleteWorkspace = async (workspace: string) => {
        if (!id) return;
        if (!confirm(`Delete all state for workspace "${workspace}"? This cannot be undone.`)) return;
        try {
            setActionLoading(`delete-${workspace}`);
            await tfStateApi.deleteWorkspace(id, workspace);
            showNotification(`Workspace "${workspace}" deleted`, 'success');
            setRawState(null);
            setRawWorkspace(null);
            await loadStates();
        } catch (err: unknown) {
            const e = err as { response?: { data?: { error?: string } } };
            showNotification(e.response?.data?.error || 'Failed to delete workspace', 'error');
        } finally {
            setActionLoading(null);
        }
    };

    useEffect(() => {
        loadStates();
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [id]);

    return (
        <div className="space-y-6">
            {/* Notification Toast */}
            {notification && (
                <div className={`fixed top-4 right-4 z-50 px-6 py-4 rounded-lg shadow-lg border transition-all duration-300 ${
                    notification.type === 'success'
                        ? 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800 text-green-800 dark:text-green-200'
                        : 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800 text-red-800 dark:text-red-200'
                }`}>
                    <div className="flex items-center gap-3">
                        <p className="font-medium">{notification.message}</p>
                        <button onClick={() => setNotification(null)} className="ml-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200">
                            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                                <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
                            </svg>
                        </button>
                    </div>
                </div>
            )}

            <div>
                <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Terraform State</h1>
                <p className="mt-1 text-gray-600 dark:text-gray-400 text-sm">
                    State files stored locally using the platform's built-in HTTP backend.
                </p>
            </div>

            {/* Backend URL info box */}
            <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
                <h3 className="text-sm font-semibold text-blue-800 dark:text-blue-200 mb-2">Backend Configuration</h3>
                <p className="text-xs text-blue-700 dark:text-blue-300 mb-2">
                    Configure the HTTP backend manually in your Terraform/OpenTofu code:
                </p>
                <pre className="text-xs bg-blue-100 dark:bg-blue-900/40 rounded p-3 overflow-x-auto text-blue-900 dark:text-blue-100 font-mono">{`terraform {
  backend "http" {
    address        = "<REGISTRY_URL>/api/tfstate/${id}?workspace=default"
    lock_address   = "<REGISTRY_URL>/api/tfstate/${id}?workspace=default"
    unlock_address = "<REGISTRY_URL>/api/tfstate/${id}?workspace=default"
    lock_method    = "LOCK"
    unlock_method  = "UNLOCK"
  }
}`}</pre>
            </div>

            {error && (
                <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 text-red-700 dark:text-red-400 px-4 py-3 rounded">
                    {error}
                </div>
            )}

            {/* Workspaces table */}
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow">
                <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
                    <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Workspaces</h2>
                    <button
                        onClick={loadStates}
                        className="flex items-center gap-1 px-3 py-1 text-sm text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded-md transition-colors"
                    >
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                        </svg>
                        Refresh
                    </button>
                </div>

                {loading ? (
                    <div className="flex justify-center py-12">
                        <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600"></div>
                    </div>
                ) : states.length === 0 ? (
                    <div className="px-6 py-12 text-center">
                        <svg className="mx-auto h-12 w-12 text-gray-400 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
                        </svg>
                        <p className="text-gray-500 dark:text-gray-400 text-sm">No state files yet.</p>
                        <p className="text-gray-400 dark:text-gray-500 text-xs mt-1">State will appear here after the first successful run.</p>
                    </div>
                ) : (
                    <div className="divide-y divide-gray-200 dark:divide-gray-700">
                        {states.map((s) => (
                            <div key={s.id} className="px-6 py-4 flex items-center justify-between gap-4">
                                <div className="flex items-center gap-4 min-w-0">
                                    <div>
                                        <div className="flex items-center gap-2">
                                            <span className="font-mono text-sm font-semibold text-gray-900 dark:text-white">{s.workspace}</span>
                                            {s.lock_id ? (
                                                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-300">
                                                    <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                                                        <path fillRule="evenodd" d="M5 9V7a5 5 0 0110 0v2a2 2 0 012 2v5a2 2 0 01-2 2H5a2 2 0 01-2-2v-5a2 2 0 012-2zm8-2v2H7V7a3 3 0 016 0z" clipRule="evenodd" />
                                                    </svg>
                                                    Locked
                                                </span>
                                            ) : (
                                                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300">
                                                    <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                                                        <path d="M10 2a5 5 0 00-5 5v2a2 2 0 00-2 2v5a2 2 0 002 2h10a2 2 0 002-2v-5a2 2 0 00-2-2H7V7a3 3 0 015.905-.75 1 1 0 001.937-.5A5.002 5.002 0 0010 2z" />
                                                    </svg>
                                                    Unlocked
                                                </span>
                                            )}
                                        </div>
                                        <div className="mt-1 flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400">
                                            <span>Serial: <strong className="text-gray-700 dark:text-gray-300">{s.state_serial}</strong></span>
                                            <span>Updated: {new Date(s.updated_at).toLocaleString()}</span>
                                            {s.locked_at && <span>Locked since: {new Date(s.locked_at).toLocaleString()}</span>}
                                        </div>
                                    </div>
                                </div>

                                <div className="flex items-center gap-2 flex-shrink-0">
                                    <button
                                        onClick={() => handleInspect(s.workspace)}
                                        disabled={inspecting === s.workspace}
                                        className="px-3 py-1.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-md hover:bg-gray-200 dark:hover:bg-gray-600 disabled:opacity-50 transition-colors"
                                    >
                                        {inspecting === s.workspace ? 'Loading...' : 'Inspect'}
                                    </button>
                                    {s.lock_id && (
                                        <button
                                            onClick={() => handleForceUnlock(s.workspace)}
                                            disabled={actionLoading === `unlock-${s.workspace}`}
                                            className="px-3 py-1.5 text-xs bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-300 rounded-md hover:bg-yellow-200 dark:hover:bg-yellow-900/50 disabled:opacity-50 transition-colors"
                                        >
                                            {actionLoading === `unlock-${s.workspace}` ? 'Unlocking...' : 'Force Unlock'}
                                        </button>
                                    )}
                                    <button
                                        onClick={() => handleDeleteWorkspace(s.workspace)}
                                        disabled={actionLoading === `delete-${s.workspace}`}
                                        className="px-3 py-1.5 text-xs bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 rounded-md hover:bg-red-200 dark:hover:bg-red-900/50 disabled:opacity-50 transition-colors"
                                    >
                                        {actionLoading === `delete-${s.workspace}` ? 'Deleting...' : 'Delete'}
                                    </button>
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </div>

            {/* Raw state inspector */}
            {rawState && rawWorkspace && (
                <div className="bg-white dark:bg-gray-800 rounded-lg shadow">
                    <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
                        <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
                            State: <span className="font-mono text-blue-600 dark:text-blue-400">{rawWorkspace}</span>
                        </h2>
                        <div className="flex items-center gap-2">
                            <button
                                onClick={() => {
                                    const blob = new Blob([rawState], { type: 'application/json' });
                                    const url = URL.createObjectURL(blob);
                                    const a = document.createElement('a');
                                    a.href = url;
                                    a.download = `terraform.tfstate.${rawWorkspace}.json`;
                                    a.click();
                                    URL.revokeObjectURL(url);
                                }}
                                className="px-3 py-1.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-md hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors flex items-center gap-1"
                            >
                                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                                </svg>
                                Download
                            </button>
                            <button
                                onClick={() => { setRawState(null); setRawWorkspace(null); }}
                                className="px-3 py-1.5 text-xs text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 transition-colors"
                            >
                                Close
                            </button>
                        </div>
                    </div>
                    <div className="p-4 overflow-x-auto">
                        <pre className="text-xs font-mono text-gray-800 dark:text-gray-200 bg-gray-50 dark:bg-gray-900 rounded p-4 overflow-x-auto max-h-[500px] overflow-y-auto">
                            {rawState}
                        </pre>
                    </div>
                </div>
            )}
        </div>
    );
}

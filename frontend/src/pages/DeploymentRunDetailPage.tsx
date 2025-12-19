import { useState, useEffect, useRef } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { deploymentsApi } from '../api';
import { DeploymentRun } from '../types';
import AnsiOutput from '../components/AnsiOutput';

export default function DeploymentRunDetailPage() {
    const { id, runId } = useParams<{ id: string; runId: string }>();
    const navigate = useNavigate();
    const [run, setRun] = useState<DeploymentRun | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [approving, setApproving] = useState(false);
    const [cancelling, setCancelling] = useState(false);
    const [deleting, setDeleting] = useState(false);
    const initLogRef = useRef<HTMLDivElement>(null);
    const planLogRef = useRef<HTMLDivElement>(null);
    const applyLogRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        loadRun(true); // Show loading spinner on first load
    }, [id, runId]);

    // Auto-refresh polling when run is active - aggressive polling for near-real-time updates
    useEffect(() => {
        const activeStatuses = ['pending', 'initializing', 'planning', 'awaiting_approval', 'applying'];
        if (!run || !activeStatuses.includes(run.status)) {
            return;
        }

        const interval = setInterval(() => {
            loadRun(false); // Don't show loading spinner on auto-refresh
        }, 1000); // Poll every 1 second for near-real-time log updates

        return () => clearInterval(interval);
    }, [run?.status, id, runId]);

    // Auto-scroll to bottom of active log section
    useEffect(() => {
        if (!run) return;

        if (run.status === 'initializing' && initLogRef.current) {
            initLogRef.current.scrollTop = initLogRef.current.scrollHeight;
        } else if (run.status === 'planning' && planLogRef.current) {
            planLogRef.current.scrollTop = planLogRef.current.scrollHeight;
        } else if (run.status === 'applying' && applyLogRef.current) {
            applyLogRef.current.scrollTop = applyLogRef.current.scrollHeight;
        }
    }, [run?.init_log, run?.plan_log, run?.apply_log, run?.status]);

    const loadRun = async (showLoading = true) => {
        if (!id || !runId) return;

        try {
            if (showLoading) {
                setLoading(true);
            }
            const data = await deploymentsApi.getRun(id, runId);
            setRun(data);
            setError(null);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load run');
        } finally {
            if (showLoading) {
                setLoading(false);
            }
        }
    };

    const downloadLog = (content: string, filename: string) => {
        const blob = new Blob([content], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    };

    const handleApprove = async (approved: boolean) => {
        if (!id || !runId) return;

        setApproving(true);
        try {
            // Send approval/rejection to server
            await deploymentsApi.approveRun(id, runId, {
                approved,
                approved_by: 'User' // TODO: Get from auth context
            });

            // Update local state immediately for smooth transition
            if (run) {
                setRun({
                    ...run,
                    status: approved ? 'applying' : 'cancelled',
                    approved_by: approved ? 'User' : 'REJECTED'
                });
            }
        } catch (err) {
            console.error('Approval error:', err);
            setError(err instanceof Error ? err.message : 'Failed to approve run');
        } finally {
            setApproving(false);
        }
    };

    const handleCancel = async () => {
        if (!id || !runId) return;

        if (!confirm('Are you sure you want to cancel this deployment? This action cannot be undone.')) {
            return;
        }

        try {
            setCancelling(true);
            await deploymentsApi.cancelRun(id, runId);
            await loadRun();
        } catch (err: any) {
            console.error('Cancel error:', err);
            const errorMsg = err.response?.data?.error || err.message || 'Failed to cancel run';
            setError(`Cancel failed: ${errorMsg}`);
        } finally {
            setCancelling(false);
        }
    };

    const handleDelete = async () => {
        if (!id || !runId) return;

        if (!confirm('Are you sure you want to delete this deployment run? This action cannot be undone.')) {
            return;
        }

        try {
            setDeleting(true);
            await deploymentsApi.deleteRun(id, runId);
            navigate(`/deployments/${id}/runs`);
        } catch (err: any) {
            console.error('Delete error:', err);
            const errorMsg = err.response?.data?.error || err.message || 'Failed to delete run';
            setError(`Delete failed: ${errorMsg}`);
        } finally {
            setDeleting(false);
        }
    };

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'success':
                return 'bg-green-100 dark:bg-green-900/20 text-green-800 dark:text-green-300';
            case 'pending':
            case 'initializing':
            case 'planning':
            case 'applying':
                return 'bg-yellow-100 dark:bg-yellow-900/20 text-yellow-800 dark:text-yellow-300';
            case 'awaiting_approval':
                return 'bg-purple-100 dark:bg-purple-900/20 text-purple-800 dark:text-purple-300';
            case 'failed':
                return 'bg-red-100 dark:bg-red-900/20 text-red-800 dark:text-red-300';
            case 'cancelled':
                return 'bg-gray-100 dark:bg-gray-900/20 text-gray-800 dark:text-gray-300';
            default:
                return 'bg-blue-100 dark:bg-blue-900/20 text-blue-800 dark:text-blue-300';
        }
    };

    if (loading && !run) {
        return (
            <div className="flex justify-center py-12">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
            </div>
        );
    }

    if (error && !run) {
        return (
            <div className="max-w-7xl mx-auto">
                <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
                    <p className="text-red-700 dark:text-red-400">{error}</p>
                </div>
            </div>
        );
    }

    if (!run) return null;

    const canCancel = ['pending', 'initializing', 'planning', 'awaiting_approval', 'applying'].includes(run.status);
    const canDelete = !canCancel; // Can only delete completed/failed/cancelled runs

    return (
        <div className="max-w-7xl mx-auto space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <div className="flex items-center gap-3 mb-2">
                        <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Deployment Run</h1>
                        <span className={`px-3 py-1 rounded-full text-sm font-medium ${getStatusColor(run.status)}`}>
                            {run.status.toUpperCase()}
                        </span>
                    </div>
                    <p className="text-gray-600 dark:text-gray-400">
                        <span className="font-mono">{run.path || '/'}</span> @ <span className="font-mono">{run.ref}</span>
                    </p>
                </div>
                <div className="flex gap-3">
                    {canCancel && (
                        <button
                            onClick={handleCancel}
                            disabled={cancelling}
                            className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors flex items-center gap-2"
                        >
                            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                            </svg>
                            {cancelling ? 'Cancelling...' : 'Stop Run'}
                        </button>
                    )}
                    {canDelete && (
                        <button
                            onClick={handleDelete}
                            disabled={deleting}
                            className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors flex items-center gap-2"
                        >
                            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                            </svg>
                            {deleting ? 'Deleting...' : 'Delete Run'}
                        </button>
                    )}
                    <Link
                        to={`/deployments/${id}/runs`}
                        className="px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors"
                    >
                        Back to Runs
                    </Link>
                </div>
            </div>

            {/* Run Info */}
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">Run Information</h2>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <div>
                        <p className="text-sm text-gray-500 dark:text-gray-400">Tool</p>
                        <p className="font-medium text-gray-900 dark:text-white capitalize">{run.tool}</p>
                    </div>
                    <div>
                        <p className="text-sm text-gray-500 dark:text-gray-400">Created</p>
                        <p className="font-medium text-gray-900 dark:text-white">{new Date(run.created_at).toLocaleString()}</p>
                    </div>
                    <div>
                        <p className="text-sm text-gray-500 dark:text-gray-400">Started</p>
                        <p className="font-medium text-gray-900 dark:text-white">
                            {run.started_at ? new Date(run.started_at).toLocaleString() : '-'}
                        </p>
                    </div>
                    <div>
                        <p className="text-sm text-gray-500 dark:text-gray-400">Completed</p>
                        <p className="font-medium text-gray-900 dark:text-white">
                            {run.completed_at ? new Date(run.completed_at).toLocaleString() : '-'}
                        </p>
                    </div>
                </div>

                {run.env_vars && Object.keys(run.env_vars).length > 0 && (
                    <div className="mt-4">
                        <p className="text-sm text-gray-500 dark:text-gray-400 mb-2">Environment Variables</p>
                        <div className="bg-gray-50 dark:bg-gray-900 rounded p-3 font-mono text-sm">
                            {Object.keys(run.env_vars).map((key) => (
                                <div key={key} className="text-gray-900 dark:text-white">
                                    <span className="text-blue-600 dark:text-blue-400">{key}</span>=
                                    <span className="text-gray-500 dark:text-gray-400">********</span>
                                </div>
                            ))}
                        </div>
                    </div>
                )}

                {run.tfvars_files && run.tfvars_files.length > 0 && (
                    <div className="mt-4">
                        <p className="text-sm text-gray-500 dark:text-gray-400 mb-2">Terraform Variables Files</p>
                        <div className="bg-gray-50 dark:bg-gray-900 rounded p-3">
                            <div className="flex flex-wrap gap-2">
                                {run.tfvars_files.map((file, index) => (
                                    <span key={index} className="px-3 py-1 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 rounded-full text-sm font-mono">
                                        {file}
                                    </span>
                                ))}
                            </div>
                        </div>
                    </div>
                )}

                {run.init_flags && (
                    <div className="mt-4">
                        <p className="text-sm text-gray-500 dark:text-gray-400 mb-2">Custom Init Flags</p>
                        <div className="bg-gray-50 dark:bg-gray-900 rounded p-3 font-mono text-sm text-gray-900 dark:text-white">
                            {run.init_flags}
                        </div>
                    </div>
                )}

                {run.plan_flags && (
                    <div className="mt-4">
                        <p className="text-sm text-gray-500 dark:text-gray-400 mb-2">Custom Plan Flags</p>
                        <div className="bg-gray-50 dark:bg-gray-900 rounded p-3 font-mono text-sm text-gray-900 dark:text-white">
                            {run.plan_flags}
                        </div>
                    </div>
                )}

                {run.approved_by && (
                    <div className="mt-4">
                        <p className="text-sm text-gray-500 dark:text-gray-400">Approved By</p>
                        <p className="font-medium text-gray-900 dark:text-white">
                            {run.approved_by} at {run.approved_at ? new Date(run.approved_at).toLocaleString() : ''}
                        </p>
                    </div>
                )}
            </div>

            {/* Error Message */}
            {run.error_message && (
                <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
                    <h3 className="text-lg font-semibold text-red-900 dark:text-red-300 mb-2">Error</h3>
                    <pre className="text-red-700 dark:text-red-400 whitespace-pre-wrap font-mono text-sm">
                        {run.error_message}
                    </pre>
                </div>
            )}

            {/* Approval Section - Always shown when awaiting approval */}
            {run.status === 'awaiting_approval' && (
                <div className="bg-purple-50 dark:bg-purple-900/20 border border-purple-200 dark:border-purple-800 rounded-lg p-6">
                    <h3 className="text-lg font-semibold text-purple-900 dark:text-purple-300 mb-4">
                        Plan Awaiting Approval
                    </h3>
                    <p className="text-purple-700 dark:text-purple-400 mb-4">
                        Review the plan output below and approve or reject this deployment.
                    </p>
                    <div className="flex gap-3">
                        <button
                            onClick={() => handleApprove(true)}
                            disabled={approving}
                            className="px-6 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors flex items-center gap-2"
                        >
                            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                            </svg>
                            Approve
                        </button>
                        <button
                            onClick={() => handleApprove(false)}
                            disabled={approving}
                            className="px-6 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors flex items-center gap-2"
                        >
                            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                            </svg>
                            Reject
                        </button>
                    </div>
                </div>
            )}

            {/* Init Log */}
            {(run.init_log || ['initializing', 'planning', 'awaiting_approval', 'applying', 'success', 'failed', 'cancelled'].includes(run.status)) && (
                <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
                    <div className="bg-gray-50 dark:bg-gray-700 px-4 py-3 border-b border-gray-200 dark:border-gray-600 flex items-center justify-between">
                        <h3 className="font-semibold text-gray-900 dark:text-white font-mono">
                            {run.tool} init{run.init_flags ? ` ${run.init_flags}` : ''}
                            {run.status === 'initializing' && (
                                <span className="ml-2 text-yellow-600 dark:text-yellow-400 text-sm">Running...</span>
                            )}
                        </h3>
                        {run.init_log && (
                            <button
                                onClick={() => downloadLog(run.init_log, `${run.tool}-init-${runId}.log`)}
                                className="px-3 py-1 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors flex items-center gap-1 text-sm"
                                title="Download log"
                            >
                                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                                </svg>
                                Download
                            </button>
                        )}
                    </div>
                    <div ref={initLogRef} className="p-4 overflow-x-auto max-h-[600px] overflow-y-auto">
                        {run.init_log ? (
                            <AnsiOutput content={run.init_log} />
                        ) : (
                            <div className="text-gray-500 dark:text-gray-400 font-mono text-sm">
                                {run.status === 'initializing' ? (
                                    <span>Waiting for logs to stream...</span>
                                ) : (
                                    <span>Waiting to start...</span>
                                )}
                            </div>
                        )}
                    </div>
                </div>
            )}

            {/* Plan Log */}
            {(run.plan_log || ['planning', 'awaiting_approval', 'applying', 'success', 'failed', 'cancelled'].includes(run.status)) && (
                <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
                    <div className="bg-gray-50 dark:bg-gray-700 px-4 py-3 border-b border-gray-200 dark:border-gray-600 flex items-center justify-between">
                        <h3 className="font-semibold text-gray-900 dark:text-white font-mono">
                            {run.tool} plan -out=tfplan{run.tfvars_files && run.tfvars_files.length > 0 && run.tfvars_files.map(f => ` -var-file=${f}`).join('')}{run.plan_flags ? ` ${run.plan_flags}` : ''}
                            {run.status === 'planning' && (
                                <span className="ml-2 text-yellow-600 dark:text-yellow-400 text-sm">Running...</span>
                            )}
                        </h3>
                        {run.plan_log && (
                            <button
                                onClick={() => downloadLog(run.plan_log, `${run.tool}-plan-${runId}.log`)}
                                className="px-3 py-1 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors flex items-center gap-1 text-sm"
                                title="Download log"
                            >
                                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                                </svg>
                                Download
                            </button>
                        )}
                    </div>
                    <div ref={planLogRef} className="p-4 overflow-x-auto max-h-[600px] overflow-y-auto">
                        {run.plan_log ? (
                            <AnsiOutput content={run.plan_log} />
                        ) : (
                            <div className="text-gray-500 dark:text-gray-400 font-mono text-sm">
                                {run.status === 'planning' ? (
                                    <span>Waiting for logs to stream...</span>
                                ) : (
                                    <span>Waiting to start...</span>
                                )}
                            </div>
                        )}
                    </div>
                </div>
            )}

            {/* Apply Log */}
            {(run.apply_log || ['applying', 'success', 'failed'].includes(run.status)) && (
                <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
                    <div className="bg-gray-50 dark:bg-gray-700 px-4 py-3 border-b border-gray-200 dark:border-gray-600 flex items-center justify-between">
                        <h3 className="font-semibold text-gray-900 dark:text-white font-mono">
                            {run.tool} apply tfplan
                            {run.status === 'applying' && (
                                <span className="ml-2 text-yellow-600 dark:text-yellow-400 text-sm">Running...</span>
                            )}
                        </h3>
                        {run.apply_log && (
                            <button
                                onClick={() => downloadLog(run.apply_log, `${run.tool}-apply-${runId}.log`)}
                                className="px-3 py-1 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors flex items-center gap-1 text-sm"
                                title="Download log"
                            >
                                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                                </svg>
                                Download
                            </button>
                        )}
                    </div>
                    <div ref={applyLogRef} className="p-4 overflow-x-auto max-h-[600px] overflow-y-auto">
                        {run.apply_log ? (
                            <AnsiOutput content={run.apply_log} />
                        ) : (
                            <div className="text-gray-500 dark:text-gray-400 font-mono text-sm">
                                {run.status === 'applying' ? (
                                    <span>Waiting for logs to stream...</span>
                                ) : (
                                    <span>Waiting to start...</span>
                                )}
                            </div>
                        )}
                    </div>
                </div>
            )}

            {/* Terraform/Tofu Outputs */}
            {run.apply_output && (
                <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
                    <div className="bg-gray-50 dark:bg-gray-700 px-4 py-3 border-b border-gray-200 dark:border-gray-600 flex items-center justify-between">
                        <h3 className="font-semibold text-gray-900 dark:text-white font-mono">{run.tool} output -json</h3>
                        <button
                            onClick={() => downloadLog(run.apply_output, `${run.tool}-output-${runId}.json`)}
                            className="px-3 py-1 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors flex items-center gap-1 text-sm"
                            title="Download output"
                        >
                            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                            </svg>
                            Download
                        </button>
                    </div>
                    <div className="p-4 overflow-x-auto max-h-[600px] overflow-y-auto">
                        <AnsiOutput content={run.apply_output} />
                    </div>
                </div>
            )}
        </div>
    );
}

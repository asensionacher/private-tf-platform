import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { deploymentsApi } from '../api';
import { DeploymentRun } from '../types';
import AnsiOutput from '../components/AnsiOutput';

export default function DeploymentRunDetailPage() {
    const { id, runId } = useParams<{ id: string; runId: string }>();
    const [run, setRun] = useState<DeploymentRun | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [approving, setApproving] = useState(false);

    useEffect(() => {
        loadRun();

        // Auto-refresh every 3 seconds if run is active
        const interval = setInterval(() => {
            const activeStatuses = ['pending', 'initializing', 'planning', 'awaiting_approval', 'applying'];
            if (run && activeStatuses.includes(run.status)) {
                loadRun();
            }
        }, 3000);

        return () => clearInterval(interval);
    }, [id, runId]);

    const loadRun = async () => {
        if (!id || !runId) return;

        try {
            setLoading(true);
            const data = await deploymentsApi.getRun(id, runId);
            setRun(data);
            setError(null);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load run');
        } finally {
            setLoading(false);
        }
    };

    const handleApprove = async (approved: boolean) => {
        if (!id || !runId) return;

        try {
            setApproving(true);
            await deploymentsApi.approveRun(id, runId, {
                approved,
                approved_by: 'User' // TODO: Get from auth context
            });
            await loadRun();
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to approve run');
        } finally {
            setApproving(false);
        }
    };

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'success':
                return 'bg-green-100 dark:bg-green-900/20 text-green-800 dark:text-green-300';
            case 'initializing':
            case 'planning':
            case 'applying':
                return 'bg-yellow-100 dark:bg-yellow-900/20 text-yellow-800 dark:text-yellow-300';
            case 'awaiting_approval':
                return 'bg-orange-100 dark:bg-orange-900/20 text-orange-800 dark:text-orange-300';
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
                <Link
                    to={`/deployments/${id}/runs`}
                    className="px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors"
                >
                    Back to Runs
                </Link>
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
                            {Object.entries(run.env_vars).map(([key, value]) => (
                                <div key={key} className="text-gray-900 dark:text-white">
                                    <span className="text-blue-600 dark:text-blue-400">{key}</span>=
                                    <span className="text-green-600 dark:text-green-400">{value}</span>
                                </div>
                            ))}
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

            {/* Approval Section */}
            {run.status === 'awaiting_approval' && (
                <div className="bg-orange-50 dark:bg-orange-900/20 border border-orange-200 dark:border-orange-800 rounded-lg p-6">
                    <h3 className="text-lg font-semibold text-orange-900 dark:text-orange-300 mb-4">
                        Plan Awaiting Approval
                    </h3>
                    <p className="text-orange-700 dark:text-orange-400 mb-4">
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

            {/* Error Message */}
            {run.error_message && (
                <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
                    <h3 className="text-lg font-semibold text-red-900 dark:text-red-300 mb-2">Error</h3>
                    <pre className="text-red-700 dark:text-red-400 whitespace-pre-wrap font-mono text-sm">
                        {run.error_message}
                    </pre>
                </div>
            )}

            {/* Init Log */}
            {run.init_log && (
                <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
                    <div className="bg-gray-50 dark:bg-gray-700 px-4 py-3 border-b border-gray-200 dark:border-gray-600">
                        <h3 className="font-semibold text-gray-900 dark:text-white">Init Output</h3>
                    </div>
                    <div className="p-4 overflow-x-auto">
                        <AnsiOutput content={run.init_log} />
                    </div>
                </div>
            )}

            {/* Plan Log */}
            {run.plan_log && (
                <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
                    <div className="bg-gray-50 dark:bg-gray-700 px-4 py-3 border-b border-gray-200 dark:border-gray-600">
                        <h3 className="font-semibold text-gray-900 dark:text-white">Plan Output</h3>
                    </div>
                    <div className="p-4 overflow-x-auto">
                        <AnsiOutput content={run.plan_log} />
                    </div>
                </div>
            )}

            {/* Apply Log */}
            {run.apply_log && (
                <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
                    <div className="bg-gray-50 dark:bg-gray-700 px-4 py-3 border-b border-gray-200 dark:border-gray-600">
                        <h3 className="font-semibold text-gray-900 dark:text-white">Apply Output</h3>
                    </div>
                    <div className="p-4 overflow-x-auto">
                        <AnsiOutput content={run.apply_log} />
                    </div>
                </div>
            )}
        </div>
    );
}

import { useState, useEffect } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { deploymentsApi } from '../api';
import { DeploymentRun } from '../types';

export default function DeploymentRunsPage() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();

    const [runs, setRuns] = useState<DeploymentRun[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [deleting, setDeleting] = useState<string | null>(null);

    useEffect(() => {
        loadRuns(true); // Show loading on initial load
    }, [id]);

    useEffect(() => {
        // Auto-refresh every 3 seconds if there are running/pending deployments
        const activeStatuses = ['pending', 'initializing', 'planning', 'awaiting_approval', 'applying'];
        const hasActiveRuns = runs.some(run => activeStatuses.includes(run.status));
        
        if (hasActiveRuns) {
            const interval = setInterval(() => {
                loadRuns(false); // Don't show loading spinner on auto-refresh
            }, 3000);

            return () => clearInterval(interval);
        }
    }, [runs, id]);

    const loadRuns = async (showLoading = true) => {
        if (!id) return;

        try {
            if (showLoading) {
                setLoading(true);
            }
            const data = await deploymentsApi.getRuns(id!);
            setRuns(data);
            setError(null);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load runs');
        } finally {
            if (showLoading) {
                setLoading(false);
            }
        }
    };

    const handleDelete = async (runId: string, e: React.MouseEvent) => {
        e.stopPropagation(); // Prevent row click navigation

        if (!confirm('Are you sure you want to delete this deployment run? This action cannot be undone.')) {
            return;
        }

        try {
            setDeleting(runId);
            await deploymentsApi.deleteRun(id!, runId);
            await loadRuns(true); // Refresh the list with loading indicator
        } catch (err: any) {
            const errorMsg = err.response?.data?.error || err.message || 'Failed to delete run';
            setError(`Delete failed: ${errorMsg}`);
        } finally {
            setDeleting(null);
        }
    };

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'success':
                return 'bg-green-500';
            case 'initializing':
            case 'planning':
            case 'applying':
                return 'bg-yellow-500 animate-pulse';
            case 'awaiting_approval':
                return 'bg-orange-500 animate-pulse';
            case 'failed':
                return 'bg-red-500';
            case 'cancelled':
                return 'bg-gray-500';
            default:
                return 'bg-blue-500';
        }
    };

    const getStatusTextColor = (status: string) => {
        switch (status) {
            case 'success':
                return 'text-green-600 dark:text-green-400';
            case 'initializing':
            case 'planning':
            case 'applying':
                return 'text-yellow-600 dark:text-yellow-400';
            case 'awaiting_approval':
                return 'text-orange-600 dark:text-orange-400';
            case 'failed':
                return 'text-red-600 dark:text-red-400';
            case 'cancelled':
                return 'text-gray-600 dark:text-gray-400';
            default:
                return 'text-blue-600 dark:text-blue-400';
        }
    };

    return (
        <div className="max-w-7xl mx-auto">
            <div className="mb-6">
                <div className="flex items-center justify-between mb-4">
                    <div>
                        <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Deployment Runs</h1>
                    </div>
                    <Link
                        to={`/deployments/${id}`}
                        className="px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors"
                    >
                        Back to Deployment
                    </Link>
                </div>
            </div>

            {loading && runs.length === 0 ? (
                <div className="flex justify-center py-12">
                    <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
                </div>
            ) : error ? (
                <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
                    <p className="text-red-700 dark:text-red-400">{error}</p>
                </div>
            ) : runs.length === 0 ? (
                <div className="text-center py-12">
                    <svg className="w-16 h-16 mx-auto mb-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
                    </svg>
                    <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-2">No Deployment Runs</h2>
                    <p className="text-gray-600 dark:text-gray-400">
                        No deployment runs found for this path
                    </p>
                </div>
            ) : (
                <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
                    <div className="overflow-x-auto">
                        <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                            <thead className="bg-gray-50 dark:bg-gray-700">
                                <tr>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                        Status
                                    </th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                        Path
                                    </th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                        Branch/Tag
                                    </th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                        Created
                                    </th>
                                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                        Duration
                                    </th>
                                    <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                        Actions
                                    </th>
                                </tr>
                            </thead>
                            <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                                {runs.map((run) => (
                                    <tr
                                        key={run.id}
                                        className="hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer"
                                        onClick={() => navigate(`/deployments/${id}/runs/${run.id}`)}
                                    >
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <div className="flex items-center gap-3">
                                                <span className={`inline-flex h-3 w-3 rounded-full ${getStatusColor(run.status)}`}></span>
                                                <span className={`text-sm font-medium ${getStatusTextColor(run.status)}`}>
                                                    {run.status.toUpperCase()}
                                                </span>
                                            </div>
                                        </td>
                                        <td className="px-6 py-4">
                                            <div className="text-sm font-mono text-gray-900 dark:text-white">
                                                {run.path || '/'}
                                            </div>
                                            {run.error_message && (
                                                <div className="mt-1 text-xs text-red-600 dark:text-red-400">
                                                    {run.error_message}
                                                </div>
                                            )}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200">
                                                {run.ref}
                                            </span>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                                            {new Date(run.created_at).toLocaleString()}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                                            {run.completed_at ? (
                                                `${Math.round((new Date(run.completed_at).getTime() - new Date(run.created_at).getTime()) / 1000)}s`
                                            ) : ['pending', 'initializing', 'planning', 'awaiting_approval', 'applying'].includes(run.status) ? (
                                                <span className="text-yellow-600 dark:text-yellow-400">In Progress...</span>
                                            ) : (
                                                '-'
                                            )}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                                            {!['pending', 'initializing', 'planning', 'awaiting_approval', 'applying'].includes(run.status) && (
                                                <button
                                                    onClick={(e) => handleDelete(run.id, e)}
                                                    disabled={deleting === run.id}
                                                    className="text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-300 disabled:opacity-50 disabled:cursor-not-allowed"
                                                    title="Delete run"
                                                >
                                                    {deleting === run.id ? (
                                                        <svg className="w-5 h-5 animate-spin" fill="none" viewBox="0 0 24 24">
                                                            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                                                            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                                                        </svg>
                                                    ) : (
                                                        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                                        </svg>
                                                    )}
                                                </button>
                                            )}
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                </div>
            )}
        </div>
    );
}

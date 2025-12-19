import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { deploymentsApi, namespacesApi } from '../api';
import type { Deployment, Namespace } from '../types';

export default function DeploymentsPage() {
    const [deployments, setDeployments] = useState<Deployment[]>([]);
    const [namespaces, setNamespaces] = useState<Namespace[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [showCreateForm, setShowCreateForm] = useState(false);
    const [formData, setFormData] = useState({
        namespace_id: '',
        name: '',
        description: '',
        git_url: '',
        is_private: false,
        git_username: '',
        git_password: '',
    });

    useEffect(() => {
        loadData(true); // Show loading on initial load
        
        // Auto-refresh every 5 seconds to keep deployment statuses updated
        const interval = setInterval(() => {
            loadData(false); // Don't show loading spinner on auto-refresh
        }, 5000);

        return () => clearInterval(interval);
    }, []);

    const loadData = async (showLoading = true) => {
        try {
            if (showLoading) {
                setLoading(true);
            }
            const [deploymentsData, namespacesData] = await Promise.all([
                deploymentsApi.getAll(),
                namespacesApi.getAll(),
            ]);
            setDeployments(deploymentsData);
            setNamespaces(namespacesData);
            setError(null);
        } catch (err: any) {
            setError(err.response?.data?.error || 'Failed to load deployments');
        } finally {
            if (showLoading) {
                setLoading(false);
            }
        }
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        try {
            await deploymentsApi.create(formData);
            setShowCreateForm(false);
            setFormData({
                namespace_id: '',
                name: '',
                description: '',
                git_url: '',
                is_private: false,
                git_username: '',
                git_password: '',
            });
            loadData();
        } catch (err: any) {
            setError(err.response?.data?.error || 'Failed to create deployment');
        }
    };

    const handleDelete = async (id: string) => {
        if (!confirm('Are you sure you want to delete this deployment?')) return;
        try {
            await deploymentsApi.delete(id);
            loadData();
        } catch (err: any) {
            setError(err.response?.data?.error || 'Failed to delete deployment');
        }
    };

    if (loading) {
        return (
            <div className="flex items-center justify-center h-64">
                <div className="text-gray-500 dark:text-gray-400">Loading deployments...</div>
            </div>
        );
    }

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Deployments</h1>
                    <p className="mt-2 text-gray-600 dark:text-gray-400">
                        Manage IaC deployments from Git repositories
                    </p>
                </div>
                <button
                    onClick={() => setShowCreateForm(true)}
                    className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600"
                >
                    New Deployment
                </button>
            </div>

            {error && (
                <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 text-red-700 dark:text-red-400 px-4 py-3 rounded">
                    {error}
                </div>
            )}

            {showCreateForm && (
                <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
                    <div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto">
                        <h2 className="text-2xl font-bold mb-4 text-gray-900 dark:text-white">Create Deployment</h2>
                        <form onSubmit={handleSubmit} className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    Namespace
                                </label>
                                <select
                                    required
                                    value={formData.namespace_id}
                                    onChange={(e) => setFormData({ ...formData, namespace_id: e.target.value })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                                >
                                    <option value="">Select namespace...</option>
                                    {namespaces.map((ns) => (
                                        <option key={ns.id} value={ns.id}>
                                            {ns.name}
                                        </option>
                                    ))}
                                </select>
                            </div>

                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    Name
                                </label>
                                <input
                                    type="text"
                                    required
                                    value={formData.name}
                                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                                    placeholder="my-iac-deployment"
                                />
                            </div>

                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    Description
                                </label>
                                <textarea
                                    value={formData.description}
                                    onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                                    rows={2}
                                    placeholder="Description of this deployment"
                                />
                            </div>

                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    Git URL
                                </label>
                                <input
                                    type="url"
                                    required
                                    value={formData.git_url}
                                    onChange={(e) => setFormData({ ...formData, git_url: e.target.value })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                                    placeholder="https://github.com/org/repo or https://dev.azure.com/org/project/_git/repo"
                                />
                            </div>

                            <div className="flex items-center">
                                <input
                                    type="checkbox"
                                    id="is_private"
                                    checked={formData.is_private}
                                    onChange={(e) => setFormData({ ...formData, is_private: e.target.checked })}
                                    className="mr-2"
                                />
                                <label htmlFor="is_private" className="text-sm font-medium text-gray-700 dark:text-gray-300">
                                    Private repository (requires authentication)
                                </label>
                            </div>

                            {formData.is_private && (
                                <>
                                    <div>
                                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                            Git Username
                                        </label>
                                        <input
                                            type="text"
                                            value={formData.git_username}
                                            onChange={(e) => setFormData({ ...formData, git_username: e.target.value })}
                                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                                            placeholder="git username or email"
                                        />
                                    </div>

                                    <div>
                                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                            Git Password / Personal Access Token
                                        </label>
                                        <input
                                            type="password"
                                            value={formData.git_password}
                                            onChange={(e) => setFormData({ ...formData, git_password: e.target.value })}
                                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                                            placeholder="Personal Access Token"
                                        />
                                    </div>
                                </>
                            )}

                            <div className="flex gap-3 pt-4">
                                <button
                                    type="submit"
                                    className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600"
                                >
                                    Create
                                </button>
                                <button
                                    type="button"
                                    onClick={() => setShowCreateForm(false)}
                                    className="flex-1 px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600"
                                >
                                    Cancel
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            )}

            <div className="grid gap-4">
                {deployments.length === 0 ? (
                    <div className="text-center py-12 text-gray-500 dark:text-gray-400">
                        No deployments yet. Create one to get started.
                    </div>
                ) : (
                    deployments.map((deployment) => (
                        <div
                            key={deployment.id}
                            className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 hover:shadow-lg transition-shadow"
                        >
                            <div className="flex items-start justify-between">
                                <div className="flex-1">
                                    <Link
                                        to={`/deployments/${deployment.id}`}
                                        className="text-xl font-semibold text-blue-600 dark:text-blue-400 hover:underline"
                                    >
                                        {deployment.name}
                                    </Link>
                                    {deployment.namespace && (
                                        <span className="ml-3 text-sm text-gray-500 dark:text-gray-400">
                                            @{deployment.namespace}
                                        </span>
                                    )}
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
                                        {deployment.is_private && (
                                            <span className="flex items-center gap-1">
                                                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                                                </svg>
                                                Private
                                            </span>
                                        )}
                                    </div>
                                </div>
                                <button
                                    onClick={() => handleDelete(deployment.id)}
                                    className="ml-4 px-3 py-1 text-sm text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20 rounded"
                                >
                                    Delete
                                </button>
                            </div>
                        </div>
                    ))
                )}
            </div>
        </div>
    );
}

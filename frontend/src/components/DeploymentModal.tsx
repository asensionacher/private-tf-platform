import { useState, useEffect } from 'react';
import { deploymentsApi } from '../api';

interface DeploymentModalProps {
    deploymentId: string;
    path: string;
    gitRef: string;
    onClose: () => void;
    onSuccess: () => void;
}

export default function DeploymentModal({ deploymentId, path, gitRef, onClose, onSuccess }: DeploymentModalProps) {
    const [tool, setTool] = useState<'terraform' | 'tofu'>('terraform');
    const [envVars, setEnvVars] = useState<Array<{ key: string; value: string }>>([]);
    const [availableTfvars, setAvailableTfvars] = useState<string[]>([]);
    const [selectedTfvars, setSelectedTfvars] = useState<string[]>([]);
    const [initFlags, setInitFlags] = useState<string>('');
    const [planFlags, setPlanFlags] = useState<string>('');
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);

    // Load available tfvars files
    useEffect(() => {
        const loadTfvars = async () => {
            try {
                const result = await deploymentsApi.getTfvarsFiles(deploymentId, gitRef, path);
                setAvailableTfvars(result.tfvars_files || []);
            } catch (err) {
                console.error('Failed to load tfvars files:', err);
            }
        };
        loadTfvars();
    }, [deploymentId, gitRef, path]);

    // Load last deployment configuration from localStorage
    useEffect(() => {
        const storageKey = `lastDeployment_${deploymentId}`;
        const savedConfig = localStorage.getItem(storageKey);
        if (savedConfig) {
            try {
                const config = JSON.parse(savedConfig);
                if (config.tool) setTool(config.tool);
                // Load env vars with values (values are hidden but reused)
                if (config.envVars) setEnvVars(config.envVars);
                if (config.selectedTfvars) setSelectedTfvars(config.selectedTfvars);
                if (config.initFlags) setInitFlags(config.initFlags);
                if (config.planFlags) setPlanFlags(config.planFlags);
            } catch (err) {
                console.error('Failed to load saved configuration:', err);
            }
        }
    }, [deploymentId]);

    const handleAddEnvVar = () => {
        setEnvVars([...envVars, { key: '', value: '' }]);
    };

    const handleRemoveEnvVar = (index: number) => {
        setEnvVars(envVars.filter((_, i) => i !== index));
    };

    const handleEnvVarChange = (index: number, field: 'key' | 'value', value: string) => {
        const newEnvVars = [...envVars];
        newEnvVars[index][field] = value;
        setEnvVars(newEnvVars);
    };

    const handleTfvarsToggle = (tfvarsFile: string) => {
        if (selectedTfvars.includes(tfvarsFile)) {
            setSelectedTfvars(selectedTfvars.filter(f => f !== tfvarsFile));
        } else {
            setSelectedTfvars([...selectedTfvars, tfvarsFile]);
        }
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError(null);
        setIsSubmitting(true);

        try {
            // Convert env vars array to object
            const envVarsObj: Record<string, string> = {};
            envVars.forEach(({ key, value }) => {
                if (key.trim()) {
                    envVarsObj[key] = value;
                }
            });

            // Save configuration to localStorage (including values for reuse)
            const storageKey = `lastDeployment_${deploymentId}`;
            localStorage.setItem(storageKey, JSON.stringify({
                tool,
                envVars, // Store full key-value pairs for reuse
                selectedTfvars,
                initFlags,
                planFlags,
            }));

            await deploymentsApi.createRun(deploymentId, {
                path,
                ref: gitRef,
                tool,
                env_vars: Object.keys(envVarsObj).length > 0 ? envVarsObj : undefined,
                tfvars_files: selectedTfvars.length > 0 ? selectedTfvars : undefined,
                init_flags: initFlags.trim() || undefined,
                plan_flags: planFlags.trim() || undefined,
            });

            onSuccess();
            onClose();
        } catch (err: any) {
            setError(err.response?.data?.error || 'Failed to start deployment');
        } finally {
            setIsSubmitting(false);
        }
    };

    return (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" onClick={onClose}>
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
                <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex justify-between items-center">
                    <div>
                        <h2 className="text-xl font-bold text-gray-900 dark:text-white">Configure Deployment</h2>
                        <p className="text-sm text-gray-600 dark:text-gray-400 font-mono mt-1">
                            {path || '/'} @ {gitRef}
                        </p>
                    </div>
                    <button
                        onClick={onClose}
                        className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
                    >
                        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>

                <form onSubmit={handleSubmit} className="p-6 space-y-6">
                    {error && (
                        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 text-red-700 dark:text-red-400 px-4 py-3 rounded">
                            {error}
                        </div>
                    )}

                    <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                            IaC Tool
                        </label>
                        <div className="grid grid-cols-2 gap-4">
                            <button
                                type="button"
                                onClick={() => setTool('terraform')}
                                className={`px-4 py-3 border-2 rounded-lg transition-colors ${tool === 'terraform'
                                    ? 'border-blue-600 bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300'
                                    : 'border-gray-300 dark:border-gray-600 hover:border-gray-400'
                                    }`}
                            >
                                <div className="font-semibold">Terraform</div>
                                <div className="text-xs text-gray-500 dark:text-gray-400">HashiCorp</div>
                            </button>
                            <button
                                type="button"
                                onClick={() => setTool('tofu')}
                                className={`px-4 py-3 border-2 rounded-lg transition-colors ${tool === 'tofu'
                                    ? 'border-blue-600 bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300'
                                    : 'border-gray-300 dark:border-gray-600 hover:border-gray-400'
                                    }`}
                            >
                                <div className="font-semibold">OpenTofu</div>
                                <div className="text-xs text-gray-500 dark:text-gray-400">Open Source</div>
                            </button>
                        </div>
                    </div>

                    <div>
                        <div className="flex items-center justify-between mb-2">
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                                Environment Variables
                            </label>
                            <button
                                type="button"
                                onClick={handleAddEnvVar}
                                className="text-sm text-blue-600 dark:text-blue-400 hover:underline flex items-center gap-1"
                            >
                                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                                </svg>
                                Add Variable
                            </button>
                        </div>

                        {envVars.length === 0 ? (
                            <p className="text-sm text-gray-500 dark:text-gray-400 italic">
                                No environment variables added
                            </p>
                        ) : (
                            <>
                                <div className="space-y-2">
                                    {envVars.map((envVar, index) => (
                                        <div key={index} className="flex items-center gap-2">
                                            <input
                                                type="text"
                                                placeholder="KEY"
                                                value={envVar.key}
                                                onChange={(e) => handleEnvVarChange(index, 'key', e.target.value)}
                                                className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono text-sm"
                                            />
                                            <span className="text-gray-500">=</span>
                                            <input
                                                type="password"
                                                placeholder="value"
                                                value={envVar.value}
                                                onChange={(e) => handleEnvVarChange(index, 'value', e.target.value)}
                                                className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono text-sm"
                                            />
                                            <button
                                                type="button"
                                                onClick={() => handleRemoveEnvVar(index)}
                                                className="text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300"
                                            >
                                                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                                </svg>
                                            </button>
                                        </div>
                                    ))}
                                </div>
                                <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
                                    Values are hidden for security but will be reused from last deployment
                                </p>
                            </>
                        )}
                    </div>

                    {availableTfvars.length > 0 && (
                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                                Terraform Variables Files
                            </label>
                            <div className="space-y-2 max-h-40 overflow-y-auto border border-gray-300 dark:border-gray-600 rounded-lg p-3">
                                {availableTfvars.map((tfvarsFile) => (
                                    <label key={tfvarsFile} className="flex items-center gap-2 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700 p-2 rounded">
                                        <input
                                            type="checkbox"
                                            checked={selectedTfvars.includes(tfvarsFile)}
                                            onChange={() => handleTfvarsToggle(tfvarsFile)}
                                            className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                                        />
                                        <span className="text-sm font-mono text-gray-900 dark:text-white">{tfvarsFile}</span>
                                    </label>
                                ))}
                            </div>
                            {selectedTfvars.length > 0 && (
                                <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
                                    {selectedTfvars.length} file{selectedTfvars.length > 1 ? 's' : ''} selected
                                </p>
                            )}
                        </div>
                    )}

                    <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                            Custom Init Flags
                            <span className="text-xs text-gray-500 dark:text-gray-400 font-normal ml-2">(optional)</span>
                        </label>
                        <input
                            type="text"
                            value={initFlags}
                            onChange={(e) => setInitFlags(e.target.value)}
                            placeholder="e.g., -upgrade -reconfigure"
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono text-sm"
                        />
                        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                            Additional flags for terraform/tofu init command
                        </p>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                            Custom Plan Flags
                            <span className="text-xs text-gray-500 dark:text-gray-400 font-normal ml-2">(optional)</span>
                        </label>
                        <input
                            type="text"
                            value={planFlags}
                            onChange={(e) => setPlanFlags(e.target.value)}
                            placeholder="e.g., -lock=false -parallelism=10"
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono text-sm"
                        />
                        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                            Additional flags for terraform/tofu plan command
                        </p>
                    </div>

                    <div className="flex items-center justify-end gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">"
                        <button
                            type="button"
                            onClick={onClose}
                            className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
                        >
                            Cancel
                        </button>
                        <button
                            type="submit"
                            disabled={isSubmitting}
                            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors flex items-center gap-2"
                        >
                            {isSubmitting ? (
                                <>
                                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
                                    Starting...
                                </>
                            ) : (
                                <>
                                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                                    </svg>
                                    Start Deployment
                                </>
                            )}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
}

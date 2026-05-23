import { useState, useEffect } from 'react';
import { ChevronRight, ChevronDown, Download, RefreshCw, Database } from 'lucide-react';
import { tfStateApi } from '../api';
import type { TFStateSummary } from '../types';
import { useAuth } from '../context/AuthContext';

// ─── JSON tree viewer ──────────────────────────────────────────────────────────

type JsonValue = string | number | boolean | null | JsonValue[] | { [key: string]: JsonValue };

function JsonNode({ label, value, depth = 0 }: { label: string; value: JsonValue; depth?: number }) {
  const [open, setOpen] = useState(depth < 2);

  const isObject = value !== null && typeof value === 'object';
  const isArray = Array.isArray(value);
  const childCount = isObject ? Object.keys(value as object).length : 0;
  const indent = depth * 16;

  if (!isObject) {
    let display: string;
    let colorClass: string;
    if (value === null) { display = 'null'; colorClass = 'text-gray-400 dark:text-gray-500'; }
    else if (typeof value === 'boolean') { display = String(value); colorClass = 'text-purple-600 dark:text-purple-400'; }
    else if (typeof value === 'number') { display = String(value); colorClass = 'text-blue-600 dark:text-blue-400'; }
    else { display = `"${value}"`; colorClass = 'text-green-700 dark:text-green-400'; }

    return (
      <div className="flex items-baseline gap-1 py-0.5 hover:bg-gray-50 dark:hover:bg-gray-800/50 rounded px-1" style={{ paddingLeft: indent + 4 }}>
        <span className="text-gray-600 dark:text-gray-400 text-xs font-mono shrink-0">{label}:</span>
        <span className={`text-xs font-mono break-all ${colorClass}`}>{display}</span>
      </div>
    );
  }

  const entries = isArray
    ? (value as JsonValue[]).map((v, i) => [String(i), v] as [string, JsonValue])
    : Object.entries(value as { [key: string]: JsonValue });

  return (
    <div>
      <button
        onClick={() => setOpen(o => !o)}
        className="flex items-center gap-1 py-0.5 hover:bg-gray-50 dark:hover:bg-gray-800/50 rounded px-1 w-full text-left"
        style={{ paddingLeft: indent + 4 }}
      >
        {open
          ? <ChevronDown className="h-3 w-3 text-gray-400 shrink-0" />
          : <ChevronRight className="h-3 w-3 text-gray-400 shrink-0" />}
        <span className="text-gray-700 dark:text-gray-300 text-xs font-mono font-medium">{label}</span>
        <span className="text-gray-400 dark:text-gray-500 text-xs font-mono ml-1">
          {isArray ? `[${childCount}]` : `{${childCount}}`}
        </span>
      </button>
      {open && (
        <div>
          {entries.map(([k, v]) => (
            <JsonNode key={k} label={k} value={v} depth={depth + 1} />
          ))}
        </div>
      )}
    </div>
  );
}

function JsonTree({ data }: { data: JsonValue }) {
  if (data === null || typeof data !== 'object') {
    return <span className="text-xs font-mono text-gray-700 dark:text-gray-300">{JSON.stringify(data)}</span>;
  }
  const entries = Array.isArray(data)
    ? (data as JsonValue[]).map((v, i) => [String(i), v] as [string, JsonValue])
    : Object.entries(data as { [key: string]: JsonValue });
  return (
    <div className="py-1">
      {entries.map(([k, v]) => (
        <JsonNode key={k} label={k} value={v} depth={0} />
      ))}
    </div>
  );
}

// ─── Main page ─────────────────────────────────────────────────────────────────

export default function TFStateBrowserPage() {
  const { isAdmin } = useAuth();
  const [deploymentIds, setDeploymentIds] = useState<string[]>([]);
  const [selectedDeploymentId, setSelectedDeploymentId] = useState<string | null>(null);
  const [workspaces, setWorkspaces] = useState<TFStateSummary[]>([]);
  const [selectedWorkspace, setSelectedWorkspace] = useState<TFStateSummary | null>(null);
  const [rawJson, setRawJson] = useState<JsonValue | null>(null);
  const [rawString, setRawString] = useState<string>('');

  const [loadingDeployments, setLoadingDeployments] = useState(true);
  const [loadingWorkspaces, setLoadingWorkspaces] = useState(false);
  const [loadingState, setLoadingState] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [notification, setNotification] = useState<{ message: string; type: 'success' | 'error' } | null>(null);

  const showNotification = (message: string, type: 'success' | 'error' = 'success') => {
    setNotification({ message, type });
    setTimeout(() => setNotification(null), 4000);
  };

  // Load deployment IDs from disk on mount
  useEffect(() => {
    tfStateApi.listDeployments()
      .then(data => setDeploymentIds(Array.isArray(data) ? data : []))
      .catch(() => setError('Failed to load state deployments'))
      .finally(() => setLoadingDeployments(false));
  }, []);

  // Load workspaces when deployment changes
  useEffect(() => {
    if (!selectedDeploymentId) { setWorkspaces([]); return; }
    setLoadingWorkspaces(true);
    setSelectedWorkspace(null);
    setRawJson(null);
    tfStateApi.getWorkspaces(selectedDeploymentId)
      .then(data => setWorkspaces(Array.isArray(data) ? data : []))
      .catch(() => setError('Failed to load workspaces'))
      .finally(() => setLoadingWorkspaces(false));
  }, [selectedDeploymentId]);

  // Load raw state when workspace changes
  useEffect(() => {
    if (!selectedDeploymentId || !selectedWorkspace) { setRawJson(null); return; }
    setLoadingState(true);
    setError(null);
    tfStateApi.getRaw(selectedDeploymentId, selectedWorkspace.workspace)
      .then(data => {
        setRawJson(data as JsonValue);
        setRawString(JSON.stringify(data, null, 2));
      })
      .catch(() => setError('Failed to load state'))
      .finally(() => setLoadingState(false));
  }, [selectedDeploymentId, selectedWorkspace]);

  const handleDownload = () => {
    if (!rawString || !selectedWorkspace) return;
    const blob = new Blob([rawString], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `terraform.tfstate.${selectedWorkspace.workspace}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const refreshWorkspaces = () => {
    if (!selectedDeploymentId) return;
    setLoadingWorkspaces(true);
    tfStateApi.getWorkspaces(selectedDeploymentId)
      .then(data => setWorkspaces(Array.isArray(data) ? data : []))
      .catch(() => setError('Failed to refresh workspaces'))
      .finally(() => setLoadingWorkspaces(false));
  };

  const handleForceUnlock = async (ws: TFStateSummary) => {
    if (!selectedDeploymentId) return;
    if (!confirm(`Force-unlock workspace "${ws.workspace}"? Only do this if no Terraform process is running.`)) return;
    setActionLoading(`unlock-${ws.id}`);
    try {
      await tfStateApi.forceUnlock(selectedDeploymentId, ws.workspace);
      showNotification(`Workspace "${ws.workspace}" unlocked`, 'success');
      if (selectedWorkspace?.id === ws.id) setSelectedWorkspace({ ...ws, lock_id: undefined, locked_at: undefined });
      refreshWorkspaces();
    } catch {
      showNotification('Failed to unlock', 'error');
    } finally {
      setActionLoading(null);
    }
  };

  const handleDeleteWorkspace = async (ws: TFStateSummary) => {
    if (!selectedDeploymentId) return;
    if (!confirm(`Delete all state for workspace "${ws.workspace}"? This cannot be undone.`)) return;
    setActionLoading(`delete-${ws.id}`);
    try {
      await tfStateApi.deleteWorkspace(selectedDeploymentId, ws.workspace);
      showNotification(`Workspace "${ws.workspace}" deleted`, 'success');
      if (selectedWorkspace?.id === ws.id) { setSelectedWorkspace(null); setRawJson(null); }
      refreshWorkspaces();
    } catch {
      showNotification('Failed to delete workspace', 'error');
    } finally {
      setActionLoading(null);
    }
  };

  return (
    <div className="space-y-6">
      {notification && (
        <div className={`fixed top-4 right-4 z-50 px-6 py-4 rounded-lg shadow-lg border transition-all duration-300 ${
          notification.type === 'success'
            ? 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800 text-green-800 dark:text-green-200'
            : 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800 text-red-800 dark:text-red-200'
        }`}>
          <div className="flex items-center gap-3">
            <p className="font-medium text-sm">{notification.message}</p>
            <button onClick={() => setNotification(null)} className="ml-2 text-gray-500 hover:text-gray-700">
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
              </svg>
            </button>
          </div>
        </div>
      )}
      {/* Header */}
      <div className="flex items-center gap-3">
        <Database className="h-7 w-7 text-blue-500" />
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">TF State Browser</h1>
          <p className="text-sm text-gray-500 dark:text-gray-400">Browse and inspect state files across deployments</p>
        </div>
      </div>

      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 text-red-700 dark:text-red-400 px-4 py-3 rounded-lg text-sm">
          {error}
        </div>
      )}

      {/* Three-column layout */}
      <div className="flex gap-4 h-[calc(100vh-220px)] min-h-[500px]">

        {/* Column 1 — Deployments */}
        <div className="w-56 shrink-0 bg-white dark:bg-gray-800 rounded-lg shadow flex flex-col">
          <div className="px-4 py-3 border-b border-gray-200 dark:border-gray-700">
            <h2 className="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Deployments</h2>
          </div>
          <div className="flex-1 overflow-y-auto">
            {loadingDeployments ? (
              <div className="flex justify-center py-8">
                <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-blue-600" />
              </div>
            ) : deploymentIds.length === 0 ? (
              <p className="px-4 py-6 text-xs text-gray-400 dark:text-gray-500 text-center">No state files found</p>
            ) : (
              deploymentIds.map(id => (
                <button
                  key={id}
                  onClick={() => setSelectedDeploymentId(id)}
                  className={`w-full text-left px-4 py-3 text-sm transition-colors border-l-2 ${
                    selectedDeploymentId === id
                      ? 'bg-blue-50 dark:bg-blue-900/20 border-blue-500 text-blue-700 dark:text-blue-300 font-medium'
                      : 'border-transparent text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700'
                  }`}
                >
                  <div className="truncate font-mono text-xs">{id}</div>
                </button>
              ))
            )}
          </div>
        </div>

        {/* Column 2 — Workspaces */}
        <div className="w-48 shrink-0 bg-white dark:bg-gray-800 rounded-lg shadow flex flex-col">
          <div className="px-4 py-3 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
            <h2 className="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Workspaces</h2>
            {selectedDeploymentId && (
              <button onClick={refreshWorkspaces} title="Refresh" className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-200">
                <RefreshCw className={`h-3.5 w-3.5 ${loadingWorkspaces ? 'animate-spin' : ''}`} />
              </button>
            )}
          </div>
          <div className="flex-1 overflow-y-auto">
            {!selectedDeploymentId ? (
              <p className="px-4 py-6 text-xs text-gray-400 dark:text-gray-500 text-center">Select a deployment</p>
            ) : loadingWorkspaces ? (
              <div className="flex justify-center py-8">
                <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-blue-600" />
              </div>
            ) : workspaces.length === 0 ? (
              <p className="px-4 py-6 text-xs text-gray-400 dark:text-gray-500 text-center">No state files yet</p>
            ) : (
              workspaces.map(ws => (
                <div key={ws.id} className={`border-l-2 transition-colors ${
                  selectedWorkspace?.id === ws.id
                    ? 'bg-blue-50 dark:bg-blue-900/20 border-blue-500'
                    : 'border-transparent hover:bg-gray-50 dark:hover:bg-gray-700'
                }`}>
                  <button
                    onClick={() => setSelectedWorkspace(ws)}
                    className="w-full text-left px-4 py-3 text-sm"
                  >
                    <div className={`truncate font-mono text-xs font-semibold ${selectedWorkspace?.id === ws.id ? 'text-blue-700 dark:text-blue-300' : 'text-gray-700 dark:text-gray-300'}`}>{ws.workspace}</div>
                    <div className="flex items-center gap-1 mt-0.5">
                      {ws.lock_id ? (
                        <span className="text-xs text-yellow-600 dark:text-yellow-400">locked</span>
                      ) : (
                        <span className="text-xs text-green-600 dark:text-green-400">unlocked</span>
                      )}
                      <span className="text-xs text-gray-400 dark:text-gray-500">· serial {ws.state_serial}</span>
                    </div>
                  </button>
                  {isAdmin && (
                    <div className="px-3 pb-2 flex gap-1">
                      {ws.lock_id && (
                        <button
                          onClick={() => handleForceUnlock(ws)}
                          disabled={actionLoading === `unlock-${ws.id}`}
                          className="flex-1 text-center py-1 text-xs bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-300 rounded hover:bg-yellow-200 dark:hover:bg-yellow-900/50 disabled:opacity-50 transition-colors"
                        >
                          {actionLoading === `unlock-${ws.id}` ? '...' : 'Unlock'}
                        </button>
                      )}
                      <button
                        onClick={() => handleDeleteWorkspace(ws)}
                        disabled={actionLoading === `delete-${ws.id}`}
                        className="flex-1 text-center py-1 text-xs bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 rounded hover:bg-red-200 dark:hover:bg-red-900/50 disabled:opacity-50 transition-colors"
                      >
                        {actionLoading === `delete-${ws.id}` ? '...' : 'Delete'}
                      </button>
                    </div>
                  )}
                </div>
              ))
            )}
          </div>
        </div>

        {/* Column 3 — State viewer */}
        <div className="flex-1 min-w-0 bg-white dark:bg-gray-800 rounded-lg shadow flex flex-col">
          <div className="px-4 py-3 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between shrink-0">
            <div className="flex items-center gap-2 min-w-0">
              <h2 className="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider shrink-0">State</h2>
              {selectedWorkspace && (
                <span className="text-xs font-mono text-blue-600 dark:text-blue-400 truncate">
                  {selectedDeploymentId} / {selectedWorkspace.workspace}
                </span>
              )}
              {selectedWorkspace && (
                selectedWorkspace.lock_id ? (
                  <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-300 shrink-0">
                    <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M5 9V7a5 5 0 0110 0v2a2 2 0 012 2v5a2 2 0 01-2 2H5a2 2 0 01-2-2v-5a2 2 0 012-2zm8-2v2H7V7a3 3 0 016 0z" clipRule="evenodd" />
                    </svg>
                    Locked
                  </span>
                ) : (
                  <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 shrink-0">
                    <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                      <path d="M10 2a5 5 0 00-5 5v2a2 2 0 00-2 2v5a2 2 0 002 2h10a2 2 0 002-2v-5a2 2 0 00-2-2H7V7a3 3 0 015.905-.75 1 1 0 001.937-.5A5.002 5.002 0 0010 2z" />
                    </svg>
                    Unlocked
                  </span>
                )
              )}
            </div>
            {rawJson && (
              <button
                onClick={handleDownload}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-md hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors shrink-0"
              >
                <Download className="h-3.5 w-3.5" />
                Download
              </button>
            )}
          </div>

          <div className="flex-1 overflow-auto p-2">
            {!selectedWorkspace ? (
              <div className="flex flex-col items-center justify-center h-full text-gray-400 dark:text-gray-500 gap-2">
                <Database className="h-10 w-10 opacity-30" />
                <p className="text-sm">Select a workspace to inspect its state</p>
              </div>
            ) : loadingState ? (
              <div className="flex justify-center py-12">
                <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600" />
              </div>
            ) : rawJson ? (
              <JsonTree data={rawJson} />
            ) : null}
          </div>
        </div>
      </div>
    </div>
  );
}

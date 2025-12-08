import { Settings, Terminal } from 'lucide-react';

export default function SettingsPage() {
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
              {window.location.origin}/.well-known/terraform.json
            </dd>
          </div>
          <div>
            <dt className="text-sm text-gray-500 dark:text-gray-400">Modules API</dt>
            <dd className="text-sm font-mono text-gray-900 dark:text-white">
              {window.location.origin}/v1/modules/
            </dd>
          </div>
          <div>
            <dt className="text-sm text-gray-500 dark:text-gray-400">Providers API</dt>
            <dd className="text-sm font-mono text-gray-900 dark:text-white">
              {window.location.origin}/v1/providers/
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
          To use this registry with Terraform, you need to configure authentication. 
          First, create a namespace and generate an API key from the Namespaces page, 
          then add this to your ~/.terraformrc:
        </p>
        <pre className="text-xs text-blue-700 dark:text-blue-300 bg-blue-100 dark:bg-blue-900/40 p-3 rounded">
{`credentials "${window.location.hostname}${window.location.port ? ':' + window.location.port : ''}" {
  token = "YOUR_API_KEY"
}`}
        </pre>
      </div>

      <div className="bg-amber-50 dark:bg-amber-900/20 rounded-lg p-4">
        <h3 className="text-sm font-medium text-amber-800 dark:text-amber-200 mb-2">
          About API Keys
        </h3>
        <p className="text-xs text-amber-700 dark:text-amber-300">
          API keys are only required for Terraform CLI access to download modules and providers.
          The web interface does not require authentication - you can create and manage 
          namespaces, modules, and providers directly from this UI.
        </p>
      </div>
    </div>
  );
}

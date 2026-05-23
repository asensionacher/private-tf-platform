import { useState } from 'react';
import { Check, Copy, Database } from 'lucide-react';

type Tool = 'terraform' | 'tofu';

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);
  const handleCopy = async () => {
    await navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };
  return (
    <button
      onClick={handleCopy}
      className="absolute top-3 right-3 p-2 hover:bg-gray-200 dark:hover:bg-gray-700 rounded opacity-0 group-hover:opacity-100 transition-opacity"
      title="Copy to clipboard"
    >
      {copied ? <Check className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4 text-gray-500" />}
    </button>
  );
}

function CodeBlock({ code }: { code: string }) {
  return (
    <div className="bg-gray-50 dark:bg-gray-900 rounded-lg p-4 relative group">
      <CopyButton text={code} />
      <pre className="text-xs text-gray-800 dark:text-gray-200 overflow-x-auto font-mono whitespace-pre">{code}</pre>
    </div>
  );
}

export default function BackendPage() {
  const [tool, setTool] = useState<Tool>('terraform');

  const cmd = tool === 'terraform' ? 'terraform' : 'tofu';
  const rcFile = tool === 'terraform' ? '~/.terraformrc' : '~/.tofurc';
  const rcFileWin = tool === 'terraform' ? '%APPDATA%\\terraform.rc' : '%APPDATA%\\tofu.rc';
  const registryUrl = '<REGISTRY_URL>';   // placeholder replaced by user
  const deploymentId = '<DEPLOYMENT_ID>';

  const backendBlock = `terraform {
  backend "http" {
    address        = "${registryUrl}/api/tfstate/${deploymentId}?workspace=default"
    lock_address   = "${registryUrl}/api/tfstate/${deploymentId}?workspace=default"
    unlock_address = "${registryUrl}/api/tfstate/${deploymentId}?workspace=default"
    lock_method    = "LOCK"
    unlock_method  = "UNLOCK"
  }
}`;

  const rcBlock = `host "${registryUrl.replace(/^https?:\/\//, '')}" {
  services = {
    "providers.v1" = "${registryUrl}/v1/providers/"
    "modules.v1"   = "${registryUrl}/v1/modules/"
  }
}

credentials "${registryUrl.replace(/^https?:\/\//, '')}" {
  token = "<YOUR_API_KEY>"
}`;

  const pullCmd = `# Pull state to a local file
${cmd} state pull > terraform.tfstate

# Inspect or edit terraform.tfstate with any text editor or jq
jq '.resources[] | {type, name}' terraform.tfstate`;

  const pushCmd = `# Push the (modified) state back
${cmd} state push terraform.tfstate`;

  const rmCmd = `# Remove a resource from state without destroying it
${cmd} state rm <resource_address>

# Example
${cmd} state rm aws_instance.web`;

  const mvCmd = `# Rename / move a resource in state
${cmd} state mv <source> <destination>

# Example
${cmd} state mv aws_instance.old_name aws_instance.new_name`;

  const importCmd = `# Import an existing real resource into state
${cmd} import <resource_address> <real_resource_id>

# Example
${cmd} import aws_instance.web i-1234567890abcdef0`;

  const workspaceCmd = `# List workspaces
${cmd} workspace list

# Create and switch to a new workspace
${cmd} workspace new staging

# Switch back to default
${cmd} workspace select default`;

  const backendWorkspaceBlock = `terraform {
  backend "http" {
    # The workspace name is passed via the query parameter.
    # When using \`${cmd} workspace\` commands the client appends it automatically.
    address        = "${registryUrl}/api/tfstate/${deploymentId}"
    lock_address   = "${registryUrl}/api/tfstate/${deploymentId}"
    unlock_address = "${registryUrl}/api/tfstate/${deploymentId}"
    lock_method    = "LOCK"
    unlock_method  = "UNLOCK"
  }
}`;

  return (
    <div className="space-y-8 max-w-4xl">
      {/* Header */}
      <div>
        <div className="flex items-center gap-3 mb-1">
          <Database className="h-7 w-7 text-blue-500" />
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">HTTP State Backend</h1>
        </div>
        <p className="text-sm text-gray-500 dark:text-gray-400">
         This platform includes a built-in HTTP backend compatible with both Terraform and OpenTofu.
           State is stored by workspace, with each workspace getting its own state file.
        </p>
      </div>

      {/* Tool selector */}
      <div className="flex gap-3">
        <button
          onClick={() => setTool('terraform')}
          className={`px-5 py-2.5 rounded-lg border-2 text-sm font-semibold transition-colors ${
            tool === 'terraform'
              ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/20 text-purple-700 dark:text-purple-300'
              : 'border-gray-200 dark:border-gray-700 text-gray-500 dark:text-gray-400 hover:border-gray-300 dark:hover:border-gray-600'
          }`}
        >
          Terraform
        </button>
        <button
          onClick={() => setTool('tofu')}
          className={`px-5 py-2.5 rounded-lg border-2 text-sm font-semibold transition-colors ${
            tool === 'tofu'
              ? 'border-yellow-500 bg-yellow-50 dark:bg-yellow-900/20 text-yellow-700 dark:text-yellow-300'
              : 'border-gray-200 dark:border-gray-700 text-gray-500 dark:text-gray-400 hover:border-gray-300 dark:hover:border-gray-600'
          }`}
        >
          OpenTofu
        </button>
      </div>

      {/* Step 1 — CLI config */}
      <section className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 space-y-4">
        <div className="flex items-center gap-2">
          <span className="flex items-center justify-center w-7 h-7 rounded-full bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 text-sm font-bold">1</span>
          <h2 className="text-base font-semibold text-gray-900 dark:text-white">Configure the CLI</h2>
        </div>
        <p className="text-sm text-gray-600 dark:text-gray-400">
          Create or edit{' '}
          <code className="px-1.5 py-0.5 bg-gray-100 dark:bg-gray-700 rounded text-xs font-mono">{rcFile}</code>{' '}
          (Linux/macOS) or{' '}
          <code className="px-1.5 py-0.5 bg-gray-100 dark:bg-gray-700 rounded text-xs font-mono">{rcFileWin}</code>{' '}
          (Windows) to point the CLI at this registry:
        </p>
        <CodeBlock code={rcBlock} />
        <p className="text-xs text-gray-500 dark:text-gray-400">
          Replace <code className="font-mono">&lt;REGISTRY_URL&gt;</code> with your actual registry URL (e.g.{' '}
          <code className="font-mono">http://registry.lan:9080</code>) and{' '}
          <code className="font-mono">&lt;YOUR_API_KEY&gt;</code> with a key from the{' '}
          <a href="/api-keys" className="text-blue-600 dark:text-blue-400 underline">API Keys</a> page.
          The credentials block is only required for private namespaces.
        </p>
      </section>

      {/* Step 2 — Backend block */}
      <section className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 space-y-4">
        <div className="flex items-center gap-2">
          <span className="flex items-center justify-center w-7 h-7 rounded-full bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 text-sm font-bold">2</span>
          <h2 className="text-base font-semibold text-gray-900 dark:text-white">Add the backend block to your configuration</h2>
        </div>
        <p className="text-sm text-gray-600 dark:text-gray-400">
          Add this to your <code className="px-1.5 py-0.5 bg-gray-100 dark:bg-gray-700 rounded text-xs font-mono">main.tf</code> or a dedicated{' '}
          <code className="px-1.5 py-0.5 bg-gray-100 dark:bg-gray-700 rounded text-xs font-mono">backend.tf</code>:
        </p>
        <CodeBlock code={backendBlock} />
        <p className="text-xs text-gray-500 dark:text-gray-400">
          Replace <code className="font-mono">&lt;DEPLOYMENT_ID&gt;</code> with a unique identifier for your state file group.
          This ID is used as the directory name for storing state files on the server.
        </p>
      </section>

      {/* Step 3 — Init */}
      <section className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 space-y-4">
        <div className="flex items-center gap-2">
          <span className="flex items-center justify-center w-7 h-7 rounded-full bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 text-sm font-bold">3</span>
          <h2 className="text-base font-semibold text-gray-900 dark:text-white">Initialize and use</h2>
        </div>
        <div className="space-y-2">
          <CodeBlock code={`${cmd} init`} />
          <CodeBlock code={`${cmd} plan`} />
          <CodeBlock code={`${cmd} apply`} />
        </div>
      </section>

      {/* Workspaces */}
      <section className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 space-y-4">
        <h2 className="text-base font-semibold text-gray-900 dark:text-white">Workspaces</h2>
        <p className="text-sm text-gray-600 dark:text-gray-400">
          The backend supports workspaces via the <code className="px-1.5 py-0.5 bg-gray-100 dark:bg-gray-700 rounded text-xs font-mono">workspace</code> query parameter.
          When using <code className="px-1.5 py-0.5 bg-gray-100 dark:bg-gray-700 rounded text-xs font-mono">{cmd} workspace</code> commands the client appends it automatically —
          omit the <code className="font-mono">?workspace=default</code> suffix from the address so the CLI can manage it:
        </p>
        <CodeBlock code={backendWorkspaceBlock} />
        <CodeBlock code={workspaceCmd} />
        <p className="text-xs text-gray-500 dark:text-gray-400">
          Each workspace gets its own state file. You can view and manage all workspaces from the{' '}
          <a href="/tfstate" className="text-blue-600 dark:text-blue-400 underline">TF State Browser</a> page.
        </p>
      </section>

      {/* Download / modify state */}
      <section className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 space-y-6">
        <h2 className="text-base font-semibold text-gray-900 dark:text-white">Download and modify state</h2>

        <div className="space-y-2">
          <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">Pull state locally</h3>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            Use <code className="font-mono">{cmd} state pull</code> to download the current state to a local file for inspection or editing.
          </p>
          <CodeBlock code={pullCmd} />
        </div>

        <div className="space-y-2">
          <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">Push modified state back</h3>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            After editing <code className="font-mono">terraform.tfstate</code> locally, push it back. The serial number must be incremented or the push will be rejected.
          </p>
          <CodeBlock code={pushCmd} />
          <div className="bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg p-3 text-xs text-amber-700 dark:text-amber-300">
            <strong>Warning:</strong> Manually editing state can corrupt it. Always take a backup first (<code className="font-mono">{cmd} state pull &gt; backup.tfstate</code>) and only modify state when you understand the consequences.
          </div>
        </div>

        <div className="space-y-2">
          <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">Remove a resource from state</h3>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            Removes a resource from the state file without destroying the real infrastructure.
            Useful when you want to stop managing a resource with {tool === 'terraform' ? 'Terraform' : 'OpenTofu'}.
          </p>
          <CodeBlock code={rmCmd} />
        </div>

        <div className="space-y-2">
          <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">Rename / move a resource in state</h3>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            Use <code className="font-mono">{cmd} state mv</code> when you refactor your configuration and rename a resource without destroying it.
          </p>
          <CodeBlock code={mvCmd} />
        </div>

        <div className="space-y-2">
          <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">Import existing infrastructure</h3>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            Bring an existing real resource under management by importing it into state.
          </p>
          <CodeBlock code={importCmd} />
        </div>
      </section>

      {/* UI download note */}
      <section className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4 text-sm text-blue-700 dark:text-blue-300">
        <strong>Tip:</strong> You can also download, inspect, force-unlock, and delete state files directly from the{' '}
        <a href="/tfstate" className="text-blue-600 dark:text-blue-400 underline">TF State Browser</a>.
      </section>
    </div>
  );
}

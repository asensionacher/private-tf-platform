package database

import (
	"database/sql"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init() error {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./registry.db"
	}

	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}

	if err = DB.Ping(); err != nil {
		return err
	}

	if err = createTables(); err != nil {
		return err
	}

	log.Println("Database initialized successfully at", dbPath)
	return nil
}

func createTables() error {
	// Namespaces table (authorities/organizations)
	namespacesTable := `
	CREATE TABLE IF NOT EXISTS namespaces (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		description TEXT,
		is_public BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// API Keys table for authentication (global, not tied to namespaces)
	apiKeysTable := `
	CREATE TABLE IF NOT EXISTS api_keys (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		key_hash TEXT NOT NULL UNIQUE,
		key_encrypted TEXT,
		permissions TEXT NOT NULL CHECK(permissions IN ('read', 'write', 'admin')),
		expires_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_used_at DATETIME
	);`

	// Modules table
	modulesTable := `
	CREATE TABLE IF NOT EXISTS modules (
		id TEXT PRIMARY KEY,
		namespace_id TEXT NOT NULL,
		name TEXT NOT NULL,
		provider TEXT NOT NULL,
		description TEXT,
		source_url TEXT,
		git_url TEXT,
		git_ref TEXT,
		git_auth_type TEXT,
		git_auth_data TEXT,
		synced BOOLEAN DEFAULT FALSE,
		sync_error TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (namespace_id) REFERENCES namespaces(id) ON DELETE CASCADE,
		UNIQUE(namespace_id, name, provider)
	);`

	// Module Versions table
	moduleVersionsTable := `
	CREATE TABLE IF NOT EXISTS module_versions (
		id TEXT PRIMARY KEY,
		module_id TEXT NOT NULL,
		version TEXT NOT NULL,
		download_url TEXT NOT NULL,
		documentation TEXT,
		enabled BOOLEAN DEFAULT TRUE,
		tag_date DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (module_id) REFERENCES modules(id) ON DELETE CASCADE,
		UNIQUE(module_id, version)
	);`

	// Providers table
	providersTable := `
	CREATE TABLE IF NOT EXISTS providers (
		id TEXT PRIMARY KEY,
		namespace_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		source_url TEXT,
		synced BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (namespace_id) REFERENCES namespaces(id) ON DELETE CASCADE,
		UNIQUE(namespace_id, name)
	);`

	// Provider Versions table
	providerVersionsTable := `
	CREATE TABLE IF NOT EXISTS provider_versions (
		id TEXT PRIMARY KEY,
		provider_id TEXT NOT NULL,
		version TEXT NOT NULL,
		protocols TEXT,
		enabled BOOLEAN DEFAULT TRUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE,
		UNIQUE(provider_id, version)
	);`

	// Provider Platforms table (binaries per OS/arch)
	providerPlatformsTable := `
	CREATE TABLE IF NOT EXISTS provider_platforms (
		id TEXT PRIMARY KEY,
		version_id TEXT NOT NULL,
		os TEXT NOT NULL,
		arch TEXT NOT NULL,
		filename TEXT NOT NULL,
		download_url TEXT NOT NULL,
		shasums_url TEXT,
		shasums_signature_url TEXT,
		shasum TEXT NOT NULL,
		signing_keys TEXT,
		FOREIGN KEY (version_id) REFERENCES provider_versions(id) ON DELETE CASCADE,
		UNIQUE(version_id, os, arch)
	);`

	// Deployments table
	deploymentsTable := `
	CREATE TABLE IF NOT EXISTS deployments (
		id TEXT PRIMARY KEY,
		namespace_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		terraform_version TEXT,
		backend_config TEXT,
		git_url TEXT NOT NULL,
		git_ref TEXT NOT NULL DEFAULT 'main',
		git_auth_type TEXT,
		git_auth_data TEXT,
		working_directory TEXT DEFAULT '.',
		terraform_vars TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (namespace_id) REFERENCES namespaces(id) ON DELETE CASCADE,
		UNIQUE(namespace_id, name)
	);`

	// Deployment Runs table
	deploymentRunsTable := `
	CREATE TABLE IF NOT EXISTS deployment_runs (
		id TEXT PRIMARY KEY,
		deployment_id TEXT NOT NULL,
		path TEXT,
		ref TEXT,
		tool TEXT,
		env_vars TEXT,
		tfvars_files TEXT,
		status TEXT NOT NULL CHECK(status IN ('pending', 'initializing', 'planning', 'planned', 'awaiting_approval', 'applying', 'applied', 'destroying', 'destroyed', 'success', 'failed', 'cancelled')) DEFAULT 'pending',
		init_log TEXT,
		plan_log TEXT,
		plan_output TEXT,
		plan_file_path TEXT,
		apply_log TEXT,
		apply_output TEXT,
		error_message TEXT,
		work_dir TEXT,
		approved_by TEXT,
		approved_at DATETIME,
		started_at DATETIME,
		completed_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (deployment_id) REFERENCES deployments(id) ON DELETE CASCADE
	);`

	tables := []string{
		namespacesTable,
		apiKeysTable,
		modulesTable,
		moduleVersionsTable,
		providersTable,
		providerVersionsTable,
		providerPlatformsTable,
		deploymentsTable,
		deploymentRunsTable,
	}

	for _, table := range tables {
		if _, err := DB.Exec(table); err != nil {
			return err
		}
	}

	// Create default namespace if not exists
	_, err := DB.Exec(`
		INSERT OR IGNORE INTO namespaces (id, name, description, is_public)
		VALUES ('default', 'default', 'Default namespace', true)
	`)
	if err != nil {
		return err
	}

	return nil
}

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

	// API Keys table for authentication
	apiKeysTable := `
	CREATE TABLE IF NOT EXISTS api_keys (
		id TEXT PRIMARY KEY,
		namespace_id TEXT NOT NULL,
		name TEXT NOT NULL,
		key_hash TEXT NOT NULL UNIQUE,
		permissions TEXT NOT NULL CHECK(permissions IN ('read', 'write', 'admin')),
		expires_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_used_at DATETIME,
		FOREIGN KEY (namespace_id) REFERENCES namespaces(id) ON DELETE CASCADE
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
		synced BOOLEAN DEFAULT FALSE,
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

	tables := []string{
		namespacesTable,
		apiKeysTable,
		modulesTable,
		moduleVersionsTable,
		providersTable,
		providerVersionsTable,
		providerPlatformsTable,
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

	// Run migrations for existing tables
	if err := runMigrations(); err != nil {
		return err
	}

	return nil
}

// runMigrations adds new columns to existing tables
func runMigrations() error {
	// Add enabled column to module_versions if not exists
	DB.Exec(`ALTER TABLE module_versions ADD COLUMN enabled BOOLEAN DEFAULT TRUE`)

	// Add enabled column to provider_versions if not exists
	DB.Exec(`ALTER TABLE provider_versions ADD COLUMN enabled BOOLEAN DEFAULT TRUE`)

	// Add tag_date column to module_versions if not exists
	DB.Exec(`ALTER TABLE module_versions ADD COLUMN tag_date DATETIME`)

	// Add tag_date column to provider_versions if not exists
	DB.Exec(`ALTER TABLE provider_versions ADD COLUMN tag_date DATETIME`)

	// Add synced column to providers if not exists
	DB.Exec(`ALTER TABLE providers ADD COLUMN synced BOOLEAN DEFAULT FALSE`)

	// Add synced column to modules if not exists
	DB.Exec(`ALTER TABLE modules ADD COLUMN synced BOOLEAN DEFAULT FALSE`)

	// Add sync_error column to modules if not exists
	DB.Exec(`ALTER TABLE modules ADD COLUMN sync_error TEXT`)

	// Add sync_error column to providers if not exists
	DB.Exec(`ALTER TABLE providers ADD COLUMN sync_error TEXT`)

	// Add git authentication columns to modules
	DB.Exec(`ALTER TABLE modules ADD COLUMN git_auth_type TEXT`)
	DB.Exec(`ALTER TABLE modules ADD COLUMN git_auth_data TEXT`) // Encrypted JSON with credentials

	// Add git authentication columns to providers
	DB.Exec(`ALTER TABLE providers ADD COLUMN git_auth_type TEXT`)
	DB.Exec(`ALTER TABLE providers ADD COLUMN git_auth_data TEXT`) // Encrypted JSON with credentials

	// Create deployments table if not exists
	DB.Exec(`CREATE TABLE IF NOT EXISTS deployments (
		id TEXT PRIMARY KEY,
		namespace_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		git_url TEXT NOT NULL,
		git_auth_type TEXT,
		git_auth_data TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (namespace_id) REFERENCES namespaces(id) ON DELETE CASCADE,
		UNIQUE(namespace_id, name)
	)`)

	// Create deployment_runs table if not exists
	DB.Exec(`CREATE TABLE IF NOT EXISTS deployment_runs (
		id TEXT PRIMARY KEY,
		deployment_id TEXT NOT NULL,
		path TEXT NOT NULL,
		ref TEXT NOT NULL,
		tool TEXT NOT NULL DEFAULT 'terraform',
		env_vars TEXT,
		status TEXT NOT NULL CHECK(status IN ('pending', 'initializing', 'planning', 'awaiting_approval', 'applying', 'success', 'failed', 'cancelled')) DEFAULT 'pending',
		init_log TEXT,
		plan_log TEXT,
		apply_log TEXT,
		error_message TEXT,
		work_dir TEXT,
		approved_by TEXT,
		approved_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		started_at DATETIME,
		completed_at DATETIME,
		FOREIGN KEY (deployment_id) REFERENCES deployments(id) ON DELETE CASCADE
	)`)

	// Create index for faster queries
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_deployment_runs_deployment_path ON deployment_runs(deployment_id, path, created_at DESC)`)

	// Add new columns to deployment_runs if they don't exist
	DB.Exec(`ALTER TABLE deployment_runs ADD COLUMN tool TEXT DEFAULT 'terraform'`)
	DB.Exec(`ALTER TABLE deployment_runs ADD COLUMN env_vars TEXT`)
	DB.Exec(`ALTER TABLE deployment_runs ADD COLUMN init_log TEXT`)
	DB.Exec(`ALTER TABLE deployment_runs ADD COLUMN plan_log TEXT`)
	DB.Exec(`ALTER TABLE deployment_runs ADD COLUMN apply_log TEXT`)
	DB.Exec(`ALTER TABLE deployment_runs ADD COLUMN work_dir TEXT`)
	DB.Exec(`ALTER TABLE deployment_runs ADD COLUMN approved_by TEXT`)
	DB.Exec(`ALTER TABLE deployment_runs ADD COLUMN approved_at DATETIME`)

	return nil
}

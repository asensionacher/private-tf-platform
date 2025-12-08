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

	return nil
}

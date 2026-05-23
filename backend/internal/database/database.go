package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var DB *sql.DB

func Init() error {
	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("POSTGRES_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("POSTGRES_USER")
	if user == "" {
		user = "registry"
	}
	password := os.Getenv("POSTGRES_PASSWORD")
	if password == "" {
		password = "registry"
	}
	dbname := os.Getenv("POSTGRES_DB")
	if dbname == "" {
		dbname = "registry"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return err
	}

	if err = DB.Ping(); err != nil {
		return err
	}

	if err = createTables(); err != nil {
		return err
	}

	log.Printf("Database initialized successfully (PostgreSQL at %s:%s)", host, port)
	return nil
}

func createTables() error {
	// Namespaces table (authorities/organizations)
	namespacesTable := `
	CREATE TABLE IF NOT EXISTS namespaces (
		id VARCHAR(255) PRIMARY KEY,
		name VARCHAR(255) NOT NULL UNIQUE,
		description TEXT,
		is_public BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	// Users table for web UI authentication
	usersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id VARCHAR(255) PRIMARY KEY,
		username VARCHAR(255) NOT NULL UNIQUE,
		password_hash VARCHAR(255) NOT NULL,
		role VARCHAR(50) NOT NULL CHECK(role IN ('admin', 'reader')),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	// API Keys table for authentication (global, not tied to namespaces)
	apiKeysTable := `
	CREATE TABLE IF NOT EXISTS api_keys (
		id VARCHAR(255) PRIMARY KEY,
		user_id VARCHAR(255),
		name VARCHAR(255) NOT NULL,
		key_hash VARCHAR(255) NOT NULL UNIQUE,
		key_encrypted TEXT,
		permissions VARCHAR(50) NOT NULL CHECK(permissions IN ('read', 'write', 'admin')),
		expires_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_used_at TIMESTAMP
	);`

	// Modules table
	modulesTable := `
	CREATE TABLE IF NOT EXISTS modules (
		id VARCHAR(255) PRIMARY KEY,
		namespace_id VARCHAR(255) NOT NULL,
		name VARCHAR(255) NOT NULL,
		provider VARCHAR(255) NOT NULL,
		description TEXT,
		source_url TEXT,
		git_url TEXT,
		git_ref VARCHAR(255),
		git_auth_type VARCHAR(50),
		git_auth_data TEXT,
		synced BOOLEAN DEFAULT FALSE,
		sync_error TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (namespace_id) REFERENCES namespaces(id) ON DELETE CASCADE,
		UNIQUE(namespace_id, name, provider)
	);`

	// Module Versions table
	moduleVersionsTable := `
	CREATE TABLE IF NOT EXISTS module_versions (
		id VARCHAR(255) PRIMARY KEY,
		module_id VARCHAR(255) NOT NULL,
		version VARCHAR(100) NOT NULL,
		download_url TEXT NOT NULL,
		documentation TEXT,
		enabled BOOLEAN DEFAULT TRUE,
		tag_date TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (module_id) REFERENCES modules(id) ON DELETE CASCADE,
		UNIQUE(module_id, version)
	);`

	// Providers table
	providersTable := `
	CREATE TABLE IF NOT EXISTS providers (
		id VARCHAR(255) PRIMARY KEY,
		namespace_id VARCHAR(255) NOT NULL,
		name VARCHAR(255) NOT NULL,
		description TEXT,
		source_url TEXT,
		synced BOOLEAN DEFAULT FALSE,
		sync_error TEXT,
		git_auth_type VARCHAR(50),
		git_auth_data TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (namespace_id) REFERENCES namespaces(id) ON DELETE CASCADE,
		UNIQUE(namespace_id, name)
	);`

	// Provider Versions table
	providerVersionsTable := `
	CREATE TABLE IF NOT EXISTS provider_versions (
		id VARCHAR(255) PRIMARY KEY,
		provider_id VARCHAR(255) NOT NULL,
		version VARCHAR(100) NOT NULL,
		protocols TEXT,
		enabled BOOLEAN DEFAULT TRUE,
		tag_date TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE,
		UNIQUE(provider_id, version)
	);`

	// Provider Platforms table (binaries per OS/arch)
	providerPlatformsTable := `
	CREATE TABLE IF NOT EXISTS provider_platforms (
		id VARCHAR(255) PRIMARY KEY,
		version_id VARCHAR(255) NOT NULL,
		os VARCHAR(50) NOT NULL,
		arch VARCHAR(50) NOT NULL,
		filename VARCHAR(255) NOT NULL,
		download_url TEXT NOT NULL,
		shasums_url TEXT,
		shasums_signature_url TEXT,
		shasum VARCHAR(255) NOT NULL,
		signing_keys TEXT,
		FOREIGN KEY (version_id) REFERENCES provider_versions(id) ON DELETE CASCADE,
		UNIQUE(version_id, os, arch)
	);`

	tables := []string{
		usersTable,
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
		INSERT INTO namespaces (id, name, description, is_public)
		VALUES ('default', 'default', 'Default namespace', true)
		ON CONFLICT (id) DO NOTHING
	`)
	if err != nil {
		return err
	}

	// Seed default admin user (password: admin) — only on first run
	// Generate bcrypt hash at runtime to ensure correctness
	var adminExists bool
	DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = 'admin')").Scan(&adminExists)
	if !adminExists {
		hash, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		_, err = DB.Exec(`
			INSERT INTO users (id, username, password_hash, role)
			VALUES ('admin', 'admin', $1, 'admin')
			ON CONFLICT (id) DO NOTHING
		`, string(hash))
		if err != nil {
			return err
		}
		log.Println("Seeded default admin user (username: admin, password: admin)")
	}

	// Migrations: add columns that may not exist in older deployments
	migrations := []string{
		`ALTER TABLE providers ADD COLUMN IF NOT EXISTS display_name VARCHAR(255)`,
		`ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS user_id VARCHAR(255)`,
	}
	for _, m := range migrations {
		if _, err := DB.Exec(m); err != nil {
			return err
		}
	}

	return nil
}

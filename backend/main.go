package main

import (
	"log"
	"os"

	"iac-tool/internal/api"
	"iac-tool/internal/crypto"
	"iac-tool/internal/database"
	"iac-tool/internal/gpg"
	"iac-tool/internal/registry"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize encryption
	if err := crypto.Init(); err != nil {
		log.Fatalf("Failed to initialize encryption: %v", err)
	}

	// Initialize registry token
	if err := registry.InitToken(); err != nil {
		log.Fatalf("Failed to initialize registry token: %v", err)
	}
	log.Println("✓ Registry authentication token initialized")

	// Initialize database
	if err := database.Init(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize runner API key (create if doesn't exist)
	if err := api.InitRunnerAPIKey(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize GPG for signing providers
	if err := gpg.Init(); err != nil {
		log.Printf("Warning: GPG initialization failed: %v", err)
		log.Println("Providers will not be signed")
	} else {
		log.Printf("GPG initialized with key ID: %s", gpg.GetKeyID())
	}

	r := gin.Default()

	// CORS configuration
	config := cors.DefaultConfig()

	// Get host and ports from environment
	frontendHost := os.Getenv("FRONTEND_HOST")
	if frontendHost == "" {
		frontendHost = "localhost"
	}
	frontendPort := os.Getenv("FRONTEND_PORT")
	if frontendPort == "" {
		frontendPort = "3000"
	}
	viteDevPort := os.Getenv("VITE_DEV_PORT")
	if viteDevPort == "" {
		viteDevPort = "5173"
	}

	config.AllowOrigins = []string{
		"http://" + frontendHost + ":" + frontendPort,
		"http://" + frontendHost + ":" + viteDevPort,
		"*",
	}
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-API-Key"}
	r.Use(cors.New(config))

	// =========================================================================
	// Terraform Service Discovery (/.well-known/terraform.json)
	// =========================================================================
	r.GET("/.well-known/terraform.json", api.ServiceDiscovery)

	// =========================================================================
	// Static file downloads (provider binaries)
	// =========================================================================
	buildDir := os.Getenv("BUILD_DIR")
	if buildDir == "" {
		buildDir = "/app/data/builds"
	}
	r.Static("/downloads", buildDir)

	// SHA256SUMS and signature endpoints for provider verification
	r.GET("/shasums/providers/:namespace/:name/:version", api.GetProviderSHASums)
	r.GET("/shasums/providers/:namespace/:name/:version/sig", api.GetProviderSHASumsSig)

	// =========================================================================
	// Terraform Registry Protocol v1 (for terraform init/get)
	// These endpoints require API key authentication for Terraform CLI
	// =========================================================================
	v1 := r.Group("/v1")
	v1.Use(api.TerraformAuthMiddleware()) // Only checks auth for Terraform protocol
	{
		// Module Registry Protocol
		modules := v1.Group("/modules")
		{
			modules.GET("/:namespace/:name/:provider/versions", api.TFListModuleVersions)
			modules.GET("/:namespace/:name/:provider/:version/download", api.TFDownloadModule)
		}

		// Provider Registry Protocol
		providers := v1.Group("/providers")
		{
			providers.GET("/:namespace/:name/versions", api.TFListProviderVersions)
			providers.GET("/:namespace/:name/:version/download/:os/:arch", api.TFDownloadProvider)
		}
	}

	// =========================================================================
	// Management API (for frontend) - NO AUTHENTICATION REQUIRED
	// =========================================================================
	apiGroup := r.Group("/api")
	{
		// Modules
		apiGroup.GET("/modules", api.GetModules)
		apiGroup.GET("/modules/:id", api.GetModule)
		apiGroup.GET("/modules/:id/versions", api.GetModuleVersions)
		apiGroup.GET("/modules/:id/git-tags", api.GetModuleGitTags)
		apiGroup.GET("/modules/:id/readme", api.GetModuleReadme)
		apiGroup.POST("/modules", api.CreateModuleFromGit)
		apiGroup.PUT("/modules/:id", api.UpdateModule)
		apiGroup.DELETE("/modules/:id", api.DeleteModuleByID)
		apiGroup.POST("/modules/:id/sync-tags", api.SyncModuleTags)
		apiGroup.POST("/modules/:id/versions", api.AddModuleVersion)
		apiGroup.PATCH("/modules/:id/versions/:versionId", api.ToggleModuleVersion)
		apiGroup.DELETE("/modules/:id/versions/:versionId", api.DeleteModuleVersionByID)

		// Providers
		apiGroup.GET("/providers", api.GetProviders)
		apiGroup.GET("/providers/:id", api.GetProvider)
		apiGroup.GET("/providers/:id/versions", api.GetProviderVersions)
		apiGroup.GET("/providers/:id/git-tags", api.GetProviderGitTags)
		apiGroup.GET("/providers/:id/readme", api.GetProviderReadme)
		apiGroup.POST("/providers", api.CreateProviderFromGit)
		apiGroup.DELETE("/providers/:id", api.DeleteProviderByID)
		apiGroup.POST("/providers/:id/sync-tags", api.SyncProviderTags)
		apiGroup.POST("/providers/:id/versions", api.AddProviderVersion)
		apiGroup.PATCH("/providers/:id/versions/:versionId", api.ToggleProviderVersion)
		apiGroup.DELETE("/providers/:id/versions/:versionId", api.DeleteProviderVersionByID)
		apiGroup.GET("/providers/:id/versions/:versionId/platforms", api.GetProviderPlatforms)
		apiGroup.POST("/providers/:id/versions/:versionId/platforms", api.AddProviderPlatform)
		apiGroup.DELETE("/providers/:id/versions/:versionId/platforms/:platformId", api.DeleteProviderPlatform)
		apiGroup.POST("/providers/:id/versions/:versionId/platforms/upload", api.UploadProviderPlatform)

		// Namespaces
		apiGroup.GET("/namespaces", api.GetNamespaces)
		apiGroup.GET("/namespaces/:id", api.GetNamespace)
		apiGroup.POST("/namespaces", api.CreateNamespace)
		apiGroup.PATCH("/namespaces/:id", api.UpdateNamespace)
		apiGroup.DELETE("/namespaces/:id", api.DeleteNamespace)

		// API Keys (global, for Terraform CLI access to all namespaces)
		apiGroup.GET("/api-keys", api.GetAPIKeys)
		apiGroup.POST("/api-keys", api.CreateAPIKey)
		apiGroup.DELETE("/api-keys/:keyId", api.DeleteAPIKey)

		// Deployments
		apiGroup.GET("/deployments", api.ListDeployments)
		apiGroup.GET("/deployments/:id", api.GetDeployment)
		apiGroup.POST("/deployments", api.CreateDeployment)
		apiGroup.DELETE("/deployments/:id", api.DeleteDeployment)
		apiGroup.GET("/deployments/:id/references", api.GetDeploymentReferences)
		apiGroup.GET("/deployments/:id/browse", api.GetDeploymentDirectory)
		apiGroup.GET("/deployments/:id/tfvars", api.GetTfvarsFiles)
		apiGroup.POST("/deployments/:id/runs", api.CreateDeploymentRun)
		apiGroup.GET("/deployments/:id/runs", api.ListDeploymentRuns)
		apiGroup.GET("/deployments/:id/runs/:runId", api.GetDeploymentRun)
		apiGroup.GET("/deployments/:id/runs/:runId/stream", api.StreamDeploymentRunLogs)
		apiGroup.POST("/deployments/:id/runs/:runId/approve", api.ApproveDeploymentRun)
		apiGroup.POST("/deployments/:id/runs/:runId/cancel", api.CancelDeploymentRun)
		apiGroup.DELETE("/deployments/:id/runs/:runId", api.DeleteDeploymentRun)
		apiGroup.GET("/deployments/:id/status", api.GetDirectoryStatus)

		// Internal registry token endpoint (for runner)
		apiGroup.GET("/internal/registry-token", api.GetRegistryToken)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "9080"
	}
	registryHost := os.Getenv("REGISTRY_HOST")
	if registryHost == "" {
		registryHost = "localhost"
	}

	log.Printf("Terraform Private Registry starting on :%s\n", port)
	log.Printf("Service discovery: http://%s:%s/.well-known/terraform.json\n", registryHost, port)
	log.Printf("Module registry:   http://%s:%s/v1/modules/\n", registryHost, port)
	log.Printf("Provider registry: http://%s:%s/v1/providers/\n", registryHost, port)
	log.Printf("Management API:    http://%s:%s/api/\n", registryHost, port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

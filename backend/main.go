package main

import (
	"log"
	"os"

	"iac-tool/internal/api"
	"iac-tool/internal/crypto"
	"iac-tool/internal/database"
	"iac-tool/internal/gpg"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize encryption
	if err := crypto.Init(); err != nil {
		log.Fatalf("Failed to initialize encryption: %v", err)
	}

	// Initialize database
	if err := database.Init(); err != nil {
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
	config.AllowOrigins = []string{"http://localhost:3000", "http://localhost:5173", "*"}
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
			modules.GET("/:namespace/:provider/:name/versions", api.TFListModuleVersions)
			modules.GET("/:namespace/:provider/:name/:version/download", api.TFDownloadModule)
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

		// API Keys (for Terraform CLI access)
		apiGroup.GET("/namespaces/:id/api-keys", api.GetAPIKeys)
		apiGroup.POST("/namespaces/:id/api-keys", api.CreateAPIKey)
		apiGroup.DELETE("/namespaces/:id/api-keys/:keyId", api.DeleteAPIKey)

		// Deployments
		apiGroup.GET("/deployments", api.ListDeployments)
		apiGroup.GET("/deployments/:id", api.GetDeployment)
		apiGroup.POST("/deployments", api.CreateDeployment)
		apiGroup.DELETE("/deployments/:id", api.DeleteDeployment)
		apiGroup.GET("/deployments/:id/references", api.GetDeploymentReferences)
		apiGroup.GET("/deployments/:id/browse", api.GetDeploymentDirectory)
		apiGroup.POST("/deployments/:id/runs", api.CreateDeploymentRun)
		apiGroup.GET("/deployments/:id/runs", api.ListDeploymentRuns)
		apiGroup.GET("/deployments/:id/runs/:runId", api.GetDeploymentRun)
		apiGroup.POST("/deployments/:id/runs/:runId/approve", api.ApproveDeploymentRun)
		apiGroup.GET("/deployments/:id/status", api.GetDirectoryStatus)
	}

	log.Println("Terraform Private Registry starting on :9080")
	log.Println("Service discovery: http://localhost:9080/.well-known/terraform.json")
	log.Println("Module registry:   http://localhost:9080/v1/modules/")
	log.Println("Provider registry: http://localhost:9080/v1/providers/")
	log.Println("Management API:    http://localhost:9080/api/")
	if err := r.Run(":9080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

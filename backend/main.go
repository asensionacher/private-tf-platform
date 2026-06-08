package main

import (
	"log"
	"os"
	"path/filepath"

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
	backendHost := os.Getenv("BACKEND_HOST")
	if backendHost == "" {
		backendHost = "localhost"
	}

	registryHost := os.Getenv("REGISTRY_HOST")
	if registryHost == "" {
		registryHost = "registry.lan"
	}

	platformTfstatesHost := os.Getenv("PLATFORM_TFSTATES_HOST")
	if platformTfstatesHost == "" {
		platformTfstatesHost = "tf.registry.lan"
	}

	apiRegistryHost := os.Getenv("API_REGISTRY_HOST")
	if apiRegistryHost == "" {
		apiRegistryHost = "api.registry.lan"
	}

	allowedOrigins := []string{
		"http://" + frontendHost,
		"http://" + frontendHost + ":" + frontendPort,
		"http://" + frontendHost + ":" + viteDevPort,
		"https://" + frontendHost,
		"https://" + frontendHost + ":" + frontendPort,
		"http://" + backendHost,
		"https://" + backendHost,
		"http://" + registryHost,
		"https://" + registryHost,
		"http://" + apiRegistryHost,
		"https://" + apiRegistryHost,
		"http://" + platformTfstatesHost,
		"https://" + platformTfstatesHost,
		"http://localhost:5173",
		"https://localhost:5173",
		"http://localhost:3000",
		"https://localhost:3000",
	}
	config.AllowOrigins = allowedOrigins
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "LOCK", "UNLOCK"}
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
	// Static file downloads (provider binaries) — no auth, URL is only discoverable via authenticated /v1 endpoints
	// Path: /downloads/providers/:namespace/:name/:version/:filename
	downloads := r.Group("/downloads")
	downloads.GET("/providers/:namespace/*filepath", func(c *gin.Context) {
		filePath := c.Param("filepath")
		c.File(filepath.Join(buildDir, "providers", c.Param("namespace"), filePath))
	})

	// SHA256SUMS and signature endpoints — no auth, same reasoning as /downloads
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
	// Authentication endpoints (public — no JWT required)
	// =========================================================================
	r.POST("/api/auth/login", api.Login)

	// =========================================================================
	// Management API (for frontend) - JWT AUTHENTICATION REQUIRED
	// =========================================================================
	apiGroup := r.Group("/api")
	apiGroup.Use(api.JWTAuthMiddleware())
	{
		// Current user info
		apiGroup.GET("/auth/me", api.GetCurrentUser)

		// User management (admin only)
		adminUsers := apiGroup.Group("/users")
		adminUsers.Use(api.AdminOnlyMiddleware())
		{
			adminUsers.GET("", api.GetUsers)
			adminUsers.POST("", api.CreateUser)
			adminUsers.PUT("/:id", api.UpdateUser)
			adminUsers.PATCH("/:id/password", api.ChangeUserPassword)
			adminUsers.DELETE("/:id", api.DeleteUser)
		}

		// Modules
		apiGroup.GET("/modules", api.GetModules)
		apiGroup.GET("/modules/:id", api.GetModule)
		apiGroup.GET("/modules/:id/versions", api.GetModuleVersions)
		apiGroup.GET("/modules/:id/git-tags", api.GetModuleGitTags)
		apiGroup.GET("/modules/:id/readme", api.GetModuleReadme)
		// Admin-only write operations
		adminGroup := apiGroup.Group("")
		adminGroup.Use(api.AdminOnlyMiddleware())
		{
			adminGroup.POST("/modules", api.CreateModuleFromGit)
			adminGroup.PUT("/modules/:id", api.UpdateModule)
			adminGroup.DELETE("/modules/:id", api.DeleteModuleByID)
			adminGroup.POST("/modules/:id/sync-tags", api.SyncModuleTags)
			adminGroup.POST("/modules/:id/versions", api.AddModuleVersion)
			adminGroup.PATCH("/modules/:id/versions/:versionId", api.ToggleModuleVersion)
			adminGroup.DELETE("/modules/:id/versions/:versionId", api.DeleteModuleVersionByID)

			adminGroup.POST("/providers", api.CreateProviderFromGit)
			adminGroup.DELETE("/providers/:id", api.DeleteProviderByID)
			adminGroup.POST("/providers/:id/sync-tags", api.SyncProviderTags)
			adminGroup.POST("/providers/:id/versions", api.AddProviderVersion)
			adminGroup.PATCH("/providers/:id/versions/:versionId", api.ToggleProviderVersion)
			adminGroup.DELETE("/providers/:id/versions/:versionId", api.DeleteProviderVersionByID)
			adminGroup.POST("/providers/:id/versions/:versionId/platforms", api.AddProviderPlatform)
			adminGroup.DELETE("/providers/:id/versions/:versionId/platforms/:platformId", api.DeleteProviderPlatform)
			adminGroup.POST("/providers/:id/versions/:versionId/platforms/upload", api.UploadProviderPlatform)

			adminGroup.POST("/namespaces", api.CreateNamespace)
			adminGroup.PATCH("/namespaces/:id", api.UpdateNamespace)
			adminGroup.DELETE("/namespaces/:id", api.DeleteNamespace)
		}

		// Providers (read-only, open to all authenticated users)
		apiGroup.GET("/providers", api.GetProviders)
		apiGroup.GET("/providers/:id", api.GetProvider)
		apiGroup.GET("/providers/:id/versions", api.GetProviderVersions)
		apiGroup.GET("/providers/:id/git-tags", api.GetProviderGitTags)
		apiGroup.GET("/providers/:id/readme", api.GetProviderReadme)
		apiGroup.GET("/providers/:id/versions/:versionId/platforms", api.GetProviderPlatforms)

		// Namespaces (read-only, open to all authenticated users)
		apiGroup.GET("/namespaces", api.GetNamespaces)
		apiGroup.GET("/namespaces/:id", api.GetNamespace)

		// API Keys (readers can manage their own; admins can manage all)
		apiGroup.GET("/api-keys", api.GetAPIKeys)
		apiGroup.POST("/api-keys", api.CreateAPIKey)
		apiGroup.DELETE("/api-keys/:keyId", api.DeleteAPIKey)

		// TF State management endpoints (for UI) - JWT auth
		apiGroup.GET("/tfstates", api.ListAllTFStateDeployments)
		apiGroup.GET("/deployments/:id/tfstates", api.ListTFStates)
		apiGroup.GET("/deployments/:id/tfstates/:workspace/raw", api.GetTFStateRaw)
		apiGroup.DELETE("/deployments/:id/tfstates/:workspace/lock", api.ForceUnlockTFState)
		apiGroup.DELETE("/deployments/:id/tfstates/:workspace", api.DeleteTFStateWorkspace)
	}

	// =========================================================================
	// TF State HTTP backend protocol endpoints - BASIC AUTH (username:password)
	// Terraform/OpenTofu CLI does not send JWT; it uses HTTP Basic Auth.
	// =========================================================================
	tfStateGroup := r.Group("/api/tfstate")
	tfStateGroup.Use(api.TFStateBasicAuthMiddleware())
	{
		tfStateGroup.GET("/:deploymentId", api.GetTFState)
		tfStateGroup.POST("/:deploymentId", api.UpdateTFState)
		tfStateGroup.DELETE("/:deploymentId", api.DeleteTFState)
		tfStateGroup.Handle("LOCK", "/:deploymentId", api.LockTFState)
		tfStateGroup.Handle("UNLOCK", "/:deploymentId", api.UnlockTFState)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "9080"
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

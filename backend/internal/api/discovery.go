package api

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// ServiceDiscovery handles the Terraform service discovery protocol
// Docs: https://www.terraform.io/internals/remote-service-discovery
func ServiceDiscovery(c *gin.Context) {
	baseURL := getBaseURL(c)

	c.JSON(http.StatusOK, gin.H{
		"modules.v1":   baseURL + "/v1/modules/",
		"providers.v1": baseURL + "/v1/providers/",
	})
}

// getBaseURL constructs the base URL from the request
func getBaseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	// Check for X-Forwarded-Proto header (for reverse proxies)
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}

	host := c.Request.Host
	// Use configured registry host from environment
	registryHost := os.Getenv("REGISTRY_HOST")
	if registryHost == "" {
		registryHost = "registry.lan"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "9080"
	}

	// If the request came through a reverse proxy (nginx), Host will already
	// be the public hostname without a port — use it as-is.
	// Only fall back to the internal host:port when the request is direct
	// (i.e. Host is localhost or bare internal hostname without a port).
	if host == "localhost" || host == "localhost:"+port {
		host = registryHost + ":" + port
	}

	return scheme + "://" + host
}

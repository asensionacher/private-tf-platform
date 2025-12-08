package api

import (
	"net/http"

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
	return scheme + "://" + c.Request.Host
}

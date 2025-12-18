package api

import (
	"iac-tool/internal/registry"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetRegistryToken returns the registry authentication token for internal use (runner)
func GetRegistryToken(c *gin.Context) {
	// This endpoint should only be accessible from the internal network
	// In production, you might want to add IP whitelist or other security

	token := registry.GetToken()
	c.JSON(http.StatusOK, gin.H{
		"token": token,
	})
}

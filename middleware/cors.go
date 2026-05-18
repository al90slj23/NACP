package middleware

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	if common.IsBlackboxEnabled() {
		return func(c *gin.Context) {
			if common.BlackboxMaskUnauthRelay && c.Request.Method == http.MethodOptions && !HasBrowserSession(c) && !hasRelayAuthHint(c) {
				AbortBlackboxNotFound(c)
				return
			}
			c.Next()
		}
	}
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowCredentials = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"*"}
	return cors.New(config)
}

func PoweredBy() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !(common.IsBlackboxEnabled() && common.BlackboxMaskHeaders) {
			c.Header("X-New-Api-Version", common.Version)
		}
		c.Next()
	}
}

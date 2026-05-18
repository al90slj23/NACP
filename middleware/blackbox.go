package middleware

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const BlackboxLoginHeader = "X-Login-Path"

func hasRelayAuthHint(c *gin.Context) bool {
	return c.GetHeader("Authorization") != "" ||
		c.GetHeader("x-api-key") != "" ||
		c.GetHeader("x-goog-api-key") != "" ||
		c.GetHeader("mj-api-secret") != "" ||
		c.Query("key") != ""
}

func HasBrowserSession(c *gin.Context) bool {
	session := sessions.Default(c)
	id := session.Get("id")
	if id == nil {
		return false
	}
	if status, ok := session.Get("status").(int); ok && status == common.UserStatusDisabled {
		return false
	}
	return true
}

func AbortBlackboxNotFound(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Status(http.StatusNotFound)
	c.Abort()
}

func AbortBlackboxIfEnabled(c *gin.Context) bool {
	if common.IsBlackboxEnabled() {
		AbortBlackboxNotFound(c)
		return true
	}
	return false
}

func BlackboxSessionRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if common.IsBlackboxEnabled() && !HasBrowserSession(c) {
			AbortBlackboxNotFound(c)
			return
		}
		c.Next()
	}
}

func BlackboxLoginGate() gin.HandlerFunc {
	return func(c *gin.Context) {
		if common.IsBlackboxEnabled() && c.GetHeader(BlackboxLoginHeader) != common.BlackboxLoginPath {
			AbortBlackboxNotFound(c)
			return
		}
		c.Next()
	}
}

func BlackboxRegisterGate() gin.HandlerFunc {
	return func(c *gin.Context) {
		if common.IsBlackboxEnabled() && !common.BlackboxPublicRegister {
			AbortBlackboxNotFound(c)
			return
		}
		c.Next()
	}
}

func BlackboxOAuthGate() gin.HandlerFunc {
	return func(c *gin.Context) {
		if common.IsBlackboxEnabled() && !common.BlackboxPublicOAuth {
			AbortBlackboxNotFound(c)
			return
		}
		c.Next()
	}
}

func BlackboxWebhookSignatureGate(headerNames ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if common.IsBlackboxEnabled() {
			for _, name := range headerNames {
				if strings.TrimSpace(c.GetHeader(name)) != "" {
					c.Next()
					return
				}
			}
			AbortBlackboxNotFound(c)
			return
		}
		c.Next()
	}
}

func BlackboxEpayWebhookGate() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !common.IsBlackboxEnabled() {
			c.Next()
			return
		}
		if strings.TrimSpace(c.Query("sign")) != "" {
			c.Next()
			return
		}
		if c.Request.Method == http.MethodPost {
			_ = c.Request.ParseForm()
			if strings.TrimSpace(c.Request.PostForm.Get("sign")) != "" {
				c.Next()
				return
			}
		}
		AbortBlackboxNotFound(c)
	}
}

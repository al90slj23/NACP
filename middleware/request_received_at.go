package middleware

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"

	"github.com/gin-gonic/gin"
)

func RequestReceivedAt() gin.HandlerFunc {
	return func(c *gin.Context) {
		if existing := common.GetContextKeyTime(c, constant.ContextKeyRelayReceivedAt); existing.IsZero() {
			common.SetContextKey(c, constant.ContextKeyRelayReceivedAt, time.Now())
		}
		c.Next()
	}
}

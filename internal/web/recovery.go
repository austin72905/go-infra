package web

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"log/slog"
)

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				ctx := c.Request.Context()
				slog.ErrorContext(
					ctx,
					"http request panic recovered",
					"requestID", RequestID(ctx),
					"correlationID", CorrelationID(ctx),
					"action", Action(ctx),
					"panic", recovered,
					"stackTrace", string(debug.Stack()),
				)

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"requestID": RequestID(ctx),
					"errorCode": "INTERNAL_ERROR",
					"message":   "Internal Server Error",
				})
			}
		}()

		c.Next()
	}
}

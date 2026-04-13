package middlewares

import (
	"time"

	"inventory-manage/global"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ZapLogger is a Gin middleware that logs each request with Zap.
// Required fields: trace_id (from X-Request-ID or generated), latency, status, path.
func ZapLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		traceID := c.GetHeader("X-Request-ID")
		if traceID == "" {
			traceID = c.GetString("trace_id")
		}

		logFields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
			zap.String("trace_id", traceID),
		}

		// Log at appropriate level based on status code
		switch {
		case status >= 500:
			global.Logger.Error("Server error", logFields...)
		case status >= 400:
			global.Logger.Warn("Client error", logFields...)
		default:
			global.Logger.Info("Request", logFields...)
		}
	}
}

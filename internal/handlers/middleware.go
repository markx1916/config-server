package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GinZapLogger is a Gin middleware for logging requests using Zap.
// It logs the request path, method, status code, latency, client IP, and user agent.
func GinZapLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		userAgent := c.Request.UserAgent()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String() // Get private errors

		logFields := []zap.Field{
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status_code", statusCode),
			zap.Duration("latency", latency),
			zap.String("client_ip", clientIP),
			zap.String("user_agent", userAgent),
		}
		if query != "" {
			logFields = append(logFields, zap.String("query", query))
		}
		if errorMessage != "" {
			logFields = append(logFields, zap.String("error_message", errorMessage))
		}

		if statusCode >= 400 && statusCode < 500 {
			logger.Warn("Client Error", logFields...)
		} else if statusCode >= 500 {
			logger.Error("Server Error", logFields...)
		} else {
			logger.Info("Request Handled", logFields...)
		}
	}
}

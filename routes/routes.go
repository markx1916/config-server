package routes

import (
	"net/http"
	"time"

	"config-server/handlers" // Adjust to your module path
	"config-server/utils"    // Adjust to your module path
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SetupRouter initializes and configures the Gin router.
func SetupRouter() *gin.Engine {
	// Set Gin to release mode if not in development for better performance
	// gin.SetMode(gin.ReleaseMode) // Consider making this configurable

	router := gin.New()

	// CORS Middleware
	// For development, allow common frontend dev server origins.
	// For production, this should be configured to your specific frontend domain.
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://127.0.0.1:5173", "http://localhost:3000"}, // Vite default, common alternatives
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Logger middleware
	router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Custom log format using Zap
		utils.Logger.Info("GIN",
			zap.String("method", param.Method),
			zap.String("path", param.Path),
			zap.Int("status", param.StatusCode),
			zap.Duration("latency", param.Latency),
			zap.String("ip", param.ClientIP),
			zap.String("user_agent", param.Request.UserAgent()),
			zap.String("error", param.ErrorMessage),
		)
		return "" // We've logged it with Zap, so don't have Gin log it again.
	}))

	// Recovery middleware recovers from any panics and writes a 500 if there was one.
	router.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			utils.Logger.Error("Panic recovered", zap.String("error", err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error: " + err})
		} else if err, ok := recovered.(error); ok {
			utils.Logger.Error("Panic recovered", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error: " + err.Error()})
		} else {
			utils.Logger.Error("Panic recovered", zap.Any("recovered", recovered))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		c.AbortWithStatus(http.StatusInternalServerError)
	}))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	// API v1 group
	apiV1 := router.Group("/api")
	{
		nacosInstanceRoutes := apiV1.Group("/nacos/instances")
		{
			nacosInstanceRoutes.POST("", handlers.CreateNacosInstance)
			nacosInstanceRoutes.GET("", handlers.ListNacosInstances)
			nacosInstanceRoutes.GET("/:id", handlers.GetNacosInstance)
			nacosInstanceRoutes.PUT("/:id", handlers.UpdateNacosInstance)
			nacosInstanceRoutes.DELETE("/:id", handlers.DeleteNacosInstance)
		}

		configRoutes := apiV1.Group("/nacos/configs")
		{
			configRoutes.GET("", handlers.ListConfigurations)
			configRoutes.POST("", handlers.CreateConfiguration)
			configRoutes.GET("/:id", handlers.GetConfiguration)
			configRoutes.PUT("/:id", handlers.UpdateConfiguration)
			configRoutes.GET("/:id/diff", handlers.GetConfigurationDiff)

			// Publish, History, Rollback, Deployments
			configRoutes.POST("/:id/publish/:type", handlers.PublishConfigurationToNacos) // type: gray or full
			configRoutes.GET("/:id/history", handlers.GetConfigurationHistoryList)
			configRoutes.POST("/:id/rollback/:history_id", handlers.RollbackConfiguration)
			configRoutes.GET("/:id/deployments", handlers.GetDeploymentHistoryList)
		}
	}

	utils.Logger.Info("Router setup complete with Nacos Instance, Configuration, Publish, History, and Rollback routes.")
	return router
}

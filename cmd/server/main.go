package main

import (
	"context"
	"fmt"
	"log"
	"nacos-config-tool/internal/config"
	"nacos-config-tool/internal/database"
	"nacos-config-tool/internal/handlers"
	applogger "nacos-config-tool/internal/logger" // Renamed to avoid conflict
	"nacos-config-tool/pkg/nacosclient"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.uber.org/zap"
)

var (
	serviceName = "nacos-config-tool"
	version     = "0.1.0" // Application version
)

// initTracer initializes an OpenTelemetry tracer provider.
func initTracer() (*sdktrace.TracerProvider, error) {
	// Create a new exporter. In a production environment, you would use
	// something like Jaeger, Zipkin, or an OTLP exporter.
	// For this example, we use a stdout exporter.
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, fmt.Errorf("failed to create stdouttrace exporter: %w", err)
	}

	// Create a new resource with service name and version.
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create a new tracer provider with the exporter and resource.
	// We are using BatchSpanProcessor for better performance in production.
	// For simplicity, AlwaysSample is used here. In production, you might want to use ParentBasedSampler or TraceIDRatioBased.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set the global tracer provider.
	otel.SetTracerProvider(tp)

	// Set the global propagator to W3C Trace Context.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	log.Println("OpenTelemetry tracer initialized with stdout exporter.")
	return tp, nil
}

func main() {
	// Initialize OpenTelemetry Tracer
	tp, err := initTracer()
	if err != nil {
		log.Fatalf("Failed to initialize OpenTelemetry tracer: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	// Load Configuration
	// Pass the path to the config file if needed, e.g., "./config.yaml"
	// Viper will search in default paths if no path is provided.
	appCfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize Logger
	// Determine environment for logger: "development" or "production"
	logEnv := os.Getenv("GIN_MODE") // GIN_MODE can be 'debug', 'release', 'test'
	loggerEnv := "development"
	if logEnv == gin.ReleaseMode {
		loggerEnv = "production"
	}
	applogger.InitLogger(loggerEnv) // Using the aliased import
	defer applogger.Sync()          // Flush logs at the end

	applogger.Info("Application starting...", zap.String("version", version), zap.String("port", appCfg.ServerPort))

	// Initialize Database
	db, err := database.InitDB(appCfg)
	if err != nil {
		applogger.Fatal("Failed to initialize database", zap.Error(err))
	}
	applogger.Info("Database initialized and migrations completed.")
	defer database.CloseDB() // Ensure DB connection is closed

	// Initialize Nacos Client Manager
	// The NacosConfig part of AppConfig is passed for default client settings
	nacosClientMgr := nacosclient.NewNacosClientManager(appCfg.Nacos)
	applogger.Info("Nacos Client Manager initialized.")

	// Setup Gin Router
	// Set Gin mode based on environment
	if loggerEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	router := gin.New()

	// Middlewares
	router.Use(gin.Recovery()) // Recover from any panics
	// Add OpenTelemetry Gin middleware
	router.Use(otelgin.Middleware(serviceName))
	// Custom logger middleware using Zap
	router.Use(handlers.GinZapLogger(applogger.Log))


	// Register API Routes
	// Nacos Instance Handlers
	nacosInstanceHandler := handlers.NewNacosInstanceHandler(db, nacosClientMgr, *appCfg)
	instanceRoutes := router.Group("/api/nacos-instances")
	{
		instanceRoutes.POST("", nacosInstanceHandler.CreateNacosInstance)
		instanceRoutes.GET("", nacosInstanceHandler.ListNacosInstances)
		instanceRoutes.GET("/:id", nacosInstanceHandler.GetNacosInstance)
		instanceRoutes.PUT("/:id", nacosInstanceHandler.UpdateNacosInstance)
		instanceRoutes.DELETE("/:id", nacosInstanceHandler.DeleteNacosInstance)
		instanceRoutes.POST("/:id/test-connection", nacosInstanceHandler.TestNacosInstanceConnection)
		instanceRoutes.GET("/:id/namespaces", nacosInstanceHandler.ListNacosInstanceNamespaces)
	}

	// Configuration Handlers
	configHandler := handlers.NewConfigurationHandler(db, nacosClientMgr, *appCfg)
	configRoutes := router.Group("/api/configurations")
	{
		configRoutes.POST("", configHandler.CreateConfiguration)
		configRoutes.GET("", configHandler.ListConfigurations)
		configRoutes.GET("/:id", configHandler.GetConfiguration)
		configRoutes.PUT("/:id", configHandler.UpdateConfiguration)
		configRoutes.GET("/:id/diff-nacos", configHandler.GetConfigurationDiffWithNacos)
		configRoutes.POST("/:id/fetch-from-nacos", configHandler.FetchConfigurationFromNacos)
		// History and Rollback
		configRoutes.GET("/:id/history", configHandler.ListConfigurationHistory)
		configRoutes.GET("/history/:historyId", configHandler.GetConfigurationHistoryEntry) // Note: path changed slightly to avoid conflict if :id was also /history
		configRoutes.POST("/:id/rollback/:historyId", configHandler.RollbackConfiguration)
	}

	// Publishing Handler
	publishHandler := handlers.NewPublishHandler(db, nacosClientMgr, *appCfg)
	// Publish routes are under configurations
	configRoutes.POST("/:id/publish", publishHandler.PublishConfiguration)


	// Publish Record Handler
	publishRecordHandler := handlers.NewPublishRecordHandler(db)
	recordRoutes := router.Group("/api/publish-records")
	{
		recordRoutes.GET("", publishRecordHandler.ListPublishRecords)
		recordRoutes.GET("/:id", publishRecordHandler.GetPublishRecord)
	}
	
	// Health Check endpoint
	router.GET("/health", func(c *gin.Context) {
		// TODO: more comprehensive health check (DB, Nacos connectivity)
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})


	// Start HTTP Server
	srv := &http.Server{
		Addr:    ":" + appCfg.ServerPort,
		Handler: router,
	}

	go func() {
		// Service connections
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			applogger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	applogger.Info("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the requests it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		applogger.Fatal("Server forced to shutdown:", zap.Error(err))
	}

	applogger.Info("Server exiting")
}

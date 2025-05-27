package main

import (
	"fmt"
	"os"

	"config-server/config" // Adjust to your module path
	"config-server/db"     // Adjust to your module path
	"config-server/routes" // Adjust to your module path
	"config-server/utils"  // Adjust to your module path
	"go.uber.org/zap"
)

func main() {
	// Determine environment (dev, prod, etc.) - could be from an env var
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development" // Default to development
	}

	// Initialize logger
	utils.InitLogger(env)
	defer utils.Sync() // Flush logger buffer before exiting

	// Load configuration
	// You might want to pass the config path as a flag or env var
	// Ensure config.yaml is in the same directory as the executable or specify the correct path.
	// For development, if running `go run main.go` from the project root, "." is correct.
	cfg, err := config.LoadConfig(".")
	if err != nil {
		utils.Logger.Fatal("Failed to load configuration", zap.Error(err))
	}
	utils.Logger.Info("Configuration loaded successfully", zap.String("port", cfg.ServerPort), zap.String("db_dsn_preview", cfg.Database.DSN[:15]+"..."))

	// Initialize database
	// db.InitDB will set the global db.DB variable and also return it.
	_, err = db.InitDB(&cfg)
	if err != nil {
		utils.Logger.Fatal("Failed to initialize database", zap.Error(err))
	}
	// We can now use the global db.DB or the returned one.
	// For handlers, they typically use the global db.DB.
	utils.Logger.Info("Database initialized successfully")

	// Setup router
	router := routes.SetupRouter()
	utils.Logger.Info("Router setup complete, preparing to start server...")

	// Start server
	serverAddr := fmt.Sprintf(":%s", cfg.ServerPort)
	utils.Logger.Info("Starting server", zap.String("address", serverAddr))
	if err := router.Run(serverAddr); err != nil {
		utils.Logger.Fatal("Failed to start server", zap.Error(err))
	}
}

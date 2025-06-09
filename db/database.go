package db

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/yourusername/nacos-config-center/config" // Adjust to your module path
	"github.com/yourusername/nacos-config-center/models" // Adjust to your module path
	"github.com/yourusername/nacos-config-center/utils"  // Adjust to your module path

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB initializes the database connection and auto-migrates schemas.
func InitDB(cfg *config.Config) (*gorm.DB, error) {
	var err error

	dsn := cfg.Database.DSN
	if dsn == "" {
		return nil, fmt.Errorf("database DSN is not configured")
	}

	// GORM logger configuration
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level (Silent, Error, Warn, Info)
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			Colorful:                  true,        // Disable color
		},
	)

	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})

	if err != nil {
		utils.Logger.Error("Failed to connect to database", zap.Error(err))
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	utils.Logger.Info("Database connection established.")

	// Auto migrate the schema
	// Add more models here as they are created
	err = DB.AutoMigrate(
		&models.NacosInstance{},
		&models.Configuration{},
		&models.ConfigurationHistory{},
		&models.DeploymentHistory{},
	)
	if err != nil {
		utils.Logger.Error("Failed to auto-migrate database schema", zap.Error(err))
		return nil, fmt.Errorf("failed to auto-migrate schema: %w", err)
	}
	utils.Logger.Info("Database schema auto-migration successful for all models.")

	return DB, nil
}

// GetDB returns the current database instance.
func GetDB() *gorm.DB {
	return DB
}

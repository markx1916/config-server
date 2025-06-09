package database

import (
	"fmt"
	"log"
	"nacos-config-tool/internal/config"
	applogger "nacos-config-tool/internal/logger" // Renamed to avoid conflict with gormlogger
	"nacos-config-tool/internal/models"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/go-sql-driver/mysql" // Import MySQL driver for error type checking
)

var DB *gorm.DB

// InitDB initializes the database connection and auto-migrates schemas.
func InitDB(cfg *config.Config) (*gorm.DB, error) {
	var err error

	// Configure GORM logger
	var gormLogLevel gormlogger.LogLevel
	logLevelEnv := strings.ToUpper(os.Getenv("GORM_LOG_LEVEL"))
	switch logLevelEnv {
	case "SILENT":
		gormLogLevel = gormlogger.Silent
	case "ERROR":
		gormLogLevel = gormlogger.Error
	case "WARN":
		gormLogLevel = gormlogger.Warn
	case "INFO":
		gormLogLevel = gormlogger.Info
	default:
		// Default to Warn for GORM if LOG_LEVEL is not explicitly set for GORM
		// Or use the application's log level if desired (applogger.Log.Level())
		gormLogLevel = gormlogger.Warn
		if applogger.Log != nil { // Check if app logger is initialized
			// Match GORM's level to Zap's level if possible
			// This is a basic mapping, could be more sophisticated
			switch applogger.Log.Level() {
			case zapcore.DebugLevel, zapcore.InfoLevel:
				gormLogLevel = gormlogger.Info
			case zapcore.WarnLevel:
				gormLogLevel = gormlogger.Warn
			case zapcore.ErrorLevel, zapcore.FatalLevel, zapcore.PanicLevel:
				gormLogLevel = gormlogger.Error
			}
		}
	}

	var gormWriter gormlogger.Writer
	if applogger.Log != nil && applogger.Log.Core().Enabled(zapcore.DebugLevel) { // Check if underlying zap core is available and at a reasonable level
		// Use Zap's writer if available and configured for a level that makes sense for SQL logs
		gormWriter = log.New(applogger.Log.Sugar().Writer(), "\r\n", log.LstdFlags)
	} else {
		// Fallback to os.Stdout if logger's writer is not readily available or configured for a higher level
		gormWriter = log.New(os.Stdout, "\r\n", log.LstdFlags)
		applogger.Warn("GORM logger is falling back to os.Stdout as zap writer not directly accessible or configured for higher level.")
	}


	gormLog := gormlogger.New(
		gormWriter,
		gormlogger.Config{
			SlowThreshold:             200 * time.Millisecond, // Slow SQL threshold
			LogLevel:                  gormLogLevel,           // Log level
			IgnoreRecordNotFoundError: false,                  // Don't ignore ErrRecordNotFound error
			Colorful:                  false,                  // Disable color, as logs will go through Zap
			ParameterizedQueries:      true,                   // Log SQL queries with params (good for debugging, be careful with sensitive data in production)
		},
	)

	dsn := cfg.Database.DSN
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLog,
	})

	if err != nil {
		applogger.Error("Failed to connect to database", zap.Error(err), zap.String("dsn", dsn))
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	applogger.Info("Database connection established successfully.")

	// Auto-migrate the schema
	err = autoMigrate(DB)
	if err != nil {
		// Error already logged in autoMigrate
		return nil, fmt.Errorf("failed to auto-migrate database schemas: %w", err)
	}
	applogger.Info("Database migration completed.")

	return DB, nil
}

// autoMigrate automatically migrates GORM models to create/update database tables.
func autoMigrate(db *gorm.DB) error {
	// It's good practice to ensure tables for related models are created in order if there are FK constraints
	// that might cause issues, though GORM's AutoMigrate usually handles this.
	// Explicitly listing them also serves as documentation.
	err := db.AutoMigrate(
		&models.NacosInstance{},
		&models.Configuration{},      // Depends on NacosInstance
		&models.ConfigurationHistory{}, // Depends on Configuration
		&models.PublishRecord{},        // Depends on Configuration and NacosInstance
	)
	if err != nil {
		applogger.Error("GORM auto-migration failed", zap.Error(err))
		return err
	}
	applogger.Info("GORM auto-migration successful for all models.")
	return nil
}

// GetDB returns the current database instance.
// It's generally better to pass DB instances around (dependency injection)
// rather than using a global variable, but for simplicity in this context, a global is used.
func GetDB() *gorm.DB {
	if DB == nil {
		// This case should ideally not happen if InitDB is called at startup.
		// Consider a more graceful handling or ensure InitDB is always called.
		applogger.Fatal("Database not initialized. Call InitDB first.")
		// os.Exit(1) // or panic, depending on how critical this is
	}
	return DB
}

// CloseDB closes the database connection.
func CloseDB() {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			applogger.Error("Failed to get underlying SQL DB interface for closing", zap.Error(err))
			return
		}
		err = sqlDB.Close()
		if err != nil {
			applogger.Error("Failed to close database connection", zap.Error(err))
		} else {
			applogger.Info("Database connection closed successfully.")
		}
	}
}

// IsForeignKeyConstraintError checks if the provided error is a MySQL foreign key constraint error.
// MySQL error number for foreign key constraint violation is 1451 or 1217 for parent, 1452 for child.
// We check for 1451 (Cannot delete or update a parent row: a foreign key constraint fails)
// and 1217 (Cannot delete or update a parent row: a foreign key constraint fails (foreign key ...)).
// Error 1452 (Cannot add or update a child row: a foreign key constraint fails) is also a FK error but usually
// happens on INSERT/UPDATE of the child, not DELETE of parent.
func IsForeignKeyConstraintError(err error) bool {
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		switch mysqlErr.Number {
		case 1451, 1217: // Common errors for trying to delete/update a parent row that has children.
			return true
		// case 1452: // Error for trying to add/update a child row with an invalid foreign key.
		//  return true
		default:
			return false
		}
	}
	// Fallback for generic error messages if type assertion fails, though less reliable.
	if err != nil {
		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "foreign key constraint") && (strings.Contains(errMsg, "1451") || strings.Contains(errMsg, "1217")) {
			return true
		}
	}
	return false
}

package utils

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

// InitLogger initializes a Zap logger.
// It supports different configurations for development and production.
func InitLogger(env string) {
	var err error
	var config zap.Config
	var level zapcore.Level

	// Default to info level
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	err = level.Set(logLevel)
	if err != nil {
		level = zapcore.InfoLevel
	}

	if env == "development" || env == "dev" {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // Colorized output for dev
	} else {
		config = zap.NewProductionConfig()
	}

	config.Level = zap.NewAtomicLevelAt(level)

	Logger, err = config.Build()
	if err != nil {
		panic(err)
	}

	zap.ReplaceGlobals(Logger) // Optional: Replace global logger
	Logger.Info("Logger initialized", zap.String("env", env), zap.String("level", level.String()))
}

// Sync flushes any buffered log entries.
func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}

// Example of how to get a named logger (sub-logger)
func NewNamedLogger(name string) *zap.Logger {
    return Logger.Named(name)
}

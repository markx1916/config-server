package logger

import (
	"fmt"
	"log" // For initial error logging if Zap isn't ready
	"os"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Log  *zap.Logger
	once sync.Once
)

func init() {
	// Initialize with a default logger to ensure Log is never nil.
	// This default logger will be replaced by a call to InitLogger.
	once.Do(func() {
		// Using a development logger as a fallback. Using os.Stderr for error output of the logger itself.
		fallbackLogger, err := zap.NewDevelopment(zap.ErrorOutput(zapcore.AddSync(os.Stderr)))
		if err != nil {
			// This panic is unlikely but crucial if it happens.
			panic(fmt.Sprintf("Failed to create fallback Zap logger: %v", err))
		}
		Log = fallbackLogger
		// Using standard log package here for this initial message, as our Zap logger might not be fully configured.
		// log.Println("Logger initialized with a fallback development logger. Call InitLogger for proper application-level configuration.")
	})
}

// InitLogger initializes the global logger based on the environment (development/production)
// and log level from environment variable LOG_LEVEL (e.g., DEBUG, INFO, WARN, ERROR).
// It should be called once from main.go after loading configuration.
func InitLogger(env string, outputPath ...string) {
	// The 'once' mechanism is for the fallback. Here, we are intentionally re-assigning Log.
	// No additional lock needed here as this is expected to be called once at startup.
	// If multiple goroutines could call InitLogger, a sync.Mutex would be needed around Log assignment.

	var cfg zap.Config
	var err error

	// Determine log level
	logLevelEnv := strings.ToLower(os.Getenv("LOG_LEVEL"))
	level := zap.InfoLevel // Default level
	if logLevelEnv != "" {
		parsedLevel, pErr := zapcore.ParseLevel(logLevelEnv)
		if pErr == nil {
			level = parsedLevel
		} else {
			// Use standard log for this warning as Zap might not be fully set up.
			log.Printf("Warning: Invalid LOG_LEVEL '%s'. Defaulting to INFO. Error: %v\n", logLevelEnv, pErr)
		}
	}

	if env == "production" {
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.TimeKey = "timestamp" // Standardize TimeKey
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		cfg.EncoderConfig.MessageKey = "message" // Standardize MessageKey
		// Example: Add file output for production if specified
		// if len(outputPath) > 0 && outputPath[0] != "" {
		// 	cfg.OutputPaths = []string{"stdout", outputPath[0]} // Log to stdout and file
		// 	cfg.ErrorOutputPaths = []string{"stderr", outputPath[0]}
		// } else {
		cfg.OutputPaths = []string{"stdout"}
		cfg.ErrorOutputPaths = []string{"stderr"}
		// }
	} else { // development or other
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // Colored output for development
		cfg.EncoderConfig.MessageKey = "message"                         // Standardize MessageKey for dev too
	}

	cfg.Level = zap.NewAtomicLevelAt(level)

	// Build the logger
	newLogger, buildErr := cfg.Build()
	if buildErr != nil {
		// If building the logger fails, panic as it's a critical part of the app
		panic(fmt.Sprintf("Failed to initialize Zap logger: %v", buildErr))
	}
	Log = newLogger // Replace the fallback logger
	// Log.Info("Logger successfully configured.", zap.String("environment", env), zap.String("level", level.String()))
}

// Sync flushes any buffered log entries. It's good practice to call this before application exit.
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}

// For convenience, provide global logging functions.
// These now directly use the global `Log` which is guaranteed to be non-nil by `init()`.

func Info(message string, fields ...zap.Field) {
	Log.Info(message, fields...)
}

func Debug(message string, fields ...zap.Field) {
	Log.Debug(message, fields...)
}

func Warn(message string, fields ...zap.Field) {
	Log.Warn(message, fields...)
}

func Error(message string, fields ...zap.Field) {
	Log.Error(message, fields...)
}

// Fatal logs a message at FatalLevel and then calls os.Exit(1).
func Fatal(message string, fields ...zap.Field) {
	Log.Fatal(message, fields...)
}

// Panic logs a message at PanicLevel and then panics.
func Panic(message string, fields ...zap.Field) {
	Log.Panic(message, fields...)
}

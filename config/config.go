package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config stores all configuration of the application.
// The values are read by viper from a config file or environment variables.
type Config struct {
	ServerPort string   `mapstructure:"SERVER_PORT"`
	Database   Database `mapstructure:"DATABASE"`
	Nacos      Nacos    `mapstructure:"NACOS"` // For initial/fallback Nacos client
}

// Database stores the database connection parameters.
type Database struct {
	DSN string `mapstructure:"DSN"`
}

// Nacos stores the Nacos client configuration parameters.
type Nacos struct {
	ServerAddr  string `mapstructure:"SERVER_ADDR"`
	NamespaceID string `mapstructure:"NAMESPACE_ID"`
	Username    string `mapstructure:"USERNAME"`
	Password    string `mapstructure:"PASSWORD"`
	Scheme      string `mapstructure:"SCHEME"`
	ContextPath string `mapstructure:"CONTEXT_PATH"`
	TimeoutMs   uint64 `mapstructure:"TIMEOUT_MS"`
}

var AppConfig Config

// LoadConfig loads configuration from file and environment variables.
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path) // e.g., ".", "./config"
	viper.SetConfigName("config") // config file name without extension
	viper.SetConfigType("yaml")   // config file type

	viper.AutomaticEnv() // read in environment variables that match
	viper.SetEnvPrefix("NCT") // Nacos Config Tool prefix for env vars
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	err = viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			fmt.Println("Config file not found, using defaults or environment variables.")
		} else {
			// Config file was found but another error was produced
			return Config{}, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// You can set default values here if not found in config or env
	if config.ServerPort == "" {
		config.ServerPort = "8080" // Default port
	}
	if config.Nacos.Scheme == "" {
		config.Nacos.Scheme = "http"
	}
	if config.Nacos.ContextPath == "" {
		config.Nacos.ContextPath = "/nacos"
	}
	if config.Nacos.TimeoutMs == 0 {
		config.Nacos.TimeoutMs = 5000
	}


	AppConfig = config
	return
}

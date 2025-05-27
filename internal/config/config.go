package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application.
type Config struct {
	ServerPort   string        `mapstructure:"SERVER_PORT"`
	Database     DatabaseConfig `mapstructure:"DATABASE"`
	Nacos        NacosConfig    `mapstructure:"NACOS"`
	FeishuWebook string        `mapstructure:"FEISHU_WEBHOOK_URL"`
}

// DatabaseConfig holds database connection information.
type DatabaseConfig struct {
	DSN string `mapstructure:"DSN"` // Data Source Name e.g., "user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
}

// NacosConfig holds Nacos server connection information.
// These are initial values; instances will be managed via API.
type NacosConfig struct {
	ServerAddr string `mapstructure:"SERVER_ADDR"` // Comma separated, e.g., "127.0.0.1:8848,127.0.0.2:8848"
	NamespaceID string `mapstructure:"NAMESPACE_ID"` // Default namespace ID
	Username    string `mapstructure:"USERNAME"`    // Optional
	Password    string `mapstructure:"PASSWORD"`    // Optional
	Scheme      string `mapstructure:"SCHEME"`      // http or https, default http
	ContextPath string `mapstructure:"CONTEXT_PATH"` // default /nacos
	TimeoutMs   uint64 `mapstructure:"TIMEOUT_MS"`   // default 10000
}

var AppConfig Config

// LoadConfig loads configuration from file and environment variables.
func LoadConfig(configPath ...string) (*Config, error) {
	v := viper.New()

	// Set default values
	v.SetDefault("SERVER_PORT", "8080")
	v.SetDefault("DATABASE.DSN", "user:pass@tcp(127.0.0.1:3306)/nacos_tool?charset=utf8mb4&parseTime=True&loc=Local")
	v.SetDefault("NACOS.SERVER_ADDR", "127.0.0.1:8848")
	v.SetDefault("NACOS.NAMESPACE_ID", "") // Default public namespace
	v.SetDefault("NACOS.SCHEME", "http")
	v.SetDefault("NACOS.CONTEXT_PATH", "/nacos")
	v.SetDefault("NACOS.TIMEOUT_MS", 10000)
	v.SetDefault("FEISHU_WEBHOOK_URL", "")

	// Load from config file
	if len(configPath) > 0 {
		v.SetConfigFile(configPath[0])
		if err := v.ReadInConfig(); err != nil {
			log.Printf("Warning: failed to read config file: %s, using defaults and environment variables", err)
		}
	} else {
		v.SetConfigName("config") // name of config file (without extension)
		v.SetConfigType("yaml")  // REQUIRED if the config file does not have the extension in the name
		v.AddConfigPath("./")    // path to look for the config file in
		v.AddConfigPath("./config") // call multiple times to add many search paths
		v.AddConfigPath("/etc/nacos-config-tool/") // path to look for the config file in
		v.AddConfigPath("$HOME/.nacos-config-tool") // call multiple times to add many search paths
		if err := v.ReadInConfig(); err != nil {
			log.Printf("Warning: failed to read config file: %s, using defaults and environment variables", err)
		}
	}

	// Load from environment variables
	v.SetEnvPrefix("NCT") // NCT_SERVER_PORT, NCT_DATABASE_DSN
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))


	if err := v.Unmarshal(&AppConfig); err != nil {
		return nil, err
	}

	return &AppConfig, nil
}

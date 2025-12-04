package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	GCP      GCPConfig      `mapstructure:"gcp"`
	Browser  BrowserConfig  `mapstructure:"browser"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port            int    `mapstructure:"port"`
	Host            string `mapstructure:"host"`
	ReadTimeout     int    `mapstructure:"read_timeout"`
	WriteTimeout    int    `mapstructure:"write_timeout"`
	ShutdownTimeout int    `mapstructure:"shutdown_timeout"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	Database        string `mapstructure:"database"`
	SSLMode         string `mapstructure:"ssl_mode"`
	MaxConnections  int    `mapstructure:"max_connections"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
}

// DSN returns the database connection string
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode)
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// Address returns the Redis address
func (c *RedisConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// GCPConfig holds Google Cloud Platform configuration
type GCPConfig struct {
	ProjectID string `mapstructure:"project_id"`
	Location  string `mapstructure:"location"`

	// Pub/Sub
	PubSubEnabled      bool   `mapstructure:"pubsub_enabled"`
	PubSubTopic        string `mapstructure:"pubsub_topic"`
	PubSubSubscription string `mapstructure:"pubsub_subscription"`

	// Cloud Storage
	StorageBucket string `mapstructure:"storage_bucket"`

	// Worker URL (for Cloud Tasks if used)
	WorkerURL string `mapstructure:"worker_url"`
}

// BrowserConfig holds browser pool configuration
type BrowserConfig struct {
	PoolSize        int  `mapstructure:"pool_size"`
	Headless        bool `mapstructure:"headless"`
	Timeout         int  `mapstructure:"timeout"`
	MaxConcurrency  int  `mapstructure:"max_concurrency"`
	ContextLifetime int  `mapstructure:"context_lifetime"`
}

// Load loads configuration from a file
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
	}

	// Read environment variables
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

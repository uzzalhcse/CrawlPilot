package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Browser  BrowserConfig  `mapstructure:"browser"`
	Crawler  CrawlerConfig  `mapstructure:"crawler"`
}

type ServerConfig struct {
	Port            int    `mapstructure:"port"`
	Host            string `mapstructure:"host"`
	ReadTimeout     int    `mapstructure:"read_timeout"`
	WriteTimeout    int    `mapstructure:"write_timeout"`
	ShutdownTimeout int    `mapstructure:"shutdown_timeout"`
}

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

type RedisConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type BrowserConfig struct {
	PoolSize        int         `mapstructure:"pool_size"`
	Headless        bool        `mapstructure:"headless"`
	Timeout         int         `mapstructure:"timeout"`
	MaxConcurrency  int         `mapstructure:"max_concurrency"`
	ContextLifetime int         `mapstructure:"context_lifetime"`
	Proxy           ProxyConfig `mapstructure:"proxy"`
}

type ProxyConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Server   string `mapstructure:"server"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type CrawlerConfig struct {
	MaxDepth           int    `mapstructure:"max_depth"`
	UserAgent          string `mapstructure:"user_agent"`
	RespectRobotsTxt   bool   `mapstructure:"respect_robots_txt"`
	MaxRetries         int    `mapstructure:"max_retries"`
	RetryDelay         int    `mapstructure:"retry_delay"`
	ConcurrentWorkers  int    `mapstructure:"concurrent_workers"`
	QueueCheckInterval int    `mapstructure:"queue_check_interval"`
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode)
}

func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()

	// Set defaults
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.read_timeout", 30)
	viper.SetDefault("server.write_timeout", 30)
	viper.SetDefault("server.shutdown_timeout", 10)

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.database", "crawlify")
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_connections", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", 300)

	// Redis defaults
	viper.SetDefault("redis.enabled", false)
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// Browser defaults
	viper.SetDefault("browser.pool_size", 5)
	viper.SetDefault("browser.headless", false)
	viper.SetDefault("browser.timeout", 30000)
	viper.SetDefault("browser.max_concurrency", 10)
	viper.SetDefault("browser.context_lifetime", 300)

	// Proxy defaults
	viper.SetDefault("browser.proxy.enabled", false)
	viper.SetDefault("browser.proxy.server", "")
	viper.SetDefault("browser.proxy.username", "")
	viper.SetDefault("browser.proxy.password", "")

	// Crawler defaults
	viper.SetDefault("crawler.max_depth", 3)
	viper.SetDefault("crawler.user_agent", "Crawlify/1.0")
	viper.SetDefault("crawler.respect_robots_txt", true)
	viper.SetDefault("crawler.max_retries", 3)
	viper.SetDefault("crawler.retry_delay", 1000)
	viper.SetDefault("crawler.concurrent_workers", 5)
	viper.SetDefault("crawler.queue_check_interval", 1000)
}

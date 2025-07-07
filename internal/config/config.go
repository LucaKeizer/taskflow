package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the TaskFlow application
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Redis    RedisConfig    `yaml:"redis"`
	Database DatabaseConfig `yaml:"database"`
	Worker   WorkerConfig   `yaml:"worker"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Addr         string        `yaml:"addr"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// DatabaseConfig holds PostgreSQL connection configuration
type DatabaseConfig struct {
	URL             string        `yaml:"url"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// WorkerConfig holds worker-specific configuration
type WorkerConfig struct {
	Count        int           `yaml:"count"`
	PollInterval time.Duration `yaml:"poll_interval"`
	Timeout      time.Duration `yaml:"timeout"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"` // "json" or "text"
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Addr:         getEnv("SERVER_ADDR", ":8080"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getIntEnv("REDIS_DB", 0),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "postgres://taskflow:taskflow@localhost/taskflow?sslmode=disable"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Worker: WorkerConfig{
			Count:        getIntEnv("WORKER_COUNT", 3),
			PollInterval: getDurationEnv("WORKER_POLL_INTERVAL", 5*time.Second),
			Timeout:      getDurationEnv("WORKER_TIMEOUT", 30*time.Second),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate server configuration
	if c.Server.Addr == "" {
		return fmt.Errorf("server address cannot be empty")
	}

	// Validate Redis configuration
	if c.Redis.Addr == "" {
		return fmt.Errorf("redis address cannot be empty")
	}

	if c.Redis.DB < 0 || c.Redis.DB > 15 {
		return fmt.Errorf("redis DB must be between 0 and 15")
	}

	// Validate database configuration
	if c.Database.URL == "" {
		return fmt.Errorf("database URL cannot be empty")
	}

	if c.Database.MaxOpenConns < 1 {
		return fmt.Errorf("max open connections must be at least 1")
	}

	if c.Database.MaxIdleConns < 0 {
		return fmt.Errorf("max idle connections cannot be negative")
	}

	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		return fmt.Errorf("max idle connections cannot exceed max open connections")
	}

	// Validate worker configuration
	if c.Worker.Count < 1 {
		return fmt.Errorf("worker count must be at least 1")
	}

	if c.Worker.Count > 100 {
		return fmt.Errorf("worker count cannot exceed 100")
	}

	// Validate logging configuration
	validLogLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLogLevels, strings.ToLower(c.Logging.Level)) {
		return fmt.Errorf("invalid log level: %s (valid: %v)", c.Logging.Level, validLogLevels)
	}

	validLogFormats := []string{"json", "text"}
	if !contains(validLogFormats, strings.ToLower(c.Logging.Format)) {
		return fmt.Errorf("invalid log format: %s (valid: %v)", c.Logging.Format, validLogFormats)
	}

	return nil
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

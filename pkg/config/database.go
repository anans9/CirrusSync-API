package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// DatabaseConfig holds all database configuration settings
type DatabaseConfig struct {
	// Connection parameters
	Username string
	Password string
	Host     string
	Port     string
	Name     string
	SSLMode  string

	// Connection pool settings
	PoolMinSize     int
	PoolMaxSize     int
	ConnectTimeout  time.Duration
	MaxIdleTime     time.Duration
	MaxLifetime     time.Duration
	CommandTimeout  time.Duration
	PrepareCached   bool
	MigrateOnBoot   bool
	DefaultTimeZone string
}

// GetDatabaseURL returns a formatted connection string for PostgreSQL
func (c *DatabaseConfig) GetDatabaseURL() string {
	// Format: postgresql://username:password@host:port/dbname?sslmode=disable&TimeZone=UTC
	baseURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		c.Username, c.Password, c.Host, c.Port, c.Name)

	// Add query parameters
	params := fmt.Sprintf("?sslmode=%s&TimeZone=%s",
		c.SSLMode, c.DefaultTimeZone)

	return baseURL + params
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// LoadDatabaseConfig loads database configuration from environment variables
func LoadDatabaseConfig() *DatabaseConfig {
	config := &DatabaseConfig{
		// Connection parameters with defaults
		Username: getEnv("DB_USERNAME", "sury"),
		Password: getEnv("DB_PASSWORD", "1234"),
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		Name:     getEnv("DB_NAME", "cirrussync"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),

		// Connection pool with defaults
		PoolMinSize:     getEnvAsInt("DB_POOL_MIN_SIZE", 5),
		PoolMaxSize:     getEnvAsInt("DB_POOL_MAX_SIZE", 20),
		ConnectTimeout:  getEnvAsDuration("DB_CONNECT_TIMEOUT", 10*time.Second),
		MaxIdleTime:     getEnvAsDuration("DB_MAX_IDLE_TIME", 30*time.Minute),
		MaxLifetime:     getEnvAsDuration("DB_MAX_LIFETIME", 1*time.Hour),
		CommandTimeout:  getEnvAsDuration("DB_COMMAND_TIMEOUT", 30*time.Second),
		DefaultTimeZone: getEnv("DB_TIMEZONE", "UTC"),

		// Features
		PrepareCached: getEnvAsBool("DB_PREPARE_CACHED", true),
		MigrateOnBoot: getEnvAsBool("DB_MIGRATE_ON_BOOT", true),
	}

	return config
}

// Helper to get environment variables as int
func getEnvAsInt(key string, defaultVal int) int {
	if val, exists := os.LookupEnv(key); exists {
		intVal, err := strconv.Atoi(val)
		if err == nil {
			return intVal
		}
	}
	return defaultVal
}

// Helper to get environment variables as duration
func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	if val, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(val); err == nil {
			return time.Duration(intVal) * time.Second
		}
	}
	return defaultVal
}

// Helper to get environment variables as boolean
func getEnvAsBool(key string, defaultVal bool) bool {
	if val, exists := os.LookupEnv(key); exists {
		boolVal, err := strconv.ParseBool(val)
		if err == nil {
			return boolVal
		}
	}
	return defaultVal
}

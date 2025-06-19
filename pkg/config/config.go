package config

import (
	"cirrussync-api/pkg/redis"
	"log"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

// AppConfig holds all configuration settings for the application
type AppConfig struct {
	// Server settings
	Port            string
	Host            string
	Environment     string
	RequestTimeout  int
	ShutdownTimeout int

	// Mail settings (from mail.go)
	Mail *MailConfig

	// TOTP settings (from totp.go)
	TOTP *TOTPConfig

	// Database settings (from database.go)
	Database *DatabaseConfig

	// Redis settings (from redis.go)
	Redis *redis.Config

	// S3 settings (from s3.go)
	S3 *S3Config
}

var (
	appConfig *AppConfig
	once      sync.Once
)

// LoadConfig loads all configuration from environment variables
func LoadConfig() *AppConfig {
	once.Do(func() {
		// Load environment variables from .env file if it exists
		loadEnvFile()

		appConfig = &AppConfig{
			// Server settings
			Port:            getEnv("PORT", "8000"),
			Host:            getEnv("HOST", "localhost"),
			Environment:     getEnv("ENVIRONMENT", "development"),
			RequestTimeout:  getEnvAsInt("REQUEST_TIMEOUT", 30),
			ShutdownTimeout: getEnvAsInt("SHUTDOWN_TIMEOUT", 10),

			// Load database and redis configurations
			Database: LoadDatabaseConfig(),
			Redis:    LoadRedisConfig(),
			S3:       LoadS3Config(),
			Mail:     LoadMailConfig(),
			TOTP:     LoadTOTPConfig(),
		}
	})

	return appConfig
}

// GetConfig returns the already loaded configuration
// Panics if config not yet loaded
func GetConfig() *AppConfig {
	if appConfig == nil {
		panic("Configuration not loaded. Call LoadConfig() first")
	}
	return appConfig
}

// IsDevelopment returns true if the app is in development mode
func (c *AppConfig) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if the app is in production mode
func (c *AppConfig) IsProduction() bool {
	return c.Environment == "production"
}

// IsTest returns true if the app is in test mode
func (c *AppConfig) IsTest() bool {
	return c.Environment == "test"
}

// loadEnvFile tries to load environment variables from .env file
func loadEnvFile() {
	// Try to load environment from .env file (prioritize based on environment)
	envFiles := []string{
		".env." + os.Getenv("ENVIRONMENT") + ".local", // .env.development.local
		".env.local",                       // .env.local
		".env." + os.Getenv("ENVIRONMENT"), // .env.development
		".env",                             // .env
	}

	for _, file := range envFiles {
		if _, err := os.Stat(file); err == nil {
			err = godotenv.Load(file)
			if err == nil {
				log.Printf("Loaded environment from %s", file)
				break
			}
		}
	}
}

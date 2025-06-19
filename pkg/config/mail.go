package config

import (
	"time"
)

// S3Config holds configuration for the S3 client
type MailConfig struct {
	// Email configuration
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	BaseURL      string // Base URL for verification links
	TokenExpiry  time.Duration
}

// LoadS3Config loads S3 configuration from environment variables
func LoadMailConfig() *MailConfig {
	config := &MailConfig{
		SMTPHost:     getEnv("SMTP_HOST", "smtp.mail.me.com"),
		SMTPPort:     getEnvAsInt("SMTP_PORT", 587),
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		FromEmail:    getEnv("SMTP_FROM_EMAIL", "no-reply@cirrussync.me"),
		BaseURL:      getEnv("MAIL_BASE_URL", "https://cirrussync.me"),
		TokenExpiry:  10 * time.Minute,
	}

	return config
}

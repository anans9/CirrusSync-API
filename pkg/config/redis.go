package config

import (
	"os"
	"strconv"
	"time"

	redis "cirrussync-api/pkg/redis"
)

// LoadRedisConfig loads Redis configuration from environment variables
func LoadRedisConfig() *redis.Config {
	config := redis.DefaultConfig()

	// Override with environment variables if provided
	if host := os.Getenv("REDIS_HOST"); host != "" {
		config.Host = host
	}

	if port := os.Getenv("REDIS_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil && p > 0 {
			config.Port = p
		}
	}

	if db := os.Getenv("REDIS_DB"); db != "" {
		if d, err := strconv.Atoi(db); err == nil && d >= 0 {
			config.DB = d
		}
	}

	if password := os.Getenv("REDIS_PASSWORD"); password != "" {
		config.Password = password
	}

	if maxConns := os.Getenv("REDIS_MAX_CONNECTIONS"); maxConns != "" {
		if mc, err := strconv.Atoi(maxConns); err == nil && mc > 0 {
			config.MaxConnections = mc
		}
	}

	if connTimeout := os.Getenv("REDIS_CONN_TIMEOUT"); connTimeout != "" {
		if ct, err := strconv.ParseInt(connTimeout, 10, 64); err == nil && ct > 0 {
			config.ConnTimeout = time.Duration(ct) * time.Second
		}
	}

	if readTimeout := os.Getenv("REDIS_READ_TIMEOUT"); readTimeout != "" {
		if rt, err := strconv.ParseInt(readTimeout, 10, 64); err == nil && rt > 0 {
			config.ReadTimeout = time.Duration(rt) * time.Second
		}
	}

	if writeTimeout := os.Getenv("REDIS_WRITE_TIMEOUT"); writeTimeout != "" {
		if wt, err := strconv.ParseInt(writeTimeout, 10, 64); err == nil && wt > 0 {
			config.WriteTimeout = time.Duration(wt) * time.Second
		}
	}

	return config
}

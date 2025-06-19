package db

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	config "cirrussync-api/pkg/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	// DB is the global database connection
	DB *gorm.DB

	// once ensures the database is initialized only once
	once sync.Once

	// dbConfig stores the database configuration
	dbConfig *config.DatabaseConfig
)

// Initialize sets up the database connection with connection pooling
func Initialize(cfg *config.DatabaseConfig) error {
	var err error

	once.Do(func() {
		dbConfig = cfg
		DB, err = connect(cfg)
	})

	return err
}

// connect creates a new database connection with optimized settings
func connect(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	// Configure GORM logger based on environment
	logLevel := logger.Info
	if cfg.MigrateOnBoot {
		// Use higher log level during migrations
		logLevel = logger.Warn
	}

	// Configure gorm logger
	gormLogger := logger.New(
		log.New(log.Writer(), "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second, // Log queries slower than 1 second
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	// Connection URL
	dsn := cfg.GetDatabaseURL()

	// Configure connection
	gormConfig := &gorm.Config{
		Logger:                 gormLogger,
		SkipDefaultTransaction: true, // Improves performance
		PrepareStmt:            cfg.PrepareCached,
	}

	// Connect to database
	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(cfg.PoolMinSize)
	sqlDB.SetMaxOpenConns(cfg.PoolMaxSize)
	sqlDB.SetConnMaxIdleTime(cfg.MaxIdleTime)
	sqlDB.SetConnMaxLifetime(cfg.MaxLifetime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	err = sqlDB.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Connected to database %s on %s:%s (pool: %d-%d)",
		cfg.Name, cfg.Host, cfg.Port, cfg.PoolMinSize, cfg.PoolMaxSize)

	return db, nil
}

// GetDB returns the database connection
// Panics if database is not initialized
func GetDB() *gorm.DB {
	if DB == nil {
		panic("Database not initialized. Call Initialize() first")
	}

	return DB
}

// Close closes the database connection
func Close() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	err = sqlDB.Close()
	if err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	log.Println("Database connection closed")
	return nil
}

// Health checks database connection health
func Health() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = sqlDB.PingContext(ctx)
	if err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// Stats returns database connection pool statistics
func Stats() map[string]interface{} {
	stats := make(map[string]interface{})

	if DB == nil {
		stats["initialized"] = false
		return stats
	}

	sqlDB, err := DB.DB()
	if err != nil {
		stats["error"] = err.Error()
		return stats
	}

	stats["initialized"] = true
	stats["open_connections"] = sqlDB.Stats().OpenConnections
	stats["in_use"] = sqlDB.Stats().InUse
	stats["idle"] = sqlDB.Stats().Idle
	stats["max_open_connections"] = dbConfig.PoolMaxSize
	stats["max_idle_connections"] = dbConfig.PoolMinSize
	stats["name"] = dbConfig.Name
	stats["host"] = dbConfig.Host
	stats["port"] = dbConfig.Port

	return stats
}

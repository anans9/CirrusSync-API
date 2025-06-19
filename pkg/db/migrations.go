package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"gorm.io/gorm"
)

// MigrationConfig holds configuration for database migrations
type MigrationConfig struct {
	// Path to migration files
	MigrationsPath string

	// Whether to force version (bypassing dirty state checks)
	ForceVersion bool

	// Whether to auto-migrate GORM models (alternative to SQL migrations)
	AutoMigrateModels bool

	// Target version (0 means latest)
	TargetVersion uint
}

// NewMigrationConfig creates a new migration configuration with default values
func NewMigrationConfig() *MigrationConfig {
	return &MigrationConfig{
		MigrationsPath:    "migrations",
		ForceVersion:      false,
		AutoMigrateModels: false,
		TargetVersion:     0, // Latest
	}
}

// RunMigrations runs database migrations
func RunMigrations(cfg *MigrationConfig, models ...interface{}) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	// GORM auto-migration (development/simple cases)
	if cfg.AutoMigrateModels && len(models) > 0 {
		log.Println("Running GORM auto-migrations...")

		start := time.Now()
		err := DB.AutoMigrate(models...)
		if err != nil {
			return fmt.Errorf("auto-migration failed: %w", err)
		}

		log.Printf("Auto-migration completed in %v", time.Since(start))
		return nil
	}

	// Use golang-migrate for production-ready migrations
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	// Setup postgres driver
	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", cfg.MigrationsPath),
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	// Handle dirty state if force version is enabled
	if cfg.ForceVersion {
		log.Println("Warning: Force version is enabled - resetting dirty state if needed")
		if err := m.Force(int(cfg.TargetVersion)); err != nil {
			return fmt.Errorf("failed to force version: %w", err)
		}
	}

	// Run migrations
	if cfg.TargetVersion > 0 {
		log.Printf("Migrating to version %d...", cfg.TargetVersion)
		if err := m.Migrate(cfg.TargetVersion); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("migration failed: %w", err)
		}
	} else {
		log.Println("Migrating to latest version...")
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	// Get current version
	version, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	log.Printf("Migration completed successfully. Current version: %d, Dirty: %v", version, dirty)
	return nil
}

// SeedDatabase seeds the database with initial data
func SeedDatabase(ctx context.Context, seedFn func(tx *gorm.DB) error) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	log.Println("Seeding database...")
	start := time.Now()

	// Run seeding in a transaction
	err := WithTransaction(ctx, seedFn)
	if err != nil {
		return fmt.Errorf("database seeding failed: %w", err)
	}

	log.Printf("Database seeding completed in %v", time.Since(start))
	return nil
}

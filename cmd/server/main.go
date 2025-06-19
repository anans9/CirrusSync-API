package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cirrussync-api/internal/models"
	"cirrussync-api/pkg/config"
	"cirrussync-api/pkg/db"
	"cirrussync-api/pkg/redis"
	"cirrussync-api/pkg/s3"
	"cirrussync-api/router"
)

func main() {
	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	setupGracefulShutdown(cancel)

	// Load configuration
	log.Println("Loading configuration...")
	appConfig := config.LoadConfig()

	// Initialize database
	log.Println("Initializing database connection...")
	err := db.Initialize(appConfig.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Run migrations if enabled
	if appConfig.Database.MigrateOnBoot {
		log.Println("Running database migrations...")
		migrationCfg := db.NewMigrationConfig()

		// Adjust migration settings based on environment
		if appConfig.IsDevelopment() {
			migrationCfg.AutoMigrateModels = true

			// Auto-migrate models instead of SQL migrations in development
			err = db.RunMigrations(migrationCfg, &models.User{},
				&models.UserSRP{},
				&models.UserCredit{},
				&models.UserSecurityEvent{},
				&models.SecurityEventDownload{},
				&models.UserSecuritySettings{},
				&models.UserKey{},
				&models.UserRecoveryKit{},
				&models.UserSession{},
				&models.VolumeAllocation{},
				&models.UserStorage{},
				&models.UserDevice{},
				&models.EmailMethods{},
				&models.PhoneMethods{},
				&models.TOTPMethods{},
				&models.UserMFASettings{},
				&models.UserNotifications{},
				&models.UserPreferences{},

				// Billing models
				&models.BillingInfo{},
				&models.Plan{},
				&models.UserPlan{},
				&models.UserBilling{},
				&models.UserPaymentMethod{},
				&models.GiftCard{},

				// Drive models
				&models.DriveVolume{},
				&models.DriveShare{},
				&models.DriveShareMembership{},
				&models.DriveItem{},
				&models.FileRevision{},
				&models.DriveThumbnail{},
				&models.FileBlock{})
		} else {
			// Use SQL migrations in production
			err = db.RunMigrations(migrationCfg)
		}

		if err != nil {
			log.Fatalf("Failed to run database migrations: %v", err)
		}
	}

	// Initialize Redis connection
	log.Println("Initializing Redis connection...")
	redis.InitDefault(appConfig.Redis)
	redisClient := redis.GetDefault()

	err = redisClient.Ping(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis successfully")

	// Initialize S3 client
	s3Err := s3.InitS3(appConfig.S3)
	if s3Err != nil {
		log.Fatalf("Failed to initialize S3 client: %v", err)
	}
	log.Printf("S3 client initialized with bucket: %s", appConfig.S3.BucketName)

	// Initialize HTTP server, background workers, etc.

	// Get the database connection for the router
	database := db.GetDB()

	// Setup the Gin router
	log.Println("Setting up router...")
	ginEngine, _ := router.SetupRouter(database)

	// Create server with Gin handler
	srv := &http.Server{
		Addr:    appConfig.Host + ":" + appConfig.Port,
		Handler: ginEngine,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server started on %s:%s", appConfig.Host, appConfig.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	<-ctx.Done()
	log.Println("Shutting down server...")

	// Create shutdown context with timeout
	shutdownTimeout := time.Duration(appConfig.ShutdownTimeout) * time.Second
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	// Shutdown the server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	// Perform other cleanup
	gracefulShutdown(shutdownTimeout)
}

// setupGracefulShutdown sets up signal handling for graceful shutdown
func setupGracefulShutdown(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("Received shutdown signal")
		cancel()
	}()
}

// gracefulShutdown performs cleanup before exiting
func gracefulShutdown(timeout time.Duration) {
	// Create context with timeout for shutdown operations
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Close database connections
	log.Println("Closing database connections...")
	if err := db.Close(); err != nil {
		log.Printf("Error closing database connection: %v", err)
	}

	// Close Redis connections
	log.Println("Closing Redis connections...")
	redis.CloseAll()

	// Wait for context or proceed if timeout
	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			log.Println("Shutdown timed out, forcing exit")
		}
	case <-time.After(100 * time.Millisecond):
		// Add a small buffer to allow logging before exit
	}

	log.Println("Shutdown complete")
}

package router

import (
	"errors"
	"net/http"
	"os"
	"strconv"
	"time"

	authAPI "cirrussync-api/api/v1/auth"
	csrfAPI "cirrussync-api/api/v1/csrf"
	driveAPI "cirrussync-api/api/v1/drive"
	mfaAPI "cirrussync-api/api/v1/mfa"
	sessionAPI "cirrussync-api/api/v1/sessions"
	userAPI "cirrussync-api/api/v1/users"
	internalAuth "cirrussync-api/internal/auth"
	internalDrive "cirrussync-api/internal/drive"
	jwt "cirrussync-api/internal/jwt"
	log "cirrussync-api/internal/logger"
	"cirrussync-api/internal/mfa"
	internalMfa "cirrussync-api/internal/mfa"
	"cirrussync-api/internal/middleware"
	"cirrussync-api/internal/session"
	srp "cirrussync-api/internal/srp"
	internalUser "cirrussync-api/internal/user"
	"cirrussync-api/pkg/config"
	"cirrussync-api/pkg/db"
	"cirrussync-api/pkg/redis"

	"github.com/getsentry/sentry-go"
	sentrylogrus "github.com/getsentry/sentry-go/logrus"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Package-level services to avoid recreation
var (
	jwtService     *jwt.JWTService
	sessionService *session.Service
	userService    *internalUser.Service
	authService    *internalAuth.Service
	driveService   *internalDrive.Service
	logger         *logrus.Logger
	customLogger   *log.Logger
)

// InitServices initializes all required services
func InitServices(database *gorm.DB, redisClient *redis.Client) error {
	// Initialize logger with Sentry hook
	logger = logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Setup Sentry hook for logrus if DSN is provided
	sentryDSN := os.Getenv("SENTRY_DSN")
	if sentryDSN != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:         sentryDSN,
			Environment: os.Getenv("APP_ENV"),
			Release:     os.Getenv("APP_VERSION"),
		})
		if err != nil {
			return errors.New("failed to initialize Sentry: " + err.Error())
		}

		// Add Sentry hook to logrus
		levels := []logrus.Level{logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel}
		hook, err := sentrylogrus.New(levels, sentry.ClientOptions{
			Dsn: sentryDSN,
		})
		if err != nil {
			logger.WithError(err).Error("Failed to initialize Sentry hook")
		} else {
			logger.AddHook(hook)
			logger.Info("Sentry integration initialized successfully")
		}
	}

	// Initialize custom logger wrapper
	customLogger = log.New(logger)

	// Initialize JWT service
	var err error
	jwtService, err = jwt.NewJWTService(
		"./keys/private.pem",
		"./keys/public.pem",
		"app.cirrussync.me",
		1*time.Hour,
		24*time.Hour,
	)
	if err != nil {
		logger.WithError(err).Error("Failed to initialize JWT service")
		return err
	}

	// Initialize Drive service
	driveRepo := internalDrive.NewRepository(database)
	driveService = internalDrive.NewService(driveRepo, redisClient, customLogger)

	// Initialize user repository and service
	userRepo := internalUser.NewRepository(database)
	userService = internalUser.NewService(userRepo, redisClient, driveService)

	// Initialize session repository and service
	sessionRepo := session.NewRepository(database)
	sessionService = session.NewService(sessionRepo, redisClient, customLogger)

	// Initialize SRP repository
	srpRepo := srp.NewRepository(database)

	// Initialize Auth service with all dependencies
	authService = internalAuth.NewService(redisClient, customLogger, srpRepo, userService)

	logger.Info("All services initialized successfully")
	return nil
}

// CSRFMiddleware creates a middleware for CSRF protection
func CSRFMiddleware(secret string, secure bool) gin.HandlerFunc {
	csrfMiddleware := csrf.Protect(
		[]byte(secret),
		csrf.Secure(secure),
		csrf.HttpOnly(true),
		csrf.Path("/"),
		csrf.CookieName("csrfToken"),
		csrf.MaxAge(3600), // 1 hour
		csrf.SameSite(csrf.SameSiteStrictMode),
		csrf.Domain("localhost"),
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Ensure CORS headers are set even for CSRF errors
			c, _ := gin.CreateTestContext(w)
			c.Request = r

			// Log CSRF error for monitoring
			logger.WithFields(logrus.Fields{
				"remoteIP":  c.ClientIP(),
				"path":      r.URL.Path,
				"method":    r.Method,
				"userAgent": r.UserAgent(),
			}).Error("CSRF token mismatch")

			c.IndentedJSON(http.StatusForbidden, gin.H{"error": "CSRF token mismatch"})
			c.Abort()
		})),
	)

	return func(c *gin.Context) {
		csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Request = r
			c.Next()
		})).ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}

// SetupEngine creates a new Gin engine with default middleware
func SetupEngine() *gin.Engine {
	return gin.Default()
}

// SetupCsrfRoutes configures CSRF-related routes
func SetupCsrfRoutes(r *gin.Engine) {
	// Create API v1 group
	v1 := r.Group("/api/v1")

	// Create csrf handler
	csrfHandler := csrfAPI.NewHandler(customLogger)

	csrfAPI.RegisterPublicRoutes(v1, csrfHandler)
}

// SetupMFARoutes configures MFA-related routes
func SetupMFARoutes(r *gin.Engine, database *gorm.DB) {
	// Create API v1 group
	v1 := r.Group("/api/v1")

	// Configure MFA service
	mailConfig := config.LoadMailConfig()
	totpConfig := config.LoadTOTPConfig()

	MFAConfig := internalMfa.MFAConfig{
		MailConfig: *mailConfig,
		TOTPConfig: *totpConfig,
	}

	// Get Redis client
	redisClient := redis.GetDefault()

	mfaRepo := internalMfa.NewRepository(database)

	// Create MFA service
	mfaService := mfa.NewService(mfaRepo, MFAConfig, redisClient, customLogger)

	// Create MFA handler
	mfaHandler := mfaAPI.NewHandler(mfaService, customLogger)

	// Register routes
	mfaAPI.RegisterProtectedRoutes(v1, mfaHandler)
}

// SetupAuthRoutes configures auth-related routes
func SetupAuthRoutes(r *gin.Engine) {
	// Create API v1 group
	v1 := r.Group("/api/v1")

	// Create auth handler using the global services
	authHandler := authAPI.NewHandler(authService, userService, jwtService, sessionService, customLogger)

	// Register public auth routes
	authAPI.RegisterPublicRoutes(v1, authHandler)

	// Create authenticated route group
	authGroup := v1.Group("/auth")
	authGroup.Use(middleware.JWTAuthMiddleware(jwtService, sessionService))
	authAPI.RegisterProtectedRoutes(authGroup, authHandler)
}

// SetupSessionsRoutes configures user-related routes
func SetupSessionsRoutes(r *gin.Engine) {
	// Create API v1 group
	v1 := r.Group("/api/v1")

	// Create user handler using the global service
	userHandler := userAPI.NewHandler(userService, customLogger)

	// Create user route group with auth middleware
	userGroup := v1.Group("/users")
	userGroup.Use(middleware.JWTAuthMiddleware(jwtService, sessionService))
	userAPI.RegisterProtectedRoutes(userGroup, userHandler)
}

// SetupUserRoutes configures user-related routes
func SetupUsersRoutes(r *gin.Engine) {
	// Create API v1 group
	v1 := r.Group("/api/v1")

	// Create user handler using the global service
	sessionHandler := sessionAPI.NewHandler(sessionService, customLogger)

	// Create session route group with auth middleware
	sessionGroup := v1.Group("/sessions")
	sessionGroup.Use(middleware.JWTAuthMiddleware(jwtService, sessionService))
	sessionAPI.RegisterProtectedRoutes(sessionGroup, sessionHandler)
}

// SetupUserRoutes configures user-related routes
func SetupDriveRoutes(r *gin.Engine, database *gorm.DB) {
	// Create API v1 group
	v1 := r.Group("/api/v1")

	// Create drive handler using the global service
	driveHandler := driveAPI.NewHandler(driveService, userService, customLogger)

	// Create drive route group with auth middleware
	driveGroup := v1.Group("/drive")
	driveGroup.Use(middleware.JWTAuthMiddleware(jwtService, sessionService))
	driveAPI.RegisterProtectedRoutes(driveGroup, driveHandler)
}

// SetupCSRFProtection configures CSRF protection
func SetupCSRFProtection(r *gin.Engine) error {
	csrfSecret := os.Getenv("CSRF_SECRET")
	if csrfSecret == "" {
		logger.Error("CSRF_SECRET environment variable is required")
		return errors.New("CSRF_SECRET environment variable is required")
	}

	csrfSecureStr := os.Getenv("CSRF_SECURE")
	csrfSecure, _ := strconv.ParseBool(csrfSecureStr)

	r.Use(CSRFMiddleware(csrfSecret, csrfSecure))

	return nil
}

// SetupCORS configures CORS settings
func SetupCORS(r *gin.Engine) {
	// Trusted Proxies
	r.SetTrustedProxies([]string{"http://localhost:1420"})

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"http://localhost:1420"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-CSRF-TOKEN", "X-App-Version", "X-Client-UID", "X-Client-Name"}
	corsConfig.AllowCredentials = true
	corsConfig.MaxAge = 24 * time.Hour

	r.Use(cors.New(corsConfig))
}

// SetupRouter creates and configures the main router with all routes
func SetupRouter(database *gorm.DB) (*gin.Engine, error) {
	// Set global database reference
	db.DB = database

	// Get Redis client
	redisClient := redis.GetDefault()

	// Initialize all services
	if err := InitServices(database, redisClient); err != nil {
		// This error is already logged in InitServices
		return nil, err
	}

	// Create and configure Gin router
	r := SetupEngine()

	// Setup CORS
	SetupCORS(r)

	// Setup CSRF protection
	if err := SetupCSRFProtection(r); err != nil {
		logger.WithError(err).Error("Failed to setup CSRF protection")
		return nil, err
	}

	// Configure routes
	SetupCsrfRoutes(r)
	SetupAuthRoutes(r)
	SetupUsersRoutes(r)
	SetupSessionsRoutes(r)
	SetupMFARoutes(r, database)
	SetupDriveRoutes(r, database)

	logger.Info("Router setup completed successfully")
	return r, nil
}

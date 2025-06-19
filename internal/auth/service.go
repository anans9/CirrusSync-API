package auth

import (
	"cirrussync-api/internal/logger"
	"cirrussync-api/internal/models"
	"cirrussync-api/internal/srp"
	"cirrussync-api/internal/user"
	"cirrussync-api/pkg/redis"
	"context"
)

// Service handles authentication operations
type Service struct {
	srpService  *srp.Service
	userService *user.Service
	logger      *logger.Logger
}

// NewService creates a new auth service
func NewService(
	redisClient *redis.Client,
	logger *logger.Logger,
	srpRepo srp.Repository,
	userService *user.Service,
) *Service {
	// Create SRP service
	srpService := srp.NewService(srpRepo, redisClient, logger)

	return &Service{
		srpService:  srpService,
		userService: userService,
		logger:      logger,
	}
}

// CreateUser delegates user creation to the user service
func (s *Service) CreateUser(ctx context.Context, email, username string, key user.UserKey) (*models.User, error) {
	// Delegate to user service
	return s.userService.CreateUser(ctx, email, username, key)
}

// LoginInit initiates SRP authentication
func (s *Service) LoginInit(ctx context.Context, email, clientPublic, ipAddress, userAgent string) (*srp.InitResponse, error) {
	return s.srpService.InitAuthentication(ctx, email, clientPublic, ipAddress, userAgent)
}

// LoginVerify verifies SRP proof and completes authentication
func (s *Service) LoginVerify(ctx context.Context, sessionID, clientProof, ipAddress string) (*srp.VerifyResponse, *models.UserSRP, error) {
	return s.srpService.VerifyAuthentication(ctx, sessionID, clientProof, ipAddress)
}

// RegisterSRP registers SRP credentials for a user
func (s *Service) RegisterSRP(ctx context.Context, userID string, email string, salt string, verifier string) error {
	return s.srpService.RegisterSRPCredentials(ctx, userID, email, salt, verifier)
}

// ChangePassword changes a user's password using client-generated credentials
func (s *Service) ChangePassword(ctx context.Context, userID, email, oldSalt, oldVerifier, newSalt, newVerifier string) error {
	// Verify old credentials by checking if they match stored credentials
	userSRP, err := s.srpService.GetUserSRPByID(userID)
	if err != nil {
		return err
	}

	// Critical security check: Ensure provided old salt matches stored salt
	// This is to prevent attackers from bypassing verification
	if userSRP.Salt != oldSalt {
		return ErrInvalidCredentials
	}

	// Verify old verifier
	if userSRP.Verifier != oldVerifier {
		return ErrInvalidCredentials
	}

	// Store new SRP credentials
	return s.srpService.RegisterSRPCredentials(ctx, userID, email, newSalt, newVerifier)
}

package session

import (
	"cirrussync-api/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// redisKeyForSession generates a Redis key for a session
func redisKeyForSession(sessionID string) string {
	return fmt.Sprintf("session:%s", sessionID)
}

// redisKeyForUserSessions generates a Redis key for a user's sessions list
func redisKeyForUserSessions(userID string) string {
	return fmt.Sprintf("user:%s:sessions", userID)
}

// cacheSession saves a session to Redis cache
func (s *Service) cacheSession(ctx context.Context, session *models.UserSession) error {
	// Marshal session to JSON
	sessionJSON, err := json.Marshal(session)
	if err != nil {
		s.logger.Error("Failed to marshal session for caching", "error", err)
		return ErrCacheError
	}

	// Cache with an hour expiration
	sessionKey := redisKeyForSession(session.ID)
	err = s.redisClient.Set(ctx, sessionKey, string(sessionJSON), time.Hour)
	if err != nil {
		s.logger.Error("Failed to cache session", "error", err)
		return ErrCacheError
	}

	// Add to the user's sessions set
	userSessionsKey := redisKeyForUserSessions(session.UserID)
	_, err = s.redisClient.SAdd(ctx, userSessionsKey, session.ID)
	if err != nil {
		s.logger.Warn("Failed to add session to user's sessions set", "error", err)
		// Not returning error as this is not critical
	}

	// Set expiration on the user's sessions set
	_, err = s.redisClient.Expire(ctx, userSessionsKey, time.Hour)
	if err != nil {
		s.logger.Warn("Failed to set expiration on user's sessions set", "error", err)
		// Not returning error as this is not critical
	}

	return nil
}

// getSessionFromCache retrieves a session from Redis cache
func (s *Service) getSessionFromCache(ctx context.Context, sessionID string) (*models.UserSession, error) {
	// Get from cache
	sessionKey := redisKeyForSession(sessionID)
	sessionJSON, err := s.redisClient.Get(ctx, sessionKey)
	if err != nil || sessionJSON == "" {
		return nil, ErrSessionNotFound
	}

	// Unmarshal JSON to session
	var session models.UserSession
	err = json.Unmarshal([]byte(sessionJSON), &session)
	if err != nil {
		s.logger.Error("Failed to unmarshal cached session", "error", err)
		return nil, ErrCacheError
	}

	return &session, nil
}

// invalidateSessionCache removes a session from Redis cache
func (s *Service) invalidateSessionCache(ctx context.Context, sessionID string, userID string) error {
	// Delete session key
	sessionKey := redisKeyForSession(sessionID)
	_, err := s.redisClient.Delete(ctx, sessionKey)
	if err != nil {
		s.logger.Warn("Failed to remove session from cache", "error", err)
		// Not returning error as this is not critical
	}

	// Remove from user's sessions set if userID is provided
	if userID != "" {
		userSessionsKey := redisKeyForUserSessions(userID)
		_, err = s.redisClient.SRem(ctx, userSessionsKey, sessionID)
		if err != nil {
			s.logger.Warn("Failed to remove session from user's sessions set", "error", err)
			// Not returning error as this is not critical
		}
	}

	return nil
}

// invalidateUserSessionsCache removes all of a user's sessions from Redis cache
func (s *Service) invalidateUserSessionsCache(ctx context.Context, userID string) error {
	// Get all session IDs for the user
	userSessionsKey := redisKeyForUserSessions(userID)
	sessionIDs, err := s.redisClient.SMembers(ctx, userSessionsKey)
	if err != nil {
		s.logger.Warn("Failed to get user's sessions from cache", "error", err)
		return nil // Not returning error as this is not critical
	}

	// Delete each session key
	for _, sessionID := range sessionIDs {
		sessionKey := redisKeyForSession(sessionID)
		_, err := s.redisClient.Delete(ctx, sessionKey)
		if err != nil {
			s.logger.Warn("Failed to remove session from cache", "sessionID", sessionID, "error", err)
			// Continue with other sessions
		}
	}

	// Delete the user's sessions set
	_, err = s.redisClient.Delete(ctx, userSessionsKey)
	if err != nil {
		s.logger.Warn("Failed to remove user's sessions set from cache", "error", err)
		// Not returning error as this is not critical
	}

	return nil
}

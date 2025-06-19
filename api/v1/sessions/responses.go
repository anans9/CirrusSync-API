package session

import (
	"cirrussync-api/internal/models"
	"cirrussync-api/internal/utils"
	"time"
)

// BaseResponse provides the base structure for all API responses
type BaseResponse struct {
	Code   int16  `json:"code"`
	Detail string `json:"detail"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	BaseResponse
	Error string `json:"error,omitempty"`
}

// SuccessResponse represents a simple success message
type SuccessResponse struct {
	BaseResponse
	Message string `json:"message,omitempty"`
}

// SessionData represents session data in the response
type SessionData struct {
	ID         string `json:"id"`
	UserID     string `json:"userId"`
	DeviceName string `json:"deviceName"`
	DeviceID   string `json:"deviceId"`
	AppVersion string `json:"appVersion"`
	IPAddress  string `json:"ipAddress"`
	UserAgent  string `json:"userAgent"`
	ExpiresAt  int64  `json:"expiresAt"`
	CreatedAt  int64  `json:"createdAt"`
	LastActive int64  `json:"lastActive"`
	IsActive   bool   `json:"isActive"`
	IsCurrent  bool   `json:"isCurrent"`
}

// SessionResponse represents a session response
type SessionResponse struct {
	BaseResponse
	Session SessionData `json:"session"`
}

// SessionsListResponse represents a list of sessions response
type SessionsListResponse struct {
	BaseResponse
	Sessions []SessionData `json:"sessions"`
}

// NewErrorResponse creates a new error response
func NewErrorResponse(message string, code int16) ErrorResponse {
	return ErrorResponse{
		BaseResponse: BaseResponse{
			Code:   code,
			Detail: "Error with requestId " + utils.GenerateShortID(),
		},
		Error: message,
	}
}

// NewSuccessResponse creates a new success response
func NewSuccessResponse(message string, code int16) SuccessResponse {
	return SuccessResponse{
		BaseResponse: BaseResponse{
			Code:   code,
			Detail: "Success with requestId " + utils.GenerateShortID(),
		},
		Message: message,
	}
}

// NewSessionResponse creates a new session response
func NewSessionResponse(session *models.UserSession, code int16) SessionResponse {
	return SessionResponse{
		BaseResponse: BaseResponse{
			Code:   code,
			Detail: "Success with requestId " + utils.GenerateShortID(),
		},
		Session: convertModelSessionToResponse(session, session.ID),
	}
}

// NewSessionsListResponse creates a new sessions list response
func NewSessionsListResponse(sessions []*models.UserSession, code int16) SessionsListResponse {
	// Get current session ID from the first session that has IsValid=true
	// This is a placeholder - in real implementation, you'd get this from context
	var currentSessionID string
	for _, s := range sessions {
		if s.IsValid && s.ExpiresAt > time.Now().Unix() {
			currentSessionID = s.ID
			break
		}
	}
	sessionDataList := make([]SessionData, len(sessions))
	for i, session := range sessions {
		sessionDataList[i] = convertModelSessionToResponse(session, currentSessionID)
	}
	return SessionsListResponse{
		BaseResponse: BaseResponse{
			Code:   code,
			Detail: "Success with requestId " + utils.GenerateShortID(),
		},
		Sessions: sessionDataList,
	}
}

// Helper function to convert model session to response session data
func convertModelSessionToResponse(session *models.UserSession, currentSessionID string) SessionData {
	return SessionData{
		ID:         session.ID,
		UserID:     session.UserID,
		DeviceName: session.DeviceName,
		DeviceID:   session.DeviceID,
		AppVersion: session.AppVersion,
		IPAddress:  session.IPAddress,
		UserAgent:  session.UserAgent,
		ExpiresAt:  session.ExpiresAt,
		CreatedAt:  session.CreatedAt,
		LastActive: session.ModifiedAt,
		IsActive:   session.IsValid && session.ExpiresAt > time.Now().Unix(),
		IsCurrent:  session.ID == currentSessionID,
	}
}

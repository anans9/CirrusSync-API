package auth

import (
	"cirrussync-api/internal/jwt"
	"cirrussync-api/internal/models"
	"strings"

	"github.com/go-playground/validator/v10"
)

// User represents a user in the response
type User struct {
	ID string `json:"id"`
}

// Session represents a session in the response
type Session struct {
	ID        string `json:"id"`
	ExpiresAt int64  `json:"expiresAt"`
}

// BaseResponse contains fields common to all responses
type BaseResponse struct {
	Code int16 `json:"code"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	BaseResponse
	Detail string `json:"detail"`
}

// LoginInitResponse represents the response from SRP initialization
type LoginInitResponse struct {
	BaseResponse
	SessionID    string `json:"sessionId"`
	Salt         string `json:"salt"`
	ServerPublic string `json:"serverPublic"`
}

// LoginVerifyResponse represents the response from successful authentication
type LoginVerifyResponse struct {
	BaseResponse
	AccessToken  string   `json:"accessToken"`
	RefreshToken string   `json:"refreshToken"`
	TokenType    string   `json:"tokenType"`
	Scopes       []string `json:"scopes"`
	Session      Session  `json:"session"`
	User         User     `json:"user"`
	ServerProof  string   `json:"serverProof"`
	ExpiresIn    int64    `json:"expiresIn"`
}

// SignupResponse represents the response from successful registration
type SignupResponse struct {
	BaseResponse
	User User `json:"user"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	BaseResponse
	Detail string `json:"detail"`
}

// RefreshTokenResponse represents the response from token refresh
type RefreshTokenResponse struct {
	BaseResponse
	AccessToken  string   `json:"accessToken"`
	RefreshToken string   `json:"refreshToken"`
	TokenType    string   `json:"tokenType"`
	Scopes       []string `json:"scopes"`
	Session      Session  `json:"session"`
	User         User     `json:"user"`
	ExpiresIn    int64    `json:"expiresIn"`
}

// NewValidationError creates a new validation error response
func NewValidationError(err error, code int16) ErrorResponse {
	if errs, ok := err.(validator.ValidationErrors); ok && len(errs) > 0 {
		full := errs[0].Error()
		parts := strings.SplitN(full, "Error:", 2)
		if len(parts) == 2 {
			return NewErrorResponse(strings.TrimSpace(parts[1]), code)
		}
		return NewErrorResponse(full, code)
	}
	return NewErrorResponse("Invalid request format", code)
}

// NewErrorResponse creates a new error response
func NewErrorResponse(message string, code int16) ErrorResponse {
	return ErrorResponse{
		BaseResponse: BaseResponse{Code: code},
		Detail:       message,
	}
}

// NewLoginInitResponse creates a new login initialization response
func NewLoginInitResponse(sessionID, salt, serverPublic string, code int16) LoginInitResponse {
	return LoginInitResponse{
		BaseResponse: BaseResponse{Code: code},
		SessionID:    sessionID,
		Salt:         salt,
		ServerPublic: serverPublic,
	}
}

// NewLoginVerifyResponse creates a new login verification response
func NewLoginVerifyResponse(
	token jwt.TokenPair,
	user *models.User,
	session *models.UserSession,
	serverProof string,
	code int16,
) LoginVerifyResponse {
	return LoginVerifyResponse{
		BaseResponse: BaseResponse{Code: code},
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Scopes:       token.Scopes,
		User: User{
			ID: user.ID,
		},
		Session: Session{
			ID:        session.ID,
			ExpiresAt: session.ExpiresAt,
		},
		ServerProof: serverProof,
		ExpiresIn:   token.ExpiresIn,
	}
}

// NewSignupResponse creates a new signup response
func NewSignupResponse(userID string, code int16) SignupResponse {
	return SignupResponse{
		BaseResponse: BaseResponse{Code: code},
		User: User{
			ID: userID,
		},
	}
}

// NewSuccessResponse creates a new success response
func NewSuccessResponse(message string, code int16) SuccessResponse {
	return SuccessResponse{
		BaseResponse: BaseResponse{Code: code},
		Detail:       message,
	}
}

// NewRefreshTokenResponse creates a new token refresh response
func NewRefreshTokenResponse(
	token jwt.TokenPair,
	user *models.User,
	session *models.UserSession,
	code int16,
) RefreshTokenResponse {
	return RefreshTokenResponse{
		BaseResponse: BaseResponse{Code: code},
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Scopes:       token.Scopes,
		User: User{
			ID: user.ID,
		},
		Session: Session{
			ID:        session.ID,
			ExpiresAt: session.ExpiresAt,
		},
		ExpiresIn: token.ExpiresIn,
	}
}

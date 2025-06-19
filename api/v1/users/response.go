package user

import (
	"cirrussync-api/internal/user"
	"cirrussync-api/internal/utils"
)

// UserResponse is the complete response structure for user-related API responses
type UserResponse struct {
	BaseResponse
	User user.User `json:"user"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	BaseResponse
	Detail string `json:"detail,omitempty"`
}

// NewSuccessResponse creates a success response with user data
func NewSuccessResponse(message string, user user.User, code int16) UserResponse {
	return UserResponse{
		BaseResponse: BaseResponse{
			Code:   code,
			Detail: "Success with requestId " + utils.GenerateShortID(),
		},
		User: user,
	}
}

// NewErrorResponse creates a new error response
func NewErrorResponse(message string, code int16) ErrorResponse {
	return ErrorResponse{
		BaseResponse: BaseResponse{
			Code:   code,
			Detail: "Error with requestId " + utils.GenerateShortID(),
		},
		Detail: message,
	}
}

// NewValidationError creates a validation error response
func NewValidationError(err error, code int16) ErrorResponse {
	return ErrorResponse{
		BaseResponse: BaseResponse{
			Code:   code,
			Detail: "Validation Error with requestId " + utils.GenerateShortID(),
		},
		Detail: err.Error(),
	}
}

// RegisterResponse represents a response to a successful registration
type RegisterResponse struct {
	BaseResponse
	UserID string `json:"userId"`
}

// NewRegisterResponse creates a new registration response
func NewRegisterResponse(userID string, code int16) RegisterResponse {
	return RegisterResponse{
		BaseResponse: BaseResponse{
			Code:   code,
			Detail: "Registration successful with requestId " + utils.GenerateShortID(),
		},
		UserID: userID,
	}
}

// SimpleResponse represents a simple success/error response without user data
type SimpleResponse struct {
	BaseResponse
	Message string `json:"message,omitempty"`
}

// NewSimpleResponse creates a new simple response
func NewSimpleResponse(message string, code int16) SimpleResponse {
	return SimpleResponse{
		BaseResponse: BaseResponse{
			Code:   code,
			Detail: "Success with requestId " + utils.GenerateShortID(),
		},
		Message: message,
	}
}

// KeyResponse represents a response with key information
type KeyResponse struct {
	BaseResponse
	Key Key `json:"key"`
}

// NewKeyResponse creates a new key response
func NewKeyResponse(key Key, code int16) KeyResponse {
	return KeyResponse{
		BaseResponse: BaseResponse{
			Code:   code,
			Detail: "Success with requestId " + utils.GenerateShortID(),
		},
		Key: key,
	}
}

// PreferencesResponse represents a response with user preferences
type PreferencesResponse struct {
	BaseResponse
	ThemeMode string `json:"themeMode"`
	Language  string `json:"language"`
	Timezone  string `json:"timezone"`
}

// SecuritySettingsResponse represents a response with security settings
type SecuritySettingsResponse struct {
	BaseResponse
	TwoFactorRequired           bool `json:"twoFactorRequired"`
	DarkWebMonitoring           bool `json:"darkWebMonitoring"`
	SuspiciousActivityDetection bool `json:"suspiciousActivityDetection"`
	DetailedEvents              bool `json:"detailedEvents"`
}

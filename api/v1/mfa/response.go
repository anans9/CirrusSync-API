package mfa

import (
	"cirrussync-api/pkg/status"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// BaseResponse represents the base structure for responses
type BaseResponse struct {
	Code    int16  `json:"code"`
	Message string `json:"message"`
}

// ErrorDetails represents error details in a response
type ErrorDetails struct {
	Code    int16  `json:"code"`
	Message string `json:"message"`
}

// GenericResponse represents a generic API response
type GenericResponse struct {
	Data    interface{}   `json:"data,omitempty"`
	Success bool          `json:"success"`
	Error   *ErrorDetails `json:"error,omitempty"`
	Message string        `json:"message,omitempty"`
}

// SendVerificationResponseData represents the data returned from a send verification request
type SendVerificationResponseData struct {
	Success           bool   `json:"success"`
	RemainingRequests int    `json:"remainingRequests,omitempty"`
	ResetTime         string `json:"resetTime,omitempty"`
	NextAllowedTime   string `json:"nextAllowedTime,omitempty"`
}

// VerifyEmailResponseData represents the data returned from a verify email request
type VerifyEmailResponseData struct {
	Verified bool   `json:"verified"`
	Email    string `json:"email,omitempty"`
}

// CheckVerificationResponseData represents the data returned from a check verification request
type CheckVerificationResponseData struct {
	Verified bool   `json:"verified"`
	Email    string `json:"email,omitempty"`
}

// GenerateTOTPResponseData represents the data returned from a generate TOTP request
type GenerateTOTPResponseData struct {
	Secret       string   `json:"secret"`
	QRCodeURL    string   `json:"qrCodeUrl"`
	RecoveryKeys []string `json:"recoveryKeys"`
}

// VerifyTOTPResponseData represents the data returned from a verify TOTP request
type VerifyTOTPResponseData struct {
	Success bool `json:"success"`
}

// ValidateTOTPResponseData represents the data returned from a validate TOTP request
type ValidateTOTPResponseData struct {
	Valid bool `json:"valid"`
}

// DisableTOTPResponseData represents the data returned from a disable TOTP request
type DisableTOTPResponseData struct {
	Success bool `json:"success"`
}

// ErrorResponse sends an error response to the client
func ErrorResponse(c *gin.Context, httpStatus int, apiStatus int16, message string) {
	c.JSON(httpStatus, GenericResponse{
		Success: false,
		Error: &ErrorDetails{
			Code:    apiStatus,
			Message: message,
		},
	})
}

// ValidationErrorResponse formats and sends validation error responses
func ValidationErrorResponse(c *gin.Context, err error) {
	if errs, ok := err.(validator.ValidationErrors); ok && len(errs) > 0 {
		full := errs[0].Error()
		parts := strings.SplitN(full, "Error:", 2)
		message := full
		if len(parts) == 2 {
			message = strings.TrimSpace(parts[1])
		}
		ErrorResponse(c, http.StatusBadRequest, status.StatusValidationFailed, message)
		return
	}
	ErrorResponse(c, http.StatusBadRequest, status.StatusValidationFailed, "Invalid request format")
}

// SuccessResponse sends a success response to the client
func SuccessResponse(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, GenericResponse{
		Data:    data,
		Success: true,
		Message: message,
	})
}

package mfa

import (
	"errors"
	"net/http"

	"cirrussync-api/internal/logger"
	"cirrussync-api/internal/mfa"
	"cirrussync-api/pkg/status"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Handler handles MFA HTTP requests
type Handler struct {
	service *mfa.Service
	logger  *logger.Logger
}

// NewHandler creates a new MFA handler
func NewHandler(service *mfa.Service, logger *logger.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// secureLog logs errors without sensitive data
func (h *Handler) secureLog(err error, message string, route string) {
	// Use the logger to log errors
	h.logger.WithFields(logrus.Fields{
		"route":    route,
		"errorMsg": err.Error(),
	}).Error(message)
}

// handleErrorResponse maps errors to HTTP responses
func (h *Handler) handleErrorResponse(c *gin.Context, err error, result *mfa.EmailVerificationResult) {
	statusCode := http.StatusInternalServerError
	apiStatus := status.StatusInternalServerError
	message := err.Error()

	// Map errors to appropriate HTTP status codes
	switch {
	case errors.Is(err, mfa.ErrInvalidEmail) || errors.Is(err, mfa.ErrInvalidInput):
		statusCode = http.StatusBadRequest
		apiStatus = status.StatusBadRequest
	case errors.Is(err, mfa.ErrEmailAlreadySent) || errors.Is(err, mfa.ErrRateLimitExceeded):
		statusCode = http.StatusTooManyRequests
		apiStatus = status.StatusTooManyRequests
	case errors.Is(err, mfa.ErrEmailExists):
		statusCode = http.StatusConflict
		apiStatus = status.StatusConflict
	case errors.Is(err, mfa.ErrUsernameExists):
		statusCode = http.StatusConflict
		apiStatus = status.StatusConflict
	case errors.Is(err, mfa.ErrTOTPAlreadyEnabled) || errors.Is(err, mfa.ErrTOTPNotEnabled) ||
		errors.Is(err, mfa.ErrTOTPNotInitialized) || errors.Is(err, mfa.ErrInvalidTOTPCode):
		statusCode = http.StatusBadRequest
		apiStatus = status.StatusBadRequest
	case errors.Is(err, mfa.ErrTOTPSetupInProgress) || errors.Is(err, mfa.ErrTOTPOperationInProgress):
		statusCode = http.StatusConflict
		apiStatus = status.StatusConflict
	}

	ErrorResponse(c, statusCode, apiStatus, message)
}

// HandleSendVerification handles requests to send verification emails
func (h *Handler) HandleSendVerification(c *gin.Context) {
	var req SendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.secureLog(err, "Invalid request format", "sendVerification")
		ValidationErrorResponse(c, err)
		return
	}

	// Call service with intent parameter
	result, err := h.service.SendVerificationEmail(c.Request.Context(), req.Email, req.Username, req.Intent)
	if err != nil {
		h.secureLog(err, "Failed to send verification email", "sendVerification")
		h.handleErrorResponse(c, err, result)
		return
	}

	// Send successful response
	resp := SendVerificationResponseData{
		Success: true,
	}

	// Include rate limit info if available
	if result != nil {
		resp.RemainingRequests = result.RemainingRequests
		resp.ResetTime = result.ResetTime
		resp.NextAllowedTime = result.NextAllowedTime
	}

	SuccessResponse(c, resp, "Verification email sent successfully")
}

// HandleVerifyEmail handles requests to verify emails with tokens
func (h *Handler) HandleVerifyEmail(c *gin.Context) {
	// Try to get token from query parameter first
	token := c.Query("token")

	// If not in query, try to get it from JSON body
	if token == "" {
		var req VerifyEmailRequest
		if err := c.ShouldBindJSON(&req); err == nil {
			token = req.Token
		}
	}

	if token == "" {
		ErrorResponse(c, http.StatusBadRequest, status.StatusBadRequest, "Missing token")
		return
	}

	email, verified := h.service.VerifyEmail(c.Request.Context(), token)

	if verified {
		SuccessResponse(c, VerifyEmailResponseData{
			Verified: true,
			Email:    email,
		}, "Email verified successfully")
	} else {
		ErrorResponse(c, http.StatusBadRequest, status.StatusInvalidToken, "Invalid or expired verification token")
	}
}

// HandleCheckVerification handles requests to check if an email is verified
func (h *Handler) HandleCheckVerification(c *gin.Context) {
	email := c.Query("email")

	// Try to get email from JSON body if not in query
	if email == "" {
		var req CheckVerificationRequest
		if err := c.ShouldBindJSON(&req); err == nil {
			email = req.Email
		}
	}

	if email == "" {
		ErrorResponse(c, http.StatusBadRequest, status.StatusBadRequest, "Email parameter is required")
		return
	}

	verified, err := h.service.IsEmailVerified(c.Request.Context(), email)
	if err != nil {
		h.secureLog(err, "Failed to check email verification", "checkVerification")
		h.handleErrorResponse(c, err, nil)
		return
	}

	SuccessResponse(c, CheckVerificationResponseData{
		Verified: verified,
		Email:    email,
	}, "Email verification status retrieved successfully")
}

// HandleGenerateTOTP handles requests to generate TOTP
func (h *Handler) HandleGenerateTOTP(c *gin.Context) {
	var req GenerateTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.secureLog(err, "Invalid request format", "generateTOTP")
		ValidationErrorResponse(c, err)
		return
	}

	totpData, err := h.service.EnableTOTP(c.Request.Context(), req.UserID)
	if err != nil {
		h.secureLog(err, "Failed to generate TOTP", "generateTOTP")
		h.handleErrorResponse(c, err, nil)
		return
	}

	SuccessResponse(c, GenerateTOTPResponseData{
		Secret:       totpData.Secret,
		QRCodeURL:    totpData.QRCodeURL,
		RecoveryKeys: totpData.RecoveryKeys,
	}, "TOTP generated successfully")
}

// HandleVerifyTOTP handles requests to verify TOTP
func (h *Handler) HandleVerifyTOTP(c *gin.Context) {
	var req VerifyTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.secureLog(err, "Invalid request format", "verifyTOTP")
		ValidationErrorResponse(c, err)
		return
	}

	valid, err := h.service.VerifyTOTP(c.Request.Context(), req.UserID, req.Code)
	if err != nil {
		h.secureLog(err, "Failed to verify TOTP", "verifyTOTP")
		h.handleErrorResponse(c, err, nil)
		return
	}

	if !valid {
		ErrorResponse(c, http.StatusBadRequest, status.StatusInvalidToken, "Invalid TOTP code")
		return
	}

	SuccessResponse(c, VerifyTOTPResponseData{
		Success: true,
	}, "TOTP verified and enabled successfully")
}

// HandleValidateTOTP handles requests to validate a TOTP code
func (h *Handler) HandleValidateTOTP(c *gin.Context) {
	var req ValidateTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.secureLog(err, "Invalid request format", "validateTOTP")
		ValidationErrorResponse(c, err)
		return
	}

	valid, err := h.service.ValidateTOTPCode(c.Request.Context(), req.UserID, req.Code)
	if err != nil {
		h.secureLog(err, "Failed to validate TOTP", "validateTOTP")
		h.handleErrorResponse(c, err, nil)
		return
	}

	message := "TOTP code is invalid"
	if valid {
		message = "TOTP code is valid"
	}
	SuccessResponse(c, ValidateTOTPResponseData{
		Valid: valid,
	}, message)
}

// HandleDisableTOTP handles requests to disable TOTP
func (h *Handler) HandleDisableTOTP(c *gin.Context) {
	var req DisableTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.secureLog(err, "Invalid request format", "disableTOTP")
		ValidationErrorResponse(c, err)
		return
	}

	err := h.service.DisableTOTP(c.Request.Context(), req.UserID, req.ConfirmationCode)
	if err != nil {
		h.secureLog(err, "Failed to disable TOTP", "disableTOTP")
		h.handleErrorResponse(c, err, nil)
		return
	}

	SuccessResponse(c, DisableTOTPResponseData{
		Success: true,
	}, "TOTP disabled successfully")
}

package auth

import (
	"context"
	"net/http"
	"strings"
	"time"

	"cirrussync-api/internal/auth"
	"cirrussync-api/internal/jwt"
	"cirrussync-api/internal/logger"
	"cirrussync-api/internal/models"
	"cirrussync-api/internal/session"
	"cirrussync-api/internal/srp"
	"cirrussync-api/internal/user"
	"cirrussync-api/internal/utils"
	"cirrussync-api/pkg/status"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// NewHandler creates a new auth handler
func NewHandler(authService *auth.Service, userService *user.Service, jwtService *jwt.JWTService, sessionService *session.Service, log *logger.Logger) *Handler {
	return &Handler{
		authService:    authService,
		userService:    userService,
		jwtService:     jwtService,
		sessionService: sessionService,
		logger:         log,
	}
}

// secureLog logs errors without sensitive data that might expose code or credentials
func (h *Handler) secureLog(err error, message string, route string) {
	requestID := utils.GenerateShortID()
	h.logger.WithFields(logrus.Fields{
		"requestID": requestID,
		"route":     route,
		"errorMsg":  err.Error(),
	}).Error(message)
}

// HandleLoginInit handles SRP authentication initialization
func (h *Handler) HandleLoginInit(c *gin.Context) {
	var req LoginInitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.secureLog(err, "Invalid request format", "loginInit")
		c.JSON(http.StatusUnprocessableEntity, NewValidationError(err, status.StatusValidationFailed))
		return
	}

	// Get client IP address (with Cloudflare support)
	ipAddress := srp.GetClientIPFromRequest(c.Request)
	userAgent := c.GetHeader("User-Agent")

	// Initialize SRP authentication
	response, err := h.authService.LoginInit(
		c.Request.Context(),
		req.Email,
		req.ClientPublic,
		ipAddress,
		userAgent,
	)

	if err != nil {
		statusCode := http.StatusInternalServerError
		apiStatusCode := status.StatusInternalServerError

		switch err {
		case srp.ErrInvalidInput:
			statusCode = http.StatusBadRequest
			apiStatusCode = status.StatusBadRequest
		case srp.ErrRateLimited:
			statusCode = http.StatusTooManyRequests
			apiStatusCode = status.StatusTooManyRequests
		case srp.ErrInvalidClientPublic:
			statusCode = http.StatusBadRequest
			apiStatusCode = status.StatusBadRequest
		}

		h.secureLog(err, err.Error(), "loginInit")
		c.JSON(statusCode, NewErrorResponse(err.Error(), apiStatusCode))
		return
	}

	c.JSON(http.StatusOK, NewLoginInitResponse(
		response.SessionID,
		response.Salt,
		response.ServerPublic,
		status.StatusSRPChallengeIssued,
	))
}

// HandleLoginVerify handles SRP authentication verification
func (h *Handler) HandleLoginVerify(c *gin.Context) {
	var req LoginVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.secureLog(err, "Invalid request format", "loginVerify")
		c.JSON(http.StatusUnprocessableEntity, NewValidationError(err, status.StatusValidationFailed))
		return
	}

	// Get client IP address (with Cloudflare support)
	ipAddress := srp.GetClientIPFromRequest(c.Request)

	// Verify SRP authentication
	response, userSRP, err := h.authService.LoginVerify(
		c.Request.Context(),
		req.SessionID,
		req.ClientProof,
		ipAddress,
	)
	if err != nil {
		statusCode := http.StatusInternalServerError
		apiStatusCode := status.StatusInternalServerError

		switch err {
		case srp.ErrInvalidInput:
			statusCode = http.StatusBadRequest
			apiStatusCode = status.StatusBadRequest
		case srp.ErrRateLimited:
			statusCode = http.StatusTooManyRequests
			apiStatusCode = status.StatusTooManyRequests
		case srp.ErrInvalidSession:
			statusCode = http.StatusBadRequest
			apiStatusCode = status.StatusInvalidToken
		case srp.ErrInvalidClientProof:
			statusCode = http.StatusBadRequest
			apiStatusCode = status.StatusInvalidCredentials
		case srp.ErrUserNotFound:
			statusCode = http.StatusBadRequest
			apiStatusCode = status.StatusNotFound
		}
		h.secureLog(err, err.Error(), "loginVerify")
		c.JSON(statusCode, NewErrorResponse(err.Error(), apiStatusCode))
		return
	}

	// Create channels for parallel operations
	userChan := make(chan *models.User)
	userErrChan := make(chan error)
	sessionChan := make(chan *models.UserSession)
	sessionErrChan := make(chan error)
	tokenChan := make(chan jwt.TokenPair)
	tokenErrChan := make(chan error)

	deviceInfo := GetDeviceDetails(c)
	ctx := c.Request.Context()

	// Get user in parallel
	go func() {
		user, err := h.userService.GetUserById(c.Request.Context(), userSRP.UserID)
		if err != nil {
			userErrChan <- err
			return
		}
		userChan <- user
	}()

	// Wait for user to continue with session creation
	var user *models.User
	select {
	case user = <-userChan:
		// User retrieved successfully
	case err := <-userErrChan:
		h.secureLog(err, "Failed to get user after successful SRP authentication", "loginVerify")
		c.JSON(http.StatusInternalServerError, NewErrorResponse("Failed to get user information", status.StatusInternalServerError))
		return
	}

	// Create session and generate token in parallel
	go func() {
		// Create session directly using the session service
		sessionDeviceInfo := session.DeviceInfo{
			ClientName: deviceInfo.ClientName,
			ClientUID:  deviceInfo.ClientUID,
			AppVersion: deviceInfo.AppVersion,
			UserAgent:  deviceInfo.UserAgent,
		}

		session, err := h.sessionService.CreateSession(
			ctx,
			user,
			sessionDeviceInfo,
			ipAddress,
		)
		if err != nil {
			sessionErrChan <- err
			return
		}
		sessionChan <- session

		// Generate token once we have the session
		token, err := h.jwtService.GenerateAuthTokens(*user, session.ID)
		if err != nil {
			tokenErrChan <- err
			return
		}
		tokenChan <- token
	}()

	// Get session and token results
	var userSession *models.UserSession
	var token jwt.TokenPair

	// Get session result
	select {
	case userSession = <-sessionChan:
		// Session created successfully
	case err := <-sessionErrChan:
		h.secureLog(err, err.Error(), "loginVerify")
		c.JSON(http.StatusInternalServerError, NewErrorResponse(err.Error(), status.StatusInternalServerError))
		return
	}

	// Get token result
	select {
	case token = <-tokenChan:
		// Token generated successfully
	case err := <-tokenErrChan:
		h.secureLog(err, err.Error(), "loginVerify")
		c.JSON(http.StatusInternalServerError, NewErrorResponse(err.Error(), status.StatusJWTError))
		return
	}

	maxAge := int(userSession.ExpiresAt - time.Now().Unix()) // Lifetime in seconds

	// Set SameSite once before setting any cookies
	c.SetSameSite(http.SameSiteStrictMode)

	// Set cookies
	c.SetCookie("sessionID", userSession.ID, maxAge, "/", "localhost", false, true)
	c.SetCookie("accessToken", token.AccessToken, 60*60, "/api/v1", "localhost", false, true)
	c.SetCookie("refreshToken", token.RefreshToken, 24*60*60, "/api/v1/auth/refresh", "localhost", false, true)

	// Return the response
	c.JSON(http.StatusOK, NewLoginVerifyResponse(
		token,
		user,
		userSession,
		response.ServerProof,
		status.StatusLoginSuccess,
	))
}

// HandleSignup handles user registration
func (h *Handler) HandleSignup(c *gin.Context) {
	var req SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.secureLog(err, "Invalid request format", "signup")
		c.JSON(http.StatusUnprocessableEntity, NewValidationError(err, status.StatusValidationFailed))
		return
	}

	// Normalize email
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Create user
	user, err := h.authService.CreateUser(c.Request.Context(), email, req.Username, req.Keys)
	if err != nil {
		statusCode := http.StatusInternalServerError
		apiStatusCode := status.StatusInternalServerError

		switch err {
		case auth.ErrInvalidEmail:
			statusCode = http.StatusBadRequest
			apiStatusCode = status.StatusBadRequest
		case auth.ErrInvalidUsername:
			statusCode = http.StatusBadRequest
			apiStatusCode = status.StatusBadRequest
		case auth.ErrInvalidInput:
			statusCode = http.StatusBadRequest
			apiStatusCode = status.StatusBadRequest
		case auth.ErrUsernameAlreadyExists:
			statusCode = http.StatusConflict
			apiStatusCode = status.StatusEmailAlreadyExists
		case auth.ErrEmailAlreadyExists:
			statusCode = http.StatusConflict
			apiStatusCode = status.StatusEmailAlreadyExists
		}

		h.secureLog(err, err.Error(), "signup")
		c.JSON(statusCode, NewErrorResponse(err.Error(), apiStatusCode))
		return
	}

	// Store SRP credentials that were generated client-side
	err = h.authService.RegisterSRP(
		c.Request.Context(),
		user.ID,
		email,
		req.SRPSalt,     // Client-generated salt
		req.SRPVerifier, // Client-generated verifier
	)
	if err != nil {
		// If SRP registration fails, delete the user
		h.userService.DeleteUser(c, user.ID)
		h.secureLog(err, err.Error(), "signup")
		c.JSON(http.StatusInternalServerError, NewErrorResponse(err.Error(), status.StatusSRPError))
		return
	}

	c.JSON(http.StatusCreated, NewSignupResponse(
		user.ID,
		status.StatusSignupSuccess,
	))
}

// HandleChangePassword handles changing a user's password
func (h *Handler) HandleChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.secureLog(err, "Invalid request format", "changePassword")
		c.JSON(http.StatusUnprocessableEntity, NewValidationError(err, status.StatusValidationFailed))
		return
	}

	// Get user ID and email from authenticated context
	userID, _ := c.Get("userID")
	email, _ := c.Get("email")

	// Change password using client-generated credentials
	err := h.authService.ChangePassword(
		c.Request.Context(),
		userID.(string),
		email.(string),
		req.OldSRPSalt,
		req.OldSRPVerifier,
		req.NewSRPSalt,
		req.NewSRPVerifier,
	)

	if err != nil {
		statusCode := http.StatusInternalServerError
		apiStatusCode := status.StatusInternalServerError

		if err == auth.ErrInvalidCredentials {
			statusCode = http.StatusUnauthorized
			apiStatusCode = status.StatusInvalidCredentials
		}

		h.secureLog(err, err.Error(), "changePassword")
		c.JSON(statusCode, NewErrorResponse(err.Error(), apiStatusCode))
		return
	}

	c.JSON(http.StatusOK, NewSuccessResponse("Password changed successfully", status.StatusPasswordChanged))
}

// HandleLogout handles user logout
func (h *Handler) HandleLogout(c *gin.Context) {
	logoutErrChan := make(chan error, 1)

	// Get session ID from token claims
	sessionID, exists := c.Get("sessionID")
	if !exists {
		c.JSON(http.StatusBadRequest, NewErrorResponse("Session not found", status.StatusBadRequest))
		return
	}

	// Set SameSite once before setting any cookies
	c.SetSameSite(http.SameSiteStrictMode)

	// Set cookies to expire immediately - do this first for good UX
	c.SetCookie("sessionID", "", -1, "/", "localhost", false, true)
	c.SetCookie("accessToken", "", -1, "/api/v1", "localhost", false, true)
	c.SetCookie("refreshToken", "", -1, "/api/v1/auth/refresh", "localhost", false, true)

	// Return success response immediately
	c.JSON(http.StatusOK, NewSuccessResponse("Logged out successfully", status.StatusLogoutSuccess))

	// Invalidate the session asynchronously with a channel for logging
	go func() {
		ctx := context.Background()
		err := h.sessionService.InvalidateSession(ctx, sessionID.(string))
		logoutErrChan <- err
	}()

	// Log the result in a separate goroutine
	go func() {
		err := <-logoutErrChan
		if err != nil {
			h.secureLog(err, err.Error(), "logout")
		}
	}()
}

// HandleRefreshToken refreshes a JWT token
func (h *Handler) HandleRefreshToken(c *gin.Context) {
	var sessionID, userID string
	var needsValidation bool = true

	// Check if middleware has already verified refresh token
	isRefreshToken, hasRefreshToken := c.Get("isRefreshToken")
	if hasRefreshToken && isRefreshToken.(bool) {
		// Middleware already authenticated via refresh token
		sessionIDVal, _ := c.Get("sessionID")
		userIDVal, _ := c.Get("userID")

		// Extract values if they exist
		if sid, ok := sessionIDVal.(string); ok && sid != "" {
			sessionID = sid
			if uid, ok := userIDVal.(string); ok && uid != "" {
				userID = uid
				needsValidation = false
			}
		}
	}

	// If we need to validate the token manually
	if needsValidation {
		refreshToken := extractTokenFromSources(c, "Authorization", "refreshToken")
		if refreshToken == "" {
			c.JSON(http.StatusUnauthorized, NewErrorResponse("No refresh token provided", status.StatusUnauthorized))
			return
		}

		// Validate the refresh token
		claims, err := h.jwtService.ValidateToken(refreshToken)
		if err != nil || !*claims.IsRefreshToken {
			h.secureLog(err, "Invalid refresh token", "refreshToken")
			c.JSON(http.StatusUnauthorized, NewErrorResponse("Invalid refresh token", status.StatusInvalidToken))
			return
		}

		// Extract the required IDs
		sessionID = claims.SessionID
		userID = claims.UserID
		if sessionID == "" || userID == "" {
			c.JSON(http.StatusUnauthorized, NewErrorResponse("Invalid token claims", status.StatusInvalidToken))
			return
		}
	}

	// Create channels for parallel operations
	userChan := make(chan *models.User)
	userErrChan := make(chan error)
	sessionChan := make(chan *models.UserSession)
	sessionErrChan := make(chan error)
	ctx := c.Request.Context()

	// Fetch user in parallel
	go func() {
		user, err := h.userService.GetUserById(ctx, userID)
		if err != nil {
			userErrChan <- err
			return
		}
		userChan <- user
	}()

	// Fetch session in parallel
	go func() {
		currentSession, err := h.sessionService.GetSessionByID(ctx, sessionID)
		if err != nil {
			sessionErrChan <- err
			return
		}

		// Check session validity
		if !currentSession.IsValid {
			sessionErrChan <- session.ErrSessionInvalid
			return
		}

		// Check session expiration
		if currentSession.ExpiresAt < time.Now().Unix() {
			sessionErrChan <- session.ErrSessionExpired
			return
		}

		sessionChan <- currentSession
	}()

	// Wait for both operations to complete
	var user *models.User
	var userSession *models.UserSession

	// Get user result
	select {
	case user = <-userChan:
		// User retrieved successfully
	case err := <-userErrChan:
		c.JSON(http.StatusInternalServerError, NewErrorResponse(err.Error(), status.StatusInternalServerError))
		return
	}

	// Get session result
	select {
	case userSession = <-sessionChan:
		// Session retrieved successfully
	case err := <-sessionErrChan:
		c.JSON(http.StatusInternalServerError, NewErrorResponse(err.Error(), status.StatusInternalServerError))
		return
	}

	// Generate new tokens
	token, err := h.jwtService.GenerateAuthTokens(*user, sessionID)
	if err != nil {
		h.secureLog(err, err.Error(), "refreshToken")
		c.JSON(http.StatusUnauthorized, NewErrorResponse(err.Error(), status.StatusInvalidToken))
		return
	}

	// Update session last activity asynchronously
	go h.sessionService.UpdateSessionByID(context.Background(), sessionID)

	// Set cookies
	sessionTTL := int(userSession.ExpiresAt - time.Now().Unix())
	if sessionTTL < 0 {
		sessionTTL = 0 // Prevent negative TTL
	}

	// Set SameSite once before setting any cookies
	c.SetSameSite(http.SameSiteStrictMode)

	// Then set all cookies (they'll inherit the SameSite setting)
	c.SetCookie("sessionID", userSession.ID, sessionTTL, "/", "localhost", false, true)
	c.SetCookie("accessToken", token.AccessToken, 60*60, "/api/v1", "localhost", false, true)                   // 1 hour
	c.SetCookie("refreshToken", token.RefreshToken, 24*60*60, "/api/v1/auth/refresh", "localhost", false, true) // 1 day

	// Return response
	c.JSON(http.StatusOK, NewRefreshTokenResponse(
		token,
		user,
		userSession,
		status.StatusTokenRefreshed,
	))
}

// Helper to extract token from sources (reused from middleware)
func extractTokenFromSources(c *gin.Context, headerName, cookieName string) string {
	header := c.GetHeader(headerName)
	if headerName == "Authorization" && header != "" {
		parts := strings.Split(header, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	} else if header != "" {
		return header
	}

	cookie, err := c.Cookie(cookieName)
	if err == nil && cookie != "" {
		return cookie
	}

	return ""
}

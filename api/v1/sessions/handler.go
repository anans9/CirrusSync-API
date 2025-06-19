package session

import (
	"net/http"

	"cirrussync-api/internal/logger"
	"cirrussync-api/internal/session"
	"cirrussync-api/internal/utils"
	"cirrussync-api/pkg/status"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Handler handles session-related requests
type Handler struct {
	sessionService *session.Service
	logger         *logger.Logger
}

// NewHandler creates a new session handler
func NewHandler(sessionService *session.Service, log *logger.Logger) *Handler {
	return &Handler{
		sessionService: sessionService,
		logger:         log,
	}
}

// secureLog logs errors without sensitive data that might expose code or credentials
func (h *Handler) secureLog(err error, message string, route string) {
	// Generate request ID internally
	requestID := utils.GenerateShortID()

	// Log only necessary information, avoid including stack traces or request bodies
	h.logger.WithFields(logrus.Fields{
		"requestID": requestID,
		"route":     route,
		"errorMsg":  err.Error(),
	}).Error(message)
}

// GetCurrentSession retrieves the current session information
func (h *Handler) GetCurrentSession(c *gin.Context) {
	// Get session ID from context (set by auth middleware)
	sessionID, exists := c.Get("sessionID")
	if !exists {
		h.secureLog(session.ErrSessionNotFound, "Session ID not found in context", "getSession")
		c.JSON(http.StatusUnauthorized, NewErrorResponse("Session not found", status.StatusUnauthorized))
		return
	}

	// Convert to string
	sessionIDStr, ok := sessionID.(string)
	if !ok {
		h.secureLog(session.ErrInvalidInput, "Invalid session ID format", "getSession")
		c.JSON(http.StatusUnauthorized, NewErrorResponse("Invalid session format", status.StatusUnauthorized))
		return
	}

	// Get session from service
	userSession, err := h.sessionService.GetSessionByID(c, sessionIDStr)
	if err != nil {
		h.secureLog(err, err.Error(), "getSession")

		statusCode := http.StatusInternalServerError
		apiStatus := status.StatusInternalServerError

		switch err {
		case session.ErrSessionNotFound:
			statusCode = http.StatusNotFound
			apiStatus = status.StatusNotFound
		case session.ErrSessionExpired:
			statusCode = http.StatusUnauthorized
			apiStatus = status.StatusSessionExpired
		case session.ErrSessionInvalid:
			statusCode = http.StatusUnauthorized
			apiStatus = status.StatusInvalidSession
		}

		c.JSON(statusCode, NewErrorResponse(err.Error(), apiStatus))
		return
	}

	// Return the session information
	c.JSON(http.StatusOK, NewSessionResponse(userSession, status.StatusOK))
}

// InvalidateSession invalidates the current session (logout)
func (h *Handler) InvalidateSession(c *gin.Context) {
	// Get session ID from context
	sessionID, exists := c.Get("sessionID")
	if !exists {
		h.secureLog(session.ErrSessionNotFound, "Session ID not found in context", "invalidateSession")
		c.JSON(http.StatusUnauthorized, NewErrorResponse("Session not found", status.StatusUnauthorized))
		return
	}

	// Convert to string
	sessionIDStr, ok := sessionID.(string)
	if !ok {
		h.secureLog(session.ErrInvalidInput, "Invalid session ID format", "invalidateSession")
		c.JSON(http.StatusUnauthorized, NewErrorResponse("Invalid session format", status.StatusUnauthorized))
		return
	}

	// Invalidate session
	err := h.sessionService.InvalidateSession(c, sessionIDStr)
	if err != nil {
		h.secureLog(err, err.Error(), "invalidateSession")

		statusCode := http.StatusInternalServerError
		apiStatus := status.StatusInternalServerError

		switch err {
		case session.ErrSessionNotFound:
			statusCode = http.StatusNotFound
			apiStatus = status.StatusNotFound
		case session.ErrInvalidInput:
			statusCode = http.StatusBadRequest
			apiStatus = status.StatusBadRequest
		}

		c.JSON(statusCode, NewErrorResponse(err.Error(), apiStatus))
		return
	}

	// Clear cookies if they exist
	c.SetCookie("sessionID", "", -1, "/", "localhost", false, true)
	c.SetCookie("accessToken", "", -1, "/", "localhost", false, true)
	c.SetCookie("refreshToken", "", -1, "/", "localhost", false, true)

	c.JSON(http.StatusOK, NewSuccessResponse("Session invalidated successfully", status.StatusOK))
}

// GetAllActiveSessions retrieves all active sessions for the current user
func (h *Handler) GetAllActiveSessions(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		h.secureLog(session.ErrInvalidInput, "User ID not found in context", "getAllSessions")
		c.JSON(http.StatusUnauthorized, NewErrorResponse("User not authenticated", status.StatusUnauthorized))
		return
	}

	// Convert to string
	userIDStr, ok := userID.(string)
	if !ok {
		h.secureLog(session.ErrInvalidInput, "Invalid user ID format", "getAllSessions")
		c.JSON(http.StatusUnauthorized, NewErrorResponse("Invalid user ID format", status.StatusUnauthorized))
		return
	}

	// Get all user sessions
	sessions, err := h.sessionService.GetUserSessions(c, userIDStr)
	if err != nil {
		h.secureLog(err, err.Error(), "getAllSessions")
		c.JSON(http.StatusInternalServerError, NewErrorResponse(err.Error(), status.StatusInternalServerError))
		return
	}

	// Return all active sessions
	c.JSON(http.StatusOK, NewSessionsListResponse(sessions, status.StatusOK))
}

// InvalidateAllSessions invalidates all sessions for the current user
func (h *Handler) InvalidateAllSessions(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		h.secureLog(session.ErrInvalidInput, "User ID not found in context", "invalidateAllSessions")
		c.JSON(http.StatusUnauthorized, NewErrorResponse("User not authenticated", status.StatusUnauthorized))
		return
	}

	// Convert to string
	userIDStr, ok := userID.(string)
	if !ok {
		h.secureLog(session.ErrInvalidInput, "Invalid user ID format", "invalidateAllSessions")
		c.JSON(http.StatusUnauthorized, NewErrorResponse("Invalid user ID format", status.StatusUnauthorized))
		return
	}

	// Invalidate all sessions
	err := h.sessionService.InvalidateAllUserSessions(c, userIDStr)
	if err != nil {
		h.secureLog(err, err.Error(), "invalidateAllSessions")
		c.JSON(http.StatusInternalServerError, NewErrorResponse(err.Error(), status.StatusInternalServerError))
		return
	}

	// Clear cookies if they exist
	c.SetCookie("sessionID", "", -1, "/", "localhost", false, true)
	c.SetCookie("accessToken", "", -1, "/", "localhost", false, true)
	c.SetCookie("refreshToken", "", -1, "/", "localhost", false, true)

	c.JSON(http.StatusOK, NewSuccessResponse("All sessions invalidated successfully", status.StatusOK))
}

// InvalidateSessionByID invalidates a specific session by ID
func (h *Handler) InvalidateSessionByID(c *gin.Context) {
	// Get session ID from URL parameters
	sessionID := c.Param("id")
	if sessionID == "" {
		h.secureLog(session.ErrInvalidInput, "Session ID not provided", "invalidateSessionById")
		c.JSON(http.StatusBadRequest, NewErrorResponse("Session ID required", status.StatusBadRequest))
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		h.secureLog(session.ErrInvalidInput, "User ID not found in context", "invalidateSessionById")
		c.JSON(http.StatusUnauthorized, NewErrorResponse("User not authenticated", status.StatusUnauthorized))
		return
	}

	// Convert to string
	userIDStr, ok := userID.(string)
	if !ok {
		h.secureLog(session.ErrInvalidInput, "Invalid user ID format", "invalidateSessionById")
		c.JSON(http.StatusUnauthorized, NewErrorResponse("Invalid user ID format", status.StatusUnauthorized))
		return
	}

	// Verify the session belongs to the user
	currentSession, err := h.sessionService.GetSessionByID(c, sessionID)
	if err != nil {
		h.secureLog(err, err.Error(), "invalidateSessionById")

		statusCode := http.StatusInternalServerError
		apiStatus := status.StatusInternalServerError

		switch err {
		case session.ErrSessionNotFound:
			statusCode = http.StatusNotFound
			apiStatus = status.StatusNotFound
		case session.ErrSessionExpired:
			statusCode = http.StatusUnauthorized
			apiStatus = status.StatusSessionExpired
		case session.ErrSessionInvalid:
			statusCode = http.StatusUnauthorized
			apiStatus = status.StatusInvalidSession
		}

		c.JSON(statusCode, NewErrorResponse(err.Error(), apiStatus))
		return
	}

	// Check if session belongs to user
	if currentSession.UserID != userIDStr {
		h.secureLog(session.ErrUnauthorized, "Unauthorized session access", "invalidateSessionById")
		c.JSON(http.StatusForbidden, NewErrorResponse("You do not have permission to invalidate this session", status.StatusForbidden))
		return
	}

	// Invalidate the session
	err = h.sessionService.InvalidateSession(c, sessionID)
	if err != nil {
		h.secureLog(err, err.Error(), "invalidateSessionById")
		c.JSON(http.StatusInternalServerError, NewErrorResponse(err.Error(), status.StatusInternalServerError))
		return
	}

	// Clear cookies if the invalidated session is the current one
	currentSessionID, _ := c.Get("sessionID")
	if currentSessionID == sessionID {
		c.SetCookie("sessionID", "", -1, "/", "localhost", false, true)
		c.SetCookie("accessToken", "", -1, "/", "localhost", false, true)
		c.SetCookie("refreshToken", "", -1, "/", "localhost", false, true)
	}

	c.JSON(http.StatusOK, NewSuccessResponse("Session invalidated successfully", status.StatusOK))
}

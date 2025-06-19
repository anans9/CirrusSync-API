package csrf

import (
	"errors"
	"net/http"
	"time"

	"cirrussync-api/internal/logger"
	"cirrussync-api/internal/utils"
	"cirrussync-api/pkg/status"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/sirupsen/logrus"
)

// Handler handles HTTP requests for CSRF tokens
type Handler struct {
	logger *logger.Logger
}

// NewHandler creates a new CSRF handler
func NewHandler(logger *logger.Logger) *Handler {
	return &Handler{
		logger: logger,
	}
}

// secureLog logs errors without sensitive data that might expose code
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

// HandleCSRFToken generates and returns a CSRF token
func (h *Handler) HandleCSRFToken(c *gin.Context) {
	token := csrf.Token(c.Request)
	if token == "" {
		h.secureLog(errors.New("returned empty token"), "Failed to generate CSRF token", "/csrf")
		c.JSON(http.StatusInternalServerError, NewErrorResponse(
			status.StatusBadRequest,
			"Internal server error, please try again later",
		))
		return
	}
	expiresAt := time.Now().Add(time.Hour).Unix()
	c.JSON(http.StatusOK, NewResponse(token, expiresAt))
}

package logger

import (
	"cirrussync-api/internal/utils"

	"github.com/sirupsen/logrus"
)

// Logger wraps the logrus logger with additional functionality
type Logger struct {
	log *logrus.Logger
}

// New creates a new Logger instance
func New(log *logrus.Logger) *Logger {
	return &Logger{
		log: log,
	}
}

// SecureLog logs errors without sensitive data that might expose code or credentials
func (l *Logger) SecureLog(err error, message string, route string) {
	// Generate request ID internally
	requestID := utils.GenerateShortID()

	// Log only necessary information, avoid including stack traces or request bodies
	l.log.WithFields(logrus.Fields{
		"request_id": requestID,
		"route":      route,
		"error_msg":  err.Error(),
	}).Error(message)
}

// WithField adds a field to the logger
func (l *Logger) WithField(key string, value interface{}) *logrus.Entry {
	return l.log.WithField(key, value)
}

// WithFields adds multiple fields to the logger
func (l *Logger) WithFields(fields logrus.Fields) *logrus.Entry {
	return l.log.WithFields(fields)
}

// Info logs an info message
func (l *Logger) Info(args ...interface{}) {
	l.log.Info(args...)
}

// Infof logs an info message with formatting
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log.Infof(format, args...)
}

// Error logs an error message
func (l *Logger) Error(args ...interface{}) {
	l.log.Error(args...)
}

// Errorf logs an error message with formatting
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log.Errorf(format, args...)
}

// Debug logs a debug message
func (l *Logger) Debug(args ...interface{}) {
	l.log.Debug(args...)
}

// Debugf logs a debug message with formatting
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log.Debugf(format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(args ...interface{}) {
	l.log.Warn(args...)
}

// Warnf logs a warning message with formatting
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log.Warnf(format, args...)
}

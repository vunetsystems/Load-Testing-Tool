package logger

import (
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var Logger zerolog.Logger

// LogEntry represents a structured log entry for API consumption
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Node      string    `json:"node"`
	Module    string    `json:"module"`
	Message   string    `json:"message"`
	Type      string    `json:"type"` // info, error, warning, success, metric
	Level     string    `json:"level"`
}

// InitLogger initializes the global logger with console and file output
func InitLogger(logFilePath string) error {
	// Create logs directory if it doesn't exist
	logDir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// Open log file
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	// Create multi-writer for console and file
	multi := zerolog.MultiLevelWriter(
		zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		},
		logFile,
	)

	// Configure logger
	Logger = zerolog.New(multi).With().Timestamp().Logger()

	// Set global logger
	log.Logger = Logger

	return nil
}

// Info logs an info message
func Info() *zerolog.Event {
	return Logger.Info()
}

// Error logs an error message
func Error() *zerolog.Event {
	return Logger.Error()
}

// Warn logs a warning message
func Warn() *zerolog.Event {
	return Logger.Warn()
}

// Debug logs a debug message
func Debug() *zerolog.Event {
	return Logger.Debug()
}

// Fatal logs a fatal message and exits
func Fatal() *zerolog.Event {
	return Logger.Fatal()
}

// WithFields creates a logger with additional fields
func WithFields(fields map[string]interface{}) zerolog.Logger {
	ctx := Logger.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return ctx.Logger()
}

// LogWithNode logs a message with node context
func LogWithNode(node, module, message string, level string) {
	event := Logger.Info()
	if level == "error" {
		event = Logger.Error()
	} else if level == "warn" {
		event = Logger.Warn()
	} else if level == "debug" {
		event = Logger.Debug()
	}

	event.
		Str("node", node).
		Str("module", module).
		Str("type", level).
		Msg(message)
}

// LogMetric logs a metric-related message
func LogMetric(node, module, message string) {
	Logger.Info().
		Str("node", node).
		Str("module", module).
		Str("type", "metric").
		Msg(message)
}

// LogSuccess logs a success message
func LogSuccess(node, module, message string) {
	Logger.Info().
		Str("node", node).
		Str("module", module).
		Str("type", "success").
		Msg(message)
}

// LogError logs an error message
func LogError(node, module, message string) {
	Logger.Error().
		Str("node", node).
		Str("module", module).
		Str("type", "error").
		Msg(message)
}

// LogWarning logs a warning message
func LogWarning(node, module, message string) {
	Logger.Warn().
		Str("node", node).
		Str("module", module).
		Str("type", "warning").
		Msg(message)
}

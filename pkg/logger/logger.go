package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// LogLevel represents the severity level of a log message
type LogLevel int

const (
	// DEBUG level for detailed debugging information
	DEBUG LogLevel = iota
	// INFO level for general informational messages
	INFO
	// WARN level for warning messages
	WARN
	// ERROR level for error messages
	ERROR
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel converts a string to LogLevel
func ParseLogLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN", "WARNING":
		return WARN
	case "ERROR":
		return ERROR
	default:
		return INFO
	}
}

// Logger is a structured logger with log levels
type Logger struct {
	level   LogLevel
	format  string
	zlogger zerolog.Logger
	mu      sync.RWMutex
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// Init initializes the default logger with the specified log level
func Init(level LogLevel) {
	InitWithFormat(level, "json")
}

// InitWithFormat initializes the default logger with the specified log level and format
func InitWithFormat(level LogLevel, format string) {
	once.Do(func() {
		defaultLogger = newLogger(level, format, os.Stdout)
	})
}

// newLogger creates a new logger instance
func newLogger(level LogLevel, format string, output io.Writer) *Logger {
	// Configure zerolog based on format
	var zlogger zerolog.Logger
	if format == "text" {
		// Use console writer for human-readable text format
		consoleWriter := zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		}
		zlogger = zerolog.New(consoleWriter).With().Timestamp().Logger()
	} else {
		// Use JSON format (default)
		zlogger = zerolog.New(output).With().Timestamp().Logger()
	}

	// Set zerolog level
	zlogger = zlogger.Level(toZerologLevel(level))

	return &Logger{
		level:   level,
		format:  format,
		zlogger: zlogger,
	}
}

// toZerologLevel converts LogLevel to zerolog.Level
func toZerologLevel(level LogLevel) zerolog.Level {
	switch level {
	case DEBUG:
		return zerolog.DebugLevel
	case INFO:
		return zerolog.InfoLevel
	case WARN:
		return zerolog.WarnLevel
	case ERROR:
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// SetLevel sets the log level for the default logger
func SetLevel(level LogLevel) {
	if defaultLogger == nil {
		Init(INFO)
	}
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.level = level
	defaultLogger.zlogger = defaultLogger.zlogger.Level(toZerologLevel(level))
}

// GetLevel returns the current log level
func GetLevel() LogLevel {
	if defaultLogger == nil {
		Init(INFO)
	}
	defaultLogger.mu.RLock()
	defer defaultLogger.mu.RUnlock()
	return defaultLogger.level
}

// log writes a log message if the level is enabled
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	l.mu.RLock()
	zlogger := l.zlogger
	l.mu.RUnlock()

	message := fmt.Sprintf(format, args...)

	switch level {
	case DEBUG:
		zlogger.Debug().Msg(message)
	case INFO:
		zlogger.Info().Msg(message)
	case WARN:
		zlogger.Warn().Msg(message)
	case ERROR:
		zlogger.Error().Msg(message)
	}
}

// logWithFields writes a log message with structured fields
func (l *Logger) logWithFields(level LogLevel, fields map[string]interface{}, format string, args ...interface{}) {
	l.mu.RLock()
	zlogger := l.zlogger
	l.mu.RUnlock()

	message := fmt.Sprintf(format, args...)

	var event *zerolog.Event
	switch level {
	case DEBUG:
		event = zlogger.Debug()
	case INFO:
		event = zlogger.Info()
	case WARN:
		event = zlogger.Warn()
	case ERROR:
		event = zlogger.Error()
	default:
		return
	}

	// Add fields
	for key, value := range fields {
		event = event.Interface(key, value)
	}

	event.Msg(message)
}

// Debug logs a debug message
func Debug(format string, args ...interface{}) {
	if defaultLogger == nil {
		Init(INFO)
	}
	defaultLogger.log(DEBUG, format, args...)
}

// Info logs an info message
func Info(format string, args ...interface{}) {
	if defaultLogger == nil {
		Init(INFO)
	}
	defaultLogger.log(INFO, format, args...)
}

// Warn logs a warning message
func Warn(format string, args ...interface{}) {
	if defaultLogger == nil {
		Init(INFO)
	}
	defaultLogger.log(WARN, format, args...)
}

// Error logs an error message
func Error(format string, args ...interface{}) {
	if defaultLogger == nil {
		Init(INFO)
	}
	defaultLogger.log(ERROR, format, args...)
}

// Debugf is an alias for Debug
func Debugf(format string, args ...interface{}) {
	Debug(format, args...)
}

// Infof is an alias for Info
func Infof(format string, args ...interface{}) {
	Info(format, args...)
}

// Warnf is an alias for Warn
func Warnf(format string, args ...interface{}) {
	Warn(format, args...)
}

// Errorf is an alias for Error
func Errorf(format string, args ...interface{}) {
	Error(format, args...)
}

// DebugWithFields logs a debug message with structured fields
func DebugWithFields(fields map[string]interface{}, format string, args ...interface{}) {
	if defaultLogger == nil {
		Init(INFO)
	}
	defaultLogger.logWithFields(DEBUG, fields, format, args...)
}

// InfoWithFields logs an info message with structured fields
func InfoWithFields(fields map[string]interface{}, format string, args ...interface{}) {
	if defaultLogger == nil {
		Init(INFO)
	}
	defaultLogger.logWithFields(INFO, fields, format, args...)
}

// WarnWithFields logs a warning message with structured fields
func WarnWithFields(fields map[string]interface{}, format string, args ...interface{}) {
	if defaultLogger == nil {
		Init(INFO)
	}
	defaultLogger.logWithFields(WARN, fields, format, args...)
}

// ErrorWithFields logs an error message with structured fields
func ErrorWithFields(fields map[string]interface{}, format string, args ...interface{}) {
	if defaultLogger == nil {
		Init(INFO)
	}
	defaultLogger.logWithFields(ERROR, fields, format, args...)
}

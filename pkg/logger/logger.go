package logger

import (
	"context"
	"log/slog"
	"os"
)

var globalLogger *slog.Logger

func init() {
	Init("cab-booking")
}

// Init configures the process-wide structured logger with a service name.
func Init(serviceName string) {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	globalLogger = slog.New(handler).With("service", serviceName)
	slog.SetDefault(globalLogger)
}

// Get returns default structured logger instance
func Get() *slog.Logger {
	return globalLogger
}

// Info logs formatted info message with context
func Info(ctx context.Context, msg string, args ...any) {
	globalLogger.InfoContext(ctx, msg, args...)
}

// Error logs formatted error message with context
func Error(ctx context.Context, msg string, args ...any) {
	globalLogger.ErrorContext(ctx, msg, args...)
}

// Warn logs formatted warning message with context
func Warn(ctx context.Context, msg string, args ...any) {
	globalLogger.WarnContext(ctx, msg, args...)
}

// Debug logs formatted debug message with context
func Debug(ctx context.Context, msg string, args ...any) {
	globalLogger.DebugContext(ctx, msg, args...)
}

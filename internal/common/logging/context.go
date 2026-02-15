package logging

import (
	"context"

	"go.uber.org/zap"
)

type loggerKey struct{}

// WithContext returns a new context with the given fields zap.Field.
func WithContext(ctx context.Context, fields ...zap.Field) context.Context {
	baseLogger := FromContext(ctx)
	newLogger := baseLogger.With(fields...)
	return context.WithValue(ctx, loggerKey{}, newLogger)
}

// WithRequestID is a helper function to add a request ID to the context for logging.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return WithContext(ctx, zap.String("request_id", requestID))
}

// FromContext retrieves the logger from the context, if available. If not, it returns the global logger.
func FromContext(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return L() // Return the global logger if context is nil
	}
	if logger, ok := ctx.Value(loggerKey{}).(*zap.Logger); ok {
		return logger
	}
	return L() // Return the global logger if no logger is found in context
}

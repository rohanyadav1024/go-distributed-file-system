package logging

import (
	"context"

	"go.uber.org/zap"
)

type loggerKey struct{}

// WithContext returns a context whose logger includes the given fields.
func WithContext(ctx context.Context, fields ...zap.Field) context.Context {
	baseLogger := FromContext(ctx)
	newLogger := baseLogger.With(fields...)
	return context.WithValue(ctx, loggerKey{}, newLogger)
}

// WithRequestID adds a request ID field to the context logger.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return WithContext(ctx, zap.String("request_id", requestID))
}

// FromContext returns the logger stored in context or the global logger.
func FromContext(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return L()
	}
	if logger, ok := ctx.Value(loggerKey{}).(*zap.Logger); ok {
		return logger
	}
	return L()
}

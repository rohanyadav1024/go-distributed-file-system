package logging

import (
	customerrors "github.com/rohanyadav1024/dfs/internal/common/errors"
	"go.uber.org/zap"
)

func WithError(err error) []zap.Field {
	if err == nil {
		return nil
	}

	e := customerrors.From(err)
	fields := []zap.Field{
		zap.String("error_code", string(e.Code)),
		zap.Bool("retryable", e.Retryable()),
		zap.String("error_message", e.Message),
	}
	if e.Cause != nil {
		fields = append(fields, zap.String("error_cause", e.Cause.Error()))
	}
	return fields
}

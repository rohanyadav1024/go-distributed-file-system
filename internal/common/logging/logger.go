package logging

import (
	"strings"

	"go.uber.org/zap"
)

var logger *zap.Logger

// Init initializes the logger with the given configuration.
func Init(cfg Config) error {
	var zapConfig zap.Config
	if cfg.Production {
		zapConfig = zap.NewProductionConfig()
	} else {
		zapConfig = zap.NewDevelopmentConfig()
	}
	
	level := strings.ToLower(cfg.Level)
	switch level {
	case "debug":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		zapConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel) // Default to info level
	}

	l, err := zapConfig.Build()
	if err != nil {
		return err
	}
	logger = l
	return nil
}

// L returns the global logger instance.
func L() *zap.Logger {
	return logger
}

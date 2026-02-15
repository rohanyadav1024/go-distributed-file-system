package config

import (
	"os"
	"strings"

	"github.com/rohanyadav1024/dfs/internal/common/logging"
)

type Config struct {
	ServiceName string
	Log logging.Config
}

func Load() Config {
	return Config{
		ServiceName: getEnv("SERVICE_NAME", "dfs-service"),
		Log: logging.Config{
			Level:      getEnv("LOG_LEVEL", "debug"),
			Production: getEnv("LOG_PRODUCTION", "false") == "true",
		},
	}
}

func getEnv(key string, fallback string) string {
	val := os.Getenv(key)
	if strings.TrimSpace(val) == "" {
		return fallback
	}
	return val
}
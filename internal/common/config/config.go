package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/rohanyadav1024/dfs/internal/common/logging"
)

type Config struct {
	ServiceName string
	Log         logging.Config

	// Metadata
	MetadataDBPath         string
	ReplicationFactor      int
	FailureTimeoutSeconds  int
	MonitorIntervalSeconds int
}

func Load() Config {
	_ = godotenv.Load()

	cfg := Config{
		ServiceName: getEnv("SERVICE_NAME", "dfs-service"),

		Log: logging.Config{
			Level:      getEnv("LOG_LEVEL", "debug"),
			Production: parseBool(getEnv("LOG_PRODUCTION", "false")),
		},

		MetadataDBPath:         getEnv("DFS_METADATA_DB_PATH", "./data/metad/metad.db"),
		ReplicationFactor:      parseInt(getEnv("DFS_REPLICATION_FACTOR", "2")),
		FailureTimeoutSeconds:  parseInt(getEnv("DFS_FAILURE_TIMEOUT_SECONDS", "10")),
		MonitorIntervalSeconds: parseInt(getEnv("DFS_MONITOR_INTERVAL_SECONDS", "3")),
	}

	return cfg
}

func getEnv(key string, fallback string) string {
	val := os.Getenv(key)
	if strings.TrimSpace(val) == "" {
		return fallback
	}
	return val
}

func parseBool(val string) bool {
	b, err := strconv.ParseBool(val)
	if err != nil {
		return false
	}
	return b
}

func parseInt(val string) int {
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}
	return i
}

// EnsureDir ensures parent directory for a file path exists.
func EnsureDir(filePath string) error {
	dir := filepath.Dir(filePath)
	return os.MkdirAll(dir, 0755)
}
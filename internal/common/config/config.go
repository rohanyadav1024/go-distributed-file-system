// Package config loads runtime configuration for DFS services.
package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/rohanyadav1024/dfs/internal/common/logging"
	"github.com/rohanyadav1024/dfs/internal/constants"
)

// Config contains service settings loaded from environment variables.
type Config struct {
	ServiceName string
	Log         logging.Config

	// Metadata
	MetadataDBPath         string
	MetadataAddr           string
	MetadataMetricsAddr    string
	ReplicationFactor      int
	FailureTimeoutSeconds  int
	MonitorIntervalSeconds int

	// Storage
	StorageDataPath      string
	StorageListenAddr    string
	StorageMetricsAddr   string
	StorageNodeID        string
	StorageCapacityBytes int64
}

// Load reads environment variables and returns a populated Config.
func Load() Config {
	_ = godotenv.Load()

	cfg := Config{
		ServiceName: getEnv("SERVICE_NAME", "dfs-service"),

		Log: logging.Config{
			Level:      getEnv("LOG_LEVEL", "debug"),
			Production: parseBool(getEnv("LOG_PRODUCTION", "false")),
		},

		MetadataDBPath:         getEnv(constants.EnvMetadataDBPath, "./data/metad/metad.db"),
		MetadataAddr:           getEnv(constants.EnvMetadataAddr, ":50051"),
		MetadataMetricsAddr:    getEnv("DFS_METRICS_ADDR", ":9090"),
		ReplicationFactor:      parseInt(getEnv(constants.EnvReplicationFactor, "2")),
		FailureTimeoutSeconds:  parseInt(getEnv("DFS_FAILURE_TIMEOUT_SECONDS", "10")),
		MonitorIntervalSeconds: parseInt(getEnv("DFS_MONITOR_INTERVAL_SECONDS", "3")),
		StorageDataPath:        getEnv("DFS_STORAGE_DATA_PATH", "./data/storaged"),
		StorageListenAddr:      getEnv(constants.EnvStorageListenAddr, ":50052"),
		StorageMetricsAddr:     getEnv("DFS_STORAGE_METRICS_ADDR", ":9091"),
		StorageNodeID:          getEnv(constants.EnvStorageNodeID, "storage-1"),
		StorageCapacityBytes:   int64(parseInt(getEnv("DFS_STORAGE_CAPACITY_BYTES", "10737418240"))), // 10GB default
	}

	return cfg
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func parseBool(value string) bool {
	b, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}
	return b
}

func parseInt(value string) int {
	i, err := strconv.Atoi(value)
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

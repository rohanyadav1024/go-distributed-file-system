package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rohanyadav1024/dfs/internal/common/config"
	"github.com/rohanyadav1024/dfs/internal/common/ids"
	"github.com/rohanyadav1024/dfs/internal/common/logging"
	"github.com/rohanyadav1024/dfs/internal/metadata/manifest"
	"github.com/rohanyadav1024/dfs/internal/metadata/placement"
	"github.com/rohanyadav1024/dfs/internal/metadata/registry"
	"github.com/rohanyadav1024/dfs/internal/metadata/store"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize structured logger
	if err := logging.Init(cfg.Log); err != nil {
		panic(err)
	}

	// Root lifecycle context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Attach request ID for startup traceability
	ctx = logging.WithRequestID(ctx, ids.NewRequestID())

	log := logging.FromContext(ctx)
	log.Info("metad starting up")

// ----------------------------
// Initialize metadata store
// ----------------------------

// Ensure DB directory exists
if err := config.EnsureDir(cfg.MetadataDBPath); err != nil {
	log.Fatal("failed to create metadata directory", logging.WithError(err)...)
}

metaStore, err := store.NewSQLite(cfg.MetadataDBPath)
if err != nil {
	log.Fatal("failed to initialize metadata store", logging.WithError(err)...)
}
defer metaStore.Close()

// ----------------------------
// Initialize registry
// ----------------------------

failureTimeout := time.Duration(cfg.FailureTimeoutSeconds) * time.Second
reg := registry.NewManager(metaStore, failureTimeout)

monitorInterval := time.Duration(cfg.MonitorIntervalSeconds) * time.Second
reg.StartMonitor(ctx, monitorInterval)

// ----------------------------
// Initialize placement engine
// ----------------------------

place := placement.NewEngine(cfg.ReplicationFactor)

// ----------------------------
// Initialize manifest manager
// ----------------------------

_ = manifest.NewManager(metaStore, reg, place)


	// ----------------------------
	// Initialize manifest manager
	// ----------------------------

	_ = manifest.NewManager(metaStore, reg, place)

	log.Info("metad bootstrap complete")

	// ----------------------------
	// Graceful shutdown handling
	// ----------------------------

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh

	log.Info("metad shutting down")

	cancel() // stop background workers

	_ = logging.L().Sync()
}

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"net"

	"github.com/rohanyadav1024/dfs/internal/common/config"
	"github.com/rohanyadav1024/dfs/internal/common/ids"
	"github.com/rohanyadav1024/dfs/internal/common/logging"
	"github.com/rohanyadav1024/dfs/internal/metadata/manifest"
	"github.com/rohanyadav1024/dfs/internal/metadata/placement"
	"github.com/rohanyadav1024/dfs/internal/metadata/policy"
	"github.com/rohanyadav1024/dfs/internal/metadata/registry"
	"github.com/rohanyadav1024/dfs/internal/metadata/repair"
	"github.com/rohanyadav1024/dfs/internal/metadata/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	metadatapb "github.com/rohanyadav1024/dfs/internal/protocol/metadata"

	nodepb "github.com/rohanyadav1024/dfs/internal/protocol/node"
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
	// Initialize replication repair manager
	// ----------------------------

	repairManager := repair.NewManager(metaStore, reg, cfg.ReplicationFactor, logging.L())
	repairManager.StartScanner(ctx, 5*time.Second)

	// ----------------------------
	// Initialize chunk size policy
	// ----------------------------

	chunkPolicy := policy.NewPolicy()
	chunkSize, err := chunkPolicy.DetermineChunkSize("", 0)
	if err != nil {
		log.Fatal("failed to determine chunk size", logging.WithError(err)...)
	}

	// ----------------------------
	// Initialize manifest manager
	// ----------------------------

	manifestManager := manifest.NewManager(metaStore, reg, place, chunkSize)
	registryManager := registry.NewManager(metaStore, failureTimeout)

	log.Info("metad bootstrap complete")

	// ----------------------------
	// Start gRPC Server
	// ----------------------------

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal("failed to listen", logging.WithError(err)...)
	}

	grpcServer := grpc.NewServer()

	// Register gRPC services
	reflection.Register(grpcServer)

	metadatapb.RegisterMetadataServiceServer(grpcServer, &metadataServer{
		manifest: manifestManager,
	})
	// nodepb.RegisterNodeServiceServer(grpcServer, &nodeServer{})
	nodepb.RegisterNodeServiceServer(grpcServer, &nodeServer{
		registry: registryManager,
	})

	log.Info("gRPC server listening on :50051")

	// Run gRPC server in background
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal("failed to serve gRPC", logging.WithError(err)...)
		}
	}()

	// ----------------------------
	// Graceful shutdown handling
	// ----------------------------

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh

	log.Info("metad shutting down")

	// Stop background workers
	cancel()

	// Gracefully stop gRPC server (finish in-flight RPCs)
	grpcServer.GracefulStop()

	// Close listener explicitly (defensive cleanup)
	_ = lis.Close()

	// Flush logger buffers
	_ = logging.L().Sync()
}

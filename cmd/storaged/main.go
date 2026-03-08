package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/rohanyadav1024/dfs/internal/common/config"
	"github.com/rohanyadav1024/dfs/internal/common/ids"
	"github.com/rohanyadav1024/dfs/internal/common/logging"
	nodepb "github.com/rohanyadav1024/dfs/internal/protocol/node"
	storagepb "github.com/rohanyadav1024/dfs/internal/protocol/storage"
	"github.com/rohanyadav1024/dfs/internal/storage/chunkstore"
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
	log.Info("storaged starting up")

	log.Info("storaged starting up",
		zap.String("node_id", cfg.StorageNodeID),
		zap.String("data_path", cfg.StorageDataPath),
		zap.String("listen_addr", cfg.StorageListenAddr),
	)

	// ----------------------------
	// Initialize chunkstore (before metad client)
	// ----------------------------

	store, err := chunkstore.New(cfg.StorageDataPath, cfg.StorageCapacityBytes)
	if err != nil {
		log.Fatal("failed to initialize chunkstore", logging.WithError(err)...)
	}

	// ----------------------------
	// Initialize metad client for heartbeats
	// ----------------------------

	metadClient, err := newMetadClient(
		ctx,
		logging.L(),
		cfg.StorageNodeID,
		cfg.StorageListenAddr,
		cfg.MetadataAddr,
		store,
	)
	if err != nil {
		log.Fatal("failed to connect to metad", logging.WithError(err)...)
	}
	defer metadClient.close()

	// Start background heartbeat loop (every 3 seconds)
	metadClient.startHeartbeat(ctx, logging.L(), 3*time.Second)

	log.Info("storaged bootstrap complete")

	// ----------------------------
	// Start gRPC Server
	// ----------------------------

	lis, err := net.Listen("tcp", cfg.StorageListenAddr)
	if err != nil {
		log.Fatal("failed to listen", logging.WithError(err)...)
	}

	grpcServer := grpc.NewServer()

	// Register gRPC services and reflection for debugging
	reflection.Register(grpcServer)

	storagepb.RegisterStorageServiceServer(grpcServer, &storageServer{
		store: store,
	})

	nodepb.RegisterNodeServiceServer(grpcServer, &nodeServer{
		store: store,
	})

	log.Info("gRPC server listening",
		zap.String("node_id", cfg.StorageNodeID),
		zap.String("listen_addr", cfg.StorageListenAddr),
	)

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

	log.Info("storaged shutting down")

	// Stop background workers
	cancel()

	// Gracefully stop gRPC server (finish in-flight RPCs)
	grpcServer.GracefulStop()

	// Close listener explicitly (defensive cleanup)
	_ = lis.Close()

	// Flush logger buffers
	_ = logging.L().Sync()
}

package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/rohanyadav1024/dfs/internal/common/config"
	"github.com/rohanyadav1024/dfs/internal/common/ids"
	"github.com/rohanyadav1024/dfs/internal/common/logging"
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
	// Initialize chunkstore
	// ----------------------------

	store, err := chunkstore.New(cfg.StorageDataPath)
	if err != nil {
		log.Fatal("failed to initialize chunkstore", logging.WithError(err)...)
	}

	log.Info("storaged bootstrap complete")

	// ----------------------------
	// Start gRPC Server
	// ----------------------------

	lis, err := net.Listen("tcp", cfg.StorageListenAddr)
	if err != nil {
		log.Fatal("failed to listen", logging.WithError(err)...)
	}

	grpcServer := grpc.NewServer()

	storagepb.RegisterStorageServiceServer(grpcServer, &storageServer{
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

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	"github.com/rohanyadav1024/dfs/internal/common/config"
	"github.com/rohanyadav1024/dfs/internal/common/ids"
	"github.com/rohanyadav1024/dfs/internal/common/logging"
	nodepb "github.com/rohanyadav1024/dfs/internal/protocol/node"
	storagepb "github.com/rohanyadav1024/dfs/internal/protocol/storage"
	"github.com/rohanyadav1024/dfs/internal/storage/chunkstore"
	storagemetrics "github.com/rohanyadav1024/dfs/internal/storage/metrics"
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
	storagemetrics.Register()

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
	storagemetrics.SetAvailableBytes(store.AvailableBytes())

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

	serverCert, err := tls.LoadX509KeyPair("/certs/server.crt", "/certs/server.key")
	if err != nil {
		log.Fatal("failed to load server certificate/key", logging.WithError(err)...)
	}

	caCertPEM, err := os.ReadFile("/certs/ca.crt")
	if err != nil {
		log.Fatal("failed to read CA certificate", logging.WithError(err)...)
	}

	caPool := x509.NewCertPool()
	if ok := caPool.AppendCertsFromPEM(caCertPEM); !ok {
		log.Fatal("failed to append CA certificate to pool")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}

	grpcServer := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))

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

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsServer := &http.Server{
		Addr:    cfg.StorageMetricsAddr,
		Handler: metricsMux,
	}

	go func() {
		log.Info("metrics server listening",
			zap.String("node_id", cfg.StorageNodeID),
			zap.String("listen_addr", cfg.StorageMetricsAddr),
		)
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("failed to serve metrics", logging.WithError(err)...)
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

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		log.Warn("failed to shutdown metrics server cleanly", logging.WithError(err)...)
	}

	// Close listener explicitly (defensive cleanup)
	_ = lis.Close()

	// Flush logger buffers
	_ = logging.L().Sync()
}

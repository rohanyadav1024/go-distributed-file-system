// Package main runs the metadata daemon process.
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
	"github.com/rohanyadav1024/dfs/internal/auth"
	"github.com/rohanyadav1024/dfs/internal/common/config"
	"github.com/rohanyadav1024/dfs/internal/common/ids"
	"github.com/rohanyadav1024/dfs/internal/common/logging"
	"github.com/rohanyadav1024/dfs/internal/constants"
	metadupdater "github.com/rohanyadav1024/dfs/internal/metadata"
	"github.com/rohanyadav1024/dfs/internal/metadata/manifest"
	metadmetrics "github.com/rohanyadav1024/dfs/internal/metadata/metrics"
	"github.com/rohanyadav1024/dfs/internal/metadata/placement"
	"github.com/rohanyadav1024/dfs/internal/metadata/policy"
	"github.com/rohanyadav1024/dfs/internal/metadata/registry"
	"github.com/rohanyadav1024/dfs/internal/metadata/repair"
	"github.com/rohanyadav1024/dfs/internal/metadata/store"
	servermetrics "github.com/rohanyadav1024/dfs/internal/metrics"
	nodeclient "github.com/rohanyadav1024/dfs/internal/node"

	metadatapb "github.com/rohanyadav1024/dfs/internal/protocol/metadata"
	nodepb "github.com/rohanyadav1024/dfs/internal/protocol/node"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

// main initializes dependencies and serves metadata gRPC and metrics endpoints.
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

	jwtSecret := os.Getenv(constants.EnvJWTSecret)
	if jwtSecret == "" {
		panic(constants.EnvJWTSecret + " is required and cannot be empty")
	}

	metadmetrics.Register()

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

	metricsUpdater := metadupdater.NewMetricsUpdater(metaStore)
	go metricsUpdater.Start(ctx)

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

	nodeClient := &nodeclient.Client{}
	repairManager := repair.NewManager(
		metaStore,
		reg,
		cfg.ReplicationFactor,
		logging.FromContext(ctx),
		nodeClient,
	)
	repairManager.StartScanner(ctx, constants.DefaultRepairScanInterval)

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

	log.Info("metad bootstrap complete")

	// ----------------------------
	// Start gRPC Server
	// ----------------------------

	lis, err := net.Listen("tcp", ":50051")
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

	grpcServer := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(tlsConfig)),
		grpc.ChainUnaryInterceptor(
			auth.UnaryServerInterceptor(jwtSecret),
			servermetrics.UnaryMetricsInterceptor(),
		),
	)

	// Register gRPC services
	reflection.Register(grpcServer)

	metadatapb.RegisterMetadataServiceServer(grpcServer, &metadataServer{
		manifest: manifestManager,
	})
	nodepb.RegisterNodeServiceServer(grpcServer, &nodeServer{
		registry: reg,
	})

	log.Info("gRPC server listening on :50051")

	// Run gRPC server in background
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal("failed to serve gRPC", logging.WithError(err)...)
		}
	}()

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsServer := &http.Server{
		Addr:    cfg.MetadataMetricsAddr,
		Handler: metricsMux,
	}

	go func() {
		log.Info("metrics server listening", zap.String("addr", cfg.MetadataMetricsAddr))
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

	log.Info("metad shutting down")

	// Stop background workers
	cancel()

	// Gracefully stop gRPC server (finish in-flight RPCs)
	grpcServer.GracefulStop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), constants.DefaultNodeTimeout)
	defer shutdownCancel()
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		log.Warn("failed to shutdown metrics server cleanly", logging.WithError(err)...)
	}

	// Close listener explicitly (defensive cleanup)
	_ = lis.Close()

	// Flush logger buffers
	_ = logging.L().Sync()
}

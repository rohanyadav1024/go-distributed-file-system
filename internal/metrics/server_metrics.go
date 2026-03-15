package metrics

import (
	"context"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var MetadataRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "dfs",
		Subsystem: "grpc",
		Name:      "request_duration_seconds",
		Help:      "Latency of gRPC requests.",
		Buckets: []float64{
			0.001, 0.005, 0.01,
			0.05, 0.1, 0.25,
			0.5, 1, 2, 5,
		},
	},
	[]string{"method"},
)

var (
	ClusterHealthyNodes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "dfs",
			Subsystem: "cluster",
			Name:      "healthy_nodes",
			Help:      "Number of healthy storage nodes.",
		},
	)

	ClusterTotalNodes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "dfs",
			Subsystem: "cluster",
			Name:      "total_nodes",
			Help:      "Total registered storage nodes.",
		},
	)

	ClusterTotalChunks = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "dfs",
			Subsystem: "cluster",
			Name:      "total_chunks",
			Help:      "Total number of chunks in metadata.",
		},
	)

	ClusterTotalReplicas = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "dfs",
			Subsystem: "cluster",
			Name:      "total_replicas",
			Help:      "Total number of chunk replicas in metadata.",
		},
	)
)

func init() {
	prometheus.MustRegister(
		MetadataRequestDuration,
		ClusterHealthyNodes,
		ClusterTotalNodes,
		ClusterTotalChunks,
		ClusterTotalReplicas,
	)
}

func UnaryMetricsInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req)
		duration := time.Since(start).Seconds()

		parts := strings.Split(info.FullMethod, "/")
		methodName := parts[len(parts)-1]

		MetadataRequestDuration.
			WithLabelValues(methodName).
			Observe(duration)

		if err != nil {
			_ = status.Code(err)
		}

		return resp, err
	}
}

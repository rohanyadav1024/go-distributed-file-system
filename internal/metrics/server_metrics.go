// Package metrics exposes gRPC and cluster-level server Prometheus metrics.
package metrics

import (
	"context"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rohanyadav1024/dfs/internal/constants"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var MetadataRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: constants.MetricNamespaceDFS,
		Subsystem: constants.MetricSubsystemGRPC,
		Name:      constants.MetricNameRequestDurationSeconds,
		Help:      "Latency of gRPC requests.",
		Buckets: []float64{
			0.001, 0.005, 0.01,
			0.05, 0.1, 0.25,
			0.5, 1, 2, 5,
		},
	},
	[]string{constants.MetricLabelMethod},
)

var (
	ClusterHealthyNodes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: constants.MetricNamespaceDFS,
			Subsystem: constants.MetricSubsystemCluster,
			Name:      constants.MetricNameHealthyNodes,
			Help:      "Number of healthy storage nodes.",
		},
	)

	ClusterTotalNodes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: constants.MetricNamespaceDFS,
			Subsystem: constants.MetricSubsystemCluster,
			Name:      constants.MetricNameTotalNodes,
			Help:      "Total registered storage nodes.",
		},
	)

	ClusterTotalChunks = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: constants.MetricNamespaceDFS,
			Subsystem: constants.MetricSubsystemCluster,
			Name:      constants.MetricNameTotalChunks,
			Help:      "Total number of chunks in metadata.",
		},
	)

	ClusterTotalReplicas = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: constants.MetricNamespaceDFS,
			Subsystem: constants.MetricSubsystemCluster,
			Name:      constants.MetricNameTotalReplicas,
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

// UnaryMetricsInterceptor records per-method latency for unary gRPC requests.
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

		methodName := info.FullMethod
		switch {
		case strings.HasPrefix(info.FullMethod, constants.MetadataServicePrefix):
			methodName = strings.TrimPrefix(info.FullMethod, constants.MetadataServicePrefix)
		case strings.HasPrefix(info.FullMethod, constants.NodeServicePrefix):
			methodName = strings.TrimPrefix(info.FullMethod, constants.NodeServicePrefix)
		default:
			parts := strings.Split(info.FullMethod, "/")
			methodName = parts[len(parts)-1]
		}

		MetadataRequestDuration.
			WithLabelValues(methodName).
			Observe(duration)

		if err != nil {
			_ = status.Code(err)
		}

		return resp, err
	}
}

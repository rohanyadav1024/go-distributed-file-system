// Package metadata provides metadata-service helpers shared at process startup.
package metadata

import (
	"context"
	"time"

	"github.com/rohanyadav1024/dfs/internal/constants"
	"github.com/rohanyadav1024/dfs/internal/metadata/store"
	servermetrics "github.com/rohanyadav1024/dfs/internal/metrics"
)

// MetricsUpdater periodically refreshes cluster-level metadata metrics.
type MetricsUpdater struct {
	store *store.SQLiteStore
}

// NewMetricsUpdater builds a periodic updater for cluster gauge metrics.
func NewMetricsUpdater(store *store.SQLiteStore) *MetricsUpdater {
	return &MetricsUpdater{store: store}
}

// Start polls metadata counters and updates Prometheus gauges until context ends.
func (m *MetricsUpdater) Start(ctx context.Context) {
	ticker := time.NewTicker(constants.DefaultMetricsPollInterval)
	defer ticker.Stop()

	update := func() {
		if totalNodes, err := m.store.CountTotalNodes(ctx); err == nil {
			servermetrics.ClusterTotalNodes.Set(float64(totalNodes))
		}
		if healthyNodes, err := m.store.CountHealthyNodes(ctx); err == nil {
			servermetrics.ClusterHealthyNodes.Set(float64(healthyNodes))
		}
		if totalChunks, err := m.store.CountTotalChunks(ctx); err == nil {
			servermetrics.ClusterTotalChunks.Set(float64(totalChunks))
		}
		if totalReplicas, err := m.store.CountTotalReplicas(ctx); err == nil {
			servermetrics.ClusterTotalReplicas.Set(float64(totalReplicas))
		}
	}

	update()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			update()
		}
	}
}

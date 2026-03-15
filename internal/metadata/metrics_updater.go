package metadata

import (
	"context"
	"time"

	"github.com/rohanyadav1024/dfs/internal/metadata/store"
	servermetrics "github.com/rohanyadav1024/dfs/internal/metrics"
)

type MetricsUpdater struct {
	store *store.SQLiteStore
}

func NewMetricsUpdater(store *store.SQLiteStore) *MetricsUpdater {
	return &MetricsUpdater{store: store}
}

func (m *MetricsUpdater) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
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

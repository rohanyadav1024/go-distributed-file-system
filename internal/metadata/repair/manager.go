package repair

import (
	"context"
	"time"

	"github.com/rohanyadav1024/dfs/internal/metadata/registry"
	"github.com/rohanyadav1024/dfs/internal/metadata/store"
	"go.uber.org/zap"
)

// Manager handles replication maintenance and repair logic.
type Manager struct {
	store             store.Store
	registry          *registry.Manager
	replicationFactor int
	logger            *zap.Logger
}

// NewManager creates a new replication repair manager.
func NewManager(
	s store.Store,
	reg *registry.Manager,
	replicationFactor int,
	log *zap.Logger,
) *Manager {
	return &Manager{
		store:             s,
		registry:          reg,
		replicationFactor: replicationFactor,
		logger:            log,
	}
}

// StartScanner launches a goroutine that periodically scans for under-replicated chunks.
func (m *Manager) StartScanner(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				m.logger.Info("stopping replication scanner")
				return

			case <-ticker.C:
				m.scanOnce(ctx)
			}
		}
	}()
}

// scanOnce performs one cycle of under-replication detection.
func (m *Manager) scanOnce(ctx context.Context) {
	// ToDo: Optimize db calls and if possible use some mutable cache to track changes instead of scanning everything every time.
	// List all chunks in the system
	chunks, err := m.store.ListAllChunks(ctx)
	if err != nil {
		m.logger.Warn("failed to list chunks",
			zap.Error(err),
		)
		return
	}

	// Check each chunk for under-replication
	for _, chunk := range chunks {
		// Get all replica locations for this chunk
		locations, err := m.store.GetChunkLocations(ctx, chunk.ChunkID)
		if err != nil {
			m.logger.Warn("failed to get chunk locations",
				zap.String("chunk_id", chunk.ChunkID),
				zap.Error(err),
			)
			continue
		}

		// Count healthy replicas
		healthyCount := 0
		for _, loc := range locations {
			healthy, err := m.registry.IsNodeHealthy(ctx, loc.NodeID)
			if err != nil {
				m.logger.Warn("failed to check node health",
					zap.String("node_id", loc.NodeID),
					zap.Error(err),
				)
				continue
			}
			if healthy {
				healthyCount++
			}
		}

		// Check if under-replicated
		if healthyCount < m.replicationFactor {
			m.logger.Warn("under-replicated chunk detected",
				zap.String("chunk_id", chunk.ChunkID),
				zap.Int("healthy_replicas", healthyCount),
				zap.Int("required_replicas", m.replicationFactor),
			)
		}
	}
}

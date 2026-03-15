// Package repair detects and heals under-replicated chunks.
package repair

import (
	"context"
	"time"

	"github.com/rohanyadav1024/dfs/internal/metadata/metrics"
	"github.com/rohanyadav1024/dfs/internal/metadata/registry"
	"github.com/rohanyadav1024/dfs/internal/metadata/store"
	nodeclient "github.com/rohanyadav1024/dfs/internal/node"
	"go.uber.org/zap"
)

// Manager handles replication maintenance and repair logic.
type Manager struct {
	store             store.Store
	registry          *registry.Manager
	replicationFactor int
	logger            *zap.Logger
	nodeClient        *nodeclient.Client
}

// NewManager creates a new replication repair manager.
func NewManager(
	s store.Store,
	reg *registry.Manager,
	replicationFactor int,
	log *zap.Logger,
	nc *nodeclient.Client,
) *Manager {
	return &Manager{
		store:             s,
		registry:          reg,
		replicationFactor: replicationFactor,
		logger:            log,
		nodeClient:        nc,
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
	chunks, err := m.store.ListCommittedChunks(ctx)
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
			isHealthy, err := m.registry.IsNodeHealthy(ctx, loc.NodeID)
			if err != nil {
				m.logger.Warn("failed to check node health",
					zap.String("node_id", loc.NodeID),
					zap.Error(err),
				)
				continue
			}
			if isHealthy {
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

			if err := m.repairChunk(ctx, chunk.ChunkID, locations); err != nil {
				m.logger.Warn("repair attempt failed",
					zap.String("chunk_id", chunk.ChunkID),
					zap.Error(err),
				)
			}
		}
	}
}

func (m *Manager) repairChunk(ctx context.Context, chunkID string, locations []store.ChunkLocation) error {
	type replica struct {
		NodeID  string
		Address string
	}

	var healthyReplicas []replica
	for _, loc := range locations {
		isHealthy, err := m.registry.IsNodeHealthy(ctx, loc.NodeID)
		if err != nil {
			m.logger.Warn("failed to check node health during repair",
				zap.String("node_id", loc.NodeID),
				zap.Error(err),
			)
			continue
		}
		if isHealthy {
			// Fetch node details from healthy node list
			healthyNodes, err := m.registry.ListHealthyNodes(ctx)
			if err != nil {
				m.logger.Warn("failed to list healthy nodes",
					zap.Error(err),
				)
				continue
			}

			for _, n := range healthyNodes {
				if n.NodeID == loc.NodeID {
					healthyReplicas = append(healthyReplicas, replica{
						NodeID:  n.NodeID,
						Address: n.Address,
					})
					break
				}
			}
		}
	}

	if len(healthyReplicas) == 0 {
		m.logger.Warn("no healthy replicas available for repair",
			zap.String("chunk_id", chunkID),
		)
		return nil
	}

	healthyNodes, err := m.registry.ListHealthyNodes(ctx)
	if err != nil {
		m.logger.Warn("failed to list healthy nodes for repair",
			zap.Error(err),
		)
		return err
	}

	hasReplicaOnNode := make(map[string]bool)
	for _, loc := range locations {
		hasReplicaOnNode[loc.NodeID] = true
	}

	var targetNode *store.Node
	for _, n := range healthyNodes {
		if !hasReplicaOnNode[n.NodeID] {
			targetNode = &n
			break
		}
	}

	if targetNode == nil {
		m.logger.Warn("no target node available for repair",
			zap.String("chunk_id", chunkID),
		)
		return nil
	}

	source := healthyReplicas[0]
	target := targetNode

	m.logger.Info("repair started",
		zap.String("chunk_id", chunkID),
		zap.String("source_node", source.NodeID),
		zap.String("target_node", target.NodeID),
	)
	metrics.IncRepairAttempts()

	err = m.nodeClient.CopyChunk(ctx, target.Address, source.Address, chunkID)
	if err != nil {
		metrics.IncRepairFailures()
		m.logger.Error("repair failed",
			zap.String("chunk_id", chunkID),
			zap.Error(err),
		)
		return err
	}

	err = m.store.AddChunkLocation(ctx, chunkID, target.NodeID)
	if err != nil {
		metrics.IncRepairFailures()
		m.logger.Error("failed to update metadata after repair",
			zap.String("chunk_id", chunkID),
			zap.Error(err),
		)
		return err
	}

	m.logger.Info("repair completed",
		zap.String("chunk_id", chunkID),
		zap.String("new_replica_node", target.NodeID),
	)
	metrics.IncRepairSuccess()

	return nil
}

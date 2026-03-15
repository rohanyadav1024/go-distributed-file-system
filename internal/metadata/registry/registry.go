// Package registry manages storage node liveness and health state.
package registry

import (
	"context"
	"fmt"
	"time"

	"github.com/rohanyadav1024/dfs/internal/constants"
	"github.com/rohanyadav1024/dfs/internal/metadata/metrics"
	"github.com/rohanyadav1024/dfs/internal/metadata/store"
)

// Manager manages storage node lifecycle and health transitions.
type Manager struct {
	store          store.Store
	failureTimeout time.Duration
}

// NewManager creates a new registry manager.
func NewManager(
	s store.Store,
	failureTimeout time.Duration,
) *Manager {
	return &Manager{
		store:          s,
		failureTimeout: failureTimeout,
	}
}

// RegisterNode registers a new storage node in the system.
func (m *Manager) RegisterNode(
	ctx context.Context,
	nodeID string,
	address string,
	capacityBytes int64,
) error {

	if nodeID == "" {
		return fmt.Errorf("nodeID cannot be empty")
	}

	return m.store.RegisterNode(ctx, store.Node{
		NodeID:         nodeID,
		Address:        address,
		CapacityBytes:  capacityBytes,
		AvailableBytes: capacityBytes,
		Status:         constants.NodeStatusHealthy,
		LastHeartbeat:  time.Now().Unix(),
	})
}

// Heartbeat updates node liveness timestamp and node metadata.
func (m *Manager) Heartbeat(
	ctx context.Context,
	nodeID string,
	address string,
	capacityBytes int64,
	availableBytes int64,
) error {
	if nodeID == "" {
		return fmt.Errorf("nodeID cannot be empty")
	}

	return m.store.UpsertNodeHeartbeat(ctx, nodeID, address, capacityBytes, availableBytes)
}

// ListHealthyNodes returns all currently healthy nodes.
func (m *Manager) ListHealthyNodes(ctx context.Context) ([]store.Node, error) {
	return m.store.ListHealthyNodes(ctx)
}

// IsNodeHealthy checks if a node is currently healthy.
func (m *Manager) IsNodeHealthy(ctx context.Context, nodeID string) (bool, error) {
	if nodeID == "" {
		return false, fmt.Errorf("nodeID cannot be empty")
	}

	nodes, err := m.store.ListHealthyNodes(ctx)
	if err != nil {
		return false, err
	}

	for _, node := range nodes {
		if node.NodeID == nodeID {
			return true, nil
		}
	}

	return false, nil
}

// StartMonitor runs periodic health checks until the context is canceled.
func (m *Manager) StartMonitor(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.monitorOnce(ctx)
			}
		}
	}()
}

func (m *Manager) monitorOnce(ctx context.Context) {

	nodes, err := m.store.ListAllNodes(ctx)
	if err != nil {
		return
	}

	now := time.Now()

	for _, node := range nodes {

		last := time.Unix(node.LastHeartbeat, 0)
		stale := now.Sub(last) > m.failureTimeout

		if stale && node.Status == constants.NodeStatusHealthy {
			_ = m.store.UpdateNodeStatus(ctx, node.NodeID, constants.NodeStatusDown)
		}

		if !stale && node.Status == constants.NodeStatusDown {
			_ = m.store.UpdateNodeStatus(ctx, node.NodeID, constants.NodeStatusHealthy)
		}
	}

	healthyNodes, err := m.store.ListHealthyNodes(ctx)
	if err == nil {
		metrics.SetHealthyNodes(len(healthyNodes))
	}
}

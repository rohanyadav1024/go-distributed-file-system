package registry

import (
	"context"
	"fmt"
	"time"

	"github.com/rohanyadav1024/dfs/internal/metadata/store"
)

// Manager handles storage node lifecycle and health logic.
// It wraps the store layer and adds domain-level rules.
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
	capacityBytes int64,
) error {

	if nodeID == "" {
		return fmt.Errorf("nodeID cannot be empty")
	}

	return m.store.RegisterNode(ctx, store.Node{
		NodeID:         nodeID,
		CapacityBytes:  capacityBytes,
		AvailableBytes: capacityBytes,
		Status:         "healthy",
		LastHeartbeat:  time.Now().Unix(),
	})
}

// Heartbeat updates node liveness timestamp.
func (m *Manager) Heartbeat(ctx context.Context, nodeID string) error {
	if nodeID == "" {
		return fmt.Errorf("nodeID cannot be empty")
	}

	now := time.Now().Unix()

	// Update heartbeat timestamp
	if err := m.store.UpdateNodeHeartbeat(ctx, nodeID, now); err != nil {
		return err
	}

	// Ensure node is marked healthy (recovery case)
	if err := m.store.UpdateNodeStatus(ctx, nodeID, "healthy"); err != nil {
		return err
	}

	return nil
}

// ListHealthyNodes returns all currently healthy nodes.
func (m *Manager) ListHealthyNodes(ctx context.Context) ([]store.Node, error) {
	return m.store.ListHealthyNodes(ctx)
}

func (m *Manager) StartMonitor(ctx context.Context, interval time.Duration) {
	go func() {
		// Create a ticker that sends a signal on ticker.C
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Infinite loop for continuous monitoring.
		// This loop only exits when context is cancelled.
		for {
			// select waits for one of multiple channel events.
			// It blocks until one case becomes ready.
			select {

			// Case 1: Context cancellation signal.
			// ctx.Done() is closed when:
			//   - Application is shutting down
			//   - Parent context times out
			//   - Manual cancellation occurs
			//
			// When this happens, we exit immediately to ensure graceful shutdown.
			case <-ctx.Done():
				return

			// Case 2: Ticker event.
			// ticker.C receives a value every 'interval'. This triggers one round of failure detection.
			case <-ticker.C:

				// Run one failure detection cycle.
				// This checks heartbeat timestamps
				// and updates node statuses if needed.
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

		if stale && node.Status == "healthy" {
			_ = m.store.UpdateNodeStatus(ctx, node.NodeID, "down")
		}

		if !stale && node.Status == "down" {
			_ = m.store.UpdateNodeStatus(ctx, node.NodeID, "healthy")
		}
	}
}
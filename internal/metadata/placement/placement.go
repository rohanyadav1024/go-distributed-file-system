// Package placement assigns chunk replicas to healthy storage nodes.
package placement

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/rohanyadav1024/dfs/internal/metadata/store"
)

// Engine handles replica placement logic.
type Engine struct {
	replicationFactor int
}

// NewEngine creates a new placement engine.
func NewEngine(replicationFactor int) *Engine {
	return &Engine{
		replicationFactor: replicationFactor,
	}
}

// ReplicationFactor returns configured replication factor.
func (e *Engine) ReplicationFactor() int {
	return e.replicationFactor
}

// SelectReplicas selects R unique healthy nodes for each chunk.
// Returns: chunkID -> []nodeID
func (e *Engine) SelectReplicas(
	ctx context.Context,
	chunks []store.Chunk,
	healthyNodes []store.Node,
) (map[string][]string, error) {

	if e.replicationFactor <= 0 {
		return nil, fmt.Errorf("invalid replication factor")
	}

	if len(healthyNodes) < e.replicationFactor {
		return nil, fmt.Errorf("not enough healthy nodes for replication")
	}

	rand.Seed(time.Now().UnixNano())
	result := make(map[string][]string)

	for _, chunk := range chunks {
		// Create copy of node slice
		nodes := make([]store.Node, len(healthyNodes))
		copy(nodes, healthyNodes)

		// Shuffle for randomness
		rand.Shuffle(len(nodes), func(i, j int) {
			nodes[i], nodes[j] = nodes[j], nodes[i]
		})

		selected := make([]string, 0, e.replicationFactor)

		for i := 0; i < e.replicationFactor; i++ {
			selected = append(selected, nodes[i].NodeID)
		}

		result[chunk.ChunkID] = selected
	}

	return result, nil
}

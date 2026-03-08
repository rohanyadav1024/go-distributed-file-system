package node

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	nodepb "github.com/rohanyadav1024/dfs/internal/protocol/node"
)

// Client is a gRPC client for communicating with storage nodes
type Client struct {
}

// CopyChunk requests a target node to copy a chunk from a source node
func (c *Client) CopyChunk(ctx context.Context, targetAddr string, sourceAddr string, chunkID string) error {
	// Dial the target node
	conn, err := grpc.DialContext(
		ctx,
		targetAddr,
		grpc.WithInsecure(), // Phase I: no TLS
	)
	if err != nil {
		return fmt.Errorf("failed to dial target node at %s: %w", targetAddr, err)
	}
	defer conn.Close()

	// Create NodeService client
	client := nodepb.NewNodeServiceClient(conn)

	// Call CopyChunk RPC
	resp, err := client.CopyChunk(ctx, &nodepb.CopyChunkRequest{
		ChunkId:       chunkID,
		SourceAddress: sourceAddr,
	})
	if err != nil {
		return fmt.Errorf("CopyChunk RPC failed: %w", err)
	}

	// Check if copy was successful
	if !resp.GetSuccess() {
		return fmt.Errorf("CopyChunk failed on target node %s", targetAddr)
	}

	return nil
}

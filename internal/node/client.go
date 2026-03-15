package node

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	nodepb "github.com/rohanyadav1024/dfs/internal/protocol/node"
)

// Client is a gRPC client for communicating with storage nodes
type Client struct {
}

// CopyChunk requests a target node to copy a chunk from a source node
func (c *Client) CopyChunk(ctx context.Context, targetAddr string, sourceAddr string, chunkID string) error {
	targetHost, _, err := net.SplitHostPort(targetAddr)
	if err != nil {
		return fmt.Errorf("invalid target address %s: %w", targetAddr, err)
	}

	clientCert, err := tls.LoadX509KeyPair("/certs/server.crt", "/certs/server.key")
	if err != nil {
		return fmt.Errorf("failed to load client certificate/key: %w", err)
	}

	caCertPEM, err := os.ReadFile("/certs/ca.crt")
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caPool := x509.NewCertPool()
	if ok := caPool.AppendCertsFromPEM(caCertPEM); !ok {
		return fmt.Errorf("failed to append CA certificate to pool")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      caPool,
		ServerName:   targetHost,
	}

	// Dial the target node
	conn, err := grpc.DialContext(
		ctx,
		targetAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
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

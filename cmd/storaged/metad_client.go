package main

import (
	"context"
	"time"

	nodepb "github.com/rohanyadav1024/dfs/internal/protocol/node"
	chunkstore "github.com/rohanyadav1024/dfs/internal/storage/chunkstore"
	storagemetrics "github.com/rohanyadav1024/dfs/internal/storage/metrics"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// metadClient handles communication from storaged → metad
type metadClient struct {
	conn          *grpc.ClientConn
	client        nodepb.NodeServiceClient
	nodeID        string
	nodeAddr      string
	capacityBytes int64
	store         chunkstore.Store
}

// newMetadClient dials metad and sends initial heartbeat (acts as registration)
func newMetadClient(
	ctx context.Context,
	log *zap.Logger,
	nodeID string,
	nodeAddr string,
	metadAddr string,
	store chunkstore.Store,
) (*metadClient, error) {

	conn, err := grpc.DialContext(
		ctx,
		metadAddr,
		grpc.WithInsecure(), // Phase I: no TLS
		grpc.WithBlock(),    // Wait until connection is ready
	)
	if err != nil {
		return nil, err
	}

	client := nodepb.NewNodeServiceClient(conn)

	m := &metadClient{
		conn:          conn,
		client:        client,
		nodeID:        nodeID,
		nodeAddr:      nodeAddr,
		capacityBytes: store.CapacityBytes(),
		store:         store,
	}

	// Initial heartbeat = registration
	availableBytes := m.store.AvailableBytes()
	storagemetrics.SetAvailableBytes(availableBytes)
	_, err = client.Heartbeat(ctx, &nodepb.HeartbeatRequest{
		NodeId:         nodeID,
		Address:        nodeAddr,
		CapacityBytes:  m.capacityBytes,
		AvailableBytes: availableBytes,
	})
	if err != nil {
		conn.Close()
		return nil, err
	}

	log.Info("registered with metad via heartbeat",
		zap.String("node_id", nodeID),
		zap.String("address", nodeAddr),
		zap.Int64("capacity_bytes", m.capacityBytes),
	)

	return m, nil
}

// startHeartbeat starts background heartbeat loop
func (m *metadClient) startHeartbeat(
	ctx context.Context,
	log *zap.Logger,
	interval time.Duration,
) {
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Info("stopping heartbeat loop", zap.String("node_id", m.nodeID))
				return

			case <-ticker.C:
				availableBytes := m.store.AvailableBytes()
				storagemetrics.SetAvailableBytes(availableBytes)
				_, err := m.client.Heartbeat(ctx, &nodepb.HeartbeatRequest{
					NodeId:         m.nodeID,
					Address:        m.nodeAddr,
					CapacityBytes:  m.capacityBytes,
					AvailableBytes: availableBytes,
				})
				if err != nil {
					log.Warn("heartbeat failed",
						zap.String("node_id", m.nodeID),
						zap.Error(err),
					)
				}
			}
		}
	}()
}

// close closes the gRPC connection
func (m *metadClient) close() {
	if m.conn != nil {
		m.conn.Close()
	}
}

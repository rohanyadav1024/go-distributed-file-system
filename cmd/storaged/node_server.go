package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"os"

	nodepb "github.com/rohanyadav1024/dfs/internal/protocol/node"
	storagepb "github.com/rohanyadav1024/dfs/internal/protocol/storage"
	"github.com/rohanyadav1024/dfs/internal/storage/chunkstore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

type nodeServer struct {
	nodepb.UnimplementedNodeServiceServer
	store chunkstore.Store
}

func (s *nodeServer) Heartbeat(ctx context.Context, req *nodepb.HeartbeatRequest) (*nodepb.HeartbeatResponse, error) {
	return &nodepb.HeartbeatResponse{}, nil
}

func (s *nodeServer) CopyChunk(ctx context.Context, req *nodepb.CopyChunkRequest) (*nodepb.CopyChunkResponse, error) {

	if req.GetChunkId() == "" {
		return nil, status.Error(codes.InvalidArgument, "chunk_id cannot be empty")
	}
	if req.GetSourceAddress() == "" {
		return nil, status.Error(codes.InvalidArgument, "source_address cannot be empty")
	}

	sourceHost, _, err := net.SplitHostPort(req.GetSourceAddress())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid source_address format: %v", err)
	}

	clientCert, err := tls.LoadX509KeyPair("/certs/server.crt", "/certs/server.key")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load client certificate/key: %v", err)
	}

	caCertPEM, err := os.ReadFile("/certs/ca.crt")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read CA certificate: %v", err)
	}

	caPool := x509.NewCertPool()
	if ok := caPool.AppendCertsFromPEM(caCertPEM); !ok {
		return nil, status.Error(codes.Internal, "failed to append CA certificate to pool")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      caPool,
		ServerName:   sourceHost,
	}

	conn, err := grpc.DialContext(
		ctx,
		req.GetSourceAddress(),
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
	)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "failed to connect to source node: %v", err)
	}
	defer conn.Close()

	storageClient := storagepb.NewStorageServiceClient(conn)

	getResp, err := storageClient.GetChunk(ctx, &storagepb.GetChunkRequest{
		ChunkId: req.GetChunkId(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get chunk from source: %v", err)
	}

	err = s.store.Put(ctx, req.GetChunkId(), bytes.NewReader(getResp.GetData()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to put chunk in local store: %v", err)
	}

	return &nodepb.CopyChunkResponse{Success: true}, nil
}

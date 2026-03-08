package main

import (
	"bytes"
	"context"

	nodepb "github.com/rohanyadav1024/dfs/internal/protocol/node"
	storagepb "github.com/rohanyadav1024/dfs/internal/protocol/storage"
	"github.com/rohanyadav1024/dfs/internal/storage/chunkstore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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

	conn, err := grpc.DialContext(ctx, req.GetSourceAddress(), grpc.WithInsecure())
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
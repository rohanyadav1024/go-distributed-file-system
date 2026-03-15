package main

import (
	"bytes"
	"context"
	"errors"
	"io"

	customerrors "github.com/rohanyadav1024/dfs/internal/common/errors"
	storagepb "github.com/rohanyadav1024/dfs/internal/protocol/storage"
	"github.com/rohanyadav1024/dfs/internal/storage/chunkstore"
	storagemetrics "github.com/rohanyadav1024/dfs/internal/storage/metrics"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// storageServer implements storagepb.StorageServiceServer
type storageServer struct {
	storagepb.UnimplementedStorageServiceServer
	store chunkstore.Store
}

// mapError converts domain errors to gRPC status codes using sentinel checks.
func mapError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, customerrors.ErrNotFound):
		return status.Error(codes.NotFound, "not found")

	case errors.Is(err, customerrors.ErrInvalidArgument):
		return status.Error(codes.InvalidArgument, "invalid argument")

	case errors.Is(err, customerrors.ErrAlreadyExists):
		return status.Error(codes.AlreadyExists, "already exists")
	}

	// Default fallback
	return status.Error(codes.Internal, "internal server error")
}

// PutChunk stores a chunk
func (s *storageServer) PutChunk(ctx context.Context, req *storagepb.PutChunkRequest) (*storagepb.PutChunkResponse, error) {
	// Validate inputs
	if req.GetChunkId() == "" {
		return nil, status.Error(codes.InvalidArgument, "chunk_id cannot be empty")
	}
	if len(req.GetData()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "data cannot be empty")
	}

	// Call chunkstore with data as reader
	err := s.store.Put(ctx, req.GetChunkId(), bytes.NewReader(req.GetData()))
	if err != nil {
		return nil, mapError(err)
	}
	storagemetrics.IncChunksStored()
	storagemetrics.SetAvailableBytes(s.store.AvailableBytes())

	return &storagepb.PutChunkResponse{}, nil
}

// GetChunk retrieves a chunk
func (s *storageServer) GetChunk(ctx context.Context, req *storagepb.GetChunkRequest) (*storagepb.GetChunkResponse, error) {
	// Validate inputs
	if req.GetChunkId() == "" {
		return nil, status.Error(codes.InvalidArgument, "chunk_id cannot be empty")
	}

	// Get from chunkstore
	rc, err := s.store.Get(ctx, req.GetChunkId())
	if err != nil {
		return nil, mapError(err)
	}
	defer rc.Close()

	// Read all data from the reader
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to read chunk data")
	}
	storagemetrics.IncChunksServed()

	return &storagepb.GetChunkResponse{Data: data}, nil
}

// DeleteChunk deletes a chunk
func (s *storageServer) DeleteChunk(ctx context.Context, req *storagepb.DeleteChunkRequest) (*storagepb.DeleteChunkResponse, error) {
	// Validate inputs
	if req.GetChunkId() == "" {
		return nil, status.Error(codes.InvalidArgument, "chunk_id cannot be empty")
	}

	// Delete from chunkstore
	err := s.store.Delete(ctx, req.GetChunkId())
	if err != nil {
		return nil, mapError(err)
	}
	storagemetrics.SetAvailableBytes(s.store.AvailableBytes())

	return &storagepb.DeleteChunkResponse{}, nil
}

// Health returns the health status of the storage service
func (s *storageServer) Health(ctx context.Context, req *storagepb.HealthRequest) (*storagepb.HealthResponse, error) {
	return &storagepb.HealthResponse{Healthy: true}, nil
}

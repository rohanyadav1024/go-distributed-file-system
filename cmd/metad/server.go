package main

// This file implements the main server logic for the metadata service (metad).
// Defines gRPC handlers for MetadataService and NodeService, and initializes the
// necessary components like the metadata store, registry, and placement engine.
// It defines the API calls

import (
	"context"

	"github.com/rohanyadav1024/dfs/internal/metadata/manifest"
	"github.com/rohanyadav1024/dfs/internal/metadata/metrics"
	"github.com/rohanyadav1024/dfs/internal/metadata/registry"
	metadatapb "github.com/rohanyadav1024/dfs/internal/protocol/metadata"
	nodepb "github.com/rohanyadav1024/dfs/internal/protocol/node"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// metadataServer implements MetadataService.
type metadataServer struct {
	metadatapb.UnimplementedMetadataServiceServer
	manifest *manifest.Manager
}

// nodeServer implements NodeService.
type nodeServer struct {
	nodepb.UnimplementedNodeServiceServer
	registry *registry.Manager
}

// ----------------------------
// MetadataService Methods
// ----------------------------

func (s *metadataServer) PrepareUpload(
	ctx context.Context,
	req *metadatapb.PrepareUploadRequest,
) (*metadatapb.PrepareUploadResponse, error) {

	if req.GetFileName() == "" {
		return nil, status.Error(codes.InvalidArgument, "file_name is required")
	}

	if req.GetFileSizeBytes() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "file_size_bytes must be > 0")
	}

	sessionID, chunkPlans, err :=
		s.manifest.PrepareUpload(ctx, req.GetFileName(), req.GetFileSizeBytes())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "prepare upload failed: %v", err)
	}

	var protoChunks []*metadatapb.ChunkPlan

	for chunkID, replicaAddresses := range chunkPlans {

		var replicas []*metadatapb.Replica

		for _, addr := range replicaAddresses {
			replicas = append(replicas, &metadatapb.Replica{
				NodeId:  "", // not available yet
				Address: addr,
			})
		}

		protoChunks = append(protoChunks, &metadatapb.ChunkPlan{
			ChunkId:     chunkID,
			OffsetBytes: 0, // not tracked yet
			SizeBytes:   0, // not tracked yet
			Replicas:    replicas,
		})
	}

	return &metadatapb.PrepareUploadResponse{
		UploadSessionId: sessionID,
		Chunks:          protoChunks,
	}, nil
}

func (s *metadataServer) CommitUpload(
	ctx context.Context,
	req *metadatapb.CommitUploadRequest,
) (*metadatapb.CommitUploadResponse, error) {

	// Validate input
	if req.GetSessionId() == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}

	replicaMapProto := req.GetReplicaMap()
	if replicaMapProto == nil || len(replicaMapProto) == 0 {
		return nil, status.Error(codes.InvalidArgument, "replica_map is required")
	}

	// Convert protobuf replicaMap to domain model map[string][]string
	replicaMap := make(map[string][]string, len(replicaMapProto))
	for chunkID, replicaList := range replicaMapProto {
		replicaMap[chunkID] = replicaList.GetNodeIds()
	}

	// Call manifest to commit upload
	if err := s.manifest.CommitUpload(ctx, req.GetSessionId(), replicaMap); err != nil {
		return nil, status.Errorf(codes.Internal, "commit upload failed: %v", err)
	}

	return &metadatapb.CommitUploadResponse{}, nil
}

func (s *metadataServer) GetFile(
	ctx context.Context,
	req *metadatapb.GetFileRequest,
) (*metadatapb.GetFileResponse, error) {

	// Validate input
	if req.GetFileId() == "" {
		return nil, status.Error(codes.InvalidArgument, "file_id is required")
	}

	// Load file
	file, err := s.manifest.GetFile(ctx, req.GetFileId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get file: %v", err)
	}

	if file == nil {
		return nil, status.Error(codes.NotFound, "file not found")
	}

	// Check file is committed
	if file.Status != "committed" {
		return nil, status.Error(codes.FailedPrecondition, "file is not committed")
	}

	// Load chunks for this file
	chunks, err := s.manifest.GetChunks(ctx, file.FileID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get chunks: %v", err)
	}

	if len(chunks) == 0 {
		return nil, status.Error(codes.Internal, "no chunks found for file")
	}

	// Build chunk plans with replica locations
	var protoChunks []*metadatapb.ChunkPlan

	for _, chunk := range chunks {
		// Get replica locations for this chunk
		locations, err := s.manifest.GetChunkLocations(ctx, chunk.ChunkID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get chunk locations: %v", err)
		}

		var replicas []*metadatapb.Replica
		for _, loc := range locations {
			replicas = append(replicas, &metadatapb.Replica{
				NodeId:  loc.NodeID,
				Address: "", // address not yet available in store schema
			})
		}

		protoChunks = append(protoChunks, &metadatapb.ChunkPlan{
			ChunkId:     chunk.ChunkID,
			OffsetBytes: 0, // not tracked yet
			SizeBytes:   chunk.SizeBytes,
			Replicas:    replicas,
		})
	}

	return &metadatapb.GetFileResponse{
		FileId:        file.FileID,
		FileName:      file.FileName,
		FileSizeBytes: file.SizeBytes,
		Chunks:        protoChunks,
	}, nil
}

func (s *metadataServer) DeleteFile(
	ctx context.Context,
	req *metadatapb.DeleteFileRequest,
) (*metadatapb.DeleteFileResponse, error) {

	// Validate input
	if req.GetFileId() == "" {
		return nil, status.Error(codes.InvalidArgument, "file_id is required")
	}

	// Call manifest to delete file
	if err := s.manifest.DeleteFile(ctx, req.GetFileId()); err != nil {
		if err.Error() == "file not found" {
			return nil, status.Error(codes.NotFound, "file not found")
		}
		return nil, status.Errorf(codes.Internal, "delete file failed: %v", err)
	}

	return &metadatapb.DeleteFileResponse{}, nil
}

func (s *metadataServer) ListFiles(
	ctx context.Context,
	req *metadatapb.ListFilesRequest,
) (*metadatapb.ListFilesResponse, error) {

	// Call manifest to list all committed files
	files, err := s.manifest.ListFiles(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list files failed: %v", err)
	}

	// Convert domain files to proto FileSummary
	var fileSummaries []*metadatapb.FileSummary
	for _, file := range files {
		fileSummaries = append(fileSummaries, &metadatapb.FileSummary{
			FileId:        file.FileID,
			FileName:      file.FileName,
			FileSizeBytes: file.SizeBytes,
		})
	}

	return &metadatapb.ListFilesResponse{
		Files: fileSummaries,
	}, nil
}

// ----------------------------
// NodeService Methods
// ----------------------------

func (s *nodeServer) Heartbeat(
	ctx context.Context,
	req *nodepb.HeartbeatRequest,
) (*nodepb.HeartbeatResponse, error) {

	nodeID := req.GetNodeId()
	address := req.GetAddress()
	capacityBytes := req.GetCapacityBytes()
	availableBytes := req.GetAvailableBytes()

	if nodeID == "" {
		return nil, status.Error(codes.InvalidArgument, "node_id is required")
	}

	// Upsert node: register new or update existing
	if err := s.registry.Heartbeat(ctx, nodeID, address, capacityBytes, availableBytes); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register or update node: %v", err)
	}

	metrics.IncHeartbeats()

	healthyNodes, err := s.registry.ListHealthyNodes(ctx)
	if err == nil {
		metrics.SetHealthyNodes(len(healthyNodes))
	}

	return &nodepb.HeartbeatResponse{}, nil
}

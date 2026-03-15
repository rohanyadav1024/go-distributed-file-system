// Package manifest orchestrates metadata operations for file lifecycle events.
package manifest

import (
	"context"
	"fmt"
	"time"

	"github.com/rohanyadav1024/dfs/internal/common/ids"
	"github.com/rohanyadav1024/dfs/internal/constants"
	"github.com/rohanyadav1024/dfs/internal/metadata/placement"
	"github.com/rohanyadav1024/dfs/internal/metadata/registry"
	"github.com/rohanyadav1024/dfs/internal/metadata/store"
)

// Manager orchestrates file lifecycle logic.
type Manager struct {
	store     store.Store
	registry  *registry.Manager
	placement *placement.Engine
	chunkSize int64
}

// NewManager creates a new manifest manager.
func NewManager(
	s store.Store,
	r *registry.Manager,
	p *placement.Engine,
	chunkSize int64,
) *Manager {
	return &Manager{
		store:     s,
		registry:  r,
		placement: p,
		chunkSize: chunkSize,
	}
}

// PrepareUpload creates upload metadata and returns initial chunk placements.
func (m *Manager) PrepareUpload(
	ctx context.Context,
	fileName string,
	fileSize int64,
) (string, map[string][]string, error) {

	if m.chunkSize <= 0 {
		return "", nil, fmt.Errorf("invalid configured chunk size")
	}

	fileID := ids.NewRequestID()
	sessionID := ids.NewRequestID()

	chunkCount := int((fileSize + m.chunkSize - 1) / m.chunkSize)
	chunks := make([]store.Chunk, 0, chunkCount)

	for i := 0; i < chunkCount; i++ {
		chunkID := ids.NewRequestID()

		size := m.chunkSize
		if i == chunkCount-1 {
			remaining := fileSize - int64(i)*m.chunkSize
			if remaining > 0 {
				size = remaining
			}
		}

		chunks = append(chunks, store.Chunk{
			ChunkID:   chunkID,
			FileID:    fileID,
			Index:     i,
			SizeBytes: size,
		})
	}

	// Fetch healthy nodes.
	nodes, err := m.registry.ListHealthyNodes(ctx)
	if err != nil {
		return "", nil, err
	}

	// Run placement.
	replicaMap, err := m.placement.SelectReplicas(ctx, chunks, nodes)
	if err != nil {
		return "", nil, err
	}

	// Store metadata atomically.
	err = m.store.WithTx(ctx, func(tx store.Store) error {

		if err := tx.CreateFile(ctx, store.File{
			FileID:    fileID,
			FileName:  fileName,
			SizeBytes: fileSize,
			Status:    constants.FileStatusPending,
			CreatedAt: time.Now().Unix(),
		}); err != nil {
			return err
		}

		if err := tx.CreateUploadSession(ctx, store.UploadSession{
			SessionID: sessionID,
			FileID:    fileID,
			Status:    constants.UploadStatusPreparing,
			CreatedAt: time.Now().Unix(),
		}); err != nil {
			return err
		}

		if err := tx.InsertChunks(ctx, chunks); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", nil, err
	}

	return sessionID, replicaMap, nil
}

// GetFile retrieves file metadata.
func (m *Manager) GetFile(ctx context.Context, fileID string) (*store.File, error) {
	return m.store.GetFile(ctx, fileID)
}

// GetChunks retrieves all chunks for a file.
func (m *Manager) GetChunks(ctx context.Context, fileID string) ([]store.Chunk, error) {
	return m.store.GetChunksByFileID(ctx, fileID)
}

// GetChunkLocations retrieves all replica locations for a chunk.
func (m *Manager) GetChunkLocations(ctx context.Context, chunkID string) ([]store.ChunkLocation, error) {
	return m.store.GetChunkLocations(ctx, chunkID)
}

// DeleteFile marks a file as deleted.
func (m *Manager) DeleteFile(ctx context.Context, fileID string) error {
	file, err := m.store.GetFile(ctx, fileID)
	if err != nil {
		return err
	}

	if file == nil {
		return fmt.Errorf("file not found")
	}

	return m.store.UpdateFileStatus(ctx, fileID, constants.FileStatusDeleted)
}

// ListFiles returns all committed files.
func (m *Manager) ListFiles(ctx context.Context) ([]store.File, error) {
	files, err := m.store.ListFiles(ctx)
	if err != nil {
		return nil, err
	}

	// Filter for committed files only
	var committedFiles []store.File
	for _, fileRecord := range files {
		if fileRecord.Status == constants.FileStatusCommitted {
			committedFiles = append(committedFiles, fileRecord)
		}
	}

	return committedFiles, nil
}

// CommitUpload validates replica metadata and marks the upload as committed.
func (m *Manager) CommitUpload(
	ctx context.Context,
	sessionID string,
	replicaMap map[string][]string,
) error {

	return m.store.WithTx(ctx, func(tx store.Store) error {

		// Load session.
		session, err := tx.GetUploadSession(ctx, sessionID)
		if err != nil {
			return err
		}
		if session == nil {
			return fmt.Errorf("upload session not found")
		}

		// Idempotency: already committed → success
		if session.Status == constants.UploadStatusCommitted {
			return nil
		}

		if session.Status != constants.UploadStatusPreparing {
			return fmt.Errorf("invalid session state: %s", session.Status)
		}

		// Load file.
		file, err := tx.GetFile(ctx, session.FileID)
		if err != nil {
			return err
		}
		if file == nil {
			return fmt.Errorf("file not found for session")
		}

		if file.Status == constants.FileStatusCommitted {
			return nil
		}

		if file.Status != constants.FileStatusPending {
			return fmt.Errorf("file is not in pending state")
		}

		// Load chunks for this file.
		chunks, err := tx.GetChunksByFileID(ctx, file.FileID)
		if err != nil {
			return err
		}
		if len(chunks) == 0 {
			return fmt.Errorf("no chunks found for file")
		}

		expectedChunkCount := len(chunks)

		// Validate replica map coverage.
		if len(replicaMap) != expectedChunkCount {
			return fmt.Errorf("replica map incomplete: expected %d chunks, got %d",
				expectedChunkCount, len(replicaMap))
		}

		replicationFactor := m.placement.ReplicationFactor()

		// Validate each chunk.
		validChunkIDs := make(map[string]struct{}, expectedChunkCount)
		for _, c := range chunks {
			validChunkIDs[c.ChunkID] = struct{}{}
		}

		var locations []store.ChunkLocation

		for chunkID, nodeIDs := range replicaMap {

			// Check chunk belongs to file
			if _, ok := validChunkIDs[chunkID]; !ok {
				return fmt.Errorf("invalid chunkID in replica map: %s", chunkID)
			}

			if len(nodeIDs) < replicationFactor {
				return fmt.Errorf("chunk %s has insufficient replicas: %d < %d",
					chunkID, len(nodeIDs), replicationFactor)
			}

			for _, nodeID := range nodeIDs {
				if nodeID == "" {
					return fmt.Errorf("empty nodeID for chunk %s", chunkID)
				}

				locations = append(locations, store.ChunkLocation{
					ChunkID: chunkID,
					NodeID:  nodeID,
				})
			}
		}

		// Insert replica mappings.
		if err := tx.InsertChunkLocations(ctx, locations); err != nil {
			return err
		}

		// Mark file committed.
		if err := tx.UpdateFileStatus(ctx, file.FileID, constants.FileStatusCommitted); err != nil {
			return err
		}

		// Mark session committed.
		if err := tx.UpdateUploadSessionStatus(ctx, sessionID, constants.UploadStatusCommitted); err != nil {
			return err
		}

		return nil
	})
}

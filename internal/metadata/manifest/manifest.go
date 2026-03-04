package manifest

import (
	"context"
	"fmt"
	"time"

	"github.com/rohanyadav1024/dfs/internal/common/ids"
	"github.com/rohanyadav1024/dfs/internal/metadata/placement"
	"github.com/rohanyadav1024/dfs/internal/metadata/registry"
	"github.com/rohanyadav1024/dfs/internal/metadata/store"
)

// Manager orchestrates file lifecycle logic.
type Manager struct {
	store     store.Store
	registry  *registry.Manager
	placement *placement.Engine
}

// NewManager creates a new manifest manager.
func NewManager(
	s store.Store,
	r *registry.Manager,
	p *placement.Engine,
) *Manager {
	return &Manager{
		store:     s,
		registry:  r,
		placement: p,
	}
}

// PrepareUpload creates a pending file record, upload session,
// chunk records, and returns replica placements for the client to upload to.
// It does not modify node or chunk location metadata - that is done in CommitUpload.
func (m *Manager) PrepareUpload(
	ctx context.Context,
	fileName string,
	fileSize int64,
	chunkSize int64,
) (string, map[string][]string, error) {

	if chunkSize <= 0 {
		return "", nil, fmt.Errorf("invalid chunk size")
	}

	fileID := ids.NewRequestID()
	sessionID := ids.NewRequestID()

	chunkCount := int((fileSize + chunkSize - 1) / chunkSize)
	chunks := make([]store.Chunk, 0, chunkCount)

	for i := 0; i < chunkCount; i++ {
		chunkID := ids.NewRequestID()

		size := chunkSize
		if i == chunkCount-1 {
			remaining := fileSize - int64(i)*chunkSize
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

	// 🔹 1. Fetch healthy nodes
	nodes, err := m.registry.ListHealthyNodes(ctx)
	if err != nil {
		return "", nil, err
	}

	// 🔹 2. Run placement
	replicaMap, err := m.placement.SelectReplicas(ctx, chunks, nodes)
	if err != nil {
		return "", nil, err
	}

	// 🔹 3. Store metadata atomically
	err = m.store.WithTx(ctx, func(tx store.Store) error {

		if err := tx.CreateFile(ctx, store.File{
			FileID:    fileID,
			FileName:  fileName,
			SizeBytes: fileSize,
			Status:    "pending",
			CreatedAt: time.Now().Unix(),
		}); err != nil {
			return err
		}

		if err := tx.CreateUploadSession(ctx, store.UploadSession{
			SessionID: sessionID,
			FileID:    fileID,
			Status:    "preparing",
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

// CommitUpload finalizes an upload session by:
// - validating session state
// - validating replica integrity
// - inserting chunk replica mappings
// - marking file as committed
// - marking session as committed
func (m *Manager) CommitUpload(
	ctx context.Context,
	sessionID string,
	replicaMap map[string][]string,
) error {

	return m.store.WithTx(ctx, func(tx store.Store) error {

		// 1️⃣ Load session
		session, err := tx.GetUploadSession(ctx, sessionID)
		if err != nil {
			return err
		}
		if session == nil {
			return fmt.Errorf("upload session not found")
		}

		// Idempotency: already committed → success
		if session.Status == "committed" {
			return nil
		}

		if session.Status != "preparing" {
			return fmt.Errorf("invalid session state: %s", session.Status)
		}

		// 2️⃣ Load file
		file, err := tx.GetFile(ctx, session.FileID)
		if err != nil {
			return err
		}
		if file == nil {
			return fmt.Errorf("file not found for session")
		}

		if file.Status == "committed" {
			return nil
		}

		if file.Status != "pending" {
			return fmt.Errorf("file is not in pending state")
		}

		// 3️⃣ Load chunks for this file
		chunks, err := tx.GetChunksByFileID(ctx, file.FileID)
		if err != nil {
			return err
		}
		if len(chunks) == 0 {
			return fmt.Errorf("no chunks found for file")
		}

		expectedChunkCount := len(chunks)

		// 4️⃣ Validate replicaMap coverage
		if len(replicaMap) != expectedChunkCount {
			return fmt.Errorf("replica map incomplete: expected %d chunks, got %d",
				expectedChunkCount, len(replicaMap))
		}

		replicationFactor := m.placement.ReplicationFactor()

		// 5️⃣ Validate each chunk
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

		// 6️⃣ Insert replica mappings
		if err := tx.InsertChunkLocations(ctx, locations); err != nil {
			return err
		}

		// 7️⃣ Mark file committed
		if err := tx.UpdateFileStatus(ctx, file.FileID, "committed"); err != nil {
			return err
		}

		// 8️⃣ Mark session committed
		if err := tx.UpdateUploadSessionStatus(ctx, sessionID, "committed"); err != nil {
			return err
		}

		return nil
	})
}
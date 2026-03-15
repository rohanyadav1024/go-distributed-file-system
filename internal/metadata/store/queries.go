package store

import (
	"fmt"

	"github.com/rohanyadav1024/dfs/internal/constants"
)

// -----------------------------
// File Queries
// -----------------------------

var (
	queryInsertFile = fmt.Sprintf(`
INSERT INTO %s (file_id, file_name, size_bytes, status, created_at)
VALUES (?, ?, ?, ?, ?)
`, constants.TableFiles)

	querySelectFileByID = fmt.Sprintf(`
SELECT file_id, file_name, size_bytes, status, created_at
FROM %s
WHERE file_id = ?
`, constants.TableFiles)

	querySelectAllFiles = fmt.Sprintf(`
SELECT file_id, file_name, size_bytes, status, created_at
FROM %s
ORDER BY created_at DESC
`, constants.TableFiles)

	queryUpdateFileStatus = fmt.Sprintf(`
UPDATE %s
SET status = ?
WHERE file_id = ?
`, constants.TableFiles)
)

// -----------------------------
// Chunk Queries
// -----------------------------

var (
	queryInsertChunk = fmt.Sprintf(`
INSERT INTO %s (chunk_id, file_id, chunk_index, size_bytes)
VALUES (?, ?, ?, ?)
`, constants.TableChunks)

	querySelectChunksByFileID = fmt.Sprintf(`
SELECT chunk_id, file_id, chunk_index, size_bytes
FROM %s
WHERE file_id = ?
ORDER BY chunk_index ASC
`, constants.TableChunks)

	querySelectAllChunks = fmt.Sprintf(`
SELECT chunk_id, file_id, chunk_index, size_bytes
FROM %s
`, constants.TableChunks)

	querySelectCommittedChunks = fmt.Sprintf(`
SELECT c.chunk_id, c.file_id, c.chunk_index, c.size_bytes
FROM %s c
JOIN %s f ON f.file_id = c.file_id
WHERE f.status = '%s'
`, constants.TableChunks, constants.TableFiles, constants.FileStatusCommitted)
)

// -----------------------------
// Chunk Location Queries
// -----------------------------

var (
	queryInsertChunkLocation = fmt.Sprintf(`
INSERT INTO %s (chunk_id, node_id)
VALUES (?, ?)
`, constants.TableChunkLocation)

	querySelectChunkLocationsByChunkID = fmt.Sprintf(`
SELECT chunk_id, node_id
FROM %s
WHERE chunk_id = ?
`, constants.TableChunkLocation)
)

// -----------------------------
// Node Queries
// -----------------------------

var (
	queryInsertNode = fmt.Sprintf(`
INSERT INTO %s (node_id, address, capacity_bytes, available_bytes, status, last_heartbeat)
VALUES (?, ?, ?, ?, ?, ?)
`, constants.TableNodes)

	queryUpdateNodeHeartbeat = fmt.Sprintf(`
UPDATE %s
SET last_heartbeat = ?
WHERE node_id = ?
`, constants.TableNodes)

	querySelectHealthyNodes = fmt.Sprintf(`
SELECT node_id, address, capacity_bytes, available_bytes, status, last_heartbeat
FROM %s
WHERE status = '%s'
`, constants.TableNodes, constants.NodeStatusHealthy)

	querySelectAllNodes = fmt.Sprintf(`
SELECT node_id, address, capacity_bytes, available_bytes, status, last_heartbeat
FROM %s
`, constants.TableNodes)

	queryCountTotalNodes = fmt.Sprintf(`
SELECT COUNT(*)
FROM %s
`, constants.TableNodes)

	queryCountHealthyNodes = fmt.Sprintf(`
SELECT COUNT(*)
FROM %s
WHERE status = '%s'
`, constants.TableNodes, constants.NodeStatusHealthy)

	queryUpdateNodeStatus = fmt.Sprintf(`
UPDATE %s
SET status = ?
WHERE node_id = ?
`, constants.TableNodes)

	queryUpsertNodeHeartbeat = fmt.Sprintf(`
UPDATE %s
SET address = ?, capacity_bytes = ?, available_bytes = ?, last_heartbeat = ?, status = '%s'
WHERE node_id = ?
`, constants.TableNodes, constants.NodeStatusHealthy)
)

// -----------------------------
// Upload Session Queries
// -----------------------------

var (
	queryInsertUploadSession = fmt.Sprintf(`
INSERT INTO %s (session_id, file_id, status, created_at)
VALUES (?, ?, ?, ?)
`, constants.TableUploadSession)

	querySelectUploadSessionByID = fmt.Sprintf(`
SELECT session_id, file_id, status, created_at
FROM %s
WHERE session_id = ?
`, constants.TableUploadSession)

	queryUpdateUploadSessionStatus = fmt.Sprintf(`
UPDATE %s
SET status = ?
WHERE session_id = ?
`, constants.TableUploadSession)

	queryCountTotalChunks = fmt.Sprintf(`
SELECT COUNT(*)
FROM %s
`, constants.TableChunks)

	queryCountTotalReplicas = fmt.Sprintf(`
SELECT COUNT(*)
FROM %s
`, constants.TableChunkLocation)
)

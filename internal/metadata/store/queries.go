package store

// -----------------------------
// File Queries
// -----------------------------

const (
	queryInsertFile = `
INSERT INTO files (file_id, file_name, size_bytes, status, created_at)
VALUES (?, ?, ?, ?, ?)
`

	querySelectFileByID = `
SELECT file_id, file_name, size_bytes, status, created_at
FROM files
WHERE file_id = ?
`

	querySelectAllFiles = `
SELECT file_id, file_name, size_bytes, status, created_at
FROM files
ORDER BY created_at DESC
`

	queryUpdateFileStatus = `
UPDATE files
SET status = ?
WHERE file_id = ?
`
)

// -----------------------------
// Chunk Queries
// -----------------------------

const (
	queryInsertChunk = `
INSERT INTO chunks (chunk_id, file_id, chunk_index, size_bytes)
VALUES (?, ?, ?, ?)
`

	querySelectChunksByFileID = `
SELECT chunk_id, file_id, chunk_index, size_bytes
FROM chunks
WHERE file_id = ?
ORDER BY chunk_index ASC
`

	querySelectAllChunks = `
SELECT chunk_id, file_id, chunk_index, size_bytes
FROM chunks
`

	querySelectCommittedChunks = `
SELECT c.chunk_id, c.file_id, c.chunk_index, c.size_bytes
FROM chunks c
JOIN files f ON f.file_id = c.file_id
WHERE f.status = 'committed'
`
)

// -----------------------------
// Chunk Location Queries
// -----------------------------

const (
	queryInsertChunkLocation = `
INSERT INTO chunk_locations (chunk_id, node_id)
VALUES (?, ?)
`

	querySelectChunkLocationsByChunkID = `
SELECT chunk_id, node_id
FROM chunk_locations
WHERE chunk_id = ?
`
)

// -----------------------------
// Node Queries
// -----------------------------

const (
	queryInsertNode = `
INSERT INTO nodes (node_id, address, capacity_bytes, available_bytes, status, last_heartbeat)
VALUES (?, ?, ?, ?, ?, ?)
`

	queryUpdateNodeHeartbeat = `
UPDATE nodes
SET last_heartbeat = ?
WHERE node_id = ?
`

	querySelectHealthyNodes = `
SELECT node_id, address, capacity_bytes, available_bytes, status, last_heartbeat
FROM nodes
WHERE status = 'healthy'
`

	querySelectAllNodes = `
SELECT node_id, address, capacity_bytes, available_bytes, status, last_heartbeat
FROM nodes
`

	queryCountTotalNodes = `
SELECT COUNT(*)
FROM nodes
`

	queryCountHealthyNodes = `
SELECT COUNT(*)
FROM nodes
WHERE status = 'healthy'
`

	queryUpdateNodeStatus = `
UPDATE nodes
SET status = ?
WHERE node_id = ?
`

	queryUpsertNodeHeartbeat = `
UPDATE nodes
SET address = ?, capacity_bytes = ?, available_bytes = ?, last_heartbeat = ?, status = 'healthy'
WHERE node_id = ?
`
)

// -----------------------------
// Upload Session Queries
// -----------------------------

const (
	queryInsertUploadSession = `
INSERT INTO upload_sessions (session_id, file_id, status, created_at)
VALUES (?, ?, ?, ?)
`

	querySelectUploadSessionByID = `
SELECT session_id, file_id, status, created_at
FROM upload_sessions
WHERE session_id = ?
`

	queryUpdateUploadSessionStatus = `
UPDATE upload_sessions
SET status = ?
WHERE session_id = ?
`

	queryCountTotalChunks = `
SELECT COUNT(*)
FROM chunks
`

	queryCountTotalReplicas = `
SELECT COUNT(*)
FROM chunk_locations
`
)

package store

import (
	"fmt"

	"github.com/rohanyadav1024/dfs/internal/constants"
)

// schema defines the SQLite schema for the metadata layer.
// It is executed during bootstrap.
var schema = fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
    file_id TEXT PRIMARY KEY,
    file_name TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    status TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS %s (
    chunk_id TEXT PRIMARY KEY,
    file_id TEXT NOT NULL,
    chunk_index INTEGER NOT NULL,
    size_bytes INTEGER NOT NULL,
    FOREIGN KEY(file_id) REFERENCES %s(file_id) ON DELETE CASCADE,
    UNIQUE(file_id, chunk_index)
);

CREATE TABLE IF NOT EXISTS %s (
    chunk_id TEXT NOT NULL,
    node_id TEXT NOT NULL,
    PRIMARY KEY (chunk_id, node_id),
    FOREIGN KEY(chunk_id) REFERENCES %s(chunk_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS %s (
    node_id TEXT PRIMARY KEY,
    address TEXT NOT NULL,
    capacity_bytes INTEGER NOT NULL,
    available_bytes INTEGER NOT NULL,
    status TEXT NOT NULL,
    last_heartbeat INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS %s (
    session_id TEXT PRIMARY KEY,
    file_id TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY(file_id) REFERENCES %s(file_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_chunks_file_id 
    ON %s(file_id);

CREATE INDEX IF NOT EXISTS idx_chunk_locations_chunk_id 
    ON %s(chunk_id);

CREATE INDEX IF NOT EXISTS idx_chunk_locations_node_id 
    ON %s(node_id);

CREATE INDEX IF NOT EXISTS idx_nodes_status 
    ON %s(status);
`,
	constants.TableFiles,
	constants.TableChunks,
	constants.TableFiles,
	constants.TableChunkLocation,
	constants.TableChunks,
	constants.TableNodes,
	constants.TableUploadSession,
	constants.TableFiles,
	constants.TableChunks,
	constants.TableChunkLocation,
	constants.TableChunkLocation,
	constants.TableNodes,
)

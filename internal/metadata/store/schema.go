

package store

// schema defines the SQLite schema for the metadata layer.
// It is executed during bootstrap.
const schema = `
CREATE TABLE IF NOT EXISTS files (
    file_id TEXT PRIMARY KEY,
    file_name TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    status TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS chunks (
    chunk_id TEXT PRIMARY KEY,
    file_id TEXT NOT NULL,
    chunk_index INTEGER NOT NULL,
    size_bytes INTEGER NOT NULL,
    FOREIGN KEY(file_id) REFERENCES files(file_id) ON DELETE CASCADE,
    UNIQUE(file_id, chunk_index)
);

CREATE TABLE IF NOT EXISTS chunk_locations (
    chunk_id TEXT NOT NULL,
    node_id TEXT NOT NULL,
    PRIMARY KEY (chunk_id, node_id),
    FOREIGN KEY(chunk_id) REFERENCES chunks(chunk_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS nodes (
    node_id TEXT PRIMARY KEY,
    capacity_bytes INTEGER NOT NULL,
    available_bytes INTEGER NOT NULL,
    status TEXT NOT NULL,
    last_heartbeat INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS upload_sessions (
    session_id TEXT PRIMARY KEY,
    file_id TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY(file_id) REFERENCES files(file_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_chunks_file_id 
    ON chunks(file_id);

CREATE INDEX IF NOT EXISTS idx_chunk_locations_chunk_id 
    ON chunk_locations(chunk_id);

CREATE INDEX IF NOT EXISTS idx_chunk_locations_node_id 
    ON chunk_locations(node_id);

CREATE INDEX IF NOT EXISTS idx_nodes_status 
    ON nodes(status);
`
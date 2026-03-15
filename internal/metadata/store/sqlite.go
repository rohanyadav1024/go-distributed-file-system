package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rohanyadav1024/dfs/internal/constants"
	_ "modernc.org/sqlite"
)

// execer abstracts both *sql.DB and *sql.Tx
type execer interface {
	// ExecContext executes a query with context and arguments. Does not return rows.
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	// QueryContext executes a query that returns rows, typically a SELECT.
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	// QueryRowContext executes a query that returns a single row, typically a SELECT.
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// SQLiteStore implements Store using SQLite.
type SQLiteStore struct {
	db   *sql.DB
	exec execer
}

// NewSQLite creates a new SQLite-backed metadata store.
func NewSQLite(path string) (*SQLiteStore, error) {
	if path == "" {
		return nil, fmt.Errorf("sqlite path cannot be empty")
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// SQLite allows only one writer at a time; keep one shared connection
	// to avoid write lock contention across pooled connections.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	store := &SQLiteStore{
		db:   db,
		exec: db,
	}

	if err := store.initPragmas(); err != nil {
		return nil, err
	}

	if err := store.bootstrapSchema(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *SQLiteStore) initPragmas() error {
	pragmas := []string{
		`PRAGMA journal_mode = WAL;`,
		`PRAGMA foreign_keys = ON;`,
		`PRAGMA busy_timeout = 5000;`,
	}

	for _, p := range pragmas {
		if _, err := s.db.Exec(p); err != nil {
			return fmt.Errorf("failed to execute pragma %s: %w", p, err)
		}
	}

	return nil
}

func (s *SQLiteStore) bootstrapSchema() error {
	_, err := s.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to bootstrap schema: %w", err)
	}
	return nil
}

// WithTx runs fn inside a transaction and commits on success.
func (s *SQLiteStore) WithTx(ctx context.Context, fn func(Store) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	txStore := &SQLiteStore{
		db:   s.db,
		exec: tx,
	}

	if err := fn(txStore); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// CreateFile inserts a new file record into the database.
func (s *SQLiteStore) CreateFile(ctx context.Context, file File) error {
	_, err := s.exec.ExecContext(
		ctx,
		queryInsertFile,
		file.FileID,
		file.FileName,
		file.SizeBytes,
		file.Status,
		file.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert file: %w", err)
	}

	return nil
}

// GetFile retrieves a file record by file_id.
func (s *SQLiteStore) GetFile(ctx context.Context, fileID string) (*File, error) {
	row := s.exec.QueryRowContext(ctx, querySelectFileByID, fileID)

	var file File
	err := row.Scan(
		&file.FileID,
		&file.FileName,
		&file.SizeBytes,
		&file.Status,
		&file.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	return &file, nil
}

// ListFiles retrieves all files ordered by creation time (newest first).
func (s *SQLiteStore) ListFiles(ctx context.Context) ([]File, error) {
	rows, err := s.exec.QueryContext(ctx, querySelectAllFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		var fileRecord File
		if err := rows.Scan(&fileRecord.FileID, &fileRecord.FileName, &fileRecord.SizeBytes, &fileRecord.Status, &fileRecord.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}
		files = append(files, fileRecord)
	}

	return files, nil
}

// UpdateFileStatus updates the status of a file.
func (s *SQLiteStore) UpdateFileStatus(ctx context.Context, fileID string, status string) error {
	_, err := s.exec.ExecContext(ctx, queryUpdateFileStatus, status, fileID)
	if err != nil {
		return fmt.Errorf("failed to update file status: %w", err)
	}
	return nil
}

// InsertChunks inserts multiple chunk records.
func (s *SQLiteStore) InsertChunks(ctx context.Context, chunks []Chunk) error {
	for _, c := range chunks {
		_, err := s.exec.ExecContext(ctx, queryInsertChunk, c.ChunkID, c.FileID, c.Index, c.SizeBytes)
		if err != nil {
			return fmt.Errorf("failed to insert chunk: %w", err)
		}
	}
	return nil
}

// GetChunksByFileID retrieves all chunks for a file ordered by index.
func (s *SQLiteStore) GetChunksByFileID(ctx context.Context, fileID string) ([]Chunk, error) {
	rows, err := s.exec.QueryContext(ctx, querySelectChunksByFileID, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to query chunks: %w", err)
	}
	defer rows.Close()

	var chunks []Chunk
	for rows.Next() {
		var chunk Chunk
		if err := rows.Scan(&chunk.ChunkID, &chunk.FileID, &chunk.Index, &chunk.SizeBytes); err != nil {
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// ListAllChunks retrieves all chunks from the system.
func (s *SQLiteStore) ListAllChunks(ctx context.Context) ([]Chunk, error) {
	rows, err := s.exec.QueryContext(ctx, querySelectAllChunks)
	if err != nil {
		return nil, fmt.Errorf("failed to query all chunks: %w", err)
	}
	defer rows.Close()

	var chunks []Chunk
	for rows.Next() {
		var chunk Chunk
		if err := rows.Scan(&chunk.ChunkID, &chunk.FileID, &chunk.Index, &chunk.SizeBytes); err != nil {
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// ListCommittedChunks retrieves only chunks whose parent files are committed.
func (s *SQLiteStore) ListCommittedChunks(ctx context.Context) ([]Chunk, error) {
	rows, err := s.exec.QueryContext(ctx, querySelectCommittedChunks)
	if err != nil {
		return nil, fmt.Errorf("failed to query committed chunks: %w", err)
	}
	defer rows.Close()

	var chunks []Chunk
	for rows.Next() {
		var chunk Chunk
		if err := rows.Scan(&chunk.ChunkID, &chunk.FileID, &chunk.Index, &chunk.SizeBytes); err != nil {
			return nil, fmt.Errorf("failed to scan committed chunk: %w", err)
		}
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// InsertChunkLocations inserts replica mappings.
func (s *SQLiteStore) InsertChunkLocations(ctx context.Context, locations []ChunkLocation) error {
	for _, l := range locations {
		_, err := s.exec.ExecContext(ctx, queryInsertChunkLocation, l.ChunkID, l.NodeID)
		if err != nil {
			return fmt.Errorf("failed to insert chunk location: %w", err)
		}
	}
	return nil
}

// GetChunkLocations retrieves all replica locations for a chunk.
func (s *SQLiteStore) GetChunkLocations(ctx context.Context, chunkID string) ([]ChunkLocation, error) {
	rows, err := s.exec.QueryContext(ctx, querySelectChunkLocationsByChunkID, chunkID)
	if err != nil {
		return nil, fmt.Errorf("failed to query chunk locations: %w", err)
	}
	defer rows.Close()

	var locations []ChunkLocation
	for rows.Next() {
		var location ChunkLocation
		if err := rows.Scan(&location.ChunkID, &location.NodeID); err != nil {
			return nil, fmt.Errorf("failed to scan chunk location: %w", err)
		}
		locations = append(locations, location)
	}

	return locations, nil
}

// AddChunkLocation adds a replica location for a chunk (idempotent insert).
func (s *SQLiteStore) AddChunkLocation(ctx context.Context, chunkID string, nodeID string) error {
	// Idempotent insert: ignore if already exists
	_, err := s.exec.ExecContext(
		ctx,
		queryInsertChunkLocation,
		chunkID,
		nodeID,
	)
	if err != nil {
		return fmt.Errorf("failed to add chunk location: %w", err)
	}
	return nil
}

// RegisterNode inserts a new storage node.
func (s *SQLiteStore) RegisterNode(ctx context.Context, node Node) error {
	_, err := s.exec.ExecContext(
		ctx,
		queryInsertNode,
		node.NodeID,
		node.Address,
		node.CapacityBytes,
		node.AvailableBytes,
		node.Status,
		node.LastHeartbeat,
	)
	if err != nil {
		return fmt.Errorf("failed to register node: %w", err)
	}
	return nil
}

// UpdateNodeHeartbeat updates heartbeat timestamp for a node.
func (s *SQLiteStore) UpdateNodeHeartbeat(ctx context.Context, nodeID string, ts int64) error {
	_, err := s.exec.ExecContext(ctx, queryUpdateNodeHeartbeat, ts, nodeID)
	if err != nil {
		return fmt.Errorf("failed to update node heartbeat: %w", err)
	}
	return nil
}

// ListHealthyNodes returns all healthy nodes.
func (s *SQLiteStore) ListHealthyNodes(ctx context.Context) ([]Node, error) {
	rows, err := s.exec.QueryContext(ctx, querySelectHealthyNodes)
	if err != nil {
		return nil, fmt.Errorf("failed to list healthy nodes: %w", err)
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		var node Node
		if err := rows.Scan(
			&node.NodeID,
			&node.Address,
			&node.CapacityBytes,
			&node.AvailableBytes,
			&node.Status,
			&node.LastHeartbeat,
		); err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// CreateUploadSession inserts a new upload session.
func (s *SQLiteStore) CreateUploadSession(ctx context.Context, session UploadSession) error {
	_, err := s.exec.ExecContext(
		ctx,
		queryInsertUploadSession,
		session.SessionID,
		session.FileID,
		session.Status,
		session.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create upload session: %w", err)
	}
	return nil
}

// GetUploadSession retrieves an upload session by ID.
func (s *SQLiteStore) GetUploadSession(ctx context.Context, sessionID string) (*UploadSession, error) {
	row := s.exec.QueryRowContext(ctx, querySelectUploadSessionByID, sessionID)

	var session UploadSession
	err := row.Scan(
		&session.SessionID,
		&session.FileID,
		&session.Status,
		&session.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get upload session: %w", err)
	}

	return &session, nil
}

// UpdateUploadSessionStatus updates the status of an upload session.
func (s *SQLiteStore) UpdateUploadSessionStatus(ctx context.Context, sessionID string, status string) error {
	_, err := s.exec.ExecContext(ctx, queryUpdateUploadSessionStatus, status, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update upload session status: %w", err)
	}
	return nil
}

// Close closes the underlying database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// ListAllNodes returns all nodes regardless of status.
func (s *SQLiteStore) ListAllNodes(ctx context.Context) ([]Node, error) {
	rows, err := s.exec.QueryContext(ctx, querySelectAllNodes)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		var node Node
		if err := rows.Scan(
			&node.NodeID,
			&node.Address,
			&node.CapacityBytes,
			&node.AvailableBytes,
			&node.Status,
			&node.LastHeartbeat,
		); err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// CountTotalNodes returns total registered nodes.
func (s *SQLiteStore) CountTotalNodes(ctx context.Context) (int, error) {
	row := s.exec.QueryRowContext(ctx, queryCountTotalNodes)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count total nodes: %w", err)
	}
	return count, nil
}

// CountHealthyNodes returns healthy nodes using the same logic as ListHealthyNodes.
func (s *SQLiteStore) CountHealthyNodes(ctx context.Context) (int, error) {
	row := s.exec.QueryRowContext(ctx, queryCountHealthyNodes)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count healthy nodes: %w", err)
	}
	return count, nil
}

// CountTotalChunks returns total metadata chunks.
func (s *SQLiteStore) CountTotalChunks(ctx context.Context) (int, error) {
	row := s.exec.QueryRowContext(ctx, queryCountTotalChunks)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count total chunks: %w", err)
	}
	return count, nil
}

// CountTotalReplicas returns total chunk replica records.
func (s *SQLiteStore) CountTotalReplicas(ctx context.Context) (int, error) {
	row := s.exec.QueryRowContext(ctx, queryCountTotalReplicas)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count total replicas: %w", err)
	}
	return count, nil
}

// UpdateNodeStatus updates node health status.
func (s *SQLiteStore) UpdateNodeStatus(
	ctx context.Context,
	nodeID string,
	status string,
) error {

	_, err := s.exec.ExecContext(ctx, queryUpdateNodeStatus, status, nodeID)
	if err != nil {
		return fmt.Errorf("failed to update node status: %w", err)
	}
	return nil
}

// UpsertNodeHeartbeat updates an existing node or inserts a new one if not found.
// It updates address, capacity, available bytes, heartbeat timestamp, and marks status as healthy.
func (s *SQLiteStore) UpsertNodeHeartbeat(
	ctx context.Context,
	nodeID string,
	address string,
	capacityBytes int64,
	availableBytes int64,
) error {
	now := time.Now().Unix()

	// Attempt to UPDATE existing node
	result, err := s.exec.ExecContext(
		ctx,
		queryUpsertNodeHeartbeat,
		address,
		capacityBytes,
		availableBytes,
		now,
		nodeID,
	)
	if err != nil {
		return fmt.Errorf("failed to update node heartbeat: %w", err)
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// If no rows were updated, insert new node
	if rowsAffected == 0 {
		_, err := s.exec.ExecContext(
			ctx,
			queryInsertNode,
			nodeID,
			address,
			capacityBytes,
			availableBytes,
			constants.NodeStatusHealthy,
			now,
		)
		if err != nil {
			return fmt.Errorf("failed to insert node: %w", err)
		}
	}

	return nil
}

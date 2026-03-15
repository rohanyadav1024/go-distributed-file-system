package store

import "context"

// ============================
// Domain Models
// ============================

type File struct {
	FileID    string
	FileName  string
	SizeBytes int64
	Status    string // pending, committed, deleted
	CreatedAt int64  // unix timestamp
}

type Chunk struct {
	ChunkID   string
	FileID    string
	Index     int
	SizeBytes int64
}

type ChunkLocation struct {
	ChunkID string
	NodeID  string
}

type Node struct {
	NodeID         string
	Address        string
	CapacityBytes  int64
	AvailableBytes int64
	Status         string // healthy, down
	LastHeartbeat  int64  // unix timestamp
}

type UploadSession struct {
	SessionID string
	FileID    string
	Status    string // preparing, committed, aborted
	CreatedAt int64
}

// ============================
// Store Interface
// ============================

type Store interface {

	// -------- File operations --------
	CreateFile(ctx context.Context, file File) error
	GetFile(ctx context.Context, fileID string) (*File, error)
	ListFiles(ctx context.Context) ([]File, error)
	UpdateFileStatus(ctx context.Context, fileID string, status string) error

	// -------- Chunk operations --------
	InsertChunks(ctx context.Context, chunks []Chunk) error
	GetChunksByFileID(ctx context.Context, fileID string) ([]Chunk, error)
	ListAllChunks(ctx context.Context) ([]Chunk, error)
	ListCommittedChunks(ctx context.Context) ([]Chunk, error)

	// -------- Replica operations --------
	InsertChunkLocations(ctx context.Context, locations []ChunkLocation) error
	GetChunkLocations(ctx context.Context, chunkID string) ([]ChunkLocation, error)
	AddChunkLocation(ctx context.Context, chunkID string, nodeID string) error

	// -------- Node operations --------
	RegisterNode(ctx context.Context, node Node) error
	UpdateNodeHeartbeat(ctx context.Context, nodeID string, ts int64) error
	ListHealthyNodes(ctx context.Context) ([]Node, error)
	ListAllNodes(ctx context.Context) ([]Node, error)
	CountTotalNodes(ctx context.Context) (int, error)
	CountHealthyNodes(ctx context.Context) (int, error)
	CountTotalChunks(ctx context.Context) (int, error)
	CountTotalReplicas(ctx context.Context) (int, error)
	UpdateNodeStatus(ctx context.Context, nodeID string, status string) error
	UpsertNodeHeartbeat(ctx context.Context, nodeID string, address string, capacityBytes int64, availableBytes int64) error

	// -------- Upload session operations --------
	CreateUploadSession(ctx context.Context, session UploadSession) error
	GetUploadSession(ctx context.Context, sessionID string) (*UploadSession, error)
	UpdateUploadSessionStatus(ctx context.Context, sessionID string, status string) error

	// -------- Transaction wrapper --------
	WithTx(ctx context.Context, fn func(Store) error) error

	// -------- Lifecycle --------
	Close() error
}

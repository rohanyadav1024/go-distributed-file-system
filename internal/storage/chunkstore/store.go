package chunkstore

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sync"

	customerrors "github.com/rohanyadav1024/dfs/internal/common/errors"
)

const (
	checksumSize = 32
	lengthSize   = 8
	headerSize   = checksumSize + lengthSize
	// minSize      = 1          // Minimum chunk size in bytes
	maxSize = 16 * 1024 * 1024 // Maximum chunk size in bytes (16MB)
)

// All the methods are designed to be context-aware,
// allowing for cancellation and timeouts.
// The Put method ensures atomic writes with embedded checksums
// for data integrity, while Get returns a ReadCloser that
// verifies the checksum on-the-fly during reads. Delete
// and Exists provide basic chunk management capabilities.

// All methods are implemneted considering chunk which will be
// having a fixed size and immutable once written,
// which is a common pattern in chunk storage systems.

// Store defines the interface for a chunk storage system.
type Store interface {
	Put(ctx context.Context, chunkID string, r io.Reader) error
	Get(ctx context.Context, chunkID string) (io.ReadCloser, error)
	Delete(ctx context.Context, chunkID string) error
	Exists(ctx context.Context, chunkID string) (bool, error)
	AvailableBytes() int64
	CapacityBytes() int64
}

// DiskStore implements Store using the local filesystem.
type DiskStore struct {
	baseDir       string
	capacityBytes int64
	usedBytes     int64
	mu            sync.Mutex
	// minSize int
	// maxSize int
}

func New(baseDir string, capacityBytes int64) (*DiskStore, error) {
	if baseDir == "" {
		return nil, customerrors.New(customerrors.CodeInvalidArgument, "base directory cannot be empty")
	}

	path := filepath.Join(baseDir, "chunks")
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, customerrors.Wrap(customerrors.CodeInternal, "failed to create base directory", err)
	}

	// Scan existing chunk files and compute used bytes
	usedBytes, err := scanUsedBytes(path)
	if err != nil {
		return nil, err
	}

	return &DiskStore{
		baseDir:       path,
		capacityBytes: capacityBytes,
		usedBytes:     usedBytes,
	}, nil
}

// scanUsedBytes walks the directory tree and sums actual file sizes.
func scanUsedBytes(baseDir string) (int64, error) {
	var total int64
	if err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	}); err != nil {
		return 0, customerrors.Wrap(customerrors.CodeInternal, "failed to scan chunk directory", err)
	}
	return total, nil
}

// AvailableBytes returns remaining capacity.
func (ds *DiskStore) AvailableBytes() int64 {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	return ds.capacityBytes - ds.usedBytes
}

// CapacityBytes returns total capacity.
func (ds *DiskStore) CapacityBytes() int64 {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	return ds.capacityBytes
}

// Path resolves chunkID into deterministic sharded path.
func (ds *DiskStore) Path(chunkID string) string {
	if len(chunkID) <= 2 {
		return filepath.Join(ds.baseDir, chunkID+".chunk")
	}

	dir, file := chunkID[:2], chunkID[2:]
	return filepath.Join(ds.baseDir, dir, file+".chunk")
}

func (ds *DiskStore) Put(ctx context.Context, chunkID string, r io.Reader) error {
	select {
	case <-ctx.Done():
		return customerrors.Wrap(customerrors.CodeInternal, "put cancelled", ctx.Err())
	default:
	}

	finalPath := ds.Path(chunkID)

	data, err := io.ReadAll(r)
	if err != nil {
		return customerrors.Wrap(customerrors.CodeInternal, "failed to read input data", err)
	}

	// enforce chunk size constraints if configured
	if len(data) > maxSize {
		return customerrors.New(customerrors.CodeInvalidArgument, "chunk size out of allowed range")
	}

	hashVal := sha256.Sum256(data)
	dataLen := uint64(len(data))

	// Idempotency check (header only)
	if f, err := os.Open(finalPath); err == nil {
		defer f.Close()

		header := make([]byte, headerSize)
		if _, err := io.ReadFull(f, header); err == nil {
			existingChecksum := header[:checksumSize]
			existingLen := binary.BigEndian.Uint64(header[checksumSize:])

			if existingLen == dataLen && string(existingChecksum) == string(hashVal[:]) {
				return nil // idempotent success
			}
		}
		return customerrors.New(customerrors.CodeIntegrityViolation,
			fmt.Sprintf("existing chunk mismatch for %s", chunkID))
	}

	// directory shard for the final file
	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		return customerrors.Wrap(customerrors.CodeInternal, "failed to create shard directory", err)
	}

	tempFile, err := os.CreateTemp(filepath.Dir(finalPath), "tmp-*.chunk")
	if err != nil {
		return customerrors.Wrap(customerrors.CodeInternal, "failed to create temp file", err)
	}

	// Write checksum
	if _, err := tempFile.Write(hashVal[:]); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return customerrors.Wrap(customerrors.CodeInternal, "failed to write checksum", err)
	}

	// Write length
	lengthBuf := make([]byte, lengthSize)
	binary.BigEndian.PutUint64(lengthBuf, dataLen)
	if _, err := tempFile.Write(lengthBuf); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return customerrors.Wrap(customerrors.CodeInternal, "failed to write length header", err)
	}

	// Write data
	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return customerrors.Wrap(customerrors.CodeInternal, "failed to write chunk data", err)
	}

	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return customerrors.Wrap(customerrors.CodeInternal, "failed to fsync temp file", err)
	}

	if err := tempFile.Close(); err != nil {
		os.Remove(tempFile.Name())
		return customerrors.Wrap(customerrors.CodeInternal, "failed to close temp file", err)
	}

	if err := os.Rename(tempFile.Name(), finalPath); err != nil {
		os.Remove(tempFile.Name())
		return customerrors.Wrap(customerrors.CodeInternal, "failed to rename temp file", err)
	}

	// Track disk usage
	ds.mu.Lock()
	ds.usedBytes += int64(headerSize) + int64(len(data))
	ds.mu.Unlock()

	return nil
}

func (ds *DiskStore) Get(ctx context.Context, chunkID string) (io.ReadCloser, error) {
	select {
	case <-ctx.Done():
		return nil, customerrors.Wrap(customerrors.CodeInternal, "get cancelled", ctx.Err())
	default:
	}

	path := ds.Path(chunkID)

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, customerrors.New(customerrors.CodeNotFound,
				fmt.Sprintf("chunk not found: %s", chunkID))
		}
		return nil, customerrors.Wrap(customerrors.CodeInternal,
			"failed to open chunk file", err)
	}

	header := make([]byte, headerSize)
	if _, err := io.ReadFull(f, header); err != nil {
		f.Close()
		return nil, customerrors.Wrap(customerrors.CodeIntegrityViolation,
			"failed to read chunk header", err)
	}

	expectedChecksum := header[:checksumSize]
	expectedLen := binary.BigEndian.Uint64(header[checksumSize:])

	return &verifiedReadCloser{
		file:           f,
		expectedHash:   expectedChecksum,
		expectedLength: expectedLen,
		hasher:         sha256.New(),
	}, nil
}

// verifiedReadCloser verifies checksum while streaming data.
type verifiedReadCloser struct {
	file           *os.File
	expectedHash   []byte
	expectedLength uint64
	hasher         hash.Hash
	readBytes      uint64
	done           bool
}

// Read reads data from verifiedReadCloser files and stores in p
// Returns number of bytes read and error if any
func (v *verifiedReadCloser) Read(p []byte) (int, error) {
	n, err := v.file.Read(p)
	if n > 0 {
		v.hasher.Write(p[:n])
		v.readBytes += uint64(n)
	}

	if err == io.EOF && !v.done {
		computed := v.hasher.Sum(nil)

		if v.readBytes != v.expectedLength ||
			!bytes.Equal(computed, v.expectedHash) {
			v.file.Close()
			return 0, customerrors.New(customerrors.CodeIntegrityViolation,
				"checksum or length mismatch detected")
		}
		v.done = true
	}

	return n, err
}

func (v *verifiedReadCloser) Close() error {
	return v.file.Close()
}

func (ds *DiskStore) Delete(ctx context.Context, chunkID string) error {
	path := ds.Path(chunkID)

	// Get file size before deletion
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return customerrors.Wrap(customerrors.CodeInternal,
			"failed to stat chunk", err)
	}

	fileSize := info.Size()

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return customerrors.Wrap(customerrors.CodeInternal,
			"failed to delete chunk", err)
	}

	// Track disk usage
	ds.mu.Lock()
	ds.usedBytes -= fileSize
	ds.mu.Unlock()

	return nil
}

func (ds *DiskStore) Exists(ctx context.Context, chunkID string) (bool, error) {
	_, err := os.Stat(ds.Path(chunkID))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, customerrors.Wrap(customerrors.CodeInternal,
		"failed to check chunk existence", err)
}

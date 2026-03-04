# Storage Chunkstore Package Documentation

## Overview

The `chunkstore` package provides disk-based immutable chunk storage with built-in integrity verification using SHA256 checksums. It's designed for distributed file systems where data durability and integrity are critical.

**Package Path:** `internal/storage/chunkstore`

**Key Features:**
- Atomic writes with crash-safety
- SHA256 checksum verification
- On-the-fly validation during reads
- Context-aware operations with timeout support
- Efficient directory sharding for scalability
- Idempotent put operations

---

## Design Principles

1. **Immutability**: Chunks are write-once, read-many (WORM) semantics
2. **Atomicity**: Writes are all-or-nothing using atomic file operations
3. **Integrity**: Every chunk includes checksums for verification
4. **Efficiency**: Streaming reads with on-the-fly validation
5. **Scalability**: Directory sharding prevents large directory listings
6. **Context-Aware**: All operations respect cancellation and timeouts

---

## Types

### Store Interface

```go
type Store interface {
    Put(ctx context.Context, chunkID string, r io.Reader) error
    Get(ctx context.Context, chunkID string) (io.ReadCloser, error)
    Delete(ctx context.Context, chunkID string) error
    Exists(ctx context.Context, chunkID string) (bool, error)
}
```

**Description:** Standard interface for chunk storage implementations.

---

### DiskStore

```go
type DiskStore struct {
    baseDir string
}
```

**Description:** File-system based implementation of the Store interface.

**Fields:**
- `baseDir`: Root directory for chunk storage

---

## File Format

```
[32 bytes: SHA256 checksum]
[8 bytes: big-endian data length]
[variable: chunk data]
```

**Properties:**
- **Checksum**: SHA256 hash of chunk data only (not including header)
- **Length**: Uint64 in big-endian format
- **Data**: Raw chunk data

**Total Header Size:** 40 bytes

---

## Directory Structure

Chunks are stored with deterministic sharding based on chunk ID:

```
baseDir/chunks/
├── ab/
│   ├── cdef123456.chunk
│   └── cdef789012.chunk
├── cd/
│   └── efgh.chunk
└── short.chunk  (for IDs with ≤2 characters)
```

---

## Constants

```go
const (
    checksumSize = 32                     // SHA256 output size
    lengthSize   = 8                      // Uint64 size
    headerSize   = 40                     // Total header size
    maxSize      = 16 * 1024 * 1024      // Maximum chunk size (16MB)
)
```

---

## Functions and Methods

### New()

```go
func New(baseDir string) (*DiskStore, error)
```

**Description:** Creates a new DiskStore instance and initializes the storage directory.

**Parameters:**
- `baseDir`: Base directory where chunk directory will be created

**Returns:** DiskStore instance or error

**Errors:**
- `CodeInvalidArgument`: If baseDir is empty
- `CodeInternal`: If unable to create directories

**Example:**
```go
store, err := chunkstore.New("/var/dfs/storage")
if err != nil {
    log.Fatal(err)
}
```

---

### Put()

```go
func (ds *DiskStore) Put(ctx context.Context, chunkID string, r io.Reader) error
```

**Description:** Stores a chunk atomically with checksum and length headers.

**Parameters:**
- `ctx`: Context for cancellation/timeout control
- `chunkID`: Unique identifier for the chunk
- `r`: Reader providing chunk data

**Returns:** Error if operation fails

**Behavior:**
1. Checks context cancellation
2. Reads entire chunk into memory
3. Validates chunk size (≤16MB)
4. Computes SHA256 checksum
5. **Idempotency check**: If chunk exists with same data, returns success
6. Creates shard directory if needed
7. Writes to temporary file:
   - SHA256 checksum (32 bytes)
   - Data length as uint64 (8 bytes)
   - Chunk data
8. Syncs file to disk
9. Atomically renames temp file to final location

**Errors:**
- Context related:
  - `CodeInternal`: If context cancelled before starting
- Input related:
  - `CodeInvalidArgument`: If chunk exceeds maxSize
  - `CodeInternal`: If unable to read input
- File related:
  - `CodeInternal`: If unable to create directories or files
  - `CodeIntegrityViolation`: If chunk exists with different content

**Idempotency:**
If a chunk with the same ID and content already exists, `Put` returns success without modifying the file. This allows safe retries of failed operations.

**Example:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

chunkData := bytes.NewReader([]byte("chunk content"))
err := store.Put(ctx, "chunk-abc123", chunkData)
if err != nil {
    if dfsErr, ok := err.(*errors.Error); ok && dfsErr.Code == errors.CodeInvalidArgument {
        log.Println("Chunk too large")
    } else {
        log.Fatal(err)
    }
}
```

---

### Get()

```go
func (ds *DiskStore) Get(ctx context.Context, chunkID string) (io.ReadCloser, error)
```

**Description:** Retrieves a chunk and returns a reader that validates checksums during streaming reads.

**Parameters:**
- `ctx`: Context for cancellation/timeout control
- `chunkID`: ID of chunk to retrieve

**Returns:** ReadCloser for validated chunk data, or error

**Behavior:**
1. Checks context cancellation
2. Opens chunk file
3. Reads and validates header (checksum and length)
4. Returns `verifiedReadCloser` that:
   - Streams chunk data
   - Computes SHA256 checksum during read
   - Validates checksum and length at EOF

**Errors:**
- `CodeNotFound`: Chunk file doesn't exist
- `CodeInternal`: Unable to open chunk file
- `CodeIntegrityViolation`: 
  - Invalid/unreadable header
  - Checksum mismatch
  - Data length mismatch

**Example:**
```go
reader, err := store.Get(ctx, "chunk-abc123")
if err != nil {
    if dfsErr, ok := err.(*errors.Error); ok {
        if dfsErr.Code == errors.CodeNotFound {
            log.Println("Chunk not found")
        } else if dfsErr.Code == errors.CodeIntegrityViolation {
            log.Println("Data corruption detected")
        }
    }
    return
}
defer reader.Close()

// Read and verify data
data, err := io.ReadAll(reader)
if err != nil {
    log.Fatal(err)
}
```

---

### Delete()

```go
func (ds *DiskStore) Delete(ctx context.Context, chunkID string) error
```

**Description:** Deletes a chunk from storage.

**Parameters:**
- `ctx`: Context (currently not actively used)
- `chunkID`: ID of chunk to delete

**Returns:** Error if deletion fails (nil if successful or chunk doesn't exist)

**Behavior:**
- Attempts to remove chunk file
- Returns success if file doesn't exist (idempotent)
- Returns error only for actual failure conditions

**Errors:**
- `CodeInternal`: Actual file system errors

**Idempotency:** Safe to call multiple times; not finding the chunk is not an error.

**Example:**
```go
err := store.Delete(ctx, "chunk-abc123")
if err != nil {
    log.Fatal(err)
}
// Success whether chunk existed or not
```

---

### Exists()

```go
func (ds *DiskStore) Exists(ctx context.Context, chunkID string) (bool, error)
```

**Description:** Checks if a chunk exists in storage.

**Parameters:**
- `ctx`: Context (currently not actively used)
- `chunkID`: ID of chunk to check

**Returns:** Boolean existence flag and error if check fails

**Errors:**
- `CodeInternal`: File stat errors

**Example:**
```go
exists, err := store.Exists(ctx, "chunk-abc123")
if err != nil {
    log.Fatal(err)
}
if exists {
    log.Println("Chunk found")
} else {
    log.Println("Chunk not found")
}
```

---

### Path()

```go
func (ds *DiskStore) Path(chunkID string) string
```

**Description:** Resolves a chunk ID to its filesystem path with directory sharding.

**Parameters:**
- `chunkID`: Chunk identifier

**Returns:** Full file path

**Sharding Strategy:**
- IDs ≤2 chars: `{baseDir}/{chunkID}.chunk`
- IDs >2 chars: `{baseDir}/{first2chars}/{remaining}.chunk`

**Example:**
```go
path := store.Path("ab1234567890")
// Returns: "/var/dfs/storage/chunks/ab/1234567890.chunk"

path := store.Path("xy")
// Returns: "/var/dfs/storage/chunks/xy.chunk"
```

---

## Usage Examples

### Basic Upload and Download

```go
import (
    "context"
    "bytes"
    "io"
    "github.com/rohanyadav1024/dfs/internal/storage/chunkstore"
)

func example() error {
    // Initialize store
    store, err := chunkstore.New("/var/dfs/storage")
    if err != nil {
        return err
    }
    
    ctx := context.Background()
    
    // Upload a chunk
    chunkData := []byte("Hello, Distributed File System!")
    if err := store.Put(ctx, "greeting-chunk", bytes.NewReader(chunkData)); err != nil {
        return err
    }
    
    // Download the chunk
    reader, err := store.Get(ctx, "greeting-chunk")
    if err != nil {
        return err
    }
    defer reader.Close()
    
    // Read and verify
    retrieved, err := io.ReadAll(reader)
    if err != nil {
        return err
    }
    
    println("Retrieved:", string(retrieved))
    return nil
}
```

### Cached Upload with Idempotency

```go
func cachedUpload(store chunkstore.Store, chunkID string, data []byte) error {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // First attempt
    err := store.Put(ctx, chunkID, bytes.NewReader(data))
    if err != nil {
        // Retry is safe - Put is idempotent
        // Sleep and retry
        time.Sleep(1 * time.Second)
        return store.Put(ctx, chunkID, bytes.NewReader(data))
    }
    return nil
}
```

### Integrity Verification

```go
func verifyChunkIntegrity(store chunkstore.Store, chunkID string, expectedData []byte) error {
    ctx := context.Background()
    
    reader, err := store.Get(ctx, chunkID)
    if err != nil {
        return err
    }
    defer reader.Close()
    
    actualData, err := io.ReadAll(reader)
    if err != nil {
        // Includes integrity errors from checksum mismatch
        return err
    }
    
    if !bytes.Equal(actualData, expectedData) {
        return errors.New(errors.CodeIntegrityViolation, "data mismatch")
    }
    
    return nil
}
```

### Chunk Storage with Timeout

```go
func storeChunkWithTimeout(store chunkstore.Store, chunkID string, reader io.Reader, timeout time.Duration) error {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    
    return store.Put(ctx, chunkID, reader)
}
```

### Batch Operations

```go
type ChunkData struct {
    ID   string
    Data []byte
}

func storeBatch(store chunkstore.Store, chunks []ChunkData) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    
    for i, chunk := range chunks {
        if err := store.Put(ctx, chunk.ID, bytes.NewReader(chunk.Data)); err != nil {
            return errors.Wrap(errors.CodeInternal, 
                fmt.Sprintf("failed to store chunk %d", i), err)
        }
    }
    
    return nil
}
```

---

## Directory Sharding Strategy

The two-level directory sharding prevents having too many files in a single directory:

**Advantages:**
- Avoids file system performance degradation with millions of files in one directory
- Distributes load across directories
- Makes filesystem operations more efficient
- Scales to billions of chunks

**Distribution:**
With 26 alphanumeric characters (2^129 possibilities per ULID), first 2 characters create ~676 max buckets, each containing potentially millions of chunks.

---

## Size Constraints

**Maximum Chunk Size:** 16MB (16,777,216 bytes)

**Reasoning:**
- Balances memory usage (read entirely into memory for checksum)
- Suitable for most file systems and networks
- Allows multiple chunks in flight simultaneously

**Handling Large Files:**
For files larger than 16MB, implement multi-chunk uploads:
```go
func uploadLargeFile(store chunkstore.Store, filePath string) error {
    file, err := os.Open(filePath)
    if err != nil {
        return err
    }
    defer file.Close()
    
    chunkIndex := 0
    for {
        chunk := make([]byte, 16*1024*1024)
        n, err := file.Read(chunk)
        if n == 0 && err == io.EOF {
            break
        }
        if err != nil {
            return err
        }
        
        chunkID := fmt.Sprintf("file_%d", chunkIndex)
        if err := store.Put(context.Background(), chunkID, bytes.NewReader(chunk[:n])); err != nil {
            return err
        }
        
        chunkIndex++
    }
    
    return nil
}
```

---

## Performance Characteristics

| Operation | Time | Notes |
|-----------|------|-------|
| Put (small chunk) | ~10-50ms | Includes disk I/O |
| Put (large chunk) | Scales with size | 16MB ≈ 100-200ms on typical disk |
| Get (small chunk) | ~5-20ms | Stream with verification |
| Get (large chunk) | Scales with size | Streaming reduces memory |
| Exists | ~1-5ms | File stat only |
| Delete | ~1-5ms | File removal |

---

## Error Handling Checklist

- [ ] Check `CodeNotFound` for missing chunks in Get
- [ ] Handle `CodeIntegrityViolation` for data corruption
- [ ] Check `CodeInvalidArgument` for oversized chunks
- [ ] Use retryable() for transient errors
- [ ] Log all storage errors with context
- [ ] Implement monitoring for integrity violations

---

## Best Practices

1. **Always Defer Close**: Close readers obtained from Get() to avoid file handle leaks.

2. **Use Context**: Always pass contexts with appropriate timeouts.

3. **Handle Integrity Errors**: Monitor and alert on CodeIntegrityViolation errors.

4. **Batch Operations**: Use goroutines with sync.WaitGroup for concurrent uploads.

5. **Verify After Upload**: Optionally verify chunk after upload for critical data.

6. **Monitor Storage**: Track disk usage and implement cleanup policies.

7. **Idempotent Uploads**: Rely on Put idempotency for retry logic.

8. **Size Chunks**: Split large files into 16MB or smaller chunks.

---

## Troubleshooting

### "Chunk too large" Error

**Problem:** Uploaded chunk exceeds 16MB limit

**Solution:** Split large files into smaller chunks:
```go
const chunkSize = 16 * 1024 * 1024 // 16MB

// Split and upload
for offset := 0; offset < totalSize; offset += chunkSize {
    end := offset + chunkSize
    if end > totalSize {
        end = totalSize
    }
    
    chunkID := fmt.Sprintf("file_%d", offset)
    store.Put(ctx, chunkID, bytes.NewReader(data[offset:end]))
}
```

### Checksum Mismatch After Upload

**Problem:** CodeIntegrityViolation when reading recently written chunk

**Possible Causes:**
- Disk corruption
- File system error
- Bug in stored data

**Investigation:**
- Check disk health
- Verify system logs
- Re-upload chunk with new ID
- Monitor for pattern

### Context Deadline Exceeded

**Problem:** Operations timeout during Put/Get

**Solution:** Increase timeout:
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()
```


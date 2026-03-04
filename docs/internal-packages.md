# Internal Packages Documentation

This document provides an overview of all packages in the `/internal` directory of the Distributed File System (DFS) project.

## Table of Contents

- [Common Package](#common-package)
  - [Config](#config)
  - [Errors](#errors)
  - [IDs](#ids)
  - [Logging](#logging)
- [Storage Package](#storage-package)
  - [Chunkstore](#chunkstore)
- [Future Packages](#future-packages)
  - [Metadata](#metadata)
  - [Observability](#observability)
  - [Protocol](#protocol)
  - [Security](#security)

---

## Common Package

The `common` package contains shared utilities and infrastructure components used across the DFS system.

### Config

**Location:** `internal/common/config/`

**Purpose:** Environment-based configuration management for the DFS services.

**Key Components:**
- `Config struct`: Main configuration structure containing service configuration and logging settings
  - `ServiceName`: Name of the service (default: "dfs-service")
  - `Log`: Logging configuration

**Key Functions:**
- `Load() Config`: Loads configuration from environment variables with sensible defaults
- `getEnv(key string, fallback string) string`: Helper function to safely read environment variables

**Environment Variables:**
- `SERVICE_NAME`: Service name (default: "dfs-service")
- `LOG_LEVEL`: Logging level (default: "debug")
- `LOG_PRODUCTION`: Production mode flag (default: "false")

**Usage Example:**
```go
import "github.com/rohanyadav1024/dfs/internal/common/config"

func main() {
    cfg := config.Load()
    // Use cfg for service initialization
}
```

---

### Errors

**Location:** `internal/common/errors/`

**Purpose:** Custom error handling with structured error codes to support proper error categorization and handling throughout the system.

**Key Components:**

#### Error Type
- `Error struct`: Custom error type containing:
  - `Code`: Error code categorizing the error type
  - `Message`: Human-readable error message
  - `Cause`: Underlying wrapped error

#### Error Codes
The following error codes are defined:
- `CodeInternal`: Internal/unexpected errors
- `CodeInvalidArgument`: Invalid input arguments
- `CodeNotFound`: Resource not found
- `CodeAlreadyExists`: Resource already exists
- `CodeUnauthenticated`: Authentication required
- `CodePermissionDenied`: Permission denied
- `CodeUnavailable`: Service unavailable
- `CodeIntegrityViolation`: Data integrity violation
- `CodeConflict`: Conflicting state/operation
- `CodeTimeout`: Operation timeout

**Key Methods:**
- `Error() string`: Implements error interface with cause chain
- `Unwrap() error`: Returns the underlying wrapped error for error chain inspection
- `Retryable() bool`: Determines if error is transient and retryable

**Key Functions:**
- `New(code Code, message string) *Error`: Creates a new error
- `Wrap(code Code, message string, cause error) *Error`: Wraps an existing error with context
- `From(err error) *Error`: Converts standard errors to custom Error type

**Usage Example:**
```go
import "github.com/rohanyadav1024/dfs/internal/common/errors"

func someOperation() error {
    // Create a new error
    return errors.New(errors.CodeNotFound, "resource not found")
}

func wrappingError() error {
    // Wrap an existing error
    data, err := io.ReadAll(reader)
    if err != nil {
        return errors.Wrap(errors.CodeInternal, "failed to read data", err)
    }
    return nil
}
```

---

### IDs

**Location:** `internal/common/ids/`

**Purpose:** Unique identifier generation for requests and uploads using ULID (Universally Unique Lexicographically Sortable Identifiers).

**Key Components:**
- ULID-based ID generation using monotonic entropy for uniqueness and ordering

**Key Functions:**
- `NewRequestID() string`: Generates a unique request ID using ULID with monotonic entropy
  - Guarantees uniqueness and ordering even when IDs are generated in the same millisecond
  - Uses current UTC timestamp and cryptographically secure random entropy

**Properties:**
- Lexicographically sortable (can be used for range queries)
- Monotonic (preserves ordering within the same millisecond)
- Case-insensitive alphanumeric format

**Usage Example:**
```go
import "github.com/rohanyadav1024/dfs/internal/common/ids"

func handleRequest(w http.ResponseWriter, r *http.Request) {
    requestID := ids.NewRequestID()
    // Use requestID for tracing and logging
}
```

---

### Logging

**Location:** `internal/common/logging/`

**Purpose:** Structured logging using Uber's Zap logger for the entire DFS system.

**Key Components:**

#### Config Type
```go
type Config struct {
    Level      string  // Log level: "debug", "info", "warn", "error"
    Production bool    // Production mode flag
}
```

**Key Functions:**
- `Init(cfg Config) error`: Initializes the global logger with the given configuration
  - Supports development and production log formats
  - Configurable log levels
  - Development mode includes stack traces; production mode optimizes for performance

- `L() *zap.Logger`: Returns the initialized global logger instance

**Log Levels:**
- `debug`: Detailed debug information
- `info`: General informational messages
- `warn`: Warning messages
- `error`: Error messages

**Usage Example:**
```go
import "github.com/rohanyadav1024/dfs/internal/common/logging"

func init() {
    cfg := logging.Config{
        Level:      "info",
        Production: false,
    }
    err := logging.Init(cfg)
    if err != nil {
        panic(err)
    }
}

func someFunction() {
    logging.L().Info("operation completed", zap.Int("duration", 100))
    logging.L().Error("operation failed", zap.Error(err))
}
```

---

## Storage Package

The `storage` package handles data persistence and retrieval in the DFS system.

### Chunkstore

**Location:** `internal/storage/chunkstore/`

**Purpose:** Disk-based storage system for immutable data chunks with built-in integrity verification using SHA256 checksums.

**Key Components:**

#### Store Interface
```go
type Store interface {
    Put(ctx context.Context, chunkID string, r io.Reader) error
    Get(ctx context.Context, chunkID string) (io.ReadCloser, error)
    Delete(ctx context.Context, chunkID string) error
    Exists(ctx context.Context, chunkID string) (bool, error)
}
```

#### DiskStore Implementation
File-system based implementation with the following features:
- **Atomic writes**: Uses temp files and atomic rename to prevent partial writes
- **Data integrity**: Stores SHA256 checksums with each chunk
- **On-the-fly verification**: Verifies checksums during reads
- **Deterministic sharding**: Distributes chunks in directory structure based on chunk ID
- **Idempotency**: Detects and handles duplicate writes correctly
- **Size constraints**: Enforces maximum chunk size (16MB)

**File Format:**
```
[32 bytes: SHA256 checksum][8 bytes: big-endian data length][variable: chunk data]
```

**Chunk Path Resolution:**
- Short IDs (≤2 chars): `{baseDir}/{chunkID}.chunk`
- Long IDs: `{baseDir}/{first2chars}/{remaining}.chunk`

**Key Methods:**

#### `New(baseDir string) (*DiskStore, error)`
Creates a new DiskStore instance and initializes the storage directory.

**Parameters:**
- `baseDir`: Base directory where chunks will be stored

**Returns:** DiskStore instance or error

#### `Put(ctx context.Context, chunkID string, r io.Reader) error`
Stores a chunk atomically with checksum verification.

**Features:**
- Reads entire chunk into memory
- Validates chunk size (max 16MB)
- Writes checksum header followed by length and data
- Uses atomic `Rename` for crash safety
- Idempotent: Returns success if identical chunk already exists

#### `Get(ctx context.Context, chunkID string) (io.ReadCloser, error)`
Retrieves a chunk with on-the-fly integrity verification.

**Features:**
- Reads checksum and length from header
- Returns a `verifiedReadCloser` that verifies data during streaming
- Detects mismatches in checksum or data length
- Proper file handle cleanup

#### `Delete(ctx context.Context, chunkID string) error`
Deletes a chunk from storage.

**Features:**
- Safely removes chunk file
- Idempotent: Succeeds even if chunk doesn't exist

#### `Exists(ctx context.Context, chunkID string) (bool, error)`
Checks if a chunk exists in storage.

**Returns:** Boolean indicating existence and any error

#### `Path(chunkID string) string`
Resolves a chunk ID to its filesystem path with sharding.

**Usage Example:**
```go
import "github.com/rohanyadav1024/dfs/internal/storage/chunkstore"

func storeChunk() {
    ctx := context.Background()
    store, err := chunkstore.New("/var/dfs/storage")
    if err != nil {
        log.Fatal(err)
    }

    // Store a chunk
    data := bytes.NewReader([]byte("chunk data"))
    err = store.Put(ctx, "abc123", data)
    if err != nil {
        log.Fatal(err)
    }

    // Retrieve the chunk
    reader, err := store.Get(ctx, "abc123")
    if err != nil {
        log.Fatal(err)
    }
    defer reader.Close()

    // Read verified data
    content, err := io.ReadAll(reader)
    if err != nil {
        log.Fatal(err)
    }

    // Check existence
    exists, err := store.Exists(ctx, "abc123")
    
    // Delete chunk
    err = store.Delete(ctx, "abc123")
}
```

**Context-Aware Operations:**
All methods respect context cancellation and timeouts, allowing proper cleanup and resource management in concurrent scenarios.

---

## Future Packages

The following packages are placeholders for future functionality:

### Metadata

**Location:** `internal/metadata/`

**Status:** Not yet implemented

**Purpose:** Will handle metadata operations for files, including directory structures, file properties, and hierarchical organization.

---

### Observability

**Location:** `internal/observability/`

**Status:** Not yet implemented

**Purpose:** Will handle observability features including metrics collection, tracing, and health checks for system monitoring.

---

### Protocol

**Location:** `internal/protocol/`

**Status:** Not yet implemented

**Purpose:** Will define communication protocols and message formats for inter-service communication.

---

### Security

**Location:** `internal/security/`

**Status:** Not yet implemented

**Purpose:** Will handle authentication, authorization, and encryption for the DFS system.

---

## Dependency Graph

```
storage/chunkstore
    └── common/errors

common/logging
    └── (external: go.uber.org/zap)

common/ids
    └── (external: github.com/oklog/ulid/v2)

common/config
    └── common/logging
```

---

## Best Practices

1. **Error Handling**: Always use the `errors` package for consistent error handling throughout the codebase.

2. **Logging**: Initialize logging at application startup and use `logging.L()` for all log statements.

3. **Configuration**: Load configuration once at startup using `config.Load()` and pass it to services.

4. **ID Generation**: Use `ids.NewRequestID()` for request tracking and distributed tracing.

5. **Chunk Storage**: 
   - Always use context with appropriate timeouts
   - Handle `CodeIntegrityViolation` errors specially as they indicate data corruption
   - Implement retry logic for `CodeUnavailable` errors


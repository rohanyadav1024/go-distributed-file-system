# Future Packages Documentation

This document describes the planned packages that are currently empty but will be part of the DFS (Distributed File System) architecture.

---

## Table of Contents

- [Metadata Package](#metadata-package)
- [Observability Package](#observability-package)
- [Protocol Package](#protocol-package)
- [Security Package](#security-package)

---

## Metadata Package

**Location:** `internal/metadata/`

**Status:** ⏳ Not yet implemented

### Purpose

The metadata package will handle all file and directory metadata operations in the DFS system. This includes:

- **File Metadata**: Properties like size, creation time, modification time, permissions
- **Directory Structure**: Hierarchical organization of files and folders
- **Metadata Storage**: Persistent storage of file/directory information
- **Metadata Operations**: CRUD operations for files and directories
- **Versioning**: Support for file versioning and snapshots
- **Transactions**: Atomic metadata operations across multiple files

### Planned Components

#### Types
- `File`: Represents a file with metadata
- `Directory`: Represents a directory with contents
- `FileInfo`: Detailed file information and properties
- `Metadata`: Common metadata attributes

#### Interfaces
- `Store`: Interface for metadata persistence
- `Service`: Service for metadata operations

#### Key Functions (Planned)
- `CreateFile()`: Create a new file entry
- `DeleteFile()`: Delete a file
- `GetFile()`: Retrieve file metadata
- `ListDirectory()`: List directory contents
- `UpdateFile()`: Update file properties
- `CreateDirectory()`: Create a directory
- `DeleteDirectory()`: Delete a directory
- `GetDirectory()`: Retrieve directory metadata

### Example Usage (Planned)

```go
// Will be used like this in the future
metadata, err := metadataService.GetFile(ctx, "/path/to/file.txt")
if err != nil {
    return err
}

fmt.Println("File size:", metadata.Size)
fmt.Println("Created:", metadata.CreatedAt)
fmt.Println("Modified:", metadata.ModifiedAt)
```

### Integration Points

- **Storage Package**: Will use chunkstore for managing actual file data
- **Protocol Package**: Will serialize metadata for RPC communication
- **Security Package**: Will enforce permission checks on metadata operations
- **Observability Package**: Will provide metrics for metadata operations

---

## Observability Package

**Location:** `internal/observability/`

**Status:** ⏳ Not yet implemented

### Purpose

The observability package will provide comprehensive monitoring, telemetry, and observability features for the DFS system:

- **Metrics**: Performance and operational metrics collection
- **Tracing**: Distributed request tracing across services
- **Health Checks**: Service health and readiness probes
- **Monitoring**: Real-time system monitoring capabilities
- **Alerting**: Integration with alerting systems

### Planned Components

#### Metrics Collection
- Request latency histograms
- Error rates and types
- Throughput (requests/sec)
- Resource utilization (CPU, memory, disk)
- Chunk storage metrics (size, count, access patterns)

#### Tracing
- Distributed request tracing with context propagation
- Trace sampling for performance
- Request attribution and correlation
- Service dependency mapping

#### Health Checks
- Liveness probes (service is running)
- Readiness probes (service can handle requests)
- Dependency health (storage, network)

#### Key Interfaces (Planned)
- `Metrics`: Interface for metrics collection
- `Tracer`: Interface for distributed tracing
- `HealthChecker`: Interface for health checks

#### Key Functions (Planned)
- `NewMetricsCollector()`: Create metrics collector
- `RecordLatency()`: Record operation latency
- `RecordError()`: Record error occurrence
- `StartSpan()`: Start a trace span
- `CheckHealth()`: Get service health status

### Example Usage (Planned)

```go
// Metrics
metrics := observability.NewMetrics()
start := time.Now()

// Do operation
data, err := store.Get(ctx, chunkID)

// Record metrics
metrics.RecordLatency("chunk_get", time.Since(start))
if err != nil {
    metrics.RecordError("chunk_get", err)
}

// Tracing
span := tracer.StartSpan("upload_chunk")
defer span.End()

// Health
health := observability.CheckHealth()
if !health.Ready {
    return errors.New(errors.CodeUnavailable, "service not ready")
}
```

### Integration Points

- **Logging Package**: Will work with existing logging infrastructure
- **All Packages**: Will instrument all operations with metrics and traces
- **Protocol Package**: Will serialize and propagate trace context

### Typical Metrics to Track

- Chunk upload latency (P50, P95, P99)
- Chunk download latency
- Error rate by error type
- Storage capacity and utilization
- Active connections
- Request queue depth

---

## Protocol Package

**Location:** `internal/protocol/`

**Status:** ⏳ Not yet implemented

### Purpose

The protocol package will define and implement communication protocols for inter-service communication in the DFS system:

- **Message Definitions**: Structured message formats for RPC
- **Serialization**: Encoding/decoding of messages
- **Service Interfaces**: Defined service contracts
- **Error Propagation**: Standardized error serialization
- **Versioning**: Protocol version management

### Planned Components

#### Protocol Types
- `Message`: Base message structure
- `Request`: RPC request format
- `Response`: RPC response format
- `Error`: Standardized error format

#### Serialization Formats (Candidate)
- protobuf (recommended for performance and schema evolution)
- JSON (for HTTP/REST APIs)

#### Planned Services
- **StorageService**: Upload/download chunk operations
- **MetadataService**: File/directory metadata operations
- **DiscoveryService**: Service registration and discovery

#### Key Components (Planned)

```go
// Message definitions
type ChunkUploadRequest struct {
    ChunkID  string
    Data     []byte
    Checksum string
}

type ChunkUploadResponse struct {
    Success  bool
    Message  string
    ChunkID  string
}

// Service definition
type StorageService interface {
    UploadChunk(ctx context.Context, req ChunkUploadRequest) (ChunkUploadResponse, error)
    DownloadChunk(ctx context.Context, chunkID string) (ChunkDownloadResponse, error)
    DeleteChunk(ctx context.Context, chunkID string) error
}
```

### Example Usage (Planned)

```go
// Client-side
client := protocol.NewStorageClient(serverAddr)
resp, err := client.UploadChunk(ctx, ChunkUploadRequest{
    ChunkID: "abc123",
    Data:    chunkData,
})

// Server-side
server := protocol.NewStorageServer(storage)
server.Register(metadataService)
server.Listen(":8080")
```

### Integration Points

- **Storage Package**: Exposes storage operations via protocol
- **Metadata Package**: Exposes metadata operations via protocol
- **Security Package**: Will add authentication/authorization to protocol
- **Observability Package**: Will instrument protocol messages

### Communication Patterns

- **RPC (Remote Procedure Call)**: Synchronous request-response
- **Streaming**: For large data transfers
- **Pub/Sub**: For event distribution (future)

---

## Security Package

**Location:** `internal/security/`

**Status:** ⏳ Not yet implemented

### Purpose

The security package will provide comprehensive security features for the DFS system:

- **Authentication**: User and service authentication
- **Authorization**: Access control and permission management
- **Encryption**: Data encryption in transit and at rest
- **Key Management**: Cryptographic key management
- **Audit Logging**: Security event logging

### Planned Components

#### Authentication
- User authentication (username/password, tokens)
- Service-to-service authentication (mTLS, API keys)
- Token validation and refresh
- Session management

#### Authorization
- Role-Based Access Control (RBAC)
- Attribute-Based Access Control (ABAC)
- Permission evaluation
- Resource-level access control

#### Encryption
- TLS/SSL for data in transit
- AES encryption for data at rest
- End-to-end encryption support

#### Key Management
- Key generation and storage
- Key rotation policies
- Key distribution and revocation
- Hardware security module (HSM) support

#### Key Interfaces (Planned)
- `Authenticator`: Interface for authentication
- `Authorizer`: Interface for authorization
- `EncryptionService`: Interface for encryption/decryption
- `KeyManager`: Interface for key management

#### Key Functions (Planned)
- `Authenticate()`: Verify user/service identity
- `Authorize()`: Check if action is permitted
- `Encrypt()`: Encrypt data
- `Decrypt()`: Decrypt data
- `ValidateToken()`: Validate authentication token

### Example Usage (Planned)

```go
// Authentication
token, err := security.Authenticate(username, password)
if err != nil {
    return errors.New(errors.CodeUnauthenticated, "invalid credentials")
}

// Authorization
allowed, err := security.Authorize(token, "upload_chunk", resourceID)
if err != nil || !allowed {
    return errors.New(errors.CodePermissionDenied, "not authorized")
}

// Encryption
encrypted, err := security.Encrypt(data, encryptionKey)
if err != nil {
    return err
}

// Storage
err = store.Put(ctx, chunkID, bytes.NewReader(encrypted))
```

### Integration Points

- **All Packages**: Will add security checks to all operations
- **Protocol Package**: Will add authentication to protocol messages
- **Logging Package**: Will log security events
- **Observability Package**: Will track security metrics

### Security Considerations

- **Minimum Privilege**: Start with no permissions, grant specifically
- **Defense in Depth**: Multiple layers of security
- **Audit Trail**: Log all security-relevant operations
- **Encryption Everywhere**: Data in transit and at rest
- **Key Rotation**: Regular cryptographic key rotation
- **Secret Management**: Secure storage of passwords and keys

---

## Development Timeline

The anticipated timeline for implementing these packages (subject to change):

| Package | Phase | Estimated Timeline |
|---------|-------|-------------------|
| Metadata | Phase 1 | Early development |
| Protocol | Phase 1 | Early development |
| Observability | Phase 2 | Post-initial release |
| Security | Phase 2 | Post-initial release |

---

## Implementation Notes

When implementing these packages, follow these principles:

1. **Interface-Based Design**: Define clear interfaces for each component
2. **Error Handling**: Use the existing error package consistently
3. **Logging**: Instrument code with proper logging
4. **Testing**: Include comprehensive unit and integration tests
5. **Documentation**: Document all public APIs
6. **Backward Compatibility**: Plan for API versioning
7. **Performance**: Consider performance implications of new features
8. **Dependencies**: Minimize external dependencies where possible

---

## Related Documentation

- [Internal Packages Overview](./internal-packages.md)
- [Common Errors Package](./common-errors-package.md)
- [Storage Chunkstore Package](./storage-chunkstore-package.md)
- [Common Logging Package](./common-logging-package.md)


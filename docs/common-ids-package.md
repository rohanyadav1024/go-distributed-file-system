# Common IDs Package Documentation

## Overview

The `ids` package provides unique identifier generation for distributed tracing and request tracking in the DFS system. It uses ULIDs (Universally Unique Lexicographically Sortable Identifiers) for guaranteed uniqueness, sortability, and monotonic ordering.

**Package Path:** `internal/common/ids`

---

## What is ULID?

ULID is a specification for universally unique IDs with the following properties:

- **128-bit identifier** (256-bit representation in string form)
- **Timestamp-based** (48-bit millisecond precision)
- **Lexicographically sortable** (can be used for range queries and sorting)
- **Monotonic** (IDs generated within the same millisecond are ordered)
- **Non-sequential** (includes random component for security)
- **Case-insensitive** (alphanumeric without confusing characters like l, o, 0, O)

**Format:** `[timestamp (10 chars)][randomness (16 chars)]`

**Example:** `01ARZ3NDEKTSV4RRFFQ69G5FAV`

---

## Functions

### NewRequestID()

```go
func NewRequestID() string
```

**Description:** Generates a new unique request ID using ULID with monotonic entropy.

**Returns:** String representation of the ULID (26 characters)

**Features:**
- Uses current UTC timestamp for temporal ordering
- Generates monotonic random entropy for uniqueness within millisecond
- Implements cryptographically secure randomness using `crypto/rand`
- Safe for concurrent use
- Panics on error (rare, only if random source fails)

**Example:**
```go
requestID := ids.NewRequestID()
// Output: "01ARZ3NDEKTSV4RRFFQ69G5FAV"
```

---

## Usage Patterns

### Request Tracing

```go
import "github.com/rohanyadav1024/dfs/internal/common/ids"

func handleRequest(w http.ResponseWriter, r *http.Request) {
    requestID := ids.NewRequestID()
    
    // Add to response headers
    w.Header().Set("X-Request-ID", requestID)
    
    // Log with request ID
    logging.L().Info("request received",
        zap.String("request_id", requestID),
        zap.String("method", r.Method),
        zap.String("path", r.URL.Path),
    )
}
```

### Distributed Tracing in Context

```go
type RequestContext struct {
    RequestID string
    UserID    string
    Timestamp time.Time
}

func createRequestContext() RequestContext {
    return RequestContext{
        RequestID: ids.NewRequestID(),
        Timestamp: time.Now(),
    }
}
```

### Upload Tracking

```go
type UploadSession struct {
    UploadID  string    // Generated with NewRequestID()
    ChunkIDs  []string
    CreatedAt time.Time
}

func startUpload() UploadSession {
    return UploadSession{
        UploadID:  ids.NewRequestID(),
        ChunkIDs:  make([]string, 0),
        CreatedAt: time.Now(),
    }
}
```

### Batch Operation Tracking

```go
func processBatch(items []Item) error {
    batchID := ids.NewRequestID()
    
    logging.L().Info("batch processing started",
        zap.String("batch_id", batchID),
        zap.Int("item_count", len(items)),
    )
    
    for i, item := range items {
        itemID := ids.NewRequestID()
        if err := processItem(item, itemID); err != nil {
            logging.L().Error("item processing failed",
                zap.String("batch_id", batchID),
                zap.String("item_id", itemID),
                zap.Error(err),
            )
            return err
        }
    }
    
    logging.L().Info("batch processing completed",
        zap.String("batch_id", batchID),
    )
    return nil
}
```

### API Request-Response Correlation

```go
type APIRequest struct {
    ID      string
    Path    string
    Method  string
    Headers map[string]string
}

type APIResponse struct {
    RequestID  string
    StatusCode int
    Body       interface{}
}

func handleAPI(w http.ResponseWriter, r *http.Request) {
    requestID := ids.NewRequestID()
    
    apiReq := APIRequest{
        ID:      requestID,
        Path:    r.URL.Path,
        Method:  r.Method,
        Headers: r.Header,
    }
    
    result, err := processRequest(apiReq)
    
    resp := APIResponse{
        RequestID:  requestID,
        StatusCode: http.StatusOK,
        Body:       result,
    }
    
    w.Header().Set("X-Request-ID", requestID)
    // ... write response
}
```

---

## Properties and Characteristics

### Sortability

ULIDs are lexicographically sortable, meaning they can be compared as strings:

```go
id1 := ids.NewRequestID() // Generated at time T1
time.Sleep(1 * time.Millisecond)
id2 := ids.NewRequestID() // Generated at time T2

// id1 < id2 because T1 < T2
if id1 < id2 {
    println("Correctly ordered")
}
```

### Monotonicity within Millisecond

Multiple IDs generated in the same millisecond maintain their generation order:

```go
id1 := ids.NewRequestID() // Generated at time T, randomness R1
id2 := ids.NewRequestID() // Generated at time T, randomness R2, where R2 > R1

// id1 < id2 due to monotonic randomness
```

### Uniqueness Guarantees

The combination of timestamp and cryptographic randomness ensures:
- Different milliseconds: Guaranteed unique (timestamp differs)
- Same millisecond: Guaranteed unique (randomness differs)
- Probability of collision: ~1 in 2^80 (astronomically low)

---

## Integration with Logging

```go
import (
    "github.com/rohanyadav1024/dfs/internal/common/ids"
    "github.com/rohanyadav1024/dfs/internal/common/logging"
)

func withRequestID(requestID string) zapcore.Field {
    return zap.String("request_id", requestID)
}

func logWithContext(requestID string, message string) {
    logging.L().Info(message, withRequestID(requestID))
}
```

---

## Performance Characteristics

- **Generation Time**: ~microseconds per ID
- **Memory**: One-time allocation for entropy source
- **Concurrency**: Safe for concurrent use from multiple goroutines
- **No External Dependencies**: Uses only standard Go libraries and crypto/rand

---

## Best Practices

1. **Generate Once Per Request**: Create one ID at the start of request handling.

2. **Propagate Consistently**: Pass request ID through function calls and external API calls.

3. **Include in Logs**: Always include request ID in log statements for traceability.

4. **Add to Response Headers**: Include request ID in API responses for client reference.

5. **Database Indexing**: Create indices on request ID columns for query performance.

6. **Use for Correlation**: Use request IDs to correlate logs across multiple services in distributed systems.

7. **Never Reuse**: Never attempt to reuse or recreate IDs; generation is cheap.

---

## Comparison with Alternatives

| Aspect | ULID | UUID | Sequential |
|--------|------|------|-----------|
| Sortable | ✅ | ❌ | ✅ |
| Unique | ✅ | ✅ | ❌ |
| Random | ✅ | ✅ | ❌ |
| Text Size | 26 chars | 36 chars | Variable |
| Database Index | Efficient | Inefficient | Efficient |
| Monotonic | ✅ | ❌ | ✅ |

---

## Troubleshooting

### Panics on NewRequestID()

The only cause is a failure in the cryptographic random number generator, which is extremely rare and indicates a system-level issue.

```go
// Current implementation panics on error
// For production systems, consider wrapping in error handling
func SafeNewRequestID() (string, error) {
    defer func() {
        if r := recover(); r != nil {
            // Handle panic
        }
    }()
    return ids.NewRequestID(), nil
}
```

---

## Testing with IDs

```go
func TestRequestTracking(t *testing.T) {
    id1 := ids.NewRequestID()
    id2 := ids.NewRequestID()
    
    if id1 == id2 {
        t.Fatal("IDs should be unique")
    }
    
    if id1 >= id2 {
        t.Fatal("IDs should be ordered")
    }
    
    if len(id1) != 26 {
        t.Fatalf("Expected 26 characters, got %d", len(id1))
    }
}
```


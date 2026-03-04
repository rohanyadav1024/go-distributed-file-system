# Common Errors Package Documentation

## Overview

The `errors` package provides a structured error handling system with categorized error codes. This enables consistent error handling, proper error propagation, and intelligent retry logic throughout the DFS system.

**Package Path:** `internal/common/errors`

---

## Types

### Code

```go
type Code string
```

**Description:** String-based error code type for categorizing errors.

---

### Error

```go
type Error struct {
    Code    Code
    Message string
    Cause   error
}
```

**Fields:**
- `Code`: Categorizes the type of error (see Error Codes section)
- `Message`: Human-readable error description
- `Cause`: The underlying error that was wrapped (can be nil)

---

## Error Codes

The following error codes are defined as constants:

| Code | Usage |
|------|-------|
| `CodeInternal` | Unexpected internal errors |
| `CodeInvalidArgument` | Invalid input arguments or parameters |
| `CodeNotFound` | Requested resource not found |
| `CodeAlreadyExists` | Resource already exists |
| `CodeUnauthenticated` | Authentication required but not provided |
| `CodePermissionDenied` | User lacks required permissions |
| `CodeUnavailable` | Service temporarily unavailable (transient) |
| `CodeIntegrityViolation` | Data integrity check failed |
| `CodeConflict` | Operation conflicts with current state (transient) |
| `CodeTimeout` | Operation exceeded timeout (transient) |

### Transient vs Permanent Errors

**Transient errors** (safe to retry):
- `CodeUnavailable`
- `CodeTimeout`
- `CodeConflict`

**Permanent errors** (not safe to retry):
- `CodeNotFound`
- `CodeInvalidArgument`
- `CodePermissionDenied`
- `CodeUnauthenticated`

---

## Methods

### Error()

```go
func (e *Error) Error() string
```

**Description:** Implements the standard Go error interface.

**Example:**
```go
err := errors.New(errors.CodeNotFound, "user not found")
println(err.Error()) // Output: "user not found"
```

---

### Unwrap()

```go
func (e *Error) Unwrap() error
```

**Description:** Returns the wrapped error, enabling error chain inspection. Implements Go 1.13+ error unwrapping.

**Example:**
```go
original := errors.New(codes.CodeInternal, "read failed")
wrapped := errors.Wrap(codes.CodeInternal, "failed to load data", original)
fmt.Printf("%v\n", errors.Unwrap(wrapped)) // Original error
```

---

### Retryable()

```go
func (e *Error) Retryable() bool
```

**Description:** Determines if the error is transient and safe to retry.

**Returns:** `true` for transient errors (`CodeUnavailable`, `CodeTimeout`, `CodeConflict`)

**Example:**
```go
err := someOperation()
if dfsErr, ok := err.(*errors.Error); ok {
    if dfsErr.Retryable() {
        // Safe to retry with backoff
        retry()
    } else {
        // Permanent error, return to user
        return err
    }
}
```

---

## Functions

### New()

```go
func New(code Code, message string) *Error
```

**Description:** Creates a new error with the specified code and message.

**Parameters:**
- `code`: Error code categorizing the error
- `message`: Human-readable error message

**Returns:** Pointer to new Error

**Example:**
```go
err := errors.New(errors.CodeNotFound, "chunk abc123 not found")
```

---

### Wrap()

```go
func Wrap(code Code, message string, cause error) *Error
```

**Description:** Wraps an existing error with additional context while preserving the original error.

**Parameters:**
- `code`: Error code for this error
- `message`: New error message/context
- `cause`: The underlying error to wrap

**Returns:** Pointer to new Error

**Example:**
```go
data, err := ioutil.ReadFile(path)
if err != nil {
    return errors.Wrap(errors.CodeInternal, "failed to read chunk", err)
}
```

---

### From()

```go
func From(err error) *Error
```

**Description:** Converts a standard Go error to an Error type. If already an Error, returns as-is.

**Parameters:**
- `err`: Error to convert

**Returns:** 
- nil if input is nil
- The Error unchanged if it's already an Error
- New Error with CodeInternal if it's a standard error

**Example:**
```go
// Convert standard library errors
file, err := os.Open("chunk.dat")
if err != nil {
    dfsErr := errors.From(err)
    // Now it's properly typed with CodeInternal
}

// Safe with already-typed errors
var existingErr *errors.Error
dfsErr := errors.From(existingErr) // Returns existingErr unchanged
```

---

## Usage Patterns

### Simple Error Creation

```go
func validateChunkID(chunkID string) error {
    if chunkID == "" {
        return errors.New(errors.CodeInvalidArgument, "chunkID cannot be empty")
    }
    return nil
}
```

### Error Wrapping with Context

```go
func readChunk(path string) ([]byte, error) {
    data, err := ioutil.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, errors.New(errors.CodeNotFound, "chunk file not found")
        }
        return nil, errors.Wrap(errors.CodeInternal, "failed to read chunk file", err)
    }
    return data, nil
}
```

### Retry Logic with Error Inspection

```go
func retryableOperation(fn func() error) error {
    maxRetries := 3
    for attempt := 0; attempt < maxRetries; attempt++ {
        err := fn()
        if err == nil {
            return nil
        }
        
        dfsErr, ok := err.(*errors.Error)
        if !ok {
            return errors.From(err)
        }
        
        if !dfsErr.Retryable() {
            return err
        }
        
        // Exponential backoff and retry
        time.Sleep(time.Duration(math.Pow(2, float64(attempt))) * time.Second)
    }
    return errors.New(errors.CodeUnavailable, "operation failed after retries")
}
```

### Error Chain Inspection

```go
func handleError(err error) {
    dfsErr, ok := err.(*errors.Error)
    if !ok {
        dfsErr = errors.From(err)
    }
    
    switch dfsErr.Code {
    case errors.CodeNotFound:
        // Return 404
    case errors.CodePermissionDenied:
        // Return 403
    case errors.CodeInvalidArgument:
        // Return 400
    case errors.CodeInternal:
        // Return 500
    default:
        // Return 500
    }
}
```

### Preserving Error Chains

```go
func complexOperation() error {
    if err := step1(); err != nil {
        return errors.Wrap(errors.CodeInternal, "step1 failed", err)
    }
    
    if err := step2(); err != nil {
        return errors.Wrap(errors.CodeInternal, "step2 failed", err)
    }
    
    return nil
}

// Inspecting error chains
err := complexOperation()
for err != nil {
    dfsErr := errors.From(err)
    log.Println(dfsErr.Message)
    err = dfsErr.Unwrap()
}
```

---

## Best Practices

1. **Use Specific Codes**: Choose error codes that accurately represent the failure reason.

2. **Add Context**: When wrapping errors, include relevant context in the message.

3. **Check Retryability**: Use `Retryable()` to implement intelligent retry logic.

4. **Error Chain Preservation**: Always wrap errors to maintain the full context chain.

5. **Consistent Handling**: Convert all errors to the Error type for consistent handling.

6. **HTTP Response Mapping**: Map error codes to appropriate HTTP status codes:
   - 400: CodeInvalidArgument
   - 401: CodeUnauthenticated
   - 403: CodePermissionDenied
   - 404: CodeNotFound
   - 409: CodeAlreadyExists, CodeConflict
   - 500: CodeInternal, CodeIntegrityViolation
   - 503: CodeUnavailable
   - 504: CodeTimeout

---

## Error Handling Checklist

- [ ] Use appropriate error codes for the error type
- [ ] Wrap errors with additional context
- [ ] Check error retryability before implementing retry logic
- [ ] Log full error chains for debugging
- [ ] Map errors to correct HTTP status codes in APIs
- [ ] Test error cases alongside success paths


# Common Logging Package Documentation

## Overview

The `logging` package provides structured logging using Uber's Zap logger. It offers a high-performance, production-ready logging solution with support for different log levels and output formats.

**Package Path:** `internal/common/logging`

**External Dependency:** [go.uber.org/zap](https://github.com/uber-go/zap)

---

## Types

### Config

```go
type Config struct {
    Level      string
    Production bool
}
```

**Fields:**
- `Level` (string): Logging level controlling verbosity ("debug", "info", "warn", "error")
- `Production` (bool): Production mode flag affecting log formatting and performance

---

## Functions

### Init()

```go
func Init(cfg Config) error
```

**Description:** Initializes the global logger with the provided configuration. Must be called once at application startup before any logging.

**Parameters:**
- `cfg`: Configuration struct for logging setup

**Returns:** Error if initialization fails

**Behavior:**
- **Development Mode** (`Production: false`):
  - Pretty-printed output for readability
  - Includes stack traces for all messages
  - Should not be used in production (lower performance)

- **Production Mode** (`Production: true`):
  - JSON-formatted output for machine parsing
  - Optimized for performance
  - Only stack traces for error level and above

**Log Levels Supported:**
| Level | Text | Verbosity |
|-------|------|-----------|
| Debug | "debug" | All messages |
| Info | "info" | Info and above |
| Warn | "warn" | Warn and above |
| Error | "error" | Errors only |

**Example:**
```go
func main() {
    cfg := logging.Config{
        Level:      "info",
        Production: true,
    }
    
    if err := logging.Init(cfg); err != nil {
        panic(err)
    }
    
    // Logging is now ready
    logging.L().Info("Application started")
}
```

---

### L()

```go
func L() *zap.Logger
```

**Description:** Returns the initialized global logger instance. Must be called after `Init()`.

**Returns:** Pointer to the global Zap logger

**Panics:** If logger hasn't been initialized with `Init()`

**Example:**
```go
logging.L().Info("Information message")
logging.L().Warn("Warning message")
logging.L().Error("Error message", zap.Error(err))
```

---

## Zap Logger Usage

The package returns a `*zap.Logger` instance. Here are common logging patterns:

### Basic Logging

```go
logger := logging.L()

// Info level
logger.Info("Operation completed")

// Warn level
logger.Warn("Configuration not found, using defaults")

// Error level
logger.Error("Failed to process request", zap.Error(err))

// Debug level
logger.Debug("Request details", zap.Any("payload", data))
```

### Structured Logging with Fields

```go
logger := logging.L()

logger.Info("User created",
    zap.String("username", "john_doe"),
    zap.Int("user_id", 123),
    zap.Time("created_at", time.Now()),
)

logger.Error("Database error",
    zap.Error(err),
    zap.String("query", "SELECT * FROM users"),
    zap.Duration("duration", 5*time.Second),
)
```

### Common Zap Field Types

```go
import "go.uber.org/zap"

logger := logging.L()

// String fields
logger.Info("message", zap.String("key", "value"))

// Integer fields
logger.Info("message", zap.Int("count", 42))
logger.Info("message", zap.Int64("size", 1024))

// Boolean fields
logger.Info("message", zap.Bool("enabled", true))

// Error field
logger.Error("message", zap.Error(err))

// Duration field
logger.Info("message", zap.Duration("elapsed", 100*time.Millisecond))

// Time field
logger.Info("message", zap.Time("timestamp", time.Now()))

// Any value (uses reflection, slower)
logger.Info("message", zap.Any("data", complexObject))

// Multiple values of same type
logger.Info("message",
    zap.Strings("tags", []string{"tag1", "tag2"}),
    zap.Ints("counts", []int{1, 2, 3}),
)
```

---

## Integration Patterns

### Initialization with Config Package

```go
import (
    "github.com/rohanyadav1024/dfs/internal/common/config"
    "github.com/rohanyadav1024/dfs/internal/common/logging"
)

func initializeDFS() error {
    cfg := config.Load()
    
    if err := logging.Init(cfg.Log); err != nil {
        return err
    }
    
    logging.L().Info("DFS initialized",
        zap.String("service", cfg.ServiceName),
    )
    
    return nil
}
```

### Service Initialization

```go
type Service struct {
    logger *zap.Logger
}

func NewService() (*Service, error) {
    logger := logging.L()
    
    logger.Info("Service initializing")
    
    return &Service{
        logger: logger,
    }, nil
}

func (s *Service) Operation() error {
    s.logger.Info("Operation started")
    
    // Perform operation
    
    if err != nil {
        s.logger.Error("Operation failed", zap.Error(err))
        return err
    }
    
    s.logger.Info("Operation completed")
    return nil
}
```

### Middleware for HTTP Requests

```go
import "net/http"

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        logger := logging.L()
        logger.Info("HTTP request received",
            zap.String("method", r.Method),
            zap.String("path", r.URL.Path),
            zap.String("remote_addr", r.RemoteAddr),
        )
        
        next.ServeHTTP(w, r)
        
        duration := time.Since(start)
        logger.Info("HTTP request completed",
            zap.String("method", r.Method),
            zap.String("path", r.URL.Path),
            zap.Duration("duration", duration),
        )
    })
}
```

---

## Usage Examples

### Development Setup

```go
package main

import (
    "github.com/rohanyadav1024/dfs/internal/common/logging"
)

func main() {
    cfg := logging.Config{
        Level:      "debug",
        Production: false,
    }
    
    if err := logging.Init(cfg); err != nil {
        panic(err)
    }
    
    // Example logs
    logging.L().Debug("Debug message for development")
    logging.L().Info("Info message")
    logging.L().Warn("Warning message")
}
```

### Production Setup

```go
package main

import (
    "github.com/rohanyadav1024/dfs/internal/common/logging"
    "go.uber.org/zap"
)

func main() {
    cfg := logging.Config{
        Level:      "info",
        Production: true,
    }
    
    if err := logging.Init(cfg); err != nil {
        panic(err)
    }
    
    logger := logging.L()
    logger.Info("Application started", zap.String("version", "1.0.0"))
    
    // Debug messages won't appear in production
    logger.Debug("Debug info") // Not logged in production
}
```

### Error Handling and Logging

```go
func readFile(path string) ([]byte, error) {
    data, err := ioutil.ReadFile(path)
    if err != nil {
        logging.L().Error("Failed to read file",
            zap.String("path", path),
            zap.Error(err),
        )
        return nil, err
    }
    
    logging.L().Info("File read successfully",
        zap.String("path", path),
        zap.Int("bytes", len(data)),
    )
    
    return data, nil
}
```

### Context-Aware Logging

```go
type contextLogger struct {
    fields []zap.Field
}

func (cl *contextLogger) Log(message string) {
    logging.L().Info(message, append(cl.fields, zap.String("context", "operation"))...)
}

func processRequest(requestID string, userID string) {
    ctx := &contextLogger{
        fields: []zap.Field{
            zap.String("request_id", requestID),
            zap.String("user_id", userID),
        },
    }
    
    ctx.Log("Processing started")
    // ... processing
    ctx.Log("Processing completed")
}
```

---

## Log Output Examples

### Development Output
```
2024-03-03T10:15:22.123-0800	INFO	main/main.go:123	Application started	{"component": "dfs", "version": "1.0.0"}
2024-03-03T10:15:22.134-0800	WARN	main/main.go:145	Deprecated setting used	{"setting": "old_config"}
main.main
	/path/to/main.go:145
```

### Production Output (JSON)
```json
{"level":"info","ts":1709471722.123,"caller":"main/main.go:123","msg":"Application started","component":"dfs","version":"1.0.0"}
{"level":"warn","ts":1709471722.134,"caller":"main/main.go:145","msg":"Deprecated setting used","setting":"old_config"}
```

---

## Performance Considerations

1. **Lazy Evaluation**: Zap doesn't format messages until needed
2. **Allocation-free**: Structured logging with minimal allocations
3. **Level-based Filtering**: Messages below configured level are skipped entirely
4. **Production vs Dev**: Production mode is significantly faster due to JSON output

### Performance Tips

1. **Use correct log level**: Don't use Info for debug messages
2. **Prefer structured fields**: Use `zap.String()` instead of `zap.Any()`
3. **Avoid fmt.Sprintf**: Use structured logging, not string formatting
4. **Batch operations**: Don't log inside tight loops

---

## Best Practices

1. **Initialize Once**: Call `Init()` once at application startup.

2. **Use Global Instance**: Access logger via `L()` rather than passing logger dependencies.

3. **Structured Fields**: Use typed fields (e.g., `zap.String`) not `zap.Any()`.

4. **Meaningful Messages**: Keep log messages concise but descriptive.

5. **Error Context**: Always include relevant error context when logging errors.

6. **Appropriate Levels**:
   - **Debug**: Development/troubleshooting information
   - **Info**: Normal operational events
   - **Warn**: Potentially problematic situations
   - **Error**: Error conditions requiring attention

7. **Request Tracing**: Include request IDs in logs for correlation.

---

## Troubleshooting

### Logger Returns Nil
**Problem:** Calling `L()` before `Init()`

**Solution:** Ensure `Init()` is called during application startup:
```go
func main() {
    cfg := logging.Config{Level: "info", Production: false}
    logging.Init(cfg) // Do this first
    logging.L().Info("Ready")
}
```

### No Output in Development
**Problem:** Log level set too high

**Solution:** Set level to "debug" for development:
```go
cfg := logging.Config{
    Level:      "debug",  // Include all levels
    Production: false,
}
```

### JSON Format in Development
**Problem:** Config set to Production mode

**Solution:** Use development mode for readability:
```go
cfg := logging.Config{
    Level:      "debug",
    Production: false,  // Pretty-printed output
}
```


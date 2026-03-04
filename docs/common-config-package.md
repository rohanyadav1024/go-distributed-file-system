# Common Config Package Documentation

## Overview

The `config` package provides environment-based configuration management for DFS services. It simplifies configuration loading with sensible defaults and validates environment variable availability.

**Package Path:** `internal/common/config`

---

## Types

### Config

```go
type Config struct {
    ServiceName string
    Log logging.Config
}
```

**Fields:**
- `ServiceName` (string): Human-readable name of the service. Used for identification in logs and monitoring.
- `Log` (logging.Config): Logging configuration including log level and production mode flag.

---

## Functions

### Load()

```go
func Load() Config
```

**Description:** Loads configuration from environment variables with fallback defaults.

**Environment Variables:**
| Variable | Default | Description |
|----------|---------|-------------|
| `SERVICE_NAME` | `dfs-service` | Name of the service |
| `LOG_LEVEL` | `debug` | Logging level (debug, info, warn, error) |
| `LOG_PRODUCTION` | `false` | Production mode flag (true/false) |

**Returns:** A Config struct with loaded values

**Example:**
```go
cfg := config.Load()
fmt.Println(cfg.ServiceName) // "dfs-service" or custom value
```

---

## Usage Examples

### Basic Configuration Loading

```go
package main

import (
    "log"
    "github.com/rohanyadav1024/dfs/internal/common/config"
    "github.com/rohanyadav1024/dfs/internal/common/logging"
)

func main() {
    // Load configuration from environment
    cfg := config.Load()
    
    // Initialize logging with loaded config
    if err := logging.Init(cfg.Log); err != nil {
        log.Fatal(err)
    }
    
    // Use configuration
    println("Service:", cfg.ServiceName)
}
```

### Setting Custom Configuration in Development

```bash
# Export environment variables before running
export SERVICE_NAME=metadata-service
export LOG_LEVEL=info
export LOG_PRODUCTION=false

# Run application
go run ./cmd/metad/main.go
```

### Production Setup

```bash
# Production environment
export SERVICE_NAME=storage-service
export LOG_LEVEL=error
export LOG_PRODUCTION=true

# Run application
./storaged
```

---

## Integration with Other Packages

The Config package integrates closely with the `logging` package:

```go
import (
    "github.com/rohanyadav1024/dfs/internal/common/config"
    "github.com/rohanyadav1024/dfs/internal/common/logging"
)

func initializeService() error {
    // Load all configuration
    cfg := config.Load()
    
    // Initialize logging with config
    if err := logging.Init(cfg.Log); err != nil {
        return err
    }
    
    // Now logging is ready
    logging.L().Info("Service started", zap.String("name", cfg.ServiceName))
    
    return nil
}
```

---

## Best Practices

1. **Single Load**: Load configuration once at application startup, not on every request.

2. **Pass Configuration**: Pass the loaded Config struct to service constructors rather than reloading.

3. **Environment-First**: Rely on environment variables for configuration to support containerized deployments.

4. **Defaults**: The package provides sensible defaults for all configuration values.

5. **Logging Integration**: Always initialize logging immediately after loading configuration.

---

## Testing Configuration

When testing code that uses configuration:

```go
package main

import (
    "testing"
    "github.com/rohanyadav1024/dfs/internal/common/config"
)

func TestWithConfiguration(t *testing.T) {
    // Load default configuration for testing
    cfg := config.Load()
    
    if cfg.ServiceName == "" {
        t.Fatal("ServiceName should not be empty")
    }
}
```

---

## Future Enhancements

- Support for configuration files (YAML/JSON)
- Configuration validation
- Hot reload capabilities
- Environment-specific profiles

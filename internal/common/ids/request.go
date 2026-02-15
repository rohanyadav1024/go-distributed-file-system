package ids

import (
	"github.com/oklog/ulid/v2"
	"crypto/rand"
	"time"
)

// NewRequestID generates a new unique request ID using ULID.
// ULID is a universally unique lexicographically sortable identifier.
func NewRequestID() string {
	t := time.Now().UTC()

	// Monotonic entropy source for ULID generation ensures 
	// that IDs generated in the same millisecond are unique and ordered.
	entropy := ulid.Monotonic(rand.Reader, 0)

	// Generate a new ULID using the current timestamp and the monotonic entropy source.
	id, err := ulid.New(ulid.Timestamp(t), entropy)

	if err != nil {
		panic("failed to generate request ID: " + err.Error()) // Handle error appropriately in production code
	}

	return id.String()
}
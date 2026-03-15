// Package ids generates stable identifiers used across DFS services.
package ids

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

// NewRequestID returns a lexicographically sortable unique request ID.
func NewRequestID() string {
	t := time.Now().UTC()

	entropy := ulid.Monotonic(rand.Reader, 0)

	id, err := ulid.New(ulid.Timestamp(t), entropy)
	if err != nil {
		panic("failed to generate request ID: " + err.Error())
	}

	return id.String()
}

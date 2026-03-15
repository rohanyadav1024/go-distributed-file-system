// Package policy defines chunk sizing policy for uploaded files.
package policy

import "fmt"

// ChunkSizeSlot describes a named chunk size option.
type ChunkSizeSlot struct {
	Label string
	Bytes int64
}

// Policy chooses chunk sizes for metadata planning.
type Policy struct {
	slots       []ChunkSizeSlot
	defaultSize int64
}

// NewPolicy returns the default chunk size policy.
func NewPolicy() *Policy {
	return &Policy{
		slots: []ChunkSizeSlot{
			{Label: "small", Bytes: 1 * 1024 * 1024},   // 1 MB
			{Label: "medium", Bytes: 4 * 1024 * 1024},  // 4 MB
			{Label: "large", Bytes: 8 * 1024 * 1024},   // 8 MB
			{Label: "xlarge", Bytes: 16 * 1024 * 1024}, // 16 MB
		},
		defaultSize: 4 * 1024 * 1024, // 4 MB default
	}
}

// DetermineChunkSize returns the chunk size to use for a file upload.
func (p *Policy) DetermineChunkSize(fileName string, fileSize int64) (int64, error) {
	if fileSize < 0 {
		return 0, fmt.Errorf("invalid file size: %d", fileSize)
	}

	return p.defaultSize, nil
}

// GetSlots returns all configured chunk size options.
func (p *Policy) GetSlots() []ChunkSizeSlot {
	return p.slots
}

// GetDefaultSize returns the policy default chunk size.
func (p *Policy) GetDefaultSize() int64 {
	return p.defaultSize
}

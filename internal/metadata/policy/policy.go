package policy

import "fmt"

// ChunkSizeSlot represents a predefined chunk size with a label
type ChunkSizeSlot struct {
	Label string
	Bytes int64
}

// Policy determines chunk sizes based on file characteristics
type Policy struct {
	slots       []ChunkSizeSlot
	defaultSize int64
}

// NewPolicy creates a new chunk size policy with predefined slots
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

// DetermineChunkSize determines the appropriate chunk size for a file
// Currently returns the default size (4 MB)
// TODO: In the future, implement logic based on fileName, fileSize, access patterns, etc.
func (p *Policy) DetermineChunkSize(fileName string, fileSize int64) (int64, error) {
	if fileSize < 0 {
		return 0, fmt.Errorf("invalid file size: %d", fileSize)
	}

	// For now, always return the default chunk size
	// Future enhancements could select based on:
	// - File extension (video files → larger chunks)
	// - File size (very large files → larger chunks)
	// - File type prefix conventions
	// - Dynamic configuration per file class

	return p.defaultSize, nil
}

// GetSlots returns all available chunk size slots
func (p *Policy) GetSlots() []ChunkSizeSlot {
	return p.slots
}

// GetDefaultSize returns the default chunk size
func (p *Policy) GetDefaultSize() int64 {
	return p.defaultSize
}

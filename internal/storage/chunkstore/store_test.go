package chunkstore

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	customerrors "github.com/rohanyadav1024/dfs/internal/common/errors"
)

func newTestStore(t *testing.T) (*DiskStore, string) {
	t.Helper()

	dir := t.TempDir()

	store, err := New(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	return store, dir
}

func TestPutGetRoundTrip(t *testing.T) {
	store, _ := newTestStore(t)

	data := bytes.Repeat([]byte("a"), 4*1024) // 4KB for test
	ctx := context.Background()

	err := store.Put(ctx, "chunk1", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	rc, err := store.Get(ctx, "chunk1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if !bytes.Equal(got, data) {
		t.Fatalf("data mismatch")
	}
}

func TestIdempotentPut(t *testing.T) {
	store, _ := newTestStore(t)

	data := []byte("hello world")
	ctx := context.Background()

	if err := store.Put(ctx, "chunk2", bytes.NewReader(data)); err != nil {
		t.Fatalf("first Put failed: %v", err)
	}

	// same data again
	if err := store.Put(ctx, "chunk2", bytes.NewReader(data)); err != nil {
		t.Fatalf("idempotent Put should succeed, got: %v", err)
	}
}

func TestIntegrityViolation(t *testing.T) {
	store, _ := newTestStore(t)

	data := []byte("original data")
	ctx := context.Background()

	if err := store.Put(ctx, "chunk3", bytes.NewReader(data)); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Manually corrupt file
	path := store.Path("chunk3")
	f, err := os.OpenFile(path, os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("failed to open file for corruption: %v", err)
	}

	// overwrite part of data (after header)
	if _, err := f.WriteAt([]byte("corrupt"), headerSize); err != nil {
		t.Fatalf("failed to corrupt file: %v", err)
	}
	f.Close()

	rc, err := store.Get(ctx, "chunk3")
	if err != nil {
		t.Fatalf("Get should succeed initially: %v", err)
	}
	defer rc.Close()

	_, err = io.ReadAll(rc)
	if err == nil {
		t.Fatalf("expected integrity violation error")
	}

	e := customerrors.From(err)
	if e.Code != customerrors.CodeIntegrityViolation {
		t.Fatalf("expected integrity violation, got %v", e.Code)
	}
}

func TestExistsAndDelete(t *testing.T) {
	store, _ := newTestStore(t)

	ctx := context.Background()
	data := []byte("test")

	if err := store.Put(ctx, "chunk4", bytes.NewReader(data)); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	exists, err := store.Exists(ctx, "chunk4")
	if err != nil || !exists {
		t.Fatalf("Exists should return true")
	}

	if err := store.Delete(ctx, "chunk4"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	exists, err = store.Exists(ctx, "chunk4")
	if err != nil || exists {
		t.Fatalf("Exists should return false after delete")
	}
}

func TestContextCancellation(t *testing.T) {
	store, _ := newTestStore(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := store.Put(ctx, "chunk5", bytes.NewReader([]byte("data")))
	if err == nil {
		t.Fatalf("expected cancellation error")
	}
}

func TestChunkNotFound(t *testing.T) {
	store, _ := newTestStore(t)

	ctx := context.Background()

	_, err := store.Get(ctx, "does-not-exist")
	if err == nil {
		t.Fatalf("expected not found error")
	}

	e := customerrors.From(err)
	if e.Code != customerrors.CodeNotFound {
		t.Fatalf("expected CodeNotFound, got %v", e.Code)
	}
}
package watcher_test

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/donaldgifford/mdp/internal/watcher"
)

func TestWatcher_DetectsFileChange(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "test.md")
	if err := os.WriteFile(file, []byte("# Original"), 0o644); err != nil {
		t.Fatalf("writing initial file: %v", err)
	}

	var called atomic.Int32
	w, err := watcher.New(file, 20*time.Millisecond, func() {
		called.Add(1)
	})
	if err != nil {
		t.Fatalf("creating watcher: %v", err)
	}
	go w.Start()
	defer func() {
		if closeErr := w.Close(); closeErr != nil {
			t.Errorf("closing watcher: %v", closeErr)
		}
	}()

	// Give fsnotify time to register.
	time.Sleep(50 * time.Millisecond)

	// Modify the file.
	if err := os.WriteFile(file, []byte("# Updated"), 0o644); err != nil {
		t.Fatalf("updating file: %v", err)
	}

	// Wait for debounce + processing.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if called.Load() > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("onChange was not called within timeout")
}

func TestWatcher_DebouncesBurstWrites(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "test.md")
	if err := os.WriteFile(file, []byte("# Original"), 0o644); err != nil {
		t.Fatalf("writing initial file: %v", err)
	}

	var called atomic.Int32
	w, err := watcher.New(file, 100*time.Millisecond, func() {
		called.Add(1)
	})
	if err != nil {
		t.Fatalf("creating watcher: %v", err)
	}
	go w.Start()
	defer func() {
		if closeErr := w.Close(); closeErr != nil {
			t.Errorf("closing watcher: %v", closeErr)
		}
	}()

	time.Sleep(50 * time.Millisecond)

	// Write 5 times in rapid succession.
	for i := range 5 {
		data := []byte("# Update " + string(rune('A'+i)))
		if err := os.WriteFile(file, data, 0o644); err != nil {
			t.Fatalf("writing file: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for debounce to settle.
	time.Sleep(300 * time.Millisecond)

	count := called.Load()
	if count > 2 {
		t.Errorf("expected at most 2 callback invocations after debounce, got %d", count)
	}
	if count == 0 {
		t.Error("expected at least 1 callback invocation")
	}
}

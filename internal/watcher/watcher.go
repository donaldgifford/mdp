// Package watcher provides debounced file change notifications using fsnotify.
package watcher

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors a file for changes and calls a callback after a debounce
// period.
type Watcher struct {
	fsw      *fsnotify.Watcher
	file     string
	debounce time.Duration
	onChange func()
	done     chan struct{}
}

// New creates a Watcher that monitors the given file.
func New(file string, debounce time.Duration, onChange func()) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating watcher: %w", err)
	}

	if err := fsw.Add(file); err != nil {
		if closeErr := fsw.Close(); closeErr != nil {
			slog.Error("closing watcher after add failure", "error", closeErr)
		}
		return nil, fmt.Errorf("watching %s: %w", file, err)
	}

	return &Watcher{
		fsw:      fsw,
		file:     file,
		debounce: debounce,
		onChange: onChange,
		done:     make(chan struct{}),
	}, nil
}

// Start begins watching for file changes in a blocking loop.
// Call Close to stop.
func (w *Watcher) Start() {
	defer close(w.done)

	var timer *time.Timer

	for {
		select {
		case event, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				slog.Debug("file changed", "file", event.Name, "op", event.Op)
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(w.debounce, w.onChange)
			}
		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			slog.Error("watcher error", "error", err)
		}
	}
}

// Close stops the watcher.
func (w *Watcher) Close() error {
	err := w.fsw.Close()
	<-w.done
	return err
}

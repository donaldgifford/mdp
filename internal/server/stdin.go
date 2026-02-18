package server

import (
	"bufio"
	"encoding/json"
	"io"
	"log/slog"
)

// stdinMessage represents a JSON message from the Neovim plugin via stdin.
type stdinMessage struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"` // Markdown content for "content" type.
	File string `json:"file,omitempty"` // File path for "content" type.
	Line int    `json:"line,omitempty"` // Cursor line for "cursor" type.
}

// ReadStdin reads newline-delimited JSON messages from r and dispatches them.
// It blocks until r is closed or an unrecoverable error occurs.
func (s *Server) ReadStdin(r io.Reader) {
	scanner := bufio.NewScanner(r)
	// Allow large buffer for big markdown files.
	const maxTokenSize = 10 * 1024 * 1024 // 10MB
	scanner.Buffer(make([]byte, 0, 64*1024), maxTokenSize)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg stdinMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			slog.Debug("invalid stdin message", "error", err)
			continue
		}

		switch msg.Type {
		case "content":
			if err := s.Broadcast([]byte(msg.Data)); err != nil {
				slog.Error("broadcast from stdin failed", "error", err)
			}
		case "cursor":
			if err := s.SendCursor(msg.Line); err != nil {
				slog.Error("cursor from stdin failed", "error", err)
			}
		default:
			slog.Debug("unknown stdin message type", "type", msg.Type)
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Debug("stdin reader error", "error", err)
	}
}

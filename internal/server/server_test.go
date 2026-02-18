package server_test

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/donaldgifford/mdp/internal/server"
)

func TestServer_ServesRenderedMarkdown(t *testing.T) {
	t.Parallel()

	// Create a temporary markdown file.
	dir := t.TempDir()
	mdFile := filepath.Join(dir, "test.md")
	content := []byte("# Test Page\n\nHello from **mdp**.\n")
	if err := os.WriteFile(mdFile, content, 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	srv, err := server.New(server.Config{
		File:        mdFile,
		Port:        0,
		OpenBrowser: false,
	})
	if err != nil {
		t.Fatalf("creating server: %v", err)
	}

	addr := srv.Addr()

	go func() {
		if serveErr := srv.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			t.Logf("server error: %v", serveErr)
		}
	}()

	// Wait for the server to be ready.
	url := "http://" + addr
	waitForServer(t, url, 2*time.Second)

	resp, err := http.Get(url) //nolint:noctx // Test code.
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Errorf("expected text/html content type, got %q", ct)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}

	got := string(body)
	for _, want := range []string{"<h1", "Test Page", "<strong>", "mdp"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in response body", want)
		}
	}
}

// waitForServer polls the URL until it responds or the timeout expires.
func waitForServer(t *testing.T, url string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url) //nolint:noctx // Test helper.
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server did not start within %v", timeout)
}

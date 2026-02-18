package server_test

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donaldgifford/mdp/internal/server"
)

func TestServer_ServesVendorScripts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte("# Test"), 0o644); err != nil {
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
	defer srv.Close()

	addr := srv.Addr()
	go func() {
		_ = srv.ListenAndServe()
	}()

	waitForServer(t, "http://"+addr)

	// Verify the index page references vendor scripts.
	resp, err := http.Get("http://" + addr) //nolint:noctx // Test code.
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}

	page := string(body)
	scripts := []string{
		"/vendor/mermaid.min.js",
		"/vendor/katex/katex.min.js",
		"/vendor/katex/auto-render.min.js",
		"/vendor/hljs/highlight.min.js",
		"/vendor/katex/katex.min.css",
	}

	for _, script := range scripts {
		if !strings.Contains(page, script) {
			t.Errorf("expected reference to %q in page", script)
		}
	}

	// Verify vendor assets are actually served.
	vendorPaths := []string{
		"/vendor/mermaid.min.js",
		"/vendor/katex/katex.min.js",
		"/vendor/katex/katex.min.css",
		"/vendor/hljs/highlight.min.js",
		"/vendor/hljs/github.min.css",
	}

	for _, path := range vendorPaths {
		vResp, vErr := http.Get("http://" + addr + path) //nolint:noctx // Test code.
		if vErr != nil {
			t.Errorf("GET %s: %v", path, vErr)
			continue
		}
		vResp.Body.Close()
		if vResp.StatusCode != http.StatusOK {
			t.Errorf("GET %s: expected 200, got %d", path, vResp.StatusCode)
		}
	}
}

func TestServer_RendersMermaidBlocks(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mdFile := filepath.Join(dir, "test.md")
	content := []byte("```mermaid\ngraph LR\n    A-->B\n```\n")
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
	defer srv.Close()

	addr := srv.Addr()
	go func() {
		_ = srv.ListenAndServe()
	}()

	waitForServer(t, "http://"+addr)

	resp, err := http.Get("http://" + addr) //nolint:noctx // Test code.
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}

	got := string(body)
	// goldmark-mermaid in client mode emits <pre class="mermaid">.
	if !strings.Contains(got, "mermaid") {
		t.Error("expected mermaid class in output")
	}
}

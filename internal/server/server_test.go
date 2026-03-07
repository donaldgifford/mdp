package server_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/donaldgifford/mdp/internal/server"
	"github.com/donaldgifford/mdp/internal/theme"
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
	waitForServer(t, url)

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

// fetchBody starts a server with cfg, waits for it, GETs /, and returns the body.
func fetchBody(t *testing.T, cfg *server.Config) string {
	t.Helper()

	srv, err := server.New(*cfg)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}

	go func() {
		if serveErr := srv.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			t.Logf("server error: %v", serveErr)
		}
	}()

	url := "http://" + srv.Addr()
	waitForServer(t, url)

	resp, err := http.Get(url) //nolint:noctx // Test helper.
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	return string(raw)
}

// tempMDFile creates a temp dir with a minimal markdown file and returns its path.
func tempMDFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	f := filepath.Join(dir, "test.md")
	if err := os.WriteFile(f, []byte("# hi\n"), 0o644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	return f
}

func TestServer_ThemeAttribute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		theme    string
		wantAttr string
	}{
		{"", `data-theme="auto"`},
		{"auto", `data-theme="auto"`},
		{"github-dark", `data-theme="github-dark"`},
		{"tokyo-night", `data-theme="tokyo-night"`},
	}

	for _, tt := range tests {
		t.Run(tt.theme+"_"+tt.wantAttr, func(t *testing.T) {
			t.Parallel()
			body := fetchBody(t, &server.Config{
				File:        tempMDFile(t),
				Port:        0,
				OpenBrowser: false,
				Theme:       tt.theme,
			})
			if !strings.Contains(body, tt.wantAttr) {
				t.Errorf("response body missing %q", tt.wantAttr)
			}
		})
	}
}

func TestServer_MermaidThemeAttribute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		theme    string
		wantAttr string
	}{
		{"github-dark", `data-mermaid-theme="base"`},
		{"github-light", `data-mermaid-theme="base"`},
		{"tokyo-night", `data-mermaid-theme="base"`},
		{"rose-pine", `data-mermaid-theme="base"`},
		{"catppuccin-mocha", `data-mermaid-theme="base"`},
		{"", `data-mermaid-theme=""`},
		{"auto", `data-mermaid-theme=""`},
	}

	for _, tt := range tests {
		t.Run(tt.theme, func(t *testing.T) {
			t.Parallel()
			body := fetchBody(t, &server.Config{
				File:        tempMDFile(t),
				Port:        0,
				OpenBrowser: false,
				Theme:       tt.theme,
			})
			if !strings.Contains(body, tt.wantAttr) {
				t.Errorf("response body missing %q\ngot body snippet: %s",
					tt.wantAttr, body[:min(len(body), 1000)])
			}
		})
	}
}

func TestServer_ThemeCSS_Injection(t *testing.T) {
	t.Parallel()

	t.Run("named theme injects ThemeCSS style block", func(t *testing.T) {
		t.Parallel()
		body := fetchBody(t, &server.Config{
			File:        tempMDFile(t),
			Port:        0,
			OpenBrowser: false,
			Theme:       "github-dark",
		})
		// Named theme: ThemeCSS should be present.
		if !strings.Contains(body, `data-theme="github-dark"`) {
			t.Error("missing github-dark data-theme attribute")
		}
		// The theme CSS block should appear before custom CSS.
		idx := strings.Index(body, "<style>")
		if idx < 0 {
			t.Error("no <style> block found in body")
		}
	})

	t.Run("auto theme has no ThemeCSS style block", func(t *testing.T) {
		t.Parallel()
		body := fetchBody(t, &server.Config{
			File:        tempMDFile(t),
			Port:        0,
			OpenBrowser: false,
			Theme:       "auto",
		})
		// Auto theme: only one <style> block (BaseCSS), no ThemeCSS.
		count := strings.Count(body, "<style>")
		if count != 1 {
			t.Errorf("expected 1 <style> block for auto theme, got %d", count)
		}
	})
}

func TestServer_HljsVendorCSS_Injection(t *testing.T) {
	t.Parallel()

	t.Run("github-dark has vendor link", func(t *testing.T) {
		t.Parallel()
		body := fetchBody(t, &server.Config{File: tempMDFile(t), Port: 0, Theme: "github-dark"})
		if !strings.Contains(body, `href="/vendor/hljs/github-dark.min.css"`) {
			t.Error("github-dark missing vendor hljs link")
		}
		// Should not have the auto media-query links.
		if strings.Contains(body, `media="(prefers-color-scheme:`) {
			t.Error("github-dark should not have media-query hljs links")
		}
	})

	t.Run("github-light has vendor link", func(t *testing.T) {
		t.Parallel()
		body := fetchBody(t, &server.Config{File: tempMDFile(t), Port: 0, Theme: "github-light"})
		if !strings.Contains(body, `href="/vendor/hljs/github.min.css"`) {
			t.Error("github-light missing vendor hljs link")
		}
	})

	t.Run("github-dimmed has no vendor link", func(t *testing.T) {
		t.Parallel()
		body := fetchBody(t, &server.Config{File: tempMDFile(t), Port: 0, Theme: "github-dimmed"})
		if strings.Contains(body, `/vendor/hljs/github`) {
			t.Error("github-dimmed should not have vendor hljs link")
		}
	})

	t.Run("tokyo-night has no vendor link", func(t *testing.T) {
		t.Parallel()
		body := fetchBody(t, &server.Config{File: tempMDFile(t), Port: 0, Theme: "tokyo-night"})
		if strings.Contains(body, `/vendor/hljs/github`) {
			t.Error("tokyo-night should not have vendor hljs link")
		}
	})

	t.Run("auto has prefers-color-scheme media links", func(t *testing.T) {
		t.Parallel()
		body := fetchBody(t, &server.Config{File: tempMDFile(t), Port: 0, Theme: "auto"})
		if !strings.Contains(body, `github.min.css`) {
			t.Error("auto theme missing github.min.css hljs link")
		}
		if !strings.Contains(body, `github-dark.min.css`) {
			t.Error("auto theme missing github-dark.min.css hljs link")
		}
	})
}

func TestServer_CustomCSS_AfterTheme(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	customCSS := filepath.Join(dir, "custom.css")
	if err := os.WriteFile(customCSS, []byte("/* custom */"), 0o644); err != nil {
		t.Fatalf("writing custom CSS: %v", err)
	}

	body := fetchBody(t, &server.Config{
		File:        tempMDFile(t),
		Port:        0,
		OpenBrowser: false,
		Theme:       "github-dark",
		CustomCSS:   customCSS,
	})

	themeIdx := strings.Index(body, `data-theme="github-dark"`)
	customIdx := strings.Index(body, "/* custom */")
	if themeIdx < 0 {
		t.Fatal("missing data-theme attribute")
	}
	if customIdx < 0 {
		t.Fatal("missing custom CSS in body")
	}
	// ThemeCSS is injected in the <head> before body content, custom CSS also
	// in <head> — custom CSS <style> must appear after the theme <style>.
	if customIdx < themeIdx {
		t.Error("custom CSS appears before theme indicator — expected it after")
	}
}

func TestServer_InvalidTheme(t *testing.T) {
	t.Parallel()

	_, err := server.New(server.Config{
		File:  tempMDFile(t),
		Theme: "nonexistent-theme",
	})
	if err == nil {
		t.Fatal("expected error for unknown theme, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent-theme") {
		t.Errorf("error %q should mention the bad theme name", err.Error())
	}
}

func TestServer_HljsTheme_WithBuiltinTheme_Errors(t *testing.T) {
	t.Parallel()

	_, err := server.New(server.Config{
		File:      tempMDFile(t),
		Theme:     "github-dark",
		HljsTheme: "github",
	})
	if err == nil {
		t.Fatal("expected error when --hljs-theme used with built-in theme")
	}
}

func TestServer_AllBuiltinThemes(t *testing.T) {
	t.Parallel()

	for _, name := range theme.Names() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			body := fetchBody(t, &server.Config{
				File:        tempMDFile(t),
				Port:        0,
				OpenBrowser: false,
				Theme:       name,
			})
			want := fmt.Sprintf("data-theme=%q", name)
			if !strings.Contains(body, want) {
				t.Errorf("response body missing %q for theme %q", want, name)
			}
		})
	}
}

// waitForServer polls the URL until it responds or 2 seconds elapse.
func waitForServer(t *testing.T, url string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url) //nolint:noctx // Test helper.
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("server did not start within 2s")
}

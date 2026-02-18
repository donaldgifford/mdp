.PHONY: build lint fmt test test-coverage clean update-vendor

BINARY := mdp
MAIN := ./cmd/mdp
VENDOR := assets/vendor

## Build the binary.
build:
	go build -o $(BINARY) $(MAIN)

## Run golangci-lint.
lint:
	golangci-lint run

## Format code via golangci-lint formatters.
fmt:
	golangci-lint fmt

## Run all tests.
test:
	go test ./...

## Run tests with coverage output.
test-coverage:
	go test -race -coverprofile=coverage.out ./...

## Remove build artifacts.
clean:
	rm -f $(BINARY) coverage.out

## Update vendored JS libraries from CDN.
update-vendor:
	curl -sL -o $(VENDOR)/mermaid.min.js "https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js"
	curl -sL -o $(VENDOR)/katex/katex.min.js "https://cdn.jsdelivr.net/npm/katex@0.16/dist/katex.min.js"
	curl -sL -o $(VENDOR)/katex/katex.min.css "https://cdn.jsdelivr.net/npm/katex@0.16/dist/katex.min.css"
	curl -sL -o $(VENDOR)/katex/auto-render.min.js "https://cdn.jsdelivr.net/npm/katex@0.16/dist/contrib/auto-render.min.js"
	curl -sL -o $(VENDOR)/hljs/highlight.min.js "https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@11/build/highlight.min.js"
	curl -sL -o $(VENDOR)/hljs/github.min.css "https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@11/build/styles/github.min.css"
	curl -sL -o $(VENDOR)/hljs/github-dark.min.css "https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@11/build/styles/github-dark.min.css"
	@echo "Vendor libraries updated. Don't forget to update KaTeX fonts if the version changed."

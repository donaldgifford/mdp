## Project Variables

PROJECT_NAME  := mdp
PROJECT_OWNER := donaldgifford
PROJECT_URL   := https://github.com/$(PROJECT_OWNER)/$(PROJECT_NAME)

## Go Variables

GO         ?= go
GO_PACKAGE := github.com/$(PROJECT_OWNER)/$(PROJECT_NAME)

## Build Directories

VENDOR := assets/vendor

## Version Information

COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
DATE        ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

## Build Variables

LDFLAGS_PKG  := $(GO_PACKAGE)/internal/cli
LDFLAGS      := -X $(LDFLAGS_PKG).version=$(VERSION) -X $(LDFLAGS_PKG).commit=$(COMMIT_HASH) -X $(LDFLAGS_PKG).date=$(DATE)
COVERAGE_OUT := coverage.out

###############
##@ Build

.PHONY: build clean

build: ## Build the mdp binary
	@ $(MAKE) --no-print-directory log-$@
	@$(GO) build -ldflags "$(LDFLAGS)" -o $(PROJECT_NAME) ./cmd/$(PROJECT_NAME)
	@echo "✓ $(PROJECT_NAME) built"

clean: ## Remove build artifacts
	@ $(MAKE) --no-print-directory log-$@
	@rm -f $(PROJECT_NAME) $(COVERAGE_OUT)
	@echo "✓ Build artifacts cleaned"

###############
##@ Testing

.PHONY: test test-coverage

test: ## Run all tests
	@ $(MAKE) --no-print-directory log-$@
	@$(GO) test ./...

test-coverage: ## Run tests with race detector and coverage output
	@ $(MAKE) --no-print-directory log-$@
	@$(GO) test -race -coverprofile=$(COVERAGE_OUT) ./...

###############
##@ Code Quality

.PHONY: lint lint-fix fmt

lint: ## Run golangci-lint
	@ $(MAKE) --no-print-directory log-$@
	@golangci-lint run

lint-fix: ## Run golangci-lint with auto-fix
	@ $(MAKE) --no-print-directory log-$@
	@golangci-lint run --fix

fmt: ## Format code via golangci-lint formatters
	@ $(MAKE) --no-print-directory log-$@
	@golangci-lint fmt

###############
##@ License

.PHONY: license-check

license-check: ## Check dependency licenses against allowed list
	@ $(MAKE) --no-print-directory log-$@
	@go-licenses check ./... \
		--allowed_licenses=Apache-2.0,MIT,BSD-2-Clause,BSD-3-Clause,ISC,MPL-2.0 \
		--ignore=github.com/litao91/goldmark-mathjax

###############
##@ Assets

.PHONY: update-vendor

update-vendor: ## Update vendored JS/CSS libraries from CDN
	@ $(MAKE) --no-print-directory log-$@
	@curl -sL -o $(VENDOR)/mermaid.min.js "https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js"
	@curl -sL -o $(VENDOR)/katex/katex.min.js "https://cdn.jsdelivr.net/npm/katex@0.16/dist/katex.min.js"
	@curl -sL -o $(VENDOR)/katex/katex.min.css "https://cdn.jsdelivr.net/npm/katex@0.16/dist/katex.min.css"
	@curl -sL -o $(VENDOR)/katex/auto-render.min.js "https://cdn.jsdelivr.net/npm/katex@0.16/dist/contrib/auto-render.min.js"
	@curl -sL -o $(VENDOR)/hljs/highlight.min.js "https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@11/build/highlight.min.js"
	@curl -sL -o $(VENDOR)/hljs/github.min.css "https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@11/build/styles/github.min.css"
	@curl -sL -o $(VENDOR)/hljs/github-dark.min.css "https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@11/build/styles/github-dark.min.css"
	@echo "✓ Vendor libraries updated"
	@echo "  Note: update KaTeX fonts manually if the version changed"

###############
##@ CI/CD

.PHONY: ci check release-check release-local

ci: lint test build license-check ## Run full CI pipeline
	@ $(MAKE) --no-print-directory log-$@
	@echo "✓ CI pipeline complete"

check: lint test ## Quick pre-commit check (lint + test)
	@ $(MAKE) --no-print-directory log-$@
	@echo "✓ Pre-commit checks passed"

release-check: ## Validate goreleaser config
	@ $(MAKE) --no-print-directory log-$@
	@goreleaser check

release-local: ## Test release build locally without publishing
	@ $(MAKE) --no-print-directory log-$@
	@goreleaser release --snapshot --clean --skip=publish --skip=sign

########################################################################
## Self-Documenting Makefile Help                                     ##
## https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html ##
########################################################################

##@ Help

.PHONY: help
.DEFAULT_GOAL := help

help: ## Display this help
	@awk -v "col=\033[36m" -v "nocol=\033[0m" ' \
		BEGIN { FS = ":.*##" ; printf "Usage:\n  make %s<target>%s\n\n", col, nocol } \
		/^[a-zA-Z_0-9-]+:.*?##/ { printf "  %s%-25s%s %s\n", col, $$1, nocol, $$2 } \
		/^##@/ { printf "\n%s%s%s\n", nocol, substr($$0, 5), nocol } \
	' $(MAKEFILE_LIST)

## Log Pattern
## Automatically logs what a target does by extracting its ## comment
log-%:
	@grep -h -E '^$*:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN { FS = ":.*?## " }; { printf "\033[36m==> %s\033[0m\n", $$2 }'

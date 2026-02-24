#!/usr/bin/env bash
# Install script for mdp binary.
# Tries to download a pre-built binary from GitHub releases.
# Falls back to building from source if download fails.
#
# Usage: ./scripts/install.sh [--source]
#   --source  Skip download, build from source directly

set -Eeuo pipefail

REPO="donaldgifford/mdp"
BINARY_NAME="mdp"

# Plugin root (parent of scripts/).
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLUGIN_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
BIN_DIR="${PLUGIN_DIR}/bin"

log() { printf '[mdp] %s\n' "$*"; }
err() { printf '[mdp] ERROR: %s\n' "$*" >&2; }

detect_platform() {
  local os arch

  case "$(uname -s)" in
    Darwin) os="darwin" ;;
    Linux)  os="linux" ;;
    MINGW*|MSYS*|CYGWIN*) os="windows" ;;
    *) err "Unsupported OS: $(uname -s)"; return 1 ;;
  esac

  case "$(uname -m)" in
    x86_64|amd64)  arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *) err "Unsupported arch: $(uname -m)"; return 1 ;;
  esac

  printf '%s_%s' "${os}" "${arch}"
}

get_latest_release() {
  local api_url tag
  api_url="https://api.github.com/repos/${REPO}/releases/latest"

  if command -v curl >/dev/null 2>&1; then
    tag=$(curl -fsSL "${api_url}" 2>/dev/null \
      | grep '"tag_name"' | head -1 \
      | sed 's/.*"tag_name": *"//;s/".*//')
  elif command -v wget >/dev/null 2>&1; then
    tag=$(wget -qO- "${api_url}" 2>/dev/null \
      | grep '"tag_name"' | head -1 \
      | sed 's/.*"tag_name": *"//;s/".*//')
  fi

  if [[ -z "${tag:-}" ]]; then
    return 1
  fi

  printf '%s' "${tag}"
}

download_binary() {
  local platform tag base_url archive_url tmp_dir

  platform="$(detect_platform)" || return 1
  log "Detected platform: ${platform}"

  tag="$(get_latest_release)" \
    || { err "Could not determine latest release"; return 1; }
  log "Latest release: ${tag}"

  base_url="https://github.com/${REPO}/releases/download"
  archive_url="${base_url}/${tag}/${BINARY_NAME}_${platform}.tar.gz"
  log "Downloading ${archive_url}"

  tmp_dir="$(mktemp -d)"
  cleanup() { rm -rf "${tmp_dir}"; }

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "${archive_url}" -o "${tmp_dir}/archive.tar.gz" \
      || { cleanup; return 1; }
  elif command -v wget >/dev/null 2>&1; then
    wget -q "${archive_url}" -O "${tmp_dir}/archive.tar.gz" \
      || { cleanup; return 1; }
  else
    err "Neither curl nor wget found"
    cleanup
    return 1
  fi

  tar -xzf "${tmp_dir}/archive.tar.gz" -C "${tmp_dir}" \
    || { cleanup; return 1; }

  mkdir -p "${BIN_DIR}"

  if [[ -f "${tmp_dir}/${BINARY_NAME}" ]]; then
    mv "${tmp_dir}/${BINARY_NAME}" "${BIN_DIR}/${BINARY_NAME}"
  else
    local found
    found="$(find "${tmp_dir}" -name "${BINARY_NAME}" \
      -type f | head -1)"
    if [[ -z "${found}" ]]; then
      err "Binary not found in archive"
      cleanup
      return 1
    fi
    mv "${found}" "${BIN_DIR}/${BINARY_NAME}"
  fi

  cleanup
  chmod +x "${BIN_DIR}/${BINARY_NAME}"
  log "Installed ${BIN_DIR}/${BINARY_NAME} (${tag})"
}

build_from_source() {
  log "Building from source..."

  if ! command -v go >/dev/null 2>&1; then
    err "Go toolchain not found."
    err "  Install Go: https://go.dev/dl/"
    err "  Or: go install github.com/${REPO}/cmd/mdp@latest"
    return 1
  fi

  local ldflags_pkg="github.com/donaldgifford/${BINARY_NAME}/internal/cli"
  local version commit date ldflags
  version="$(git -C "${PLUGIN_DIR}" describe --tags --always --dirty 2>/dev/null || echo dev)"
  commit="$(git -C "${PLUGIN_DIR}" rev-parse --short HEAD 2>/dev/null || echo none)"
  date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  ldflags="-X ${ldflags_pkg}.version=${version} -X ${ldflags_pkg}.commit=${commit} -X ${ldflags_pkg}.date=${date}"

  mkdir -p "${BIN_DIR}"
  (cd "${PLUGIN_DIR}" \
    && go build -ldflags "${ldflags}" -o "${BIN_DIR}/${BINARY_NAME}" ./cmd/mdp)
  log "Built ${BIN_DIR}/${BINARY_NAME} from source (${version})"
}

main() {
  local force_source=false

  if [[ "${1:-}" == "--source" ]]; then
    force_source=true
  fi

  if [[ "${force_source}" == "true" ]]; then
    build_from_source
    return
  fi

  if download_binary; then
    return
  fi

  log "Download failed, falling back to source build..."
  build_from_source
}

main "$@"

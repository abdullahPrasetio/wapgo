#!/usr/bin/env bash
# install.sh — Download and install the wapgo CLI binary
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/abdullahPrasetio/wapgo/main/install.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/abdullahPrasetio/wapgo/main/install.sh | bash -s -- --version v1.0.0
#   curl -fsSL https://raw.githubusercontent.com/abdullahPrasetio/wapgo/main/install.sh | bash -s -- --dir /usr/local/bin

set -euo pipefail

REPO="abdullahPrasetio/wapgo"
BINARY="wapgo"
DEFAULT_DIR="${HOME}/.local/bin"

# ── Parse flags ──────────────────────────────────────────────────────────────

VERSION=""
INSTALL_DIR=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version|-v) VERSION="$2"; shift 2 ;;
    --dir|-d)     INSTALL_DIR="$2"; shift 2 ;;
    --help|-h)
      echo "Usage: install.sh [--version v1.0.0] [--dir /usr/local/bin]"
      exit 0 ;;
    *) echo "Unknown flag: $1"; exit 1 ;;
  esac
done

INSTALL_DIR="${INSTALL_DIR:-$DEFAULT_DIR}"

# ── Detect OS and architecture ───────────────────────────────────────────────

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1 ;;
esac

case "$OS" in
  linux|darwin) ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1 ;;
esac

# ── Resolve version ──────────────────────────────────────────────────────────

if [[ -z "$VERSION" ]]; then
  echo "Fetching latest release..."
  if command -v curl &>/dev/null; then
    VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
      | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\(.*\)".*/\1/')"
  elif command -v wget &>/dev/null; then
    VERSION="$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" \
      | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\(.*\)".*/\1/')"
  else
    echo "curl or wget is required"
    exit 1
  fi
fi

if [[ -z "$VERSION" ]]; then
  echo "Could not determine version. Pass --version v1.0.0 explicitly."
  exit 1
fi

echo "Installing ${BINARY} ${VERSION} (${OS}/${ARCH}) → ${INSTALL_DIR}"

# ── Download ─────────────────────────────────────────────────────────────────

TARBALL="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${TARBALL}"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

if command -v curl &>/dev/null; then
  curl -fsSL "$URL" -o "${TMP}/${TARBALL}"
else
  wget -qO "${TMP}/${TARBALL}" "$URL"
fi

# ── Extract and install ───────────────────────────────────────────────────────

tar -xzf "${TMP}/${TARBALL}" -C "$TMP"

mkdir -p "$INSTALL_DIR"
install -m 755 "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"

# ── Verify ───────────────────────────────────────────────────────────────────

if ! command -v "${BINARY}" &>/dev/null; then
  echo ""
  echo "Installed to ${INSTALL_DIR}/${BINARY}"
  echo "Add to PATH: export PATH=\"\$PATH:${INSTALL_DIR}\""
else
  echo ""
  echo "${BINARY} $(${BINARY} version) installed successfully"
fi

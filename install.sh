#!/bin/sh
set -euo pipefail

REPO="lutrarutra/lazypush"
BIN="lazypush"

# --- Detect OS and arch ---
detect_os() {
	case "$(uname -s)" in
	Linux)  echo "linux" ;;
	Darwin) echo "darwin" ;;
	MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
	*)      echo "" ;;
	esac
}

detect_arch() {
	case "$(uname -m)" in
	x86_64|amd64) echo "amd64" ;;
	aarch64|arm64) echo "arm64" ;;
	*)             echo "" ;;
	esac
}

OS="${LAZYPUSH_OS:-$(detect_os)}"
ARCH="${LAZYPUSH_ARCH:-$(detect_arch)}"

if [ -z "$OS" ]; then
	echo "error: unsupported OS ($(uname -s)). Set LAZYPUSH_OS to override." >&2
	exit 1
fi
if [ -z "$ARCH" ]; then
	echo "error: unsupported architecture ($(uname -m)). Set LAZYPUSH_ARCH to override." >&2
	exit 1
fi

# Windows users should use go install or scoop
if [ "$OS" = "windows" ]; then
	echo "error: Windows is not supported by this script. Use: go install github.com/$REPO/cmd/$BIN@latest" >&2
	exit 1
fi

# --- Get latest version ---
echo "-> Fetching latest release..." >&2
VERSION=$(curl -sSfL "https://api.github.com/repos/$REPO/releases/latest" \
	| grep '"tag_name":' \
	| sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')

if [ -z "$VERSION" ]; then
	echo "error: could not determine latest version from GitHub API" >&2
	exit 1
fi
echo "   Latest: $VERSION" >&2

# GoReleaser strips the v prefix in archive filenames
VERSION_NO_V="${VERSION#v}"

# --- Download archive ---
ARCHIVE="${BIN}_${VERSION_NO_V}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$VERSION/$ARCHIVE"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

echo "-> Downloading $ARCHIVE ..." >&2
curl -sSfL "$URL" -o "$TMPDIR/$ARCHIVE"

# --- Determine install directory ---
if [ -w /usr/local/bin ]; then
	INSTALL_DIR="/usr/local/bin"
elif [ -w "$HOME/.local/bin" ]; then
	INSTALL_DIR="$HOME/.local/bin"
else
	INSTALL_DIR="$HOME/.local/bin"
	mkdir -p "$INSTALL_DIR"
fi

# --- Extract binary ---
echo "-> Extracting..." >&2
tar -xzf "$TMPDIR/$ARCHIVE" -C "$TMPDIR"
BIN_PATH="$TMPDIR/$BIN"
if [ ! -f "$BIN_PATH" ]; then
	# Some GoReleaser archives include a subdirectory
	BIN_PATH=$(find "$TMPDIR" -type f -name "$BIN" 2>/dev/null | head -1)
fi

if [ ! -f "$BIN_PATH" ]; then
	echo "error: binary not found in archive" >&2
	exit 1
fi

# --- Install ---
echo "-> Installing to $INSTALL_DIR/$BIN ..." >&2
install -m 755 "$BIN_PATH" "$INSTALL_DIR/$BIN"

# --- Verify ---
echo "-> Verifying..." >&2
if command -v "$BIN" >/dev/null 2>&1 || [ -x "$INSTALL_DIR/$BIN" ]; then
	echo "   Installed: $INSTALL_DIR/$BIN"
	echo ""
	echo "   Make sure $INSTALL_DIR is in your PATH."
	echo "   Run: export PATH=\"\$PATH:$INSTALL_DIR\""
	echo ""
	echo "lazypush $VERSION installed successfully!"
else
	echo "error: installation failed -- $INSTALL_DIR/$BIN not found" >&2
	exit 1
fi

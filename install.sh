#!/bin/sh
set -euo pipefail

REPO="lutrarutra/lazypush"
BIN="lazypush"

# --- Help ---
if [ "${1:-}" = "--help" ] || [ "${1:-}" = "-h" ]; then
	cat <<EOF
Usage: install.sh [INSTALL_DIR]

Install lazypush — an interactive commit, tag, release, and PR tool.

If INSTALL_DIR is given, install there (uses sudo if needed).
Otherwise, installs to /usr/local/bin (writable) or ~/.local/bin (fallback).

Examples:
  curl -sSfL https://raw.githubusercontent.com/$REPO/main/install.sh | sh
  curl -sSfL https://raw.githubusercontent.com/$REPO/main/install.sh | sh -s /opt/bin
  curl -sSfL https://raw.githubusercontent.com/$REPO/main/install.sh | INSTALL_DIR=~/bin sh

Environment:
  LAZYPUSH_OS       Override OS detection (linux, darwin)
  LAZYPUSH_ARCH     Override arch detection (amd64, arm64)
  LAZYPUSH_INSTALL_DIR  Install directory (same as first argument)
EOF
	exit 0
fi

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
INSTALL_DIR="${LAZYPUSH_INSTALL_DIR:-${1:-}}"
if [ -z "$INSTALL_DIR" ]; then
	if [ -w /usr/local/bin ]; then
		INSTALL_DIR="/usr/local/bin"
	else
		INSTALL_DIR="$HOME/.local/bin"
		mkdir -p "$INSTALL_DIR"
	fi
fi

# Resolve ~ to $HOME
case "$INSTALL_DIR" in
	"~"/*) INSTALL_DIR="$HOME/${INSTALL_DIR#~/}" ;;
	"~")   INSTALL_DIR="$HOME" ;;
esac

# --- Extract binary ---
echo "-> Extracting..." >&2
tar -xzf "$TMPDIR/$ARCHIVE" -C "$TMPDIR"
BIN_PATH="$TMPDIR/$BIN"
if [ ! -f "$BIN_PATH" ]; then
	BIN_PATH=$(find "$TMPDIR" -type f -name "$BIN" 2>/dev/null | head -1)
fi

if [ ! -f "$BIN_PATH" ]; then
	echo "error: binary not found in archive" >&2
	exit 1
fi

# --- Install ---
mkdir -p "$INSTALL_DIR"
if [ -w "$INSTALL_DIR" ]; then
	echo "-> Installing to $INSTALL_DIR/$BIN ..." >&2
	install -m 755 "$BIN_PATH" "$INSTALL_DIR/$BIN"
else
	echo "-> Installing to $INSTALL_DIR/$BIN (using sudo)..." >&2
	sudo install -m 755 "$BIN_PATH" "$INSTALL_DIR/$BIN"
fi

# --- Verify ---
INSTALLED="$INSTALL_DIR/$BIN"
if [ ! -x "$INSTALLED" ]; then
	echo "error: installation failed -- $INSTALLED not found" >&2
	exit 1
fi

echo ""
echo "   Installed: $INSTALLED"
echo ""
echo "✅ lazypush $VERSION installed successfully!"
echo ""

# Check if install dir is in PATH
case ":${PATH:-}:" in
	*":${INSTALL_DIR}:"*) ;;
	*)
		case "$(basename "$SHELL" 2>/dev/null)" in
			zsh)  PROFILE="$HOME/.zshrc" ;;
			bash) PROFILE="$HOME/.bashrc" ;;
			fish) PROFILE="$HOME/.config/fish/config.fish" ;;
			*)    PROFILE="$HOME/.profile" ;;
		esac
		echo "   ⚠  $INSTALL_DIR is not in your PATH."
		echo "   Add this to $PROFILE:"
		echo ""
		echo "       export PATH=\"\$PATH:$INSTALL_DIR\""
		echo ""
		echo "   Then reload: source $PROFILE"
		echo ""
		;;
esac

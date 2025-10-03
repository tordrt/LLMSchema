#!/bin/sh
set -e

# LLMSchema installer script

REPO="tordrt/llmschema"
INSTALL_DIR="/usr/local/bin"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  darwin) OS="Darwin" ;;
  linux) OS="Linux" ;;
  mingw* | msys* | cygwin*) OS="Windows" ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH="x86_64" ;;
  amd64) ARCH="x86_64" ;;
  arm64) ARCH="arm64" ;;
  aarch64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

# Get latest release version
echo "Fetching latest release..."
LATEST_VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
  echo "Failed to fetch latest release version"
  exit 1
fi

echo "Latest version: $LATEST_VERSION"

# Build download URL
if [ "$OS" = "Windows" ]; then
  ARCHIVE_EXT="zip"
else
  ARCHIVE_EXT="tar.gz"
fi

ARCHIVE_NAME="LLMSchema_${LATEST_VERSION#v}_${OS}_${ARCH}.${ARCHIVE_EXT}"
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_VERSION/$ARCHIVE_NAME"

echo "Downloading $ARCHIVE_NAME..."
TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

if ! curl -sL "$DOWNLOAD_URL" -o "$ARCHIVE_NAME"; then
  echo "Failed to download $DOWNLOAD_URL"
  exit 1
fi

# Extract archive
echo "Extracting..."
if [ "$OS" = "Windows" ]; then
  unzip -q "$ARCHIVE_NAME"
else
  tar -xzf "$ARCHIVE_NAME"
fi

# Install binary
echo "Installing to $INSTALL_DIR..."
if [ -w "$INSTALL_DIR" ]; then
  mv llmschema "$INSTALL_DIR/llmschema"
  chmod +x "$INSTALL_DIR/llmschema"
else
  echo "Requesting sudo for installation to $INSTALL_DIR..."
  sudo mv llmschema "$INSTALL_DIR/llmschema"
  sudo chmod +x "$INSTALL_DIR/llmschema"
fi

# Cleanup
cd -
rm -rf "$TMP_DIR"

echo "✓ llmschema installed successfully!"
echo "Run 'llmschema --help' to get started"
#!/bin/bash

set -e

OS=$(uname -s)
ARCH=$(uname -m)

case "$OS" in
Linux)
  case "$ARCH" in
  x86_64)
    BINARY_NAME="gai-linux-amd64"
    ;;
  aarch64)
    BINARY_NAME="gai-linux-arm64"
    ;;
  *)
    echo "Unsupported architecture: $ARCH for Linux"
    exit 1
    ;;
  esac
  ;;
Darwin)
  case "$ARCH" in
  x86_64)
    BINARY_NAME="gai-darwin-amd64"
    ;;
  arm64)
    BINARY_NAME="gai-darwin-arm64"
    ;;
  *)
    echo "Unsupported architecture: $ARCH for macOS"
    exit 1
    ;;
  esac
  ;;
*)
  echo "Unsupported OS: $OS"
  exit 1
  ;;
esac

DOWNLOAD_URL=$(curl -s https://api.github.com/repos/tnfssc/gai/releases/latest | jq -r ".assets[] | select(.name == \"$BINARY_NAME\") | .browser_download_url")

if [ -z "$DOWNLOAD_URL" ]; then
  echo "Error: Download URL not found for $BINARY_NAME"
  exit 1
fi

echo "Downloading gai from $DOWNLOAD_URL"
curl -Lo gai "$DOWNLOAD_URL"

chmod +x gai

INSTALL_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR"

if [ -d "$INSTALL_DIR" ]; then
  mv gai "$INSTALL_DIR"
  echo "gai installed to $INSTALL_DIR"
  echo "Please ensure '$INSTALL_DIR' is in your PATH environment variable."
  if grep -q "export PATH=\"\$HOME/.local/bin:\$PATH\"" ~/.bashrc; then
    echo "~/.bashrc already configured for \$HOME/.local/bin"
  elif grep -q "export PATH=\"\$HOME/.local/bin:\$PATH\"" ~/.zshrc; then
    echo "~/.zshrc already configured for \$HOME/.local/bin"
  else
    echo "Add 'export PATH=\"\$HOME/.local/bin:\$PATH\"' to your ~/.bashrc or ~/.zshrc and restart your terminal."
  fi

else
  echo "Error: Installation directory '$INSTALL_DIR' not found or not a directory."
  exit 1
fi

# TESTING
# docker run --rm -v "$(pwd):/app" ubuntu:latest bash -c "apt-get update && apt-get install -y curl jq && cd /app && chmod +x install.sh && ./install.sh && ls /root/bin/gai"

# Written by LLM, untested

#!/bin/bash
# This script downloads and installs the correct "gai" binary for macOS or Linux

set -e

# Check for a download tool (curl or wget)
if command -v curl >/dev/null; then
  downloader="curl -L -o"
elif command -v wget >/dev/null; then
  downloader="wget -O"
else
  echo "Error: Please install curl or wget to continue."
  exit 1
fi

# Determine the operating system
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
# Determine the system architecture
ARCH=$(uname -m)

case "$ARCH" in
x86_64)
  arch="amd64"
  ;;
arm64 | aarch64)
  arch="arm64"
  ;;
*)
  echo "Unsupported architecture: $ARCH"
  exit 1
  ;;
esac

# Select the correct binary based on the OS
if [[ "$OS" == "darwin" ]]; then
  binary="gai-darwin-${arch}"
elif [[ "$OS" == "linux" ]]; then
  binary="gai-linux-${arch}"
else
  echo "Unsupported operating system: $OS"
  exit 1
fi

# Construct the download URL using the latest release
url="https://github.com/tnfssc/gai/releases/latest/download/${binary}"
echo "Downloading ${binary} from ${url}..."

# Download the binary
$downloader "${binary}" "${url}"

# Make the binary executable
chmod +x "${binary}"

# Attempt to install the binary to /usr/local/bin if possible
install_path="/usr/local/bin/${binary}"
if [ -w "/usr/local/bin" ]; then
  mv "${binary}" "${install_path}"
  echo "Installed ${binary} to /usr/local/bin/"
else
  echo "Warning: You do not have write permissions for /usr/local/bin."
  echo "You may need to move '${binary}' to a directory in your PATH manually, e.g.:"
  echo "  sudo mv '${binary}' /usr/local/bin/"
fi

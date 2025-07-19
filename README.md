# gai

⚡ The fastest AI agent for CLI - powered by intelligent models

[![Release](https://github.com/tnfssc/gai/actions/workflows/release.yml/badge.svg)](https://github.com/tnfssc/gai/actions/workflows/release.yml)

## Installation

### Using bash (Linux and macOS only)

```bash
bash <(curl https://raw.githubusercontent.com/tnfssc/gai/refs/heads/develop/install.sh)
```

### Using powershell (Windows, macOS and Linux)

```bash
pwsh -c "irm https://raw.githubusercontent.com/tnfssc/gai/refs/heads/develop/install.ps1 | iex"
```

### Using `go install`

```bash
go install github.com/tnfssc/gai@latest
```

### Direct download

Download the binary directly from [Releases](https://github.com/tnfssc/gai/releases/latest)

## Setup

### Automatic

Follow `gai`'s instructions when you use it for the first time

### Manual

Choose your preferred provider:

- **Groq**: Get your key from [groqcloud](https://console.groq.com/keys)
- **OpenRouter**: Get your key from [openrouter.ai](https://openrouter.ai)
- **Kimi (Kimi-K2)**: Get your key from [openrouter.ai](https://openrouter.ai) (uses openrouter/moonshotai/kimi-k2 model)

Add it to your `~/.bashrc` file as follows:

```bash
# For Groq
export GROQ_API_KEY=your-api-key

# For OpenRouter
export OPENROUTER_API_KEY=your-api-key

# For Kimi (Kimi-K2)
export KIMI_API_KEY=your-api-key
```

## Usage

### Basic Usage

```bash
# Using default provider (Groq)
./gai list all files that contain the word "hello"

# Using specific provider
./gai --provider kimi analyze this log file for errors
./gai --provider openrouter create a backup of my documents

# The command is already copied to your clipboard, Ctrl+Shift+V away immediately
```

### Agentic Capabilities

gai is now an intelligent AI agent that can:

- **Execute commands**: Generate and run shell commands
- **Analyze files**: Process and analyze text, logs, and data
- **System administration**: Help with system tasks and configuration
- **File operations**: Manage files and directories efficiently
- **Network operations**: Handle network-related tasks
- **Package management**: Assist with software installation and updates
- **Text processing**: Analyze and transform text data

### Using Standard Input

```bash
cat /etc/hosts | gai block youtube.com
cat logfile.txt | gai --provider kimi find all error messages
```

### Using Docker

```bash
# Using Groq
docker run -e GROQ_API_KEY=$GROQ_API_KEY ghcr.io/tnfssc/gai:latest list all files that contain the word "hello"

# Using Kimi
docker run -e KIMI_API_KEY=$KIMI_API_KEY ghcr.io/tnfssc/gai:latest --provider kimi analyze this system

# Copy to clipboard fails if you use Docker
```

## Supported Providers

- **Groq**: Fast inference with various models
- **OpenRouter**: Access to multiple AI models
- **Kimi (Kimi-K2)**: Agentic model from Moonshot AI via OpenRouter

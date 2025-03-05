# gai

⚡ The fastest AI command generator for CLI

[![Release](https://github.com/tnfssc/gai/actions/workflows/release.yml/badge.svg)](https://github.com/tnfssc/gai/actions/workflows/release.yml)

## Installation

Using `go install`

```bash
go install github.com/tnfssc/gai@latest
```

Or download the binary directly from [Releases](https://github.com/tnfssc/gai/releases/latest)

## Setup

Get your key from [groqcloud](https://console.groq.com/keys)

Add it in your `~/.bashrc` file as follows

```bash
export GROQ_API_KEY=your-api-key
```

## Usage

```bash
./gai list all files that contain the word "hello"
# or
gai list all files that contain the word "hello" # if installed globally
# the command is already copied to your clipboard, Ctrl+Shift+V away immediately
```

Use standard input

```bash
cat /etc/hosts | gai block youtube.com
```

# webcatch

[![Go Version](https://img.shields.io/github/go-mod/go-version/alwedo/webcatch)](https://go.dev/)
[![Tests](https://github.com/alwedo/webcatch/actions/workflows/test.yml/badge.svg)](https://github.com/alwedo/webcatch/actions/workflows/test.yml)
[![Release](https://img.shields.io/github/v/release/alwedo/webcatch)](https://github.com/alwedo/webcatch/releases)

A simple HTTP request inspector that captures and displays incoming requests in real-time.

## Installation

### Homebrew (macOS/Linux)

```bash
brew install alwedo/tap/webcatch
```

### From Source

```bash
go install github.com/alwedo/webcatch@latest
```

### Download Binary

Download the latest release from the [releases page](https://github.com/alwedo/webcatch/releases).

## Usage

```bash
webcatch
```

This starts two servers:
- **Capture server** on `:8080` - send requests here to capture them
- **Viewer server** on `:8081` - view captured requests in your browser

### Flags

```bash
--capture-port  Port for capture server (default: 8080)
--viewer-port   Port for viewer server (default: 8081)
--version       Print version and exit
```

Example with custom ports:
```bash
webcatch --capture-port 9000 --viewer-port 9001
```

### Example

Send a test request:
```bash
curl -X POST http://localhost:8080/test \
  -H "Content-Type: application/json" \
  -d '{"hello":"world"}'
```

View it at http://localhost:8081

## Using with ngrok

To capture webhooks from external services, expose the capture server with ngrok:

```bash
ngrok http 8080
```

Then use the ngrok URL (e.g., `https://abc123.ngrok.io`) as your webhook endpoint. All requests will be captured and visible at http://localhost:8081.

## Features

- Captures method, path, headers, and body
- Live updates via Server-Sent Events
- Clear captured calls with one click
- Graceful shutdown with Ctrl+C

## Building

```bash
go build -o webcatch
```

## Testing

```bash
go test ./...
```

## Version

```bash
webcatch --version
```

## Development

Run from source:
```bash
go run .
```

Build locally:
```bash
go build -o webcatch
./webcatch
```

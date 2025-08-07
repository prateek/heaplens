# HeapLens

A Go heap dump analysis tool providing Web UI and CLI for memory profiling and debugging.

## Features

- **Embeddable Web UI**: Mount at `/debug/heaplens` in your application
- **CLI Tools**: Analyze dumps from the command line
- **Graph Analysis**: Paths-to-roots, dominator tree, retained size calculation
- **Multiple Formats**: Pluggable parser system for different dump formats
- **Performance**: Handle 5-10GB dumps with streaming parse

## Installation

```bash
go install github.com/prateek/heaplens/cmd/heaplens@latest
```

## Usage

### CLI

```bash
# Analyze top memory consumers
heaplens top heap.dump -n 20

# Find paths to roots for an object
heaplens paths heap.dump --id=0x12345

# Calculate dominators
heaplens dominators heap.dump -n 20

# Launch Web UI
heaplens web heap.dump --open
```

### Embedded Web UI

```go
import "github.com/prateek/heaplens/heaplenshttp"

// Mount the handler in your application
http.Handle("/debug/heaplens/", heaplenshttp.Handler())
```

## Development

```bash
# Run tests
go test ./...

# Build
go build -o heaplens cmd/heaplens/main.go

# Run with race detector
go test -race ./...
```

## Requirements

- Go 1.22 or later
- No CGO dependencies

## License

MIT
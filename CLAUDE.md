# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

HeapLens is a Go heap dump analysis tool that provides:
- Embeddable Web UI mountable at `/debug/heaplens` (pprof-style)
- CLI with web, TUI, and query subcommands
- Graph analysis algorithms: paths-to-roots, dominator tree, retained size calculation
- Pluggable parser system for different dump formats

## Architecture

### Core Packages

- **`graph/`**: Object model, indices, and algorithms (paths, dominators, retained size)
  - Contains graph data structures and core analysis algorithms
  - Implements Lengauer-Tarjan for dominators, BFS for paths-to-roots
  
- **`heapdump/`**: Parser interface and registry system
  - Defines `Parser` interface for pluggable dump format support
  - Includes JSON stub parser for testing
  - Will contain Go heap dump parser implementation
  
- **`heaplenshttp/`**: HTTP handlers and Web UI
  - Mountable handler at configurable base path
  - Server-side rendered templates (no JS build required)
  - Views: top types, paths, dominators, object search
  
- **`tui/`**: Terminal UI using Bubble Tea (optional dependency)
  
- **`cmd/heaplens/`**: CLI entry point with subcommands

## Development Commands

### Project Setup
```bash
# Initialize module (if not exists)
go mod init github.com/prateek/heaplens

# Download dependencies
go mod download

# Verify module
go mod verify
```

### Building
```bash
# Build CLI binary
go build -o heaplens cmd/heaplens/main.go

# Build with optimizations
go build -ldflags="-s -w" -o heaplens cmd/heaplens/main.go
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detector
go test -race ./...

# Run specific package tests
go test ./graph/...
go test ./heapdump/...

# Run tests with verbose output
go test -v ./...

# Run benchmarks
go test -bench=. -benchmem ./...

# Run a single test
go test -run TestPathsToRoots ./graph
```

### Development Workflow
```bash
# Format code
go fmt ./...

# Run go vet
go vet ./...

# Run staticcheck (if installed)
staticcheck ./...

# Check for module issues
go mod tidy

# Update dependencies
go get -u ./...
```

### Running the Application
```bash
# CLI commands (once built)
./heaplens web <dump-file> --open
./heaplens tui <dump-file>
./heaplens top <dump-file> -n 20
./heaplens paths <dump-file> --id=<objID>
./heaplens dominators <dump-file> -n 20
./heaplens retained <dump-file> --ids=<id1>,<id2>

# Run directly without building
go run cmd/heaplens/main.go web testdata/simple.json
```

## Implementation Status

The project is in early development following a TDD approach with these increments:

1. **Skateboard (Basic Graph Analysis)** - NOT STARTED
   - Core data model, parser registry, JSON parser, paths algorithm, CLI paths command
   
2. **Bicycle (Complete Analysis Suite)** - NOT STARTED
   - Dominators, retained size, type aggregation, remaining CLI commands
   
3. **Motorcycle (Web UI)** - NOT STARTED
   - HTTP handler, templates, all web views
   
4. **Car (Production Ready)** - NOT STARTED
   - Go heap parser, TUI, performance optimizations

## Key Design Decisions

- **TDD Approach**: Write tests before implementation
- **Minimal Dependencies**: Stdlib + optional TUI libraries only
- **No JS Build**: Server-side rendered templates for Web UI
- **Streaming Parse**: Handle 5-10GB dumps with bounded memory usage
- **Pluggable Parsers**: Support multiple dump formats via interface
- **Mountable Handler**: Web UI can be embedded at any path (e.g., `/debug/heaplens`)

## Testing Strategy

### Unit Tests
- Test each algorithm independently with known inputs/outputs
- Use golden files for regression testing
- Maintain 80% code coverage minimum

### Integration Tests
- Generate real dumps in test binaries
- Validate known objects appear in analysis results
- Test CLI commands against fixtures

### Performance Tests
- Benchmark algorithms on graphs with 1M, 10M, 100M objects
- Ensure memory usage stays under 2x dump size
- Target <10s for 10M object analysis

## Current TODOs

Refer to `todo.md` for detailed task tracking. Current focus is on Increment 1 (Skateboard):
1. Project setup and structure
2. Core data model implementation
3. Parser registry system
4. JSON stub parser
5. Paths-to-roots algorithm
6. CLI paths command
7. Integration tests

## Performance Targets

- Parse 5GB dump in <2GB RAM
- Complete algorithms in <30s for 10M objects
- Web UI loads in <500ms
- Dominators: O(E α(E,V)) complexity

## Python Development (if needed)

### Package Management with uv
- **Package manager**: Use `uv` for all Python package management
- **No requirements.txt**: Packages are stored in `pyproject.toml`
- **Run scripts**: `uv run <script.py>`
- **Add packages**: `uv add <package>`
- **Initialize project**: `uv init` (if pyproject.toml doesn't exist)

## Workflow Requirements

### Task Tracking
- **Always check todo.md**: Mark completed tasks as done when finishing work
- **Update progress**: Keep todo.md current with implementation status

### Quality Gates
- **Tests MUST pass**: Never mark a task complete until all tests pass
  - Run `go test ./...` before considering any task done
  - Fix any failing tests before moving on
  
- **Linting MUST pass**: Code must pass linting checks
  - Run `go fmt ./...` to format code
  - Run `go vet ./...` to check for issues
  - Run `staticcheck ./...` if available
  - All linting issues must be resolved before task completion

## Important Notes

- Go version requirement: 1.22+
- No CGO dependencies
- Works on Linux/macOS
- Security: Web UI should be mounted behind auth in production
- Path traversal prevention required for dump file access
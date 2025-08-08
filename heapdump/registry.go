// ABOUTME: Registry for heap dump parsers
// ABOUTME: Manages parser plugins and selects appropriate parser for dumps

package heapdump

import (
	"bytes"
	"errors"
	"io"
	"sync"

	"github.com/prateek/heaplens/graph"
)

var (
	// ErrNoParser is returned when no parser can handle the dump format
	ErrNoParser = errors.New("no parser found for dump format")
)

// parserRegistry holds registered parsers
type parserRegistry struct {
	mu      sync.RWMutex
	parsers []Parser
}

// Global registry instance
var registry = &parserRegistry{
	parsers: make([]Parser, 0),
}

// Register adds a parser to the registry
func Register(p Parser) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.parsers = append(registry.parsers, p)
}

// Open reads a heap dump and returns a graph
// It tries each registered parser to find one that can handle the format
func Open(r io.Reader) (graph.Graph, error) {
	// Read some bytes for format detection
	// We need to buffer since we'll try multiple parsers
	buf := new(bytes.Buffer)
	tee := io.TeeReader(r, buf)
	
	// Try to read enough for format detection
	detectBuf := make([]byte, 4096)
	n, err := tee.Read(detectBuf)
	if err != nil && err != io.EOF {
		return nil, err
	}
	
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	
	// Try each parser
	for _, parser := range registry.parsers {
		// Create a fresh reader for CanParse check
		checkReader := bytes.NewReader(detectBuf[:n])
		if parser.CanParse(checkReader) {
			// Create fresh reader for actual parsing
			parseReader := io.MultiReader(bytes.NewReader(detectBuf[:n]), r)
			return parser.Parse(parseReader)
		}
	}
	
	return nil, ErrNoParser
}
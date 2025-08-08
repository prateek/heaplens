// ABOUTME: Parser interface for heap dump formats
// ABOUTME: Defines the contract for pluggable dump parsers

package heapdump

import (
	"io"

	"github.com/prateek/heaplens/graph"
)

// Parser is the interface for heap dump parsers
type Parser interface {
	// CanParse checks if this parser can handle the given dump format
	// The reader should be treated as a preview - implementations should
	// read a small amount to detect format and not consume the entire stream
	CanParse(r io.Reader) bool
	
	// Parse reads the dump and builds a graph
	// The reader will be a fresh reader positioned at the start
	Parse(r io.Reader) (graph.Graph, error)
}
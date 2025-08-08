// ABOUTME: Tests for the parser registry system
// ABOUTME: Validates parser registration and selection

package heapdump

import (
	"io"
	"strings"
	"testing"

	"github.com/prateek/heaplens/graph"
)

// mockParser is a test parser implementation
type mockParser struct {
	name string
}

func (p *mockParser) CanParse(r io.Reader) bool {
	// Check if first line contains parser name
	buf := make([]byte, 100)
	n, _ := r.Read(buf)
	return strings.Contains(string(buf[:n]), p.name)
}

func (p *mockParser) Parse(r io.Reader) (graph.Graph, error) {
	return graph.NewMemGraph(), nil
}

func TestRegister(t *testing.T) {
	// Clear registry for test
	registry = &parserRegistry{
		parsers: make([]Parser, 0),
	}
	
	parser1 := &mockParser{name: "parser1"}
	parser2 := &mockParser{name: "parser2"}
	
	Register(parser1)
	Register(parser2)
	
	if len(registry.parsers) != 2 {
		t.Errorf("Expected 2 parsers registered, got %d", len(registry.parsers))
	}
}

func TestOpen(t *testing.T) {
	// Clear and setup registry
	registry = &parserRegistry{
		parsers: make([]Parser, 0),
	}
	
	jsonParser := &mockParser{name: "json"}
	goParser := &mockParser{name: "goheap"}
	
	Register(jsonParser)
	Register(goParser)
	
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "JSON file",
			content: "json dump data",
			wantErr: false,
		},
		{
			name:    "Go heap file",
			content: "goheap dump data",
			wantErr: false,
		},
		{
			name:    "Unknown format",
			content: "unknown format",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.content)
			_, err := Open(r)
			
			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestMultipleParserRegistration(t *testing.T) {
	// Clear registry
	registry = &parserRegistry{
		parsers: make([]Parser, 0),
	}
	
	// Register multiple parsers that can handle same format
	// Last registered should take precedence
	oldParser := &mockParser{name: "json"}
	newParser := &mockParser{name: "json"}
	
	Register(oldParser)
	Register(newParser)
	
	// Both should be in registry
	if len(registry.parsers) != 2 {
		t.Errorf("Expected 2 parsers, got %d", len(registry.parsers))
	}
}

func TestParserSelection(t *testing.T) {
	// Clear registry
	registry = &parserRegistry{
		parsers: make([]Parser, 0),
	}
	
	// Register parsers in specific order
	fallbackParser := &mockParser{name: "fallback"}
	specificParser := &mockParser{name: "specific"}
	
	Register(fallbackParser)
	Register(specificParser)
	
	// Test that the right parser is selected
	r := strings.NewReader("specific format data")
	g, err := Open(r)
	
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if g == nil {
		t.Error("Expected graph, got nil")
	}
}

func TestThreadSafeRegistry(t *testing.T) {
	// Clear registry
	registry = &parserRegistry{
		parsers: make([]Parser, 0),
	}
	
	// Concurrent registration should be safe
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			parser := &mockParser{name: string(rune('a' + id))}
			Register(parser)
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	if len(registry.parsers) != 10 {
		t.Errorf("Expected 10 parsers after concurrent registration, got %d", len(registry.parsers))
	}
}
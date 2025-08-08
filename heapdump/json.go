// ABOUTME: JSON stub parser for testing heap analysis algorithms
// ABOUTME: Reads a simple JSON format with objects and roots

package heapdump

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/prateek/heaplens/graph"
)

// JSONStub is a parser for JSON test dumps
type JSONStub struct{}

// jsonDump represents the JSON dump format
type jsonDump struct {
	Objects []jsonObject   `json:"objects"`
	Roots   []graph.ObjID  `json:"roots"`
}

// jsonObject represents an object in the JSON format
type jsonObject struct {
	ID   graph.ObjID   `json:"id"`
	Type string        `json:"type"`
	Size uint64        `json:"size"`
	Ptrs []graph.ObjID `json:"ptrs"`
}

// CanParse checks if the input looks like our JSON format
func (p *JSONStub) CanParse(r io.Reader) bool {
	// Read a small amount to check format
	buf := make([]byte, 1024)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		return false
	}
	
	if n == 0 {
		return false
	}
	
	// Check if it has the expected structure
	// We check for the presence of "objects" key in the JSON
	var test struct {
		Objects json.RawMessage `json:"objects"`
	}
	
	err = json.Unmarshal(buf[:n], &test)
	if err != nil {
		return false
	}
	
	// Must have objects field and it must not be null
	return test.Objects != nil
}

// Parse reads the JSON dump and builds a graph
func (p *JSONStub) Parse(r io.Reader) (graph.Graph, error) {
	var dump jsonDump
	
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&dump); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}
	
	// Validate required fields
	for i, obj := range dump.Objects {
		if obj.ID == 0 {
			return nil, fmt.Errorf("object at index %d missing ID", i)
		}
	}
	
	// Build the graph
	g := graph.NewMemGraph()
	
	for _, obj := range dump.Objects {
		graphObj := &graph.Object{
			ID:   obj.ID,
			Type: obj.Type,
			Size: obj.Size,
			Ptrs: obj.Ptrs,
		}
		if graphObj.Ptrs == nil {
			graphObj.Ptrs = []graph.ObjID{}
		}
		g.AddObject(graphObj)
	}
	
	// Set roots
	roots := graph.Roots{IDs: dump.Roots}
	if roots.IDs == nil {
		roots.IDs = []graph.ObjID{}
	}
	g.SetRoots(roots)
	
	return g, nil
}

// init registers the JSON parser
func init() {
	Register(&JSONStub{})
}
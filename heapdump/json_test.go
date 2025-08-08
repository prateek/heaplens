// ABOUTME: Tests for the JSON stub parser
// ABOUTME: Validates JSON parsing and error handling

package heapdump

import (
	"strings"
	"testing"
)

func TestJSONParse(t *testing.T) {
	jsonData := `{
		"objects": [
			{"id": 1, "type": "root", "size": 100, "ptrs": [2]},
			{"id": 2, "type": "child", "size": 50, "ptrs": []}
		],
		"roots": [1]
	}`
	
	parser := &JSONStub{}
	r := strings.NewReader(jsonData)
	
	g, err := parser.Parse(r)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	
	if g.NumObjects() != 2 {
		t.Errorf("Expected 2 objects, got %d", g.NumObjects())
	}
	
	obj1 := g.GetObject(1)
	if obj1 == nil {
		t.Fatal("Object 1 not found")
	}
	if obj1.Type != "root" {
		t.Errorf("Expected type 'root', got %s", obj1.Type)
	}
	if obj1.Size != 100 {
		t.Errorf("Expected size 100, got %d", obj1.Size)
	}
	if len(obj1.Ptrs) != 1 || obj1.Ptrs[0] != 2 {
		t.Errorf("Expected ptrs [2], got %v", obj1.Ptrs)
	}
	
	roots := g.GetRoots()
	if len(roots.IDs) != 1 || roots.IDs[0] != 1 {
		t.Errorf("Expected roots [1], got %v", roots.IDs)
	}
}

func TestJSONCanParse(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "Valid JSON object",
			content: `{"objects": [], "roots": []}`,
			want:    true,
		},
		{
			name:    "JSON with objects key",
			content: `{"objects": [{"id": 1}]}`,
			want:    true,
		},
		{
			name:    "Non-JSON",
			content: `not json at all`,
			want:    false,
		},
		{
			name:    "JSON without objects key",
			content: `{"data": []}`,
			want:    false,
		},
		{
			name:    "Empty",
			content: ``,
			want:    false,
		},
	}
	
	parser := &JSONStub{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.content)
			got := parser.CanParse(r)
			if got != tt.want {
				t.Errorf("CanParse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMalformedJSON(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "Invalid JSON syntax",
			content: `{"objects": [}`,
		},
		{
			name:    "Missing required fields",
			content: `{"objects": [{"type": "test"}]}`, // missing id
		},
		{
			name:    "Wrong type for objects",
			content: `{"objects": "not an array", "roots": []}`,
		},
	}
	
	parser := &JSONStub{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.content)
			_, err := parser.Parse(r)
			if err == nil {
				t.Error("Expected error for malformed JSON")
			}
		})
	}
}

func TestJSONWithComplexGraph(t *testing.T) {
	// Test with cycles and multiple roots
	jsonData := `{
		"objects": [
			{"id": 1, "type": "root1", "size": 10, "ptrs": [2, 3]},
			{"id": 2, "type": "node", "size": 20, "ptrs": [3]},
			{"id": 3, "type": "node", "size": 30, "ptrs": [1]},
			{"id": 4, "type": "root2", "size": 40, "ptrs": [2]}
		],
		"roots": [1, 4]
	}`
	
	parser := &JSONStub{}
	r := strings.NewReader(jsonData)
	
	g, err := parser.Parse(r)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	
	if g.NumObjects() != 4 {
		t.Errorf("Expected 4 objects, got %d", g.NumObjects())
	}
	
	roots := g.GetRoots()
	if len(roots.IDs) != 2 {
		t.Errorf("Expected 2 roots, got %d", len(roots.IDs))
	}
}
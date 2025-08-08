// ABOUTME: Integration tests for the complete HeapLens system
// ABOUTME: Validates end-to-end functionality with JSON dumps

package heaplens_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/prateek/heaplens/graph"
	"github.com/prateek/heaplens/heapdump"
)

func TestEndToEndJSONParsing(t *testing.T) {
	// Open the test data file
	file, err := os.Open("testdata/simple.json")
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer file.Close()
	
	// Parse the dump
	g, err := heapdump.Open(file)
	if err != nil {
		t.Fatalf("Failed to parse dump: %v", err)
	}
	
	// Verify objects
	if g.NumObjects() != 5 {
		t.Errorf("Expected 5 objects, got %d", g.NumObjects())
	}
	
	// Verify specific objects
	obj1 := g.GetObject(1)
	if obj1 == nil {
		t.Fatal("Object 1 not found")
	}
	if obj1.Type != "root" {
		t.Errorf("Expected type 'root', got %s", obj1.Type)
	}
	
	// Verify roots
	roots := g.GetRoots()
	if len(roots.IDs) != 1 || roots.IDs[0] != 1 {
		t.Errorf("Expected roots [1], got %v", roots.IDs)
	}
}

func TestPathFindingIntegration(t *testing.T) {
	// Open and parse
	file, err := os.Open("testdata/simple.json")
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer file.Close()
	
	g, err := heapdump.Open(file)
	if err != nil {
		t.Fatalf("Failed to parse dump: %v", err)
	}
	
	// Test paths from different objects
	tests := []struct {
		name     string
		from     graph.ObjID
		wantLen  int
		checkEnd graph.ObjID // Last element should be this root
	}{
		{
			name:     "Path from leaf object 4",
			from:     4,
			wantLen:  3, // 4 -> 3 -> 1
			checkEnd: 1,
		},
		{
			name:     "Path from leaf object 5",
			from:     5,
			wantLen:  3, // 5 -> 3 -> 1
			checkEnd: 1,
		},
		{
			name:     "Path from middle object 3",
			from:     3,
			wantLen:  2, // 3 -> 1
			checkEnd: 1,
		},
		{
			name:     "Path from object 2",
			from:     2,
			wantLen:  2, // 2 -> 1
			checkEnd: 1,
		},
		{
			name:     "Path from root itself",
			from:     1,
			wantLen:  1, // Just 1
			checkEnd: 1,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := graph.PathsToRoots(g, tt.from, 5)
			
			if len(paths) == 0 {
				t.Fatal("No paths found")
			}
			
			path := paths[0]
			if len(path.IDs) != tt.wantLen {
				t.Errorf("Path length = %d, want %d", len(path.IDs), tt.wantLen)
			}
			
			// Check that path ends at expected root
			if path.IDs[len(path.IDs)-1] != tt.checkEnd {
				t.Errorf("Path ends at %d, want %d", path.IDs[len(path.IDs)-1], tt.checkEnd)
			}
			
			// Check that path starts at the requested object
			if path.IDs[0] != tt.from {
				t.Errorf("Path starts at %d, want %d", path.IDs[0], tt.from)
			}
		})
	}
}

func TestComplexGraphIntegration(t *testing.T) {
	// Create a more complex test file
	complexJSON := `{
		"objects": [
			{"id": 1, "type": "root1", "size": 10, "ptrs": [3, 4]},
			{"id": 2, "type": "root2", "size": 20, "ptrs": [4, 5]},
			{"id": 3, "type": "shared1", "size": 30, "ptrs": [6]},
			{"id": 4, "type": "shared2", "size": 40, "ptrs": [6, 7]},
			{"id": 5, "type": "branch", "size": 50, "ptrs": [7]},
			{"id": 6, "type": "leaf1", "size": 60, "ptrs": []},
			{"id": 7, "type": "leaf2", "size": 70, "ptrs": []}
		],
		"roots": [1, 2]
	}`
	
	// Write to temp file
	tmpdir := t.TempDir()
	tmpfile := filepath.Join(tmpdir, "complex.json")
	if err := os.WriteFile(tmpfile, []byte(complexJSON), 0644); err != nil {
		t.Fatal(err)
	}
	
	// Parse
	file, err := os.Open(tmpfile)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	
	g, err := heapdump.Open(file)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	
	// Object 6 should have multiple paths (through both roots)
	paths := graph.PathsToRoots(g, 6, 10)
	if len(paths) < 2 {
		t.Errorf("Expected multiple paths for object 6, got %d", len(paths))
	}
	
	// Object 7 should also have multiple paths
	paths = graph.PathsToRoots(g, 7, 10)
	if len(paths) < 2 {
		t.Errorf("Expected multiple paths for object 7, got %d", len(paths))
	}
	
	// Verify both roots are present
	roots := g.GetRoots()
	if len(roots.IDs) != 2 {
		t.Errorf("Expected 2 roots, got %d", len(roots.IDs))
	}
}

func TestCyclicGraphIntegration(t *testing.T) {
	// Create a graph with cycles
	cyclicJSON := `{
		"objects": [
			{"id": 1, "type": "root", "size": 10, "ptrs": [2]},
			{"id": 2, "type": "node1", "size": 20, "ptrs": [3]},
			{"id": 3, "type": "node2", "size": 30, "ptrs": [4, 2]},
			{"id": 4, "type": "node3", "size": 40, "ptrs": [3]}
		],
		"roots": [1]
	}`
	
	tmpdir := t.TempDir()
	tmpfile := filepath.Join(tmpdir, "cyclic.json")
	if err := os.WriteFile(tmpfile, []byte(cyclicJSON), 0644); err != nil {
		t.Fatal(err)
	}
	
	file, err := os.Open(tmpfile)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	
	g, err := heapdump.Open(file)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	
	// All objects should find path to root despite cycles
	for id := graph.ObjID(2); id <= 4; id++ {
		paths := graph.PathsToRoots(g, id, 5)
		if len(paths) == 0 {
			t.Errorf("No path found for object %d despite being reachable", id)
		}
	}
}

func TestEmptyGraph(t *testing.T) {
	emptyJSON := `{
		"objects": [],
		"roots": []
	}`
	
	tmpdir := t.TempDir()
	tmpfile := filepath.Join(tmpdir, "empty.json")
	if err := os.WriteFile(tmpfile, []byte(emptyJSON), 0644); err != nil {
		t.Fatal(err)
	}
	
	file, err := os.Open(tmpfile)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	
	g, err := heapdump.Open(file)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	
	if g.NumObjects() != 0 {
		t.Errorf("Expected 0 objects in empty graph, got %d", g.NumObjects())
	}
	
	roots := g.GetRoots()
	if len(roots.IDs) != 0 {
		t.Errorf("Expected 0 roots in empty graph, got %d", len(roots.IDs))
	}
}
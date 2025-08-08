// ABOUTME: Tests for the paths-to-roots algorithm
// ABOUTME: Validates BFS path finding and cycle handling

package graph

import (
	"reflect"
	"testing"
)

func TestPathsToRoots(t *testing.T) {
	// Create test graph:
	// 1 (root) -> 2 -> 3
	//          -> 4
	g := NewMemGraph()
	g.AddObject(&Object{ID: 1, Type: "root", Ptrs: []ObjID{2}})
	g.AddObject(&Object{ID: 2, Type: "middle", Ptrs: []ObjID{3, 4}})
	g.AddObject(&Object{ID: 3, Type: "leaf1", Ptrs: []ObjID{}})
	g.AddObject(&Object{ID: 4, Type: "leaf2", Ptrs: []ObjID{}})
	g.SetRoots(Roots{IDs: []ObjID{1}})
	
	tests := []struct {
		name     string
		from     ObjID
		maxPaths int
		want     []Path
	}{
		{
			name:     "Direct path from root",
			from:     1,
			maxPaths: 5,
			want: []Path{
				{IDs: []ObjID{1}},
			},
		},
		{
			name:     "One hop from root",
			from:     2,
			maxPaths: 5,
			want: []Path{
				{IDs: []ObjID{2, 1}},
			},
		},
		{
			name:     "Two hops from root",
			from:     3,
			maxPaths: 5,
			want: []Path{
				{IDs: []ObjID{3, 2, 1}},
			},
		},
		{
			name:     "Another two hops path",
			from:     4,
			maxPaths: 5,
			want: []Path{
				{IDs: []ObjID{4, 2, 1}},
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := PathsToRoots(g, tt.from, tt.maxPaths)
			if !reflect.DeepEqual(paths, tt.want) {
				t.Errorf("PathsToRoots() = %v, want %v", paths, tt.want)
			}
		})
	}
}

func TestPathsWithCycles(t *testing.T) {
	// Create graph with cycle:
	// 1 (root) -> 2 -> 3 -> 2 (cycle)
	g := NewMemGraph()
	g.AddObject(&Object{ID: 1, Type: "root", Ptrs: []ObjID{2}})
	g.AddObject(&Object{ID: 2, Type: "cycle1", Ptrs: []ObjID{3}})
	g.AddObject(&Object{ID: 3, Type: "cycle2", Ptrs: []ObjID{2}})
	g.SetRoots(Roots{IDs: []ObjID{1}})
	
	// Should find path without getting stuck in cycle
	paths := PathsToRoots(g, 3, 5)
	want := []Path{{IDs: []ObjID{3, 2, 1}}}
	
	if !reflect.DeepEqual(paths, want) {
		t.Errorf("PathsToRoots() with cycle = %v, want %v", paths, want)
	}
}

func TestUnreachableObject(t *testing.T) {
	// Create disconnected graph
	g := NewMemGraph()
	g.AddObject(&Object{ID: 1, Type: "root", Ptrs: []ObjID{2}})
	g.AddObject(&Object{ID: 2, Type: "connected", Ptrs: []ObjID{}})
	g.AddObject(&Object{ID: 3, Type: "disconnected", Ptrs: []ObjID{}})
	g.SetRoots(Roots{IDs: []ObjID{1}})
	
	// Object 3 is not reachable from any root
	paths := PathsToRoots(g, 3, 5)
	
	if len(paths) != 0 {
		t.Errorf("Expected no paths for unreachable object, got %v", paths)
	}
}

func TestMultipleRoots(t *testing.T) {
	// Create graph with multiple roots:
	// 1 (root) -> 3
	// 2 (root) -> 3
	g := NewMemGraph()
	g.AddObject(&Object{ID: 1, Type: "root1", Ptrs: []ObjID{3}})
	g.AddObject(&Object{ID: 2, Type: "root2", Ptrs: []ObjID{3}})
	g.AddObject(&Object{ID: 3, Type: "shared", Ptrs: []ObjID{}})
	g.SetRoots(Roots{IDs: []ObjID{1, 2}})
	
	paths := PathsToRoots(g, 3, 5)
	
	// Should find 2 paths (one through each root)
	if len(paths) != 2 {
		t.Errorf("Expected 2 paths with multiple roots, got %d", len(paths))
	}
	
	// Check that we got paths through both roots
	hasPath1 := false
	hasPath2 := false
	for _, p := range paths {
		if len(p.IDs) == 2 {
			if p.IDs[1] == 1 {
				hasPath1 = true
			}
			if p.IDs[1] == 2 {
				hasPath2 = true
			}
		}
	}
	
	if !hasPath1 || !hasPath2 {
		t.Errorf("Expected paths through both roots, got %v", paths)
	}
}

func TestMaxPaths(t *testing.T) {
	// Create graph with many paths:
	// 1 (root) -> 4
	// 2 (root) -> 4  
	// 3 (root) -> 4
	g := NewMemGraph()
	g.AddObject(&Object{ID: 1, Type: "root1", Ptrs: []ObjID{4}})
	g.AddObject(&Object{ID: 2, Type: "root2", Ptrs: []ObjID{4}})
	g.AddObject(&Object{ID: 3, Type: "root3", Ptrs: []ObjID{4}})
	g.AddObject(&Object{ID: 4, Type: "target", Ptrs: []ObjID{}})
	g.SetRoots(Roots{IDs: []ObjID{1, 2, 3}})
	
	// Request only 2 paths
	paths := PathsToRoots(g, 4, 2)
	
	if len(paths) != 2 {
		t.Errorf("Expected at most 2 paths, got %d", len(paths))
	}
}

func TestSelfReference(t *testing.T) {
	// Object pointing to itself
	g := NewMemGraph()
	g.AddObject(&Object{ID: 1, Type: "root", Ptrs: []ObjID{2}})
	g.AddObject(&Object{ID: 2, Type: "self", Ptrs: []ObjID{2}}) // points to itself
	g.SetRoots(Roots{IDs: []ObjID{1}})
	
	paths := PathsToRoots(g, 2, 5)
	want := []Path{{IDs: []ObjID{2, 1}}}
	
	if !reflect.DeepEqual(paths, want) {
		t.Errorf("PathsToRoots() with self-reference = %v, want %v", paths, want)
	}
}
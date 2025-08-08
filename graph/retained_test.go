// ABOUTME: Tests for retained memory size calculation using dominator trees
// ABOUTME: Verifies that retained sizes are correctly computed for various graph topologies
package graph

import (
	"reflect"
	"testing"
)

func TestRetainedSize(t *testing.T) {
	tests := []struct {
		name     string
		graph    Graph
		expected map[ObjID]uint64 // node -> retained size
	}{
		{
			name: "simple linear chain",
			graph: func() Graph {
				g := NewMemGraph()
				g.AddObject(&Object{ID: 1, Type: "root", Size: 100, Ptrs: []ObjID{2}})
				g.AddObject(&Object{ID: 2, Type: "node", Size: 50, Ptrs: []ObjID{3}})
				g.AddObject(&Object{ID: 3, Type: "leaf", Size: 25})
				g.SetRoots(Roots{IDs: []ObjID{1}})
				return g
			}(),
			expected: map[ObjID]uint64{
				1: 175, // 100 + 50 + 25 (retains everything)
				2: 75,  // 50 + 25 (retains itself and 3)
				3: 25,  // 25 (retains only itself)
			},
		},
		{
			name: "diamond pattern",
			graph: func() Graph {
				g := NewMemGraph()
				g.AddObject(&Object{ID: 1, Type: "root", Size: 100, Ptrs: []ObjID{2, 3}})
				g.AddObject(&Object{ID: 2, Type: "left", Size: 30, Ptrs: []ObjID{4}})
				g.AddObject(&Object{ID: 3, Type: "right", Size: 40, Ptrs: []ObjID{4}})
				g.AddObject(&Object{ID: 4, Type: "merge", Size: 20})
				g.SetRoots(Roots{IDs: []ObjID{1}})
				return g
			}(),
			expected: map[ObjID]uint64{
				1: 190, // 100 + 30 + 40 + 20 (root retains all)
				2: 30,  // 30 (only itself, as 4 is dominated by 1)
				3: 40,  // 40 (only itself, as 4 is dominated by 1)
				4: 20,  // 20 (only itself)
			},
		},
		{
			name: "tree structure",
			graph: func() Graph {
				g := NewMemGraph()
				g.AddObject(&Object{ID: 1, Type: "root", Size: 100, Ptrs: []ObjID{2, 3}})
				g.AddObject(&Object{ID: 2, Type: "left", Size: 30, Ptrs: []ObjID{4}})
				g.AddObject(&Object{ID: 3, Type: "right", Size: 40, Ptrs: []ObjID{5}})
				g.AddObject(&Object{ID: 4, Type: "left-child", Size: 15})
				g.AddObject(&Object{ID: 5, Type: "right-child", Size: 25})
				g.SetRoots(Roots{IDs: []ObjID{1}})
				return g
			}(),
			expected: map[ObjID]uint64{
				1: 210, // 100 + 30 + 40 + 15 + 25 (retains all)
				2: 45,  // 30 + 15 (retains itself and 4)
				3: 65,  // 40 + 25 (retains itself and 5)
				4: 15,  // 15 (only itself)
				5: 25,  // 25 (only itself)
			},
		},
		{
			name: "multiple roots",
			graph: func() Graph {
				g := NewMemGraph()
				g.AddObject(&Object{ID: 1, Type: "root1", Size: 100, Ptrs: []ObjID{3}})
				g.AddObject(&Object{ID: 2, Type: "root2", Size: 200, Ptrs: []ObjID{3}})
				g.AddObject(&Object{ID: 3, Type: "shared", Size: 50})
				g.SetRoots(Roots{IDs: []ObjID{1, 2}})
				return g
			}(),
			expected: map[ObjID]uint64{
				1: 100, // 100 (only itself, as 3 is dominated by super-root)
				2: 200, // 200 (only itself, as 3 is dominated by super-root)
				3: 50,  // 50 (only itself)
			},
		},
		{
			name: "unreachable objects",
			graph: func() Graph {
				g := NewMemGraph()
				g.AddObject(&Object{ID: 1, Type: "root", Size: 100, Ptrs: []ObjID{2}})
				g.AddObject(&Object{ID: 2, Type: "reachable", Size: 50})
				g.AddObject(&Object{ID: 3, Type: "unreachable", Size: 75})
				g.SetRoots(Roots{IDs: []ObjID{1}})
				return g
			}(),
			expected: map[ObjID]uint64{
				1: 150, // 100 + 50 (only reachable objects)
				2: 50,  // 50 (only itself)
				// 3 is unreachable, not in retained sizes
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retained := RetainedSize(tt.graph)
			
			if len(retained) != len(tt.expected) {
				t.Errorf("got %d retained sizes, want %d", len(retained), len(tt.expected))
			}
			
			for node, expectedSize := range tt.expected {
				if gotSize, ok := retained[node]; !ok {
					t.Errorf("node %d: missing from retained sizes", node)
				} else if gotSize != expectedSize {
					t.Errorf("node %d: retained size = %d, want %d", node, gotSize, expectedSize)
				}
			}
			
			// Check that we don't have unexpected retained sizes
			for node, gotSize := range retained {
				if expectedSize, ok := tt.expected[node]; !ok {
					t.Errorf("node %d: unexpected retained size %d", node, gotSize)
				} else if gotSize != expectedSize {
					t.Errorf("node %d: retained size = %d, want %d", node, gotSize, expectedSize)
				}
			}
		})
	}
}

func TestRetainedSizePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}
	
	// Create a large tree structure
	n := 10000
	graph := NewMemGraph()
	
	for i := 1; i <= n; i++ {
		obj := &Object{
			ID:   ObjID(i),
			Type: "node",
			Size: uint64(10 + i%100), // varied sizes
		}
		
		// Create tree structure: each node has up to 3 children
		for j := 1; j <= 3; j++ {
			child := i*3 + j
			if child <= n {
				obj.Ptrs = append(obj.Ptrs, ObjID(child))
			}
		}
		
		graph.AddObject(obj)
	}
	graph.SetRoots(Roots{IDs: []ObjID{1}})
	
	retained := RetainedSize(graph)
	
	// Check that we got retained sizes for all reachable nodes
	if len(retained) == 0 {
		t.Error("no retained sizes computed")
	}
	
	// Verify root has largest retained size
	rootRetained, exists := retained[1]
	if !exists {
		t.Error("no retained size for root")
	}
	
	for _, size := range retained {
		if size > rootRetained {
			t.Error("found node with larger retained size than root")
		}
	}
	
	t.Logf("computed retained sizes for %d nodes", len(retained))
}

// TestRetainedSizeWithDominators ensures that retained size calculation
// is consistent with dominator relationships
func TestRetainedSizeWithDominators(t *testing.T) {
	graph := func() Graph {
		g := NewMemGraph()
		g.AddObject(&Object{ID: 1, Type: "root", Size: 100, Ptrs: []ObjID{2, 3}})
		g.AddObject(&Object{ID: 2, Type: "a", Size: 30, Ptrs: []ObjID{4}})
		g.AddObject(&Object{ID: 3, Type: "b", Size: 40, Ptrs: []ObjID{4, 5}})
		g.AddObject(&Object{ID: 4, Type: "c", Size: 20})
		g.AddObject(&Object{ID: 5, Type: "d", Size: 15})
		g.SetRoots(Roots{IDs: []ObjID{1}})
		return g
	}()
	
	dominators := Dominators(graph)
	retained := RetainedSize(graph)
	
	// Verify that if A dominates B, then A's retained size >= B's retained size
	// Skip super-root (nodeA == 0) since it has size 0 but dominates everything
	for nodeB, nodeA := range dominators {
		if nodeA == 0 {
			continue // Skip super-root comparisons
		}
		
		retainedA := retained[nodeA]
		retainedB := retained[nodeB]
		
		if retainedA < retainedB {
			t.Errorf("dominator %d has smaller retained size (%d) than dominated %d (%d)",
				nodeA, retainedA, nodeB, retainedB)
		}
	}
	
	// Verify that retained size of a node is at least its own size
	graph.ForEachObject(func(obj *Object) {
		if retainedSize, exists := retained[obj.ID]; exists {
			if retainedSize < obj.Size {
				t.Errorf("node %d: retained size %d < object size %d", 
					obj.ID, retainedSize, obj.Size)
			}
		}
	})
}

// TestRetainedSizeSubsets tests that RetainedSizeSubsets works correctly
func TestRetainedSizeSubsets(t *testing.T) {
	graph := func() Graph {
		g := NewMemGraph()
		g.AddObject(&Object{ID: 1, Type: "root", Size: 100, Ptrs: []ObjID{2, 3}})
		g.AddObject(&Object{ID: 2, Type: "a", Size: 30, Ptrs: []ObjID{4}})
		g.AddObject(&Object{ID: 3, Type: "b", Size: 40})
		g.AddObject(&Object{ID: 4, Type: "c", Size: 20})
		g.SetRoots(Roots{IDs: []ObjID{1}})
		return g
	}()
	
	tests := []struct {
		name     string
		ids      []ObjID
		expected map[ObjID]uint64
	}{
		{
			name: "single node",
			ids:  []ObjID{2},
			expected: map[ObjID]uint64{
				2: 50, // 30 + 20 (node 2 retains node 4)
			},
		},
		{
			name: "multiple nodes",
			ids:  []ObjID{2, 3},
			expected: map[ObjID]uint64{
				2: 50, // 30 + 20
				3: 40, // 40
			},
		},
		{
			name: "nonexistent node",
			ids:  []ObjID{999},
			expected: map[ObjID]uint64{
				// empty - node doesn't exist
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retained := RetainedSizeSubsets(graph, tt.ids)
			
			if !reflect.DeepEqual(retained, tt.expected) {
				t.Errorf("retained sizes = %v, want %v", retained, tt.expected)
			}
		})
	}
}
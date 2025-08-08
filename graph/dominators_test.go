// ABOUTME: Tests for dominator tree computation using Lengauer-Tarjan algorithm
// ABOUTME: Verifies immediate dominators, dominator tree, and performance characteristics
package graph

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestDominators(t *testing.T) {
	tests := []struct {
		name     string
		graph    Graph
		expected map[ObjID]ObjID // node -> immediate dominator
	}{
		{
			name: "simple linear chain",
			graph: func() Graph {
				g := NewMemGraph()
				g.AddObject(&Object{ID: 1, Type: "root"})
				g.AddObject(&Object{ID: 2, Type: "node", Ptrs: []ObjID{3}})
				g.AddObject(&Object{ID: 3, Type: "node", Ptrs: []ObjID{4}})
				g.AddObject(&Object{ID: 4, Type: "leaf"})
				g.SetRoots(Roots{IDs: []ObjID{2}})
				return g
			}(),
			expected: map[ObjID]ObjID{
				2: 0, // root has no dominator
				3: 2,
				4: 3,
			},
		},
		{
			name: "diamond pattern",
			graph: func() Graph {
				g := NewMemGraph()
				g.AddObject(&Object{ID: 1, Type: "root", Ptrs: []ObjID{2, 3}})
				g.AddObject(&Object{ID: 2, Type: "left", Ptrs: []ObjID{4}})
				g.AddObject(&Object{ID: 3, Type: "right", Ptrs: []ObjID{4}})
				g.AddObject(&Object{ID: 4, Type: "merge"})
				g.SetRoots(Roots{IDs: []ObjID{1}})
				return g
			}(),
			expected: map[ObjID]ObjID{
				1: 0, // root
				2: 1,
				3: 1,
				4: 1, // dominated by root, not by 2 or 3
			},
		},
		{
			name: "complex graph with multiple paths",
			graph: func() Graph {
				g := NewMemGraph()
				g.AddObject(&Object{ID: 1, Type: "root", Ptrs: []ObjID{2, 3}})
				g.AddObject(&Object{ID: 2, Type: "a", Ptrs: []ObjID{4}})
				g.AddObject(&Object{ID: 3, Type: "b", Ptrs: []ObjID{4, 5}})
				g.AddObject(&Object{ID: 4, Type: "c", Ptrs: []ObjID{6}})
				g.AddObject(&Object{ID: 5, Type: "d", Ptrs: []ObjID{6}})
				g.AddObject(&Object{ID: 6, Type: "target"})
				g.SetRoots(Roots{IDs: []ObjID{1}})
				return g
			}(),
			expected: map[ObjID]ObjID{
				1: 0, // root
				2: 1,
				3: 1,
				4: 1,
				5: 3,
				6: 1,
			},
		},
		{
			name: "unreachable nodes",
			graph: func() Graph {
				g := NewMemGraph()
				g.AddObject(&Object{ID: 1, Type: "root", Ptrs: []ObjID{2}})
				g.AddObject(&Object{ID: 2, Type: "reachable"})
				g.AddObject(&Object{ID: 3, Type: "unreachable"})
				g.SetRoots(Roots{IDs: []ObjID{1}})
				return g
			}(),
			expected: map[ObjID]ObjID{
				1: 0,
				2: 1,
				// 3 is unreachable, not in dominators
			},
		},
		{
			name: "cycle in graph",
			graph: func() Graph {
				g := NewMemGraph()
				g.AddObject(&Object{ID: 1, Type: "root", Ptrs: []ObjID{2}})
				g.AddObject(&Object{ID: 2, Type: "a", Ptrs: []ObjID{3}})
				g.AddObject(&Object{ID: 3, Type: "b", Ptrs: []ObjID{4}})
				g.AddObject(&Object{ID: 4, Type: "c", Ptrs: []ObjID{2, 5}}) // back edge to 2
				g.AddObject(&Object{ID: 5, Type: "exit"})
				g.SetRoots(Roots{IDs: []ObjID{1}})
				return g
			}(),
			expected: map[ObjID]ObjID{
				1: 0,
				2: 1,
				3: 2,
				4: 3,
				5: 4,
			},
		},
		{
			name: "multiple roots",
			graph: func() Graph {
				g := NewMemGraph()
				g.AddObject(&Object{ID: 1, Type: "root1", Ptrs: []ObjID{3}})
				g.AddObject(&Object{ID: 2, Type: "root2", Ptrs: []ObjID{3}})
				g.AddObject(&Object{ID: 3, Type: "shared"})
				g.SetRoots(Roots{IDs: []ObjID{1, 2}})
				return g
			}(),
			expected: map[ObjID]ObjID{
				1: 0,
				2: 0,
				3: 0, // dominated by super-root
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dom := Dominators(tt.graph)
			
			if len(dom) != len(tt.expected) {
				t.Errorf("got %d dominators, want %d", len(dom), len(tt.expected))
			}
			
			for node, expectedDom := range tt.expected {
				if gotDom, ok := dom[node]; !ok {
					t.Errorf("node %d: missing from dominators", node)
				} else if gotDom != expectedDom {
					t.Errorf("node %d: dominator = %d, want %d", node, gotDom, expectedDom)
				}
			}
			
			// Check that we don't have unexpected dominators
			for node, gotDom := range dom {
				if expectedDom, ok := tt.expected[node]; !ok {
					t.Errorf("node %d: unexpected dominator %d", node, gotDom)
				} else if gotDom != expectedDom {
					t.Errorf("node %d: dominator = %d, want %d", node, gotDom, expectedDom)
				}
			}
		})
	}
}

func TestDominatorTree(t *testing.T) {
	graph := func() Graph {
		g := NewMemGraph()
		g.AddObject(&Object{ID: 1, Type: "root", Ptrs: []ObjID{2, 3}})
		g.AddObject(&Object{ID: 2, Type: "a", Ptrs: []ObjID{4}})
		g.AddObject(&Object{ID: 3, Type: "b", Ptrs: []ObjID{4, 5}})
		g.AddObject(&Object{ID: 4, Type: "c"})
		g.AddObject(&Object{ID: 5, Type: "d"})
		g.SetRoots(Roots{IDs: []ObjID{1}})
		return g
	}()
	
	dom := Dominators(graph)
	tree := DominatorTree(dom)
	
	expectedTree := map[ObjID][]ObjID{
		0: {1},    // super-root dominates root
		1: {2, 3, 4}, // root dominates a, b, c
		2: {},     // a has no dominated nodes
		3: {5},    // b dominates d
		4: {},     // c has no dominated nodes
		5: {},     // d has no dominated nodes
	}
	
	for parent, expectedChildren := range expectedTree {
		gotChildren := tree[parent]
		sort.Slice(gotChildren, func(i, j int) bool { return gotChildren[i] < gotChildren[j] })
		sort.Slice(expectedChildren, func(i, j int) bool { return expectedChildren[i] < expectedChildren[j] })
		
		if !reflect.DeepEqual(gotChildren, expectedChildren) {
			t.Errorf("node %d: children = %v, want %v", parent, gotChildren, expectedChildren)
		}
	}
}

func TestDominatorsPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}
	
	sizes := []int{1000, 10000, 100000}
	for _, n := range sizes {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			// Create a graph with branching factor 10
			graph := NewMemGraph()
			
			// Create tree-like structure with some cross-edges
			for i := 1; i <= n; i++ {
				obj := &Object{
					ID:   ObjID(i),
					Type: fmt.Sprintf("node%d", i),
				}
				
				// Add edges to create tree with cross-edges
				if i > 1 {
					parent := (i - 2) / 10 + 1
					obj.Ptrs = append(obj.Ptrs, ObjID(parent))
				}
				
				// Add some children
				for j := 1; j <= 10 && i*10+j <= n; j++ {
					child := i*10 + j
					if child <= n {
						obj.Ptrs = append(obj.Ptrs, ObjID(child))
					}
				}
				
				graph.AddObject(obj)
			}
			graph.SetRoots(Roots{IDs: []ObjID{1}})
			
			start := time.Now()
			dom := Dominators(graph)
			elapsed := time.Since(start)
			
			// Check that we got dominators for reachable nodes
			if len(dom) == 0 {
				t.Error("no dominators computed")
			}
			
			// Performance expectation: should be roughly O(n log n)
			// For 100k nodes, allow up to 60 seconds (very generous for CI environments)
			maxTime := time.Duration(n) * time.Microsecond * 600 // generous bound
			if n >= 100000 {
				maxTime = 60 * time.Second
			}
			if elapsed > maxTime {
				t.Errorf("took %v for n=%d, expected < %v", elapsed, n, maxTime)
			}
			
			t.Logf("n=%d: computed %d dominators in %v", n, len(dom), elapsed)
		})
	}
}

func BenchmarkDominators(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, n := range sizes {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			// Create test graph
			graph := NewMemGraph()
			
			for i := 1; i <= n; i++ {
				obj := &Object{
					ID:   ObjID(i),
					Type: "node",
				}
				
				// Create edges to form a complex graph
				if i > 1 {
					obj.Ptrs = append(obj.Ptrs, ObjID((i-1)/2+1)) // parent
				}
				if i*2 <= n {
					obj.Ptrs = append(obj.Ptrs, ObjID(i*2)) // left child
				}
				if i*2+1 <= n {
					obj.Ptrs = append(obj.Ptrs, ObjID(i*2+1)) // right child
				}
				
				graph.AddObject(obj)
			}
			graph.SetRoots(Roots{IDs: []ObjID{1}})
			
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				_ = Dominators(graph)
			}
		})
	}
}
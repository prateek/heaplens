// ABOUTME: Tests for the graph data structures and interfaces
// ABOUTME: Validates object creation, relationships, and graph operations

package graph

import (
	"testing"
)

func TestObjectCreation(t *testing.T) {
	obj := &Object{
		ID:      1,
		Type:    "string",
		Size:    42,
		Ptrs:    []ObjID{2, 3},
	}

	if obj.ID != 1 {
		t.Errorf("Expected ID 1, got %d", obj.ID)
	}
	if obj.Type != "string" {
		t.Errorf("Expected type 'string', got %s", obj.Type)
	}
	if obj.Size != 42 {
		t.Errorf("Expected size 42, got %d", obj.Size)
	}
	if len(obj.Ptrs) != 2 {
		t.Errorf("Expected 2 pointers, got %d", len(obj.Ptrs))
	}
}

func TestGraphInterface(t *testing.T) {
	g := NewMemGraph()
	
	// Add some objects
	obj1 := &Object{ID: 1, Type: "root", Size: 10, Ptrs: []ObjID{2}}
	obj2 := &Object{ID: 2, Type: "child", Size: 20, Ptrs: []ObjID{}}
	
	g.AddObject(obj1)
	g.AddObject(obj2)
	
	// Test object retrieval
	retrieved := g.GetObject(1)
	if retrieved == nil {
		t.Fatal("Expected to retrieve object 1")
	}
	if retrieved.ID != 1 {
		t.Errorf("Expected ID 1, got %d", retrieved.ID)
	}
	
	// Test object count
	if g.NumObjects() != 2 {
		t.Errorf("Expected 2 objects, got %d", g.NumObjects())
	}
	
	// Test iteration
	count := 0
	g.ForEachObject(func(obj *Object) {
		count++
	})
	if count != 2 {
		t.Errorf("Expected to iterate over 2 objects, got %d", count)
	}
	
	// Test roots
	g.SetRoots(Roots{IDs: []ObjID{1}})
	roots := g.GetRoots()
	if len(roots.IDs) != 1 || roots.IDs[0] != 1 {
		t.Errorf("Expected root [1], got %v", roots.IDs)
	}
}

func TestIDUniqueness(t *testing.T) {
	g := NewMemGraph()
	
	obj1 := &Object{ID: 1, Type: "first", Size: 10}
	obj2 := &Object{ID: 1, Type: "duplicate", Size: 20}
	
	g.AddObject(obj1)
	g.AddObject(obj2) // Should replace the first one
	
	if g.NumObjects() != 1 {
		t.Errorf("Expected 1 object after duplicate ID, got %d", g.NumObjects())
	}
	
	retrieved := g.GetObject(1)
	if retrieved.Type != "duplicate" {
		t.Errorf("Expected duplicate to replace first, got type %s", retrieved.Type)
	}
}

func TestObjectRelationships(t *testing.T) {
	g := NewMemGraph()
	
	// Create a simple graph: 1 -> 2 -> 3
	//                            -> 4
	obj1 := &Object{ID: 1, Type: "root", Size: 10, Ptrs: []ObjID{2}}
	obj2 := &Object{ID: 2, Type: "middle", Size: 20, Ptrs: []ObjID{3, 4}}
	obj3 := &Object{ID: 3, Type: "leaf1", Size: 30, Ptrs: []ObjID{}}
	obj4 := &Object{ID: 4, Type: "leaf2", Size: 40, Ptrs: []ObjID{}}
	
	g.AddObject(obj1)
	g.AddObject(obj2)
	g.AddObject(obj3)
	g.AddObject(obj4)
	
	// Verify relationships
	o1 := g.GetObject(1)
	if len(o1.Ptrs) != 1 || o1.Ptrs[0] != 2 {
		t.Errorf("Expected obj1 to point to [2], got %v", o1.Ptrs)
	}
	
	o2 := g.GetObject(2)
	if len(o2.Ptrs) != 2 {
		t.Errorf("Expected obj2 to have 2 pointers, got %d", len(o2.Ptrs))
	}
}

func TestNilObjectHandling(t *testing.T) {
	g := NewMemGraph()
	
	// Test getting non-existent object
	obj := g.GetObject(999)
	if obj != nil {
		t.Error("Expected nil for non-existent object")
	}
	
	// Test with empty graph
	if g.NumObjects() != 0 {
		t.Errorf("Expected 0 objects in empty graph, got %d", g.NumObjects())
	}
}
// ABOUTME: Graph interface and in-memory implementation
// ABOUTME: Provides methods for storing and querying heap object graphs

package graph

import "sync"

// Graph represents a heap object graph
type Graph interface {
	// AddObject adds an object to the graph
	AddObject(obj *Object)
	
	// GetObject retrieves an object by ID
	GetObject(id ObjID) *Object
	
	// NumObjects returns the total number of objects
	NumObjects() int
	
	// ForEachObject iterates over all objects
	ForEachObject(fn func(*Object))
	
	// SetRoots sets the GC roots
	SetRoots(roots Roots)
	
	// GetRoots returns the GC roots
	GetRoots() Roots
}

// MemGraph is an in-memory implementation of Graph
type MemGraph struct {
	mu      sync.RWMutex
	objects map[ObjID]*Object
	roots   Roots
}

// NewMemGraph creates a new in-memory graph
func NewMemGraph() *MemGraph {
	return &MemGraph{
		objects: make(map[ObjID]*Object),
	}
}

// AddObject adds an object to the graph
func (g *MemGraph) AddObject(obj *Object) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.objects[obj.ID] = obj
}

// GetObject retrieves an object by ID
func (g *MemGraph) GetObject(id ObjID) *Object {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.objects[id]
}

// NumObjects returns the total number of objects
func (g *MemGraph) NumObjects() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.objects)
}

// ForEachObject iterates over all objects
func (g *MemGraph) ForEachObject(fn func(*Object)) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, obj := range g.objects {
		fn(obj)
	}
}

// SetRoots sets the GC roots
func (g *MemGraph) SetRoots(roots Roots) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.roots = roots
}

// GetRoots returns the GC roots
func (g *MemGraph) GetRoots() Roots {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.roots
}
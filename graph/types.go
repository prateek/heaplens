// ABOUTME: Core data types for the heap object graph
// ABOUTME: Defines Object, ObjID, and Roots structures

package graph

// ObjID is a unique identifier for a heap object
type ObjID uint64

// Object represents a single heap object
type Object struct {
	ID   ObjID   // Unique identifier
	Type string  // Type name (e.g. "string", "*MyStruct")
	Size uint64  // Size in bytes
	Ptrs []ObjID // IDs of objects this object points to
}

// Roots represents the set of GC root objects
type Roots struct {
	IDs []ObjID // Object IDs that are roots
}
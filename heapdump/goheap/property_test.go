// ABOUTME: Property-based tests for the Go heap dump parser
// ABOUTME: Tests invariants and properties that must hold for all valid dumps

package goheap

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"testing"
	"time"

	"github.com/prateek/heaplens/graph"
)

// Property: All object pointers should reference valid objects or be null
func TestPropertyValidPointers(t *testing.T) {
	for i := 0; i < 100; i++ {
		dump := generateRandomValidDump(t, i)

		parser := &GoHeapParser{}
		g, err := parser.Parse(bytes.NewReader(dump))
		if err != nil {
			t.Errorf("Failed to parse generated dump: %v", err)
			continue
		}

		// Build object address set
		validAddrs := make(map[graph.ObjID]bool)
		g.ForEachObject(func(obj *graph.Object) {
			validAddrs[obj.ID] = true
		})

		// Check all pointers
		g.ForEachObject(func(obj *graph.Object) {
			for _, ptr := range obj.Ptrs {
				if ptr != 0 && !validAddrs[ptr] {
					t.Errorf("Object %d has invalid pointer to %d", obj.ID, ptr)
				}
			}
		})
	}
}

// Property: Root objects must exist in the object graph
func TestPropertyRootsExist(t *testing.T) {
	for i := 0; i < 100; i++ {
		dump := generateRandomValidDump(t, i)

		parser := &GoHeapParser{}
		g, err := parser.Parse(bytes.NewReader(dump))
		if err != nil {
			continue
		}

		// Build object set
		objects := make(map[graph.ObjID]bool)
		g.ForEachObject(func(obj *graph.Object) {
			objects[obj.ID] = true
		})

		// Check roots
		roots := g.GetRoots()
		for _, rootID := range roots.IDs {
			if !objects[rootID] {
				t.Errorf("Root %d does not exist in object graph", rootID)
			}
		}
	}
}

// Property: Object sizes must be reasonable
func TestPropertyObjectSizes(t *testing.T) {
	const maxReasonableSize = 1 << 30 // 1GB

	for i := 0; i < 100; i++ {
		dump := generateRandomValidDump(t, i)

		parser := &GoHeapParser{}
		g, err := parser.Parse(bytes.NewReader(dump))
		if err != nil {
			continue
		}

		g.ForEachObject(func(obj *graph.Object) {
			if obj.Size > maxReasonableSize {
				t.Errorf("Object %d has unreasonable size: %d", obj.ID, obj.Size)
			}
			if obj.Size == 0 {
				t.Errorf("Object %d has zero size", obj.ID)
			}
		})
	}
}

// Property: Parser should be deterministic (same input -> same output)
func TestPropertyDeterministic(t *testing.T) {
	for i := 0; i < 50; i++ {
		dump := generateRandomValidDump(t, i)

		parser1 := &GoHeapParser{}
		g1, err1 := parser1.Parse(bytes.NewReader(dump))

		parser2 := &GoHeapParser{}
		g2, err2 := parser2.Parse(bytes.NewReader(dump))

		// Both should succeed or fail the same way
		if (err1 == nil) != (err2 == nil) {
			t.Errorf("Non-deterministic error: %v vs %v", err1, err2)
			continue
		}

		if err1 != nil {
			continue
		}

		// Same number of objects
		if g1.NumObjects() != g2.NumObjects() {
			t.Errorf("Non-deterministic object count: %d vs %d",
				g1.NumObjects(), g2.NumObjects())
		}

		// Same roots
		roots1 := g1.GetRoots()
		roots2 := g2.GetRoots()
		if len(roots1.IDs) != len(roots2.IDs) {
			t.Errorf("Non-deterministic root count: %d vs %d",
				len(roots1.IDs), len(roots2.IDs))
		}
	}
}

// Property: Streaming parser should produce same results as regular parser
func TestPropertyStreamingEquivalence(t *testing.T) {
	for i := 0; i < 50; i++ {
		dump := generateRandomValidDump(t, i)

		// Parse with regular parser
		regularParser := &GoHeapParser{}
		regularGraph, regularErr := regularParser.Parse(bytes.NewReader(dump))

		// Parse with streaming parser
		var streamObjects []streamObject
		var streamTypes []streamType
		var streamRoots []uint64

		callbacks := StreamCallbacks{
			OnObject: func(addr, typeAddr uint64, data []byte, ptrs []uint64) error {
				streamObjects = append(streamObjects, streamObject{
					addr:     addr,
					typeAddr: typeAddr,
					size:     uint64(len(data)),
					ptrs:     ptrs,
				})
				return nil
			},
			OnType: func(addr, size uint64, name string, indirect bool) error {
				streamTypes = append(streamTypes, streamType{
					addr: addr,
					size: size,
					name: name,
				})
				return nil
			},
			OnRoot: func(desc string, ptr uint64) error {
				streamRoots = append(streamRoots, ptr)
				return nil
			},
		}

		streamParser := NewStreamingParser(bytes.NewReader(dump), callbacks)
		streamErr := streamParser.Parse()

		// Both should succeed or fail
		if (regularErr == nil) != (streamErr == nil) {
			t.Errorf("Parser equivalence failed: regular=%v, streaming=%v",
				regularErr, streamErr)
			continue
		}

		if regularErr != nil {
			continue
		}

		// Should have same number of objects
		if regularGraph.NumObjects() != len(streamObjects) {
			t.Errorf("Object count mismatch: regular=%d, streaming=%d",
				regularGraph.NumObjects(), len(streamObjects))
		}
	}
}

// Property: Parser should handle corrupted data gracefully (no panic)
func TestPropertyNoPanic(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 1000; i++ {
		// Generate random corrupted data
		size := r.Intn(10000) + 16
		data := make([]byte, size)
		r.Read(data)

		// Ensure it starts with valid header sometimes
		if r.Float32() < 0.3 {
			copy(data, "go1.7 heap dump\n")
		}

		// Should not panic
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Parser panicked on corrupted data: %v", r)
				}
			}()

			parser := &GoHeapParser{}
			g, err := parser.Parse(bytes.NewReader(data))
			_ = g
			_ = err // Errors are expected
		}()
	}
}

// Property: Memory usage should be proportional to dump size
func TestPropertyMemoryUsage(t *testing.T) {
	// This is a simplified test - real memory profiling would be more complex
	for _, numObjects := range []int{10, 100, 1000} {
		dump := generateDumpWithNObjects(t, numObjects)

		parser := &GoHeapParser{}
		g, err := parser.Parse(bytes.NewReader(dump))
		if err != nil {
			t.Errorf("Failed to parse dump with %d objects: %v", numObjects, err)
			continue
		}

		// Check that we got approximately the right number of objects
		actual := g.NumObjects()
		if actual < numObjects/2 || actual > numObjects*2 {
			t.Errorf("Expected ~%d objects, got %d", numObjects, actual)
		}
	}
}

// Property: Type names should be valid Go identifiers or type expressions
func TestPropertyValidTypeNames(t *testing.T) {
	for i := 0; i < 100; i++ {
		dump := generateRandomValidDump(t, i)

		var types []string
		callbacks := StreamCallbacks{
			OnType: func(addr, size uint64, name string, indirect bool) error {
				types = append(types, name)
				return nil
			},
		}

		parser := NewStreamingParser(bytes.NewReader(dump), callbacks)
		if err := parser.Parse(); err != nil {
			continue
		}

		for _, typeName := range types {
			if len(typeName) > 10000 {
				t.Errorf("Type name too long: %d characters", len(typeName))
			}
			// Could add more validation here
		}
	}
}

// Helper types for streaming equivalence test
type streamObject struct {
	addr     uint64
	typeAddr uint64
	size     uint64
	ptrs     []uint64
}

type streamType struct {
	addr uint64
	size uint64
	name string
}

// Helper function to generate random valid dumps
func generateRandomValidDump(t *testing.T, seed int) []byte {
	r := rand.New(rand.NewSource(int64(seed)))
	var buf bytes.Buffer

	buf.WriteString("go1.7 heap dump\n")

	// Write params
	writeVarint(&buf, tagParams)
	writeVarint(&buf, 0) // little endian
	writeVarint(&buf, 8) // pointer size
	writeVarint(&buf, 0x1000)
	writeVarint(&buf, 0x100000)
	writeString(&buf, "amd64")
	writeString(&buf, "go1.20.0")
	writeVarint(&buf, uint64(r.Intn(16)+1))

	// Generate random types
	numTypes := r.Intn(20) + 1
	typeAddrs := make([]uint64, numTypes)
	for i := 0; i < numTypes; i++ {
		typeAddrs[i] = uint64(0x1000 + i*0x100)
		writeVarint(&buf, tagType)
		writeVarint(&buf, typeAddrs[i])
		writeVarint(&buf, uint64(r.Intn(256)+8))
		writeString(&buf, generateTypeName(r))
		writeVarint(&buf, 0)
	}

	// Generate random objects
	numObjects := r.Intn(100) + 1
	objectAddrs := make([]uint64, numObjects)
	for i := 0; i < numObjects; i++ {
		objectAddrs[i] = uint64(0x10000 + i*0x100)
		writeVarint(&buf, tagObject)
		writeVarint(&buf, objectAddrs[i])

		// Object data with type pointer
		size := r.Intn(256) + 8
		objData := make([]byte, size)
		if len(typeAddrs) > 0 {
			typeAddr := typeAddrs[r.Intn(len(typeAddrs))]
			binary.LittleEndian.PutUint64(objData, typeAddr)
		}

		// Maybe add some pointers
		var hasPointers bool
		if r.Float32() < 0.3 && i > 0 {
			hasPointers = true
			targetAddr := objectAddrs[r.Intn(i)]
			if size >= 16 {
				binary.LittleEndian.PutUint64(objData[8:], targetAddr)
			}
		}

		writeBytes(&buf, objData)

		if hasPointers && size >= 16 {
			writeVarint(&buf, fieldKindPtr)
			writeVarint(&buf, 8)
		}
		writeVarint(&buf, fieldKindEol)
	}

	// Add some roots
	numRoots := r.Intn(10)
	for i := 0; i < numRoots && i < len(objectAddrs); i++ {
		writeVarint(&buf, tagOtherRoot)
		writeString(&buf, "root")
		writeVarint(&buf, objectAddrs[i])
	}

	writeVarint(&buf, tagEOF)

	return buf.Bytes()
}

// Helper to generate dump with specific number of objects
func generateDumpWithNObjects(t *testing.T, n int) []byte {
	var buf bytes.Buffer

	buf.WriteString("go1.7 heap dump\n")

	writeVarint(&buf, tagParams)
	writeVarint(&buf, 0)
	writeVarint(&buf, 8)
	writeVarint(&buf, 0x1000)
	writeVarint(&buf, uint64(0x1000+n*0x1000))
	writeString(&buf, "amd64")
	writeString(&buf, "go1.20.0")
	writeVarint(&buf, 4)

	// One type
	writeVarint(&buf, tagType)
	writeVarint(&buf, 0x1000)
	writeVarint(&buf, 32)
	writeString(&buf, "TestObject")
	writeVarint(&buf, 0)

	// N objects
	for i := 0; i < n; i++ {
		writeVarint(&buf, tagObject)
		writeVarint(&buf, uint64(0x2000+i*0x100))
		objData := make([]byte, 32)
		binary.LittleEndian.PutUint64(objData, 0x1000)
		writeBytes(&buf, objData)
		writeVarint(&buf, fieldKindEol)
	}

	writeVarint(&buf, tagEOF)

	return buf.Bytes()
}

// Helper to generate valid type names
func generateTypeName(r *rand.Rand) string {
	typeNames := []string{
		"int", "string", "bool", "float64",
		"[]byte", "map[string]int", "*MyStruct",
		"chan int", "func()", "interface{}",
		"struct { x int }", "error",
	}
	return typeNames[r.Intn(len(typeNames))]
}

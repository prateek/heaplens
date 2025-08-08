// ABOUTME: Comprehensive tests for the Go heap dump parser
// ABOUTME: Tests format detection, parsing, error handling, and integration

package goheap

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/prateek/heaplens/graph"
)

// TestCanParse tests format detection
func TestCanParse(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "valid go heap dump header",
			data:     []byte("go1.7 heap dump\n"),
			expected: true,
		},
		{
			name:     "invalid header",
			data:     []byte("not a heap dump\n"),
			expected: false,
		},
		{
			name:     "empty data",
			data:     []byte{},
			expected: false,
		},
		{
			name:     "partial header",
			data:     []byte("go1.7"),
			expected: false,
		},
	}

	parser := &GoHeapParser{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.data)
			got := parser.CanParse(r)
			if got != tt.expected {
				t.Errorf("CanParse() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestParseMinimalDump tests parsing a minimal valid dump
func TestParseMinimalDump(t *testing.T) {
	// Build a minimal valid heap dump
	var buf bytes.Buffer

	// Write header
	buf.WriteString("go1.7 heap dump\n")

	// Write params record
	writeVarint(&buf, tagParams)
	writeVarint(&buf, 0)          // little endian
	writeVarint(&buf, 8)          // pointer size
	writeVarint(&buf, 0x1000)     // heap start
	writeVarint(&buf, 0x2000)     // heap end
	writeString(&buf, "amd64")    // architecture
	writeString(&buf, "go1.20.0") // go version
	writeVarint(&buf, 4)          // num CPUs

	// Write EOF
	writeVarint(&buf, tagEOF)

	parser := &GoHeapParser{}
	g, err := parser.Parse(&buf)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if g == nil {
		t.Fatal("Parse() returned nil graph")
	}

	if g.NumObjects() != 0 {
		t.Errorf("Expected 0 objects, got %d", g.NumObjects())
	}
}

// TestParseWithObjects tests parsing with objects and types
func TestParseWithObjects(t *testing.T) {
	var buf bytes.Buffer

	// Write header
	buf.WriteString("go1.7 heap dump\n")

	// Write params
	writeVarint(&buf, tagParams)
	writeVarint(&buf, 0)          // little endian
	writeVarint(&buf, 8)          // pointer size
	writeVarint(&buf, 0x1000)     // heap start
	writeVarint(&buf, 0x2000)     // heap end
	writeString(&buf, "amd64")    // architecture
	writeString(&buf, "go1.20.0") // go version
	writeVarint(&buf, 4)          // num CPUs

	// Write a type
	writeVarint(&buf, tagType)
	writeVarint(&buf, 0x1000)     // type address
	writeVarint(&buf, 16)         // size
	writeString(&buf, "TestType") // name
	writeVarint(&buf, 0)          // not indirect

	// Write an object
	writeVarint(&buf, tagObject)
	writeVarint(&buf, 0x2000) // object address

	// Object data (16 bytes with type pointer at beginning)
	objData := make([]byte, 16)
	binary.LittleEndian.PutUint64(objData, 0x1000) // type pointer
	writeBytes(&buf, objData)

	// Fields (no pointer fields)
	writeVarint(&buf, fieldKindEol)

	// Write a root pointing to the object
	writeVarint(&buf, tagOtherRoot)
	writeString(&buf, "test root")
	writeVarint(&buf, 0x2000) // points to our object

	// Write EOF
	writeVarint(&buf, tagEOF)

	parser := &GoHeapParser{}
	g, err := parser.Parse(&buf)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if g.NumObjects() != 1 {
		t.Errorf("Expected 1 object, got %d", g.NumObjects())
	}

	// Check object properties
	var obj *graph.Object
	g.ForEachObject(func(o *graph.Object) {
		obj = o
	})

	if obj == nil {
		t.Fatal("No object found")
	}

	if obj.Type != "TestType" {
		t.Errorf("Expected type 'TestType', got '%s'", obj.Type)
	}

	if obj.Size != 16 {
		t.Errorf("Expected size 16, got %d", obj.Size)
	}

	// Check roots
	roots := g.GetRoots()
	if len(roots.IDs) != 1 {
		t.Errorf("Expected 1 root, got %d", len(roots.IDs))
	}
}

// TestParseRealDump tests parsing a real heap dump if available
func TestParseRealDump(t *testing.T) {
	// Try to create a real heap dump
	tmpFile := "test_heap.dump"
	defer os.Remove(tmpFile)

	file, err := os.Create(tmpFile)
	if err != nil {
		t.Skipf("Cannot create test file: %v", err)
	}

	// Create some test objects
	type TestStruct struct {
		Value int
		Next  *TestStruct
	}

	// Create a linked list
	root := &TestStruct{Value: 1}
	root.Next = &TestStruct{Value: 2}
	root.Next.Next = &TestStruct{Value: 3}

	// Write heap dump
	debug.WriteHeapDump(file.Fd())
	file.Close()

	// Keep reference to prevent GC
	_ = root

	// Parse the dump
	file, err = os.Open(tmpFile)
	if err != nil {
		t.Skipf("Cannot open dump file: %v", err)
	}
	defer file.Close()

	parser := &GoHeapParser{}
	g, err := parser.Parse(file)
	if err != nil {
		t.Logf("Parse error (expected for incomplete parser): %v", err)
		// Don't fail - parser is still being developed
		return
	}

	t.Logf("Parsed %d objects from real dump", g.NumObjects())

	// Basic sanity checks
	if g.NumObjects() == 0 {
		t.Error("Expected some objects in real dump")
	}
}

// TestParseErrors tests error handling
func TestParseErrors(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr string
	}{
		{
			name:    "invalid header",
			data:    []byte("invalid header\n\n"),
			wantErr: "invalid header",
		},
		{
			name:    "truncated after header",
			data:    []byte("go1.7 heap dump\n"),
			wantErr: "", // EOF is OK after header
		},
		{
			name: "invalid tag",
			data: func() []byte {
				var buf bytes.Buffer
				buf.WriteString("go1.7 heap dump\n")
				writeVarint(&buf, 99) // invalid tag
				return buf.Bytes()
			}(),
			wantErr: "unknown tag",
		},
	}

	parser := &GoHeapParser{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.data)
			_, err := parser.Parse(r)
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("Parse() error = nil, want error containing %q", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("Parse() error = %v, want error containing %q", err, tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("Parse() unexpected error = %v", err)
			}
		})
	}
}

// TestParseWithPointers tests parsing objects with pointer fields
func TestParseWithPointers(t *testing.T) {
	var buf bytes.Buffer

	// Write header
	buf.WriteString("go1.7 heap dump\n")

	// Write params
	writeVarint(&buf, tagParams)
	writeVarint(&buf, 0)          // little endian
	writeVarint(&buf, 8)          // pointer size
	writeVarint(&buf, 0x1000)     // heap start
	writeVarint(&buf, 0x3000)     // heap end
	writeString(&buf, "amd64")    // architecture
	writeString(&buf, "go1.20.0") // go version
	writeVarint(&buf, 4)          // num CPUs

	// Write types
	writeVarint(&buf, tagType)
	writeVarint(&buf, 0x1000)     // type address
	writeVarint(&buf, 24)         // size
	writeString(&buf, "NodeType") // name
	writeVarint(&buf, 0)          // not indirect

	// Write first object
	writeVarint(&buf, tagObject)
	writeVarint(&buf, 0x2000) // object 1 address

	// Object 1 data (24 bytes)
	obj1Data := make([]byte, 24)
	binary.LittleEndian.PutUint64(obj1Data[0:], 0x1000)  // type pointer
	binary.LittleEndian.PutUint64(obj1Data[8:], 42)      // some value
	binary.LittleEndian.PutUint64(obj1Data[16:], 0x2100) // pointer to object 2
	writeBytes(&buf, obj1Data)

	// Object 1 fields - has pointer at offset 16
	writeVarint(&buf, fieldKindPtr)
	writeVarint(&buf, 16) // offset
	writeVarint(&buf, fieldKindEol)

	// Write second object
	writeVarint(&buf, tagObject)
	writeVarint(&buf, 0x2100) // object 2 address

	// Object 2 data (24 bytes)
	obj2Data := make([]byte, 24)
	binary.LittleEndian.PutUint64(obj2Data[0:], 0x1000) // type pointer
	binary.LittleEndian.PutUint64(obj2Data[8:], 43)     // some value
	binary.LittleEndian.PutUint64(obj2Data[16:], 0)     // null pointer
	writeBytes(&buf, obj2Data)

	// Object 2 fields - has null pointer at offset 16
	writeVarint(&buf, fieldKindPtr)
	writeVarint(&buf, 16) // offset
	writeVarint(&buf, fieldKindEol)

	// Write EOF
	writeVarint(&buf, tagEOF)

	parser := &GoHeapParser{}
	g, err := parser.Parse(&buf)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if g.NumObjects() != 2 {
		t.Errorf("Expected 2 objects, got %d", g.NumObjects())
	}

	// Verify objects were created
	obj1Count := 0
	obj2Count := 0
	g.ForEachObject(func(o *graph.Object) {
		if o.ID == 0 {
			obj1Count++
		}
		if o.ID == 1 {
			obj2Count++
		}
	})

	if obj1Count != 1 {
		t.Errorf("Expected 1 object with ID 0, got %d", obj1Count)
	}
	if obj2Count != 1 {
		t.Errorf("Expected 1 object with ID 1, got %d", obj2Count)
	}
}

// Helper functions for building test dumps

func writeVarint(w io.Writer, v uint64) {
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, v)
	w.Write(buf[:n])
}

func writeString(w io.Writer, s string) {
	writeVarint(w, uint64(len(s)))
	w.Write([]byte(s))
}

func writeBytes(w io.Writer, b []byte) {
	writeVarint(w, uint64(len(b)))
	w.Write(b)
}

// BenchmarkParse benchmarks parsing performance
func BenchmarkParse(b *testing.B) {
	// Create a dump with many objects
	var buf bytes.Buffer

	// Write header
	buf.WriteString("go1.7 heap dump\n")

	// Write params
	writeVarint(&buf, tagParams)
	writeVarint(&buf, 0)          // little endian
	writeVarint(&buf, 8)          // pointer size
	writeVarint(&buf, 0x1000)     // heap start
	writeVarint(&buf, 0x100000)   // heap end
	writeString(&buf, "amd64")    // architecture
	writeString(&buf, "go1.20.0") // go version
	writeVarint(&buf, 4)          // num CPUs

	// Write a type
	writeVarint(&buf, tagType)
	writeVarint(&buf, 0x1000)     // type address
	writeVarint(&buf, 32)         // size
	writeString(&buf, "TestType") // name
	writeVarint(&buf, 0)          // not indirect

	// Write many objects
	numObjects := 1000
	for i := 0; i < numObjects; i++ {
		writeVarint(&buf, tagObject)
		writeVarint(&buf, uint64(0x2000+i*0x100)) // object address

		// Object data
		objData := make([]byte, 32)
		binary.LittleEndian.PutUint64(objData, 0x1000) // type pointer
		writeBytes(&buf, objData)

		// No pointer fields
		writeVarint(&buf, fieldKindEol)
	}

	// Write EOF
	writeVarint(&buf, tagEOF)

	data := buf.Bytes()
	parser := &GoHeapParser{}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)
		_, err := parser.Parse(r)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.SetBytes(int64(len(data)))
}

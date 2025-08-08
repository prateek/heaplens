// ABOUTME: Tests for the streaming parser API
// ABOUTME: Validates streaming callbacks, progress reporting, and error recovery

package goheap

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestStreamingParseBasic tests basic streaming parse functionality
func TestStreamingParseBasic(t *testing.T) {
	var buf bytes.Buffer

	// Build a test dump
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
	objData := make([]byte, 16)
	binary.LittleEndian.PutUint64(objData, 0x1000) // type pointer
	writeBytes(&buf, objData)
	writeVarint(&buf, fieldKindEol)

	// Write EOF
	writeVarint(&buf, tagEOF)

	// Track callbacks
	var paramsCalled, typeCalled, objectCalled bool

	callbacks := StreamCallbacks{
		OnParams: func(params DumpParams) error {
			paramsCalled = true
			if params.Arch != "amd64" {
				t.Errorf("Expected arch amd64, got %s", params.Arch)
			}
			if params.PointerSize != 8 {
				t.Errorf("Expected pointer size 8, got %d", params.PointerSize)
			}
			return nil
		},
		OnType: func(addr, size uint64, name string, indirect bool) error {
			typeCalled = true
			if name != "TestType" {
				t.Errorf("Expected type name TestType, got %s", name)
			}
			if size != 16 {
				t.Errorf("Expected type size 16, got %d", size)
			}
			return nil
		},
		OnObject: func(addr, typeAddr uint64, data []byte, ptrs []uint64) error {
			objectCalled = true
			if addr != 0x2000 {
				t.Errorf("Expected object addr 0x2000, got 0x%x", addr)
			}
			if typeAddr != 0x1000 {
				t.Errorf("Expected type addr 0x1000, got 0x%x", typeAddr)
			}
			if len(data) != 16 {
				t.Errorf("Expected data length 16, got %d", len(data))
			}
			return nil
		},
	}

	parser := NewStreamingParser(&buf, callbacks)
	err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if !paramsCalled {
		t.Error("OnParams callback not called")
	}
	if !typeCalled {
		t.Error("OnType callback not called")
	}
	if !objectCalled {
		t.Error("OnObject callback not called")
	}
}

// TestStreamingProgress tests progress reporting
func TestStreamingProgress(t *testing.T) {
	var buf bytes.Buffer

	// Build a larger dump
	buf.WriteString("go1.7 heap dump\n")

	// Write params
	writeVarint(&buf, tagParams)
	writeVarint(&buf, 0)
	writeVarint(&buf, 8)
	writeVarint(&buf, 0x1000)
	writeVarint(&buf, 0x10000)
	writeString(&buf, "amd64")
	writeString(&buf, "go1.20.0")
	writeVarint(&buf, 4)

	// Write many objects
	for i := 0; i < 100; i++ {
		writeVarint(&buf, tagObject)
		writeVarint(&buf, uint64(0x2000+i*0x100))
		objData := make([]byte, 32)
		writeBytes(&buf, objData)
		writeVarint(&buf, fieldKindEol)
	}

	writeVarint(&buf, tagEOF)

	// Track progress
	var progressCalls atomic.Int32
	var lastBytesRead int64
	var lastRecords int64

	callbacks := StreamCallbacks{
		OnProgress: func(bytesRead, records int64, elapsed time.Duration) {
			progressCalls.Add(1)
			if bytesRead <= lastBytesRead {
				t.Errorf("Progress not advancing: %d <= %d", bytesRead, lastBytesRead)
			}
			if records < lastRecords {
				t.Errorf("Record count went backwards: %d < %d", records, lastRecords)
			}
			lastBytesRead = bytesRead
			lastRecords = records
		},
	}

	parser := NewStreamingParser(&buf, callbacks)
	err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// The final progress update should always be called
	// so we should have at least one update 
	if progressCalls.Load() == 0 {
		t.Error("No progress callbacks received")
	}

	t.Logf("Received %d progress updates", progressCalls.Load())
}

// TestStreamingErrorRecovery tests error recovery mechanisms
func TestStreamingErrorRecovery(t *testing.T) {
	var buf bytes.Buffer

	// Build a dump with some corrupted data
	buf.WriteString("go1.7 heap dump\n")

	// Write params
	writeVarint(&buf, tagParams)
	writeVarint(&buf, 0)
	writeVarint(&buf, 8)
	writeVarint(&buf, 0x1000)
	writeVarint(&buf, 0x2000)
	writeString(&buf, "amd64")
	writeString(&buf, "go1.20.0")
	writeVarint(&buf, 4)

	// Write a valid object
	writeVarint(&buf, tagObject)
	writeVarint(&buf, 0x2000)
	objData := make([]byte, 16)
	writeBytes(&buf, objData)
	writeVarint(&buf, fieldKindEol)

	// Write invalid tag
	writeVarint(&buf, 99)

	// Write another valid object after the error
	writeVarint(&buf, tagObject)
	writeVarint(&buf, 0x3000)
	writeBytes(&buf, objData)
	writeVarint(&buf, fieldKindEol)

	writeVarint(&buf, tagEOF)

	// Track errors and objects
	errorCount := 0
	objectCount := 0

	callbacks := StreamCallbacks{
		OnObject: func(addr, typeAddr uint64, data []byte, ptrs []uint64) error {
			objectCount++
			return nil
		},
		OnError: func(err error, canRecover bool) error {
			errorCount++
			t.Logf("Error: %v (canRecover=%v)", err, canRecover)
			// Allow recovery
			return nil
		},
	}

	parser := NewStreamingParser(&buf, callbacks)
	parser.SetErrorRecovery(10, true) // Allow up to 10 errors

	err := parser.Parse()
	if err != nil {
		t.Logf("Parse completed with error: %v", err)
	}

	if errorCount == 0 {
		t.Error("Expected at least one error")
	}

	// Should have parsed at least the first object
	if objectCount == 0 {
		t.Error("Expected at least one object to be parsed")
	}

	t.Logf("Parsed %d objects with %d errors", objectCount, errorCount)
}

// TestStreamingWithPointers tests streaming parse of objects with pointers
func TestStreamingWithPointers(t *testing.T) {
	var buf bytes.Buffer

	// Build dump with linked objects
	buf.WriteString("go1.7 heap dump\n")

	// Write params
	writeVarint(&buf, tagParams)
	writeVarint(&buf, 0) // little endian
	writeVarint(&buf, 8) // pointer size
	writeVarint(&buf, 0x1000)
	writeVarint(&buf, 0x4000)
	writeString(&buf, "amd64")
	writeString(&buf, "go1.20.0")
	writeVarint(&buf, 4)

	// Write object 1 pointing to object 2
	writeVarint(&buf, tagObject)
	writeVarint(&buf, 0x2000) // object 1 address
	obj1Data := make([]byte, 24)
	binary.LittleEndian.PutUint64(obj1Data[0:], 0x1000)  // type pointer
	binary.LittleEndian.PutUint64(obj1Data[8:], 42)      // value
	binary.LittleEndian.PutUint64(obj1Data[16:], 0x3000) // pointer to object 2
	writeBytes(&buf, obj1Data)
	writeVarint(&buf, fieldKindPtr)
	writeVarint(&buf, 16) // pointer at offset 16
	writeVarint(&buf, fieldKindEol)

	// Write object 2
	writeVarint(&buf, tagObject)
	writeVarint(&buf, 0x3000) // object 2 address
	obj2Data := make([]byte, 24)
	binary.LittleEndian.PutUint64(obj2Data[0:], 0x1000) // type pointer
	writeBytes(&buf, obj2Data)
	writeVarint(&buf, fieldKindEol)

	// Write root pointing to object 1
	writeVarint(&buf, tagOtherRoot)
	writeString(&buf, "test root")
	writeVarint(&buf, 0x2000)

	writeVarint(&buf, tagEOF)

	// Track objects and their relationships
	objects := make(map[uint64][]uint64) // addr -> pointers
	roots := []uint64{}

	callbacks := StreamCallbacks{
		OnObject: func(addr, typeAddr uint64, data []byte, ptrs []uint64) error {
			objects[addr] = ptrs
			return nil
		},
		OnRoot: func(desc string, ptr uint64) error {
			roots = append(roots, ptr)
			return nil
		},
	}

	parser := NewStreamingParser(&buf, callbacks)
	err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Verify objects
	if len(objects) != 2 {
		t.Errorf("Expected 2 objects, got %d", len(objects))
	}

	// Verify object 1 points to object 2
	if ptrs, ok := objects[0x2000]; ok {
		if len(ptrs) != 1 || ptrs[0] != 0x3000 {
			t.Errorf("Object 1 should point to 0x3000, got %v", ptrs)
		}
	} else {
		t.Error("Object 1 not found")
	}

	// Verify object 2 has no pointers
	if ptrs, ok := objects[0x3000]; ok {
		if len(ptrs) != 0 {
			t.Errorf("Object 2 should have no pointers, got %v", ptrs)
		}
	} else {
		t.Error("Object 2 not found")
	}

	// Verify root
	if len(roots) != 1 || roots[0] != 0x2000 {
		t.Errorf("Expected root pointing to 0x2000, got %v", roots)
	}
}

// TestStreamingCallbackError tests handling of callback errors
func TestStreamingCallbackError(t *testing.T) {
	var buf bytes.Buffer

	// Build simple dump
	buf.WriteString("go1.7 heap dump\n")

	writeVarint(&buf, tagParams)
	writeVarint(&buf, 0)
	writeVarint(&buf, 8)
	writeVarint(&buf, 0x1000)
	writeVarint(&buf, 0x2000)
	writeString(&buf, "amd64")
	writeString(&buf, "go1.20.0")
	writeVarint(&buf, 4)

	writeVarint(&buf, tagType)
	writeVarint(&buf, 0x1000)
	writeVarint(&buf, 16)
	writeString(&buf, "TestType")
	writeVarint(&buf, 0)

	writeVarint(&buf, tagEOF)

	// Callback that returns error
	callbacks := StreamCallbacks{
		OnType: func(addr, size uint64, name string, indirect bool) error {
			return errors.New("callback error")
		},
	}

	parser := NewStreamingParser(&buf, callbacks)
	err := parser.Parse()

	if err == nil {
		t.Error("Expected error from callback")
	} else if err.Error() != "parsing type: callback error" {
		t.Errorf("Unexpected error: %v", err)
	}
}

// BenchmarkStreamingParse benchmarks streaming parse performance
func BenchmarkStreamingParse(b *testing.B) {
	// Create a large dump
	var buf bytes.Buffer

	buf.WriteString("go1.7 heap dump\n")

	writeVarint(&buf, tagParams)
	writeVarint(&buf, 0)
	writeVarint(&buf, 8)
	writeVarint(&buf, 0x1000)
	writeVarint(&buf, 0x100000)
	writeString(&buf, "amd64")
	writeString(&buf, "go1.20.0")
	writeVarint(&buf, 4)

	// Write many objects
	numObjects := 10000
	for i := 0; i < numObjects; i++ {
		writeVarint(&buf, tagObject)
		writeVarint(&buf, uint64(0x2000+i*0x100))
		objData := make([]byte, 64)
		binary.LittleEndian.PutUint64(objData, 0x1000)
		writeBytes(&buf, objData)

		// Add some pointer fields
		if i > 0 && i%2 == 0 {
			writeVarint(&buf, fieldKindPtr)
			writeVarint(&buf, 8)
			writeVarint(&buf, fieldKindPtr)
			writeVarint(&buf, 16)
		}
		writeVarint(&buf, fieldKindEol)
	}

	writeVarint(&buf, tagEOF)

	data := buf.Bytes()

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)

		objectCount := 0
		callbacks := StreamCallbacks{
			OnObject: func(addr, typeAddr uint64, data []byte, ptrs []uint64) error {
				objectCount++
				return nil
			},
		}

		parser := NewStreamingParser(r, callbacks)
		err := parser.Parse()
		if err != nil {
			b.Fatal(err)
		}

		if objectCount != numObjects {
			b.Fatalf("Expected %d objects, got %d", numObjects, objectCount)
		}
	}
}

// TestStreamingLargeStrings tests handling of large strings
func TestStreamingLargeStrings(t *testing.T) {
	var buf bytes.Buffer

	buf.WriteString("go1.7 heap dump\n")

	writeVarint(&buf, tagParams)
	writeVarint(&buf, 0)
	writeVarint(&buf, 8)
	writeVarint(&buf, 0x1000)
	writeVarint(&buf, 0x2000)

	// Write an oversized string for arch field
	writeVarint(&buf, 1<<21) // 2MB, over the 1MB limit
	// Don't write the actual string data - readString will fail on the size check

	callbacks := StreamCallbacks{}
	parser := NewStreamingParser(&buf, callbacks)
	err := parser.Parse()

	if err == nil {
		t.Error("Expected error for oversized string")
	} else if !strings.Contains(err.Error(), "string too long") {
		t.Errorf("Unexpected error: %v", err)
	} else {
		t.Logf("Got expected error: %v", err)
	}
}

// TestStreamingGoroutines tests parsing of goroutine records
func TestStreamingGoroutines(t *testing.T) {
	var buf bytes.Buffer

	buf.WriteString("go1.7 heap dump\n")

	writeVarint(&buf, tagParams)
	writeVarint(&buf, 0)
	writeVarint(&buf, 8)
	writeVarint(&buf, 0x1000)
	writeVarint(&buf, 0x2000)
	writeString(&buf, "amd64")
	writeString(&buf, "go1.20.0")
	writeVarint(&buf, 4)

	// Write a goroutine record
	writeVarint(&buf, tagGoroutine)
	writeVarint(&buf, 0x5000) // address
	writeVarint(&buf, 0x5100) // stack pointer
	writeVarint(&buf, 1)      // ID
	writeVarint(&buf, 2)      // status (running)
	writeVarint(&buf, 0)      // not system
	writeVarint(&buf, 0)      // not background
	writeVarint(&buf, 0)      // wait since
	writeString(&buf, "")     // wait reason
	// Skip fields
	for i := 0; i < 4; i++ {
		writeVarint(&buf, 0)
	}

	writeVarint(&buf, tagEOF)

	goroutineCount := 0
	callbacks := StreamCallbacks{
		OnGoroutine: func(id, status uint64, waitReason string) error {
			goroutineCount++
			if id != 1 {
				t.Errorf("Expected goroutine ID 1, got %d", id)
			}
			if status != 2 {
				t.Errorf("Expected status 2, got %d", status)
			}
			return nil
		},
	}

	parser := NewStreamingParser(&buf, callbacks)
	err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if goroutineCount != 1 {
		t.Errorf("Expected 1 goroutine, got %d", goroutineCount)
	}
}

// TestStreamingParsePerformance validates performance characteristics
func TestStreamingParsePerformance(t *testing.T) {
	// Create a 10MB dump
	var buf bytes.Buffer

	buf.WriteString("go1.7 heap dump\n")

	writeVarint(&buf, tagParams)
	writeVarint(&buf, 0)
	writeVarint(&buf, 8)
	writeVarint(&buf, 0x1000)
	writeVarint(&buf, 0x1000000)
	writeString(&buf, "amd64")
	writeString(&buf, "go1.20.0")
	writeVarint(&buf, 8)

	// Write objects until we reach approximately 10MB
	targetSize := 10 * 1024 * 1024
	objectSize := 1024 // 1KB per object
	numObjects := 0

	for buf.Len() < targetSize {
		writeVarint(&buf, tagObject)
		writeVarint(&buf, uint64(0x2000+numObjects*0x1000))
		objData := make([]byte, objectSize)
		writeBytes(&buf, objData)
		writeVarint(&buf, fieldKindEol)
		numObjects++
	}

	writeVarint(&buf, tagEOF)

	data := buf.Bytes()
	t.Logf("Created dump with %d objects, size: %d bytes", numObjects, len(data))

	// Parse and measure
	start := time.Now()

	objectCount := 0
	callbacks := StreamCallbacks{
		OnObject: func(addr, typeAddr uint64, data []byte, ptrs []uint64) error {
			objectCount++
			return nil
		},
		OnProgress: func(bytesRead, records int64, elapsed time.Duration) {
			// Log progress periodically
			if records%1000 == 0 {
				rate := float64(bytesRead) / elapsed.Seconds() / 1024 / 1024
				t.Logf("Progress: %d records, %.2f MB/s", records, rate)
			}
		},
	}

	parser := NewStreamingParser(bytes.NewReader(data), callbacks)
	err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	elapsed := time.Since(start)
	rate := float64(len(data)) / elapsed.Seconds() / 1024 / 1024

	t.Logf("Parsed %d objects in %v (%.2f MB/s)", objectCount, elapsed, rate)

	// Performance assertions
	if rate < 10 {
		t.Errorf("Parse rate too slow: %.2f MB/s (expected > 10 MB/s)", rate)
	}

	if objectCount != numObjects {
		t.Errorf("Expected %d objects, got %d", numObjects, objectCount)
	}
}

// ABOUTME: Fuzz tests for the Go heap dump parser
// ABOUTME: Uses Go 1.18+ native fuzzing to test parser robustness

//go:build go1.18
// +build go1.18

package goheap

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/prateek/heaplens/graph"
)

// FuzzParser tests the parser with random inputs
func FuzzParser(f *testing.F) {
	// Add seed corpus with valid dumps
	f.Add(createValidDumpSeed())
	f.Add(createMinimalDumpSeed())
	f.Add(createComplexDumpSeed())

	// Also add some corrupted seeds
	f.Add(createCorruptedHeaderSeed())
	f.Add(createTruncatedDumpSeed())

	f.Fuzz(func(t *testing.T, data []byte) {
		// Skip if data is too small to be a valid dump
		if len(data) < 16 {
			t.Skip()
			return
		}

		parser := &GoHeapParser{}
		r := bytes.NewReader(data)

		// Parser should not panic on any input
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Parser panicked: %v", r)
				}
			}()

			// Try to parse
			g, err := parser.Parse(r)

			// If parsing succeeded, validate the result
			if err == nil && g != nil {
				// Basic sanity checks
				numObjects := g.NumObjects()
				if numObjects < 0 {
					t.Errorf("Negative object count: %d", numObjects)
				}

				// Ensure we can iterate without panic
				g.ForEachObject(func(obj *graph.Object) {
					if obj == nil {
						t.Error("Nil object in graph")
					}
				})

				// Check roots
				roots := g.GetRoots()
				if roots.IDs == nil {
					t.Error("Nil roots IDs")
				}
			}
		}()
	})
}

// FuzzStreamingParser tests the streaming parser with random inputs
func FuzzStreamingParser(f *testing.F) {
	// Add seed corpus
	f.Add(createValidDumpSeed())
	f.Add(createStreamingDumpSeed())

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 16 {
			t.Skip()
			return
		}

		r := bytes.NewReader(data)

		// Track what we parsed
		var objectCount, typeCount, rootCount int
		errorCount := 0

		callbacks := StreamCallbacks{
			OnObject: func(addr, typeAddr uint64, data []byte, ptrs []uint64) error {
				objectCount++
				// Validate data
				if len(data) > 1<<30 { // 1GB max object size
					t.Errorf("Object too large: %d bytes", len(data))
				}
				return nil
			},
			OnType: func(addr, size uint64, name string, indirect bool) error {
				typeCount++
				// Validate type
				if len(name) > 1<<20 { // 1MB max type name
					t.Errorf("Type name too long: %d bytes", len(name))
				}
				return nil
			},
			OnRoot: func(desc string, ptr uint64) error {
				rootCount++
				return nil
			},
			OnError: func(err error, canRecover bool) error {
				errorCount++
				if errorCount > 1000 {
					return err // Stop after too many errors
				}
				return nil // Continue parsing
			},
		}

		parser := NewStreamingParser(r, callbacks)
		parser.SetErrorRecovery(1000, true)

		// Should not panic
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Streaming parser panicked: %v", r)
				}
			}()

			err := parser.Parse()
			_ = err // Error is OK for fuzz input

			// Validate counts
			if objectCount < 0 || typeCount < 0 || rootCount < 0 {
				t.Error("Negative counts detected")
			}
		}()
	})
}

// FuzzVarint tests varint decoding with random data
func FuzzVarint(f *testing.F) {
	// Add valid varint seeds
	f.Add([]byte{0x00})                   // 0
	f.Add([]byte{0x01})                   // 1
	f.Add([]byte{0x7f})                   // 127
	f.Add([]byte{0x80, 0x01})             // 128
	f.Add([]byte{0xff, 0xff, 0xff, 0x7f}) // large number

	// Add invalid/corrupted varints
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})

	f.Fuzz(func(t *testing.T, data []byte) {
		r := bytes.NewReader(data)

		// Should not panic
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Varint decoding panicked: %v", r)
				}
			}()

			v, err := binary.ReadUvarint(r)

			// If successful, validate the result
			if err == nil {
				// Re-encode and verify round-trip
				buf := make([]byte, binary.MaxVarintLen64)
				n := binary.PutUvarint(buf, v)

				if n > len(data) {
					// OK - varint might have been truncated
				} else if !bytes.Equal(buf[:n], data[:n]) {
					t.Errorf("Varint round-trip failed: %x != %x", buf[:n], data[:n])
				}
			}
		}()
	})
}

// Helper functions to create seed dumps

func createValidDumpSeed() []byte {
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

	writeVarint(&buf, tagType)
	writeVarint(&buf, 0x1000)
	writeVarint(&buf, 16)
	writeString(&buf, "TestType")
	writeVarint(&buf, 0)

	writeVarint(&buf, tagObject)
	writeVarint(&buf, 0x2000)
	objData := make([]byte, 16)
	binary.LittleEndian.PutUint64(objData, 0x1000)
	writeBytes(&buf, objData)
	writeVarint(&buf, fieldKindEol)

	writeVarint(&buf, tagEOF)

	return buf.Bytes()
}

func createMinimalDumpSeed() []byte {
	var buf bytes.Buffer
	buf.WriteString("go1.7 heap dump\n")
	writeVarint(&buf, tagEOF)
	return buf.Bytes()
}

func createComplexDumpSeed() []byte {
	var buf bytes.Buffer

	buf.WriteString("go1.7 heap dump\n")

	// Params
	writeVarint(&buf, tagParams)
	writeVarint(&buf, 0)
	writeVarint(&buf, 8)
	writeVarint(&buf, 0x1000)
	writeVarint(&buf, 0x10000)
	writeString(&buf, "amd64")
	writeString(&buf, "go1.20.0")
	writeVarint(&buf, 8)

	// Multiple types
	for i := 0; i < 10; i++ {
		writeVarint(&buf, tagType)
		writeVarint(&buf, uint64(0x1000+i*0x100))
		writeVarint(&buf, uint64(16+i*8))
		writeString(&buf, "Type"+string(rune('A'+i)))
		writeVarint(&buf, 0)
	}

	// Objects with pointers
	for i := 0; i < 20; i++ {
		writeVarint(&buf, tagObject)
		writeVarint(&buf, uint64(0x2000+i*0x100))

		objData := make([]byte, 32)
		binary.LittleEndian.PutUint64(objData, 0x1000)
		if i < 19 {
			binary.LittleEndian.PutUint64(objData[16:], uint64(0x2000+(i+1)*0x100))
		}
		writeBytes(&buf, objData)

		if i < 19 {
			writeVarint(&buf, fieldKindPtr)
			writeVarint(&buf, 16)
		}
		writeVarint(&buf, fieldKindEol)
	}

	// Roots
	for i := 0; i < 5; i++ {
		writeVarint(&buf, tagOtherRoot)
		writeString(&buf, "root"+string(rune('0'+i)))
		writeVarint(&buf, uint64(0x2000+i*0x100))
	}

	// Goroutines
	writeVarint(&buf, tagGoroutine)
	for i := 0; i < 12; i++ {
		writeVarint(&buf, uint64(i))
	}
	writeString(&buf, "waiting")

	// Stack frames
	writeVarint(&buf, tagStackFrame)
	writeVarint(&buf, 0x8000)
	writeVarint(&buf, 1)
	writeVarint(&buf, 0x8100)
	writeBytes(&buf, make([]byte, 64))
	writeVarint(&buf, 0x400000)
	writeVarint(&buf, 0x400100)
	writeVarint(&buf, 0x400200)
	writeString(&buf, "main.main")
	writeVarint(&buf, fieldKindEol)

	// MemStats
	writeVarint(&buf, tagMemStats)
	for i := 0; i < 61; i++ {
		writeVarint(&buf, uint64(i*1000))
	}

	writeVarint(&buf, tagEOF)

	return buf.Bytes()
}

func createCorruptedHeaderSeed() []byte {
	return []byte("corrupted dump\n\x00")
}

func createTruncatedDumpSeed() []byte {
	var buf bytes.Buffer
	buf.WriteString("go1.7 heap dump\n")
	writeVarint(&buf, tagParams)
	// Truncated - missing params data
	return buf.Bytes()
}

func createStreamingDumpSeed() []byte {
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

	// Large number of small objects for streaming
	for i := 0; i < 1000; i++ {
		writeVarint(&buf, tagObject)
		writeVarint(&buf, uint64(0x2000+i*0x100))
		writeBytes(&buf, make([]byte, 32))
		writeVarint(&buf, fieldKindEol)
	}

	writeVarint(&buf, tagEOF)

	return buf.Bytes()
}

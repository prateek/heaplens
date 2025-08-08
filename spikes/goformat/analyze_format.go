// ABOUTME: Spike to understand Go heap dump binary format
// ABOUTME: Parses and documents the format structure

package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// Based on runtime/heapdump.go format
const (
	tagEOF          = 0
	tagObject       = 1
	tagOtherRoot    = 2
	tagType         = 3
	tagGoroutine    = 4
	tagStackFrame   = 5
	tagParams       = 6
	tagFinalizer    = 7
	tagItab         = 8
	tagOSThread     = 9
	tagMemStats     = 10
	tagQueuedFinalizer = 11
	tagData         = 12
	tagBSS          = 13
	tagDefer        = 14
	tagPanic        = 15
	tagMemProf      = 16
	tagAllocSample  = 17
)

func main() {
	f, err := os.Open("test.heapdump")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	
	// Read header
	header := make([]byte, 16)
	n, err := f.Read(header)
	if err != nil || n != 16 {
		panic("Failed to read header")
	}
	
	fmt.Printf("Header: %s\n", string(header[:16]))
	fmt.Printf("Format version: %s\n", string(header[:4])) // "go1."
	
	// The format uses a variable-length encoding (uvarint)
	// Let's try to parse the records
	
	buf := make([]byte, 1024*1024) // 1MB buffer
	n, err = f.Read(buf)
	if err != nil && err != io.EOF {
		panic(err)
	}
	
	fmt.Printf("\nRead %d bytes\n", n)
	
	// Skip header and start parsing records
	offset := 0
	recordCount := 0
	
	// Parse records
	for offset < n && recordCount < 20 { // Limit to first 20 records
		if offset >= len(buf)-10 {
			break
		}
		
		// Read record tag
		tag := buf[offset]
		offset++
		
		switch tag {
		case tagEOF:
			fmt.Printf("Record %d: EOF\n", recordCount)
			return
			
		case tagObject:
			fmt.Printf("Record %d: Object", recordCount)
			// Object format: address, type, kind, content
			addr, adv := binary.Uvarint(buf[offset:])
			offset += adv
			fmt.Printf(" addr=0x%x", addr)
			
		case tagType:
			fmt.Printf("Record %d: Type", recordCount)
			// Type definition
			
		case tagOtherRoot:
			fmt.Printf("Record %d: OtherRoot", recordCount)
			
		case tagGoroutine:
			fmt.Printf("Record %d: Goroutine", recordCount)
			
		case tagParams:
			fmt.Printf("Record %d: Params", recordCount)
			// This contains runtime parameters
			
		case tagMemStats:
			fmt.Printf("Record %d: MemStats", recordCount)
			
		default:
			fmt.Printf("Record %d: Unknown tag %d (0x%x)", recordCount, tag, tag)
		}
		
		fmt.Println()
		
		// For now, skip to find next valid tag
		// In real parser, we'd need to properly parse each record type
		if tag > tagAllocSample {
			// Skip bytes until we find a valid tag
			for offset < n-1 && buf[offset] > tagAllocSample {
				offset++
			}
		}
		
		recordCount++
	}
	
	fmt.Printf("\nFormat analysis:\n")
	fmt.Println("- Starts with 'go1.7 heap dump' header")
	fmt.Println("- Uses tag-based record format")
	fmt.Println("- Uses variable-length integer encoding (uvarint)")
	fmt.Println("- Contains objects, types, roots, goroutines, etc.")
	fmt.Println("\nThis format is complex and would require significant effort to parse fully.")
	fmt.Println("Recommendation: Use runtime/pprof heap profile format instead (protobuf-based)")
}
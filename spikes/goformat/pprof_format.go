// ABOUTME: Test using runtime/pprof heap profile format instead
// ABOUTME: This format is better documented and easier to parse

package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
)

type TestStruct struct {
	Name   string
	Value  int
	Next   *TestStruct
	Slice  []int
	Map    map[string]int
}

func main() {
	// Create some known objects
	root := &TestStruct{
		Name:  "root",
		Value: 42,
		Slice: make([]int, 1000), // Large slice to show in profile
		Map:   map[string]int{"key1": 100, "key2": 200},
	}
	
	child := &TestStruct{
		Name:  "child",
		Value: 99,
		Slice: make([]int, 2000), // Larger slice
	}
	
	root.Next = child
	
	// Force GC to update heap profile
	runtime.GC()
	
	// Write heap profile (pprof format)
	f, err := os.Create("heap.pprof")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	
	// Write heap profile
	if err := pprof.WriteHeapProfile(f); err != nil {
		panic(err)
	}
	
	fmt.Println("Heap profile written to heap.pprof")
	fmt.Println("This is in protobuf format and can be analyzed with 'go tool pprof'")
	
	// Also get heap stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("\nMemory stats:\n")
	fmt.Printf("Alloc: %d bytes\n", m.Alloc)
	fmt.Printf("TotalAlloc: %d bytes\n", m.TotalAlloc)
	fmt.Printf("HeapAlloc: %d bytes\n", m.HeapAlloc)
	fmt.Printf("HeapObjects: %d\n", m.HeapObjects)
}
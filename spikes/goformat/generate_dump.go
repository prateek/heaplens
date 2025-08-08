// ABOUTME: Test program to generate a Go heap dump for format analysis
// ABOUTME: Creates known objects and dumps heap for reverse engineering

package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
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
		Slice: []int{1, 2, 3, 4, 5},
		Map:   map[string]int{"key1": 100, "key2": 200},
	}
	
	child := &TestStruct{
		Name:  "child",
		Value: 99,
		Slice: []int{10, 20, 30},
	}
	
	root.Next = child
	
	// Create a cycle
	child.Next = root
	
	// Force GC to clean up any garbage
	runtime.GC()
	runtime.GC()
	
	// Write heap dump
	f, err := os.Create("test.heapdump")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	
	debug.WriteHeapDump(f.Fd())
	
	fmt.Println("Heap dump written to test.heapdump")
	fmt.Printf("Root object: %p\n", root)
	fmt.Printf("Child object: %p\n", child)
}
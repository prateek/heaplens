// ABOUTME: Basic Go heap dump parser implementation based on format analysis
// ABOUTME: Parses binary heap dumps from debug.WriteHeapDump()

package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// Record type constants from runtime/heapdump.go
const (
	tagEOF             = 0
	tagObject          = 1
	tagOtherRoot       = 2
	tagType            = 3
	tagGoroutine       = 4
	tagStackFrame      = 5
	tagParams          = 6
	tagFinalizer       = 7
	tagItab            = 8
	tagOSThread        = 9
	tagMemStats        = 10
	tagQueuedFinalizer = 11
	tagData            = 12
	tagBSS             = 13
	tagDefer           = 14
	tagPanic           = 15
	tagMemProf         = 16
	tagAllocSample     = 17
)

// Field kinds
const (
	fieldKindEol   = 0
	fieldKindPtr   = 1
	fieldKindIface = 2
	fieldKindEface = 3
)

// HeapDump represents a parsed heap dump
type HeapDump struct {
	// Parameters
	BigEndian    bool
	PointerSize  uint64
	HeapStart    uint64
	HeapEnd      uint64
	Architecture string
	GoVersion    string
	NumCPUs      uint64
	
	// Data
	Types        map[uint64]*Type
	Objects      map[uint64]*Object
	Roots        []*Root
	Goroutines   []*Goroutine
	StackFrames  []*StackFrame
	MemStats     *MemStats
}

// Type represents a Go type in the heap
type Type struct {
	Address  uint64
	Size     uint64
	Name     string
	Indirect bool
}

// Object represents a heap object
type Object struct {
	Address uint64
	Size    uint64
	TypeAddr uint64
	Data    []byte
	Fields  []Field
}

// Field represents a pointer field in an object
type Field struct {
	Kind   uint64
	Offset uint64
}

// Root represents a GC root
type Root struct {
	Description string
	Pointer     uint64
}

// Goroutine represents a goroutine
type Goroutine struct {
	Address        uint64
	StackPointer   uint64
	ID             uint64
	Status         uint64
	IsSystem       bool
	IsBackground   bool
	WaitSince      uint64
	WaitReason     string
}

// StackFrame represents a stack frame
type StackFrame struct {
	SP       uint64
	Depth    uint64
	ChildSP  uint64
	Data     []byte
	EntryPC  uint64
	PC       uint64
	ContPC   uint64
	Name     string
}

// MemStats represents memory statistics
type MemStats struct {
	Alloc      uint64
	TotalAlloc uint64
	Sys        uint64
	NLookup    uint64
	NMalloc    uint64
	NFree      uint64
	HeapAlloc  uint64
	HeapSys    uint64
}

// Parser for heap dumps
type Parser struct {
	r    *bufio.Reader
	dump *HeapDump
}

// NewParser creates a new heap dump parser
func NewParser(r io.Reader) *Parser {
	return &Parser{
		r: bufio.NewReader(r),
		dump: &HeapDump{
			Types:   make(map[uint64]*Type),
			Objects: make(map[uint64]*Object),
		},
	}
}

// Parse parses a heap dump
func (p *Parser) Parse() (*HeapDump, error) {
	// Read and verify header
	header := make([]byte, 16)
	if _, err := io.ReadFull(p.r, header); err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}
	if string(header) != "go1.7 heap dump\n" {
		return nil, fmt.Errorf("invalid header: %s", header)
	}
	
	// Read records
	for {
		tag, err := p.readVarint()
		if err != nil {
			return nil, fmt.Errorf("reading tag: %w", err)
		}
		
		switch tag {
		case tagEOF:
			return p.dump, nil
			
		case tagParams:
			if err := p.parseParams(); err != nil {
				return nil, fmt.Errorf("parsing params: %w", err)
			}
			
		case tagType:
			if err := p.parseType(); err != nil {
				return nil, fmt.Errorf("parsing type: %w", err)
			}
			
		case tagObject:
			if err := p.parseObject(); err != nil {
				return nil, fmt.Errorf("parsing object: %w", err)
			}
			
		case tagOtherRoot:
			if err := p.parseOtherRoot(); err != nil {
				return nil, fmt.Errorf("parsing root: %w", err)
			}
			
		case tagGoroutine:
			if err := p.parseGoroutine(); err != nil {
				return nil, fmt.Errorf("parsing goroutine: %w", err)
			}
			
		case tagStackFrame:
			if err := p.parseStackFrame(); err != nil {
				return nil, fmt.Errorf("parsing stack frame: %w", err)
			}
			
		case tagMemStats:
			if err := p.parseMemStats(); err != nil {
				return nil, fmt.Errorf("parsing memstats: %w", err)
			}
			
		case tagItab:
			// Skip interface table for now
			if err := p.skipItab(); err != nil {
				return nil, fmt.Errorf("skipping itab: %w", err)
			}
			
		case tagFinalizer, tagQueuedFinalizer:
			// Skip finalizers for now
			if err := p.skipFinalizer(); err != nil {
				return nil, fmt.Errorf("skipping finalizer: %w", err)
			}
			
		case tagData, tagBSS:
			// Skip data/BSS segments for now
			if err := p.skipDataSegment(); err != nil {
				return nil, fmt.Errorf("skipping data segment: %w", err)
			}
			
		case tagDefer, tagPanic:
			// Skip defer/panic records for now
			if err := p.skipDeferPanic(); err != nil {
				return nil, fmt.Errorf("skipping defer/panic: %w", err)
			}
			
		case tagOSThread:
			// Skip OS thread for now
			if err := p.skipOSThread(); err != nil {
				return nil, fmt.Errorf("skipping OS thread: %w", err)
			}
			
		case tagMemProf, tagAllocSample:
			// Skip memory profiling for now
			if err := p.skipMemProf(); err != nil {
				return nil, fmt.Errorf("skipping mem prof: %w", err)
			}
			
		default:
			// Unknown tag
			fmt.Printf("Warning: unknown tag %d (0x%x, char='%c')\n", tag, tag, rune(tag))
			// Try to see if we can read ahead a bit for debugging
			peek, _ := p.r.Peek(20)
			if len(peek) > 0 {
				fmt.Printf("Next bytes: %x\n", peek)
			}
			return p.dump, fmt.Errorf("unknown tag: %d", tag)
		}
	}
}

// readVarint reads a variable-length integer
func (p *Parser) readVarint() (uint64, error) {
	return binary.ReadUvarint(p.r)
}

// readString reads a length-prefixed string
func (p *Parser) readString() (string, error) {
	length, err := p.readVarint()
	if err != nil {
		return "", err
	}
	if length > 1<<20 { // Sanity check: 1MB max string
		return "", fmt.Errorf("string too long: %d", length)
	}
	
	data := make([]byte, length)
	if _, err := io.ReadFull(p.r, data); err != nil {
		return "", err
	}
	return string(data), nil
}

// readBytes reads a length-prefixed byte slice
func (p *Parser) readBytes() ([]byte, error) {
	length, err := p.readVarint()
	if err != nil {
		return nil, err
	}
	if length > 1<<30 { // Sanity check: 1GB max
		return nil, fmt.Errorf("byte slice too long: %d", length)
	}
	
	data := make([]byte, length)
	if _, err := io.ReadFull(p.r, data); err != nil {
		return nil, err
	}
	return data, nil
}

// parseParams parses a parameters record
func (p *Parser) parseParams() error {
	bigEndian, err := p.readVarint()
	if err != nil {
		return err
	}
	p.dump.BigEndian = bigEndian != 0
	
	p.dump.PointerSize, err = p.readVarint()
	if err != nil {
		return err
	}
	
	p.dump.HeapStart, err = p.readVarint()
	if err != nil {
		return err
	}
	
	p.dump.HeapEnd, err = p.readVarint()
	if err != nil {
		return err
	}
	
	p.dump.Architecture, err = p.readString()
	if err != nil {
		return err
	}
	
	p.dump.GoVersion, err = p.readString()
	if err != nil {
		return err
	}
	
	p.dump.NumCPUs, err = p.readVarint()
	if err != nil {
		return err
	}
	
	return nil
}

// parseType parses a type record
func (p *Parser) parseType() error {
	addr, err := p.readVarint()
	if err != nil {
		return err
	}
	
	size, err := p.readVarint()
	if err != nil {
		return err
	}
	
	name, err := p.readString()
	if err != nil {
		return err
	}
	
	indirect, err := p.readVarint()
	if err != nil {
		return err
	}
	
	p.dump.Types[addr] = &Type{
		Address:  addr,
		Size:     size,
		Name:     name,
		Indirect: indirect != 0,
	}
	
	return nil
}

// parseObject parses an object record
func (p *Parser) parseObject() error {
	addr, err := p.readVarint()
	if err != nil {
		return err
	}
	
	data, err := p.readBytes()
	if err != nil {
		return err
	}
	
	// Parse fields
	var fields []Field
	for {
		kind, err := p.readVarint()
		if err != nil {
			return err
		}
		if kind == fieldKindEol {
			break
		}
		
		offset, err := p.readVarint()
		if err != nil {
			return err
		}
		
		fields = append(fields, Field{
			Kind:   kind,
			Offset: offset,
		})
	}
	
	p.dump.Objects[addr] = &Object{
		Address: addr,
		Size:    uint64(len(data)),
		Data:    data,
		Fields:  fields,
	}
	
	return nil
}

// parseOtherRoot parses a root record
func (p *Parser) parseOtherRoot() error {
	desc, err := p.readString()
	if err != nil {
		return err
	}
	
	ptr, err := p.readVarint()
	if err != nil {
		return err
	}
	
	p.dump.Roots = append(p.dump.Roots, &Root{
		Description: desc,
		Pointer:     ptr,
	})
	
	return nil
}

// parseGoroutine parses a goroutine record
func (p *Parser) parseGoroutine() error {
	g := &Goroutine{}
	var err error
	
	g.Address, err = p.readVarint()
	if err != nil {
		return err
	}
	
	g.StackPointer, err = p.readVarint()
	if err != nil {
		return err
	}
	
	g.ID, err = p.readVarint()
	if err != nil {
		return err
	}
	
	g.Status, err = p.readVarint()
	if err != nil {
		return err
	}
	
	isSystem, err := p.readVarint()
	if err != nil {
		return err
	}
	g.IsSystem = isSystem != 0
	
	isBackground, err := p.readVarint()
	if err != nil {
		return err
	}
	g.IsBackground = isBackground != 0
	
	g.WaitSince, err = p.readVarint()
	if err != nil {
		return err
	}
	
	g.WaitReason, err = p.readString()
	if err != nil {
		return err
	}
	
	// Skip remaining fields for now
	for i := 0; i < 4; i++ {
		if _, err := p.readVarint(); err != nil {
			return err
		}
	}
	
	p.dump.Goroutines = append(p.dump.Goroutines, g)
	return nil
}

// parseStackFrame parses a stack frame record
func (p *Parser) parseStackFrame() error {
	sf := &StackFrame{}
	var err error
	
	sf.SP, err = p.readVarint()
	if err != nil {
		return err
	}
	
	sf.Depth, err = p.readVarint()
	if err != nil {
		return err
	}
	
	sf.ChildSP, err = p.readVarint()
	if err != nil {
		return err
	}
	
	sf.Data, err = p.readBytes()
	if err != nil {
		return err
	}
	
	sf.EntryPC, err = p.readVarint()
	if err != nil {
		return err
	}
	
	sf.PC, err = p.readVarint()
	if err != nil {
		return err
	}
	
	sf.ContPC, err = p.readVarint()
	if err != nil {
		return err
	}
	
	sf.Name, err = p.readString()
	if err != nil {
		return err
	}
	
	// Parse fields (simplified - skip for now)
	for {
		kind, err := p.readVarint()
		if err != nil {
			return err
		}
		if kind == fieldKindEol {
			break
		}
		// Skip offset
		if _, err := p.readVarint(); err != nil {
			return err
		}
	}
	
	p.dump.StackFrames = append(p.dump.StackFrames, sf)
	return nil
}

// parseMemStats parses memory statistics
func (p *Parser) parseMemStats() error {
	ms := &MemStats{}
	var err error
	
	ms.Alloc, err = p.readVarint()
	if err != nil {
		return err
	}
	
	ms.TotalAlloc, err = p.readVarint()
	if err != nil {
		return err
	}
	
	ms.Sys, err = p.readVarint()
	if err != nil {
		return err
	}
	
	ms.NLookup, err = p.readVarint()
	if err != nil {
		return err
	}
	
	ms.NMalloc, err = p.readVarint()
	if err != nil {
		return err
	}
	
	ms.NFree, err = p.readVarint()
	if err != nil {
		return err
	}
	
	ms.HeapAlloc, err = p.readVarint()
	if err != nil {
		return err
	}
	
	ms.HeapSys, err = p.readVarint()
	if err != nil {
		return err
	}
	
	// Skip remaining fields for now (there are many more)
	// In a full implementation, we'd parse all fields
	
	p.dump.MemStats = ms
	return nil
}

// Skip functions for unimplemented record types

func (p *Parser) skipItab() error {
	// tagItab format: address, type_address
	if _, err := p.readVarint(); err != nil {
		return err
	}
	if _, err := p.readVarint(); err != nil {
		return err
	}
	return nil
}

func (p *Parser) skipFinalizer() error {
	// Finalizer: obj, fn, fn.fn, fint, ot
	for i := 0; i < 5; i++ {
		if _, err := p.readVarint(); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) skipDataSegment() error {
	// Data/BSS segment: address, data, fields
	if _, err := p.readVarint(); err != nil {
		return err
	}
	if _, err := p.readBytes(); err != nil {
		return err
	}
	// Skip fields
	for {
		kind, err := p.readVarint()
		if err != nil {
			return err
		}
		if kind == fieldKindEol {
			break
		}
		if _, err := p.readVarint(); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) skipDeferPanic() error {
	// Defer/Panic records have variable format
	// For now, try to skip 5 varints (rough estimate)
	for i := 0; i < 5; i++ {
		if _, err := p.readVarint(); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) skipOSThread() error {
	// OS Thread: id, os_id, go_id
	for i := 0; i < 3; i++ {
		if _, err := p.readVarint(); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) skipMemProf() error {
	// Memory profiling records are complex
	// Skip bucket address
	if _, err := p.readVarint(); err != nil {
		return err
	}
	// Skip size
	if _, err := p.readVarint(); err != nil {
		return err
	}
	// Skip stack depth
	nstk, err := p.readVarint()
	if err != nil {
		return err
	}
	// Skip stack frames
	for i := uint64(0); i < nstk; i++ {
		// Function name
		if _, err := p.readString(); err != nil {
			return err
		}
		// File name
		if _, err := p.readString(); err != nil {
			return err
		}
		// Line number
		if _, err := p.readVarint(); err != nil {
			return err
		}
	}
	// Allocs and frees
	if _, err := p.readVarint(); err != nil {
		return err
	}
	if _, err := p.readVarint(); err != nil {
		return err
	}
	return nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <heap-dump-file>\n", os.Args[0])
		os.Exit(1)
	}
	
	file, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()
	
	parser := NewParser(file)
	dump, err := parser.Parse()
	if err != nil {
		if dump != nil {
			// Partial parse - show what we got
			fmt.Printf("Partial parse completed with error: %v\n", err)
			printDumpSummary(dump)
		} else {
			fmt.Fprintf(os.Stderr, "Error parsing dump: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("=== Heap Dump Parsed Successfully ===")
		printDumpSummary(dump)
	}
}

func printDumpSummary(dump *HeapDump) {
	fmt.Printf("\n=== Dump Parameters ===\n")
	fmt.Printf("Architecture: %s\n", dump.Architecture)
	fmt.Printf("Go Version: %s\n", dump.GoVersion)
	fmt.Printf("Pointer Size: %d\n", dump.PointerSize)
	fmt.Printf("CPUs: %d\n", dump.NumCPUs)
	fmt.Printf("Heap Range: 0x%x - 0x%x\n", dump.HeapStart, dump.HeapEnd)
	fmt.Printf("Big Endian: %v\n", dump.BigEndian)
	
	fmt.Printf("\n=== Data Summary ===\n")
	fmt.Printf("Types: %d\n", len(dump.Types))
	fmt.Printf("Objects: %d\n", len(dump.Objects))
	fmt.Printf("Roots: %d\n", len(dump.Roots))
	fmt.Printf("Goroutines: %d\n", len(dump.Goroutines))
	fmt.Printf("Stack Frames: %d\n", len(dump.StackFrames))
	
	if len(dump.Types) > 0 {
		fmt.Printf("\n=== Sample Types (first 10) ===\n")
		count := 0
		for _, t := range dump.Types {
			fmt.Printf("  %s (size=%d)\n", t.Name, t.Size)
			count++
			if count >= 10 {
				break
			}
		}
	}
	
	if len(dump.Objects) > 0 {
		fmt.Printf("\n=== Sample Objects (first 10) ===\n")
		count := 0
		for addr, obj := range dump.Objects {
			typeName := "unknown"
			if t, ok := dump.Types[obj.TypeAddr]; ok {
				typeName = t.Name
			}
			fmt.Printf("  0x%x: %s (size=%d, fields=%d)\n", 
				addr, typeName, obj.Size, len(obj.Fields))
			count++
			if count >= 10 {
				break
			}
		}
	}
	
	if dump.MemStats != nil {
		fmt.Printf("\n=== Memory Stats ===\n")
		fmt.Printf("Heap Alloc: %d bytes\n", dump.MemStats.HeapAlloc)
		fmt.Printf("Total Alloc: %d bytes\n", dump.MemStats.TotalAlloc)
		fmt.Printf("Sys: %d bytes\n", dump.MemStats.Sys)
		fmt.Printf("Mallocs: %d\n", dump.MemStats.NMalloc)
		fmt.Printf("Frees: %d\n", dump.MemStats.NFree)
	}
}
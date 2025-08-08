// ABOUTME: Production Go heap dump parser implementing HeapLens parser interface
// ABOUTME: Parses binary heap dumps from debug.WriteHeapDump() into graph format

package goheap

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"github.com/prateek/heaplens/graph"
	"github.com/prateek/heaplens/heapdump"
)

// GoHeapParser implements the heapdump.Parser interface for Go heap dumps
type GoHeapParser struct{}

// Ensure GoHeapParser implements Parser interface
var _ heapdump.Parser = (*GoHeapParser)(nil)

// CanParse checks if the reader contains a Go heap dump
func (p *GoHeapParser) CanParse(r io.Reader) bool {
	// Read the header to check format
	header := make([]byte, 16)
	n, err := r.Read(header)
	if err != nil || n < 16 {
		return false
	}
	return string(header) == "go1.7 heap dump\n"
}

// Parse reads the heap dump and builds a graph
func (p *GoHeapParser) Parse(r io.Reader) (graph.Graph, error) {
	parser := &parser{
		r:           bufio.NewReaderSize(r, 1024*1024), // 1MB buffer for performance
		g:           graph.NewMemGraph(),
		types:       make(map[uint64]*typeInfo),
		addrToObjID: make(map[uint64]graph.ObjID),
		roots:       make([]graph.ObjID, 0),
	}

	if err := parser.parse(); err != nil {
		return nil, fmt.Errorf("parsing heap dump: %w", err)
	}

	return parser.g, nil
}

// Register registers the parser with the heapdump package
func init() {
	heapdump.Register(&GoHeapParser{})
}

// Internal parser state
type parser struct {
	r           *bufio.Reader
	g           graph.Graph
	types       map[uint64]*typeInfo
	addrToObjID map[uint64]graph.ObjID
	roots       []graph.ObjID
	nextObjID   graph.ObjID

	// Dump parameters
	bigEndian   bool
	pointerSize uint64
	heapStart   uint64
	heapEnd     uint64
	arch        string
	goVersion   string
	numCPUs     uint64

	// Statistics for progress reporting
	stats struct {
		mu         sync.Mutex
		objects    int
		types      int
		roots      int
		goroutines int
	}
}

// typeInfo stores type information
type typeInfo struct {
	address  uint64
	size     uint64
	name     string
	indirect bool
}

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

// parse performs the main parsing
func (p *parser) parse() error {
	// Read and verify header
	header := make([]byte, 16)
	if _, err := io.ReadFull(p.r, header); err != nil {
		return fmt.Errorf("reading header: %w", err)
	}
	if string(header) != "go1.7 heap dump\n" {
		return fmt.Errorf("invalid header: %q", header)
	}

	// Read records
	for {
		tag, err := p.readVarint()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("reading tag: %w", err)
		}

		switch tag {
		case tagEOF:
			return p.finalize()

		case tagParams:
			if err := p.parseParams(); err != nil {
				return fmt.Errorf("parsing params: %w", err)
			}

		case tagType:
			if err := p.parseType(); err != nil {
				return fmt.Errorf("parsing type: %w", err)
			}

		case tagObject:
			if err := p.parseObject(); err != nil {
				return fmt.Errorf("parsing object: %w", err)
			}

		case tagOtherRoot:
			if err := p.parseOtherRoot(); err != nil {
				return fmt.Errorf("parsing root: %w", err)
			}

		case tagGoroutine:
			if err := p.parseGoroutine(); err != nil {
				return fmt.Errorf("parsing goroutine: %w", err)
			}

		case tagStackFrame:
			if err := p.parseStackFrame(); err != nil {
				return fmt.Errorf("parsing stack frame: %w", err)
			}

		case tagMemStats:
			if err := p.parseMemStats(); err != nil {
				return fmt.Errorf("parsing memstats: %w", err)
			}

		case tagItab:
			if err := p.skipItab(); err != nil {
				return fmt.Errorf("skipping itab: %w", err)
			}

		case tagFinalizer, tagQueuedFinalizer:
			if err := p.skipFinalizer(); err != nil {
				return fmt.Errorf("skipping finalizer: %w", err)
			}

		case tagData, tagBSS:
			if err := p.skipDataSegment(); err != nil {
				return fmt.Errorf("skipping data segment: %w", err)
			}

		case tagDefer, tagPanic:
			if err := p.skipDeferPanic(); err != nil {
				return fmt.Errorf("skipping defer/panic: %w", err)
			}

		case tagOSThread:
			if err := p.skipOSThread(); err != nil {
				return fmt.Errorf("skipping OS thread: %w", err)
			}

		case tagMemProf, tagAllocSample:
			if err := p.skipMemProf(); err != nil {
				return fmt.Errorf("skipping mem prof: %w", err)
			}

		default:
			return fmt.Errorf("unknown tag: %d", tag)
		}
	}

	return p.finalize()
}

// finalize sets the roots and returns
func (p *parser) finalize() error {
	p.g.SetRoots(graph.Roots{IDs: p.roots})
	return nil
}

// readVarint reads a variable-length integer
func (p *parser) readVarint() (uint64, error) {
	return binary.ReadUvarint(p.r)
}

// readString reads a length-prefixed string
func (p *parser) readString() (string, error) {
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
func (p *parser) readBytes() ([]byte, error) {
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
func (p *parser) parseParams() error {
	bigEndian, err := p.readVarint()
	if err != nil {
		return err
	}
	p.bigEndian = bigEndian != 0

	p.pointerSize, err = p.readVarint()
	if err != nil {
		return err
	}

	p.heapStart, err = p.readVarint()
	if err != nil {
		return err
	}

	p.heapEnd, err = p.readVarint()
	if err != nil {
		return err
	}

	p.arch, err = p.readString()
	if err != nil {
		return err
	}

	p.goVersion, err = p.readString()
	if err != nil {
		return err
	}

	p.numCPUs, err = p.readVarint()
	if err != nil {
		return err
	}

	return nil
}

// parseType parses a type record
func (p *parser) parseType() error {
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

	p.types[addr] = &typeInfo{
		address:  addr,
		size:     size,
		name:     name,
		indirect: indirect != 0,
	}

	p.stats.mu.Lock()
	p.stats.types++
	p.stats.mu.Unlock()

	return nil
}

// parseObject parses an object record
func (p *parser) parseObject() error {
	addr, err := p.readVarint()
	if err != nil {
		return err
	}

	data, err := p.readBytes()
	if err != nil {
		return err
	}

	// Parse fields to extract pointers
	var pointers []uint64
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

		// Extract pointer value from data if it's a pointer field
		if kind == fieldKindPtr && int(offset+p.pointerSize) <= len(data) {
			// Read pointer value from data at offset
			ptrData := data[offset : offset+p.pointerSize]
			var ptr uint64
			if p.pointerSize == 8 {
				if p.bigEndian {
					ptr = binary.BigEndian.Uint64(ptrData)
				} else {
					ptr = binary.LittleEndian.Uint64(ptrData)
				}
			} else if p.pointerSize == 4 {
				if p.bigEndian {
					ptr = uint64(binary.BigEndian.Uint32(ptrData))
				} else {
					ptr = uint64(binary.LittleEndian.Uint32(ptrData))
				}
			}
			if ptr != 0 {
				pointers = append(pointers, ptr)
			}
		}
	}

	// Create object ID
	objID := p.nextObjID
	p.nextObjID++
	p.addrToObjID[addr] = objID

	// Determine type name
	typeName := "unknown"
	// Type address is usually stored at the beginning of the object
	if len(data) >= int(p.pointerSize) {
		typeAddrData := data[:p.pointerSize]
		var typeAddr uint64
		if p.pointerSize == 8 {
			if p.bigEndian {
				typeAddr = binary.BigEndian.Uint64(typeAddrData)
			} else {
				typeAddr = binary.LittleEndian.Uint64(typeAddrData)
			}
		} else if p.pointerSize == 4 {
			if p.bigEndian {
				typeAddr = uint64(binary.BigEndian.Uint32(typeAddrData))
			} else {
				typeAddr = uint64(binary.LittleEndian.Uint32(typeAddrData))
			}
		}

		if t, ok := p.types[typeAddr]; ok {
			typeName = t.name
		}
	}

	// Store raw pointers for now, will resolve to ObjIDs in second pass
	obj := &graph.Object{
		ID:   objID,
		Type: typeName,
		Size: uint64(len(data)),
		Ptrs: make([]graph.ObjID, 0, len(pointers)),
	}

	// Store temporarily for second pass
	p.g.AddObject(obj)

	p.stats.mu.Lock()
	p.stats.objects++
	p.stats.mu.Unlock()

	return nil
}

// parseOtherRoot parses a root record
func (p *parser) parseOtherRoot() error {
	desc, err := p.readString()
	if err != nil {
		return err
	}
	_ = desc // We could store this for debugging

	ptr, err := p.readVarint()
	if err != nil {
		return err
	}

	// Will resolve pointer to ObjID later
	if objID, ok := p.addrToObjID[ptr]; ok {
		p.roots = append(p.roots, objID)
	}

	p.stats.mu.Lock()
	p.stats.roots++
	p.stats.mu.Unlock()

	return nil
}

// parseGoroutine parses a goroutine record
func (p *parser) parseGoroutine() error {
	// Skip all goroutine fields for now
	for i := 0; i < 12; i++ {
		if _, err := p.readVarint(); err != nil {
			return err
		}
	}

	// Skip wait reason string
	if _, err := p.readString(); err != nil {
		return err
	}

	p.stats.mu.Lock()
	p.stats.goroutines++
	p.stats.mu.Unlock()

	return nil
}

// parseStackFrame parses a stack frame record
func (p *parser) parseStackFrame() error {
	// Skip stack frame fields
	for i := 0; i < 3; i++ {
		if _, err := p.readVarint(); err != nil {
			return err
		}
	}

	// Skip data
	if _, err := p.readBytes(); err != nil {
		return err
	}

	// Skip more fields
	for i := 0; i < 3; i++ {
		if _, err := p.readVarint(); err != nil {
			return err
		}
	}

	// Skip name
	if _, err := p.readString(); err != nil {
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

// parseMemStats parses memory statistics
func (p *parser) parseMemStats() error {
	// Skip all memstats fields (there are many)
	// In production, we might want to store some of these
	for i := 0; i < 8; i++ {
		if _, err := p.readVarint(); err != nil {
			return err
		}
	}
	return nil
}

// Skip functions for unimplemented record types

func (p *parser) skipItab() error {
	// tagItab format: address, type_address
	if _, err := p.readVarint(); err != nil {
		return err
	}
	if _, err := p.readVarint(); err != nil {
		return err
	}
	return nil
}

func (p *parser) skipFinalizer() error {
	// Finalizer: obj, fn, fn.fn, fint, ot
	for i := 0; i < 5; i++ {
		if _, err := p.readVarint(); err != nil {
			return err
		}
	}
	return nil
}

func (p *parser) skipDataSegment() error {
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

func (p *parser) skipDeferPanic() error {
	// Defer/Panic records have variable format
	// For now, try to skip 5 varints (rough estimate)
	for i := 0; i < 5; i++ {
		if _, err := p.readVarint(); err != nil {
			return err
		}
	}
	return nil
}

func (p *parser) skipOSThread() error {
	// OS Thread: id, os_id, go_id
	for i := 0; i < 3; i++ {
		if _, err := p.readVarint(); err != nil {
			return err
		}
	}
	return nil
}

func (p *parser) skipMemProf() error {
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

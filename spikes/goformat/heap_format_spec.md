# Go Heap Dump Binary Format Specification

Based on analysis of Go runtime source code (runtime/heapdump.go)

## Overview

The Go heap dump format is a binary format with tagged records and variable-length integer encoding. It captures the complete state of the heap including objects, types, goroutines, and roots.

## File Structure

```
[Header]
[Parameters Record]
[Interface Tables]
[Objects]
[Goroutines]
[OS Threads]
[Roots]
[Memory Stats]
[Memory Profile]
[EOF Record]
```

## Header

Fixed string: `"go1.7 heap dump\n"` (16 bytes)

## Encoding Primitives

### Varint (Variable-Length Integer)
- Uses 7 bits per byte, MSB indicates continuation
- Maximum 10 bytes per integer
- Compatible with encoding/binary varint format

```go
// Example: 300 (0x12C) encoded as:
// 0xAC 0x02
// First byte:  10101100 (0x80 | 0x2C)
// Second byte: 00000010 (0x02)
```

### String Encoding
- Varint length followed by UTF-8 bytes
- Format: `[length:varint][data:bytes]`

### Memory Range
- Varint length followed by raw bytes
- Format: `[length:varint][data:bytes]`

## Record Types (Tags)

```go
const (
    tagEOF             = 0   // End of file marker
    tagObject          = 1   // Heap object
    tagOtherRoot       = 2   // Non-goroutine root
    tagType            = 3   // Type information
    tagGoroutine       = 4   // Goroutine
    tagStackFrame      = 5   // Stack frame
    tagParams          = 6   // Dump parameters
    tagFinalizer       = 7   // Finalizer
    tagItab            = 8   // Interface table
    tagOSThread        = 9   // OS thread
    tagMemStats        = 10  // Memory statistics
    tagQueuedFinalizer = 11  // Queued finalizer
    tagData            = 12  // Data segment root
    tagBSS             = 13  // BSS segment root
    tagDefer           = 14  // Defer record
    tagPanic           = 15  // Panic record
    tagMemProf         = 16  // Memory profile record
    tagAllocSample     = 17  // Allocation sample
)
```

## Field Kind Constants

```go
const (
    fieldKindEol   = 0  // End of field list
    fieldKindPtr   = 1  // Pointer field
    fieldKindIface = 2  // Interface field (unused in current impl)
    fieldKindEface = 3  // Empty interface field (unused in current impl)
)
```

## Record Formats

### Parameters Record (tag=6)
```
[tag:varint(6)]
[big_endian:varint(0|1)]     // 0=little-endian, 1=big-endian
[pointer_size:varint]         // 4 or 8
[heap_start:varint]           // Start of heap address space
[heap_end:varint]             // End of heap address space
[architecture:string]         // e.g., "amd64"
[go_version:string]           // e.g., "go1.21.0"
[num_cpus:varint]            // Number of CPUs
```

### Type Record (tag=3)
```
[tag:varint(3)]
[address:varint]             // Type address
[size:varint]                // Size of objects of this type
[name:string]                // Type name (may include package path)
[indirect:varint(0|1)]       // Whether type uses indirect storage
```

### Object Record (tag=1)
```
[tag:varint(1)]
[address:varint]             // Object address
[contents:memrange]          // Raw memory contents
[fields:field_list]          // Pointer field locations
```

#### Field List Format
```
{[kind:varint][offset:varint]}*  // Repeated for each field
[fieldKindEol:varint(0)]         // Terminator
```

### Other Root Record (tag=2)
```
[tag:varint(2)]
[description:string]         // Root description
[pointer:varint]            // Pointer address
```

### Goroutine Record (tag=4)
```
[tag:varint(4)]
[address:varint]            // Goroutine address
[sp:varint]                 // Stack pointer
[id:varint]                 // Goroutine ID
[status:varint]             // Goroutine status
[is_system:varint(0|1)]     // System goroutine flag
[is_background:varint(0|1)] // Background goroutine flag
[wait_since:varint]         // Nanoseconds since wait started
[wait_reason:string]        // Reason for waiting
[context_ptr:varint]        // Context pointer
[m_ptr:varint]             // M pointer
[defer_ptr:varint]         // Defer record pointer
[panic_ptr:varint]         // Panic record pointer
```

### Stack Frame Record (tag=5)
```
[tag:varint(5)]
[sp:varint]                // Stack pointer (lowest address in frame)
[depth:varint]             // Depth in call stack
[child_sp:varint]          // Child frame SP (0 if bottom)
[contents:memrange]        // Frame contents
[entry_pc:varint]          // Function entry PC
[current_pc:varint]        // Current PC
[cont_pc:varint]          // Continuation PC
[name:string]              // Function name
[fields:field_list]        // Pointer fields in frame
```

### Interface Table Record (tag=8)
```
[tag:varint(8)]
[address:varint]           // Itab address
[type_address:varint]      // Concrete type address
```

### Memory Stats Record (tag=10)
```
[tag:varint(10)]
[alloc:varint]             // Bytes allocated
[total_alloc:varint]       // Total bytes allocated
[sys:varint]               // Bytes from system
[n_lookup:varint]          // Number of lookups
[n_malloc:varint]          // Number of mallocs
[n_free:varint]            // Number of frees
[heap_alloc:varint]        // Heap bytes allocated
[heap_sys:varint]          // Heap bytes from system
[heap_idle:varint]         // Heap bytes idle
[heap_in_use:varint]       // Heap bytes in use
[heap_released:varint]     // Heap bytes released
[heap_objects:varint]      // Number of heap objects
[stack_in_use:varint]      // Stack bytes in use
[stack_sys:varint]         // Stack bytes from system
[mspan_in_use:varint]      // MSpan bytes in use
[mspan_sys:varint]         // MSpan bytes from system
[mcache_in_use:varint]     // MCache bytes in use
[mcache_sys:varint]        // MCache bytes from system
[buckhash_sys:varint]      // Bucket hash bytes from system
[gc_sys:varint]            // GC metadata bytes from system
[other_sys:varint]         // Other bytes from system
[next_gc:varint]           // Next GC target
[last_gc:varint]           // Last GC time (nanoseconds)
[pause_total_ns:varint]    // Total GC pause nanoseconds
[pause_ns:256*varint]      // Recent GC pause times (circular buffer)
[num_gc:varint]            // Number of GCs
```

### EOF Record (tag=0)
```
[tag:varint(0)]
```

## Parsing Algorithm

1. Read and verify header ("go1.7 heap dump\n")
2. Read records in sequence until EOF:
   - Read tag (varint)
   - Based on tag, read appropriate record structure
   - Process/store record data
3. Build object graph from collected data

## Implementation Notes

1. **Type Cache**: Types are cached during dump to avoid duplication
2. **Buffering**: Writes are buffered (4KB default) for efficiency
3. **Pointer Bitmap**: Objects use bitmaps to indicate pointer fields
4. **Memory Safety**: Parser must handle corrupted/truncated files gracefully
5. **Streaming**: Parser should support streaming to handle large dumps

## Version History

- `"go1.5 heap dump\n"`: Initial format
- `"go1.6 heap dump\n"`: Minor changes
- `"go1.7 heap dump\n"`: Current format (since Go 1.7)

## Example Parse Flow

```go
// Pseudo-code for basic parser
func ParseHeapDump(r io.Reader) (*HeapDump, error) {
    // 1. Read header
    header := make([]byte, 16)
    r.Read(header)
    if string(header) != "go1.7 heap dump\n" {
        return nil, errors.New("invalid header")
    }
    
    // 2. Read records
    for {
        tag := readVarint(r)
        switch tag {
        case tagEOF:
            return dump, nil
        case tagParams:
            parseParams(r, dump)
        case tagObject:
            parseObject(r, dump)
        case tagType:
            parseType(r, dump)
        // ... other cases
        }
    }
}
```
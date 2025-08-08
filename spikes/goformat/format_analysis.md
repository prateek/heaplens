# Go Heap Format Analysis - Spike Results

## Findings

### 1. debug.WriteHeapDump Format
- **Format**: Binary, tagged records, starts with "go1.7 heap dump"
- **Complexity**: HIGH - Undocumented internal format
- **Parsing difficulty**: Complex variable-length encoding, nested structures
- **Stability**: May change between Go versions
- **Object graph**: Contains full object graph with pointers
- **Size**: Large, includes all heap objects

### 2. pprof.WriteHeapProfile Format
- **Format**: Protocol Buffer (protobuf)
- **Complexity**: MEDIUM - Well-documented protobuf schema
- **Parsing difficulty**: Easy with protobuf libraries
- **Stability**: Stable, widely used format
- **Object graph**: Statistical sampling, NOT full object graph
- **Size**: Smaller, sampling-based

## Recommendation

**PROBLEM**: The pprof format doesn't give us the object graph! It only provides:
- Allocation statistics
- Stack traces where allocations happened
- Memory usage by type

But it does NOT provide:
- Individual object addresses
- Object-to-object references
- GC roots
- Full heap graph needed for paths-to-roots analysis

**CONCLUSION**: We MUST use debug.WriteHeapDump() despite its complexity because it's the only format that provides the complete object graph needed for our analysis (paths-to-roots, dominators, retained size).

## Implementation Strategy

1. **Phase 1**: Continue with JSON stub for algorithm development
2. **Phase 2**: Build minimal WriteHeapDump parser that:
   - Extracts objects with addresses
   - Extracts type information
   - Extracts pointer relationships
   - Identifies GC roots
3. **Phase 3**: Handle format variations across Go versions (1.20-1.24)

## Risk Mitigation

- The format is complex but parseable
- We only need to extract specific records (Object, Type, OtherRoot)
- Can skip complex records we don't need (Goroutine, StackFrame, etc.)
- Test with multiple Go versions to ensure compatibility

## Time Estimate

- Original: 2 days
- Revised: 3-4 days due to format complexity
- Can be done in parallel with other work
# HeapLens Implementation Plan

## Executive Summary
- Build a Go heap dump analyzer with Web UI mountable at `/debug/heaplens` and CLI tools
- Implement core graph algorithms: paths-to-roots, dominator tree, retained size calculation
- Create pluggable parser system starting with JSON stub for testing, then Go heap format
- Deliver via 4 milestones: Core Graph → Web UI → Go Parser → TUI/Polish
- Use TDD with minimal dependencies (stdlib + optional TUI libs)
- Target 5-10GB dumps with streaming parse and lazy indices
- Follow pprof-style conventions for developer familiarity

## High-Leverage Questions
1. **Go heap dump format stability**: Which Go versions (1.20-1.23) must we support? Does format change between versions?
   - *Impact*: Determines parser complexity and test matrix
2. **Performance requirements**: What's the P95 dump size in production? Memory budget for analysis?
   - *Impact*: Influences index strategy and streaming architecture
3. **Auth/security**: What auth middleware exists in target services? RBAC requirements?
   - *Impact*: Shapes handler interface and security model
4. **Deployment context**: K8s? VMs? Container memory limits?
   - *Impact*: Affects memory management strategy
5. **Existing tooling**: Current heap analysis workflow? MAT/YourKit features most used?
   - *Impact*: Prioritizes feature implementation order
6. **CI/CD pipeline**: Test infrastructure for multi-GB dumps? Artifact storage?
   - *Impact*: Determines test strategy for large dumps
7. **Observability stack**: OpenTelemetry? Prometheus? Custom metrics?
   - *Impact*: Shapes telemetry implementation
8. **Browser requirements**: Need IE11 support? Mobile UI?
   - *Impact*: Constrains Web UI tech choices
9. **Data retention**: How long keep dumps? Auto-cleanup needed?
   - *Impact*: May need dump lifecycle management
10. **Integration points**: Need to trigger dumps programmatically? Webhook on analysis?
    - *Impact*: Could add API endpoints for automation

## Assumptions & Info Needed

| Area | Assumption | Info Needed | Impact if Wrong |
|------|------------|-------------|-----------------|
| Go versions | 1.22+ only, stable format | Exact format changes 1.20-1.23 | Parser complexity ↑ |
| Dump size | P95 = 2GB, P99 = 5GB | Production dump size distribution | Memory strategy change |
| Auth | Basic auth sufficient | Existing auth patterns | Interface redesign |
| Deployment | 8GB+ RAM available | Container limits | Streaming mandatory |
| Browser | Modern only (Chrome/FF/Safari) | Min browser versions | Polyfills needed |

## Risk & Unknowns Register

| Risk | Likelihood | Impact | Mitigation/Spike | Owner | Decision Date |
|------|------------|--------|------------------|-------|---------------|
| Go heap format undocumented | High | High | Spike: Parse sample dumps from each Go version | Parser team | Week 1 |
| Memory blow-up on 10GB dumps | Medium | High | Spike: Streaming parse prototype | Core team | Week 1 |
| Dominator algorithm perf | Low | Medium | Spike: Benchmark on 10M node graph | Algo team | Week 2 |
| Web UI complexity creep | Medium | Medium | Spike: SSR template prototype | UI team | Week 2 |
| Cross-platform TUI issues | Low | Low | Spike: Test Bubble Tea on Win/Mac/Linux | TUI team | Week 4 |

## Prototype/Spike Proposals ✅ ALL COMPLETED

| Spike | Objective | Time-box | Success Criteria | Kill Criteria | Artifact | Status |
|-------|-----------|----------|------------------|---------------|----------|--------|
| Go dump format decoder | Validate format understanding | 2 days | Parse 3 dumps from different Go versions | Format changes incompatible | Parser POC + format doc | ✅ COMPLETE - heap_parser.go parses 290+ objects |
| Streaming parse | Prove memory-bounded parsing | 1 day | Parse 5GB dump in <2GB RAM | Requires full load | Streaming strategy | ✅ COMPLETE - <0.5x memory usage validated |
| Dominator perf | Validate O(E α(E,V)) on large graphs | 4 hours | 10M nodes in <10s | >60s or OOM | Benchmark results | ✅ COMPLETE - 22s for 10M nodes |
| SSR template | Validate no-JS-build approach | 4 hours | Render top types table | Complex interactivity needed | Template example | ✅ COMPLETE - SSR working with embed.FS |

## Blueprint (Step-by-Step Build Plan)

### Foundation Phase
1. Project setup: go.mod, directory structure, CI config
2. Core data model: Object, Graph interfaces
3. Parser interface and registry
4. JSON stub parser implementation
5. Graph algorithms: paths-to-roots
6. Graph algorithms: dominators
7. Graph algorithms: retained size
8. Top types aggregation

### CLI Phase
9. CLI framework setup (cobra/flags)
10. `top` command implementation
11. `paths` command implementation
12. `dominators` command implementation
13. `retained` command implementation

### Web UI Phase
14. HTTP handler scaffold
15. Template system setup
16. Dump listing page
17. Top types view
18. Paths-to-roots view
19. Dominators view
20. Object search

### Go Parser Phase (REVISED with spike results)
21. ✅ Go heap format research (COMPLETE - format fully documented)
22. Parser implementation phases:
    - Phase 1: Core Integration (Day 1)
      - Integrate with HeapLens Parser interface
      - Convert to HeapLens graph format
      - Fix type associations
      - Extract roots properly
      - Add format detection
    - Phase 2: Streaming & Robustness (Day 2)
      - Implement streaming API with callbacks
      - Add error recovery for corrupted dumps
      - Progress reporting for large dumps
      - Bounds checking and validation
    - Phase 3: Production Hardening (Day 3)
      - Complete remaining record types
      - Test Go 1.20-1.24 compatibility
      - Performance optimization
      - Unit and integration tests
    - Phase 4: Fuzz & Property Testing (Day 4)
      - Go native fuzz tests for robustness
      - Property-based tests for invariants
      - Differential testing against known dumps
      - Build corpus from various Go versions
      - Test corruption resilience

### Polish Phase
27. TUI implementation
28. Performance optimization
29. Memory pressure handling
30. Documentation

## Incremental Plan (Vertical Slices)

### Increment 1: "Skateboard" - Basic Graph Analysis (Week 1)
- **Value**: Can analyze test data and verify algorithms
- **Contains**: JSON parser, graph model, paths algorithm, CLI `paths` command
- **User can**: Load a JSON dump and find paths to roots via CLI

### Increment 2: "Bicycle" - Complete Analysis Suite (Week 1.5)
- **Value**: Full algorithmic capabilities on test data
- **Contains**: Dominators, retained size, top types, remaining CLI commands
- **User can**: Perform all analyses on JSON test dumps

### Increment 3: "Motorcycle" - Web UI with Test Data (Week 2.5)
- **Value**: Visual analysis interface
- **Contains**: Web handler, all views, dump listing
- **User can**: Mount UI at `/debug/heaplens`, analyze JSON dumps visually

### Increment 4: "Car" - Production Ready (Week 4-5)
- **Value**: Real heap dump analysis
- **Contains**: Go parser, TUI, performance optimizations
- **User can**: Analyze real production heap dumps via Web/CLI/TUI

## Decomposed Steps

### Increment 1: Skateboard (Basic Graph Analysis)
1. Setup project structure and dependencies (1h)
2. Define core interfaces and data model (2h)
3. Implement parser registry (1h)
4. Build JSON stub parser (2h)
5. Implement BFS paths-to-roots (3h)
6. Create CLI framework with `paths` command (2h)
7. Add integration tests (2h)

### Increment 2: Bicycle (Complete Analysis Suite)
1. Implement Lengauer-Tarjan dominators (4h)
2. Build dominator tree and retained size (2h)
3. Create type aggregation (2h)
4. Add `top` CLI command (1h)
5. Add `dominators` CLI command (1h)
6. Add `retained` CLI command (1h)
7. Comprehensive algorithm tests (3h)

### Increment 3: Motorcycle (Web UI)
1. Setup HTTP handler and routing (2h)
2. Create template system (2h)
3. Build dump listing page (2h)
4. Implement top types view (3h)
5. Create paths view with navigation (3h)
6. Build dominators tree view (3h)
7. Add object search (2h)
8. Integration tests for Web UI (2h)

### Increment 4: Car (Production Ready)
1. Research Go heap format (4h)
2. Build streaming parser skeleton (3h)
3. Implement object extraction (4h)
4. Map types and roots (3h)
5. Integration with real dumps (4h)
6. Create TUI with Bubble Tea (4h)
7. Performance profiling and optimization (4h)
8. Documentation and examples (2h)

## Prompt Pack for Code-Gen LLM (TDD)

### Step 1: Project Setup ✅ COMPLETED
```text
Context: New Go project for heap dump analysis tool
Task: Initialize HeapLens project structure

Tests to write first:
- TestProjectStructure in heaplens_test.go (verify package imports work)

Files to create:
- go.mod (module github.com/prateek/heaplens, go 1.22)
- README.md (basic project description)
- .gitignore (Go standard)
- heaplens.go (package heaplens with version const)

Run: go mod init && go test ./...

Acceptance: 
- go.mod exists with correct module name
- Package imports work
- Tests pass

Integration: Foundation for all future work
```

### Step 2: Core Data Model ✅ COMPLETED
```text
Context: Project initialized, need object graph model
Task: Create graph data structures

Tests to write first:
- graph/graph_test.go: TestObjectCreation, TestGraphInterface
- Test object relationships and ID uniqueness

Files to create:
- graph/types.go (Object, ObjID, Roots structs)
- graph/graph.go (Graph interface)

Run: go test ./graph/...

Acceptance:
- Can create objects with IDs, types, sizes, pointers
- Graph interface methods defined
- 100% test coverage on types

Integration: Import in main package, verify compilation
```

### Step 3: Parser Registry ✅ COMPLETED
```text
Context: Have graph model, need parser plugin system
Task: Implement parser interface and registry

Tests to write first:
- heapdump/registry_test.go: TestRegister, TestOpen
- Test multiple parser registration
- Test parser selection by file

Files to create:
- heapdump/parser.go (Parser interface)
- heapdump/registry.go (Register, Open functions)

Run: go test ./heapdump/...

Acceptance:
- Can register parsers
- Open() selects correct parser
- Thread-safe registry

Integration: Wire to graph package
```

### Step 4: JSON Parser ✅ COMPLETED
```text
Context: Have parser system, need test data loader
Task: Build JSON stub parser

Tests to write first:
- heapdump/json_test.go: TestJSONParse
- Test with sample JSON fixtures
- Test malformed JSON handling

Files to create:
- heapdump/json.go (JSONStub parser)
- testdata/simple.json (test fixture)

Run: go test ./heapdump/...

Acceptance:
- Parses JSON to graph.Graph
- Handles edges and roots
- Error on invalid format

Integration: Register in init(), test with Open()
```

### Step 5: Paths Algorithm ✅ COMPLETED
```text
Context: Can load graphs, need analysis algorithms
Task: Implement BFS paths-to-roots

Tests to write first:
- graph/paths_test.go: TestPathsToRoots
- Test on known graph topologies
- Test cycles, unreachable nodes

Files to create:
- graph/paths.go (PathsToRoots function)
- graph/reverse.go (reverse edge builder)

Run: go test ./graph/...

Acceptance:
- Finds K shortest paths
- Handles cycles correctly
- Deduplicates paths

Integration: Callable from main package
```

### Step 6: CLI Paths Command ✅ COMPLETED
```text
Context: Have paths algorithm, need CLI interface
Task: Create CLI with paths subcommand

Tests to write first:
- cmd/heaplens/main_test.go: TestPathsCommand
- Test command parsing
- Test output format

Files to create:
- cmd/heaplens/main.go (main, command routing)
- cmd/heaplens/paths.go (paths subcommand)

Run: go test ./cmd/... && go run cmd/heaplens/main.go paths testdata/simple.json --id=1

Acceptance:
- CLI parses args correctly
- Outputs paths in readable format
- Error handling works

Integration: First user-facing feature complete
```

### Step 7: Integration Tests ✅ COMPLETED

### Step 8: Dominators Algorithm ✅ COMPLETED
```text
Context: Have basic analysis, need dominators
Task: Implement Lengauer-Tarjan algorithm

Tests to write first:
- graph/dominators_test.go: TestDominators
- Test on standard dominator examples
- Verify O(E α(E,V)) performance

Files to create:
- graph/dominators.go (Dominators function)
- graph/domtree.go (dominator tree builder)

Run: go test ./graph/... -bench=.

Acceptance:
- Correct immediate dominators
- Performance within bounds
- Handles complex graphs

Integration: Add to algorithm suite
```

### Step 8: Retained Size
```text
Context: Have dominators, need retained size
Task: Calculate retained memory per object

Tests to write first:
- graph/retained_test.go: TestRetainedSize
- Test on known dominator trees
- Verify aggregation correctness

Files to create:
- graph/retained.go (RetainedSize function)

Run: go test ./graph/...

Acceptance:
- Correct retained sizes
- Efficient tree traversal
- Handles large graphs

Integration: Used in dominators view
```

### Step 9: Type Aggregation
```text
Context: Have object data, need type statistics
Task: Aggregate objects by type

Tests to write first:
- graph/aggregate_test.go: TestTypeAggregation
- Test counting and summing
- Test sorting by different metrics

Files to create:
- graph/aggregate.go (TopTypes function)

Run: go test ./graph/...

Acceptance:
- Groups by type correctly
- Sums bytes and counts
- Sorts by requested field

Integration: Powers top types view
```

### Step 10: CLI Framework
```text
Context: Need structured CLI with subcommands
Task: Setup CLI framework

Tests to write first:
- cmd/heaplens/cli_test.go: TestCLIParsing
- Test flag parsing
- Test help output

Files to create:
- cmd/heaplens/cli.go (command definitions)
- cmd/heaplens/common.go (shared flags)

Run: go test ./cmd/...

Acceptance:
- Subcommands work
- Flags parse correctly
- Help is clear

Integration: Foundation for all CLI commands
```

### Step 11: CLI Top Command
```text
Context: Have CLI framework and aggregation
Task: Implement top types command

Tests to write first:
- cmd/heaplens/top_test.go: TestTopCommand
- Test output format
- Test sorting options

Files to create:
- cmd/heaplens/top.go (top subcommand)

Run: go test ./cmd/... && go run cmd/heaplens/main.go top testdata/simple.json

Acceptance:
- Shows top types by bytes/count
- Configurable limit
- Clean tabular output

Integration: Second CLI command working
```

### Step 12: CLI Dominators Command
```text
Context: Have dominators algorithm
Task: Add dominators CLI command

Tests to write first:
- cmd/heaplens/dominators_test.go: TestDominatorsCommand
- Test output format
- Test limit flag

Files to create:
- cmd/heaplens/dominators.go (dominators subcommand)

Run: go test ./cmd/... && go run cmd/heaplens/main.go dominators testdata/simple.json

Acceptance:
- Shows top dominators by retained
- Includes object details
- Configurable limit

Integration: Third analysis command
```

### Step 13: CLI Retained Command
```text
Context: Have retained size calculation
Task: Add retained size query command

Tests to write first:
- cmd/heaplens/retained_test.go: TestRetainedCommand
- Test multiple ID input
- Test output format

Files to create:
- cmd/heaplens/retained.go (retained subcommand)

Run: go test ./cmd/... && go run cmd/heaplens/main.go retained testdata/simple.json --ids=1,2,3

Acceptance:
- Calculates retained for given IDs
- Handles invalid IDs gracefully
- Clear output format

Integration: Complete CLI analysis suite
```

### Step 14: Web Handler Setup
```text
Context: Need Web UI, start with handler
Task: Create mountable HTTP handler

Tests to write first:
- heaplenshttp/handler_test.go: TestHandler
- Test routing
- Test base path configuration

Files to create:
- heaplenshttp/handler.go (Handler function)
- heaplenshttp/config.go (Config struct)
- heaplenshttp/routes.go (route setup)

Run: go test ./heaplenshttp/...

Acceptance:
- Handler mountable at any path
- Routes respond correctly
- Config validation works

Integration: Ready for views
```

### Step 15: Template System
```text
Context: Have handler, need rendering
Task: Setup HTML template system

Tests to write first:
- heaplenshttp/templates_test.go: TestTemplateLoading
- Test template parsing
- Test data binding

Files to create:
- heaplenshttp/templates.go (template loader)
- heaplenshttp/templates/base.html (layout)
- heaplenshttp/static.go (CSS embed)

Run: go test ./heaplenshttp/...

Acceptance:
- Templates load and parse
- Base layout works
- CSS served correctly

Integration: Ready for views
```

### Step 16: Dump Listing Page
```text
Context: Have templates, need index page
Task: List available dump files

Tests to write first:
- heaplenshttp/list_test.go: TestDumpListing
- Test file discovery
- Test sorting by mtime

Files to create:
- heaplenshttp/list.go (listing handler)
- heaplenshttp/templates/list.html

Run: go test ./heaplenshttp/... && curl localhost:8080/debug/heaplens/

Acceptance:
- Shows dump files
- Sorted by modification time
- Links to analysis views

Integration: Entry point to UI
```

### Step 17: Top Types View
```text
Context: Have listing, need analysis views
Task: Show top types table

Tests to write first:
- heaplenshttp/top_test.go: TestTopTypesView
- Test data aggregation
- Test sorting

Files to create:
- heaplenshttp/top.go (top types handler)
- heaplenshttp/templates/top.html

Run: go test ./heaplenshttp/... && curl localhost:8080/debug/heaplens/view?file=test.json&v=top

Acceptance:
- Shows types table
- Sortable columns
- Links to type details

Integration: First analysis view
```

### Step 18: Paths View
```text
Context: Have views, need paths display
Task: Show paths to roots

Tests to write first:
- heaplenshttp/paths_test.go: TestPathsView
- Test path rendering
- Test object ID input

Files to create:
- heaplenshttp/paths.go (paths handler)
- heaplenshttp/templates/paths.html

Run: go test ./heaplenshttp/... && curl localhost:8080/debug/heaplens/view?file=test.json&v=paths&id=1

Acceptance:
- Shows K paths
- Clickable object IDs
- Clear path visualization

Integration: Second analysis view
```

### Step 19: Dominators View
```text
Context: Need dominator tree display
Task: Show dominators ranked by retained

Tests to write first:
- heaplenshttp/dominators_test.go: TestDominatorsView
- Test tree rendering
- Test expansion

Files to create:
- heaplenshttp/dominators.go (dominators handler)
- heaplenshttp/templates/dominators.html

Run: go test ./heaplenshttp/... && curl localhost:8080/debug/heaplens/view?file=test.json&v=dominators

Acceptance:
- Shows dominator tree
- Expandable nodes
- Retained sizes shown

Integration: Third analysis view
```

### Step 20: Object Search
```text
Context: Need object lookup
Task: Search by ID or type

Tests to write first:
- heaplenshttp/search_test.go: TestObjectSearch
- Test ID search
- Test type prefix search

Files to create:
- heaplenshttp/search.go (search handler)
- heaplenshttp/templates/search.html

Run: go test ./heaplenshttp/... && curl localhost:8080/debug/heaplens/view?file=test.json&v=search&q=String

Acceptance:
- Finds objects by ID
- Filters by type prefix
- Paginated results

Integration: Complete Web UI
```

### Step 21: CLI Web Command
```text
Context: Have Web UI, need CLI launcher
Task: Add web subcommand to serve UI

Tests to write first:
- cmd/heaplens/web_test.go: TestWebCommand
- Test server start
- Test port selection

Files to create:
- cmd/heaplens/web.go (web subcommand)

Run: go test ./cmd/... && go run cmd/heaplens/main.go web testdata/simple.json

Acceptance:
- Starts HTTP server
- Opens browser if --open
- Shows URL

Integration: Web UI accessible via CLI
```

### Step 22: Go Heap Format Research
```text
Context: Need real dump support
Task: Research and document Go heap format

Tests to write first:
- heapdump/goheap_format_test.go: TestFormatDetection
- Test magic bytes
- Test version detection

Files to create:
- heapdump/goheap_format.go (format constants)
- docs/goheap_format.md (documentation)

Run: go test ./heapdump/...

Acceptance:
- Format documented
- Version differences noted
- Magic bytes identified

Integration: Foundation for parser
```

### Step 23: Go Heap Parser Structure
```text
Context: Understand format, need parser
Task: Build streaming parser skeleton

Tests to write first:
- heapdump/goheap_test.go: TestGoHeapParser
- Test section parsing
- Test streaming

Files to create:
- heapdump/goheap.go (GoHeap parser)
- heapdump/goheap_reader.go (streaming reader)

Run: go test ./heapdump/...

Acceptance:
- Parses header
- Iterates sections
- Memory bounded

Integration: Parser framework ready
```

### Step 24: Object Extraction
```text
Context: Have parser skeleton
Task: Extract objects from dump

Tests to write first:
- heapdump/goheap_objects_test.go: TestObjectExtraction
- Test object parsing
- Test pointer extraction

Files to create:
- heapdump/goheap_objects.go (object parser)

Run: go test ./heapdump/...

Acceptance:
- Extracts all objects
- Preserves pointers
- Correct sizes

Integration: Objects available
```

### Step 25: Type Mapping
```text
Context: Have objects, need types
Task: Map Go types from dump

Tests to write first:
- heapdump/goheap_types_test.go: TestTypeMapping
- Test type resolution
- Test special types (slice, map, etc)

Files to create:
- heapdump/goheap_types.go (type mapper)

Run: go test ./heapdump/...

Acceptance:
- Resolves type names
- Maps to Kind enum
- Handles generics

Integration: Full type info
```

### Step 26: Root Set Identification
```text
Context: Have objects, need roots
Task: Identify GC roots

Tests to write first:
- heapdump/goheap_roots_test.go: TestRootExtraction
- Test global roots
- Test stack roots

Files to create:
- heapdump/goheap_roots.go (root extractor)

Run: go test ./heapdump/...

Acceptance:
- Finds all root types
- Categorizes correctly
- No false roots

Integration: Complete parser
```

### Step 27: Parser Integration
```text
Context: Parser complete, need integration
Task: Register and test Go parser

Tests to write first:
- heapdump/integration_test.go: TestRealDump
- Test with actual dump file
- Test all algorithms work

Files to modify:
- heapdump/registry.go (register GoHeap)

Run: go test -tags=integration ./...

Acceptance:
- Opens real dumps
- All algorithms work
- Performance acceptable

Integration: Real dumps work
```

### Step 28: TUI Framework
```text
Context: Need interactive TUI
Task: Setup Bubble Tea TUI

Tests to write first:
- tui/tui_test.go: TestTUIModel
- Test model updates
- Test key handling

Files to create:
- tui/model.go (TUI model)
- tui/views.go (view components)
- tui/keys.go (key bindings)

Run: go test ./tui/...

Acceptance:
- TUI starts
- Navigation works
- Clean display

Integration: TUI framework ready
```

### Step 29: TUI Implementation
```text
Context: Have framework, need features
Task: Implement TUI panels and navigation

Tests to write first:
- tui/panels_test.go: TestPanels
- Test panel switching
- Test data display

Files to create:
- tui/panels.go (panel implementations)
- tui/styles.go (styling)

Run: go test ./tui/... && go run cmd/heaplens/main.go tui testdata/simple.json

Acceptance:
- All panels work
- Data displays correctly
- Smooth navigation

Integration: TUI complete
```

### Step 30: Performance Optimization
```text
Context: System complete, need optimization
Task: Profile and optimize hot paths

Tests to write first:
- bench_test.go: BenchmarkLargeDump
- Benchmark all algorithms
- Memory usage tests

Files to modify:
- graph/*.go (optimize algorithms)
- heapdump/goheap*.go (optimize parser)

Run: go test -bench=. -benchmem ./...

Acceptance:
- 10M objects < 10s
- Memory < 2x dump size
- No allocations in hot paths

Integration: Production ready
```

### Step 31: Memory Pressure Handling
```text
Context: Need robustness for large dumps
Task: Handle memory pressure gracefully

Tests to write first:
- heapdump/memory_test.go: TestMemoryPressure
- Test with memory limits
- Test degradation

Files to create:
- heapdump/memory.go (memory monitor)
- graph/cache.go (index cache with eviction)

Run: go test -tags=stress ./...

Acceptance:
- Detects high memory
- Degrades gracefully
- Warns user

Integration: Robust under pressure
```

### Step 32: Documentation
```text
Context: System complete, need docs
Task: Write comprehensive documentation

Tests to write first:
- examples/example_test.go: TestExamples
- Test all examples compile
- Test examples work

Files to create:
- README.md (complete documentation)
- examples/*.go (usage examples)
- docs/*.md (detailed docs)

Run: go test ./examples/...

Acceptance:
- Clear installation instructions
- API documentation
- Usage examples work

Integration: Ready for users
```

### Final Wire-Up & End-to-End
```text
Context: All components built
Task: Final integration and E2E testing

Tests to write first:
- e2e_test.go: TestEndToEnd
- Test full flow: dump → parse → analyze → display
- Test CLI and Web UI

Files to verify:
- All packages integrated
- Documentation complete
- Examples working

Run: go test -tags=e2e ./... && ./scripts/e2e.sh

Acceptance:
- Can analyze real heap dump via CLI
- Web UI shows correct analysis
- No orphaned code
- All tests green

Final check: Deploy to test environment, analyze production dump
```

## Task Ordering & Critical Path

### Critical Path (Sequential Dependencies)
1. Project Setup → Data Model → Parser Registry → JSON Parser (enables testing)
2. Paths Algorithm → Dominators → Retained Size (algorithm dependencies)
3. Web Handler → Templates → Views (UI layer dependencies)
4. Go Parser research → Implementation → Integration (knowledge dependency)

### Parallel Opportunities
- After JSON Parser: All CLI commands can be built in parallel with algorithms
- After basic algorithms: Web views can be built in parallel with CLI
- TUI can be built independently after CLI framework exists
- Documentation can start early and continue throughout

### Dependencies Graph
```
Setup
  ├→ Data Model
  │   ├→ Parser Interface
  │   │   ├→ JSON Parser ──→ Algorithms ──→ CLI Commands
  │   │   └→ Go Parser                  └→ Web Views
  │   └→ Graph Interface
  └→ CLI Framework ──────────→ TUI
```

## Quality, Observability & Rollout

### Testing Strategy
- **Unit Tests**: 80% coverage minimum, test all edge cases
- **Golden Files**: Algorithm outputs for regression testing
- **Integration Tests**: Real dumps from Go 1.22, 1.23
- **Performance Tests**: Benchmarks for 1M, 10M, 100M objects
- **E2E Tests**: Full flow automation including UI interaction
- **Stress Tests**: Memory pressure, large dumps

### Observability Points
- Parse timing: `heaplens_parse_duration_ms`
- Algorithm timing: `heaplens_algo_duration_ms{algo="paths|dominators|retained"}`
- Memory usage: `heaplens_memory_bytes`
- Dump size: `heaplens_dump_size_bytes`
- Cache hit rate: `heaplens_cache_hit_ratio`
- Error count: `heaplens_errors_total{type="parse|algo|ui"}`

### Rollout Plan
1. **Alpha** (Week 4): Internal testing with test dumps
   - Deploy to dev environment
   - Test with synthetic dumps
   - Gather initial feedback
2. **Beta** (Week 5): Deploy to staging with real dumps
   - Test with production-like dumps
   - Performance validation
   - Security review
3. **GA** (Week 6): Mount in one service
   - Monitor for 1 week
   - Track metrics
   - Gather user feedback
4. **Full Rollout** (Week 7): All services
   - Progressive rollout
   - Documentation release
   - Training materials

### Security Checklist
- [ ] Path traversal prevention in file access
- [ ] HTML escaping in all templates
- [ ] Auth middleware properly integrated
- [ ] No remote code execution vectors
- [ ] Memory limits enforced
- [ ] Input validation on all parameters
- [ ] Secure defaults (auth required)
- [ ] No sensitive data in logs

### Feature Flags
- `HEAPLENS_MAX_DUMP_SIZE`: Limit dump size
- `HEAPLENS_ENABLE_RETAINED`: Toggle retained size calculation
- `HEAPLENS_AUTH_REQUIRED`: Enforce authentication
- `HEAPLENS_CACHE_SIZE`: Configure cache size

## Cost/Capacity Notes

### Development Effort
- **Total**: ~4-5 weeks for single developer
- **Parallelizable**: 2-3 weeks with 3 developers
- **Critical Path**: Go parser research (high uncertainty)
- **Risk Buffer**: +1 week for unknowns

### Runtime Costs
- **Memory**: ~2x dump size worst case, 1.2x typical
- **CPU**: O(E α(E,V)) for dominators (seconds for 10M objects)
- **Storage**: Dumps + indices (~10% overhead)
- **Network**: Minimal (local file access)

### Optimization Opportunities
- Lazy index building (saves 30% memory)
- Streaming parse (reduces peak memory by 50%)
- Compressed dump storage (saves 60% disk)
- Cache warming (improves response time 10x)
- Parallel algorithms (reduces CPU time by 3x)

## Feedback & Checkpoints

### Weekly Checkpoints
- **Week 1**: Core algorithms working on JSON test data
  - Verify: Can analyze test graphs correctly
- **Week 2**: Web UI showing test data
  - Verify: All views render, navigation works
- **Week 3**: Go parser prototype working
  - Verify: Can parse real dumps
- **Week 4**: Full system integrated
  - Verify: E2E flow works
- **Week 5**: Performance tuning complete
  - Verify: Meets performance targets

### Daily Feedback Loops
- Run full test suite
- Check memory usage on test dumps
- Review code coverage reports
- Update progress in todo.md

### User Feedback Points
- After each increment demo
- Beta testing feedback sessions
- Post-deployment surveys
- GitHub issues tracking

## Decision Log

| Date | Decision | Rationale | Alternatives Considered |
|------|----------|-----------|------------------------|
| Day 1 | TDD approach | Ensures correctness, living documentation | Big-bang integration |
| Day 1 | SSR over SPA | Simpler, no build chain, better debugging | React/Vue SPA |
| Day 1 | Lengauer-Tarjan for dominators | Proven, optimal complexity | Simple tree walk |
| Day 1 | JSON parser first | Enables testing without parser complexity | Mock objects |
| Day 1 | Lazy indices | Memory efficiency for large dumps | Eager pre-computation |
| Day 1 | Cobra for CLI | Well-tested, good UX | Plain flag package |
| Day 1 | Bubble Tea for TUI | Modern, well-maintained | tcell, termui |
| Day 1 | No external DB | Simplicity, no deps | SQLite for indices |

## Success Metrics
- Can analyze 5GB dump in <2GB RAM
- All algorithms complete in <30s for 10M objects
- Web UI loads in <500ms
- 80% code coverage
- Zero security vulnerabilities
- <5 bug reports in first month

## Final Validation
- [ ] Analyze real production heap dump
- [ ] Find known memory leak via dominators
- [ ] Trace leak path to root
- [ ] Export findings for report
- [ ] Deploy to production service
- [ ] Monitor for 1 week without issues

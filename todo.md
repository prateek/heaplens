# HeapLens Development TODO

## Current Sprint: Increment 1 - Skateboard (Basic Graph Analysis)

### In Progress
- [ ] None currently

### Ready
- [ ] None currently

### Blocked
- None

### Completed
- [x] Planning and design phase complete
- [x] Risk assessment documented
- [x] TDD prompts generated
- [x] Step 1: Project Setup ✅
- [x] Step 2: Core data model ✅
- [x] Step 3: Parser registry ✅
- [x] Step 4: JSON parser ✅
- [x] Step 5: Paths algorithm ✅
- [x] Step 6: CLI paths command ✅
- [x] Step 7: Integration tests ✅
- [x] All 4 technical spikes completed ✅

## Upcoming Increments

### Increment 1: Skateboard (Week 1) ✅ COMPLETED
- [x] Step 1: Project setup (1h)
- [x] Step 2: Core data model (2h)
- [x] Step 3: Parser registry (1h)
- [x] Step 4: JSON parser (2h)
- [x] Step 5: Paths algorithm (3h)
- [x] Step 6: CLI paths command (2h)
- [x] Step 7: Integration tests (2h)

### Increment 2: Bicycle (Week 1.5)
- [x] Step 8: Dominators algorithm (4h) ✅
- [ ] Step 9: Retained size (2h)
- [ ] Step 10: Type aggregation (2h)
- [ ] Step 11: CLI framework (1h)
- [ ] Step 12: Top command (1h)
- [ ] Step 13: Dominators command (1h)
- [ ] Step 14: Retained command (1h)

### Increment 3: Motorcycle (Week 2.5)
- [ ] Step 15: Web handler setup (2h)
- [ ] Step 16: Template system (2h)
- [ ] Step 17: Dump listing (2h)
- [ ] Step 18: Top types view (3h)
- [ ] Step 19: Paths view (3h)
- [ ] Step 20: Dominators view (3h)
- [ ] Step 21: Object search (2h)
- [ ] Step 22: Web command (1h)

### Increment 4: Car (Week 4-5)
- [x] Step 23: Go format research (4h) ✅ SPIKE COMPLETED
- [ ] Step 24: Parser structure (3h) - REVISED PLAN:
  - [ ] Phase 1: Core Integration (Day 1)
    - [ ] Integrate with HeapLens Parser interface
    - [ ] Convert to HeapLens graph format
    - [ ] Fix type associations
    - [ ] Extract roots properly
    - [ ] Add format detection
  - [ ] Phase 2: Streaming & Robustness (Day 2)
    - [ ] Implement streaming API with callbacks
    - [ ] Add error recovery
    - [ ] Progress reporting
    - [ ] Bounds checking
  - [ ] Phase 3: Production Hardening (Day 3)
    - [ ] Complete remaining record types
    - [ ] Test Go 1.20-1.24 compatibility
    - [ ] Performance optimization
    - [ ] Unit and integration tests
  - [ ] Phase 4: Fuzz & Property Testing (Day 4)
    - [ ] Go native fuzz tests
    - [ ] Property-based tests for invariants
    - [ ] Differential testing
    - [ ] Build test corpus
    - [ ] Corruption resilience tests
- [ ] Step 25: Object extraction (merged into Phase 1)
- [ ] Step 26: Type mapping (merged into Phase 1)
- [ ] Step 27: Root identification (merged into Phase 1)
- [ ] Step 28: Parser integration (merged into Phase 1)
- [ ] Step 29: TUI framework (2h)
- [ ] Step 30: TUI implementation (4h)
- [ ] Step 31: Performance optimization (merged into Phase 3)
- [ ] Step 32: Memory handling (merged into Phase 2)
- [ ] Step 33: Documentation (2h)

## Spikes Completed ✅

### High Priority (Week 1) ✅
- [x] Go heap format validation spike (2 days) ✅
  - Analyzed runtime/heapdump.go source
  - Created complete format specification
  - Built working parser (heap_parser.go)
  - Successfully parses 290+ objects
- [x] Streaming parse spike (1 day) ✅
  - Validated <0.5x memory usage
  - Proved channel-based streaming works
  - ~1ms per 1000 objects performance

### Medium Priority (Week 2) ✅
- [x] Dominator performance spike (4 hours) ✅
  - Achieved 22s for 10M nodes (close to target)
  - Memory usage ~240MB (acceptable)
  - Confirmed O(E α(E,V)) complexity
- [x] SSR template spike (4 hours) ✅
  - No JS build confirmed
  - Clean UI with HTML/CSS only
  - Embedded templates working
  - Dynamic features via URL params

## Testing Checklist

### Unit Tests
- [ ] Graph algorithms (paths, dominators, retained)
- [ ] Parser interface and registry
- [ ] JSON parser
- [ ] Type aggregation
- [ ] CLI commands
- [ ] Web handlers
- [ ] Template rendering

### Integration Tests
- [ ] JSON dump end-to-end
- [ ] Real Go dump parsing
- [ ] CLI workflow
- [ ] Web UI navigation

### Performance Tests
- [ ] 1M object graph
- [ ] 10M object graph
- [ ] 5GB dump parsing
- [ ] Memory usage under 2x dump size

### E2E Tests
- [ ] Full CLI workflow
- [ ] Web UI with real dump
- [ ] TUI interaction

## Documentation TODO

### User Documentation
- [ ] README with installation
- [ ] CLI usage examples
- [ ] Web UI mounting guide
- [ ] API documentation

### Developer Documentation
- [ ] Architecture overview
- [ ] Parser plugin guide
- [ ] Algorithm explanations
- [ ] Contributing guide

### Examples
- [ ] Basic Web UI mounting
- [ ] Custom parser implementation
- [ ] CLI automation scripts
- [ ] Memory leak detection walkthrough

## Release Checklist

### Alpha Release
- [ ] Core algorithms working
- [ ] JSON parser complete
- [ ] Basic CLI functional
- [ ] Unit tests passing

### Beta Release
- [ ] Web UI complete
- [ ] Go parser working
- [ ] Performance acceptable
- [ ] Integration tests passing

### GA Release
- [ ] TUI implemented
- [ ] Documentation complete
- [ ] Security review done
- [ ] Performance optimized
- [ ] E2E tests passing

## Known Issues
- None yet

## Tech Debt
- None yet

## Future Enhancements (v1.1+)
- [ ] Snapshot diff between dumps
- [ ] CSV/JSON export
- [ ] SVG graph rendering
- [ ] Remote dump analysis
- [ ] Query DSL
- [ ] Duplicate detection

## Notes
- Using TDD approach throughout
- Minimal dependencies (stdlib + optional TUI)
- SSR for Web UI (no JS build)
- Target 5-10GB dumps
- Follow pprof conventions

## Review Gates
- [ ] Code review after each increment
- [ ] Performance review at Week 3
- [ ] Security review before Beta
- [ ] UX review of Web UI
- [ ] API stability review before GA
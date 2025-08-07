# HeapLens Development TODO

## Current Sprint: Increment 1 - Skateboard (Basic Graph Analysis)

### In Progress
- [ ] None currently

### Ready
- [ ] Step 1: Project Setup
  - [ ] Create go.mod with module github.com/prateek/heaplens
  - [ ] Add .gitignore for Go
  - [ ] Create basic README.md
  - [ ] Write TestProjectStructure
  - [ ] Verify go test passes

### Blocked
- None

### Completed
- [x] Planning and design phase complete
- [x] Risk assessment documented
- [x] TDD prompts generated

## Upcoming Increments

### Increment 1: Skateboard (Week 1)
- [ ] Step 1: Project setup (1h)
- [ ] Step 2: Core data model (2h)
- [ ] Step 3: Parser registry (1h)
- [ ] Step 4: JSON parser (2h)
- [ ] Step 5: Paths algorithm (3h)
- [ ] Step 6: CLI paths command (2h)
- [ ] Step 7: Integration tests (2h)

### Increment 2: Bicycle (Week 1.5)
- [ ] Step 8: Dominators algorithm (4h)
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
- [ ] Step 23: Go format research (4h)
- [ ] Step 24: Parser structure (3h)
- [ ] Step 25: Object extraction (4h)
- [ ] Step 26: Type mapping (3h)
- [ ] Step 27: Root identification (2h)
- [ ] Step 28: Parser integration (2h)
- [ ] Step 29: TUI framework (2h)
- [ ] Step 30: TUI implementation (4h)
- [ ] Step 31: Performance optimization (4h)
- [ ] Step 32: Memory handling (3h)
- [ ] Step 33: Documentation (2h)

## Spikes Needed

### High Priority (Week 1)
- [ ] Go heap format validation spike (2 days)
  - Parse dumps from Go 1.22, 1.23
  - Document format differences
  - Create format specification
- [ ] Streaming parse spike (1 day)
  - Test memory-bounded parsing
  - Measure memory usage on 5GB dump

### Medium Priority (Week 2)
- [ ] Dominator performance spike (4 hours)
  - Benchmark on 10M node graph
  - Verify O(E α(E,V)) complexity
- [ ] SSR template spike (4 hours)
  - Build simple template example
  - Verify no JS build needed

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
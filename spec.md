```markdown
# HeapLens — Developer Specification (`spec.md`)

A drop-in **Go library + CLI** to analyze **Go heap dumps** (from `debug.WriteHeapDump`) with:
- an **embeddable Web UI** you can mount **at `/debug/heaplens`** inside your service (pprof-style),
- a **pprof-like CLI** with both **Web** and **TUI** modes,
- an internal **graph engine** that supports **paths-to-roots**, **dominator tree**, and **retained size**.

The design prioritizes: minimal deps, pluggable dump parsers, predictable APIs, and parity with the investigative features engineers actually use from MAT/YourKit.

---

## 1) Goals & Non-Goals

### 1.1 Goals
- Provide a **developer-friendly, drop-in** package offering:
  - `heaplenshttp.Handler(cfg)` to mount a **/debug/heaplens** Web UI.
  - `heaplens` CLI with **web**, **tui**, and **query** subcommands (pprof vibes).
- Support **real Go heap dumps** (from production) and a **JSON stub** for tests/CI.
- Offer **object graph analysis**:
  - **Shortest paths to GC roots** (who is holding it now).
  - **Immediate dominators** and **retained size**.
  - **Top types by live bytes / retained**.
- Be **extensible**: alternative parsers via a simple `Parser` interface.

### 1.2 Non-Goals (v1)
- Not a general JVM/HPROF/JFR tool.
- No heavy SPA build chain; SSR HTML templates + tiny progressive JS only.
- No full OQL-like query language in v1 (simple filters only).
- No multi-process aggregation (analyze one dump at a time).

---

## 2) High-Level Architecture

```

+-------------------+       +-----------------------+        +------------------+
\|  App (your code)  | <---> | heaplenshttp (Web UI) |  uses  |  heaplens/graph  |
\| imports Handler() |       |  /debug/heaplens      | <----> |  algorithms      |
+-------------------+       +----------^------------+        +--------^---------+
\|                              |
\| uses                         | via
v                              |
+-------------------+          +------------------+
\| heapdump registry |  <-----> |  Parser adapters |
\| Open(filename)    |          | (Go dump, JSON)  |
+-------------------+          +------------------+

CLI: cmd/heaplens -> same heapdump + graph + web server (for --http) or TUI

````

**Key packages**
- `heaplenshttp`: HTTP handlers + simple templates.
- `heapdump`: `Parser` interface + registry, `Open()` to return a `graph.Graph`.
- `graph`: object model, indices, algorithms (paths, dominators, retained).
- `tui`: optional Bubble Tea TUI.
- `cmd/heaplens`: CLI entry.

---

## 3) Detailed Requirements

### 3.1 Functional
- Mountable Web UI at **configurable base path** (default **`/debug/heaplens`**).
- Index page listing dump files (by mtime desc).
- Views:
  - **Top Types** (bytes, count; link → type drill-down).
  - **Paths to Roots** (shortest K paths; K configurable).
  - **Dominators** (ranked by retained size; expandable).
  - **Object Search** (by ID, type prefix).
- CLI subcommands:
  - `web <dump>`: serve a single-dump Web UI on a random port; print URL.
  - `tui <dump>`: TUI browser.
  - `top`, `paths`, `dominators`, `retained`: non-interactive queries with text output.
- Parser plugin system with at least:
  - `JSONStub` (tests/CI).
  - `GoHeap` (real Go dump adapter) — v1 target.

### 3.2 Non-Functional
- **Go version**: 1.22+.
- **Deps**: stdlib + optional TUI deps (`bubbletea`, `lipgloss`). No Node build.
- **Performance**: handle dumps up to ~5–10 GB (best effort):
  - streaming parse,
  - memory-bounded graph (compact structs),
  - on-demand indices (lazy).
- **Security**:
  - Web UI is **opt-in** and mountable only in trusted/admin endpoints.
  - No remote upload endpoints in v1 (read-only file listing).
  - Optional **basic auth**/custom middleware hook.
- **Portability**: works on Linux/macOS; no CGO.

---

## 4) Data Model & Algorithms

### 4.1 Graph Model
```go
type ObjID uint64
type Object struct {
  ID       ObjID
  Type     string    // fully-qualified Go type
  Size     uint64    // self bytes
  Pointers []ObjID   // outgoing edges
  Kind     Kind      // String, Slice, Map, Struct, ...
  Meta     map[string]string // optional (pkg, file:line, etc.)
}
type Roots struct {
  Globals []ObjID
  Stacks  []ObjID
  Finals  []ObjID
  Others  []ObjID
}
type Graph interface {
  Objects() ([]Object, error)
  Roots() (Roots, error)
  Index() Index // optional accelerators
}
````

### 4.2 Indices

* Type prefix → objects (cap at N items).
* ID → object lookup.
* Reverse edges built lazily for **paths-to-roots** and **dominators**.

### 4.3 Algorithms

**Shortest K Paths to Roots**

* Build reverse graph on demand.
* BFS from `start` toward any root; collect up to `K` distinct paths.
* Deduplicate via `(node, predecessor)` hashing.

**Immediate Dominators (Lengauer–Tarjan)**

* Create a super-root linking all GC roots.
* Run LT in O(E α(E,V)) typical.
* Store `idom` map for subsequent queries.

**Retained Size**

* Using `idom`, construct dominator tree.
* Aggregate `Size` over subtree for each node.
* Retained(X) = sum of subtree(X) (bytes that would be freed if X were gone).

**Top Types**

* Aggregate by `Type`: sum of `Size`, count of objects.
* Optional: compute *retained by representative dominator* (v1: top dominator nodes separately).

---

## 5) Parser Interface & Adapters

### 5.1 Parser Interface

```go
type Parser interface {
  CanParse(filename string, peek []byte) bool
  Parse(r io.ReaderAt, size int64) (graph.Graph, error)
  Name() string
}
func Register(p Parser)
func Open(filename string) (graph.Graph, Parser, error)
```

### 5.2 JSON Stub (ships in v1)

* Input JSON:

```json
{
  "objects": [
    {"id": 1, "type": "mypkg.Node", "size": 64, "pointers": [2,3]},
    {"id": 2, "type": "[]byte", "size": 1024, "pointers": []}
  ],
  "roots": {"globals":[1]}
}
```

* Use for unit tests and CI golden files.

### 5.3 Go Heap Adapter (v1 target)

* Parse `debug.WriteHeapDump` format:

  * Stream sections, build nodes, edges, and root sets.
  * Minimize allocations (sync.Pool internally is fine; it’s offline).
* Map Go strings/slices/maps to `Kind`.
* Optionally include `Meta` if source info present.

---

## 6) Web UI

### 6.1 Mounting (at `/debug/heaplens`)

```go
mux.Handle("/debug/heaplens/", heaplenshttp.Handler(heaplenshttp.Config{
  BasePath: "/debug/heaplens",
  DumpDir:  "/var/lib/myservice/heapdumps",
  Auth:     myAuthMiddleware, // optional
}))
```

### 6.2 Routes (SSR)

* `GET /debug/heaplens/` → list dumps (mtime, size).
* `GET /debug/heaplens/view?file=...&v=top`
* `GET /debug/heaplens/view?file=...&v=paths&id=<obj>&k=5`
* `GET /debug/heaplens/view?file=...&v=dominators`
* `GET /debug/heaplens/view?file=...&v=search&q=<typePrefix>`

### 6.3 UI Elements

* **Top Types** table (sortable: bytes, count).
* **Paths View**: breadcrumbs of IDs (click → object page); optional SVG render of K paths.
* **Dominators**: list of top retained; expand to show dominated children; show retained bytes.
* **Search**: type prefix; jump to object by ID.

### 6.4 No JS Build Required

* Server-rendered templates; tiny inline script for toggles (no bundler).
* Copy-linkable URLs (useful for bug reports).

---

## 7) CLI & TUI

### 7.1 CLI Commands

```
heaplens web <dump> [--http=:0] [--open]
heaplens tui <dump>
heaplens top <dump> [-n=20] [--by=bytes|count]
heaplens paths <dump> --id=<objID> [-k=5]
heaplens dominators <dump> [-n=20]
heaplens retained <dump> --ids=0xc000...[,0xc000...]
```

**Output examples**

* `top`: columns: TYPE | OBJECTS | BYTES.
* `dominators`: OBJID | TYPE | RETAINED\_BYTES | SELF\_BYTES.
* `paths`: prints up to K paths, each as `Root → ... → obj`.

### 7.2 TUI (optional dep)

* Panels: **Top** (left), **Details** (right).
* Keys: `p` (paths), `d` (dominators), `/` (filter), `#` (jump ID), `W` (launch web), `q`.

---

## 8) Configuration

* **Web**:

  * `BasePath` (default: **`/debug/heaplens`**).
  * `DumpDir` (required).
  * `Auth` (func(http.Handler) http.Handler) optional.
  * `MaxPaths` (default 5).
* **CLI**:

  * `--http` (default `:0` → random port).
  * `--open` → try `xdg-open`/`open`.
* **Parser**:

  * Parsers must be registered by the embedder or CLI `init()`.

---

## 9) Error Handling Strategy

* **Parser errors**:

  * If magic/format mismatch → `ErrUnsupportedFormat` (actionable).
  * If corrupted → `ErrCorruptDump` with offset context.
  * Surface `Parser.Name()` in errors.
* **Memory pressure**:

  * Detect OOM risk via size heuristic; fall back to **paged/segmented** loading (TBD in v1.1).
  * If reverse graph too big, cap K paths and warn in UI (“Truncated search at N edges”).
* **Web UI**:

  * 4xx for user errors (missing `file`, bad `id`).
  * 5xx with short message; log full cause.
* **TUI/CLI**: non-zero exit codes; stderr friendly messages.

---

## 10) Observability

* Optional logger interface; default to `log.Printf`.
* Timers around parse, index build, algorithms:

  * `parse_ms`, `index_ms`, `paths_ms`, `dominators_ms`.
* Web UI footer shows timings + peak RAM (from `runtime.ReadMemStats`).

---

## 11) Security Considerations

* **Local file access only** (no upload).
* Recommend mounting **`/debug/heaplens`** behind auth (basic auth or your admin gate).
* Escape all HTML, enforce path base (no `..` traversal).
* Don’t expose dump filenames in query params beyond basename.

---

## 12) Performance Plan

* Compact object struct (avoid maps for hot paths; slice+arena).
* Build reverse edges lazily and cache; evict on dump switch.
* Dominators:

  * Use uint32 if ID cardinality < 2^32 (optional build tag).
* For >10M objects:

  * Disable retained by default; enable via flag.
  * Stream “Top Types” by partial aggregation chunks.

---

## 13) Testing Plan

### 13.1 Unit

* **Graph**:

  * BFS paths on small synthetic graphs (goldens).
  * Dominators on canonical diagrams (goldens).
  * Retained aggregation vs hand-computed results.
* **Parser(JSON)**:

  * Edge cases: cycles, big IDs, no roots.
* **Web handlers**:

  * Table render, query params, 400/500 paths.

### 13.2 Integration

* Generate **real dumps** in a test binary:

  * Allocate graphs (maps/slices), call `runtime.GC(); debug.WriteHeapDump`.
  * Validate that known retainers appear in paths/dominators.
* CLI smoke tests:

  * `heaplens top`, `paths`, `dominators` against fixtures → compare normalized output.

### 13.3 Performance

* Synthetic generator to produce N objects, E edges:

  * Benchmark parse, index, dominators, retained.

### 13.4 Compatibility

* Dumps from Go 1.20–1.23 (build a matrix if format stable; otherwise document supported versions).

---

## 14) Developer Experience / Ergonomics

### 14.1 In-App Usage (at `/debug/heaplens`)

```go
import "github.com/yourorg/heaplens/heaplenshttp"
import "github.com/yourorg/heaplens/heapdump"

func init() {
  heapdump.Register(heapdump.JSONStub{})
  heapdump.Register(heapdump.GoHeap{}) // when ready
}

mux.Handle("/debug/heaplens/", heaplenshttp.Handler(heaplenshttp.Config{
  BasePath: "/debug/heaplens",
  DumpDir:  "./dumps",
}))
```

### 14.2 CLI

```bash
go install github.com/yourorg/heaplens/cmd/heaplens@latest
heaplens web ./dumps/heap-2025-08-07T18-12-03.dump --open
```

---

## 15) Deliverables & Milestones

**M1 (Core Graph + JSON parser + CLI text) — 1.5–2 weeks**

* `graph` package (paths, dominators, retained).
* `heapdump` registry + JSONStub.
* CLI: `top`, `paths`, `dominators`, `retained`.
* Unit tests + goldens.

**M2 (Web UI SSR + Embeddable Handler @ `/debug/heaplens`) — 1 week**

* `/debug/heaplens` pages (top/paths/dominators/search).
* Config, auth hook, listing dumps.
* Integration tests.

**M3 (Go Heap Adapter v1) — 2–3 weeks**

* Streaming parse of real dumps.
* Correct roots, kinds, types.
* Compatibility tests across Go versions.

**M4 (TUI + Perf polish) — 1 week**

* Bubble Tea TUI.
* Timings, memory caps, warnings.

**Stretch (v1.1)**

* Snapshot diff, CSV/JSON export, SVG retained graph rendering.

---

## 16) Acceptance Criteria (DoD)

* Given a real heap dump from a sample app, **Web UI** shows:

  * Top Types with correct byte totals (±0.5%).
  * Paths to roots for a known leaked object (K≥3).
  * Dominators with a leak container at the top by retained bytes.
* **CLI** returns non-interactive text results matching goldens.
* **API** stable: `heaplenshttp.Handler`, `heapdump.Open`, `graph` types.
* Works on Linux/macOS; Go 1.22+; no CGO required.

---

## 17) Risks & Mitigations

* **Dump format drift** → Pin supported Go versions; gate adapter with `CanParse`. Add CI with known dumps.
* **Memory blow-ups** → Streaming parse + lazy indices; cap path search depth; warn in UI.
* **Security exposure** → Provide auth hook; document “admin-only” deployment; disable listing by default if `DumpDir` unset.
* **Parser complexity** → Start with JSON + wrap an existing parser if available; ship adapter iteratively.

---

## 18) Example Developer Hooks

**SIGUSR1 heap dump writer (in your app)**

```go
// Typical pattern: call runtime.GC() then debug.WriteHeapDump(fd)
```
Linking from incidents
- Paste `/debug/heaplens/view?file=heap-<ts>.dump&v=dominators` into tickets.
- Export `top` as CSV (`--csv`) for Slack/PR comments (stretch).

--- 

## 19)  Licensing & Contribution
License: Apache-2.0.

Contrib:
- Stable interfaces in `graph`, `heapdump`.
- Parser adapters live under `heapdump/`.

PR checklist: unit tests, docs, benchmarks (if perf-sensitive).


--- 

## 20) Future work
- Diffing: retained deltas between two dumps.
- Duplicate strings/byte slices report.
- Simple query DSL (type:prefix, size>n, dominates(x)).
- Remote agent mode (serve dump over HTTP for offline analysis).

---

## Appendix A — Minimal Code Stubs
Algorithms entrypoints

```go
func PathsToRoots(g Graph, start ObjID, k int) ([][]ObjID, error)
func Dominators(g Graph) (map[ObjID]ObjID, error)
func RetainedSize(g Graph, idom map[ObjID]ObjID, x ObjID) (uint64, error)
```

---

## CLI glue

```bash
heaplens top --by=bytes dump
heaplens paths --id=0xc000123456 dump
heaplens dominators -n 20 dump
heaplens web dump --open
```

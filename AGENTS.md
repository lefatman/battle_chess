

## CORE (applies to every task)
- **Output:** Each file starts with a path header (e.g., `// path: pkg/file.go`) if one is not present append it if it is incorrect, correct it. No prose.
- **Stability:** Preserve provided public APIs, wire formats, routes, schemas (drop-in).
- **Perf:** Stdlib-first; zero/low allocations in hot paths; pre-size buffers; cache-friendly (SoA where apt).
- **Data plane:** Binary/compact for hot paths; JSON only for admin/debug.
- **Concurrency:** Bounded queues, backpressure, deadlines/timeouts, clean shutdown; no goroutine/thread per message.
- **Design:** Composition > inheritance; small interfaces/protocols; immutable value objects for data.
- **Determinism & Errors:** Pure core transforms; side-effects at boundaries; typed/sentinel errors; wrap with context.
- **Security:** Validate inputs, escape/encode outputs, constant-time where relevant, CSP-safe.
- **Priority:** Correctness → Safety → Performance → Minimalism.
## SHORTCODES INDEX

### Go
- **[GO:SYSNET]** WebSockets/binary protocols/backpressure
  - Use stdlib (net/http, bufio, encoding/binary, sync, time). Manual RFC6455 if needed.
  - One reader + one writer per conn; **SPSC ring (pow2 mask)**; reliable send deadline ≤ **100ms**; coalesce deltas; heartbeat PING 20s / drop @30s.
  - Zero allocs in hot paths; `sync.Pool` for buffers; no `fmt`/maps/strings in inner loops.

- **[GO:DB]** SQLite/Postgres OLTP with OCC
  - `database/sql` + prepared statements; SQLite WAL + `busy_timeout`; PG `INSERT … ON CONFLICT … DO UPDATE`.
  - OCC: `UPDATE … WHERE id=? AND version=?`; check rowcount. Explicit transactions, short-lived; **no `SELECT *`**.

- **[GO:SIM]** Realtime sim / chess / AOI
  - 20 Hz fixed tick; **SoA** arrays (`IDs, X, Y, VX, VY, Flags`); ≤128 visible entities; Manhattan sort; **no GC in tick**.
  - Chess: 12 bitboards + occupancy; Zobrist; make/unmake with undo stack; pre-cap move slices.

- **[GO:WEB]** SSR + HTTP hardening
  - `net/http` + `html/template`; strict headers (CSP/COOP/COEP); context timeouts; size limits; no globals.

### Python
- **[PY:SYS]** Binary/parsers/CLI (stdlib-only)
  - `struct`, `array`, `memoryview/bytearray`, `itertools/heapq/bisect`. Pooled buffers; zero per-iteration allocs; branchless where readable.

- **[PY:DS]** Data science, columnar, vectorized
  - Prefer **Polars + PyArrow** (or **DuckDB**); if pandas mandated → vectorized only (no `iterrows/itertuples/apply` in hot paths).
  - Explicit dtypes, no object dtype for numerics; chunked IO; memory-map large files.

- **[PY:ML]** Numerics/tensors
  - If allowed → NumPy/Numba/array-API; else fixed-stride loops over `memoryview` with preallocation; no per-step object churn.

### JavaScript (Browser, no frameworks)
- **[JS:CANVAS]** Canvas/OffscreenCanvas render loop
  - `requestAnimationFrame`; **dirty-rect/batched draws**; atlas sprites; TypedArrays/DataView; no per-frame allocs.

- **[JS:WORKER]** Workers + SAB/Atomics
  - Versioned message schema; Transferables; SharedArrayBuffer ring; no cloning MB objects; bounds checks.

- **[JS:WS]** Binary WS client
  - u8 opcode + length + payload; coalesce deltas; bounded queues; PING 20s / drop @30s; exponential backoff reconnect with jitter.

- **[JS:UI]** Minimal DOM/UI
  - Event delegation; batch read→compute→write; CSP-safe (no eval/inline handlers); accessible focus.

### HTML / HTMX
- **[HTML:SSR]** Semantic templates (Go templates allowed)
  - Landmarks (header/nav/main/footer); one `<h1>`; progressive enhancement; IDs/data-hooks; CSP-safe; images sized to avoid CLS.

- **[HTMX:FRAG]** SSR fragments with hx-*
  - Define GET/POST partial contracts; hx-target/swap; 422 partials for validation; works without JS baseline.

### CSS
- **[CSS:PERF]** Tokens, layers, minimal specificity
  - CSS variables for tokens; `@layer` reset/base/components/utilities; selector depth ≤3; transform/opacity only; reduced-motion variants.

### SQL (SQLite/Postgres)
- **[SQL:OLTP]** CRUD paths
  - Parameterized DML; covering indexes for hot queries; explicit columns; short TX; statement cache; keyset pagination.

- **[SQL:TS]** Time-series/analytics
  - PG: partitions by time; local indexes; MV rollups + scheduled refresh.
  - SQLite: segment tables; batched inserts in TX; wal_checkpoint(TRUNCATE); retention jobs.


---

## ACCEPTANCE CHECKS (fill in numbers per task)
- p95 latency ≤ **{N}ms**; **zero allocs** in hot loops `{files}`; race-free; WS reliable ≤ **100ms**.
- Frame budget ≤ **16.6ms** @ **{res}** with GC-quiet frames.
- DB: all hot queries indexed; no seq scan on tables > **{X}** rows unless intended.
- A11y: focus/labels ok; CSP passes; LCP element sized.

## HOW TO USE (prompt shape)
Task: {what to build/refactor}
APIs to preserve: {signatures/types/routes}
Shortcodes: [GO:SYSNET][JS:CANVAS][SQL:OLTP] # <-- reference sections below
Deliverables: {exact paths to emit}
Acceptance: {numerical targets}

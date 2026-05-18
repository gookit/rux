# Changelog

## v2.0.0 — 2026-05-18 (Breaking Changes)

Clean-room rewrite focused on extreme performance, with new built-in
modules for production server hosting, response rendering, and
Server-Sent Events. See [docs/MIGRATION-v1-to-v2.md](docs/MIGRATION-v1-to-v2.md)
for the complete v1 → v2 breaking-change list.

### Module path

- `github.com/gookit/rux/v2` (semver requires the `/v2` suffix for v2+).

### Added — core router

- Per-method radix tree (`[9]*radixTree`) and per-method static map
  (`[9]map[path]*Route`) — no string concat per request
- Inline `Params [16]Param` in `Context` for zero-allocation parameter
  passing on the hot path
- `Router.Freeze()` for explicit read-only mode; auto-triggered on the
  first `ServeHTTP` call
- Lock-free hot path (no mutex on serving)
- HEAD requests automatically mirror GET routes at freeze time
- Pre-merged middleware chains (no per-request `append`)
- Cookie helpers on `Context`: `Cookie` / `SetCookie` / `FastSetCookie` /
  `DelCookie`
- `FastSetCookie(name, value, maxAge, opts ...func(*http.Cookie))` —
  variadic option callbacks let HTTPS callers flip Secure / SameSite /
  any other cookie field without falling back to the long `SetCookie`
  signature
- `Context.Bind` / `ShouldBind` / `MustBind` / `AutoBind` /
  `BindForm` / `BindJSON` / `BindXML`
- Routes dumped on startup when running in debug mode (via `Server.Run`)

### Added — `server/` package

Production-ready HTTP server wrapping a `Router`:

- Sane defaults: `ReadHeaderTimeout=2s`, `ReadTimeout=10s`,
  `WriteTimeout=30s`, `IdleTimeout=120s`, `ShutdownTimeout=25s`,
  `DrainDelay=5s`, `MaxHeaderBytes=1 MiB`
- Lifecycle hooks: `PreStart` / `PostStart` / `PreShutdown` /
  `PostShutdown` (each a `[]func(ctx) error`)
- Graceful shutdown via `SIGINT` / `SIGTERM` (configurable
  `StopSignals`) or programmatic `Stop()`
- Drain mode flips `/readyz` to 503 → waits `DrainDelay` → bounded
  `http.Server.Shutdown` within `ShutdownTimeout`
- `MountHealthChecks()` adds `GET /healthz` (liveness) + `GET /readyz`
  (readiness, supports custom `ReadyChecks`)
- Optional TLS via `TLSCertFile` / `TLSKeyFile`
- TLS-less echo / debug server pre-mounted: `server.NewEchoServer()`

### Added — `server/` echo server (httpbin-style)

Single entry-point HTTP debug server, useful for client integration
tests and `/debug` subtrees in real apps:

| Method | Path | Behavior |
|---|---|---|
| GET    | `/` | HTML index of all endpoints |
| ANY    | `/anything[/*path]` | Echo full request (method, headers, query, form, JSON) |
| GET    | `/get`, POST `/post`, PUT `/put`, PATCH `/patch`, DELETE `/delete` | Method-locked; wrong verb → 405 with `Allow` |
| GET    | `/headers`, `/ip`, `/user-agent` | Inspection helpers |
| ANY    | `/status/{code}` | Reply with given status |
| GET    | `/delay/{seconds}` | Sleep then echo (capped at 10s) |
| GET    | `/redirect/{n}` | Countdown 302 chain |
| GET    | `/cookies[/set/{name}/{value}]` | Cookie inspect / set |
| GET    | `/basic-auth/{user}/{passwd}` | Basic-auth check |
| GET    | `/bytes/{n}` | N random bytes (capped at 100 KB) |
| GET    | `/uuid` | RFC 4122 v4 UUID |
| GET    | `/download/{filename}` | Synthesize content (`?size=`, `?type=bin\|text\|json`, `?inline=1`) |
| POST   | `/upload` | multipart upload, replies with per-file `{filename,size,mime,sha256}` |
| ANY    | `/*path` | Catch-all echo for unmatched paths |

Mount into an existing router via `server.MountEchoRoutes(r)`.

### Added — `pkg/render` package

Stateless helpers + a `Responder` aggregator that absorbs the
[gookit/respond](https://github.com/gookit/respond) API surface without
pulling in `easytpl`:

- Stateless: `JSON` / `JSONIndented` / `JSONP` / `XML` / `XMLPretty` /
  `Text` / `Plain` / `HTML` / `HTMLBytes` / `TextBytes` / `Blob` / `Auto`
- `Responder` type with status-aware methods: `JSON / XML / Text /
  JSONP / HTML / HTMLString / HTMLText / Binary / Content / NoContent /
  Auto`, plus `Options{JSONIndent, JSONPrefix, XMLIndent, XMLPrefix,
  Charset, AddCharset, ContentType}`
- `TemplateRenderer` interface — engine-agnostic; plug in any HTML
  template engine (easytpl, std `html/template`, etc.) via
  `SetTemplateRenderer`. Optional `TemplateLoader` sub-interface for
  `LoadGlob` / `LoadFiles`
- Package-level proxies on a default Responder: `JSONStatus`,
  `XMLStatus`, `TextStatus`, `HTMLStatus`, `HTMLStringStatus`,
  `HTMLTextStatus`, `JSONPStatus`, `BinaryStatus`, `ContentStatus`,
  `EmptyStatus`, `AutoStatus`

### Added — `pkg/sse` package

Server-Sent Events helper with lifecycle hooks and a keyed-push Hub:

- `sse.Stream(c, hooks, producer)` — common-case entry; emits a
  default `: connected\n\n` comment frame
- `sse.StreamWith(c, opts, producer)` — full options surface:
  `SendConnected` (default true), `KeepaliveInterval` (background
  `: keepalive\n\n` ticker)
- `sse.Hooks{OnConnect, OnDisconnect, OnSend, OnError}` — all
  nil-safe; `OnConnect` runs before SSE headers, so it can write a
  custom 4xx response (e.g. `http.Error(c.Resp, ..., 401)`)
- `sse.Event{ID, Name, Data, Retry}` with spec-compliant multi-line
  data encoding
- `sse.NewHub(bufSize)` — in-memory registry keyed by user-supplied ID,
  multi-connection-per-id (multi-tab fan-out), non-blocking
  per-client buffer with `dropped` counter, optional `SetOnDrop` hook,
  `Send` / `Broadcast` / `Count` / `IDs` / `Has`
- `sse.HubProducer(hub, id)` — drops in as the producer for `Stream`,
  handles register / drain / unregister

### Changed

- `Match` returns `(*Route, []Param, bool)` instead of `*MatchResult`
- `Route.handler` + `Route.handlers` unified into `Route.chain`
- `Use()` must be called before any route registration (panics otherwise)
- Static routes stored in `[9]map[path]*Route`
- Implementation moved to `internal/core/`; root `rux` package is a
  public-API shim (type aliases + helpers)
- Echo server method-locked endpoints (`/get`, `/post`, `/put`,
  `/patch`, `/delete`) now return **405 with `Allow` header** on wrong
  verb, matching httpbin semantics, instead of falling through to the
  `/*path` catch-all

### Removed

- Regex parameter support `{id:\d+}` — use validation middleware
  inside handlers instead
- `MatchResult`, `QuickMatch`, `ReleaseMatchResult`
- `EnableCaching`, `MaxNumCaches` (LRU cache)
- `fastrux/` subpackage (folded into the main package)
- `pkg/websocket` stub (no implementation was ever provided; use
  `github.com/gorilla/websocket` or similar directly with rux's
  hijack support)
- `_examples/proxyreq` (was a textbook SSRF demo)
- `_examples/fastrux` (folded into main router)

### Fixed

- `pkg/render.Auto` had a fall-through bug on `Accept: application/xml`
  (empty `case MIMEXML:` body silently returned "not supported"); now
  combined with `MIMEXML2`
- `Responder.Content` set `Content-Type` after `WriteHeader`, so the
  header was silently lost upstream (fixed during the `respond` merge)
- `Responder.Data` ignored its `contentType` parameter (fixed during
  the merge)
- CodeQL alerts addressed before release:
  - `/user/{id}` benchmark handlers (gin / gorilla-mux / chi) set
    `Content-Type: text/plain` + `X-Content-Type-Options: nosniff`
    so reflected ids can't trigger browser sniffing → XSS
  - Echo server `make([]byte, n)` sites centralized through
    `clampSize` so the upper bound is visible to static analyzers
  - `FastSetCookie` Secure=false documented as intentional (HTTP
    dev-friendly default); HTTPS callers steered to the new variadic
    opts
  - SSRF example program deleted (see Removed)

### Performance (vs v1.x — measured)

Measured under Docker `golang:1.25` (linux/amd64) on AMD Ryzen 7 5800H,
5 iterations per benchmark. Both versions exercise `ServeHTTP` end-to-end.

| Scenario          | v1.x ns/op | v2 ns/op | speedup | v1 allocs/op | v2 allocs/op |
|-------------------|-----------:|---------:|--------:|-------------:|-------------:|
| Static route      |        863 |       82 |  ~10.5× |            4 |        **0** |
| + 2 middleware    |        786 |      101 |   ~7.8× |            5 |        **0** |
| 5-param dynamic   |       1531 |      254 |   ~6.0× |            7 |        **0** |
| Wildcard (`Any`)  |        715 |       86 |   ~8.4× |            4 |        **0** |
| 404 (1 route)     |       1194 |       91 |  ~13.1× |            0 |            0 |
| 404 (8 routes)    |       1581 |      122 |  ~13.0× |            3 |        **0** |

Geomean latency: **−89.4%** (≈9.4× speedup). All routes that previously
allocated now serve with **zero heap allocations** on the hot path
(Context is reused via `sync.Pool`; params live inline in Context).

Higher throughput is achievable on bare-metal Linux without the Docker
virtualization layer; the numbers above use containerized measurements
because that's the most reproducible setup.

See `_benchmarks/v1-vs-v2-benchstat.txt` for the full benchstat output
(p-values, sample counts, reproduction commands).

### Test coverage

| Package | Coverage |
|---|---:|
| `internal/util` | 100.0% |
| `pkg/pprof` | 100.0% |
| `internal/core` | 95.4% |
| `pkg/binding` | 92.9% |
| `pkg/render` | 90.1% |
| `server` | 87.0% |
| `pkg/sse` | 82.5% |

### Documentation

- `docs/MIGRATION-v1-to-v2.md` — complete breaking-change list
- `docs/echo-server.md` — endpoint catalogue + curl recipes for the
  echo server
- `_examples/echo-server/`, `_examples/sse-server/`, `_examples/graceful/`,
  `_examples/multiport/`, `_examples/pprof/`, `_examples/serve/`,
  `_examples/simplebench/` — full runnable examples
- README & README.zh-CN restructured: "Routing & request handling" +
  "Built-in batteries" sections enumerate the new modules

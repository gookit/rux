# Changelog

## v2.0.0 — 2026-05-16 (Breaking Changes)

Clean-room rewrite focused on extreme performance.
See [docs/MIGRATION-v1-to-v2.md](docs/MIGRATION-v1-to-v2.md) for the full
breaking-change list.

### Added

- Per-method radix tree (`[9]*radixTree`) and per-method static map (`[9]map[...]`)
- Inline `Params [16]Param` in Context for zero-allocation parameter passing
- `Router.Freeze()` for explicit read-only mode (auto-triggered on first ServeHTTP)
- Lock-free hot path (no mutex on serving)
- HEAD requests automatically mirror GET routes at freeze time
- Pre-merged middleware chains (no per-request `append`)

### Changed

- `Match` returns `(*Route, []Param, bool)` instead of `*MatchResult`
- `Route.handler` + `Route.handlers` unified into `Route.chain`
- `Use()` must be called before any route registration (panics otherwise)
- Static routes stored in `[9]map[path]*Route` (no string concat per request)
- Implementation moved to `internal/core/`; root `rux` package is a public-API shim

### Removed

- Regex parameter support `{id:\d+}` (use validation middleware instead)
- `MatchResult`, `QuickMatch`, `ReleaseMatchResult`
- `EnableCaching`, `MaxNumCaches` LRU cache
- `fastrux/` subpackage (functionality merged into main package)

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
allocated now serve with **zero heap allocations** on the hot path (Context
is reused via `sync.Pool`; params live inline in Context).

Higher throughput is achievable on bare-metal Linux without the Docker
virtualization layer; the numbers above use containerized measurements
because that's the most reproducible setup.

See `_benchmarks/v1-vs-v2-benchstat.txt` for the full benchstat output
(p-values, sample counts, reproduction commands).

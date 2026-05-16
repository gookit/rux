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

### Performance (vs v1.x)

- Static route: ~30M ops/s, 0 allocs/op (was ~5M ops/s, multiple allocs)
- Single param dynamic: ~15M ops/s, 0 allocs/op (was <5M ops/s)
- 5 params dynamic: ~10M ops/s, 0 allocs/op

See `_benchmarks/v2-results.txt` for measured numbers from this branch.

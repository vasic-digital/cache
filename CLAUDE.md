# CLAUDE.md - Cache Module

## INHERITED FROM the Helix Constitution

This module is governed by the Helix Constitution. All rules in the
constitution's `CLAUDE.md` and the `Constitution.md` it references apply
unconditionally. Locate the constitution from any nested depth via its
`find_constitution.sh` helper — do NOT hardcode a path (this module stays
fully decoupled and project-agnostic per §11.4.28).

Canonical reference: https://github.com/HelixDevelopment/HelixConstitution

## Overview

`digital.vasic.cache` is a standalone, generic, reusable Go cache module providing core cache interfaces, Redis and in-memory backends, distributed cache patterns, and TTL/eviction policies.

**Module**: `digital.vasic.cache` (Go 1.24.0)

## Packages

- `pkg/cache` - Core `Cache` interface, `TypedCache[T]` generic wrapper, `Config`, `Stats`, `EvictionPolicy` enum
- `pkg/redis` - Redis cache adapter (`Client`, `ClusterClient`) using go-redis/v9
- `pkg/memory` - In-memory cache with LRU/LFU/FIFO eviction, max entries, max memory, background cleanup
- `pkg/distributed` - `ConsistentHash`, `TwoLevel` (L1+L2), `WriteThrough`, `WriteBack`, `CacheAside` strategies
- `pkg/policy` - `FixedTTL`, `SlidingTTL`, `AdaptiveTTL`, `CapacityEviction`, `AgeEviction`, `FrequencyEviction`, `CompositeEviction`
- `pkg/service` - Service-layer caching patterns (cache-aside, read-through, TTL invalidation) that wrap a backend to keep caching logic out of business code

## Build & Test

```bash
go test ./... -count=1 -race    # All tests with race detection
go test ./... -v                 # Verbose output
go test -bench=. ./...           # Benchmarks
```

### Acceptance demo for this module

```bash
# TTL + LRU eviction + concurrency safety on in-memory + Redis backends
cd Cache && GOMAXPROCS=2 nice -n 19 go test -count=1 -race -v ./tests/integration/...
```
Expect: PASS; exercises `memory.New`, `redis.New`, `distributed.NewTwoLevel`, and the `service` layer (cache-aside, read-through) per `Cache/README.md`. For live Redis set `REDIS_ADDR=localhost:6379`.

## Code Style

- Standard Go conventions, `gofmt` formatting
- Imports: stdlib, third-party, internal (blank-line separated)
- Table-driven tests with `testify`
- Line length <= 100 chars
- `context.Context` first parameter
- Error wrapping with `fmt.Errorf("...: %w", err)`

## Dependencies

- `github.com/redis/go-redis/v9` - Redis client
- `github.com/stretchr/testify` - Test assertions (test only)

## No External Dependencies

This module has ZERO dependencies on the consuming project or any project-specific code. It is fully generic and reusable.

## Integration Seams

| Direction | Sibling modules |
|-----------|-----------------|
| Upstream (this module imports) | none |
| Downstream (these import this module) | HelixLLM |

*Siblings* means other project-owned modules at the consuming project's repo root. The consuming project's root app and external systems are not listed here — the list above is intentionally scoped to module-to-module seams, because drift *between* sibling modules is where the "tests pass, product broken" class of bug most often lives. See root `CLAUDE.md` for the rules that keep these seams contract-tested.

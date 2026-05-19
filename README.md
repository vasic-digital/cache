# Cache

A standalone, generic Go cache module providing core interfaces, multiple backends (in-memory, Redis, PostgreSQL), distributed patterns (consistent hashing, two-level cache, write-through / write-back / cache-aside), and configurable TTL + eviction policies.

**Module ID**: `digital.vasic.cache` — Go 1.25+ (also tested on 1.26 inner)
**Status**: production-ready library. Zero own-org dependencies (CONST-054 manifest: `helix-deps.yaml`).
**Authority**: governed by HelixConstitution (cascaded — see `CONSTITUTION.md` + `CLAUDE.md` + `AGENTS.md` at this submodule's root).

---

## Features

- **Multiple backends**: in-memory (LRU/LFU/FIFO), Redis (`go-redis/v9`), PostgreSQL (`pgx/v5`), and distributed two-level (L1 local + L2 remote).
- **Generic typed wrapper**: `cache.TypedCache[T any]` for ergonomic JSON-serialized storage.
- **Policy toolkit**: `FixedTTL`, `SlidingTTL`, `AdaptiveTTL`, `CapacityEviction`, `AgeEviction`, `FrequencyEviction`, `CompositeEviction`.
- **Distribution strategies**: `ConsistentHash` (virtual nodes), `TwoLevel` (read-through L1→L2), `WriteThrough`, `WriteBack`, `CacheAside`.
- **Service-layer wrappers**: `pkg/service` keeps caching out of business code (cache-aside, read-through, TTL invalidation).
- **Stats**: built-in hit/miss/eviction counters with `HitRate()`.
- **Race-detector clean** + bilingual round-trip Challenge (round 242).

---

## Installation

```bash
go get digital.vasic.cache
```

---

## Quickstart

```go
package main

import (
    "context"
    "fmt"
    "time"

    "digital.vasic.cache/pkg/cache"
    "digital.vasic.cache/pkg/memory"
)

func main() {
    ctx := context.Background()

    mc := memory.New(&memory.Config{
        MaxEntries:      10_000,
        MaxMemoryBytes:  100 * 1024 * 1024, // 100 MB
        DefaultTTL:      5 * time.Minute,
        CleanupInterval: time.Minute,
        EvictionPolicy:  cache.LRU,
    })
    defer mc.Close()

    _ = mc.Set(ctx, "greeting", []byte("hello"), 0)
    v, _ := mc.Get(ctx, "greeting")
    fmt.Printf("got %q (hits=%d misses=%d)\n", string(v), mc.Stats().Hits, mc.Stats().Misses)
}
```

---

## API Surface

### `pkg/cache` — Core Interfaces

```go
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    Close() error
}

// Generic typed wrapper (JSON-serialized).
tc := cache.NewTypedCache[MyStruct](underlyingCache)
val, found, err := tc.Get(ctx, "key")
```

`EvictionPolicy` enum: `LRU`, `LFU`, `FIFO`. `Stats` struct exposes `Hits`, `Misses`, `Evictions`, `Size` + `HitRate()`.

### `pkg/redis` — Redis Backend

```go
client := redis.New(&redis.Config{
    Addr:     "localhost:6379",
    Password: "secret",
    DB:       0,
    PoolSize: 10,
})
defer client.Close()
_ = client.Set(ctx, "key", []byte("value"), 5*time.Minute)
data, _ := client.Get(ctx, "key")
_ = client.HealthCheck(ctx)
```

### `pkg/memory` — In-Memory Backend

```go
mc := memory.New(&memory.Config{
    MaxEntries:      10_000,
    MaxMemoryBytes:  100 * 1024 * 1024,
    DefaultTTL:      5 * time.Minute,
    CleanupInterval: time.Minute,
    EvictionPolicy:  cache.LRU,
})
```

### `pkg/postgres` — PostgreSQL Backend (`pgx/v5`)

Schema-backed durable cache; opt-in via `scripts/run-postgres-test.sh` for integration tests.

### `pkg/distributed` — Distribution Strategies

```go
ch := distributed.NewConsistentHash(100) // virtual-node ring
ch.AddNode("redis-1"); ch.AddNode("redis-2")
node := ch.GetNode("my-key")

tl := distributed.NewTwoLevel(localCache, remoteCache, 5*time.Minute)
wt := distributed.NewWriteThrough(cacheInstance, dataSource)
wb := distributed.NewWriteBack(cacheInstance, dataSource)
ca := distributed.NewCacheAside(cacheInstance, dataSource)
```

### `pkg/policy` — TTL + Eviction Policies

```go
fixed := policy.NewFixedTTL(5 * time.Minute)
sliding := policy.NewSlidingTTL(10 * time.Minute)
adaptive := policy.NewAdaptiveTTL(time.Minute, time.Hour)

cap := policy.NewCapacityEviction(0.9)   // evict at 90% full
age := policy.NewAgeEviction(24 * time.Hour)
freq := policy.NewFrequencyEviction(5)
composite := policy.NewCompositeEviction(cap, age)
```

### `pkg/service` — Service-Layer Wrappers

Cache-aside, read-through, and TTL-invalidation patterns wrapping any `cache.Cache` instance — keeps caching logic out of business code.

---

## Environment Variables

The library has zero hardcoded hosts (CONST-045). Backends accept config via constructor parameters. The integration / Challenge suite respects:

| Env var          | Purpose                                                                |
|------------------|------------------------------------------------------------------------|
| `REDIS_ADDR`     | Live Redis address for opt-in integration tests (e.g. `localhost:6379`).|
| `POSTGRES_URL`   | Live PostgreSQL DSN for `pkg/postgres/integration_test.go` (build-tagged). |
| `CACHE_LOG_LEVEL`| Log verbosity for diagnostic builds (`debug`, `info`, `warn`, `error`). |

When `REDIS_ADDR` is unset the Redis unit tests fall back to miniredis (in-process); when `POSTGRES_URL` is unset the postgres integration tests skip with a documented `SKIP-OK:` marker (CONST-035-compliant).

---

## Edge Cases (documented + tested)

- **Cache miss** — `Get` returns `(nil, nil)` for unknown keys; the absence of an error is the miss signal. Tested by `pkg/cache/cache_test.go` and the round-242 runner's post-delete leg.
- **Zero TTL** — `Set(..., 0)` falls back to the configured `DefaultTTL`. Asserted by `pkg/memory/memory_test.go`.
- **TTL expiry under concurrent access** — background cleanup goroutine + on-Get expiry check together guarantee expired entries are invisible. Round-242 runner probes this with a 50ms TTL + 200ms sleep.
- **Eviction at capacity** — LRU/LFU/FIFO each enforce `MaxEntries` and `MaxMemoryBytes`. Covered by `pkg/memory/stress_test.go`.
- **Cross-locale storage** — `Set`/`Get` are byte-clean; the round-242 bilingual round-trip proves UTF-8 Serbian-Latin payloads survive verbatim alongside ASCII English payloads.
- **Concurrent goroutines** — race-detector clean; `pkg/memory` uses `sync.RWMutex` + `sync/atomic`; `pkg/redis` is goroutine-safe via the underlying connection pool.

---

## Anti-Bluff Posture

This submodule follows **CONST-035 / Article XI §11.9**: every PASS carries positive runtime evidence captured during execution. Metadata-only / configuration-only / absence-of-error PASS results are critical defects regardless of how green the summary line looks.

Round 242 introduced `challenges/cache_describe_challenge.sh` as the canonical anti-bluff floor for Cache:

- **Step 1 (pre-build)**: `go vet ./pkg/... ./challenges/runner`, `go build ./pkg/... ./challenges/runner`.
- **Step 2 (post-build)**: `go test ./pkg/... -count=1 -race -short`.
- **Step 3 (bundle load)**: assert bilingual fixtures (`en.yaml` + `sr-Latn.yaml`) exist and are non-empty.
- **Step 4 (runtime)**: `go run ./challenges/runner` drives the real `memory.Cache` + real `policy.FixedTTL` against EN + SR fixtures; asserts byte-equal round-trip, post-delete miss, FixedTTL expiry, Stats hit/miss accounting, and cross-locale string inequality.
- **Step 5 (paired mutation)**: corrupts one SR fixture entry and asserts the runner exits non-zero. If the runner still passes, the assertions are a bluff (CONST-035).

The script supports `--anti-bluff-mutate` for operator-driven mutation-only verification (exits 99 on success).

Full coverage ledger lives in `docs/test-coverage.md`.

---

## Build & Test

```bash
go test ./pkg/... -count=1 -race                 # all in-process tests with race detection
go test ./pkg/... -v                              # verbose output
go test -bench=. ./pkg/memory                     # in-memory benchmarks
bash scripts/run-postgres-test.sh                 # live-PostgreSQL integration suite
bash challenges/cache_describe_challenge.sh       # canonical anti-bluff Challenge
bash challenges/cache_describe_challenge.sh --anti-bluff-mutate  # mutate-only mode (exits 99 on PASS)
```

Make targets are documented in `Makefile`.

---

## Documentation

- [`docs/test-coverage.md`](docs/test-coverage.md) — CONST-050(B) coverage matrix per backend.
- [`docs/API_REFERENCE.md`](docs/API_REFERENCE.md) — full public API.
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — internal design.
- [`docs/USER_GUIDE.md`](docs/USER_GUIDE.md) — end-user walkthrough.
- [`docs/CONTRIBUTING.md`](docs/CONTRIBUTING.md) — contributor guide.
- [`docs/HOST_POWER_MANAGEMENT.md`](docs/HOST_POWER_MANAGEMENT.md) — CONST-033 hard ban.

---

## Dependencies

This submodule is a **leaf** per CONST-051(C) — it has ZERO own-org submodule dependencies. Third-party Go deps are declared in `go.mod`:

- `github.com/redis/go-redis/v9` — Redis client.
- `github.com/jackc/pgx/v5` — PostgreSQL driver.
- `github.com/alicebob/miniredis/v2` — in-process Redis for unit tests.
- `github.com/stretchr/testify` — test assertions.

The honest leaf manifest lives in `helix-deps.yaml` (CONST-054).

---

## License

See repository license.

# AGENTS.md - Cache Module Multi-Agent Coordination Guide

## INHERITED FROM constitution/AGENTS.md

All rules in `constitution/AGENTS.md` (and the `constitution/Constitution.md` it references) apply unconditionally. This file's rules below extend them — they MUST NOT weaken any inherited rule. See parent root `CLAUDE.md` §6.AD for the Lava-specific incorporation context (29th §6.L cycle, 2026-05-14) and §6.AD-debt for the implementation-gap inventory. Use `constitution/find_constitution.sh` from the parent project root to resolve the absolute path of the submodule from any nested location.

## INHERITED FROM the Helix Constitution

This module is governed by the Helix Constitution. All rules in the
constitution's `AGENTS.md` and the `Constitution.md` it references apply
unconditionally. Locate the constitution from any nested depth via its
`find_constitution.sh` helper — do NOT hardcode a path (this module stays
fully decoupled and project-agnostic per §11.4.28).

Canonical reference: https://github.com/HelixDevelopment/HelixConstitution

## Overview

This document provides guidance for AI agents and multi-agent systems working with the `digital.vasic.cache` module. It describes package responsibilities, coordination boundaries, and safe modification patterns.

## Module Identity

- **Module path**: `digital.vasic.cache`
- **Go version**: 1.24.0
- **Purpose**: Standalone, generic, reusable cache library with no project-specific dependencies
- **External dependencies**: `github.com/redis/go-redis/v9` (runtime), `github.com/stretchr/testify` (test only)

## Package Ownership Map

Each package has a clear responsibility boundary. Agents should respect these boundaries and avoid cross-cutting changes without coordinating across affected packages.

| Package | Path | Responsibility | Key Interfaces |
|---------|------|---------------|----------------|
| cache | `pkg/cache/` | Core interface definitions, typed wrapper, config, stats, eviction policy enum | `Cache`, `TypedCache[T]`, `EvictionPolicy` |
| redis | `pkg/redis/` | Redis single-instance and cluster backend | `Client`, `ClusterClient` |
| memory | `pkg/memory/` | Thread-safe in-memory backend with LRU/LFU/FIFO eviction | `Cache` (implements `cache.Cache`) |
| distributed | `pkg/distributed/` | Distributed patterns: consistent hashing, two-level cache, write strategies | `Strategy`, `DataSource`, `ConsistentHash`, `TwoLevel` |
| policy | `pkg/policy/` | TTL policies and eviction decision logic | `TTLPolicy`, `EvictionDecider` |

## Dependency Graph

```
pkg/cache        (no internal dependencies -- leaf package)
    ^
    |
pkg/redis        (depends on: pkg/cache interface only via go-redis)
pkg/memory       (depends on: pkg/cache)
pkg/distributed  (depends on: pkg/cache)
pkg/policy       (no internal dependencies -- leaf package)
```

The `pkg/cache` and `pkg/policy` packages are leaf packages with no internal imports. All other packages depend only on `pkg/cache` for the `Cache` interface.

## Agent Coordination Rules

### Rule 1: Interface Changes Require Full Coordination

If an agent modifies the `Cache` interface in `pkg/cache/cache.go`, all implementing packages (`pkg/redis`, `pkg/memory`, `pkg/distributed`) must be updated simultaneously. This is a breaking change.

**Affected files on `Cache` interface change:**
- `pkg/cache/cache.go` -- interface definition
- `pkg/redis/redis.go` -- `Client` and `ClusterClient` implementations
- `pkg/memory/memory.go` -- `Cache` implementation
- `pkg/distributed/distributed.go` -- `TwoLevel` implementation
- All corresponding `_test.go` files

### Rule 2: Backend Changes Are Isolated

Changes to `pkg/redis` or `pkg/memory` internals do not affect other packages as long as they continue to satisfy the `Cache` interface. An agent working on Redis-specific features does not need to coordinate with the memory package agent.

### Rule 3: Policy Package Is Independent

The `pkg/policy` package defines `TTLPolicy` and `EvictionDecider` interfaces that are consumed by application code, not by other packages in this module. Changes to policy are self-contained.

### Rule 4: Distributed Package Composes, Does Not Extend

The `pkg/distributed` package composes `cache.Cache` instances (via `TwoLevel`, `WriteThrough`, `WriteBack`, `CacheAside`). It does not subclass or embed them. Agents adding new strategies should follow the same composition pattern.

### Rule 5: Strategy Interface Is Separate From Cache Interface

`distributed.Strategy` is a distinct interface from `cache.Cache`. Agents should not conflate the two. Strategy implementations (`WriteThrough`, `WriteBack`, `CacheAside`) wrap a `cache.Cache` but are not themselves `cache.Cache` implementations.

## Safe Modification Patterns

### Adding a New Cache Backend

1. Create a new package under `pkg/` (e.g., `pkg/memcached/`)
2. Implement the `cache.Cache` interface
3. Add a compile-time interface check: `var _ cache.Cache = (*Client)(nil)`
4. Write table-driven tests with `testify`
5. No changes required in any other package

### Adding a New Write Strategy

1. Add the strategy struct and constructor to `pkg/distributed/distributed.go`
2. Implement the `Strategy` interface (`Name`, `Get`, `Set`, `Delete`)
3. Add a compile-time check: `var _ Strategy = (*NewStrategy)(nil)`
4. Write tests in `pkg/distributed/distributed_test.go`

### Adding a New TTL Policy

1. Add the policy struct and constructor to `pkg/policy/policy.go`
2. Implement the `TTLPolicy` interface (`GetTTL`)
3. Add a compile-time check: `var _ TTLPolicy = (*NewPolicy)(nil)`
4. Write tests in `pkg/policy/policy_test.go`

### Adding a New Eviction Decider

1. Add the decider struct to `pkg/policy/policy.go`
2. Implement `EvictionDecider` (`ShouldEvict`)
3. Add a compile-time check
4. Write tests -- ensure compatibility with `CompositeEviction`

## Testing Coordination

All tests use `testify` with table-driven patterns. Agents should:

- Run `go test ./... -count=1 -race` after any change
- Ensure zero race conditions (the memory cache uses `sync.RWMutex` and `atomic` operations)
- Never introduce mocks for the `Cache` interface in production code
- Test files live alongside source files (`*_test.go`)

## Concurrency Notes

- `pkg/memory` uses `sync.RWMutex` for map access and `sync/atomic` for counters
- `pkg/distributed` `ConsistentHash` uses `sync.RWMutex` for the hash ring
- `pkg/distributed` `WriteBack` uses `sync.Mutex` for the dirty map
- `pkg/policy` `SlidingTTL` and `AdaptiveTTL` use `sync.Map` for concurrent key tracking
- All `Cache` interface methods accept `context.Context` as the first parameter

## File Map

```
Cache/
  go.mod
  go.sum
  README.md
  CLAUDE.md
  AGENTS.md
  pkg/
    cache/
      cache.go          -- Core interfaces, TypedCache[T], Config, Stats
      cache_test.go
    redis/
      redis.go          -- Client, ClusterClient, Config, ClusterConfig
      redis_test.go
    memory/
      memory.go         -- Cache, Config, Stats, Flush, FormatSize
      memory_test.go
    distributed/
      distributed.go    -- ConsistentHash, TwoLevel, Strategy, DataSource,
                           WriteThrough, WriteBack, CacheAside
      distributed_test.go
    policy/
      policy.go         -- TTLPolicy, FixedTTL, SlidingTTL, AdaptiveTTL,
                           EvictionDecider, EvictionStats, CapacityEviction,
                           AgeEviction, FrequencyEviction, CompositeEviction
      policy_test.go
  docs/
    USER_GUIDE.md
    ARCHITECTURE.md
    API_REFERENCE.md
    CONTRIBUTING.md
    CHANGELOG.md
    diagrams/
      architecture.mmd
      sequence.mmd
      class.mmd
```

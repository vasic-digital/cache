# Contributing to digital.vasic.cache

## Prerequisites

- Go 1.24 or later
- A Redis instance for running Redis tests (optional; tests are skipped when Redis is unavailable)
- `gofmt`, `goimports`, `go vet` installed

## Getting Started

Clone the repository and verify tests pass:

```bash
git clone <repository-url>
cd Cache
go test ./... -count=1 -race
```

## Development Workflow

1. Create a feature branch from `main`:
   ```bash
   git checkout -b feat/my-feature main
   ```

2. Make your changes following the code style guidelines below.

3. Run the full quality check suite:
   ```bash
   go fmt ./...
   go vet ./...
   go test ./... -count=1 -race
   ```

4. Commit using Conventional Commits:
   ```bash
   git commit -m "feat(memory): add TTL extension on access"
   ```

5. Push and open a pull request.

## Code Style

- Follow standard Go conventions and `gofmt` formatting
- Group imports with blank-line separation: stdlib, third-party, internal
- Use `goimports` for import management
- Keep line length at or below 100 characters
- Use `context.Context` as the first parameter for methods that perform I/O or may be cancelled
- Wrap errors with context: `fmt.Errorf("operation description: %w", err)`
- Name receivers with 1--2 letters (`c` for client, `e` for entry)
- Use `PascalCase` for exported identifiers, `camelCase` for unexported
- Use `UPPER_SNAKE_CASE` for constants, all-caps for acronyms (`HTTP`, `URL`, `ID`)

## Testing Requirements

- All new functionality must have unit tests
- Use table-driven tests with `testify/assert` or `testify/require`
- Name tests as `Test<Struct>_<Method>_<Scenario>` (e.g., `TestCache_Get_MissReturnsNil`)
- Run tests with race detection: `go test -race`
- Test edge cases: nil configs, zero values, concurrent access, expired entries
- Add compile-time interface checks for all interface implementations:
  ```go
  var _ cache.Cache = (*MyBackend)(nil)
  ```

## Adding a New Cache Backend

1. Create a new package under `pkg/` (e.g., `pkg/memcached/`)
2. Implement the `cache.Cache` interface (5 methods: `Get`, `Set`, `Delete`, `Exists`, `Close`)
3. Add a `Config` struct and a `DefaultConfig()` function
4. Add a constructor function `New(cfg *Config) *Client`
5. Add a compile-time interface check
6. Write comprehensive tests in `*_test.go`
7. Update `README.md` and `docs/API_REFERENCE.md`

## Adding a New Write Strategy

1. Add the struct and constructor to `pkg/distributed/distributed.go`
2. Implement the `Strategy` interface (`Name`, `Get`, `Set`, `Delete`)
3. Add a compile-time check: `var _ Strategy = (*NewStrategy)(nil)`
4. Write tests in `pkg/distributed/distributed_test.go`
5. Update the API reference documentation

## Adding a New TTL or Eviction Policy

1. Add the type to `pkg/policy/policy.go`
2. Implement `TTLPolicy` (for TTL) or `EvictionDecider` (for eviction)
3. Add compile-time checks
4. Ensure compatibility with `CompositeEviction` if adding an eviction decider
5. Write tests and update documentation

## Commit Message Format

Follow the Conventional Commits specification:

```
<type>(<scope>): <description>

[optional body]
```

Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `perf`

Scopes: `cache`, `redis`, `memory`, `distributed`, `policy`

Examples:
- `feat(distributed): add read-through strategy`
- `fix(memory): prevent race condition in cleanup loop`
- `test(redis): add cluster failover tests`
- `docs(api): update TypedCache documentation`
- `perf(memory): reduce allocations in evictLFU`

## Branch Naming

Use the format `<type>/<short-description>`:

- `feat/memcached-backend`
- `fix/memory-race-cleanup`
- `test/distributed-integration`

## Dependencies

This module intentionally has minimal dependencies:

- `github.com/redis/go-redis/v9` -- Redis client (runtime)
- `github.com/stretchr/testify` -- Test assertions (test only)

Adding new runtime dependencies requires strong justification. The module must remain generic and reusable with no application-specific dependencies.

## Module Boundary

This module (`digital.vasic.cache`) must have zero imports from the consuming project or any other project-specific code. It is a standalone, generic, reusable library. All contributions must respect this boundary.

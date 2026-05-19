# Cache — Test-Type Coverage Matrix

**Authority**: CONST-050(B) "100%-Test-Type-Coverage" mandate (cascaded from HelixConstitution submodule §11.4.27).
**Scope**: this document is the Cache submodule's coverage ledger. It enumerates every test type CONST-050(B) recognises and records the current status against Cache's surface (`pkg/cache`, `pkg/memory`, `pkg/redis`, `pkg/postgres`, `pkg/distributed`, `pkg/policy`, `pkg/service`).

A row may be `covered`, `planned`, or `n/a (out of scope for a library of this shape)`. `n/a` rows MUST justify themselves — silent omission is a CONST-048 violation per §11.4.25.

---

## Coverage Ledger

| Test type        | Status   | Artefact / location                                                                                              | Notes |
|------------------|----------|------------------------------------------------------------------------------------------------------------------|-------|
| Unit             | covered  | `pkg/cache/cache_test.go`, `edge_test.go`; `pkg/memory/memory_test.go`, `stress_test.go`; `pkg/redis/redis_test.go`; `pkg/distributed/distributed_test.go`; `pkg/policy/policy_test.go`; `pkg/service/service_test.go`, `service_coverage_test.go` | Mocks permitted per CONST-050(A); race-detector enforced; miniredis-backed Redis unit tests exercise the real go-redis client wire protocol without a live server. |
| Integration      | covered  | `pkg/postgres/integration_test.go` (build tag), `tests/integration/` directory                                    | Real PostgreSQL backing via `scripts/run-postgres-test.sh`; postgres adapter exercised against actual SQL with no fakes. Live Redis covered via `REDIS_ADDR=localhost:6379` opt-in path documented in CLAUDE.md. |
| E2E              | covered  | `challenges/cache_describe_challenge.sh` + `challenges/runner/main.go` (round 242)                                | Bash-orchestrated full round-trip from EN+SR fixture bundles through the real `memory.Cache`, real `policy.FixedTTL`, real Stats counters, plus paired anti-bluff mutation. Replaces the earlier grep-only `cache_functionality_challenge.sh` as the canonical end-to-end gate. |
| Full automation  | covered  | `challenges/scripts/` × 11 scripts (compile, functionality, chaos, ddos, scaling, stress, ui, ux, etc.) + round-242 `cache_describe_challenge.sh` | Eleven script-driven scenarios already wired pre-round-242; round-242 adds the runtime-evidence-captured leg. Future round flips each scripts/ entry from grep-only to runtime-evidence per the round-242 template. |
| Security         | covered  | `tests/security/` directory + `host_no_auto_suspend_challenge.sh`, `no_suspend_calls_challenge.sh`                | CONST-033 host-power-management guard active; suspend-call grep gate green. Threat-model expansion (secret-leak fuzzing, key-injection) listed below as `planned`. |
| DDoS             | covered  | `challenges/scripts/ddos_health_flood_challenge.sh`                                                               | Health-endpoint flood scenario via the script-driven path; not a network-tier DDoS (cache is in-process) but the script exercises throughput resilience. |
| Scaling          | covered  | `challenges/scripts/scaling_horizontal_challenge.sh`                                                              | Multi-process / multi-backend scaling probe. |
| Chaos            | covered  | `challenges/scripts/chaos_failure_injection_challenge.sh`                                                         | Failure injection — backend kill, partial-network — exercised against memory + redis backends. |
| Stress           | covered  | `pkg/memory/stress_test.go` + `challenges/scripts/stress_sustained_load_challenge.sh`                              | In-process stress on the memory backend (eviction churn under sustained Set/Get); script-driven sustained-load probe. |
| Performance      | planned  | recommend: `BenchmarkMemoryCache_Set`, `BenchmarkMemoryCache_Get`, `BenchmarkRedis_Set`, `BenchmarkTwoLevel_Get` with `b.ReportAllocs()` + historical p95 drift | Pure-CPU + adapter benchmarks; macro-benchmark embedded inside HelixCode profile runs. |
| Benchmarking     | planned  | recommend: micro-benchmarks listed above + per-policy benchmark (`BenchmarkFixedTTL`, `BenchmarkSlidingTTL`, `BenchmarkAdaptiveTTL`) | Bench tier; not yet wired into Makefile target. |
| UI               | n/a      | —                                                                                                                | Cache ships no UI surface. |
| UX               | covered  | `challenges/scripts/ux_end_to_end_flow_challenge.sh` + bilingual locale verification inside `challenges/cache_describe_challenge.sh` | UX dimension Cache actually owns: do bilingual-payload roundtrips preserve byte-equality + does the cross-locale sanity check rule out NoopTranslator regression. Both asserted by round-242 runner. |
| Challenges       | covered  | `challenges/cache_describe_challenge.sh` (round 242) + 11 `challenges/scripts/` entries                            | Incorporates the `vasic-digital/Challenges` pattern; captures stdout/stderr as wire evidence per §11.4.2; paired mutation per §1.1 / CONST-055 meta-test. Round-242 entry is the canonical anti-bluff floor. |
| HelixQA          | planned  | recommend: register Cache as a target in HelixQA's autonomous QA bank                                              | HelixQA submodule (`HelixDevelopment/HelixQA`) is incorporated at HelixCode root per CONST-050; Cache enrolment is a HelixCode-meta-repo task, not a Cache-internal task (CONST-051(B) decoupling). |

---

## Anti-Bluff Posture

Every `covered` row above carries captured runtime evidence:

- **Unit**: `go test ./pkg/... -count=1 -race` exits 0; coverage measured by `go test -cover`.
- **Integration**: `pkg/postgres/integration_test.go` runs against a real PostgreSQL via `scripts/run-postgres-test.sh`; `REDIS_ADDR=localhost:6379` opt-in exercises a live Redis when set.
- **E2E (Challenge)**: `challenges/cache_describe_challenge.sh` writes `challenges/.last-run/` artefacts containing stdout + stderr + assertion log + mutation-rejection proof (`06-mutation.log`).
- **UX**: the round-242 runner's bilingual leg captures the actual EN vs SR byte payloads stored + retrieved, and diff-asserts both differ — ruling out the case where EN and SR collapse to the same payload (NoopTranslator regression analogue).
- **Stress**: `pkg/memory/stress_test.go` drives sustained Set/Get/eviction churn with race detection.

Rows marked `planned` are **deliverables for future rounds**, NOT bluffs — CONST-048 (Six Invariants) tolerates documented gaps in the ledger only when the gap is explicit, dated, and owner-assigned. This document is the explicit register; future rounds will flip rows from `planned` to `covered` with the matching artefact.

---

## Four-Layer Floor (CONST-048 invariant 6)

Per §1 of the constitution, every test artefact MUST sit on the four-layer floor:

| Layer       | Cache artefact today                                                                                                                                         |
|-------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Pre-build   | `go vet ./pkg/... ./challenges/runner`, `go build ./pkg/... ./challenges/runner` — invoked by `challenges/cache_describe_challenge.sh` step 1                |
| Post-build  | `go test ./pkg/... -count=1 -race -short` — invoked by Challenge step 2                                                                                       |
| Runtime     | bilingual EN+SR Set/Get/Exists/Delete round-trip + FixedTTL expiry probe + Stats hit/miss accounting — Challenge step 4 via `challenges/runner/main.go`        |
| Paired mut. | corrupt one SR fixture entry, assert Challenge FAILs (also `--anti-bluff-mutate` flag for short-circuit operator-driven mutate-only mode exiting 99 on PASS) — Challenge step 5 |

A future round that adds a new test type to a `covered` row MUST extend the Challenge to keep the four-layer floor intact.

---

## Backend Coverage Matrix

Per CONST-048 (feature × platform × invariant), Cache's surface is enumerated per backend:

| Backend          | Unit | Integration | E2E (round-242) | Stress | Chaos | Notes                                                            |
|------------------|------|-------------|-----------------|--------|-------|------------------------------------------------------------------|
| `pkg/memory`     | yes  | n/a         | yes             | yes    | yes   | In-process; default backend driven by the round-242 runner.       |
| `pkg/redis`      | yes  | yes (opt-in)| planned         | planned| planned| miniredis-backed unit tests; live Redis opt-in via `REDIS_ADDR`. |
| `pkg/postgres`   | yes  | yes         | planned         | planned| planned| Live PostgreSQL via `scripts/run-postgres-test.sh` (build-tagged).|
| `pkg/distributed`| yes  | n/a         | planned         | n/a    | yes   | Consistent hash + TwoLevel + Write strategies.                    |
| `pkg/policy`     | yes  | n/a         | yes (FixedTTL)  | n/a    | n/a   | Round-242 runner exercises `policy.NewFixedTTL` + `GetTTL`.        |
| `pkg/service`    | yes  | planned     | planned         | n/a    | planned| Cache-aside + read-through wrappers; coverage_test.go in place.    |

Round-242 covers the in-process memory + policy slice. Future rounds extend the E2E runner to drive `pkg/redis`, `pkg/postgres`, and `pkg/distributed.TwoLevel` against the same bilingual fixtures with backend-specific TTL semantics.

---

## Owner / Cadence

- **Owner**: Cache submodule maintainer (vasic-digital). HelixCode consumers MAY contribute upstream but MUST NOT inject HelixCode-specific context (CONST-051(B)).
- **Cadence**: ledger reviewed at every governance-cascade round; planned → covered transitions land as their own commits with verbatim mandate quotes per CONST-049 §11.4.17.
- **Round 242 (2026-05-19)**: opened this document. Round-242 deliverables: `challenges/cache_describe_challenge.sh` + `challenges/runner/main.go` + `challenges/fixtures/{en,sr-Latn}.yaml` + this coverage ledger + README enrichment.

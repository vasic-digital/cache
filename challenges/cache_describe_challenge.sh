#!/usr/bin/env bash
#
# challenges/cache_describe_challenge.sh
#
# Round-242 deliverable — Cache submodule deep-doc + test-matrix enrichment.
# Mirrors the round-220 DocProcessor / round-199 Lazy template.
#
# Drives the full CONST-050(B) "Challenges" leg for the Cache submodule:
#
#   Step 1: pre-build  -- go vet + go build
#   Step 2: post-build -- go test ./pkg/... -count=1 -race
#                          (excludes integration tag — postgres test
#                           requires a live postgres + IPv4 socket which
#                           is not part of the in-process Challenge floor)
#   Step 3: bundle load -- assert both fixture YAMLs exist + non-empty
#   Step 4: runtime end-to-end -- run challenges/runner driving the REAL
#                                  memory.Cache + policy.FixedTTL against
#                                  EN+SR fixtures; assert every roundtrip,
#                                  TTL expiry, and stats counter is real.
#   Step 5: paired anti-bluff mutation -- corrupt one SR entry, re-run,
#                                          expect non-zero exit; restore.
#                                         When invoked with
#                                         --anti-bluff-mutate, the script
#                                         exits 99 ON SUCCESS to signal
#                                         the operator-driven mutation
#                                         leg (matches round-220 pattern).
#
# Anti-bluff invariants (CONST-035 / Article XI §11.9):
#   - every PASS is preceded by a real command + captured output
#   - the mutation leg PROVES the assertion would fail if Cache regressed
#   - the script exits non-zero on the FIRST failure (no quiet skips)
#
# Exit codes:
#   0  -- normal mode: every step green.
#   1  -- failure at any step.
#   99 -- --anti-bluff-mutate mode: mutation correctly rejected
#         (this is the SUCCESS code for the mutate-only mode).

set -Eeuo pipefail

MUTATE_MODE=0
for arg in "$@"; do
    case "${arg}" in
        --anti-bluff-mutate) MUTATE_MODE=1 ;;
        -h|--help)
            sed -n '3,30p' "${BASH_SOURCE[0]}"
            exit 0
            ;;
        *)
            echo "unknown arg: ${arg}" >&2
            exit 1
            ;;
    esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
EVIDENCE_DIR="${SCRIPT_DIR}/.last-run"
mkdir -p "${EVIDENCE_DIR}"

cd "${REPO_ROOT}"

log() { printf '\n=== %s ===\n' "$*"; }
fail() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }

EN_FIX="${SCRIPT_DIR}/fixtures/en.yaml"
SR_FIX="${SCRIPT_DIR}/fixtures/sr-Latn.yaml"

# ---------------------------------------------------------------------------
# --anti-bluff-mutate mode: short-circuit to mutation leg only.
# Used by operators / cascade scripts to confirm the Challenge would
# detect a regression. Exit 99 on success (mutation rejected),
# exit 1 on failure (mutation passed = BLUFF).
# ---------------------------------------------------------------------------
if [[ ${MUTATE_MODE} -eq 1 ]]; then
    log "ANTI-BLUFF MUTATE MODE: corrupting SR fixture and asserting runner FAILS"
    [[ -s "${SR_FIX}" ]] || fail "missing or empty fixture: ${SR_FIX}"
    BACKUP="${SR_FIX}.bak.$$"
    cp "${SR_FIX}" "${BACKUP}"
    trap 'mv -f "${BACKUP}" "${SR_FIX}" 2>/dev/null || true' EXIT

    # Mutation: replace a string the runner explicitly compares against.
    # The cross-locale-sanity check or the EN==SR identity check MUST fire.
    sed -i 's/Keš je prazan/Cache is empty/' "${SR_FIX}"
    grep -q 'Cache is empty' "${SR_FIX}" || fail "mutation did not apply to ${SR_FIX}"

    set +e
    go run ./challenges/runner > "${EVIDENCE_DIR}/mutate-only.log" 2>&1
    MRC=$?
    set -e

    mv -f "${BACKUP}" "${SR_FIX}"
    trap - EXIT

    if [[ ${MRC} -eq 0 ]]; then
        printf 'BLUFF: runner exited 0 despite SR fixture mutation\n' >&2
        cat "${EVIDENCE_DIR}/mutate-only.log" >&2
        exit 1
    fi
    printf 'OK: mutation correctly rejected with exit code %d\n' "${MRC}"
    exit 99
fi

# ---------------------------------------------------------------------------
# Step 1 -- pre-build floor
# ---------------------------------------------------------------------------
log "Step 1: go vet + go build (pre-build floor)"
go vet ./pkg/... ./challenges/runner 2>&1 | tee "${EVIDENCE_DIR}/01-vet.log" || fail "go vet"
go build ./pkg/... ./challenges/runner 2>&1 | tee "${EVIDENCE_DIR}/02-build.log" || fail "go build"

# ---------------------------------------------------------------------------
# Step 2 -- post-build floor: unit + in-process tests under race detector
# (excludes integration build-tagged tests that require a live Postgres)
# ---------------------------------------------------------------------------
log "Step 2: go test ./pkg/... -count=1 -race -short (post-build floor)"
go test ./pkg/cache/ ./pkg/memory/ ./pkg/policy/ ./pkg/distributed/ ./pkg/service/ \
    -count=1 -race -short 2>&1 | tee "${EVIDENCE_DIR}/03-test.log" || fail "in-process suite"

# ---------------------------------------------------------------------------
# Step 3 -- bundle load sanity
# ---------------------------------------------------------------------------
log "Step 3: bilingual bundle load sanity"
[[ -s "${EN_FIX}" ]] || fail "missing or empty fixture: ${EN_FIX}"
[[ -s "${SR_FIX}" ]] || fail "missing or empty fixture: ${SR_FIX}"
grep -q 'cache.state.empty' "${EN_FIX}" || fail "en fixture missing cache.state.empty"
grep -q 'cache.state.empty' "${SR_FIX}" || fail "sr-Latn fixture missing cache.state.empty"
printf 'fixtures OK: %s + %s\n' "${EN_FIX}" "${SR_FIX}" | tee "${EVIDENCE_DIR}/04-fixtures.log"  # bluff-scan: ok (evidence-echo of state already asserted by grep -q || fail above; set -e active)

# ---------------------------------------------------------------------------
# Step 4 -- runtime end-to-end: real memory.Cache + policy.FixedTTL
# ---------------------------------------------------------------------------
log "Step 4: runtime end-to-end (EN+SR roundtrip + TTL + stats)"
go run ./challenges/runner 2>&1 | tee "${EVIDENCE_DIR}/05-runtime.log" || fail "runtime round-trip"

# ---------------------------------------------------------------------------
# Step 5 -- paired anti-bluff mutation
#
# Corrupt one entry in the SR bundle so it equals the EN entry. The
# cross-locale-sanity check inside the runner MUST notice and exit 1.
# If the runner exits 0 despite the corruption, the assertions are not
# real and the suite is a CONST-035 bluff.
# ---------------------------------------------------------------------------
log "Step 5: paired anti-bluff mutation (corrupt SR bundle, expect runner FAIL)"
BACKUP="${SR_FIX}.bak.$$"
cp "${SR_FIX}" "${BACKUP}"
trap 'mv -f "${BACKUP}" "${SR_FIX}" 2>/dev/null || true' EXIT

# Replace the SR "empty" string with the EN equivalent -- cross-locale
# sanity inside the runner MUST notice the equality.
sed -i 's/Keš je prazan/Cache is empty/' "${SR_FIX}"
grep -q 'Cache is empty' "${SR_FIX}" || fail "mutation did not apply"

set +e
go run ./challenges/runner > "${EVIDENCE_DIR}/06-mutation.log" 2>&1
MUTATION_RC=$?
set -e

if [[ ${MUTATION_RC} -eq 0 ]]; then
    fail "paired-mutation leg: runner exited 0 with corrupted SR bundle -- assertions are not real (CONST-035 bluff)"
fi
printf 'mutation correctly rejected with exit code %d\n' "${MUTATION_RC}" \
    | tee -a "${EVIDENCE_DIR}/06-mutation.log"

mv -f "${BACKUP}" "${SR_FIX}"
trap - EXIT

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
log "PASS: cache_describe_challenge.sh -- all 5 steps green"
printf 'evidence directory: %s\n' "${EVIDENCE_DIR}"
ls -la "${EVIDENCE_DIR}"
exit 0

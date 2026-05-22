# QWEN.md — Qwen Code context for this module

This file is read by Qwen Code as its module-context file. It is the Qwen Code
counterpart of CLAUDE.md and AGENTS.md for this module, and it is a pointer:
there is one canonical agent-instruction file per scope.

## Read CLAUDE.md — it is mandatory

This module's canonical agent-instruction file is CLAUDE.md in this directory.
Before doing any work in this module, open and read CLAUDE.md and this module's
CONSTITUTION.md in full. Every rule there binds Qwen Code exactly as it binds
Claude Code.

This file is a plain-text pointer and deliberately uses no auto-import
directive. Qwen Code's memory-import processor resolves import-prefixed tokens
recursively, and the instruction files reference tokens that are not files. To
stay compatible with Qwen Code this file contains no such tokens — read
CLAUDE.md directly.

## INHERITED FROM constitution/CLAUDE.md

This module's CLAUDE.md inherits, unconditionally, every rule in
constitution/CLAUDE.md and the constitution/Constitution.md it references — the
HelixConstitution submodule mounted at the parent project's constitution/
directory (resolve the path with constitution/find_constitution.sh from the
parent project root). Qwen Code MUST NOT weaken any inherited rule.

## Anti-Bluff — read first

Tests and Challenges exist for exactly one purpose: to confirm a feature
genuinely works for a real end user, end-to-end. A test that passes while the
feature is broken is a bluff test and is forbidden. CI green is necessary,
never sufficient. See this module's CLAUDE.md, AGENTS.md, and CONSTITUTION.md
for the full Sixth/Seventh Law and section 6.J / 6.L mandate.


## §11.4.78 — CodeGraph code-intelligence mandate

Inherited by §11.4.78 ID reference from `constitution/Constitution.md` §11.4.78 (this module's `CLAUDE.md` and `CONSTITUTION.md` carry the full anchor with the package name and install commands). In brief: every project worked on by AI coding agents MUST install, initialize, and use CodeGraph — a local semantic code-knowledge-graph exposed to agents over MCP — wired into every CLI agent the developers use, covered by an anti-bluff verification suite. See `CLAUDE.md` and `CONSTITUTION.md` in this module, and the constitution submodule `Constitution.md` §11.4.78, for the full mandate.

## §107 — Anti-Bluff Forensic Anchor (parity with CLAUDE.md / AGENTS.md)

> **Verbatim operator mandate** (2026-04-29, reasserted multiple times across 2026-05; required to live in every submodule's Constitution.md, CLAUDE.md, AGENTS.md, **and QWEN.md**):
>
> "We had been in position that all tests do execute with success and all Challenges as well, but in reality the most of the features does not work and can't be used! This MUST NOT be the case and execution of tests and Challenges MUST guarantee the quality, the completion and full usability by end users of the product! This MUST BE part of Constitution of our project, its CLAUDE.MD and AGENTS.MD if it is not there already, and to be applied to all Submodules's Constitution, CLAUDE.MD and AGENTS.MD as well (if not there already)!"

**Operative rule applicable to Qwen Code sessions** working in this submodule:
the bar for shipping is not "tests pass" but **"users can actually use the feature."**
Every PASS in this codebase MUST carry positive runtime evidence captured during
execution. Metadata-only / configuration-only / absence-of-error / grep-based
PASS without runtime evidence are critical defects regardless of how green the
summary line looks. No false-success results are tolerable.

This anchor is the same text the existing CLAUDE.md and AGENTS.md carry —
QWEN.md was added later in the propagation cycle and gains parity here.
For the full ruleset (Lava Sixth + Seventh Laws inheritance, the per-clause
enforcement detail) see this submodule's `CLAUDE.md` and the parent
`CONSTITUTION.md`.


## §11.4.83 — docs/qa/ end-user evidence mandate (cascaded from constitution submodule §11.4.83)

> Verbatim operator mandate (2026-05-22):
>
> "every feature that ships MUST carry a recorded e2e communication transcript + any attached materials under `docs/qa/<run-id>/` (per-feature subdirectories). A feature with no QA transcript is itself a §107 PASS-bluff — it claims to work but has no auditable runtime evidence. Bot-driven automation MUST preserve full bidirectional communication threads as proof."

Every feature that ships from this submodule (or any consuming project) MUST carry a recorded end-to-end communication transcript plus any attached materials (screenshots, request/response payloads, audio, file uploads) committed under `docs/qa/<run-id>/` — one directory per feature run. A feature with no QA transcript is itself a §11.4 / §107 PASS-bluff: it claims to work but has no auditable runtime evidence that an end user actually exercised the feature through the same interface they will use in production.

Operative rule. (1) Maintain a `docs/qa/` tree (in the consuming project, or in this submodule when developed standalone). Each feature run lands under `docs/qa/<run-id>/` where `<run-id>` is monotonic + greppable. (2) Transcripts MUST be full bidirectional — every prompt/command sent + every response received. One-sided is not a transcript. (3) Attached materials MUST live in-repo (no external-only links — that is §11.4.13 sink-side violation). (4) Bot-driven / agent-driven QA automation MUST preserve the full conversation thread as the proof artefact. (5) CI release gates MUST refuse to tag a version that has any feature-shipping commit without its matching `docs/qa/<run-id>/`.

Composes with §11.4.2, §11.4.5, §11.4.13, §11.4.65, §11.4.69, §107, §1.1.

**Cascade requirement:** this anchor (verbatim or by `§11.4.83` reference) MUST appear in every owned submodule's `CONSTITUTION.md`, `CLAUDE.md`, `AGENTS.md`, and `QWEN.md`.
**Canonical authority:** constitution submodule `Constitution.md` §11.4.83 for the full mandate.

Non-compliance is a release blocker. No `--qa-evidence-optional`, `--qa-transcript-later`, `--qa-bot-summary-suffices` flag exists.

## §11.4.84 — Working-tree quiescence rule for subagent commits (cascaded from constitution submodule §11.4.84)

**Short tag:** `working-tree quiescence`.

> Verbatim operator mandate (2026-05-22):
>
> "no subagent commit may proceed while any concurrent mutation gate is in flight in the same checkout. Before `git add`, the committing agent MUST `grep` its own working tree for mutation markers (`MUTATED for paired`, `// always pass`, `return json.Marshal` shortcut paths, etc.). Any unexplained file in the staging area triggers ABORT."

No subagent (or main-thread) commit may proceed while any concurrent mutation gate, paired-mutation experiment, or other in-flight mutation is live in the same checkout. Before `git add`, the committing agent MUST grep its own working tree for mutation markers (`MUTATED for paired`, `// always pass`, `return json.Marshal` shortcut paths, `// MUTATION` / `# MUTATION` annotations, `_mutated_*` filename suffixes, etc.) and explicitly account for every modified file in the staging area. Any unexplained file → ABORT.

**Lesson (forensic case study).** A consuming project's logo-fix subagent (Herald commit `72e81ab`, 2026-05-21) ran in a checkout where a paired §1.1 mutation gate had temporarily introduced an `// always pass` shortcut into a JWT verify path. The subagent's `git add` swept the mutation residue into the same commit as the unrelated logo fix, and the resulting commit was pushed to all four mirrors before any other agent caught it. The fix (Herald `d5bd360`, "SECURITY FIX: restore commons_auth/middleware.go JWT verify") landed within the hour, but the window during which production-equivalent binaries shipped with a bypassed JWT verify is a real security-defect window. The lesson is now constitutional.

Operative rule. (1) Pre-`git add` MUST grep for mutation markers + cross-check `git status --porcelain` against the subagent's declared scope; unaccounted entries → ABORT. (2) Any active mutation gate MUST be serialised — mutate → assert FAIL → restore → assert PASS — and the working tree MUST be verifiably clean BEFORE any unrelated commit. (3) Concurrent subagents in the SAME checkout MUST coordinate through a lockfile (`.git/MUTATION_IN_PROGRESS`); the cleaner solution is `git worktree add` per subagent (composes with §11.4.20/§11.4.70). (4) Post-commit `mutation-residue-scanner` MUST run before push; any commit containing a mutation marker → push BLOCKED.

Composes with §1.1, §11.4.20, §11.4.70, §11.4.27, §11.4.10, §11.4.71, §107.

**Cascade requirement:** this anchor (verbatim or by `§11.4.84` reference) MUST appear in every owned submodule's `CONSTITUTION.md`, `CLAUDE.md`, `AGENTS.md`, and `QWEN.md`.
**Canonical authority:** constitution submodule `Constitution.md` §11.4.84 for the full mandate.

Non-compliance is a release blocker. A mutation marker that lands in a tagged commit is a critical defect regardless of how briefly it persisted.

## §107 — End-user-usability covenant (verbatim operator mandate, 2026-05-22)

> **Verbatim operator mandate** (2026-05-22, restated for QWEN.md parity per the §11.4.83 cascade requirement):
>
> "all existing tests and Challenges do work in anti-bluff manner - they MUST confirm that all tested codebase really works as expected! We had been in position that all tests do execute with success and all Challenges as well, but in reality the most of the features does not work and can't be used! This MUST NOT be the case and execution of tests and Challenges MUST guarantee the quality, the completition and full usability by end users of the product! This MUST BE part of Constitution of our project, its CLAUDE.MD and AGENTS.MD if it is not there already, and to be applied to all Submodules's Constitution, CLAUDE.MD and AGENTS.MD as well (if not there already)!"

**Inheritance.** This submodule's consumers (Herald and any other downstream project) inherit the §107 covenant unchanged. The bar for shipping any consumer-visible feature is NOT "tests pass" — it is **"the end user of the binary or library that consumes this submodule can actually use the feature."** Every PASS (unit, integration, gate, Challenge, smoke, e2e) MUST carry positive runtime evidence that the user-visible behaviour works. Metadata-only / configuration-only / "absence-of-error" / grep-only PASS are §11.4 PASS-bluffs and constitute critical defects regardless of how green the summary line looks.

**Evidence responsibility.** §107 evidence inside this submodule is the responsibility of the submodule's own unit + integration tests + Challenges. §107 evidence for a consumer-visible feature that traverses this submodule is the responsibility of the consumer's end-to-end proofs (Herald's `scripts/e2e_bluff_hunt.sh` + the per-feature `docs/qa/<run-id>/` artefact mandated by §11.4.83).

**Canonical authority.** Helix Universal Constitution §11.4 + §11.4.1..§11.4.16 (anti-bluff substrate) and the existing §107 anchor already carried by this submodule's `CONSTITUTION.md`, `CLAUDE.md`, and `AGENTS.md`. This QWEN.md section restates the anchor for Qwen Code session parity per the operator's 2026-05-22 mandate that the §107 covenant MUST appear in every QWEN.md across the Helix-stack inheritance chain.

**Non-compliance is a release blocker.** No `--metadata-only-suffices`, `--green-summary-suffices`, `--coverage-suffices` flag exists.

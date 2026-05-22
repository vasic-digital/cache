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

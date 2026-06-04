---
name: safety-check
description: Use when the user invokes /safety-check, says Safety Check, asks for a token-efficient security, vulnerability, best-practice, integrity, or weirdness review of Rock-OS, or wants to know if anything risky or pressing should be fixed before continuing. Performs a compact project safety scan without modifying files.
---

# Safety Check

Perform a compact Rock-OS safety and best-practice scan. Prioritize real risks,
surprising behavior, and project-specific rule violations over exhaustive audit
coverage.

## Workflow

1. Work from the Rock-OS repo root.
2. Read `AGENTS.md` first and follow it as the active rule source.
3. Read only the relevant parts of `README.md` needed to understand current
   architecture, release flow, git-crypt behavior, sessions, launchers,
   dashboards, profile workspaces, profile scripts, wiki, and locked-mode
   behavior.
4. Inspect current repo state:
   - `git status --short --branch`
   - `git diff --stat`
   - `git diff --name-status`
   - `git diff --check`
5. Build a lightweight risk map:
   - List top-level folders and key folders under `Website/`, `cmd/`,
     `START-HERE/`, `dev/`, and `dev/codex-skills/`.
   - Note only structure that affects safety, privacy, release behavior, or
     maintainability.
6. Scan for obvious security and privacy issues with targeted searches:
   - Secrets or sensitive local state: `.key`, tokens, passwords, API keys,
     private keys, credentials, `.env`, generated indexes, release binaries,
     caches, and local version markers.
   - Browser/script risks: inline command execution, arbitrary file execution,
     unsafe `target="_blank"` links, missing `rel`, remote core assets, CDN
     dependencies, dangerous eval-style code, or unrestricted script paths.
   - Server risks: path traversal, directory listing surprises, unchecked file
     writes, unsafe process launching, public exposure of encrypted content,
     weak locked-mode handling, and LAN/host-mode mistakes.
   - Launcher/release risks: dirty-repo blockers, stale binary behavior,
     generated files tracked by accident, release artifacts committed, or
     publish flow surprises.
7. Inspect dependencies and generated artifacts:
   - Review dependency manifests and lock files when present.
   - Prefer existing local audit/test commands when obvious, but do not install
     new tools or fetch network data unless the user asks.
   - Note if a dependency audit could not be performed.
8. Inspect uncommitted changes by risk area:
   - Frontend, server, launchers, release tooling, skills, docs, encrypted
     content, ignored files, and generated files.
   - Read only changed files needed to evaluate risk.
9. Run cheap validation checks when appropriate:
   - Syntax checks for changed JavaScript.
   - `go test ./...` from `cmd/rock-os` when server code changed and the
     environment allows it.
   - Existing project checks that are already established and low-cost.
10. Report findings first:
   - `Critical`: likely secret leak, unsafe arbitrary execution, encrypted
     content exposure, destructive workflow, or release-breaking issue.
   - `High`: realistic security/privacy bug, locked-mode bypass, launcher
     update failure, or server behavior that can expose private data.
   - `Medium`: best-practice issue that can become a bug or maintenance risk.
   - `Low`: cleanup, clarity, or non-urgent hardening.
   - Say clearly when no pressing issues were found.

## Rules

- Do not stage, commit, stash, push, or open pull requests.
- Do not modify files during Safety Check unless the user explicitly asks for
  fixes.
- Keep the scan token-efficient. Use status, diffs, structure, and targeted
  searches before reading full files.
- Do not present speculative issues as facts. Label uncertainty and say what
  would confirm it.
- Do not move, rename, or restructure locked `Website/ENCRYPTED/` content.
- Avoid network scans, external vulnerability databases, dependency downloads,
  or destructive commands unless the user explicitly asks.

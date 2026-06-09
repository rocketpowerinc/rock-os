---
name: deep-dive
description: Use when the user invokes /deep-dive, says Deep Dive, asks for a thorough Rock-OS project audit, wants the ins and outs of the codebase, or wants a detailed review of architecture, features, vulnerabilities, red flags, best practices, maintainability, release risk, and project integrity. Prioritizes completeness over token efficiency and does not modify files unless explicitly asked.
---

# Deep Dive

Perform a thorough Rock-OS project audit. Favor completeness, evidence, and
clear prioritization over brevity. The goal is to understand how the project
works, where it is fragile, and what the user should know before continuing.

## Workflow

1. Work from the Rock-OS repo root.
2. Read `AGENTS.md` first and follow it as the active rule source.
3. Read `README.md` closely for project purpose, setup, architecture, release
   flow, git-crypt behavior, sessions, dashboards, profile workspaces, wiki,
   locked-mode behavior, launchers, and known local-state conventions.
4. Inspect repo state and history:
   - `git status --short --branch`
   - `git diff --stat`
   - `git diff --name-status`
   - `git diff --check`
   - `git log --oneline --decorate -30`
   - Inspect relevant commits with `git show --stat --summary --name-status`.
5. Build a project map:
   - List top-level folders.
   - Map `Website/`, `cmd/`, `START-HERE/`, `dev/`, `documentation/`, and
     `dev/backups/skills/`.
   - Include encrypted-content boundaries, profile workspaces, ignored local-state
     folders/files, generated artifacts, release assets, and binary locations.
6. Audit server behavior:
   - Review `cmd/rock-os/` entry points, route registration, file serving,
     markdown rendering, link-health APIs, session APIs, update/sync behavior,
     profile workspace authorization, script execution, git-crypt detection,
     and LAN/host-mode rules.
   - Look for path traversal, unrestricted file access, unsafe writes, command
     injection, arbitrary script execution, public exposure of encrypted
     content, weak locked-mode checks, and stale release behavior.
7. Audit frontend behavior:
   - Review `Website/index.html`, shared CSS, core JS, dashboards, profile workspaces,
     sessions, wiki modules, profile workspace navigation, unlocked profile
     cards, locked-mode UI, theme behavior, and link handling.
   - Look for locked-mode flashes, hidden-but-clickable UI, unsafe external
     links, remote dependencies, broken same-tab/new-tab rules, inconsistent
     theme handling, and duplicated fragile code.
8. Audit launchers and release tooling:
   - Review `START-HERE/` platform launchers and `dev/windows-create-release.ps1`
     when present.
   - Check update behavior with dirty repos, binary replacement behavior,
     fallback behavior, version-marker handling, generated files, GitHub release
     assumptions, and user-facing prompts.
9. Audit git-crypt and local state:
   - Check `.gitattributes`, `.gitignore`, `.key` handling, encrypted folder
     boundaries, ignored mutable files, generated indexes, caches, release
     outputs, local binaries, and session-state files.
   - Do not move or restructure locked encrypted content.
10. Audit scripts and dashboards:
   - Inspect profile script discovery rules and representative scripts.
   - Confirm script execution stays restricted to approved folders and supported
     extensions.
   - Inspect dashboard/profile folder conventions, widgets, icons, and markdown
     rendering paths.
11. Audit dependencies and validation:
   - Review dependency manifests and lock files.
   - Run available local tests/checks when appropriate and not destructive.
   - Run `go test ./...` from `cmd/rock-os` if server code is relevant and the
     environment allows it.
   - Run syntax checks for changed JavaScript.
   - Do not install new dependencies, fetch network vulnerability data, or run
     destructive commands without explicit user approval.
12. Audit Codex skills:
   - Read `dev/backups/skills/*/SKILL.md`.
   - Compare important project skills with active personal skills under
     `C:\Users\rocket\.agents\skills\` when matching active skills exist.
   - Note drift, stale instructions, or unsafe automation behavior.
13. Produce a detailed report:
   - Start with the highest-severity findings.
   - Use severity labels: `Critical`, `High`, `Medium`, `Low`, `Observation`.
   - Include file references, evidence, likely impact, and practical fixes.
   - Separate confirmed issues from suspected issues.
   - Include a project map and feature-behavior summary after findings.
   - Include validation commands run and anything that could not be checked.
   - Include release implications when server, launcher, or release tooling
     changes require a new binary.

## Rules

- Do not stage, commit, stash, push, or open pull requests.
- Do not modify files during Deep Dive unless the user explicitly asks for
  fixes.
- Prioritize completeness over token efficiency, but keep the final report
  organized and actionable.
- Do not present speculation as fact. Label uncertainty and explain what would
  confirm it.
- Do not move, rename, split, or restructure locked `Website/ENCRYPTED/`
  content.
- Do not run network scans, dependency downloads, vulnerability database
  lookups, destructive commands, or broad system scans unless the user
  explicitly asks.
- If a test or audit command is blocked by the environment, report the blocker
  and continue with static review.

---
name: recap
description: Use when the user invokes /recap, says Recap, asks Codex to get re-familiarized with Rock-OS, or asks to catch up before starting or continuing a Rock-OS session. Builds a token-efficient project briefing by reviewing AGENTS.md, README.md, repo structure, working tree state, recent commits, changed files, Codex skills, release-impact files, and basic project integrity signals.
---

# Recap

Build a compact but useful Rock-OS project briefing before beginning or continuing work.

## Workflow

1. Work from the Rock-OS repo root.
2. Read `AGENTS.md` first and treat it as the active rule source.
3. Read `README.md` for current project purpose, layout, launcher behavior, release flow, git-crypt notes, sessions, dashboards, and locked-mode behavior.
4. Inspect current repo state:
   - `git status --short --branch`
   - `git diff --stat`
   - `git diff --name-status`
   - `git diff --check`
5. Build a lightweight file-structure map:
   - List top-level folders.
   - List key folders under `Website/`, `cmd/`, `START-HERE/`, `dev/`, and `dev/backups/skills/`.
   - Note missing, new, renamed, or surprising structure only; do not dump the full tree.
6. Review recent history:
   - Inspect the last 12 commits with `git log -12 --oneline --decorate`.
   - For commits that look relevant or unfamiliar, inspect changed files with `git show --stat --summary --name-status <sha>`.
   - Prefer patterns and feature changes over long commit-by-commit narration.
7. Inspect current uncommitted changes:
   - Identify changed files by area: frontend, server, launcher scripts, release tooling, skills, docs, encrypted content.
   - Read only the changed files needed to understand current work.
8. Check project integrity signals:
   - Note whether encrypted content appears locked or unlocked.
   - Note whether ignored local-state files or `.key` files are present.
   - Note whether generated files, caches, binaries, or release artifacts appear in the working tree.
   - Note whether server-source changes imply a release binary may be needed.
   - Note whether launcher, session, git-crypt, dashboard, menu, wiki, or release behavior may be affected.
9. Inspect Codex skill changes:
   - List `dev/backups/skills/` directories.
   - Read any new or recently changed `dev/backups/skills/*/SKILL.md` files.
   - Compare with active personal skills under `C:\Users\rocket\.agents\skills\` when a matching active skill exists.
10. Summarize only what matters for the next task:
   - Current branch and dirty state.
   - Important repo rules.
   - Current project structure and notable feature areas.
   - Recent feature or integrity changes.
   - Relevant uncommitted changes.
   - Skill updates.
   - Risks, blockers, or release implications.

## Rules

- Do not stage, commit, stash, push, or open pull requests.
- Do not modify files during Recap unless the user explicitly asks for fixes.
- Keep the recap compact. Prefer conclusions over raw command output.
- Do not deep-read the whole repo. Use structure, diffs, status, and targeted file reads.
- If encrypted content is locked, report that fact and avoid moving or restructuring locked `Website/ENCRYPTED/` content.

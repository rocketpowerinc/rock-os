---
name: recap
description: Use when the user invokes /recap, says Recap, asks Codex to get re-familiarized with Rock-OS, or asks to catch up before starting or continuing a Rock-OS session. Reviews AGENTS.md, README.md, the last six commits, current working tree changes, and new or changed files under dev/codex-skills.
---

# Recap

Get current on the Rock-OS repo before beginning or continuing work.

## Workflow

1. Work from the Rock-OS repo root.
2. Read `AGENTS.md` first and treat it as the active rule source.
3. Read `README.md` for project context and current layout.
4. Inspect the current worktree:
   - `git status --short --branch`
   - `git diff --stat`
   - `git diff --name-status`
5. Review recent history:
   - `git log -6 --oneline --decorate`
   - For each of the last six commits, inspect at least the commit subject and changed files with `git show --stat --summary --name-status <sha>`.
6. Inspect skill changes:
   - List `dev/codex-skills/` directories.
   - Read any new or recently changed `dev/codex-skills/*/SKILL.md` files.
   - Compare with active personal skills under `C:\Users\rocket\.agents\skills\` when a matching active skill exists.
7. Summarize what matters for the next coding task:
   - Current branch and dirty state.
   - Major changes from the last six commits.
   - Relevant repo rules from `AGENTS.md`.
   - README layout or behavior changes that affect implementation.
   - New or changed Codex skills.
   - Any blockers, risky dirty files, or release implications.

## Rules

- Do not stage, commit, stash, push, or open pull requests.
- Do not modify files during Recap unless the user explicitly asks for fixes.
- Prefer concise summaries over dumping command output.
- If encrypted content is locked, report that fact and avoid moving or restructuring locked `Website/ENCRYPTED/` content.

---
name: features-new
description: Use when the user invokes /features-new, says Features New, asks for creative new Rock-OS feature ideas, wants next-level improvements, or wants a token-efficient brainstorm grounded in the current project. Generates practical, outside-the-box feature candidates with impact, effort, risks, and first implementation steps.
---

# Features New

Generate creative, useful, and project-fitting Rock-OS feature ideas without
burning unnecessary context. Favor ideas that make Rock-OS more local-first,
encrypted, personal, LAN-friendly, useful on mobile, and easier to share or
operate.

## Workflow

1. Work from the Rock-OS repo root.
2. Read `AGENTS.md` first and follow it as the active rule source.
3. Read enough of `README.md` to understand current purpose, architecture,
   release flow, git-crypt behavior, sessions, dashboards, profile workspaces,
   launchers, wiki, and locked-mode behavior.
4. Build a compact feature map:
   - Inspect top-level folders.
   - Inspect key files under `Website/`, `cmd/rock-os/`, `START-HERE/`,
     `dev/`, `documentation/`, and `dev/codex-skills/`.
   - Use `git status --short --branch` and `git diff --name-status` to account
     for current uncommitted work.
5. Identify opportunity areas:
   - Locked vs unlocked experience.
   - Sessions and role-based visibility.
   - Dashboards, profiles, widgets, profile cards, and markdown content.
   - Wiki/documentation workflows.
   - Profile scripts and safe local automation.
   - Git-crypt, privacy, local state, backup, and recovery.
   - Release/update/install experience.
   - Mobile and LAN use.
   - Health checks, observability, and self-diagnosis.
   - Personalization, onboarding, sharing, and future private-repo use.
6. Brainstorm broadly, then filter:
   - Keep ideas that fit Rock-OS's local-first constraints.
   - Avoid ideas that require cloud accounts, hosted databases, CDNs, or a
     frontend build step unless clearly optional.
   - Prefer features that can start small and grow.
   - Avoid duplicate ideas already implemented.
7. Return a ranked list:
   - `Quick Win`: small, useful, low-risk.
   - `High Leverage`: meaningful improvement with moderate effort.
   - `Moonshot`: ambitious but plausible.
   - For each idea include: why it matters, where it fits, rough effort, risks,
     and a first implementation step.
8. End with a practical shortlist:
   - Top 3 features to build next.
   - One feature to avoid for now and why.
   - Any release, security, git-crypt, or mobile implications.

## Rules

- Do not stage, commit, stash, push, or open pull requests.
- Do not modify files during Features New unless the user explicitly asks for
  implementation.
- Keep the scan token-efficient. Use structure, README, status, and targeted
  file reads instead of deep-reading the whole repo.
- Be creative, but do not ignore project constraints from `AGENTS.md`.
- Prefer concrete feature proposals over vague product language.
- Separate proven project facts from speculative opportunities.

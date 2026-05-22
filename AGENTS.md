# AGENTS.md

Guidance for AI coding agents working on Rock-OS.

## Project North Star

Rock-OS is a modern Internet-in-a-Box style project.

Think of it as a local-first knowledge, bootstrap, and resilience appliance:
part wiki, part offline reference library, part cross-platform setup guide, and
part doomsday-friendly intranet hub.

The project should feel practical, durable, and useful when normal cloud
assumptions fail.

## Core Principles

- Prefer local-first features over cloud-first features.
- Keep the site usable on a private LAN or fully offline intranet.
- Do not depend on external CDNs, icon services, fonts, or remote assets for the
  core website experience.
- Keep markdown content readable, useful, and easy to maintain.
- Favor simple files, clear folders, and predictable scripts over complex
  infrastructure.
- Support Windows, Linux, and macOS wherever practical.
- Treat mobile access as important because phones and tablets may be local
  clients during an outage.
- Keep private markdown compatible with `git-crypt`.
- Avoid features that require a hosted database, cloud account, or always-on
  internet connection unless clearly optional.

## Website Direction

The website should feel like a modern command center, not a generic blog or
marketing page.

- The first screen should communicate Rock-OS as a serious local bootstrap and
  knowledge system.
- The wiki should remain fast, readable, searchable, and easy to navigate.
- Theme work should support the existing presets: Steel, Rugged, Cyberpunk, and
  Blue-Grass.
- UI changes should stay professional and theme-aware.
- Keep visual assets local in `Website/assets`.
- Keep wiki frontend code organized as native browser modules under
  `Website/js/wiki/` when adding reusable rendering, navigation, search, or UI
  helpers. Do not add a frontend build step unless the user explicitly asks.
- Keep generated indexes, release binaries, caches, and downloaded artifacts out
  of Git unless there is a deliberate reason to track them.

## Markdown Writing Style

Markdown notes should be clear, practical, and a little human.

- Use the same handbook tone already established in the Linux notes.
- Explain why something matters, not just what command to run.
- Add mild humor where it helps the material feel readable, but do not let jokes
  get in the way of clarity.
- Prefer headings, short sections, tables, and focused lists over walls of text.
- For tools and links, include short descriptions and why they matter.
- For risky commands or destructive steps, include warnings.
- For security topics, stay practical and ethical.

## Scripts And Binaries

- Root launcher scripts should remain friendly, colored where useful, and
  explicit about what they are checking.
- Shell scripts committed to Git should have executable mode `100755`.
- If a new `.sh` script is added, make sure the user knows it should be
  committed with executable mode `100755`.
  The purpose is to prevent Linux/macOS users from needing manual `chmod`
  changes that dirty the working tree and block `git-crypt unlock`.
- Windows `.cmd` scripts should have matching behavior where practical.
- Windows scripts should remain readable when double-clicked. If a `.cmd` script
  finishes instead of staying open as a long-running server process, pause at
  the end so the user can read the output and close the window themselves.
- User-managed website scripts live in `Website/scripts/`. The script dashboard
  should only expose allowlisted script files from that folder and should never
  become an arbitrary web command prompt.
- Organize user scripts under platform folders such as `Website/scripts/Windows/`,
  `Website/scripts/Linux/`, and `Website/scripts/Mac/`. The dashboard should
  render those folders as a collapsible tree.
- Do not update `README.md` for every new script added under `Website/scripts/`.
  The scripts folder may grow to hundreds of entries. Put useful comments inside
  scripts instead so each script explains itself where it runs.
- The script dashboard supports `.cmd`, `.bat`, `.sh`, and `.ps1` files. Keep
  preview-before-run behavior. When the user clicks Run, launch the script in
  the operating system's terminal instead of a browser-rendered pseudo-terminal.
  This keeps `sudo`, prompts, and long-running commands simple and native.
- Start scripts should prefer release binaries, then fall back to Go source.
- Start scripts should check for Git repo updates first with a safe
  `git pull --ff-only`, warn if updating fails, and continue launching from the
  local copy.
- Quick install scripts are `install-rock-os.ps1` and `install-rock-os.sh`.
  They should create a `rock-os` terminal command and desktop launcher while using
  the existing `start-rock-os.cmd` and `start-rock-os.sh` launchers.
- If `Website/main.go` or server behavior changes, remind the user that a new
  release binary should be built and published.
- Do not require Go when a release binary is available.

## Private Markdown

Private markdown lives under:

```text
Website/markdown/Private/
```

This area is intended to be encrypted with `git-crypt`.

- Do not break `git-crypt` workflows.
- Do not remove key safety checks unless the user explicitly asks.
- Do not commit `.key` files.
- Remember that GitHub ZIP downloads are not real Git clones and cannot unlock
  `git-crypt` content.

## Development Habits

- Use existing project patterns before inventing new ones.
- Keep changes scoped to the user request.
- Update `README.md` when adding user-facing features, dependencies, or workflow
  changes. Do not update it for ordinary additions under `Website/scripts/`.
- Do not commit, stash, push, or open pull requests for the user unless they
  explicitly ask for that action.
- Do not stage files unless the user explicitly asks. The user prefers to review
  and commit changes themselves. Do not leave files staged, including executable
  mode changes. If a `.sh` file needs executable mode, explain the command for
  the user to run or do it only after an explicit request.
- Avoid unnecessary dependencies.
- Do not call external assets from the website unless the user explicitly wants
  an online-only feature.
- Before finishing, run a relevant sanity check such as `git diff --check`.

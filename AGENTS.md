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
- Internal Rock-OS links should open in the same browser tab. External
  `http`/`https` links should open in a new tab with `rel="noopener noreferrer"`.
- Profiles and Dashboards landing cards should stay clean: icon plus title
  only, no secondary subtitle such as "Open local dashboard". The Dashboards
  landing kicker should read `UNENCRYPTED DASHBOARDS`, and neither landing page
  needs an explanatory paragraph under the heading.
- Keep wiki frontend code organized as native browser modules under
  `Website/js/wiki/` when adding reusable rendering, navigation, search, or UI
  helpers. Do not add a frontend build step unless the user explicitly asks.
- Markdown-style tabs should use `createMarkdownTabApp` from
  `Website/js/wiki/markdown-tab.js` with a small config wrapper instead of
  cloning full tab JavaScript files.
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
- User-managed website scripts live in `Website/menu/scripts/`. The script dashboard
  should only expose allowlisted script files from that folder and should never
  become an arbitrary web command prompt.
- Organize user scripts under platform folders such as `Website/menu/scripts/Windows/`,
  `Website/menu/scripts/Linux/`, and `Website/menu/scripts/Mac/`. The dashboard should
  render those folders as a collapsible tree.
- Do not update `README.md` for every new script added under `Website/menu/scripts/`.
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
- User-facing launcher/helper scripts live under `START-HERE/Windows/`,
  `START-HERE/Linux/`, and `START-HERE/Mac/`. Keep the `START-HERE/` root clean
  except for `instructions.md`.
- Source-only `start-rock-os-from-source` helpers also live in the matching
  platform `START-HERE` folders, not in `Website/`. The
  `start-rock-os-from-source-lan` helpers are explicit LAN-mode wrappers for
  trusted local networks.
- Quick install scripts are `START-HERE/Windows/install-rock-os.ps1`,
  `START-HERE/Linux/install-rock-os.sh`, and
  `START-HERE/Mac/install-rock-os.sh`.
  They should create a `rock-os` terminal command and desktop launcher while using
  the matching platform `start-rock-os` launcher.
- Go server source lives under `cmd/rock-os/`; website content lives under
  `Website/`.
- Run Go server tests from `cmd/rock-os` with `go test ./...`.
- If `cmd/rock-os/main.go` or server behavior changes, remind the user that a new
  release binary should be built and published.
- Do not require Go when a release binary is available.



## Profiles

Profiles is the private markdown area. It lives under:

```text
Website/profiles/
```

This area is intended to be encrypted with `git-crypt`. The Profiles page
should show a locked state instead of rendering profile folders while the
content is still encrypted.

- Do not break `git-crypt` workflows.
- Do not remove key safety checks unless the user explicitly asks.
- Do not commit `.key` files.
- If the encrypted Profiles folder needs to be moved, renamed, split, or
  restructured, tell the user to unlock it first. Never move encrypted
  git-crypt content while it is locked, because that can corrupt ciphertext and
  make files fail to decrypt later.
- Remember that GitHub ZIP downloads are not real Git clones and cannot unlock
  `git-crypt` content.

## Dashboards

Dashboards are public local command-center pages. They live under:

```text
Website/dashboards/
```

Dashboards should behave like Profiles from a UI perspective: each dashboard
folder can have its own landing card, `dashboard.json`, `widgets.txt`,
markdown notes, sidebar search, favorites, and document view. Unlike Profiles,
Dashboards are not encrypted with `git-crypt` and should always be available.
Do not put sensitive private notes in Dashboards.
Dashboards are grouped under category folders, and `dashboards.html` should
render category sections dynamically from those containing folder names. Do not
hardcode dashboard categories. Dashboard names should preferably be one word so
URLs, folder names, CSS selectors, and routing stay simple. If the user proposes
a multi-word dashboard name, warn them and ask whether they want a one-word
version before scaffolding.

Profiles and Dashboards should use matching folder conventions. Each item
folder should use `index.html` as the entry page, with `dashboard.json`,
`widgets.txt`, `Overview.md`, optional local `assets/`, and other markdown
files beside it:

```text
Website/profiles/Rocket/index.html
Website/profiles/Rocket/Overview.md
Website/profiles/Rocket/assets/Rocket-Steel.svg
Website/dashboards/OS/Windows/index.html
Website/dashboards/OS/Windows/Overview.md
Website/dashboards/OS/Windows/assets/windows.png
```

Profile and dashboard page icons should live inside their respective profile or
dashboard folder. Shared widget/feed fallback icons live under
`Website/assets/widget-icons/`.

## Development Habits

- Use existing project patterns before inventing new ones.
- Keep changes scoped to the user request.
- Update `README.md` when adding user-facing features, dependencies, or workflow
  changes. Do not update it for ordinary additions under `Website/menu/scripts/`.
- When creating a new Codex skill for this project, install the active skill in
  the user's personal skills folder and also keep a repo backup copy under
  `dev/codex-skills/<skill-name>/`. The repo copy is archival only and should
  not be treated as the active loaded skill.
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

# AGENTS.md

Rules for AI coding agents working on Rock-OS. This file is the single source of
truth for agent behavior. Descriptive docs (install, layout, how things work)
live in `README.md` — read that for context, follow this for rules.

## Inviolable Rules

Never break these without an explicit, specific request from the user:

- **Never** commit, stage, stash, push, or open pull requests. The user reviews
  and commits everything themselves. Leave nothing staged — including
  executable-mode changes. The sole standing exception is the user-invoked
  `/release-new` workflow: it may run `dev/windows-create-release.ps1`,
  which stages, commits, pushes, and publishes the reviewed release changes.
- **Never** commit `.key` files.
- **Never** move, rename, split, or restructure locked `git-crypt` content. Tell
  the user to unlock `Website/ENCRYPTED/` first; moving locked ciphertext can
  corrupt it.
- **Never** depend on external CDNs, icon services, fonts, or remote assets for
  the core website experience. Keep visual assets local in `Website/assets`.
- **Never** turn the script dashboard into an arbitrary web command prompt. It
  may only expose allowlisted script files from `Website/ENCRYPTED/menu/scripts/`.
- **Never** add a frontend build step unless the user explicitly asks.
- **Never** track generated indexes, release binaries, caches, downloaded
  artifacts, or `Website/.rock-os-version` in Git.
- **Never** require Go when a release binary is available.
- **Never** let this file drift back into description. AGENTS.md stays a
  rules-only document: keep the "Inviolable Rules" section first, express
  everything as a directive an agent can follow, and put descriptive or
  how-it-works content in `README.md` instead. When adding a rule, place it in
  the matching topic section (or "Inviolable Rules" if it is a hard constraint)
  rather than creating sprawl.

## Scope & Defaults

- Keep changes scoped to the user request. Use existing project patterns before
  inventing new ones.
- Prefer local-first, offline-capable, LAN-friendly solutions. Avoid features
  that need a hosted database, cloud account, or always-on internet unless they
  are clearly optional.
- Treat mobile access as important — phones and tablets may be local clients.
- Support Windows, Linux, and macOS wherever practical.
- Avoid unnecessary dependencies.
- All text files use LF line endings, enforced by `.gitattributes`
  (`* text=auto eol=lf`; git-crypt content is exempt). Never introduce CRLF. If a
  tool rewrites a file with CRLF, normalize it back to LF before finishing.
- Before finishing, run a relevant sanity check such as `git diff --check`
  (it also flags stray CR characters).

## Website

- Internal Rock-OS links open in the same browser tab. External `http`/`https`
  links open in a new tab with `rel="noopener noreferrer"`.
- Keep locked-mode home-page launch cards under `Website/launch-point-cards-locked/`
  as public markdown files. Treat filename order as card order, and never put
  private content in that public folder.
- Keep dashboard session definitions in `Website/Sessions/sessions.json` and
  mutable active-session state in ignored `Website/Sessions/active-session.json`.
  Treat both as public routing state only; never put private content in that
  public folder.
- UI changes stay professional and theme-aware. Support the existing presets:
  Steel, Rugged, Cyberpunk, and Blue-Grass.
- Keep wiki frontend code organized as native browser modules under
  `Website/js/wiki/`. Do not clone full tab JS files — build markdown-style tabs
  with `createMarkdownTabApp` from `Website/js/wiki/markdown-tab.js` plus a small
  config wrapper.
- Dashboard landing cards stay clean: icon plus title only, no secondary
  subtitle. The Dashboards landing kicker reads `ENCRYPTED DASHBOARDS`. The
  landing page does not need an explanatory paragraph under the heading.
- Widgets are parsed and rendered by `Website/js/profiles.js` (shared by both
  Profiles and Dashboards) from each item's `widgets.txt`. When you add a new
  widget type or change an existing widget's `widgets.txt` fields, update
  `documentation/Widgets.md` in the same change so the widget guide always documents every
  available widget type and its supported fields.

## Markdown Writing Style

- Match the handbook tone established in the Linux notes: clear, practical, a
  little human.
- Explain why something matters, not just what command to run.
- Mild humor is fine where it aids readability; never let it obscure clarity.
- Prefer headings, short sections, tables, and focused lists over walls of text.
- For tools and links, include short descriptions and why they matter.
- For risky or destructive commands, include warnings. For security topics, stay
  practical and ethical.

## Scripts & Binaries

- User-managed website scripts live in `Website/ENCRYPTED/menu/scripts/`, organized under
  platform folders (`Windows/`, `Linux/`, `Mac/`) and rendered as a collapsible
  tree. Keep the dashboard preview-before-run; on Run, launch in the OS terminal,
  not a browser pseudo-terminal. Supported types: `.cmd`, `.bat`, `.sh`, `.ps1`.
- Do not update `README.md` for ordinary additions under
  `Website/ENCRYPTED/menu/scripts/`. Put self-explaining comments inside each script
  instead.
- Shell scripts committed to Git must have executable mode `100755`. If a new
  `.sh` is added, tell the user it should be committed `100755` so Linux/macOS
  users avoid manual `chmod` that dirties the tree and blocks `git-crypt unlock`.
  Do not apply the exec-mode change yourself unless explicitly asked.
- Windows `.cmd` scripts should mirror behavior where practical and stay readable
  when double-clicked: if a `.cmd` finishes (rather than running as a server),
  pause at the end so the user can read output.
- User-facing launcher/helper scripts live under `START-HERE/Windows/`,
  `START-HERE/Linux/`, and `START-HERE/Mac/`. Keep the `START-HERE/` root clean
  except for `instructions.md`. Source-only `start-rock-os-from-source` and
  `-lan` helpers live in those platform folders too, not in `Website/`.
- Start scripts prefer release binaries, then fall back to Go source. They check
  for repo updates first with a safe `git pull --ff-only`, warn if it fails, and
  continue launching from the local copy.
- Go server source lives under `cmd/rock-os/`; website content under `Website/`.
  Run server tests from `cmd/rock-os` with `go test ./...`.
- If `cmd/rock-os/main.go` or server behavior changes, remind the user to build
  and publish a new release binary.
- When the user invokes `/release-new`, follow the `release-new` Codex skill.
  Let `dev/windows-create-release.ps1` prompt the user for the version, then
  allow that script to stage, commit, push, and publish the release without
  additional confirmation. Do not create a release for HTML, CSS, JavaScript,
  markdown, or asset-only changes unless the user explicitly requests one.

## Encrypted Content (`Website/ENCRYPTED/`)

- Keep dashboards, profile items, menu markdown, and user-managed scripts under
  `Website/ENCRYPTED/` so `git-crypt` protects all user content.
- Show a locked state instead of rendering encrypted content while
  `Website/ENCRYPTED/` is still encrypted.
- Do not break `git-crypt` workflows or remove key safety checks unless the user
  explicitly asks.
- Remember that GitHub ZIP downloads are not real clones and cannot unlock
  `git-crypt` content.
- Never document local session marker files in README.md, public website
  content, or user-facing docs. Keep them ignored and local-only.

## Dashboards (`Website/ENCRYPTED/dashboards/`)

- Live under `Website/ENCRYPTED/dashboards/<Category>/<DashboardName>/`. Group by category
  folder; `dashboards.html` renders category sections dynamically — do not
  hardcode categories. Order categories with `Profiles` first, `OS` second,
  `Mobile` third, then the rest alphabetically. In `Homelab`, keep
  `SelfHosting` first and sort the rest alphabetically.
- Dashboard names should preferably be one word. If the user proposes a
  multi-word name, warn them and ask for a one-word version before scaffolding.
  If they give exact casing or punctuation, preserve it and update all
  path-sensitive references.
- Profiles stored under `Website/ENCRYPTED/dashboards/Profiles/` and Dashboards share
  folder conventions: each item folder uses
  `index.html` as the entry page, with `dashboard.json`, `widgets.txt`,
  `Overview.md`, optional local `assets/`, and other markdown beside it.
- Item icons live inside that item's own `assets/` folder, not global
  `Website/assets/`. Shared widget/feed fallback icons live under
  `Website/assets/widget-icons/`.
- After renaming a dashboard folder, update: `index.html`, `dashboard.json`,
  `widgets.txt`, CSS avatar class paths, landing card icon selectors, and
  internal dashboard links.

## Version Marker

- `Website/.rock-os-version` is local, ignored state used by launchers to detect
  whether the downloaded release binary matches GitHub's latest release.
- The first non-comment line is the release tag. Launchers ignore empty lines and
  lines starting with `#`. Do not track this file in Git.

## Codex Skills

- When creating a new Codex skill for this project, install the active skill in
  the user's personal skills folder and keep an archival backup copy under
  `dev/codex-skills/<skill-name>/`. The repo copy is archival only, not the
  active loaded skill.

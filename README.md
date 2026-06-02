# Rock-OS

Hardened for Collapse

Rock OS is a lightweight local wiki website for keeping markdown notes,
commands, media, and setup docs in a simple folder structure. It is served by a
small cross-platform Go binary that renders markdown locally before sending it
to the browser.

For project direction and AI agent development rules, see `AGENTS.md`.

## Contents

- [Quick Install](#quick-install) — one-line installers for each platform
- [Connection Modes](#connection-modes) — Local vs LAN, and the safety rules
- [Features](#features)
- [Release Binaries](#50-release-binaries) and [Dependencies](#dependencies)
- [Profiles and Dashboards](#profiles-and-dashboards)
- [Project Layout](#project-layout)
- [Running From a Release Binary](#running-from-a-release-binary) · [From Source](#running-from-source) · [Server Options](#server-options)
- [Local Script Dashboard](#local-script-dashboard)
- [Unlocking](#unlocking-profiles) / [Locking](#locking-profiles-again) Profiles (`git-crypt`)
- [How the Wiki Works](#how-the-wiki-works) · [Wiki Links](#wiki-links) · [Personal Pins](#personal-pins)
- [Offline Assets](#offline-assets) · [Markdown and Media](#markdown-and-media)
- [License](#license)

## License

Rock OS uses a split license:

- Code: GNU Affero General Public License v3.0
- Documentation and markdown content: Creative Commons
  Attribution-NonCommercial-ShareAlike 4.0 International
- Rock-OS name, logos, theme rocks, icons, slogans, and branding assets: all
  rights reserved unless otherwise stated

See [LICENSE.md](LICENSE.md) for the full project license notice.

## Connection Modes

Rock OS starts in **Local Mode** by default. In this mode the server binds to
`127.0.0.1`, so only the computer running Rock OS can open the site. This is the
safest everyday mode and is what the `rock-os` command, desktop launcher, and
start scripts use unless you explicitly ask for LAN access.

Use **LAN Mode** when you intentionally want other trusted devices in your home
network to connect:

```bash
rock-os lan
```

LAN Mode binds the server to the local network so phones, tablets, and other
computers can reach the wiki from an address such as `http://192.168.1.2:8000/`.
That is useful for an intranet, but it also means anyone on the same network may
be able to reach Rock OS. Avoid LAN Mode on public Wi-Fi, guest networks, hotels,
schools, coffee shops, or any network you do not control. Keep private markdown
locked when you do not need it, and stop the server when you are done sharing it.

For safety, script execution stays restricted to the computer running Rock OS by
default, even in LAN Mode. Other devices on the LAN can browse the site, but they
cannot launch scripts on the host machine unless Rock OS is started with the
explicit server flag:

```bash
--enable-lan-script-runs
```

Use that flag only on a trusted private network where every connected client is
allowed to launch scripts on the Rock OS host.

## Quick Install

These installers clone Rock OS into `~/rock-os`, create a `rock-os` terminal
command, add a desktop launcher, and start Rock-OS immediately.

### Install Location

The one-line installers use a normal Git clone, not a ZIP download.

| Platform | Repo install location | Terminal command created | Desktop launcher |
| --- | --- | --- | --- |
| Windows | `%USERPROFILE%\rock-os` | `%USERPROFILE%\Bin\rock-os.cmd` | `%USERPROFILE%\Desktop\Rock-OS.lnk` |
| Linux | `$HOME/rock-os` | `$HOME/.local/bin/rock-os` | `$HOME/Desktop/Rock-OS.desktop` when a Desktop folder exists |
| macOS | `$HOME/rock-os` | `$HOME/.local/bin/rock-os` | `$HOME/Desktop/Rock-OS.app` when a Desktop folder exists |

If the install folder already exists and is a Git clone, rerunning the one-line
installer updates it with `git pull --ff-only` and refreshes the launcher files.
If the folder exists but is not a Git clone, the installer stops so it does not
overwrite unrelated files.

### Windows PowerShell

```powershell
irm https://raw.githubusercontent.com/rocketpowerinc/rock-os/main/START-HERE/Windows/install-rock-os.ps1 | iex

```

Same line but it install git first
```powershell
if (-not (Get-Command git -ErrorAction SilentlyContinue)) { winget install --id Git.Git -e --source winget; $env:Path = [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User') }; irm https://raw.githubusercontent.com/rocketpowerinc/rock-os/main/START-HERE/Windows/install-rock-os.ps1 | iex
```

After install, open a new terminal and run:

```powershell
rock-os
```

### Linux

```bash
curl -fsSL https://raw.githubusercontent.com/rocketpowerinc/rock-os/main/START-HERE/Linux/install-rock-os.sh | sh
```

After install, open a new terminal and run:

```bash
rock-os
```

### macOS

```bash
curl -fsSL https://raw.githubusercontent.com/rocketpowerinc/rock-os/main/START-HERE/Mac/install-rock-os.sh | sh
```

After install, open a new terminal and run:

```bash
rock-os
```

The Windows installer creates a Start-compatible `rock-os.cmd` shim in
`%USERPROFILE%\Bin` and a desktop shortcut using the local Rock-OS icon. The
Linux/macOS installer creates `~/.local/bin/rock-os`; Linux also gets a `.desktop`
launcher when `~/Desktop` exists, while macOS gets a `Rock-OS.app` desktop
launcher.

Every launch through `rock-os`, the desktop launcher, or the start scripts checks
the Git repo for updates first with a safe fast-forward pull. If the machine is
offline or local changes block the update, Rock OS keeps starting from the local
copy and prints a warning.

## Features

- Markdown files rendered server-side by the local Go wiki server
- Theme-aware command center landing page with launch links and status panels
- Random landing page field notes loaded from `Website/quotes.md`
- Automatic sidebar tree from nested markdown folders
- Local script dashboard with search, personal pins, preview, guarded run buttons, and OS terminal launch
- Sidebar controls for refresh, expand all, fold all, and collapse
- Instant search across file names and markdown contents, with highlights in results and opened documents
- URL-based pages, such as `wiki.html?doc=menu/wiki/Linux/Setup.md`
- Last edited note shown above rendered markdown files
- Breadcrumbs show the current markdown folder path without changing pages
- Personal wiki and script pins appear at the top of each sidebar
- Auto-generated table of contents for longer markdown documents
- Internal markdown links open through the wiki page route
- Missing internal markdown links are visibly marked as broken
- Server-side link health scan for internal markdown, page, media, and asset links
- Backlinks show which markdown files reference the current page
- Code block copy buttons, inline code click-to-copy, language labels, line numbers, and highlighting
- Markdown callouts for notes, warnings, tips, errors, and related blocks
- Theme presets: Steel, Rugged, Cyberpunk, and Blue-Grass
- Local offline icons, favicons, and bookmark assets
- Gzip compression for text/API responses on slower local networks
- Local media support for images and videos kept out of Git
- Helper scripts for unlocking and re-locking private markdown
- Cross-platform Go server for Windows, Linux, and macOS

## 5.0 Release Binaries

Prebuilt binaries are available on the
[Rock-OS releases page](https://github.com/rocketpowerinc/rock-os/releases).

Choose the binary for your system:

| System | Binary |
| --- | --- |
| Windows 64-bit Intel/AMD | `rock-os-windows-amd64.exe` or `rock-os-vX.Y-windows-amd64.exe` |
| Windows ARM64 | `rock-os-windows-arm64.exe` or `rock-os-vX.Y-windows-arm64.exe` |
| Linux 64-bit Intel/AMD | `rock-os-linux-amd64` or `rock-os-vX.Y-linux-amd64` |
| Linux ARM64 | `rock-os-linux-arm64` or `rock-os-vX.Y-linux-arm64` |
| macOS Intel | `rock-os-macos-amd64` or `rock-os-vX.Y-macos-amd64` |
| macOS Apple Silicon | `rock-os-macos-arm64` or `rock-os-vX.Y-macos-arm64` |

The release also includes `rock-os-vX.Y-checksums.txt` for verifying
downloads.

### Creating A Release On Windows

Use the release helper after reviewing server-side changes:

```powershell
.\dev\windows-create-release.ps1
```

The helper prompts for the next version number, stages pending source changes,
rejects known secrets and generated artifacts, runs `git diff --cached --check`
and `go test ./...`, creates a release-preparation commit, cross-compiles all six
supported binaries, and writes the checksum file under `.release/`. It does not
push or publish by default. When you intentionally want the helper to push the
current branch and create the GitHub release, add `-Publish`:

```powershell
.\dev\windows-create-release.ps1 -Publish
```

The `/release-new` Codex skill runs the `-Publish` workflow in a visible
PowerShell window. The version prompt is the only interactive question.

## Profiles and Dashboards

The top navigation includes a **Profiles** tab between Home and Menu. When
`Website/profiles/` is locked, the Profiles page shows a locked panel instead
of listing private profile documents. After unlocking, it behaves like the other
markdown tabs and shows profile folders such as Rocket, Kids, and Prepper. Each
profile opens as its own dashboard with its own sidebar, search, favorites, and
document view.

The top navigation also includes **Dashboards** for always-available local
command centers that are not encrypted with `git-crypt`. Dashboard folders live
under category folders in `Website/dashboards/`; each dashboard can have its
own `dashboard.json`, `widgets.txt`, markdown notes, search, favorites, and
document view. The Dashboards page groups items dynamically by their containing
category folder, so you can add as many categories as you need. Use Profiles for
sensitive/private notes and Dashboards for public local tools or
platform-specific launch points.

| Area | Folder | Encryption | Best For |
| --- | --- | --- | --- |
| Profiles | `Website/profiles/` | Encrypted with `git-crypt` | Private notes, personal configs, sensitive references |
| Dashboards | `Website/dashboards/` | Not encrypted | Public local command centers, platform launch points, shared notes |

Profiles and Dashboards use the same folder convention:

```text
Website/profiles/Rocket/index.html
Website/dashboards/OS/Windows/index.html
```

The profile folder name is the profile name. Dashboard paths use
`Website/dashboards/<Category>/<DashboardName>/`, where the category becomes a
section heading on the Dashboards page. The `index.html` file is the entry page.
Use `Overview.md` for the first note, with `dashboard.json`, `widgets.txt`, an
optional local `assets/` folder, and additional markdown files beside it.
Profile and dashboard page icons should live inside that item's own folder, for
example:

```text
Website/profiles/Rocket/assets/Rocket-Steel.svg
Website/dashboards/OS/Windows/assets/windows.png
```

Shared widget/feed fallback icons live under `Website/assets/widget-icons/`.

Each `widgets.txt` can define dynamic dashboard widgets (YouTube, Spotify,
Reddit, podcasts, news feeds, bookmarks, featured spotlights, and clickable
file cards). For every widget type, its configuration fields, and copy-paste
examples, see the widget guide: [`documentation/Widgets.md`](documentation/Widgets.md).

Internal Rock OS links, such as `/scripts.html` or `/dashboards/OS/Windows/`, open
in the same browser tab. External web links open in a new tab so the local
dashboard stays available. Dashboard/profile cards can link directly to their
markdown tree by using `?view=notes`, for example
`/dashboards/OS/Windows/?view=notes`.

## Dependencies

You can run Rock OS from a release binary without Go installed. Go is only
needed if you want to run from source or if the start script cannot find a
matching release binary and falls back to the Go source under
`cmd/rock-os`.

The Go source uses `goldmark` for local server-side markdown rendering. It is
managed through `cmd/rock-os/go.mod` and is included automatically when you
build or run from source.

`git-crypt` is only needed for unlocking, editing, or re-locking private
markdown stored under `Website/profiles/`.

### Windows

Install Go with `winget`:

```powershell
winget install -e --id GoLang.Go
```

Add the default Go user binary folder to your PowerShell profile if it is not
already there:

```powershell
if (-not (Test-Path $PROFILE)) {
    New-Item -ItemType File -Path $PROFILE -Force | Out-Null
}

if (-not (Select-String -Path $PROFILE -Pattern '\$HOME\\go\\bin' -Quiet)) {
    Add-Content -Path $PROFILE -Value '$env:PATH = "$HOME\go\bin;" + $env:PATH'
    . $PROFILE
}
```

Install Scoop if you do not already have it:

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
Invoke-RestMethod -Uri https://get.scoop.sh | Invoke-Expression
scoop bucket add extras
```

Install `git-crypt`:

```powershell
scoop install main/git-crypt
```

### macOS

Install Homebrew if you do not already have it:

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

Load Homebrew in your shell. Apple Silicon Macs usually use
`/opt/homebrew/bin/brew`; Intel Macs usually use `/usr/local/bin/brew`.

Apple Silicon:

```bash
echo 'eval "$(/opt/homebrew/bin/brew shellenv)"' >> ~/.zprofile
eval "$(/opt/homebrew/bin/brew shellenv)"
```

Intel:

```bash
echo 'eval "$(/usr/local/bin/brew shellenv)"' >> ~/.zprofile
eval "$(/usr/local/bin/brew shellenv)"
```

Install Go and `git-crypt`:

```bash
brew install go git-crypt
```

Add the Go user binary folder to your shell path:

```bash
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

### Linux

Install Go and `git-crypt` with your distro package manager.

Ubuntu:

```bash
sudo apt update
sudo apt install git-crypt
sudo snap install go --classic
```

Debian:

```bash
sudo apt update
sudo apt install golang-go git-crypt
```

Fedora:

```bash
sudo dnf install golang git-crypt
```

Arch Linux:

```bash
sudo pacman -Syu go git-crypt
```

Add the Go user binary folder to your shell path:

```bash
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

## Project Layout

```text
cmd/rock-os/      Go server source and tests
START-HERE/            Human-friendly launcher folders for Windows, Linux, and macOS
Website/               HTML, CSS, JS, assets, and media
Website/menu/wiki/     Public wiki markdown
Website/menu/guides/   Guided setup markdown
Website/menu/cheatsheets/ Quick-reference markdown
Website/menu/dotfiles/ Dotfile notes and configs
Website/menu/bookmarks/ Bookmark collections and link notes
Website/profiles/      Encrypted/private profile markdown
Website/dashboards/    Public local dashboards grouped by category
Website/menu/scripts/  User-managed runnable scripts
```

For a plain-language guide to the launcher scripts, read
`START-HERE/instructions.md`.

Run Go tests from the server module:

```bash
cd cmd/rock-os
go test ./...
```

## Running From A Release Binary

Release binaries are normally kept inside the `Website` folder after cloning or
extracting the project. The server serves that folder as the site root.

### Windows

```powershell
cd Website
.\rock-os-windows-amd64.exe
```

### Linux

```bash
cd Website
chmod +x ./rock-os-linux-amd64
./rock-os-linux-amd64
```

### macOS

```bash
cd Website
chmod +x ./rock-os-macos-arm64
./rock-os-macos-arm64
```

## Running From Source

Install Go from [go.dev/dl](https://go.dev/dl/), then run:

```bash
cd cmd/rock-os
go run .
```

The server auto-detects the repo's `Website` folder from that location. You can
also pass `--site-root` explicitly when testing unusual layouts.

Source-only helper scripts are also available inside the platform folders under
`START-HERE`.
They skip release binaries, build a local dev binary from the current source,
and run that dev binary. By default, they bind to `127.0.0.1` so only the
current computer can connect:

```powershell
.\START-HERE\Windows\start-rock-os-from-source.cmd
```

```bash
./START-HERE/Linux/start-rock-os-from-source.sh
# or on macOS
./START-HERE/Mac/start-rock-os-from-source.sh
```

Use LAN mode only on a trusted home/private network:

```powershell
.\START-HERE\Windows\start-rock-os-from-source-lan.cmd
```

```bash
./START-HERE/Linux/start-rock-os-from-source-lan.sh
# or on macOS
./START-HERE/Mac/start-rock-os-from-source-lan.sh
```

The Windows helper intentionally builds a visible local dev binary instead of
using `go run`, because some Windows Application Control policies block the
hidden temporary executable that `go run` creates inside `.gocache`.

## Local Script Dashboard

Rock OS includes `Website/scripts.html`, a local dashboard for scripts stored in
`Website/menu/scripts/`. The dashboard lists allowed scripts, renders the script
contents for review, and only then enables a run button for scripts compatible
with the current operating system. When you click Run, the server opens the
script in the operating system's default terminal so normal prompts, `sudo`,
and long-running commands behave like they would outside the browser.

Refresh buttons inside Rock OS check for GitHub updates before reloading local
content. The server runs a fixed `git pull --ff-only`, so it only accepts clean
fast-forward updates and never overwrites local work. If the update check fails,
Rock OS warns you and still refreshes the files already on disk. Live update
requests are restricted to the computer running the server, even in LAN Mode:
other LAN clients can browse refreshed content, but they cannot make the host
pull code from GitHub.

Organize scripts into folders such as `Website/menu/scripts/Windows/`,
`Website/menu/scripts/Linux/`, and `Website/menu/scripts/Mac/`. The dashboard renders
those folders as a folded collapsible tree with an expand/fold-all control,
similar to the wiki sidebar.

For safety, the Go server only exposes scripts from `Website/menu/scripts/` and does
not provide an arbitrary command prompt. Supported script types are `.cmd`,
`.bat`, `.sh`, and `.ps1`. PowerShell scripts require PowerShell to be installed
on the machine running the Go server.

Script IDs are restricted to ordinary path characters, spaces, dots, dashes,
underscores, and supported script extensions. Shell metacharacters are rejected
before a script can be resolved or launched.

Helper scripts are also included:

```powershell
.\START-HERE\Windows\start-rock-os.cmd
```

```bash
./START-HERE/Linux/start-rock-os.sh
# or on macOS
./START-HERE/Mac/start-rock-os.sh
```

Check repo, Git, `git-crypt`, private markdown, local binaries, tools, and port
status:

```powershell
.\START-HERE\Windows\repo-status.cmd
```

```bash
./START-HERE/Linux/repo-status.sh
# or on macOS
./START-HERE/Mac/repo-status.sh
```

Linux and macOS shell scripts are committed with executable permissions. If the
permissions are missing because of a ZIP extraction, file copy, or unusual
filesystem, run:

```bash
chmod +x ./START-HERE/Linux/*.sh ./START-HERE/Mac/*.sh
```

This keeps fresh clones ready to run without needing manual `chmod` commands.
It also prevents Git from reporting chmod-only file changes, which can leave the
working tree dirty and block `git-crypt unlock`.

The helper scripts detect the current operating system and CPU architecture,
then check the latest GitHub release when internet is available. If the matching
stable latest-style binary is missing or older than the latest release, they
download it into the `Website` folder. If the release check or download is not
available, they continue with local files.

Run the helper scripts from a real Git clone, not a GitHub ZIP download. ZIP
downloads do not include the hidden `.git` folder, so `git-crypt` cannot unlock
private markdown. Clone the repo instead:

```bash
git clone https://github.com/rocketpowerinc/rock-os.git
cd rock-os
```

A ZIP download will still run. If `start-rock-os` does not find a `.git` folder,
it prints a prominent warning, waits for you to press Enter, then starts in a
limited mode: automatic `git pull` updates are skipped and `git-crypt` cannot
unlock Profiles. The public wiki, dashboards, scripts, and search all work
normally. Clone the repo when you want updates and private Profiles.

The start scripts show only launcher-side activity: the real `git pull
--ff-only` output, whether Go is available for source fallback, release binary
download/current messages, and the final launch handoff. If a newer release
binary is needed, the scripts also print the download attempt and whether it
succeeded.

Once the server starts, the Go binary prints the single colored status sanity
check and request log. That keeps startup output focused: scripts update and
launch, while Go reports the actual server status, private markdown state,
folders, host mode, and request activity.

After the update check, the scripts start a stable latest-style binary such as
`rock-os-windows-amd64.exe`, `rock-os-windows-arm64.exe`,
`rock-os-linux-amd64`, `rock-os-linux-arm64`,
`rock-os-macos-amd64`, or `rock-os-macos-arm64`. If that is not
present, they try a versioned binary such as
`rock-os-vX.Y-windows-amd64.exe`. If no matching binary is present, they
fall back to the Go source under `cmd/rock-os`.

When `cmd/rock-os/main.go` changes, rebuild and publish a new release
binary. The server binary is what carries LAN binding, port handling, markdown
rendering, markdown indexing, and other server-side behavior. Markdown, CSS,
HTML, JavaScript, and asset-only changes do not require a new Go binary.

To stop a running server on the default port:

```powershell
.\START-HERE\Windows\stop-rock-os.cmd
```

```bash
./START-HERE/Linux/stop-rock-os.sh
# or on macOS
./START-HERE/Mac/stop-rock-os.sh
```

Pass a port number if you started Rock-OS on a different port:

```bash
./START-HERE/Linux/stop-rock-os.sh 8001
```

By default, the server listens on port `8000`, opens the site in your browser,
and binds to `127.0.0.1` (see [Connection Modes](#connection-modes)). To share
with other devices on a trusted network, append `lan`:

```powershell
.\START-HERE\Windows\start-rock-os.cmd lan
```

```bash
./START-HERE/Linux/start-rock-os.sh lan
# or on macOS
./START-HERE/Mac/start-rock-os.sh lan
```

## Server Options

```bash
cd cmd/rock-os
go run . --host local
go run . --host 127.0.0.1
go run . --host 0.0.0.0
go run . --port 9000
go run . --open=false
go run . --build-index
go run . --site-root ../../Website
```

Use `--host 127.0.0.1` to serve only on the current computer. Use `--host local`
or `--host lan` only when you intentionally want other devices on the trusted
LAN to connect. Use `--build-index` to rebuild all local tab index JSON files
without starting the server. The server usually finds `Website` automatically, but
`--site-root` is available for custom layouts.

## Unlocking Profiles

This repo can use `git-crypt` for Profiles, which is the private markdown
area stored under:

```text
Website/profiles/
```

Those files can be committed to the public repo, but their contents are stored
encrypted on GitHub. File and folder names are still visible, so avoid sensitive
names.

The exported `git-crypt` key is ignored by Git through `*.key` in `.gitignore`.
Do not commit the key.

### Fresh Clone Unlock Steps

1. Install `git-crypt` using the dependency instructions above.

2. Clone the repo and enter it.

```bash
git clone https://github.com/rocketpowerinc/rock-os.git
cd rock-os
```

3. Copy your exported `.key` file into the repo root.

Example:

```text
rock-os/
  your-git-crypt-key.key
  START-HERE/
    Windows/
      unlock-git-crypt.cmd
    Linux/
      unlock-git-crypt.sh
    Mac/
      unlock-git-crypt.sh
  Website/
```

The unlock script copies the key to a temporary system folder, removes the root
copy, unlocks the repo, then copies the key back to the repo root. Removing the
root copy during unlock keeps Git clean enough for `git-crypt unlock` to run.
The restored `.key` file is ignored by Git.

4. Unlock the repo.

Windows:

```powershell
.\START-HERE\Windows\unlock-git-crypt.cmd
```

macOS or Linux:

```bash
./START-HERE/Linux/unlock-git-crypt.sh
# or on macOS
./START-HERE/Mac/unlock-git-crypt.sh
```

The unlock scripts expect exactly one `.key` file in the repo root. After
unlocking, files in `Website/profiles/` should become readable, and the
key should be restored back to the repo root.

Check status:

```bash
git-crypt status
```

To export a key from an already-unlocked trusted clone:

```bash
git-crypt export-key rock-os-git-crypt.key
```

Store exported keys somewhere private and backed up, outside this repository.

### Locking Profiles Again

To re-lock Profiles after you are done editing:

Windows:

```powershell
.\START-HERE\Windows\lock-git-crypt.cmd
```

macOS or Linux:

```bash
./START-HERE/Linux/lock-git-crypt.sh
# or on macOS
./START-HERE/Mac/lock-git-crypt.sh
```

The lock scripts run `git-crypt lock` from the repo root. If locking fails,
close open private files or commit/stash pending private changes, then try
again.

## How The Wiki Works

The Go server scans:

```text
Website/menu/wiki/
```

The server exposes:

```text
Website/wiki-index.json
Website/guides-index.json
Website/cheatsheets-index.json
Website/dotfiles-index.json
Website/bookmarks-index.json
Website/profiles-index.json
```

The browser reads the matching index endpoint and builds each sidebar tree. When
you open a document, it asks the Go server to render that markdown through the
matching local API endpoint, then the browser adds wiki features such as code
copy buttons, callouts, backlinks, and the table of contents.

The generated `*-index.json` files are local state and are intentionally ignored
by Git. That keeps local private or experimental markdown files from constantly
dirtying the repo or leaking filenames into commits.

Raw HTML inside markdown is omitted for safety. That protects the local script
dashboard and other same-origin APIs from malicious markdown files.

Example:

```text
Website/menu/wiki/
  Linux/
    AnduinOS/
      Guide.md
```

## Wiki Links

Internal links between markdown files should use normal relative markdown links:

```markdown
[GNOME Cheat Sheet](../Cheat%20Sheets/Gnome-CheatSheet.md)
```

When rendered in the wiki, internal `.md` links are opened through `wiki.html`
with a `doc` parameter instead of navigating to the raw file directly.

If an internal `.md` link points to a file that is not in the current tab index,
the wiki marks it as missing so broken links are visible while editing.

Rock OS also includes a `link-health.html` page for viewing broken local links in the
browser, plus a server-side link health report at:

```text
/api/health/links
```

The scanner walks local markdown sources under the public menu folders,
dashboards, and unlocked profiles. It verifies internal markdown links, local
HTML pages, dashboard/profile folders, media, and asset paths against the real
filesystem. External `http` and `https` links are counted but not fetched, so the
scan stays local-first and does not leak browsing intent to the internet.

If a broken link is intentional, such as a planned future page, add this marker
after the link on the same line:

```markdown
[KDE](../../cheatsheets/Linux/KDE-CheatSheet.md) <!-- rock-os-ignore-link -->
```

Use this sparingly. If a page was deleted and the link is no longer useful,
remove the link instead of hiding it from the report.

Direct wiki URLs look like this:

```text
wiki.html?doc=menu/wiki/Linux/Cheat%20Sheets/Gnome-CheatSheet.md
```

## Personal Pins

The wiki and script dashboard both include small pin buttons beside each item.
Pinned items appear above the normal sidebar tree while still remaining in their
regular folder location.

Pins are personal browser settings stored in `localStorage`. They do not edit
markdown files, do not edit scripts, do not create Git changes, and are not
shared automatically with other browsers or devices on the LAN.

## Offline Assets

The website is designed to work locally on an intranet. Theme images, folder
icons, favicons, Apple touch icons, and manifest icons are stored inside the
repo under `Website/assets` or embedded directly in the local HTML/JS/CSS.

The site does not need external icon CDNs or remote assets for the wiki UI.
Wiki code highlighting is also local: Highlight.js and the Bash/PowerShell
language files are vendored under `Website/js/vendor/`.

## Markdown And Media
The wiki supports common markdown features including images, links, tables,
lists, code blocks, and callouts. Raw HTML is not rendered, so use normal
markdown links for videos and other media files.

Large images and videos should live in:

```text
Website/media/
```

That folder is ignored by Git so the repository stays small. After cloning the
project on another computer, download your media ZIP and extract it back into
`Website/media/`.

Use local media in markdown like this:

```markdown
![Screenshot](/media/screenshots/setup.png)

[Demo video](/media/videos/demo.mp4)
```

When you update media, zip `Website/media/` again and store the ZIP wherever you
keep your private media backup.

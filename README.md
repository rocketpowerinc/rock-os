# rock-os

Hardened for Collapse

Rock OS is a lightweight local wiki website for keeping markdown notes,
commands, media, and setup docs in a simple folder structure. It is served by a
small cross-platform Go binary and rendered in the browser.

## Features

- Markdown files rendered as a website wiki
- Automatic sidebar tree from nested markdown folders
- Instant search across file names and markdown contents
- URL-based pages, such as `wiki.html?doc=markdown/Linux/Setup.md`
- Last edited note shown above rendered markdown files
- Internal markdown links open through the wiki page route
- Missing internal markdown links are visibly marked as broken
- Code block copy buttons, language labels, line numbers, and highlighting
- Markdown callouts for notes, warnings, tips, errors, and related blocks
- Theme presets: Steel, Rugged, Cyberpunk, and Blue-Grass
- Local offline icons, favicons, and bookmark assets
- Local media support for images and videos kept out of Git
- Cross-platform Go server for Windows, Linux, and macOS

## 1.0 Release Binaries

Prebuilt binaries are available on the
[Rock OS 1.0 release page](https://github.com/rocketpowerinc/rock-os/releases/tag/v1.0).

Choose the binary for your system:

| System | Binary |
| --- | --- |
| Windows 64-bit Intel/AMD | `rock-os-wiki-v1.0-windows-amd64.exe` |
| Windows ARM64 | `rock-os-wiki-v1.0-windows-arm64.exe` |
| Linux 64-bit Intel/AMD | `rock-os-wiki-v1.0-linux-amd64` |
| Linux ARM64 | `rock-os-wiki-v1.0-linux-arm64` |
| macOS Intel | `rock-os-wiki-v1.0-macos-amd64` |
| macOS Apple Silicon | `rock-os-wiki-v1.0-macos-arm64` |

The release also includes `rock-os-wiki-v1.0-checksums.txt` for verifying
downloads.

## Running From A Release Binary

The wiki server serves files from the current directory, so run the binary from
inside the `Website` folder after cloning or extracting the project.

### Windows

```powershell
cd Website
.\rock-os-wiki-v1.0-windows-amd64.exe
```

### Linux

```bash
cd Website
chmod +x ./rock-os-wiki-v1.0-linux-amd64
./rock-os-wiki-v1.0-linux-amd64
```

### macOS

```bash
cd Website
chmod +x ./rock-os-wiki-v1.0-macos-arm64
./rock-os-wiki-v1.0-macos-arm64
```

## Running From Source

Install Go from [go.dev/dl](https://go.dev/dl/), then run:

```bash
cd Website
go run .
```

Helper scripts are also included:

```powershell
cd Website
.\start-rock-os.cmd
```

```bash
cd Website
sh start-rock-os.sh
```

By default, the server listens on port `8000`, opens the site in your browser,
and uses your local network IP when available. Other devices on the same network
can open the printed LAN URL.

## Server Options

```bash
go run . --host local
go run . --host 127.0.0.1
go run . --host 0.0.0.0
go run . --port 9000
go run . --open=false
go run . --build-index
```

Use `--host 127.0.0.1` to serve only on the current computer. Use
`--build-index` to rebuild `markdown-index.json` without starting the server.

## How The Wiki Works

The Go server scans:

```text
Website/markdown/
```

It writes:

```text
Website/markdown-index.json
```

The browser reads that JSON file, builds the sidebar tree, fetches the selected
markdown file, and renders it into the page.

Example:

```text
Website/markdown/
  Linux/
    AnduinOS/
      Bootstrap.md
```

## Wiki Links

Internal links between markdown files should use normal relative markdown links:

```markdown
[GNOME Cheat Sheet](../Cheat%20Sheets/Gnome-CheatSheet.md)
```

When rendered in the wiki, internal `.md` links are opened through `wiki.html`
with a `doc` parameter instead of navigating to the raw file directly.

If an internal `.md` link points to a file that is not in
`Website/markdown-index.json`, the wiki marks it as missing so broken links are
visible while editing.

Direct wiki URLs look like this:

```text
wiki.html?doc=markdown/Linux/Cheat%20Sheets/Gnome-CheatSheet.md
```

## Offline Assets

The website is designed to work locally on an intranet. Theme images, folder
icons, favicons, Apple touch icons, and manifest icons are stored inside the
repo under `Website/assets` or embedded directly in the local HTML/JS/CSS.

The site does not need external icon CDNs or remote assets for the wiki UI.

## Markdown And Media

The wiki supports common markdown features including images, links, videos,
tables, lists, code blocks, callouts, and HTML embeds.

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

<video controls src="/media/videos/demo.mp4"></video>
```

When you update media, zip `Website/media/` again and store the ZIP wherever you
keep your private media backup.

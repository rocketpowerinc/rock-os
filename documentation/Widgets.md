# Rock-OS Dashboard Widgets Guide

You can define custom, dynamic widgets on your profile dashboards by editing the relevant `widgets.txt` configuration file. Profiles use `Website/ENCRYPTED/Sessions/<SessionName>/Profiles/<ProfileName>/widgets.txt`; profile-owned dashboards use `Website/ENCRYPTED/Sessions/<SessionName>/Profiles/<ProfileName>/dashboards/<Category>/<DashboardName>/widgets.txt`.

---

## Quick Reference: Available Widget Types

| Type Name   | Source URL Options                                                                                                                                     | Render Behavior                                                                                                                           |
| :---------- | :----------------------------------------------------------------------------------------------------------------------------------------------------- | :---------------------------------------------------------------------------------------------------------------------------------------- |
| `youtube`   | YouTube channel, playlist, or query URLs                                                                                                               | Carousel cards or lists with video thumbnails                                                                                             |
| `spotify`   | Spotify playlists, tracks, albums, or shows                                                                                                            | Carousel cards or lists with cover art (oEmbed resolved)                                                                                  |
| `reddit`    | Subreddit names or full Reddit thread URLs                                                                                                             | Carousel cards or lists with post thumbnails                                                                                              |
| `podcast`   | XML podcast feed URLs or Apple Podcast URLs                                                                                                            | Carousel cards or lists with podcast cover art                                                                                            |
| `bookmarks` | `Name \| Description \| URL` entries                                                                                                                   | Standard vertical lists or a grid of nice banner cards                                                                                    |
| `featuring` | `Name \| Description \| URL` entries                                                                                                                   | Extra Big, highly prominent static spotlight banners                                                                                      |
| `news`      | Google News section/topic URLs, standard news links, or direct RSS feeds                                                                               | Auto-discovered article feeds with thumbnails                                                                                             |
| `files`     | `name_of_file` + `path` per file, with optional `description` and `command` lines (legacy `url = Name \| Path \| Description \| CopyText` still works) | Grid of file-icon cards; clicking copies the path (or `command` if given) to the clipboard, hovering shows the optional description popup |

---

## Global Configuration Parameters

All parameters must be placed in a named section block, e.g. `[My Block Title]`. The section header is automatically rendered as the card title in the user interface.

- **`type`** (Required): The application type (`youtube`, `spotify`, `reddit`, `podcast`, `bookmarks`, `featuring`, `news`, `files`).
- **`limit`** (Optional, Defaults to `5`): Maximum number of recent items to render (ignored by `bookmarks` and `featuring`).
- **`badge`** (Optional, Defaults to type accent): Top-right small metadata tag text in the UI.
- **`layout`** (Optional, Defaults to `vertical` for feeds, `horizontal` for bookmarks):
  - `vertical`: A scrolling list of rows.
  - `horizontal` / `banners`: A sideways scrolling carousel of cards (or grid of banners for bookmarks).
- **`card_size`** (Optional, Defaults to `medium`): Configures the grid column span/width of the widget card container:
  - `small`: Spans 1 grid column.
  - `medium`: Spans 2 grid columns on tablet/desktop layout.
  - `large`: Spans the full width of the dashboard grid (`1 / -1`).
- **`link_size`** (Optional, Defaults to `medium`): Configures the dimensions and visibility of content items/links inside the card:
  - `small`: Hides thumbnails/artwork entirely, showing compact text links.
  - `medium`: Standard default item dimensions.
  - `large`: Prominent content sizes (e.g. `100x100px` thumbnails for feeds, full-width banner columns for bookmarks, or large spotlight cards).
- **`url`** (Required): The source URL(s). Specify multiple `url = ...` lines to aggregate feeds or bookmarks together in one widget.

---

## Copy-Paste Examples

### 1. YouTube Widget

```ini
[Music Playlist]
type = youtube
limit = 3
badge = YouTube
layout = horizontal
card_size = large
link_size = large
url = https://www.youtube.com/channel/UCElGBUWDCa05jRzc2PfmGqQ
url = https://www.youtube.com/@chriswebby
```

### 2. Spotify Widget

```ini
[Daily Lo-Fi Beats]
type = spotify
limit = 5
badge = Lo-Fi
layout = horizontal
card_size = medium
link_size = medium
url = https://open.spotify.com/playlist/37i9dQZF1DX8Uebhnv3mq1
```

### 3. Reddit Widget

```ini
[Prepper Community]
type = reddit
limit = 5
badge = Reddit
layout = vertical
card_size = small
link_size = small
url = https://www.reddit.com/r/preppers
```

### 4. Podcast Widget

```ini
[Science Brains]
type = podcast
limit = 3
badge = Science
layout = horizontal
card_size = large
link_size = large
url = https://podcasts.apple.com/us/podcast/brains-on-science-podcast-for-kids/id669189128
```

### 5. Bookmarks Widget

```ini
[Linux Software]
type = bookmarks
badge = Flatpak
layout = horizontal
card_size = large
link_size = large
url = Endless Key | Offline educational resources and learning guide | https://flathub.org/en/apps/org.endlessos.Key
url = Bible GUI | Offline Bible reader and study app | https://flathub.org/en/apps/net.lugsole.bible_gui
```

### 6. Featuring Widget

```ini
[Featured Core]
type = featuring
badge = Core Spotlight
card_size = large
link_size = large
url = Rock-OS System | The local-first intranet and command center core | https://github.com/rocketpowerinc/rock-os
url = Endless Key | Offline learning resources & setup guide | https://flathub.org/en/apps/org.endlessos.Key
```

### 7. News Widget

```ini
[Tech & Gaming News]
type = news
limit = 5
badge = Live Feed
layout = vertical
card_size = large
link_size = large
url = https://news.google.com/topics/CAAqJggKIiBDQkFTRWdvSUwyMHZNRGRqTVhZU0FtVnVHZ0pWVXlnQVAB?hl=en-US&gl=US&ceid=US%3Aen
url = https://www.ign.com/ca
```

### 8. Files Widget

Each file entry is written with its own keys on separate lines. Start a new file with `name_of_file`, then give its `path`. You can optionally add a `description` line (shown as a hover popup explaining what the file does) and a `command` line (copied to the clipboard instead of the path — handy for an admin launch command you can paste into a terminal or "Run" dialog). The `path`, `description`, and `command` lines attach to the `name_of_file` directly above them, so keep each file's lines grouped together. Clicking a card copies the path (or the `command`, if given) to the clipboard. For security reasons the widget never opens or launches files directly — it only copies text to the clipboard.

```ini
[System Files]
type = files
badge = Files
layout = horizontal
card_size = medium
link_size = medium

name_of_file = hosts-file
path = C:\Windows\System32\drivers\etc\hosts
description = Maps hostnames to IP addresses. Edit to block or redirect domains locally (opens file in Notepad as admin).
command = powershell -Command "Start-Process notepad 'C:\Windows\System32\drivers\etc\hosts' -Verb RunAs"

name_of_file = .wslconfig
path = C:\Users\rocket\.wslconfig
description = Global WSL2 settings (memory, CPU, swap, networking) for all distros. Create it if missing; run "wsl --shutdown" after editing. Lives in your user profile, so no admin needed.
command = notepad C:\Users\rocket\.wslconfig

name_of_file = wsl.conf
path = \\wsl$\Ubuntu\etc\wsl.conf
description = Per-distro WSL settings (systemd, automount, hostname). Edit as root inside WSL, then "wsl --shutdown". Change "Ubuntu" to your distro name from "wsl -l -v".
command = wsl -d Ubuntu -u root -e nano /etc/wsl.conf
```

If `name_of_file` is omitted, the file name is derived from the `path`. The older single-line form `url = Name | Path | Description | CopyText` still works if you prefer it.

A few patterns worth noting from the examples above: the `hosts-file` card copies an elevated PowerShell command because that file needs admin to save; the `.wslconfig` card lives in your own user profile so a plain `notepad` command (no elevation) is enough; and the `wsl.conf` card points at a path inside the WSL filesystem (`\\wsl$\...`) but copies a `wsl` command to edit it as root, since that file isn't editable from Windows directly. The `path` is shown to the user, while the `command` is what actually gets copied — so you can display a friendly location and still hand over the exact launch command.

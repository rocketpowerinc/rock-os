# Rock-OS Dashboard Widgets Guide

You can define custom, dynamic widgets on your profile dashboards by editing your profile's `widgets.txt` configuration file (located at `Website/profiles/<ProfileName>/widgets.txt`).

---

## Quick Reference: Available Widget Types

| Type Name | Source URL Options | Render Behavior |
| :--- | :--- | :--- |
| `youtube` | YouTube channel, playlist, or query URLs | Carousel cards or lists with video thumbnails |
| `spotify` | Spotify playlists, tracks, albums, or shows | Carousel cards or lists with cover art (oEmbed resolved) |
| `reddit` | Subreddit names or full Reddit thread URLs | Carousel cards or lists with post thumbnails |
| `podcast` | XML podcast feed URLs or Apple Podcast URLs | Carousel cards or lists with podcast cover art |
| `bookmarks` | `Name \| Description \| URL` entries | Standard vertical lists or a grid of nice banner cards |
| `featuring` | `Name \| Description \| URL` entries | Extra Big, highly prominent static spotlight banners |
| `news` | Google News section/topic URLs, standard news links, or direct RSS feeds | Auto-discovered article feeds with thumbnails |

---

## Global Configuration Parameters

All parameters must be placed in a named section block, e.g. `[My Block Title]`. The section header is automatically rendered as the card title in the user interface.

* **`type`** (Required): The application type (`youtube`, `spotify`, `reddit`, `podcast`, `bookmarks`, `featuring`, `news`).
* **`limit`** (Optional, Defaults to `5`): Maximum number of recent items to render (ignored by `bookmarks` and `featuring`).
* **`badge`** (Optional, Defaults to type accent): Top-right small metadata tag text in the UI.
* **`layout`** (Optional, Defaults to `vertical` for feeds, `horizontal` for bookmarks):
  * `vertical`: A scrolling list of rows.
  * `horizontal` / `banners`: A sideways scrolling carousel of cards (or grid of banners for bookmarks).
* **`card_size`** (Optional, Defaults to `medium`): Configures the grid column span/width of the widget card container:
  * `small`: Spans 1 grid column.
  * `medium`: Spans 2 grid columns on tablet/desktop layout.
  * `large`: Spans the full width of the dashboard grid (`1 / -1`).
* **`link_size`** (Optional, Defaults to `medium`): Configures the dimensions and visibility of content items/links inside the card:
  * `small`: Hides thumbnails/artwork entirely, showing compact text links.
  * `medium`: Standard default item dimensions.
  * `large`: Prominent content sizes (e.g. `100x100px` thumbnails for feeds, full-width banner columns for bookmarks, or large spotlight cards).
* **`url`** (Required): The source URL(s). Specify multiple `url = ...` lines to aggregate feeds or bookmarks together in one widget.

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

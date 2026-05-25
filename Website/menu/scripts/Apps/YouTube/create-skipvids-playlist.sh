#!/usr/bin/env bash
set -u

# Rock-OS SkipVids playlist helper.
# No YouTube API key is used. This reads public YouTube channel RSS feeds,
# grabs the latest 3 videos per artist, asks YouTube to generate a temporary
# playlist URL, then opens that playlist through SkipVids so songs play one
# after another.
#
# This still avoids the YouTube Data API. The temporary playlist is created by
# YouTube's public watch_videos endpoint, then handed to SkipVids as a normal
# playlist ID.

CHANNELS=(
  "https://www.youtube.com/channel/UCHcb3FQivl6xCRcHC2zjdkQ"
  "https://www.youtube.com/@sindhuworld"
)

fetch_url() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL -A "Rock-OS SkipVids Playlist" "$1"
    return
  fi

  if command -v wget >/dev/null 2>&1; then
    wget -qO- --user-agent="Rock-OS SkipVids Playlist" "$1"
    return
  fi

  printf '%s\n' "Missing dependency: install curl or wget."
  return 1
}

html_escape() {
  sed \
    -e 's/&/\&amp;/g' \
    -e 's/</\&lt;/g' \
    -e 's/>/\&gt;/g' \
    -e 's/"/\&quot;/g'
}

resolve_channel_id() {
  case "$1" in
    *"/channel/"*)
      printf '%s\n' "$1" | sed -n 's#.*youtube\.com/channel/\([^/?&]*\).*#\1#p'
      return
      ;;
  esac

  page="$(fetch_url "$1")" || return 1
  printf '%s\n' "$page" |
    grep -oE '("channelId":"UC[^"]+|itemprop="channelId" content="UC[^"]+|youtube\.com/channel/UC[^"?&/]+)' |
    head -n 1 |
    sed -E 's/.*(UC[A-Za-z0-9_-]+).*/\1/'
}

playlist_items=""
all_video_ids=()

resolve_skipvids_playlist_url() {
  ids_csv="$1"
  youtube_playlist_url="https://www.youtube.com/watch_videos?video_ids=${ids_csv}&title=Rock-OS%20SkipVids%20Playlist"

  if command -v curl >/dev/null 2>&1; then
    final_url="$(curl -Ls -o /dev/null -w '%{url_effective}' -A "Rock-OS SkipVids Playlist" "$youtube_playlist_url")"
  elif command -v wget >/dev/null 2>&1; then
    final_url="$(wget -qO- --max-redirect=10 --server-response --user-agent="Rock-OS SkipVids Playlist" "$youtube_playlist_url" 2>&1 | awk '/^  Location: /{url=$2} END{print url}')"
  else
    final_url=""
  fi

  playlist_id="$(printf '%s\n' "$final_url" | sed -n 's/.*[?&]list=\([^&]*\).*/\1/p')"
  if [ -n "$playlist_id" ]; then
    printf '%s\n' "https://skipvids.com/playlist?list=$playlist_id"
    return
  fi

  printf '%s\n' "$youtube_playlist_url"
}

for channel in "${CHANNELS[@]}"; do
  channel_id="$(resolve_channel_id "$channel")"
  if [ -z "$channel_id" ]; then
    printf '%s\n' "Could not resolve channel: $channel"
    continue
  fi

  rss="$(fetch_url "https://www.youtube.com/feeds/videos.xml?channel_id=$channel_id")" || continue

  video_ids=()
  while IFS= read -r video_id; do
    video_ids+=("$video_id")
  done < <(printf '%s\n' "$rss" | grep -oE '<yt:videoId>[^<]+' | sed 's#<yt:videoId>##' | head -n 3)

  titles=()
  while IFS= read -r title; do
    titles+=("$title")
  done < <(printf '%s\n' "$rss" | grep -oE '<title>[^<]+' | sed 's#<title>##' | tail -n +2 | head -n 3)

  for i in "${!video_ids[@]}"; do
    video_id="${video_ids[$i]}"
    title="${titles[$i]:-$video_id}"
    title_html="$(printf '%s' "$title" | html_escape)"
    youtube_url="https://www.youtube.com/watch?v=$video_id"
    skipvid_url="https://skipvids.com/?v=$video_id"

    all_video_ids+=("$video_id")
    playlist_items="${playlist_items}
      <article class=\"video-card\">
        <img src=\"https://i.ytimg.com/vi/$video_id/hqdefault.jpg\" alt=\"\" loading=\"lazy\">
        <div>
          <h2>$title_html</h2>
          <p>$channel_id</p>
          <a href=\"$skipvid_url\" target=\"_blank\" rel=\"noopener noreferrer\">Open in SkipVids</a>
          <a href=\"$youtube_url\" target=\"_blank\" rel=\"noopener noreferrer\">YouTube source</a>
        </div>
      </article>"
  done
done

if [ "${#all_video_ids[@]}" -eq 0 ]; then
  printf '%s\n' "No videos found. Playlist was not created."
  exit 1
fi

ids_csv="$(IFS=,; printf '%s' "${all_video_ids[*]}")"
skipvid_playlist_url="$(resolve_skipvids_playlist_url "$ids_csv")"

downloads="${HOME}/Downloads"
mkdir -p "$downloads"
output="${downloads}/rock-os-skipvids-playlist.html"

cat > "$output" <<HTML
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Rock-OS SkipVids Playlist</title>
  <style>
    body { margin: 0; background: #101010; color: #f5f5f5; font-family: Arial, sans-serif; }
    main { max-width: 980px; margin: 0 auto; padding: 32px 18px; }
    h1 { margin: 0 0 8px; }
    .sub { color: #b9b9b9; margin: 0 0 24px; }
    .actions { display: flex; flex-wrap: wrap; gap: 10px; margin-bottom: 22px; }
    button, a { border: 1px solid #4f6475; color: #f5f5f5; background: #202a33; padding: 10px 12px; text-decoration: none; font-weight: 700; }
    .video-card { display: grid; grid-template-columns: 168px 1fr; gap: 16px; border: 1px solid #333; background: #171717; padding: 14px; margin-bottom: 14px; }
    .video-card img { width: 168px; max-width: 100%; aspect-ratio: 16 / 9; object-fit: cover; background: #000; }
    .video-card h2 { font-size: 1rem; margin: 0 0 8px; }
    .video-card p { color: #aaa; font-size: .82rem; margin: 0 0 12px; }
    .video-card a { display: inline-block; margin: 0 8px 8px 0; }
    @media (max-width: 640px) { .video-card { grid-template-columns: 1fr; } .video-card img { width: 100%; } }
  </style>
</head>
<body>
  <main>
    <h1>Rock-OS SkipVids Playlist</h1>
    <p class="sub">Latest 3 uploads from each configured artist. Built without a YouTube API key.</p>
    <div class="actions">
      <a href="$skipvid_playlist_url" target="_blank" rel="noopener noreferrer">Open Sequential Playlist In SkipVids</a>
    </div>
    $playlist_items
  </main>
</body>
</html>
HTML

printf '%s\n' "Created playlist:"
printf '%s\n' "$output"
printf '%s\n' "SkipVids playlist:"
printf '%s\n' "$skipvid_playlist_url"

if command -v xdg-open >/dev/null 2>&1; then
  xdg-open "$skipvid_playlist_url" >/dev/null 2>&1 &
elif command -v open >/dev/null 2>&1; then
  open "$skipvid_playlist_url" >/dev/null 2>&1 &
else
  printf '%s\n' "Open the SkipVids playlist URL above in your browser."
fi

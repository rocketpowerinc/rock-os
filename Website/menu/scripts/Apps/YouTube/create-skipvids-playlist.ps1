# Rock-OS SkipVids playlist helper.
# No YouTube API key is used. This reads public YouTube channel RSS feeds,
# grabs the latest 3 videos per artist, asks YouTube to generate a temporary
# playlist URL, then opens that playlist through SkipVids so songs play one
# after another.
#
# This still avoids the YouTube Data API. The temporary playlist is created by
# YouTube's public watch_videos endpoint, then handed to SkipVids as a normal
# playlist ID.

$ErrorActionPreference = "Stop"

$channels = @(
    "https://www.youtube.com/channel/UCHcb3FQivl6xCRcHC2zjdkQ",
    "https://www.youtube.com/@sindhuworld"
)

function Get-RockOsWebContent {
    param([Parameter(Mandatory = $true)][string]$Url)

    $response = Invoke-WebRequest `
        -Uri $Url `
        -UseBasicParsing `
        -Headers @{ "User-Agent" = "Rock-OS SkipVids Playlist" }

    return $response.Content
}

function Get-ChannelId {
    param([Parameter(Mandatory = $true)][string]$Url)

    if ($Url -match "/channel/([A-Za-z0-9_-]+)") {
        return $Matches[1]
    }

    $page = Get-RockOsWebContent -Url $Url
    $patterns = @(
        '"channelId":"(UC[A-Za-z0-9_-]+)"',
        'itemprop="channelId"\s+content="(UC[A-Za-z0-9_-]+)"',
        'youtube\.com/channel/(UC[A-Za-z0-9_-]+)'
    )

    foreach ($pattern in $patterns) {
        if ($page -match $pattern) {
            return $Matches[1]
        }
    }

    return $null
}

function ConvertTo-HtmlText {
    param([string]$Value)
    return [System.Net.WebUtility]::HtmlEncode($Value)
}

function Get-SkipVidsPlaylistUrl {
    param([Parameter(Mandatory = $true)][string[]]$VideoIds)

    $ids = $VideoIds -join ","
    $youtubePlaylistUrl = "https://www.youtube.com/watch_videos?video_ids=$ids&title=Rock-OS%20SkipVids%20Playlist"

    $handler = [System.Net.Http.HttpClientHandler]::new()
    $handler.AllowAutoRedirect = $true
    $client = [System.Net.Http.HttpClient]::new($handler)
    $client.DefaultRequestHeaders.UserAgent.ParseAdd("Rock-OS SkipVids Playlist")

    try {
        $response = $client.GetAsync($youtubePlaylistUrl).GetAwaiter().GetResult()
        $finalUrl = $response.RequestMessage.RequestUri.AbsoluteUri
    }
    finally {
        if ($response) {
            $response.Dispose()
        }
        $client.Dispose()
        $handler.Dispose()
    }

    if ($finalUrl -match "[?&]list=([^&]+)") {
        return "https://skipvids.com/playlist?list=$([System.Uri]::EscapeDataString($Matches[1]))"
    }

    return $youtubePlaylistUrl
}

$items = New-Object System.Collections.Generic.List[object]

foreach ($channel in $channels) {
    $channelId = Get-ChannelId -Url $channel
    if (-not $channelId) {
        Write-Warning "Could not resolve channel: $channel"
        continue
    }

    $rss = Get-RockOsWebContent -Url "https://www.youtube.com/feeds/videos.xml?channel_id=$channelId"
    $entries = [regex]::Matches($rss, "(?s)<entry>(.*?)</entry>")

    foreach ($entry in $entries | Select-Object -First 3) {
        $entryText = $entry.Groups[1].Value
        $videoId = [regex]::Match($entryText, "<yt:videoId>([^<]+)</yt:videoId>").Groups[1].Value
        $title = [regex]::Match($entryText, "<title>([^<]+)</title>").Groups[1].Value

        if (-not $videoId) {
            continue
        }

        $items.Add([pscustomobject]@{
            ChannelId = $channelId
            VideoId = $videoId
            Title = [System.Net.WebUtility]::HtmlDecode($title)
            YouTubeUrl = "https://www.youtube.com/watch?v=$videoId"
            SkipVidsUrl = "https://skipvids.com/?v=$videoId"
            Thumbnail = "https://i.ytimg.com/vi/$videoId/hqdefault.jpg"
        })
    }
}

if ($items.Count -eq 0) {
    Write-Host "No videos found. Playlist was not created."
    exit 1
}

$skipVidsPlaylistUrl = Get-SkipVidsPlaylistUrl -VideoIds ($items | ForEach-Object { $_.VideoId })

$downloads = Join-Path $HOME "Downloads"
New-Item -ItemType Directory -Force -Path $downloads | Out-Null
$output = Join-Path $downloads "rock-os-skipvids-playlist.html"

$cards = foreach ($item in $items) {
@"
      <article class="video-card">
        <img src="$($item.Thumbnail)" alt="" loading="lazy">
        <div>
          <h2>$(ConvertTo-HtmlText $item.Title)</h2>
          <p>$($item.ChannelId)</p>
          <a href="$($item.SkipVidsUrl)" target="_blank" rel="noopener noreferrer">Open in SkipVids</a>
          <a href="$($item.YouTubeUrl)" target="_blank" rel="noopener noreferrer">YouTube source</a>
        </div>
      </article>
"@
}

$html = @"
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
      <a href="$skipVidsPlaylistUrl" target="_blank" rel="noopener noreferrer">Open Sequential Playlist In SkipVids</a>
    </div>
$($cards -join "`n")
  </main>
</body>
</html>
"@

Set-Content -Path $output -Value $html -Encoding UTF8

Write-Host "Created playlist:"
Write-Host $output
Write-Host "SkipVids playlist:"
Write-Host $skipVidsPlaylistUrl
Start-Process $skipVidsPlaylistUrl

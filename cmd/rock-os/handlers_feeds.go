package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type feedItem struct {
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Created     string    `json:"created"`
	Author      string    `json:"author,omitempty"`
	Source      string    `json:"source,omitempty"`
	Thumbnail   string    `json:"thumbnail"`
	PublishTime time.Time `json:"-"`
}

var blockedServerFetchPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("::/128"),
	netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:2::/48"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("ff00::/8"),
}

func newPublicFetchClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: timeout}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}

			ips, err := resolvePublicFetchHost(ctx, host)
			if err != nil {
				return nil, err
			}
			if len(ips) == 0 {
				return nil, fmt.Errorf("could not resolve feed host %s", host)
			}

			return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
		},
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return http.ErrUseLastResponse
			}
			return validatePublicFetchURL(req.URL.String())
		},
	}
}

func clampFeedLimit(limit int) int {
	if limit <= 0 {
		return defaultFeedLimit
	}
	if limit > maxFeedLimit {
		return maxFeedLimit
	}
	return limit
}

func remoteResponseBodyReader(resp *http.Response) (io.Reader, error) {
	if resp.ContentLength > maxRemoteFeedResponseSize {
		return nil, fmt.Errorf("remote response too large: %d bytes", resp.ContentLength)
	}
	return io.LimitReader(resp.Body, maxRemoteFeedResponseSize), nil
}

func readRemoteResponseBody(resp *http.Response) ([]byte, error) {
	if resp.ContentLength > maxRemoteFeedResponseSize {
		return nil, fmt.Errorf("remote response too large: %d bytes", resp.ContentLength)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxRemoteFeedResponseSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxRemoteFeedResponseSize {
		return nil, fmt.Errorf("remote response exceeded %d bytes", maxRemoteFeedResponseSize)
	}
	return body, nil
}

func validatePublicFetchURL(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("url is required")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("invalid feed URL: %s", rawURL)
	}
	if parsed.User != nil {
		return fmt.Errorf("feed URL userinfo is not allowed")
	}

	_, err = resolvePublicFetchHost(context.Background(), parsed.Hostname())
	return err
}

func resolvePublicFetchHost(ctx context.Context, host string) ([]net.IP, error) {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	if host == "" {
		return nil, fmt.Errorf("feed URL host is required")
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return nil, fmt.Errorf("access to local network is restricted: %s", host)
	}

	if addr, err := netip.ParseAddr(host); err == nil {
		if err := validatePublicFetchAddr(addr, host); err != nil {
			return nil, err
		}
		return []net.IP{net.ParseIP(host)}, nil
	}

	resolved, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("could not resolve feed host %s: %w", host, err)
	}
	if len(resolved) == 0 {
		return nil, fmt.Errorf("could not resolve feed host %s", host)
	}

	ips := make([]net.IP, 0, len(resolved))
	for _, resolvedIP := range resolved {
		addr, ok := netip.AddrFromSlice(resolvedIP.IP)
		if !ok {
			return nil, fmt.Errorf("could not parse resolved IP for %s", host)
		}
		if err := validatePublicFetchAddr(addr, host); err != nil {
			return nil, err
		}
		ips = append(ips, resolvedIP.IP)
	}

	return ips, nil
}

func validatePublicFetchAddr(addr netip.Addr, host string) error {
	addr = addr.Unmap()
	for _, prefix := range blockedServerFetchPrefixes {
		if prefix.Contains(addr) {
			return fmt.Errorf("access to local network is restricted: %s", host)
		}
	}
	return nil
}

type redditChild struct {
	Data struct {
		Title      string  `json:"title"`
		Permalink  string  `json:"permalink"`
		CreatedUTC float64 `json:"created_utc"`
		Author     string  `json:"author"`
		Thumbnail  string  `json:"thumbnail"`
	} `json:"data"`
}

type redditResponse struct {
	Data struct {
		Children []redditChild `json:"children"`
	} `json:"data"`
}

type ytFeed struct {
	XMLName xml.Name  `xml:"feed"`
	Entries []ytEntry `xml:"entry"`
}

type ytEntry struct {
	Title     string `xml:"title"`
	VideoID   string `xml:"videoId"`
	Published string `xml:"published"`
	Link      ytLink `xml:"link"`
}

type ytLink struct {
	Href string `xml:"href,attr"`
}

func feedRedditHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		subreddit := r.URL.Query().Get("subreddit")
		redditURL := r.URL.Query().Get("url")

		if redditURL != "" {
			if u, err := url.Parse(redditURL); err == nil {
				path := strings.Trim(u.Path, "/")
				parts := strings.Split(path, "/")
				if len(parts) >= 2 && parts[0] == "r" {
					subreddit = parts[1]
				}
			}
		}

		if subreddit == "" {
			http.Error(w, "subreddit or url parameter is required", http.StatusBadRequest)
			return
		}

		// Sanitize subreddit: alphanumeric and underscores only, max 50 chars
		for _, char := range subreddit {
			if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_') {
				http.Error(w, "invalid subreddit parameter", http.StatusBadRequest)
				return
			}
		}
		if len(subreddit) > 50 {
			http.Error(w, "subreddit parameter is too long", http.StatusBadRequest)
			return
		}

		cacheDir := filepath.Join(siteRoot, ".gocache", "feeds")
		cachePath := filepath.Join(cacheDir, fmt.Sprintf("reddit_%s.json", subreddit))

		// Try to fetch live feed
		items, err := fetchLiveRedditFeed(subreddit)
		if err == nil {
			// Save to cache
			_ = os.MkdirAll(cacheDir, 0755)
			if data, err := json.Marshal(items); err == nil {
				_ = os.WriteFile(cachePath, data, 0644)
			}
			writeJSON(w, items)
			return
		}

		// Fallback to cache
		if data, err := os.ReadFile(cachePath); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(data)
			return
		}

		// No cache and offline/failed - return empty list so frontend fallback can load
		writeJSON(w, []feedItem{})
	}
}

func fetchLiveRedditFeed(subreddit string) ([]feedItem, error) {
	urlStr := fmt.Sprintf("https://www.reddit.com/r/%s/new.json?limit=5", subreddit)
	client := newPublicFetchClient(5 * time.Second)
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Rock-OS/1.0.0 (by rocketpowerinc)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reddit API returned HTTP %d", resp.StatusCode)
	}

	body, err := remoteResponseBodyReader(resp)
	if err != nil {
		return nil, err
	}

	var redditRes redditResponse
	if err := json.NewDecoder(body).Decode(&redditRes); err != nil {
		return nil, err
	}

	items := make([]feedItem, 0, len(redditRes.Data.Children))
	for _, child := range redditRes.Data.Children {
		createdStr := ""
		if child.Data.CreatedUTC > 0 {
			t := time.Unix(int64(child.Data.CreatedUTC), 0)
			createdStr = t.Format("2006-01-02")
		}

		thumb := child.Data.Thumbnail
		if thumb == "" || !strings.HasPrefix(thumb, "http") {
			thumb = ""
		}

		items = append(items, feedItem{
			Title:     child.Data.Title,
			URL:       "https://www.reddit.com" + child.Data.Permalink,
			Created:   createdStr,
			Author:    "u/" + child.Data.Author,
			Thumbnail: thumb,
		})
	}

	return items, nil
}

func feedYoutubeHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		channelIDs := r.URL.Query()["channel_id"]
		playlistIDs := r.URL.Query()["playlist_id"]
		urls := r.URL.Query()["url"]
		if len(urls)+len(channelIDs)+len(playlistIDs) > maxFeedURLParams {
			http.Error(w, "too many feed parameters", http.StatusBadRequest)
			return
		}

		for _, rawURL := range urls {
			t, val := resolveYoutubeURLToID(rawURL, siteRoot)
			if t == "channel_id" {
				channelIDs = append(channelIDs, val)
			} else if t == "playlist_id" {
				playlistIDs = append(playlistIDs, val)
			}
		}

		limitStr := r.URL.Query().Get("limit")

		if len(channelIDs) == 0 && len(playlistIDs) == 0 {
			http.Error(w, "at least one channel_id, playlist_id, or url parameter is required", http.StatusBadRequest)
			return
		}

		limit := defaultFeedLimit
		if limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
				limit = l
			}
		}
		limit = clampFeedLimit(limit)

		// Sanitize all IDs
		for _, id := range channelIDs {
			if !isValidFeedID(id) {
				http.Error(w, "invalid channel_id parameter", http.StatusBadRequest)
				return
			}
		}
		for _, id := range playlistIDs {
			if !isValidFeedID(id) {
				http.Error(w, "invalid playlist_id parameter", http.StatusBadRequest)
				return
			}
		}

		// Cache path based on hash of the sorted IDs to ensure consistent caching
		sortedIDs := make([]string, len(channelIDs)+len(playlistIDs))
		copy(sortedIDs, channelIDs)
		copy(sortedIDs[len(channelIDs):], playlistIDs)
		sort.Strings(sortedIDs)

		hasher := sha256.New()
		for _, id := range sortedIDs {
			hasher.Write([]byte(id))
		}
		cacheFilename := fmt.Sprintf("youtube_%x_l%d.json", hasher.Sum(nil), limit)
		cacheDir := filepath.Join(siteRoot, ".gocache", "feeds")
		cachePath := filepath.Join(cacheDir, cacheFilename)

		// Try to fetch live feed
		items, err := fetchCombinedYoutubeFeed(channelIDs, playlistIDs, limit)
		if err == nil {
			// Save to cache
			_ = os.MkdirAll(cacheDir, 0755)
			if data, err := json.Marshal(items); err == nil {
				_ = os.WriteFile(cachePath, data, 0644)
			}
			writeJSON(w, items)
			return
		}

		// Fallback to cache
		if data, err := os.ReadFile(cachePath); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(data)
			return
		}

		// Return empty list on failure
		writeJSON(w, []feedItem{})
	}
}

func isValidFeedID(id string) bool {
	if len(id) == 0 || len(id) > 100 {
		return false
	}
	for _, char := range id {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_' || char == '-') {
			return false
		}
	}
	return true
}

func fetchCombinedYoutubeFeed(channelIDs []string, playlistIDs []string, limit int) ([]feedItem, error) {
	type result struct {
		items []feedItem
		err   error
	}

	totalFeeds := len(channelIDs) + len(playlistIDs)
	ch := make(chan result, totalFeeds)

	for _, id := range channelIDs {
		go func(id string) {
			urlStr := fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?channel_id=%s", id)
			items, err := fetchSingleYoutubeFeed(urlStr)
			ch <- result{items, err}
		}(id)
	}

	for _, id := range playlistIDs {
		go func(id string) {
			urlStr := fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?playlist_id=%s", id)
			items, err := fetchSingleYoutubeFeed(urlStr)
			ch <- result{items, err}
		}(id)
	}

	var combined []feedItem
	var firstErr error

	for i := 0; i < totalFeeds; i++ {
		res := <-ch
		if res.err != nil {
			if firstErr == nil {
				firstErr = res.err
			}
		} else {
			combined = append(combined, res.items...)
		}
	}

	if len(combined) == 0 && firstErr != nil {
		return nil, firstErr
	}

	// Sort combined feed items by PublishTime descending (newest first)
	sort.Slice(combined, func(i, j int) bool {
		return combined[i].PublishTime.After(combined[j].PublishTime)
	})

	// Slice to limit
	if len(combined) > limit {
		combined = combined[:limit]
	}

	return combined, nil
}

func fetchSingleYoutubeFeed(urlStr string) ([]feedItem, error) {
	client := newPublicFetchClient(5 * time.Second)
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Rock-OS/1.0.0 (by rocketpowerinc)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("youtube API returned HTTP %d", resp.StatusCode)
	}

	body, err := remoteResponseBodyReader(resp)
	if err != nil {
		return nil, err
	}

	var yt ytFeed
	if err := xml.NewDecoder(body).Decode(&yt); err != nil {
		return nil, err
	}

	items := make([]feedItem, 0, len(yt.Entries))
	for _, entry := range yt.Entries {
		var pubTime time.Time
		dateStr := ""
		if entry.Published != "" {
			if t, err := time.Parse(time.RFC3339, entry.Published); err == nil {
				pubTime = t
				dateStr = t.Format("2006-01-02")
			} else {
				dateStr = entry.Published
			}
		}

		videoID := entry.VideoID
		if videoID == "" && entry.Link.Href != "" {
			videoID = extractYoutubeVideoID(entry.Link.Href)
		}

		thumb := ""
		if videoID != "" {
			thumb = fmt.Sprintf("https://i.ytimg.com/vi/%s/mqdefault.jpg", videoID)
		}

		linkURL := entry.Link.Href
		if linkURL == "" && videoID != "" {
			linkURL = fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
		}

		items = append(items, feedItem{
			Title:       entry.Title,
			URL:         linkURL,
			Created:     dateStr,
			Thumbnail:   thumb,
			PublishTime: pubTime,
		})
	}

	return items, nil
}

func extractYoutubeVideoID(urlStr string) string {
	if strings.Contains(urlStr, "v=") {
		parts := strings.Split(urlStr, "v=")
		if len(parts) > 1 {
			subparts := strings.Split(parts[1], "&")
			return subparts[0]
		}
	}
	if strings.Contains(urlStr, "youtu.be/") {
		parts := strings.Split(urlStr, "youtu.be/")
		if len(parts) > 1 {
			subparts := strings.Split(parts[1], "?")
			return subparts[0]
		}
	}
	if strings.Contains(urlStr, "embed/") {
		parts := strings.Split(urlStr, "embed/")
		if len(parts) > 1 {
			subparts := strings.Split(parts[1], "?")
			return subparts[0]
		}
	}
	return ""
}

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string      `xml:"title"`
	Image       rssImage    `xml:"image"`
	ItunesImage itunesImage `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd image"`
	Items       []rssItem   `xml:"item"`
}

type rssImage struct {
	URL string `xml:"url"`
}

type itunesImage struct {
	Href string `xml:"href,attr"`
}

type rssItem struct {
	Title          string       `xml:"title"`
	Link           string       `xml:"link"`
	PubDate        string       `xml:"pubDate"`
	Description    string       `xml:"description"`
	Image          rssImage     `xml:"image"`
	Enclosure      rssEnclosure `xml:"enclosure"`
	MediaThumbnail mediaImage   `xml:"http://search.yahoo.com/mrss/ thumbnail"`
	MediaContent   mediaImage   `xml:"http://search.yahoo.com/mrss/ content"`
}

type rssEnclosure struct {
	URL  string `xml:"url,attr"`
	Type string `xml:"type,attr"`
}

type mediaImage struct {
	URL    string `xml:"url,attr"`
	Medium string `xml:"medium,attr"`
	Type   string `xml:"type,attr"`
}

func feedPodcastHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		feedURL := r.URL.Query().Get("url")
		if feedURL == "" {
			http.Error(w, "url parameter is required", http.StatusBadRequest)
			return
		}

		feedURL = resolvePodcastURLToFeed(feedURL, siteRoot)
		if err := validatePublicFetchURL(feedURL); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		// Cache path based on hash of the feed URL
		hasher := sha256.New()
		hasher.Write([]byte(feedURL))
		cacheFilename := fmt.Sprintf("podcast_%x.json", hasher.Sum(nil))
		cacheDir := filepath.Join(siteRoot, ".gocache", "feeds")
		cachePath := filepath.Join(cacheDir, cacheFilename)

		// Try to fetch live feed
		items, err := fetchLivePodcastFeed(feedURL)
		if err == nil {
			// Save to cache
			_ = os.MkdirAll(cacheDir, 0755)
			if data, err := json.Marshal(items); err == nil {
				_ = os.WriteFile(cachePath, data, 0644)
			}
			writeJSON(w, items)
			return
		}

		// Fallback to cache
		if data, err := os.ReadFile(cachePath); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(data)
			return
		}

		// Return empty list on failure
		writeJSON(w, []feedItem{})
	}
}

func fetchLivePodcastFeed(feedURL string) ([]feedItem, error) {
	client := newPublicFetchClient(8 * time.Second)
	req, err := http.NewRequest("GET", feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Rock-OS/1.0.0 (by rocketpowerinc)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("podcast RSS returned HTTP %d", resp.StatusCode)
	}

	body, err := remoteResponseBodyReader(resp)
	if err != nil {
		return nil, err
	}

	var rss rssFeed
	if err := xml.NewDecoder(body).Decode(&rss); err != nil {
		return nil, err
	}

	channelThumb := rss.Channel.Image.URL
	if channelThumb == "" {
		channelThumb = rss.Channel.ItunesImage.Href
	}

	limit := len(rss.Channel.Items)
	if limit > 5 {
		limit = 5
	}

	items := make([]feedItem, 0, limit)
	for i := 0; i < limit; i++ {
		item := rss.Channel.Items[i]
		dateStr := ""
		var pubTime time.Time
		if item.PubDate != "" {
			pubTime = parseRssDate(item.PubDate)
			if !pubTime.IsZero() {
				dateStr = pubTime.Format("2006-01-02")
			} else {
				dateStr = item.PubDate
			}
		}

		items = append(items, feedItem{
			Title:       item.Title,
			URL:         item.Link,
			Created:     dateStr,
			Thumbnail:   channelThumb,
			PublishTime: pubTime,
		})
	}

	return items, nil
}

func parseRssDate(dateStr string) time.Time {
	dateStr = strings.TrimSpace(dateStr)
	// Try RFC1123
	t, err := time.Parse(time.RFC1123, dateStr)
	if err == nil {
		return t
	}
	// Try RFC1123Z
	t, err = time.Parse(time.RFC1123Z, dateStr)
	if err == nil {
		return t
	}
	// Try common layouts
	formats := []string{
		"Mon, _2 Jan 2006 15:04:05 MST",
		"Mon, _2 Jan 2006 15:04:05 -0700",
		"Mon, _2 Jan 2006 15:04:05 Z",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, format := range formats {
		t, err = time.Parse(format, dateStr)
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

var (
	channelIDRegex  = regexp.MustCompile(`/channel/(UC[a-zA-Z0-9_-]{22})`)
	itemPropRegex   = regexp.MustCompile(`<meta itemprop="channelId" content="(UC[a-zA-Z0-9_-]{22})">`)
	jsonChanIDRegex = regexp.MustCompile(`"channelId":"(UC[a-zA-Z0-9_-]{22})"`)
)

func resolveYoutubeURLToID(inputURL string, siteRoot string) (paramType string, paramVal string) {
	inputURL = strings.TrimSpace(inputURL)
	if inputURL == "" {
		return "", ""
	}

	// 1. Try to extract from local cache
	cachePath := filepath.Join(siteRoot, ".gocache", "resolved_urls.json")
	type CacheEntry struct {
		Type string `json:"type"`
		Val  string `json:"val"`
	}
	var cache map[string]CacheEntry

	if data, err := os.ReadFile(cachePath); err == nil {
		_ = json.Unmarshal(data, &cache)
	}
	if cache == nil {
		cache = make(map[string]CacheEntry)
	}

	if entry, ok := cache[inputURL]; ok {
		return entry.Type, entry.Val
	}

	// Helper to save to cache
	saveCache := func(t, v string) {
		cache[inputURL] = CacheEntry{Type: t, Val: v}
		_ = os.MkdirAll(filepath.Dir(cachePath), 0755)
		if data, err := json.Marshal(cache); err == nil {
			_ = os.WriteFile(cachePath, data, 0644)
		}
	}

	// 2. Check if it's already a raw ID
	if strings.HasPrefix(inputURL, "UC") && len(inputURL) == 24 {
		return "channel_id", inputURL
	}
	if strings.HasPrefix(inputURL, "PL") && len(inputURL) >= 18 {
		return "playlist_id", inputURL
	}

	// 3. Direct check for Playlist ID in query params of the URL
	if u, err := url.Parse(inputURL); err == nil {
		if playlistID := u.Query().Get("list"); playlistID != "" {
			saveCache("playlist_id", playlistID)
			return "playlist_id", playlistID
		}
	}

	// 4. Direct check for Channel ID in path
	if strings.Contains(inputURL, "/channel/") {
		parts := strings.Split(inputURL, "/channel/")
		if len(parts) > 1 {
			id := strings.Split(parts[1], "/")[0]
			id = strings.Split(id, "?")[0]
			if strings.HasPrefix(id, "UC") && len(id) == 24 {
				saveCache("channel_id", id)
				return "channel_id", id
			}
		}
	}

	// 5. Resolve handle/user or search query page
	if strings.Contains(inputURL, "youtube.com/@") || strings.Contains(inputURL, "youtube.com/user/") || strings.Contains(inputURL, "/results?search_query=") || strings.Contains(inputURL, "youtu.be/") {
		client := newPublicFetchClient(5 * time.Second)
		req, err := http.NewRequest("GET", inputURL, nil)
		if err != nil {
			return "", ""
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

		resp, err := client.Do(req)
		if err != nil {
			return "", ""
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			bodyBytes, err := readRemoteResponseBody(resp)
			if err == nil {
				bodyStr := string(bodyBytes)
				// Search meta tag first
				if match := itemPropRegex.FindStringSubmatch(bodyStr); len(match) > 1 {
					saveCache("channel_id", match[1])
					return "channel_id", match[1]
				}
				// Search JSON channelId
				if match := jsonChanIDRegex.FindStringSubmatch(bodyStr); len(match) > 1 {
					saveCache("channel_id", match[1])
					return "channel_id", match[1]
				}
				// Search generic channel URLs in body (especially useful for query search results)
				if match := channelIDRegex.FindStringSubmatch(bodyStr); len(match) > 1 {
					saveCache("channel_id", match[1])
					return "channel_id", match[1]
				}
			}
		}
	}

	return "", ""
}

func resolvePodcastURLToFeed(inputURL string, siteRoot string) string {
	inputURL = strings.TrimSpace(inputURL)
	if inputURL == "" {
		return ""
	}

	if !strings.Contains(inputURL, "podcasts.apple.com") {
		return inputURL // Direct RSS feed URL
	}

	// 1. Try cache
	cachePath := filepath.Join(siteRoot, ".gocache", "resolved_podcasts.json")
	var cache map[string]string
	if data, err := os.ReadFile(cachePath); err == nil {
		_ = json.Unmarshal(data, &cache)
	}
	if cache == nil {
		cache = make(map[string]string)
	}

	if resolved, ok := cache[inputURL]; ok {
		return resolved
	}

	// 2. Parse ID from URL e.g. /id284148583
	re := regexp.MustCompile(`/id(\d+)`)
	matches := re.FindStringSubmatch(inputURL)
	if len(matches) < 2 {
		return inputURL
	}
	podcastID := matches[1]

	// 3. Request iTunes Lookup API
	lookupURL := fmt.Sprintf("https://itunes.apple.com/lookup?id=%s", podcastID)
	client := newPublicFetchClient(5 * time.Second)
	resp, err := client.Get(lookupURL)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return inputURL
		}
		body, err := remoteResponseBodyReader(resp)
		if err != nil {
			return inputURL
		}
		type iTunesResult struct {
			Results []struct {
				FeedURL string `json:"feedUrl"`
			} `json:"results"`
		}
		var lookup iTunesResult
		if err := json.NewDecoder(body).Decode(&lookup); err == nil && len(lookup.Results) > 0 {
			feedURL := lookup.Results[0].FeedURL
			if feedURL != "" {
				// Save to cache
				cache[inputURL] = feedURL
				_ = os.MkdirAll(filepath.Dir(cachePath), 0755)
				if data, err := json.Marshal(cache); err == nil {
					_ = os.WriteFile(cachePath, data, 0644)
				}
				return feedURL
			}
		}
	}

	return inputURL
}

func fetchNewsFeedWithCache(feedURL string, siteRoot string, limit int) ([]feedItem, error) {
	if err := validatePublicFetchURL(feedURL); err != nil {
		return nil, err
	}

	resolvedFeedURL := resolveNewsURLToFeed(feedURL)
	if err := validatePublicFetchURL(resolvedFeedURL); err != nil {
		return nil, err
	}

	// Cache path based on hash of the resolved feed URL
	hasher := sha256.New()
	hasher.Write([]byte(resolvedFeedURL))
	cacheFilename := fmt.Sprintf("news_%x_l%d.json", hasher.Sum(nil), limit)
	cacheDir := filepath.Join(siteRoot, ".gocache", "feeds")
	cachePath := filepath.Join(cacheDir, cacheFilename)

	// Try to fetch live feed
	items, err := fetchLiveNewsFeed(resolvedFeedURL, limit)
	if err == nil {
		// Save to cache
		_ = os.MkdirAll(cacheDir, 0755)
		if data, err := json.Marshal(items); err == nil {
			_ = os.WriteFile(cachePath, data, 0644)
		}
		return items, nil
	}

	// Fallback to cache
	if data, err := os.ReadFile(cachePath); err == nil {
		var cachedItems []feedItem
		if err := json.Unmarshal(data, &cachedItems); err == nil {
			return cachedItems, nil
		}
	}

	return nil, err
}

func feedNewsHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		feedURLs := r.URL.Query()["url"]
		if len(feedURLs) == 0 {
			http.Error(w, "url parameter is required", http.StatusBadRequest)
			return
		}
		if len(feedURLs) > maxFeedURLParams {
			http.Error(w, "too many feed parameters", http.StatusBadRequest)
			return
		}

		limitStr := r.URL.Query().Get("limit")
		limit := defaultFeedLimit
		if limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
				limit = l
			}
		}
		limit = clampFeedLimit(limit)

		var allItems []feedItem
		for _, feedURL := range feedURLs {
			items, err := fetchNewsFeedWithCache(feedURL, siteRoot, limit)
			if err == nil {
				allItems = append(allItems, items...)
			}
		}

		// Sort by Created date descending (newest first)
		sort.Slice(allItems, func(i, j int) bool {
			return allItems[i].Created > allItems[j].Created
		})

		writeJSON(w, allItems)
	}
}

func fetchLiveNewsFeed(feedURL string, limit int) ([]feedItem, error) {
	client := newPublicFetchClient(8 * time.Second)
	req, err := http.NewRequest("GET", feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Rock-OS/1.0.0 (by rocketpowerinc)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("news RSS returned HTTP %d", resp.StatusCode)
	}

	body, err := remoteResponseBodyReader(resp)
	if err != nil {
		return nil, err
	}

	var rss rssFeed
	if err := xml.NewDecoder(body).Decode(&rss); err != nil {
		return nil, err
	}

	limit = clampFeedLimit(limit)
	if limit > len(rss.Channel.Items) {
		limit = len(rss.Channel.Items)
	}

	items := make([]feedItem, 0, limit)
	source := newsFeedSourceName(feedURL, rss.Channel.Title)
	for i := 0; i < limit; i++ {
		item := rss.Channel.Items[i]
		dateStr := ""
		var pubTime time.Time
		if item.PubDate != "" {
			pubTime = parseRssDate(item.PubDate)
			if !pubTime.IsZero() {
				dateStr = pubTime.Format("2006-01-02")
			} else {
				dateStr = item.PubDate
			}
		}

		items = append(items, feedItem{
			Title:     item.Title,
			URL:       item.Link,
			Created:   dateStr,
			Source:    source,
			Thumbnail: newsItemThumbnail(item),
		})
	}

	return items, nil
}

func newsFeedSourceName(feedURL string, channelTitle string) string {
	if parsed, err := url.Parse(feedURL); err == nil {
		host := strings.ToLower(parsed.Hostname())
		switch {
		case strings.Contains(host, "ign.com"):
			return "IGN"
		case strings.Contains(host, "news.google.com"):
			return "Google News"
		}
	}

	channelTitle = strings.TrimSpace(channelTitle)
	if channelTitle != "" {
		return channelTitle
	}

	return "News"
}

func newsItemThumbnail(item rssItem) string {
	candidates := []string{
		item.MediaThumbnail.URL,
		item.Image.URL,
	}

	if item.MediaContent.URL != "" && isImageMedia(item.MediaContent.Medium, item.MediaContent.Type, item.MediaContent.URL) {
		candidates = append(candidates, item.MediaContent.URL)
	}
	if item.Enclosure.URL != "" && isImageMedia("", item.Enclosure.Type, item.Enclosure.URL) {
		candidates = append(candidates, item.Enclosure.URL)
	}

	if descriptionImage := firstHTMLImageSrc(item.Description); descriptionImage != "" {
		candidates = append(candidates, descriptionImage)
	}

	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if strings.HasPrefix(candidate, "http://") || strings.HasPrefix(candidate, "https://") {
			return candidate
		}
	}

	return ""
}

func isImageMedia(medium string, mediaType string, mediaURL string) bool {
	medium = strings.ToLower(strings.TrimSpace(medium))
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	mediaURL = strings.ToLower(strings.TrimSpace(mediaURL))

	if medium == "image" || strings.HasPrefix(mediaType, "image/") {
		return true
	}

	return strings.HasSuffix(mediaURL, ".jpg") ||
		strings.HasSuffix(mediaURL, ".jpeg") ||
		strings.HasSuffix(mediaURL, ".png") ||
		strings.HasSuffix(mediaURL, ".webp") ||
		strings.HasSuffix(mediaURL, ".gif")
}

func firstHTMLImageSrc(html string) string {
	re := regexp.MustCompile(`(?is)<img[^>]+src=["']([^"']+)["']`)
	match := re.FindStringSubmatch(html)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func resolveNewsURLToFeed(inputURL string) string {
	u, err := url.Parse(inputURL)
	if err != nil {
		return inputURL
	}

	host := strings.ToLower(u.Hostname())

	// 1. Google News topics/sections translation
	if strings.Contains(host, "news.google.com") {
		path := u.Path
		if strings.HasPrefix(path, "/topics/") {
			u.Path = "/rss/topics/" + strings.TrimPrefix(path, "/topics/")
			return u.String()
		}
		if strings.HasPrefix(path, "/sections/") {
			u.Path = "/rss/sections/" + strings.TrimPrefix(path, "/sections/")
			return u.String()
		}
		if strings.HasPrefix(path, "/search") {
			u.Path = "/rss/search"
			return u.String()
		}
		if path == "" || path == "/" {
			u.Path = "/rss"
			return u.String()
		}
		return inputURL
	}

	if err := validatePublicFetchURL(inputURL); err != nil {
		return inputURL
	}

	// 3. General RSS Auto-Discovery
	client := newPublicFetchClient(5 * time.Second)
	resp, err := client.Get(inputURL)
	if err == nil {
		defer resp.Body.Close()
		body, err := readRemoteResponseBody(resp)
		if err == nil {
			re := regexp.MustCompile(`(?i)<link[^>]+type=["']application/rss\+xml["'][^>]+href=["']([^"']+)["']`)
			match := re.FindStringSubmatch(string(body))
			if len(match) > 1 {
				href := match[1]
				// Resolve relative URL
				if strings.HasPrefix(href, "/") {
					u.Path = href
					u.RawQuery = ""
					u.Fragment = ""
					return u.String()
				}
				if !strings.HasPrefix(href, "http") {
					// Prepend base schema/host
					return fmt.Sprintf("%s://%s/%s", u.Scheme, u.Hostname(), strings.TrimPrefix(href, "/"))
				}
				return href
			}
		}
	}

	return inputURL
}

func feedSpotifyHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		urls := r.URL.Query()["url"]
		if len(urls) == 0 {
			http.Error(w, "url parameter is required", http.StatusBadRequest)
			return
		}
		if len(urls) > maxFeedURLParams {
			http.Error(w, "too many feed parameters", http.StatusBadRequest)
			return
		}

		limit := defaultFeedLimit
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
				limit = l
			}
		}
		limit = clampFeedLimit(limit)

		// Cache path based on hash of the URLs combined
		hasher := sha256.New()
		for _, u := range urls {
			hasher.Write([]byte(u))
		}
		cacheFilename := fmt.Sprintf("spotify_%x.json", hasher.Sum(nil))
		cacheDir := filepath.Join(siteRoot, ".gocache", "feeds")
		cachePath := filepath.Join(cacheDir, cacheFilename)

		// Try to fetch live
		items := []feedItem{}
		var fetchErr error

		for i, spotifyURL := range urls {
			if i >= limit {
				break
			}
			item, err := fetchSpotifyOEmbed(spotifyURL)
			if err != nil {
				fetchErr = err
				break
			}
			items = append(items, item)
		}

		if fetchErr == nil && len(items) > 0 {
			// Save to cache
			_ = os.MkdirAll(cacheDir, 0755)
			if data, err := json.Marshal(items); err == nil {
				_ = os.WriteFile(cachePath, data, 0644)
			}
			writeJSON(w, items)
			return
		}

		// Fallback to cache
		if data, err := os.ReadFile(cachePath); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(data)
			return
		}

		// Return empty list on failure
		writeJSON(w, []feedItem{})
	}
}

func fetchSpotifyOEmbed(spotifyURL string) (feedItem, error) {
	apiURL := fmt.Sprintf("https://embed.spotify.com/oembed/?url=%s", url.QueryEscape(spotifyURL))

	client := newPublicFetchClient(5 * time.Second)
	resp, err := client.Get(apiURL)
	if err != nil {
		return feedItem{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return feedItem{}, fmt.Errorf("spotify oembed returned status %d", resp.StatusCode)
	}

	body, err := remoteResponseBodyReader(resp)
	if err != nil {
		return feedItem{}, err
	}

	var data struct {
		Title        string `json:"title"`
		ThumbnailURL string `json:"thumbnail_url"`
	}

	if err := json.NewDecoder(body).Decode(&data); err != nil {
		return feedItem{}, err
	}

	return feedItem{
		Title:     data.Title,
		URL:       spotifyURL,
		Created:   "Spotify",
		Thumbnail: data.ThumbnailURL,
	}, nil
}

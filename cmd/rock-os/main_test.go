package main

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestScriptRunRequestAllowedRequiresRockOSHeader(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8000/api/scripts/run", nil)
	request.RemoteAddr = "127.0.0.1:49200"

	if scriptRunRequestAllowed(request, false) {
		t.Fatal("script run request without Rock-OS header was allowed")
	}
}

func TestScriptRunRequestAllowedRejectsCrossOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8000/api/scripts/run", nil)
	request.RemoteAddr = "127.0.0.1:49200"
	request.Header.Set("X-Rock-OS-Requested", "true")
	request.Header.Set("Origin", "https://example.com")

	if scriptRunRequestAllowed(request, false) {
		t.Fatal("cross-origin script run request was allowed")
	}
}

func TestScriptRunRequestAllowedAcceptsLoopbackSameOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8000/api/scripts/run", nil)
	request.RemoteAddr = "127.0.0.1:49200"
	request.Header.Set("X-Rock-OS-Requested", "true")
	request.Header.Set("Origin", "http://127.0.0.1:8000")
	request.Header.Set("Referer", "http://127.0.0.1:8000/scripts.html")

	if !scriptRunRequestAllowed(request, false) {
		t.Fatal("loopback same-origin script run request was rejected")
	}
}

func TestScriptRunRequestAllowedRejectsLANByDefault(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://192.168.1.2:8000/api/scripts/run", nil)
	request.RemoteAddr = "192.168.1.50:49200"
	request.Header.Set("X-Rock-OS-Requested", "true")
	request.Header.Set("Origin", "http://192.168.1.2:8000")
	request.Header.Set("Referer", "http://192.168.1.2:8000/scripts.html")

	if scriptRunRequestAllowed(request, false) {
		t.Fatal("LAN script run request was allowed by default")
	}
}

func TestScriptRunRequestAllowedAcceptsLANWithExplicitOptIn(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://192.168.1.2:8000/api/scripts/run", nil)
	request.RemoteAddr = "192.168.1.50:49200"
	request.Header.Set("X-Rock-OS-Requested", "true")
	request.Header.Set("Origin", "http://192.168.1.2:8000")
	request.Header.Set("Referer", "http://192.168.1.2:8000/scripts.html")

	if !scriptRunRequestAllowed(request, true) {
		t.Fatal("LAN script run request was rejected after explicit opt-in")
	}
}

func TestServerRefreshRequestAllowedAcceptsLoopbackSameOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8000/api/server/refresh", nil)
	request.RemoteAddr = "127.0.0.1:49200"
	request.Header.Set("X-Rock-OS-Requested", "true")
	request.Header.Set("Origin", "http://127.0.0.1:8000")
	request.Header.Set("Referer", "http://127.0.0.1:8000/scripts.html")

	if !serverRefreshRequestAllowed(request) {
		t.Fatal("loopback same-origin refresh request was rejected")
	}
}

func TestServerRefreshRequestAllowedRejectsLAN(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://192.168.1.2:8000/api/server/refresh", nil)
	request.RemoteAddr = "192.168.1.50:49200"
	request.Header.Set("X-Rock-OS-Requested", "true")
	request.Header.Set("Origin", "http://192.168.1.2:8000")
	request.Header.Set("Referer", "http://192.168.1.2:8000/scripts.html")

	if serverRefreshRequestAllowed(request) {
		t.Fatal("LAN refresh request was allowed")
	}
}

func TestServerRefreshHandlerRejectsNonClone(t *testing.T) {
	siteRoot := t.TempDir()
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8000/api/server/refresh", nil)
	request.RemoteAddr = "127.0.0.1:49200"
	request.Header.Set("X-Rock-OS-Requested", "true")
	recorder := httptest.NewRecorder()

	serverRefreshHandler(siteRoot).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", recorder.Code)
	}
}

func TestResolveScriptRejectsUnsupportedCharacters(t *testing.T) {
	_, _, err := resolveScript(t.TempDir(), "Linux/update;rm.sh")
	if err == nil {
		t.Fatal("script id with shell metacharacter was allowed")
	}
}

func TestAPIRateLimiterRejectsBurstFlood(t *testing.T) {
	limiter := newAPIRateLimiter()
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8000/api/wiki/search?q=x", nil)
	request.RemoteAddr = "127.0.0.1:49200"

	for i := 0; i < apiRateLimitBurst; i++ {
		if !limiter.allow(request) {
			t.Fatalf("request %d was rejected before burst was exhausted", i)
		}
	}

	if limiter.allow(request) {
		t.Fatal("request beyond burst limit was allowed")
	}
}

func TestMarkdownSearchIndexRefreshesChangedFiles(t *testing.T) {
	siteRoot := t.TempDir()
	wikiRoot := filepath.Join(siteRoot, markdownDir)
	if err := os.MkdirAll(wikiRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	docPath := filepath.Join(wikiRoot, "Search.md")
	if err := os.WriteFile(docPath, []byte("# Search\n\nfirst needle\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	files := []markdownIndexEntry{{Path: markdownDir + "/Search.md"}}
	index := newMarkdownSearchIndex()

	results, err := searchMarkdownIndex(siteRoot, "needle", files, index)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected initial search match, got %d", len(results))
	}

	time.Sleep(2 * time.Millisecond)
	if err := os.WriteFile(docPath, []byte("# Search\n\nsecond haystack\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	results, err = searchMarkdownIndex(siteRoot, "needle", files, index)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected changed file to leave no matches, got %d", len(results))
	}
}

func TestRequestFlightGroupDeduplicatesConcurrentCalls(t *testing.T) {
	group := newRequestFlightGroup()
	start := make(chan struct{})
	var calls int
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			value, err := group.Do("feed", func() (any, error) {
				mu.Lock()
				calls++
				mu.Unlock()
				time.Sleep(20 * time.Millisecond)
				return "ok", nil
			})
			if err != nil {
				t.Error(err)
			}
			if value != "ok" {
				t.Errorf("unexpected value: %v", value)
			}
		}()
	}

	close(start)
	wg.Wait()

	if calls != 1 {
		t.Fatalf("expected one underlying call, got %d", calls)
	}
}

func TestSearchSnippetStripsMarkdownAndHTML(t *testing.T) {
	text := `# Links
This **important** [Linux guide](../Linux/Guide.md) has <strong>needle</strong> and ` + "`inline code`" + `.`

	snippet := searchSnippet(text, "needle")
	if strings.Contains(snippet, "<strong>") ||
		strings.Contains(snippet, "[Linux guide]") ||
		strings.Contains(snippet, "**") ||
		strings.Contains(snippet, "`") {
		t.Fatalf("snippet still contains markup: %q", snippet)
	}

	if !strings.Contains(snippet, "Linux guide") ||
		!strings.Contains(snippet, "needle") ||
		!strings.Contains(snippet, "inline code") {
		t.Fatalf("snippet lost readable text: %q", snippet)
	}
}

func TestValidatePublicFetchURLBlocksLocalAndPrivateTargets(t *testing.T) {
	tests := []string{
		"http://localhost/feed.xml",
		"http://127.0.0.1/feed.xml",
		"http://0.0.0.0/feed.xml",
		"http://10.0.0.5/feed.xml",
		"http://172.16.0.5/feed.xml",
		"http://172.31.255.255/feed.xml",
		"http://192.168.1.2/feed.xml",
		"http://169.254.169.254/latest/meta-data",
		"http://[::1]/feed.xml",
		"http://[fe80::1]/feed.xml",
		"http://[fc00::1]/feed.xml",
		"file:///etc/passwd",
	}

	for _, rawURL := range tests {
		t.Run(rawURL, func(t *testing.T) {
			if err := validatePublicFetchURL(rawURL); err == nil {
				t.Fatalf("expected %q to be blocked", rawURL)
			}
		})
	}
}

func TestValidatePublicFetchURLAllowsPublicIPTarget(t *testing.T) {
	if err := validatePublicFetchURL("https://8.8.8.8/feed.xml"); err != nil {
		t.Fatalf("expected public URL to be allowed, got: %v", err)
	}
}

func TestClampFeedLimit(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{name: "default for zero", in: 0, want: defaultFeedLimit},
		{name: "default for negative", in: -10, want: defaultFeedLimit},
		{name: "keeps valid value", in: 12, want: 12},
		{name: "caps large value", in: 500, want: maxFeedLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clampFeedLimit(tt.in); got != tt.want {
				t.Fatalf("expected %d, got %d", tt.want, got)
			}
		})
	}
}

func TestReadRemoteResponseBodyRejectsOversizedContentLength(t *testing.T) {
	resp := &http.Response{
		ContentLength: maxRemoteFeedResponseSize + 1,
		Body:          io.NopCloser(strings.NewReader("small body")),
	}

	if _, err := readRemoteResponseBody(resp); err == nil {
		t.Fatal("expected oversized response to be rejected")
	}
}

func TestScanLinkHealthReportsLocalAndExternalLinks(t *testing.T) {
	siteRoot := t.TempDir()
	wikiRoot := filepath.Join(siteRoot, markdownDir, "Linux")
	if err := os.MkdirAll(wikiRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(siteRoot, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(wikiRoot, "Target.md"), []byte("# Target\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(siteRoot, "assets", "icon.png"), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := strings.Join([]string{
		"[Target](Target.md)",
		"![Icon](/assets/icon.png)",
		"[Missing](Missing.md)",
		"[External](https://example.com)",
	}, "\n")
	if err := os.WriteFile(filepath.Join(wikiRoot, "Source.md"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := scanLinkHealth(siteRoot)
	if err != nil {
		t.Fatal(err)
	}

	if report.Checked != 4 {
		t.Fatalf("expected 4 checked links, got %d", report.Checked)
	}
	if report.OK != 2 {
		t.Fatalf("expected 2 ok links, got %d", report.OK)
	}
	if report.Broken != 1 {
		t.Fatalf("expected 1 broken link, got %d", report.Broken)
	}
	if report.External != 1 {
		t.Fatalf("expected 1 external link, got %d", report.External)
	}
}

func TestScanLinkHealthIgnoresMarkedLinks(t *testing.T) {
	siteRoot := t.TempDir()
	wikiRoot := filepath.Join(siteRoot, markdownDir)
	if err := os.MkdirAll(wikiRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	source := strings.Join([]string{
		"[Ignored](Future.md) <!-- rock-os-ignore-link -->",
		"[Missing](Missing.md)",
	}, "\n")
	if err := os.WriteFile(filepath.Join(wikiRoot, "Source.md"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := scanLinkHealth(siteRoot)
	if err != nil {
		t.Fatal(err)
	}

	if report.Checked != 1 {
		t.Fatalf("expected only unignored link to be checked, got %d", report.Checked)
	}
	if report.Broken != 1 {
		t.Fatalf("expected one broken unignored link, got %d", report.Broken)
	}
	for _, item := range report.Items {
		if item.Href == "Future.md" {
			t.Fatalf("ignored link was reported: %+v", item)
		}
	}
}

func TestNewsItemThumbnailFindsCommonRSSImageFields(t *testing.T) {
	tests := []struct {
		name string
		item rssItem
		want string
	}{
		{
			name: "media thumbnail",
			item: rssItem{MediaThumbnail: mediaImage{URL: "https://example.com/thumb.jpg"}},
			want: "https://example.com/thumb.jpg",
		},
		{
			name: "media content image",
			item: rssItem{MediaContent: mediaImage{URL: "https://example.com/content.webp", Medium: "image"}},
			want: "https://example.com/content.webp",
		},
		{
			name: "enclosure image",
			item: rssItem{Enclosure: rssEnclosure{URL: "https://example.com/enclosure.png", Type: "image/png"}},
			want: "https://example.com/enclosure.png",
		},
		{
			name: "description image",
			item: rssItem{Description: `<p><img src="https://example.com/desc.jpg"></p>`},
			want: "https://example.com/desc.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newsItemThumbnail(tt.item); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestResolveSiteRootFindsWebsiteFromRepoRoot(t *testing.T) {
	repoRoot := t.TempDir()
	websiteRoot := filepath.Join(repoRoot, "Website")
	createTestWebsiteRoot(t, websiteRoot)

	siteRoot, err := resolveSiteRoot(repoRoot, "")
	if err != nil {
		t.Fatal(err)
	}

	if siteRoot != websiteRoot {
		t.Fatalf("expected %q, got %q", websiteRoot, siteRoot)
	}
}

func TestResolveSiteRootFindsWebsiteFromCommandFolder(t *testing.T) {
	repoRoot := t.TempDir()
	websiteRoot := filepath.Join(repoRoot, "Website")
	commandRoot := filepath.Join(repoRoot, "cmd", "rock-os")
	createTestWebsiteRoot(t, websiteRoot)
	if err := os.MkdirAll(commandRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	siteRoot, err := resolveSiteRoot(commandRoot, "")
	if err != nil {
		t.Fatal(err)
	}

	if siteRoot != websiteRoot {
		t.Fatalf("expected %q, got %q", websiteRoot, siteRoot)
	}
}

func TestWikiDocHandlerRendersMarkdown(t *testing.T) {
	siteRoot := t.TempDir()
	markdownRoot := filepath.Join(siteRoot, markdownDir)
	if err := os.MkdirAll(markdownRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	docPath := filepath.Join(markdownRoot, "Test.md")
	if err := os.WriteFile(docPath, []byte("# Hello\n\n- one\n- two\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/wiki/doc?path=menu/wiki/Test.md", nil)
	recorder := httptest.NewRecorder()

	wikiDocHandler(siteRoot).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response wikiDocResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}

	if response.Path != "menu/wiki/Test.md" {
		t.Fatalf("unexpected response path: %q", response.Path)
	}

	if !strings.Contains(response.HTML, "<h1 id=\"hello\">Hello</h1>") {
		t.Fatalf("expected rendered heading, got: %s", response.HTML)
	}

	if !strings.Contains(response.HTML, "<li>one</li>") {
		t.Fatalf("expected rendered list item, got: %s", response.HTML)
	}
}

func TestWikiDocHandlerEscapesRawHTML(t *testing.T) {
	siteRoot := t.TempDir()
	markdownRoot := filepath.Join(siteRoot, markdownDir)
	if err := os.MkdirAll(markdownRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	docPath := filepath.Join(markdownRoot, "Unsafe.md")
	if err := os.WriteFile(docPath, []byte("<script>alert(1)</script>\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/wiki/doc?path=menu/wiki/Unsafe.md", nil)
	recorder := httptest.NewRecorder()

	wikiDocHandler(siteRoot).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response wikiDocResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}

	if strings.Contains(response.HTML, "<script>") {
		t.Fatalf("raw script tag was not escaped: %s", response.HTML)
	}

	if !strings.Contains(response.HTML, "raw HTML omitted") {
		t.Fatalf("raw HTML was not omitted by the safe renderer: %s", response.HTML)
	}
}

func TestWikiDocHandlerRejectsTraversal(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/wiki/doc?path=menu/wiki/../secret.md", nil)
	recorder := httptest.NewRecorder()

	wikiDocHandler(t.TempDir()).ServeHTTP(recorder, request)

	if recorder.Code == http.StatusOK {
		t.Fatal("path traversal request was allowed")
	}
}

func TestWikiSearchHandlerFindsFilenameAndContentMatches(t *testing.T) {
	siteRoot := t.TempDir()
	markdownRoot := filepath.Join(siteRoot, markdownDir)
	if err := os.MkdirAll(filepath.Join(markdownRoot, "Linux"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(markdownRoot, "Linux", "Booting.md"),
		[]byte("# Booting\n\nGRUB and rEFInd both matter here.\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(markdownRoot, "Linux", "Networking.md"),
		[]byte("# Networking\n\nOffline LAN notes.\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/wiki/search?q=refind", nil)
	recorder := httptest.NewRecorder()

	wikiSearchHandler(siteRoot).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response wikiSearchResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}

	if len(response.Results) != 1 {
		t.Fatalf("expected one search result, got %#v", response.Results)
	}

	if response.Results[0].Path != "menu/wiki/Linux/Booting.md" {
		t.Fatalf("unexpected result path: %#v", response.Results[0])
	}

	if !strings.Contains(response.Results[0].Snippet, "rEFInd") {
		t.Fatalf("expected snippet to include match, got %#v", response.Results[0])
	}
}

func TestWikiSearchHandlerReturnsEmptyResultsForBlankQuery(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/wiki/search?q=+", nil)
	recorder := httptest.NewRecorder()

	wikiSearchHandler(t.TempDir()).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response wikiSearchResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}

	if len(response.Results) != 0 {
		t.Fatalf("expected no search results, got %#v", response.Results)
	}
}

func TestScriptsSearchHandlerFindsFilenameAndContentMatches(t *testing.T) {
	siteRoot := t.TempDir()
	scriptsRoot := filepath.Join(siteRoot, scriptsDir, "Linux")
	if err := os.MkdirAll(scriptsRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(scriptsRoot, "firefox-setup.sh"),
		[]byte("#!/usr/bin/env sh\n# Install uBlock Origin policy\n"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/scripts/search?q=ublock", nil)
	recorder := httptest.NewRecorder()

	scriptsSearchHandler(siteRoot).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response scriptSearchResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}

	if len(response.Results) != 1 {
		t.Fatalf("expected one search result, got %#v", response.Results)
	}

	if response.Results[0].ID != "Linux/firefox-setup.sh" {
		t.Fatalf("unexpected result: %#v", response.Results[0])
	}

	if !strings.Contains(response.Results[0].Snippet, "uBlock") {
		t.Fatalf("expected snippet to include match, got %#v", response.Results[0])
	}
}

func TestCompressResponsesUsesGzipForTextResponses(t *testing.T) {
	handler := compressResponses(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/server/status", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("expected gzip response, got headers %#v", recorder.Header())
	}

	reader, err := gzip.NewReader(recorder.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != `{"ok":true}` {
		t.Fatalf("unexpected decompressed body: %s", body)
	}
}

func TestMarkdownIndexHandlerRefreshesIndexOnDemand(t *testing.T) {
	siteRoot := t.TempDir()
	markdownRoot := filepath.Join(siteRoot, markdownDir)
	if err := os.MkdirAll(markdownRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(markdownRoot, "Fresh.md"),
		[]byte("# Fresh\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/wiki-index.json", nil)
	recorder := httptest.NewRecorder()

	markdownIndexHandler(siteRoot).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var files []markdownIndexEntry
	if err := json.Unmarshal(recorder.Body.Bytes(), &files); err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 {
		t.Fatalf("expected one indexed file, got %#v", files)
	}

	if files[0].Path != "menu/wiki/Fresh.md" {
		t.Fatalf("unexpected index entry: %#v", files[0])
	}
}

func TestCollectMarkdownFilesCachesMetadata(t *testing.T) {
	siteRoot := t.TempDir()
	markdownRoot := filepath.Join(siteRoot, markdownDir)
	if err := os.MkdirAll(markdownRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	docPath := filepath.Join(markdownRoot, "Pinned.md")
	if err := os.WriteFile(docPath, []byte("# Pinned\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cache := &markdownIndexCache{entries: map[string]markdownIndexCacheEntry{}}
	files, err := collectMarkdownFilesWithCache(siteRoot, markdownDir, cache)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 {
		t.Fatalf("expected one file in index, got %#v", files)
	}

	updatedTime := time.Now().Add(2 * time.Second)
	if err := os.WriteFile(docPath, []byte("# Not pinned\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(docPath, updatedTime, updatedTime); err != nil {
		t.Fatal(err)
	}

	files, err = collectMarkdownFilesWithCache(siteRoot, markdownDir, cache)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 {
		t.Fatalf("expected changed file to refresh cache, got %#v", files)
	}
}

func TestCollectMarkdownFilesPrunesDeletedCacheEntries(t *testing.T) {
	siteRoot := t.TempDir()
	markdownRoot := filepath.Join(siteRoot, markdownDir)
	if err := os.MkdirAll(markdownRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	docPath := filepath.Join(markdownRoot, "DeleteMe.md")
	if err := os.WriteFile(docPath, []byte("# Delete me\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cache := &markdownIndexCache{entries: map[string]markdownIndexCacheEntry{}}
	if _, err := collectMarkdownFilesWithCache(siteRoot, markdownDir, cache); err != nil {
		t.Fatal(err)
	}

	if len(cache.entries) != 1 {
		t.Fatalf("expected one cache entry, got %d", len(cache.entries))
	}

	if err := os.Remove(docPath); err != nil {
		t.Fatal(err)
	}

	files, err := collectMarkdownFilesWithCache(siteRoot, markdownDir, cache)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 0 {
		t.Fatalf("expected deleted file to leave index, got %#v", files)
	}

	if len(cache.entries) != 0 {
		t.Fatalf("expected deleted file cache entry to be pruned, got %d", len(cache.entries))
	}
}

func createTestWebsiteRoot(t *testing.T, siteRoot string) {
	t.Helper()

	for _, dir := range []string{markdownDir, guidesDir, cheatsheetsDir, dotfilesDir, bookmarksDir, scriptsDir, profilesDir, dashboardsDir, "css", "js"} {
		if err := os.MkdirAll(filepath.Join(siteRoot, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	for _, file := range []string{"index.html", "wiki.html", "guides.html", "cheatsheets.html", "dotfiles.html", "bookmarks.html", "scripts.html", "profiles.html", "dashboards.html"} {
		if err := os.WriteFile(filepath.Join(siteRoot, file), []byte(file), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestServerStatusHandlerReturnsGitCryptStatus(t *testing.T) {
	siteRoot := t.TempDir()
	createTestWebsiteRoot(t, siteRoot)
	if err := os.RemoveAll(filepath.Join(siteRoot, profilesDir)); err != nil {
		t.Fatal(err)
	}

	// Write a wiki markdown file to ensure WikiCount is 1
	wikiDoc := filepath.Join(siteRoot, markdownDir, "doc.md")
	if err := os.WriteFile(wikiDoc, []byte("# Hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Case 1: missing (Private folder doesn't exist)
	req := httptest.NewRequest(http.MethodGet, "/api/server/status", nil)
	rec := httptest.NewRecorder()
	serverStatusHandler("127.0.0.1", []string{"localhost"}, 8000, siteRoot).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var status serverStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}

	if status.GitCrypt != "missing" {
		t.Errorf("expected gitCrypt to be 'missing', got %q", status.GitCrypt)
	}
	if status.WikiCount != 1 {
		t.Errorf("expected wikiCount to be 1, got %d", status.WikiCount)
	}
	if status.ScriptsCount != 0 {
		t.Errorf("expected scriptsCount to be 0, got %d", status.ScriptsCount)
	}
	if status.Uptime < 0 {
		t.Errorf("expected uptime to be non-negative, got %d", status.Uptime)
	}
	if status.LastSync < 0 {
		t.Errorf("expected lastSync to be non-negative, got %d", status.LastSync)
	}

	// Case 2: unlocked (Profiles Folder exists with non-encrypted file)
	privateDir := filepath.Join(siteRoot, profilesDir)
	if err := os.MkdirAll(privateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(privateDir, "doc.md"), []byte("plain text doc"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a test script file
	scriptsLinuxDir := filepath.Join(siteRoot, scriptsDir, "Linux")
	if err := os.MkdirAll(scriptsLinuxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scriptsLinuxDir, "test.sh"), []byte("#!/bin/sh"), 0o755); err != nil {
		t.Fatal(err)
	}

	rec2 := httptest.NewRecorder()
	serverStatusHandler("127.0.0.1", []string{"localhost"}, 8000, siteRoot).ServeHTTP(rec2, req)

	var status2 serverStatus
	if err := json.Unmarshal(rec2.Body.Bytes(), &status2); err != nil {
		t.Fatal(err)
	}

	if status2.GitCrypt != "unlocked" {
		t.Errorf("expected gitCrypt to be 'unlocked', got %q", status2.GitCrypt)
	}
	if status2.WikiCount != 1 {
		t.Errorf("expected wikiCount to be 1, got %d", status2.WikiCount)
	}
	if status2.ScriptsCount != 1 {
		t.Errorf("expected scriptsCount to be 1, got %d", status2.ScriptsCount)
	}

	// Case 3: locked (Profiles Folder exists with locked git-crypt file)
	if err := os.WriteFile(filepath.Join(privateDir, "locked-doc.md"), []byte("GITCRYPT\nencrypted data here"), 0o644); err != nil {
		t.Fatal(err)
	}

	rec3 := httptest.NewRecorder()
	serverStatusHandler("127.0.0.1", []string{"localhost"}, 8000, siteRoot).ServeHTTP(rec3, req)

	var status3 serverStatus
	if err := json.Unmarshal(rec3.Body.Bytes(), &status3); err != nil {
		t.Fatal(err)
	}

	if status3.GitCrypt != "locked" {
		t.Errorf("expected gitCrypt to be 'locked', got %q", status3.GitCrypt)
	}
	if status3.WikiCount != 1 {
		t.Errorf("expected wikiCount to be 1, got %d", status3.WikiCount)
	}
	if status3.ScriptsCount != 1 {
		t.Errorf("expected scriptsCount to be 1, got %d", status3.ScriptsCount)
	}
}

func TestResolveGuideDoc(t *testing.T) {
	siteRoot := t.TempDir()
	guidesRoot := filepath.Join(siteRoot, guidesDir)
	if err := os.MkdirAll(guidesRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	docPath := filepath.Join(guidesRoot, "Setup.md")
	if err := os.WriteFile(docPath, []byte("# Setup"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Normal resolve
	resolvedPath, fullPath, err := resolveGuideDoc(siteRoot, "menu/guides/Setup.md")
	if err != nil {
		t.Fatal(err)
	}
	if resolvedPath != "menu/guides/Setup.md" {
		t.Errorf("expected menu/guides/Setup.md, got %q", resolvedPath)
	}
	if !strings.HasSuffix(fullPath, "Setup.md") {
		t.Errorf("expected path to end with Setup.md, got %q", fullPath)
	}

	// Path traversal check
	_, _, err = resolveGuideDoc(siteRoot, "menu/guides/../secret.md")
	if err == nil {
		t.Error("expected error for path traversal attempt")
	}

	// Non-markdown file check
	_, _, err = resolveGuideDoc(siteRoot, "menu/guides/Setup.txt")
	if err == nil {
		t.Error("expected error for non-markdown extension")
	}
}

func TestGuidesIndexHandler(t *testing.T) {
	siteRoot := t.TempDir()
	createTestWebsiteRoot(t, siteRoot)
	guidesRoot := filepath.Join(siteRoot, guidesDir)
	if err := os.MkdirAll(guidesRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	docPath := filepath.Join(guidesRoot, "Install.md")
	if err := os.WriteFile(docPath, []byte("# Install"), 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/guides-index.json", nil)
	rec := httptest.NewRecorder()
	guidesIndexHandler(siteRoot).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var index []markdownIndexEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &index); err != nil {
		t.Fatal(err)
	}

	if len(index) != 1 {
		t.Fatalf("expected 1 index entry, got %d", len(index))
	}
	if index[0].Path != "menu/guides/Install.md" {
		t.Errorf("expected menu/guides/Install.md, got %q", index[0].Path)
	}
}

func TestCheatsheetsIndexHandler(t *testing.T) {
	siteRoot := t.TempDir()
	createTestWebsiteRoot(t, siteRoot)
	cheatsheetsRoot := filepath.Join(siteRoot, cheatsheetsDir)
	if err := os.MkdirAll(cheatsheetsRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	docPath := filepath.Join(cheatsheetsRoot, "Commands.md")
	if err := os.WriteFile(docPath, []byte("# Commands"), 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/cheatsheets-index.json", nil)
	rec := httptest.NewRecorder()
	cheatsheetsIndexHandler(siteRoot).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var index []markdownIndexEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &index); err != nil {
		t.Fatal(err)
	}

	if len(index) != 1 {
		t.Fatalf("expected 1 index entry, got %d", len(index))
	}
	if index[0].Path != "menu/cheatsheets/Commands.md" {
		t.Errorf("expected menu/cheatsheets/Commands.md, got %q", index[0].Path)
	}
}

func TestDotfilesIndexHandler(t *testing.T) {
	siteRoot := t.TempDir()
	createTestWebsiteRoot(t, siteRoot)
	dotfilesRoot := filepath.Join(siteRoot, dotfilesDir)
	if err := os.MkdirAll(dotfilesRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	docPath := filepath.Join(dotfilesRoot, "Shell.md")
	if err := os.WriteFile(docPath, []byte("# Shell"), 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/dotfiles-index.json", nil)
	rec := httptest.NewRecorder()
	dotfilesIndexHandler(siteRoot).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var index []markdownIndexEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &index); err != nil {
		t.Fatal(err)
	}

	if len(index) != 1 {
		t.Fatalf("expected 1 index entry, got %d", len(index))
	}
	if index[0].Path != "menu/dotfiles/Shell.md" {
		t.Errorf("expected menu/dotfiles/Shell.md, got %q", index[0].Path)
	}
}

func TestBookmarksIndexHandler(t *testing.T) {
	siteRoot := t.TempDir()
	createTestWebsiteRoot(t, siteRoot)
	bookmarksRoot := filepath.Join(siteRoot, bookmarksDir)
	if err := os.MkdirAll(bookmarksRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	docPath := filepath.Join(bookmarksRoot, "Links.md")
	if err := os.WriteFile(docPath, []byte("# Links"), 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/bookmarks-index.json", nil)
	rec := httptest.NewRecorder()
	bookmarksIndexHandler(siteRoot).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var index []markdownIndexEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &index); err != nil {
		t.Fatal(err)
	}

	if len(index) != 1 {
		t.Fatalf("expected 1 index entry, got %d", len(index))
	}
	if index[0].Path != "menu/bookmarks/Links.md" {
		t.Errorf("expected menu/bookmarks/Links.md, got %q", index[0].Path)
	}
}

func TestProfilesDocHandlerRendersMarkdown(t *testing.T) {
	siteRoot := t.TempDir()
	profilesRoot := filepath.Join(siteRoot, profilesDir)
	if err := os.MkdirAll(profilesRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	docPath := filepath.Join(profilesRoot, "Test.md")
	if err := os.WriteFile(docPath, []byte("# Secret\n\n- lock\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/profiles/doc?path=profiles/Test.md", nil)
	recorder := httptest.NewRecorder()

	profilesDocHandler(siteRoot).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response wikiDocResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}

	if response.Path != "profiles/Test.md" {
		t.Fatalf("unexpected response path: %q", response.Path)
	}

	if !strings.Contains(response.HTML, "<h1 id=\"secret\">Secret</h1>") {
		t.Fatalf("expected rendered heading, got: %s", response.HTML)
	}
}

func TestResolveProfilesDoc(t *testing.T) {
	siteRoot := t.TempDir()
	profilesRoot := filepath.Join(siteRoot, profilesDir)
	if err := os.MkdirAll(profilesRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	// Normal doc check
	docPath := filepath.Join(profilesRoot, "Target.md")
	if err := os.WriteFile(docPath, []byte("# Target"), 0o644); err != nil {
		t.Fatal(err)
	}

	normalized, target, err := resolveProfilesDoc(siteRoot, "profiles/Target.md")
	if err != nil {
		t.Fatal(err)
	}
	if normalized != "profiles/Target.md" {
		t.Errorf("unexpected normalized path: %q", normalized)
	}
	if target != docPath {
		t.Errorf("unexpected target path: %q", target)
	}

	// Traversals check
	_, _, err = resolveProfilesDoc(siteRoot, "profiles/../outside.md")
	if err == nil {
		t.Error("expected traversal error")
	}

	// Prefix check
	_, _, err = resolveProfilesDoc(siteRoot, "wiki/Target.md")
	if err == nil {
		t.Error("expected prefix error")
	}
}

func TestProfilesIndexHandlerRefreshesIndex(t *testing.T) {
	siteRoot := t.TempDir()
	profilesRoot := filepath.Join(siteRoot, profilesDir)
	if err := os.MkdirAll(profilesRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	docPath := filepath.Join(profilesRoot, "PrivateFile.md")
	if err := os.WriteFile(docPath, []byte("# PrivateFile"), 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/profiles-index.json", nil)
	rec := httptest.NewRecorder()
	profilesIndexHandler(siteRoot).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var index []markdownIndexEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &index); err != nil {
		t.Fatal(err)
	}

	if len(index) != 1 {
		t.Fatalf("expected 1 index entry, got %d", len(index))
	}
	if index[0].Path != "profiles/PrivateFile.md" {
		t.Errorf("expected profiles/PrivateFile.md, got %q", index[0].Path)
	}
}

func TestProfilesIndexHandlerFiltersProfile(t *testing.T) {
	siteRoot := t.TempDir()
	for _, profile := range []string{"Rocket", "Kids"} {
		profileRoot := filepath.Join(siteRoot, profilesDir, profile)
		if err := os.MkdirAll(profileRoot, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(profileRoot, "Profile.md"), []byte("# "+profile), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/profiles-index.json?profile=Kids", nil)
	rec := httptest.NewRecorder()
	profilesIndexHandler(siteRoot).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var index []markdownIndexEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &index); err != nil {
		t.Fatal(err)
	}

	if len(index) != 1 {
		t.Fatalf("expected 1 profile file, got %d", len(index))
	}
	if index[0].Path != "profiles/Kids/Profile.md" {
		t.Errorf("expected Kids profile file, got %q", index[0].Path)
	}
}

func TestProfilesHandlersRejectLockedContent(t *testing.T) {
	siteRoot := t.TempDir()
	profilesRoot := filepath.Join(siteRoot, profilesDir)
	if err := os.MkdirAll(profilesRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	docPath := filepath.Join(profilesRoot, "Locked.md")
	if err := os.WriteFile(docPath, []byte("\x00GITCRYPT\x00locked"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		request *http.Request
		handler http.HandlerFunc
	}{
		{
			name:    "index",
			request: httptest.NewRequest(http.MethodGet, "/profiles-index.json", nil),
			handler: profilesIndexHandler(siteRoot),
		},
		{
			name:    "doc",
			request: httptest.NewRequest(http.MethodGet, "/api/profiles/doc?path=profiles/Locked.md", nil),
			handler: profilesDocHandler(siteRoot),
		},
		{
			name:    "search",
			request: httptest.NewRequest(http.MethodGet, "/api/profiles/search?q=locked", nil),
			handler: profilesSearchHandler(siteRoot),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			tc.handler.ServeHTTP(recorder, tc.request)
			if recorder.Code != http.StatusLocked {
				t.Fatalf("expected status 423, got %d", recorder.Code)
			}
		})
	}
}

func TestFeedHandlers(t *testing.T) {
	siteRoot := t.TempDir()

	// 1. Test reddit handler with invalid/malicious subreddit parameter
	{
		req := httptest.NewRequest(http.MethodGet, "/api/feeds/reddit?subreddit=../../bad", nil)
		recorder := httptest.NewRecorder()
		feedRedditHandler(siteRoot).ServeHTTP(recorder, req)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("expected Bad Request for invalid subreddit, got %d", recorder.Code)
		}
	}

	// 2. Test reddit handler with empty subreddit parameter
	{
		req := httptest.NewRequest(http.MethodGet, "/api/feeds/reddit", nil)
		recorder := httptest.NewRecorder()
		feedRedditHandler(siteRoot).ServeHTTP(recorder, req)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("expected Bad Request for empty subreddit, got %d", recorder.Code)
		}
	}

	// 3. Test youtube handler with invalid channel_id parameter
	{
		req := httptest.NewRequest(http.MethodGet, "/api/feeds/youtube?channel_id=../../bad", nil)
		recorder := httptest.NewRecorder()
		feedYoutubeHandler(siteRoot).ServeHTTP(recorder, req)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("expected Bad Request for invalid channel_id, got %d", recorder.Code)
		}
	}

	// 4. Test youtube handler with empty channel_id parameter
	{
		req := httptest.NewRequest(http.MethodGet, "/api/feeds/youtube", nil)
		recorder := httptest.NewRecorder()
		feedYoutubeHandler(siteRoot).ServeHTTP(recorder, req)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("expected Bad Request for empty channel_id, got %d", recorder.Code)
		}
	}
}

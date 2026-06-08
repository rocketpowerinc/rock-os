package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	testProfileName = "Family/Profiles/Boys"
	markdownDir     = profilesDir + "/" + testProfileName + "/wiki"
	scriptsDir      = profilesDir + "/" + testProfileName + "/scripts"
	bootstrapsDir   = profilesDir + "/" + testProfileName + "/bootstraps"
	cheatsheetsDir  = profilesDir + "/" + testProfileName + "/cheatsheets"
	dotfilesDir     = profilesDir + "/" + testProfileName + "/dotfiles"
	bookmarksDir    = profilesDir + "/" + testProfileName + "/bookmarks"
)

func withTestProfile(siteRoot string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = writeActiveDashboardSessionState(siteRoot, "Family")
		_ = os.MkdirAll(filepath.Join(siteRoot, filepath.FromSlash(profilesDir), testProfileName), 0o755)
		query := r.URL.Query()
		query.Set("profile", testProfileName)
		r.URL.RawQuery = query.Encode()
		handler(w, r)
	}
}

func wikiDocHandler(siteRoot string) http.HandlerFunc {
	return withTestProfile(siteRoot, profileMarkdownDocHandler(siteRoot, "wiki"))
}

func markdownIndexHandler(siteRoot string) http.HandlerFunc {
	return withTestProfile(siteRoot, profileMarkdownIndexHandler(siteRoot, "wiki"))
}

func wikiSearchHandler(siteRoot string) http.HandlerFunc {
	return withTestProfile(siteRoot, profileMarkdownSearchHandler(siteRoot, "wiki"))
}

func resolveMarkdownDoc(siteRoot string, docPath string) (string, string, error) {
	return resolveProfileMarkdownDoc(siteRoot, testProfileName, "wiki", docPath)
}

func collectMarkdownFiles(siteRoot string) ([]markdownIndexEntry, error) {
	return collectProfileMarkdownFiles(siteRoot, testProfileName, "wiki")
}

func bootstrapsIndexHandler(siteRoot string) http.HandlerFunc {
	return withTestProfile(siteRoot, profileMarkdownIndexHandler(siteRoot, "bootstraps"))
}

func cheatsheetsIndexHandler(siteRoot string) http.HandlerFunc {
	return withTestProfile(siteRoot, profileMarkdownIndexHandler(siteRoot, "cheatsheets"))
}

func dotfilesIndexHandler(siteRoot string) http.HandlerFunc {
	return withTestProfile(siteRoot, profileMarkdownIndexHandler(siteRoot, "dotfiles"))
}

func bookmarksIndexHandler(siteRoot string) http.HandlerFunc {
	return withTestProfile(siteRoot, profileMarkdownIndexHandler(siteRoot, "bookmarks"))
}

func resolveBootstrapDoc(siteRoot string, docPath string) (string, string, error) {
	return resolveProfileMarkdownDoc(siteRoot, testProfileName, "bootstraps", docPath)
}

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

func TestRequireUnlockedContentRejectsLockedContent(t *testing.T) {
	siteRoot := t.TempDir()
	encryptedRoot := filepath.Join(siteRoot, encryptedDir)
	if err := os.MkdirAll(encryptedRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(encryptedRoot, "locked.md"), []byte("GITCRYPT\nencrypted data"), 0o644); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/wiki/search?q=test", nil)
	recorder := httptest.NewRecorder()
	requireUnlockedContent(siteRoot, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("locked content reached the wrapped handler")
	})(recorder, request)

	if recorder.Code != http.StatusLocked {
		t.Fatalf("expected status 423, got %d", recorder.Code)
	}
}

func TestResolveScriptRejectsUnsupportedCharacters(t *testing.T) {
	_, _, err := resolveScript(t.TempDir(), testProfileName, "Linux/update;rm.sh")
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
	if err := writeActiveDashboardSessionState(siteRoot, "Family"); err != nil {
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
	if err := writeActiveDashboardSessionState(siteRoot, "Family"); err != nil {
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

func TestLinkHealthSourceFilesRespectActiveSession(t *testing.T) {
	siteRoot := t.TempDir()
	docs := map[string]string{
		"Family/Profiles/Boys":     "Boys.md",
		"SysAdmin/Profiles/Rocket": "Rocket.md",
	}
	for profile, name := range docs {
		docPath := filepath.Join(siteRoot, filepath.FromSlash(profilesDir), filepath.FromSlash(profile), "wiki", name)
		if err := os.MkdirAll(filepath.Dir(docPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(docPath, []byte("# "+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := writeActiveDashboardSessionState(siteRoot, "Family"); err != nil {
		t.Fatal(err)
	}

	files, err := linkHealthSourceFiles(siteRoot)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 || files[0] != "ENCRYPTED/Sessions/Family/Profiles/Boys/wiki/Boys.md" {
		t.Fatalf("expected only Family/Boys link-health sources, got %#v", files)
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

	request := httptest.NewRequest(http.MethodGet, "/api/wiki/doc?path=ENCRYPTED/Sessions/Family/Profiles/Boys/wiki/Test.md", nil)
	recorder := httptest.NewRecorder()

	wikiDocHandler(siteRoot).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response wikiDocResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}

	if response.Path != "ENCRYPTED/Sessions/Family/Profiles/Boys/wiki/Test.md" {
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

	request := httptest.NewRequest(http.MethodGet, "/api/wiki/doc?path=ENCRYPTED/Sessions/Family/Profiles/Boys/wiki/Unsafe.md", nil)
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
	request := httptest.NewRequest(http.MethodGet, "/api/wiki/doc?path=ENCRYPTED/Sessions/Family/Profiles/Boys/wiki/../secret.md", nil)
	recorder := httptest.NewRecorder()

	wikiDocHandler(t.TempDir()).ServeHTTP(recorder, request)

	if recorder.Code == http.StatusOK {
		t.Fatal("path traversal request was allowed")
	}
}

func TestDashboardsDocHandlerRendersHubOverview(t *testing.T) {
	siteRoot := t.TempDir()
	if err := writeActiveDashboardSessionState(siteRoot, "Doomsday"); err != nil {
		t.Fatal(err)
	}

	docPath := filepath.Join(siteRoot, filepath.FromSlash(profilesDir), "Doomsday", "Profiles", "Prepper", "Hub-Overview.md")
	if err := os.MkdirAll(filepath.Dir(docPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(docPath, []byte("# Prepper Hub Overview\n\nPrivate notes."), 0o644); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/dashboards/doc?path=ENCRYPTED/Sessions/Doomsday/Profiles/Prepper/Hub-Overview.md", nil)
	recorder := httptest.NewRecorder()

	dashboardsDocHandler(siteRoot).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response wikiDocResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Path != "ENCRYPTED/Sessions/Doomsday/Profiles/Prepper/Hub-Overview.md" {
		t.Fatalf("unexpected response path: %q", response.Path)
	}
	if !strings.Contains(response.HTML, "<h1 id=\"prepper-hub-overview\">Prepper Hub Overview</h1>") {
		t.Fatalf("expected rendered hub heading, got: %s", response.HTML)
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

	if response.Results[0].Path != "ENCRYPTED/Sessions/Family/Profiles/Boys/wiki/Linux/Booting.md" {
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

	withTestProfile(siteRoot, scriptsSearchHandler(siteRoot)).ServeHTTP(recorder, request)

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

	if files[0].Path != "ENCRYPTED/Sessions/Family/Profiles/Boys/wiki/Fresh.md" {
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

	for _, dir := range []string{markdownDir, bootstrapsDir, cheatsheetsDir, dotfilesDir, bookmarksDir, scriptsDir, profilesDir, "css", "js"} {
		if err := os.MkdirAll(filepath.Join(siteRoot, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	for _, file := range []string{"index.html", "wiki.html", "bootstraps.html", "cheatsheets.html", "dotfiles.html", "bookmarks.html", "scripts.html", "dashboards.html"} {
		if err := os.WriteFile(filepath.Join(siteRoot, file), []byte(file), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestServerStatusHandlerReturnsGitCryptStatus(t *testing.T) {
	siteRoot := t.TempDir()
	createTestWebsiteRoot(t, siteRoot)
	if err := os.RemoveAll(filepath.Join(siteRoot, encryptedDir)); err != nil {
		t.Fatal(err)
	}

	// Case 1: missing (encrypted content folder doesn't exist)
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
	if status.WikiCount != 0 {
		t.Errorf("expected wikiCount to be 0, got %d", status.WikiCount)
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
	if status.Commit != "" {
		t.Errorf("expected commit to be empty outside a Git clone, got %q", status.Commit)
	}

	// Case 2: unlocked (encrypted content folder exists with readable files)
	privateDir := filepath.Join(siteRoot, filepath.FromSlash(profilesDir), "SysAdmin", "Profiles", "Rocket")
	if err := os.MkdirAll(privateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(privateDir, "doc.md"), []byte("plain text doc"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write a wiki markdown file to ensure WikiCount is 1.
	wikiDoc := filepath.Join(siteRoot, markdownDir, "doc.md")
	if err := os.MkdirAll(filepath.Dir(wikiDoc), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wikiDoc, []byte("# Hello"), 0o644); err != nil {
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
	if err := writeActiveDashboardSessionState(siteRoot, "Family"); err != nil {
		t.Fatal(err)
	}

	// The lock status is cached briefly; drop the cache so this call reflects the
	// content we just created rather than the earlier "missing" result.
	invalidatePrivateMarkdownStatus()

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

	// Case 3: locked (encrypted content contains a git-crypt file)
	if err := os.WriteFile(filepath.Join(privateDir, "locked-doc.md"), []byte("GITCRYPT\nencrypted data here"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Drop the cached "unlocked" result so the locked file is detected now.
	invalidatePrivateMarkdownStatus()

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

func TestServerStatusHandlerReturnsCurrentCommit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	repoRoot := t.TempDir()
	siteRoot := filepath.Join(repoRoot, "Website")
	createTestWebsiteRoot(t, siteRoot)

	runGit := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repoRoot
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
		}
		return strings.TrimSpace(string(output))
	}

	runGit("init")
	runGit("add", ".")
	runGit("-c", "user.name=Rock-OS Tests", "-c", "user.email=tests@rock-os.local", "commit", "-m", "initial")
	expectedCommit := runGit("rev-parse", "HEAD")

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

	if status.Commit != expectedCommit {
		t.Fatalf("expected commit %q, got %q", expectedCommit, status.Commit)
	}
}

func TestResolveBootstrapDoc(t *testing.T) {
	siteRoot := t.TempDir()
	bootstrapsRoot := filepath.Join(siteRoot, bootstrapsDir)
	if err := os.MkdirAll(bootstrapsRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	docPath := filepath.Join(bootstrapsRoot, "Setup.md")
	if err := os.WriteFile(docPath, []byte("# Setup"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Normal resolve
	resolvedPath, fullPath, err := resolveBootstrapDoc(siteRoot, "ENCRYPTED/Sessions/Family/Profiles/Boys/bootstraps/Setup.md")
	if err != nil {
		t.Fatal(err)
	}
	if resolvedPath != "ENCRYPTED/Sessions/Family/Profiles/Boys/bootstraps/Setup.md" {
		t.Errorf("expected ENCRYPTED/Sessions/Family/Profiles/Boys/bootstraps/Setup.md, got %q", resolvedPath)
	}
	if !strings.HasSuffix(fullPath, "Setup.md") {
		t.Errorf("expected path to end with Setup.md, got %q", fullPath)
	}

	// Path traversal check
	_, _, err = resolveBootstrapDoc(siteRoot, "ENCRYPTED/Sessions/Family/Profiles/Boys/bootstraps/../secret.md")
	if err == nil {
		t.Error("expected error for path traversal attempt")
	}

	// Non-markdown file check
	_, _, err = resolveBootstrapDoc(siteRoot, "ENCRYPTED/Sessions/Family/Profiles/Boys/bootstraps/Setup.txt")
	if err == nil {
		t.Error("expected error for non-markdown extension")
	}
}

func TestBootstrapsIndexHandler(t *testing.T) {
	siteRoot := t.TempDir()
	createTestWebsiteRoot(t, siteRoot)
	bootstrapsRoot := filepath.Join(siteRoot, bootstrapsDir)
	if err := os.MkdirAll(bootstrapsRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	docPath := filepath.Join(bootstrapsRoot, "Install.md")
	if err := os.WriteFile(docPath, []byte("# Install"), 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/bootstraps-index.json", nil)
	rec := httptest.NewRecorder()
	bootstrapsIndexHandler(siteRoot).ServeHTTP(rec, req)

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
	if index[0].Path != "ENCRYPTED/Sessions/Family/Profiles/Boys/bootstraps/Install.md" {
		t.Errorf("expected ENCRYPTED/Sessions/Family/Profiles/Boys/bootstraps/Install.md, got %q", index[0].Path)
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
	if index[0].Path != "ENCRYPTED/Sessions/Family/Profiles/Boys/cheatsheets/Commands.md" {
		t.Errorf("expected ENCRYPTED/Sessions/Family/Profiles/Boys/cheatsheets/Commands.md, got %q", index[0].Path)
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
	if index[0].Path != "ENCRYPTED/Sessions/Family/Profiles/Boys/dotfiles/Shell.md" {
		t.Errorf("expected ENCRYPTED/Sessions/Family/Profiles/Boys/dotfiles/Shell.md, got %q", index[0].Path)
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
	if index[0].Path != "ENCRYPTED/Sessions/Family/Profiles/Boys/bookmarks/Links.md" {
		t.Errorf("expected ENCRYPTED/Sessions/Family/Profiles/Boys/bookmarks/Links.md, got %q", index[0].Path)
	}
}

func TestProfileMarkdownIndexHandlerScopesFilesToRequestedProfile(t *testing.T) {
	siteRoot := t.TempDir()
	kidsWiki := filepath.Join(siteRoot, filepath.FromSlash(profilesDir), "Family", "Profiles", "Boys", "wiki")
	rocketWiki := filepath.Join(siteRoot, filepath.FromSlash(profilesDir), "SysAdmin", "Profiles", "Rocket", "wiki")
	for _, dir := range []string{kidsWiki, rocketWiki} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(kidsWiki, "Kids.md"), []byte("# Kids"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rocketWiki, "Rocket.md"), []byte("# Rocket"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeActiveDashboardSessionState(siteRoot, "Family"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/wiki-index.json?profile=Family%2FProfiles%2FBoys", nil)
	rec := httptest.NewRecorder()
	profileMarkdownIndexHandler(siteRoot, "wiki").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var index []markdownIndexEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &index); err != nil {
		t.Fatal(err)
	}
	if len(index) != 1 || index[0].Path != "ENCRYPTED/Sessions/Family/Profiles/Boys/wiki/Kids.md" {
		t.Fatalf("expected only Kids wiki content, got %#v", index)
	}
}

func TestAllowedProfileNamesIncludesActiveSessionProfiles(t *testing.T) {
	siteRoot := t.TempDir()
	for _, profile := range []string{"Family/Profiles/Boys", "Family/Profiles/Girls", "SysAdmin/Profiles/Rocket"} {
		profileRoot := filepath.Join(siteRoot, filepath.FromSlash(profilesDir), filepath.FromSlash(profile))
		if err := os.MkdirAll(profileRoot, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(profileRoot, "index.html"), []byte(profile), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := writeActiveDashboardSessionState(siteRoot, "Family"); err != nil {
		t.Fatal(err)
	}

	profiles, err := allowedProfileNames(siteRoot)
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]bool{
		"Family/Profiles/Boys":  true,
		"Family/Profiles/Girls": true,
	}
	if len(profiles) != len(expected) {
		t.Fatalf("expected active Family profiles only, got %#v", profiles)
	}
	for _, profile := range profiles {
		if !expected[profile] {
			t.Fatalf("unexpected profile %q in %#v", profile, profiles)
		}
	}
}

func TestNestedProfileMarkdownIndexScopesToRequestedChildProfile(t *testing.T) {
	siteRoot := t.TempDir()
	boysWiki := filepath.Join(siteRoot, filepath.FromSlash(profilesDir), "Family", "Profiles", "Boys", "wiki")
	girlsWiki := filepath.Join(siteRoot, filepath.FromSlash(profilesDir), "Family", "Profiles", "Girls", "wiki")
	for _, dir := range []string{boysWiki, girlsWiki} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(boysWiki, "Boys.md"), []byte("# Boys"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(girlsWiki, "Girls.md"), []byte("# Girls"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeActiveDashboardSessionState(siteRoot, "Family"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/wiki-index.json?profile=Family%2FProfiles%2FBoys", nil)
	rec := httptest.NewRecorder()
	profileMarkdownIndexHandler(siteRoot, "wiki").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var index []markdownIndexEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &index); err != nil {
		t.Fatal(err)
	}
	if len(index) != 1 || index[0].Path != "ENCRYPTED/Sessions/Family/Profiles/Boys/wiki/Boys.md" {
		t.Fatalf("expected only Boys wiki content, got %#v", index)
	}
}

func TestProfileContentHandlersRejectProfilesOutsideActiveSession(t *testing.T) {
	siteRoot := t.TempDir()
	for _, profile := range []string{"Family/Profiles/Boys", "SysAdmin/Profiles/Rocket"} {
		if err := os.MkdirAll(filepath.Join(siteRoot, filepath.FromSlash(profilesDir), filepath.FromSlash(profile)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := writeActiveDashboardSessionState(siteRoot, "Family"); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name    string
		handler http.HandlerFunc
		target  string
	}{
		{
			name:    "wiki index",
			handler: profileMarkdownIndexHandler(siteRoot, "wiki"),
			target:  "/wiki-index.json?profile=SysAdmin%2FProfiles%2FRocket",
		},
		{
			name:    "scripts list",
			handler: scriptsListHandler(siteRoot),
			target:  "/api/scripts?profile=SysAdmin%2FProfiles%2FRocket",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.target, nil)
			rec := httptest.NewRecorder()
			test.handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestDashboardsIndexExcludesProfileWorkspaceMarkdown(t *testing.T) {
	siteRoot := t.TempDir()
	if err := writeDashboardTestDoc(siteRoot, "Profiles", "Boys", "Hub-Overview.md"); err != nil {
		t.Fatal(err)
	}
	wikiDoc := filepath.Join(siteRoot, filepath.FromSlash(profilesDir), "Family", "Profiles", "Boys", "wiki", "Private.md")
	if err := os.MkdirAll(filepath.Dir(wikiDoc), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wikiDoc, []byte("# Private"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeActiveDashboardSessionState(siteRoot, "Family"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboards-index.json", nil)
	rec := httptest.NewRecorder()
	dashboardsIndexHandler(siteRoot).ServeHTTP(rec, req)

	var index []markdownIndexEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &index); err != nil {
		t.Fatal(err)
	}
	if len(index) != 1 || index[0].Path != "ENCRYPTED/Sessions/Family/Profiles/Boys/Hub-Overview.md" {
		t.Fatalf("expected only Family/Boys profile notes, got %#v", index)
	}
}

func TestDashboardsIndexHandlerIncludesProfilesCategory(t *testing.T) {
	siteRoot := t.TempDir()
	profileRoot := filepath.Join(siteRoot, filepath.FromSlash(profilesDir), "Family", "Profiles", "Boys")
	if err := os.MkdirAll(profileRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	docPath := filepath.Join(profileRoot, "Hub-Overview.md")
	if err := os.WriteFile(docPath, []byte("# Boys"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeSessionFile(siteRoot, "Family"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboards-index.json", nil)
	rec := httptest.NewRecorder()
	dashboardsIndexHandler(siteRoot).ServeHTTP(rec, req)

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
	if index[0].Path != "ENCRYPTED/Sessions/Family/Profiles/Boys/Hub-Overview.md" {
		t.Errorf("expected Family/Profiles/Boys profile file, got %q", index[0].Path)
	}
}

func TestDashboardsIndexHandlerUsesSysAdminByDefaultWithoutRocketKey(t *testing.T) {
	siteRoot := t.TempDir()
	if err := writeDashboardTestDoc(siteRoot, "Profiles", "Kids", "Hub-Overview.md"); err != nil {
		t.Fatal(err)
	}
	if err := writeDashboardTestDoc(siteRoot, "OS", "Windows", "Dashboard-Overview.md"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboards-index.json", nil)
	rec := httptest.NewRecorder()
	dashboardsIndexHandler(siteRoot).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var index []markdownIndexEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &index); err != nil {
		t.Fatal(err)
	}

	if len(index) != 0 {
		t.Fatalf("expected default SysAdmin session without Rocket key to hide Rocket files, got %#v", index)
	}
}

func TestDashboardsIndexHandlerUsesNamedProfileSession(t *testing.T) {
	siteRoot := t.TempDir()
	if err := writeDashboardTestDoc(siteRoot, "Profiles", "Kids", "Hub-Overview.md"); err != nil {
		t.Fatal(err)
	}
	if err := writeDashboardTestDoc(siteRoot, "Profiles", "Rocket", "Hub-Overview.md"); err != nil {
		t.Fatal(err)
	}
	if err := writeSessionFile(siteRoot, "Family"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboards-index.json", nil)
	rec := httptest.NewRecorder()
	dashboardsIndexHandler(siteRoot).ServeHTTP(rec, req)

	var index []markdownIndexEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &index); err != nil {
		t.Fatal(err)
	}

	if len(index) != 1 {
		t.Fatalf("expected only Kids profile file, got %#v", index)
	}
	if index[0].Path != "ENCRYPTED/Sessions/Family/Profiles/Boys/Hub-Overview.md" {
		t.Fatalf("unexpected dashboard file: %#v", index[0])
	}
}

func TestDashboardsIndexHandlerUsesSysAdminWithoutRocketKey(t *testing.T) {
	siteRoot := t.TempDir()
	if err := writeDashboardTestDoc(siteRoot, "Profiles", "Admin", "Hub-Overview.md"); err != nil {
		t.Fatal(err)
	}
	if err := writeDashboardTestDoc(siteRoot, "Profiles", "Rocket", "Hub-Overview.md"); err != nil {
		t.Fatal(err)
	}
	if err := writeDashboardTestDoc(siteRoot, "OS", "Windows", "Dashboard-Overview.md"); err != nil {
		t.Fatal(err)
	}
	if err := writeSessionFile(siteRoot, "SysAdmin"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboards-index.json", nil)
	rec := httptest.NewRecorder()
	dashboardsIndexHandler(siteRoot).ServeHTTP(rec, req)

	var index []markdownIndexEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &index); err != nil {
		t.Fatal(err)
	}

	paths := map[string]bool{}
	for _, entry := range index {
		paths[entry.Path] = true
	}

	if len(index) != 1 ||
		!paths["ENCRYPTED/Sessions/SysAdmin/Profiles/Admin/Hub-Overview.md"] {
		t.Fatalf("expected SysAdmin without rocket key to hide Rocket profile, got %#v", index)
	}
	if paths["ENCRYPTED/Sessions/SysAdmin/Profiles/Rocket/Hub-Overview.md"] ||
		paths["ENCRYPTED/Sessions/SysAdmin/Profiles/Rocket/dashboards/OS/Windows/Dashboard-Overview.md"] {
		t.Fatalf("expected SysAdmin without rocket key to hide Rocket profile, got %#v", index)
	}
}

func TestDashboardsIndexHandlerUsesSysAdminWithRocketKey(t *testing.T) {
	siteRoot := t.TempDir()
	createRocketKey(t, siteRoot)
	if err := writeDashboardTestDoc(siteRoot, "Profiles", "Admin", "Hub-Overview.md"); err != nil {
		t.Fatal(err)
	}
	if err := writeDashboardTestDoc(siteRoot, "Profiles", "Rocket", "Hub-Overview.md"); err != nil {
		t.Fatal(err)
	}
	if err := writeDashboardTestDoc(siteRoot, "OS", "Windows", "Dashboard-Overview.md"); err != nil {
		t.Fatal(err)
	}
	if err := writeSessionFile(siteRoot, "SysAdmin"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboards-index.json", nil)
	rec := httptest.NewRecorder()
	dashboardsIndexHandler(siteRoot).ServeHTTP(rec, req)

	var index []markdownIndexEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &index); err != nil {
		t.Fatal(err)
	}

	paths := map[string]bool{}
	for _, entry := range index {
		paths[entry.Path] = true
	}

	if len(index) != 3 ||
		!paths["ENCRYPTED/Sessions/SysAdmin/Profiles/Admin/Hub-Overview.md"] ||
		!paths["ENCRYPTED/Sessions/SysAdmin/Profiles/Rocket/Hub-Overview.md"] ||
		!paths["ENCRYPTED/Sessions/SysAdmin/Profiles/Rocket/dashboards/OS/Windows/Dashboard-Overview.md"] {
		t.Fatalf("expected SysAdmin with rocket key to include Rocket profile, got %#v", index)
	}
}

func TestDashboardsIndexHandlerUsesMappedSession(t *testing.T) {
	siteRoot := t.TempDir()
	createRocketKey(t, siteRoot)
	if err := writeDashboardTestDoc(siteRoot, "Homelab", "SelfHosting", "Dashboard-Overview.md"); err != nil {
		t.Fatal(err)
	}
	if err := writeDashboardTestDoc(siteRoot, "OS", "Windows", "Dashboard-Overview.md"); err != nil {
		t.Fatal(err)
	}
	config := defaultDashboardSessionsConfig()
	config.Active = "SelfHosting"
	config.Sessions = append(config.Sessions, dashboardSession{
		Name:        "SelfHosting",
		AllowedPath: "SysAdmin",
		Description: "Shows only Rocket.",
	})
	if err := writeDashboardSessionsConfig(siteRoot, config); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboards-index.json", nil)
	rec := httptest.NewRecorder()
	dashboardsIndexHandler(siteRoot).ServeHTTP(rec, req)

	var index []markdownIndexEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &index); err != nil {
		t.Fatal(err)
	}

	if len(index) != 2 {
		t.Fatalf("expected only mapped profile files, got %#v", index)
	}
	paths := map[string]bool{}
	for _, entry := range index {
		paths[entry.Path] = true
	}
	if !paths["ENCRYPTED/Sessions/SysAdmin/Profiles/Rocket/dashboards/Homelab/SelfHosting/Dashboard-Overview.md"] ||
		!paths["ENCRYPTED/Sessions/SysAdmin/Profiles/Rocket/dashboards/OS/Windows/Dashboard-Overview.md"] {
		t.Fatalf("unexpected dashboard files: %#v", index)
	}
}

func writeDashboardTestDoc(siteRoot string, category string, dashboard string, fileName string) error {
	var root string
	if category == "Profiles" {
		switch dashboard {
		case "Admin":
			dashboard = "SysAdmin/Profiles/Admin"
		case "Rocket":
			dashboard = "SysAdmin/Profiles/Rocket"
		case "Kids", "Boys":
			dashboard = "Family/Profiles/Boys"
		}
		root = filepath.Join(siteRoot, filepath.FromSlash(profilesDir), dashboard)
	} else {
		root = filepath.Join(siteRoot, filepath.FromSlash(profilesDir), "SysAdmin", "Profiles", "Rocket", dashboardsSection, category, dashboard)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, fileName), []byte("# Test"), 0o644)
}

func TestSessionsHandlerReturnsConfig(t *testing.T) {
	siteRoot := t.TempDir()
	if err := writeSessionFile(siteRoot, "Family"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	rec := httptest.NewRecorder()
	sessionsHandler(siteRoot).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var config dashboardSessionsConfig
	if err := json.Unmarshal(rec.Body.Bytes(), &config); err != nil {
		t.Fatal(err)
	}

	if config.Active != "Family" {
		t.Fatalf("expected active Family session, got %#v", config)
	}
	if !dashboardSessionExists(config.Sessions, "SysAdmin") ||
		!dashboardSessionExists(config.Sessions, "Family") ||
		!dashboardSessionExists(config.Sessions, "Doomsday") {
		t.Fatalf("expected starter sessions, got %#v", config.Sessions)
	}
	if dashboardSessionExists(config.Sessions, "Rocket") {
		t.Fatalf("did not expect Rocket as a session, got %#v", config.Sessions)
	}
}

func TestSessionsHandlerUpdatesFamilySessionFromLoopback(t *testing.T) {
	siteRoot := t.TempDir()
	if err := writeSessionFile(siteRoot, "SysAdmin"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8000/api/sessions", strings.NewReader(`{"active":"Family"}`))
	req.RemoteAddr = "127.0.0.1:49200"
	req.Header.Set("X-Rock-OS-Requested", "true")
	req.Header.Set("Origin", "http://127.0.0.1:8000")
	req.Header.Set("Referer", "http://127.0.0.1:8000/index.html")
	rec := httptest.NewRecorder()
	sessionsHandler(siteRoot).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	config := readDashboardSessionsConfig(siteRoot)
	if config.Active != "Family" {
		t.Fatalf("expected active Family session, got %#v", config)
	}
}

func TestSessionsHandlerWritesLocalActiveSessionState(t *testing.T) {
	siteRoot := t.TempDir()
	if err := writeSessionFile(siteRoot, "SysAdmin"); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(siteRoot, filepath.FromSlash(sessionsFile))
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8000/api/sessions", strings.NewReader(`{"active":"Doomsday"}`))
	req.RemoteAddr = "127.0.0.1:49200"
	req.Header.Set("X-Rock-OS-Requested", "true")
	req.Header.Set("Origin", "http://127.0.0.1:8000")
	req.Header.Set("Referer", "http://127.0.0.1:8000/index.html")
	rec := httptest.NewRecorder()
	sessionsHandler(siteRoot).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("expected tracked sessions config to remain unchanged")
	}

	statePath := filepath.Join(siteRoot, filepath.FromSlash(activeSessionFile))
	content, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}

	var state activeDashboardSessionState
	if err := json.Unmarshal(content, &state); err != nil {
		t.Fatal(err)
	}
	if state.Active != "Doomsday" {
		t.Fatalf("expected local active session to be Doomsday, got %#v", state)
	}
}

func TestRocketProfileRequiresRocketKey(t *testing.T) {
	siteRoot := t.TempDir()
	if err := writeSessionFile(siteRoot, "SysAdmin"); err != nil {
		t.Fatal(err)
	}

	if dashboardSessionAllowsPath(siteRoot, "SysAdmin/Profiles/Rocket") {
		t.Fatal("expected Rocket profile to be blocked without rocket key")
	}
	if !dashboardSessionAllowsPath(siteRoot, "SysAdmin/Profiles/Admin") {
		t.Fatal("expected Admin profile to be available without rocket key")
	}
	createRocketKey(t, siteRoot)
	if !dashboardSessionAllowsPath(siteRoot, "SysAdmin/Profiles/Rocket") {
		t.Fatal("expected Rocket profile to be available with rocket key")
	}
}

func TestSessionsHandlerRejectsRocketAsSession(t *testing.T) {
	siteRoot := t.TempDir()
	if err := writeSessionFile(siteRoot, "SysAdmin"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8000/api/sessions", strings.NewReader(`{"active":"Rocket"}`))
	req.RemoteAddr = "127.0.0.1:49200"
	req.Header.Set("X-Rock-OS-Requested", "true")
	req.Header.Set("Origin", "http://127.0.0.1:8000")
	req.Header.Set("Referer", "http://127.0.0.1:8000/index.html")
	rec := httptest.NewRecorder()
	sessionsHandler(siteRoot).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestSessionsHandlerRejectsPublicSession(t *testing.T) {
	siteRoot := t.TempDir()
	if err := writeSessionFile(siteRoot, "SysAdmin"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8000/api/sessions", strings.NewReader(`{"active":"Public"}`))
	req.RemoteAddr = "127.0.0.1:49200"
	req.Header.Set("X-Rock-OS-Requested", "true")
	req.Header.Set("Origin", "http://127.0.0.1:8000")
	req.Header.Set("Referer", "http://127.0.0.1:8000/index.html")
	rec := httptest.NewRecorder()
	sessionsHandler(siteRoot).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestSessionsHandlerRejectsUnknownSession(t *testing.T) {
	siteRoot := t.TempDir()
	if err := writeSessionFile(siteRoot, "SysAdmin"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8000/api/sessions", strings.NewReader(`{"active":"Missing"}`))
	req.RemoteAddr = "127.0.0.1:49200"
	req.Header.Set("X-Rock-OS-Requested", "true")
	req.Header.Set("Origin", "http://127.0.0.1:8000")
	req.Header.Set("Referer", "http://127.0.0.1:8000/index.html")
	rec := httptest.NewRecorder()
	sessionsHandler(siteRoot).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func writeSessionFile(siteRoot string, active string) error {
	config := defaultDashboardSessionsConfig()
	config.Active = active
	return writeDashboardSessionsConfig(siteRoot, config)
}

func createRocketKey(t *testing.T, siteRoot string) {
	t.Helper()

	keyPath := filepath.Join(siteRoot, filepath.FromSlash(sessionKeysDir), rocketKeyFile)
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, []byte("test rocket marker"), 0o644); err != nil {
		t.Fatal(err)
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

func newEncryptedGuardServer(siteRoot string) http.Handler {
	return guardEncryptedStatic(siteRoot, http.FileServer(http.Dir(siteRoot)))
}

func TestGuardEncryptedStaticBlocksSessionKeys(t *testing.T) {
	siteRoot := t.TempDir()
	keyPath := filepath.Join(siteRoot, filepath.FromSlash(sessionKeysDir), "example.key")
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, target := range []string{"/Sessions-State/Keys", "/Sessions-State/Keys/example.key"} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		rec := httptest.NewRecorder()
		newEncryptedGuardServer(siteRoot).ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected session key path %q to return 404, got %d", target, rec.Code)
		}
	}
}

func TestGuardEncryptedStaticBlocksDirectoryListing(t *testing.T) {
	siteRoot := t.TempDir()
	createRocketKey(t, siteRoot)
	if err := writeActiveDashboardSessionState(siteRoot, "SysAdmin"); err != nil {
		t.Fatal(err)
	}

	assetDir := filepath.Join(siteRoot, filepath.FromSlash(profilesDir), "SysAdmin", "Profiles", "Rocket", dashboardsSection, "Foo", "Bar", "assets")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "icon.txt"), []byte("icon-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	invalidatePrivateMarkdownStatus()

	handler := newEncryptedGuardServer(siteRoot)

	// Directory requests must return 404 (no listing) rather than leaking the
	// names of private folders/documents.
	for _, dirPath := range []string{"/ENCRYPTED", "/ENCRYPTED/", "/ENCRYPTED/dashboards", "/ENCRYPTED/Sessions/SysAdmin/Profiles/Rocket/dashboards/Foo/Bar/assets/"} {
		req := httptest.NewRequest(http.MethodGet, dirPath, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 for directory %q, got %d", dirPath, rec.Code)
		}
	}

	// Individual asset files are still served while unlocked (dashboards need
	// their local icons/images).
	req := httptest.NewRequest(http.MethodGet, "/ENCRYPTED/Sessions/SysAdmin/Profiles/Rocket/dashboards/Foo/Bar/assets/icon.txt", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for asset file, got %d", rec.Code)
	}
	if rec.Body.String() != "icon-bytes" {
		t.Fatalf("unexpected asset body: %q", rec.Body.String())
	}
}

func TestGuardEncryptedStaticBlocksRawProfileMarkdown(t *testing.T) {
	siteRoot := t.TempDir()
	docPath := filepath.Join(siteRoot, filepath.FromSlash(profilesDir), "Family", "Profiles", "Boys", "wiki", "Secret.md")
	if err := os.MkdirAll(filepath.Dir(docPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(docPath, []byte("# Secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeActiveDashboardSessionState(siteRoot, "Family"); err != nil {
		t.Fatal(err)
	}
	invalidatePrivateMarkdownStatus()

	req := httptest.NewRequest(http.MethodGet, "/ENCRYPTED/Sessions/Family/Profiles/Boys/wiki/Secret.md", nil)
	rec := httptest.NewRecorder()
	newEncryptedGuardServer(siteRoot).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for raw profile markdown, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGuardEncryptedStaticAllowsDashboardIndexDirectory(t *testing.T) {
	siteRoot := t.TempDir()
	createRocketKey(t, siteRoot)
	if err := writeActiveDashboardSessionState(siteRoot, "SysAdmin"); err != nil {
		t.Fatal(err)
	}

	dashboardDir := filepath.Join(siteRoot, filepath.FromSlash(profilesDir), "SysAdmin", "Profiles", "Rocket", dashboardsSection, "OS", "Windows")
	if err := os.MkdirAll(dashboardDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dashboardDir, "index.html"), []byte("dashboard-index"), 0o644); err != nil {
		t.Fatal(err)
	}
	invalidatePrivateMarkdownStatus()

	handler := newEncryptedGuardServer(siteRoot)
	req := httptest.NewRequest(http.MethodGet, "/ENCRYPTED/Sessions/SysAdmin/Profiles/Rocket/dashboards/OS/Windows/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for dashboard index directory, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "dashboard-index") {
		t.Fatalf("expected dashboard index body, got %q", rec.Body.String())
	}
}

func TestGuardEncryptedStaticAllowsNestedProfileIndexDirectory(t *testing.T) {
	siteRoot := t.TempDir()
	if err := writeActiveDashboardSessionState(siteRoot, "Family"); err != nil {
		t.Fatal(err)
	}

	profileDir := filepath.Join(siteRoot, filepath.FromSlash(profilesDir), "Family", "Profiles", "Boys")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profileDir, "index.html"), []byte("boys-index"), 0o644); err != nil {
		t.Fatal(err)
	}
	invalidatePrivateMarkdownStatus()

	handler := newEncryptedGuardServer(siteRoot)
	req := httptest.NewRequest(http.MethodGet, "/ENCRYPTED/Sessions/Family/Profiles/Boys/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for nested profile index directory, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "boys-index") {
		t.Fatalf("expected nested profile index body, got %q", rec.Body.String())
	}
}

func TestGuardEncryptedStaticAllowsNestedProfileRootConfigFiles(t *testing.T) {
	siteRoot := t.TempDir()
	if err := writeActiveDashboardSessionState(siteRoot, "Family"); err != nil {
		t.Fatal(err)
	}

	profileDir := filepath.Join(siteRoot, filepath.FromSlash(profilesDir), "Family", "Profiles", "Boys")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"dashboard.json", "widgets.txt"} {
		if err := os.WriteFile(filepath.Join(profileDir, file), []byte(file), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	invalidatePrivateMarkdownStatus()

	handler := newEncryptedGuardServer(siteRoot)
	for _, file := range []string{"dashboard.json", "widgets.txt"} {
		req := httptest.NewRequest(http.MethodGet, "/ENCRYPTED/Sessions/Family/Profiles/Boys/"+file, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 for nested profile %s, got %d", file, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), file) {
			t.Fatalf("expected nested profile %s body, got %q", file, rec.Body.String())
		}
	}
}

func TestGuardEncryptedStaticAllowsNestedProfileAssets(t *testing.T) {
	siteRoot := t.TempDir()
	if err := writeActiveDashboardSessionState(siteRoot, "Family"); err != nil {
		t.Fatal(err)
	}

	assetPath := filepath.Join(siteRoot, filepath.FromSlash(profilesDir), "Family", "Profiles", "Boys", "assets", "Boys-Steel.svg")
	if err := os.MkdirAll(filepath.Dir(assetPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(assetPath, []byte("boys-svg"), 0o644); err != nil {
		t.Fatal(err)
	}
	invalidatePrivateMarkdownStatus()

	handler := newEncryptedGuardServer(siteRoot)
	req := httptest.NewRequest(http.MethodGet, "/ENCRYPTED/Sessions/Family/Profiles/Boys/assets/Boys-Steel.svg", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for nested profile asset, got %d", rec.Code)
	}
	if rec.Body.String() != "boys-svg" {
		t.Fatalf("expected nested profile asset body, got %q", rec.Body.String())
	}
}

func TestGuardEncryptedStaticBlocksWhenLocked(t *testing.T) {
	siteRoot := t.TempDir()
	wikiDir := filepath.Join(siteRoot, filepath.FromSlash(profilesDir), "Family", "Profiles", "Boys", "wiki")
	if err := os.MkdirAll(wikiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wikiDir, "Secret.md"), []byte("\x00GITCRYPT\x00encrypted"), 0o644); err != nil {
		t.Fatal(err)
	}
	invalidatePrivateMarkdownStatus()

	handler := newEncryptedGuardServer(siteRoot)
	req := httptest.NewRequest(http.MethodGet, "/ENCRYPTED/Sessions/Family/Profiles/Boys/wiki/Secret.md", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusLocked {
		t.Fatalf("expected 423 while locked, got %d", rec.Code)
	}
}

func TestGuardEncryptedStaticPassesThroughNonEncrypted(t *testing.T) {
	siteRoot := t.TempDir()
	// Use a plain file (not index.html, which http.FileServer 301-redirects to "/").
	if err := os.WriteFile(filepath.Join(siteRoot, "robots.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	invalidatePrivateMarkdownStatus()

	handler := newEncryptedGuardServer(siteRoot)
	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for non-encrypted file, got %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
}

func TestPrivateMarkdownStatusCacheInvalidation(t *testing.T) {
	siteRoot := t.TempDir()
	encryptedRoot := filepath.Join(siteRoot, encryptedDir)
	if err := os.MkdirAll(encryptedRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(encryptedRoot, "Plain.md"), []byte("# Plain\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	invalidatePrivateMarkdownStatus()

	if status := privateMarkdownStatus(siteRoot); status != "unlocked" {
		t.Fatalf("expected unlocked, got %q", status)
	}

	// Lock the tree on disk; the cached value should persist until invalidated.
	if err := os.WriteFile(filepath.Join(encryptedRoot, "Plain.md"), []byte("\x00GITCRYPT\x00data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if status := privateMarkdownStatus(siteRoot); status != "unlocked" {
		t.Fatalf("expected cached unlocked before invalidation, got %q", status)
	}

	invalidatePrivateMarkdownStatus()
	if status := privateMarkdownStatus(siteRoot); status != "locked" {
		t.Fatalf("expected locked after invalidation, got %q", status)
	}
}

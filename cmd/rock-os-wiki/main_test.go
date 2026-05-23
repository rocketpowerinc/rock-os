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
	"testing"
	"time"
)

func TestScriptRunRequestAllowedRequiresRockOSHeader(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8000/api/scripts/run", nil)

	if scriptRunRequestAllowed(request) {
		t.Fatal("script run request without Rock-OS header was allowed")
	}
}

func TestScriptRunRequestAllowedRejectsCrossOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8000/api/scripts/run", nil)
	request.Header.Set("X-Rock-OS-Requested", "true")
	request.Header.Set("Origin", "https://example.com")

	if scriptRunRequestAllowed(request) {
		t.Fatal("cross-origin script run request was allowed")
	}
}

func TestScriptRunRequestAllowedAcceptsSameOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://192.168.1.2:8000/api/scripts/run", nil)
	request.Header.Set("X-Rock-OS-Requested", "true")
	request.Header.Set("Origin", "http://192.168.1.2:8000")
	request.Header.Set("Referer", "http://192.168.1.2:8000/scripts.html")

	if !scriptRunRequestAllowed(request) {
		t.Fatal("same-origin script run request was rejected")
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
	commandRoot := filepath.Join(repoRoot, "cmd", "rock-os-wiki")
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

	request := httptest.NewRequest(http.MethodGet, "/api/wiki/doc?path=markdown/Test.md", nil)
	recorder := httptest.NewRecorder()

	wikiDocHandler(siteRoot).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response wikiDocResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}

	if response.Path != "markdown/Test.md" {
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

	request := httptest.NewRequest(http.MethodGet, "/api/wiki/doc?path=markdown/Unsafe.md", nil)
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
	request := httptest.NewRequest(http.MethodGet, "/api/wiki/doc?path=markdown/../secret.md", nil)
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

	if response.Results[0].Path != "markdown/Linux/Booting.md" {
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
		[]byte("---\npinned: true\n---\n# Fresh\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/markdown-index.json", nil)
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

	if files[0].Path != "markdown/Fresh.md" || !files[0].Pinned {
		t.Fatalf("unexpected index entry: %#v", files[0])
	}
}

func TestCollectMarkdownFilesCachesPinnedMetadata(t *testing.T) {
	siteRoot := t.TempDir()
	markdownRoot := filepath.Join(siteRoot, markdownDir)
	if err := os.MkdirAll(markdownRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	docPath := filepath.Join(markdownRoot, "Pinned.md")
	if err := os.WriteFile(docPath, []byte("---\npinned: true\n---\n# Pinned\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cache := &markdownIndexCache{entries: map[string]markdownIndexCacheEntry{}}
	files, err := collectMarkdownFilesWithCache(siteRoot, cache)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 || !files[0].Pinned {
		t.Fatalf("expected pinned file in index, got %#v", files)
	}

	updatedTime := time.Now().Add(2 * time.Second)
	if err := os.WriteFile(docPath, []byte("# Not pinned\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(docPath, updatedTime, updatedTime); err != nil {
		t.Fatal(err)
	}

	files, err = collectMarkdownFilesWithCache(siteRoot, cache)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 || files[0].Pinned {
		t.Fatalf("expected changed file to refresh pinned status, got %#v", files)
	}
}

func TestCollectMarkdownFilesPrunesDeletedCacheEntries(t *testing.T) {
	siteRoot := t.TempDir()
	markdownRoot := filepath.Join(siteRoot, markdownDir)
	if err := os.MkdirAll(markdownRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	docPath := filepath.Join(markdownRoot, "DeleteMe.md")
	if err := os.WriteFile(docPath, []byte("---\npinned: true\n---\n# Delete me\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cache := &markdownIndexCache{entries: map[string]markdownIndexCacheEntry{}}
	if _, err := collectMarkdownFilesWithCache(siteRoot, cache); err != nil {
		t.Fatal(err)
	}

	if len(cache.entries) != 1 {
		t.Fatalf("expected one cache entry, got %d", len(cache.entries))
	}

	if err := os.Remove(docPath); err != nil {
		t.Fatal(err)
	}

	files, err := collectMarkdownFilesWithCache(siteRoot, cache)
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

	for _, dir := range []string{markdownDir, scriptsDir, "css", "js"} {
		if err := os.MkdirAll(filepath.Join(siteRoot, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	for _, file := range []string{"index.html", "wiki.html", "scripts.html"} {
		if err := os.WriteFile(filepath.Join(siteRoot, file), []byte(file), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

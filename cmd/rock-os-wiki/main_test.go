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

	for _, dir := range []string{markdownDir, guidesDir, cheatsheetsDir, dotfilesDir, bookmarksDir, scriptsDir, profilesDir, "css", "js"} {
		if err := os.MkdirAll(filepath.Join(siteRoot, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	for _, file := range []string{"index.html", "wiki.html", "guides.html", "cheatsheets.html", "dotfiles.html", "bookmarks.html", "scripts.html", "profiles.html"} {
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

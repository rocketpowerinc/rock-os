package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

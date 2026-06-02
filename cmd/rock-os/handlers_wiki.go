package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	searchSnippetHTMLTagPattern       = regexp.MustCompile(`(?is)<[^>]+>`)
	searchSnippetImagePattern         = regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`)
	searchSnippetLinkPattern          = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	searchSnippetInlineCodePattern    = regexp.MustCompile("`([^`]*)`")
	searchSnippetHeadingPattern       = regexp.MustCompile(`(?m)^\s{0,3}#{1,6}\s*`)
	searchSnippetBlockquotePattern    = regexp.MustCompile(`(?m)^\s{0,3}>\s?`)
	searchSnippetUnorderedListPattern = regexp.MustCompile(`(?m)^\s*[-*+]\s+`)
	searchSnippetOrderedListPattern   = regexp.MustCompile(`(?m)^\s*\d+[.)]\s+`)
	searchSnippetEmphasisPattern      = regexp.MustCompile(`[*_~]+`)
)

func wikiDocHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		docPath, path, err := resolveMarkdownDoc(siteRoot, r.URL.Query().Get("path"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		content, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "markdown document not found", http.StatusNotFound)
				return
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var rendered bytes.Buffer
		if err := wikiMarkdown.Convert(content, &rendered); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := wikiDocResponse{
			Path: docPath,
			HTML: rendered.String(),
		}

		if info, err := os.Stat(path); err == nil {
			response.LastEdited = info.ModTime().Format(time.RFC3339)
		}

		writeJSON(w, response)
	}
}

func markdownIndexHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		files, err := collectMarkdownFiles(siteRoot)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, files)
	}
}

func wikiSearchHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if query == "" {
			writeJSON(w, wikiSearchResponse{
				Results: []wikiSearchResult{},
			})
			return
		}

		results, err := searchWiki(siteRoot, query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, wikiSearchResponse{
			Results: results,
		})
	}
}

func searchWiki(siteRoot string, query string) ([]wikiSearchResult, error) {
	files, err := collectMarkdownFiles(siteRoot)
	if err != nil {
		return nil, err
	}
	return searchMarkdownIndex(siteRoot, query, files, defaultApp.Caches.Search.Markdown)
}

func fileTitle(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	return strings.TrimSuffix(parts[len(parts)-1], filepath.Ext(parts[len(parts)-1]))
}

func searchSnippet(text string, normalizedQuery string) string {
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if !strings.Contains(strings.ToLower(line), normalizedQuery) {
			continue
		}

		cleaned := cleanSearchSnippetLine(line)
		if cleaned == "" {
			continue
		}
		if len(cleaned) <= 120 {
			return cleaned
		}

		matchIndex := strings.Index(strings.ToLower(cleaned), normalizedQuery)
		if matchIndex < 0 {
			return cleaned[:min(len(cleaned), 120)]
		}
		start := max(0, matchIndex-45)
		end := min(len(cleaned), start+120)
		prefix := ""
		suffix := ""
		if start > 0 {
			prefix = "..."
		}
		if end < len(cleaned) {
			suffix = "..."
		}

		return prefix + cleaned[start:end] + suffix
	}

	return ""
}

func cleanSearchSnippetLine(line string) string {
	line = strings.TrimSpace(line)
	line = searchSnippetHTMLTagPattern.ReplaceAllString(line, " ")
	line = searchSnippetImagePattern.ReplaceAllString(line, "$1")
	line = searchSnippetLinkPattern.ReplaceAllString(line, "$1")
	line = searchSnippetInlineCodePattern.ReplaceAllString(line, "$1")
	line = searchSnippetHeadingPattern.ReplaceAllString(line, "")
	line = searchSnippetBlockquotePattern.ReplaceAllString(line, "")
	line = searchSnippetUnorderedListPattern.ReplaceAllString(line, "")
	line = searchSnippetOrderedListPattern.ReplaceAllString(line, "")
	line = searchSnippetEmphasisPattern.ReplaceAllString(line, "")
	line = strings.ReplaceAll(line, `\`, "")
	line = strings.Join(strings.Fields(line), " ")
	return line
}

func resolveMarkdownDoc(siteRoot string, docPath string) (string, string, error) {
	normalized := filepath.ToSlash(
		filepath.Clean(
			strings.ReplaceAll(docPath, "\\", "/"),
		),
	)
	normalized = strings.TrimPrefix(normalized, "/")

	if normalized == "." || normalized == "" || strings.Contains(normalized, "\x00") {
		return "", "", fmt.Errorf("markdown document path is required")
	}

	if !strings.HasPrefix(normalized, markdownDir+"/") {
		return "", "", fmt.Errorf("markdown document path must start with %s/", markdownDir)
	}

	if !strings.EqualFold(filepath.Ext(normalized), ".md") {
		return "", "", fmt.Errorf("markdown document must be a .md file")
	}

	markdownRoot, err := filepath.Abs(filepath.Join(siteRoot, markdownDir))
	if err != nil {
		return "", "", err
	}

	target, err := filepath.Abs(filepath.Join(siteRoot, filepath.FromSlash(normalized)))
	if err != nil {
		return "", "", err
	}

	relativeTarget, err := filepath.Rel(markdownRoot, target)
	if err != nil {
		return "", "", err
	}

	if relativeTarget == "." ||
		relativeTarget == ".." ||
		filepath.IsAbs(relativeTarget) ||
		strings.HasPrefix(relativeTarget, ".."+string(os.PathSeparator)) {
		return "", "", fmt.Errorf("markdown document must stay inside %s", markdownDir)
	}

	return normalized, target, nil
}

func writeMarkdownIndex(siteRoot string) (bool, error) {
	files, err := collectMarkdownFiles(siteRoot)
	if err != nil {
		return false, err
	}

	nextJSON, err := json.MarshalIndent(files, "", "  ")
	if err != nil {
		return false, err
	}

	nextJSON = append(nextJSON, '\n')

	indexPath := filepath.Join(siteRoot, indexFile)
	previousJSON, err := os.ReadFile(indexPath)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	if bytes.Equal(previousJSON, nextJSON) {
		return false, nil
	}

	return true, os.WriteFile(indexPath, nextJSON, 0o644)
}

func collectMarkdownFiles(siteRoot string) ([]markdownIndexEntry, error) {
	return collectMarkdownFilesWithCache(siteRoot, markdownDir, defaultApp.Caches.Markdown)
}

func collectMarkdownFilesWithCache(siteRoot string, subDir string, cache *markdownIndexCache) ([]markdownIndexEntry, error) {
	root := filepath.Join(siteRoot, subDir)
	files := []markdownIndexEntry{}
	seen := map[string]struct{}{}

	if _, err := os.Stat(root); os.IsNotExist(err) {
		cache.prune(seen)
		return files, nil
	}

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		if !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		relativePath, err := filepath.Rel(siteRoot, path)
		if err != nil {
			return err
		}

		absolutePath, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		seen[absolutePath] = struct{}{}
		cache.markSeen(absolutePath, info)

		files = append(files, markdownIndexEntry{
			Path: filepath.ToSlash(relativePath),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	cache.prune(seen)

	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Path) < strings.ToLower(files[j].Path)
	})

	return files, nil
}

func collectGuideFiles(siteRoot string) ([]markdownIndexEntry, error) {
	return collectGuideFilesWithCache(siteRoot, defaultApp.Caches.Guides)
}

func collectGuideFilesWithCache(siteRoot string, cache *markdownIndexCache) ([]markdownIndexEntry, error) {
	root := filepath.Join(siteRoot, guidesDir)
	files := []markdownIndexEntry{}
	seen := map[string]struct{}{}

	if _, err := os.Stat(root); os.IsNotExist(err) {
		cache.prune(seen)
		return files, nil
	}

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		if !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		relativePath, err := filepath.Rel(siteRoot, path)
		if err != nil {
			return err
		}

		absolutePath, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		seen[absolutePath] = struct{}{}
		cache.markSeen(absolutePath, info)

		files = append(files, markdownIndexEntry{
			Path: filepath.ToSlash(relativePath),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	cache.prune(seen)

	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Path) < strings.ToLower(files[j].Path)
	})

	return files, nil
}

func writeGuidesIndex(siteRoot string) (bool, error) {
	files, err := collectGuideFiles(siteRoot)
	if err != nil {
		return false, err
	}

	nextJSON, err := json.MarshalIndent(files, "", "  ")
	if err != nil {
		return false, err
	}

	nextJSON = append(nextJSON, '\n')

	indexPath := filepath.Join(siteRoot, guidesIndexFile)
	previousJSON, err := os.ReadFile(indexPath)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	if bytes.Equal(previousJSON, nextJSON) {
		return false, nil
	}

	return true, os.WriteFile(indexPath, nextJSON, 0o644)
}

func guidesIndexHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		files, err := collectGuideFiles(siteRoot)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, files)
	}
}

func guidesDocHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		docPath, path, err := resolveGuideDoc(siteRoot, r.URL.Query().Get("path"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		content, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "guide document not found", http.StatusNotFound)
				return
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var rendered bytes.Buffer
		if err := wikiMarkdown.Convert(content, &rendered); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := wikiDocResponse{
			Path: docPath,
			HTML: rendered.String(),
		}

		if info, err := os.Stat(path); err == nil {
			response.LastEdited = info.ModTime().Format(time.RFC3339)
		}

		writeJSON(w, response)
	}
}

func resolveGuideDoc(siteRoot string, docPath string) (string, string, error) {
	normalized := filepath.ToSlash(
		filepath.Clean(
			strings.ReplaceAll(docPath, "\\", "/"),
		),
	)
	normalized = strings.TrimPrefix(normalized, "/")

	if normalized == "." || normalized == "" || strings.Contains(normalized, "\x00") {
		return "", "", fmt.Errorf("guide document path is required")
	}

	if !strings.HasPrefix(normalized, guidesDir+"/") {
		return "", "", fmt.Errorf("guide document path must start with %s/", guidesDir)
	}

	if !strings.EqualFold(filepath.Ext(normalized), ".md") {
		return "", "", fmt.Errorf("guide document must be a .md file")
	}

	guidesRoot, err := filepath.Abs(filepath.Join(siteRoot, guidesDir))
	if err != nil {
		return "", "", err
	}

	target, err := filepath.Abs(filepath.Join(siteRoot, filepath.FromSlash(normalized)))
	if err != nil {
		return "", "", err
	}

	relativeTarget, err := filepath.Rel(guidesRoot, target)
	if err != nil {
		return "", "", err
	}

	if strings.HasPrefix(relativeTarget, "..") || relativeTarget == "." {
		return "", "", fmt.Errorf("guide document must stay inside %s", guidesDir)
	}

	return normalized, target, nil
}

func guidesSearchHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if query == "" {
			writeJSON(w, wikiSearchResponse{
				Results: []wikiSearchResult{},
			})
			return
		}

		results, err := searchGuides(siteRoot, query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, wikiSearchResponse{
			Results: results,
		})
	}
}

func searchGuides(siteRoot string, query string) ([]wikiSearchResult, error) {
	files, err := collectGuideFiles(siteRoot)
	if err != nil {
		return nil, err
	}
	return searchMarkdownIndex(siteRoot, query, files, defaultApp.Caches.Search.Guides)
}

func collectCheatsheetFiles(siteRoot string) ([]markdownIndexEntry, error) {
	return collectCheatsheetFilesWithCache(siteRoot, defaultApp.Caches.Cheatsheets)
}

func collectCheatsheetFilesWithCache(siteRoot string, cache *markdownIndexCache) ([]markdownIndexEntry, error) {
	root := filepath.Join(siteRoot, cheatsheetsDir)
	files := []markdownIndexEntry{}
	seen := map[string]struct{}{}

	if _, err := os.Stat(root); os.IsNotExist(err) {
		cache.prune(seen)
		return files, nil
	}

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		if !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		relativePath, err := filepath.Rel(siteRoot, path)
		if err != nil {
			return err
		}

		absolutePath, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		seen[absolutePath] = struct{}{}
		cache.markSeen(absolutePath, info)

		files = append(files, markdownIndexEntry{
			Path: filepath.ToSlash(relativePath),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	cache.prune(seen)

	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Path) < strings.ToLower(files[j].Path)
	})

	return files, nil
}

func writeCheatsheetsIndex(siteRoot string) (bool, error) {
	files, err := collectCheatsheetFiles(siteRoot)
	if err != nil {
		return false, err
	}

	nextJSON, err := json.MarshalIndent(files, "", "  ")
	if err != nil {
		return false, err
	}

	nextJSON = append(nextJSON, '\n')

	indexPath := filepath.Join(siteRoot, cheatsheetsIndexFile)
	previousJSON, err := os.ReadFile(indexPath)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	if bytes.Equal(previousJSON, nextJSON) {
		return false, nil
	}

	return true, os.WriteFile(indexPath, nextJSON, 0o644)
}

func cheatsheetsIndexHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		files, err := collectCheatsheetFiles(siteRoot)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, files)
	}
}

func cheatsheetsDocHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		docPath, path, err := resolveCheatsheetDoc(siteRoot, r.URL.Query().Get("path"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		content, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "cheatsheet document not found", http.StatusNotFound)
				return
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var rendered bytes.Buffer
		if err := wikiMarkdown.Convert(content, &rendered); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := wikiDocResponse{
			Path: docPath,
			HTML: rendered.String(),
		}

		if info, err := os.Stat(path); err == nil {
			response.LastEdited = info.ModTime().Format(time.RFC3339)
		}

		writeJSON(w, response)
	}
}

func resolveCheatsheetDoc(siteRoot string, docPath string) (string, string, error) {
	normalized := filepath.ToSlash(
		filepath.Clean(
			strings.ReplaceAll(docPath, "\\", "/"),
		),
	)
	normalized = strings.TrimPrefix(normalized, "/")

	if normalized == "." || normalized == "" || strings.Contains(normalized, "\x00") {
		return "", "", fmt.Errorf("cheatsheet document path is required")
	}

	if !strings.HasPrefix(normalized, cheatsheetsDir+"/") {
		return "", "", fmt.Errorf("cheatsheet document path must start with %s/", cheatsheetsDir)
	}

	if !strings.EqualFold(filepath.Ext(normalized), ".md") {
		return "", "", fmt.Errorf("cheatsheet document must be a .md file")
	}

	cheatsheetsRoot, err := filepath.Abs(filepath.Join(siteRoot, cheatsheetsDir))
	if err != nil {
		return "", "", err
	}

	target, err := filepath.Abs(filepath.Join(siteRoot, filepath.FromSlash(normalized)))
	if err != nil {
		return "", "", err
	}

	relativeTarget, err := filepath.Rel(cheatsheetsRoot, target)
	if err != nil {
		return "", "", err
	}

	if strings.HasPrefix(relativeTarget, "..") || relativeTarget == "." {
		return "", "", fmt.Errorf("cheatsheet document must stay inside %s", cheatsheetsDir)
	}

	return normalized, target, nil
}

func cheatsheetsSearchHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if query == "" {
			writeJSON(w, wikiSearchResponse{
				Results: []wikiSearchResult{},
			})
			return
		}

		results, err := searchCheatsheets(siteRoot, query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, wikiSearchResponse{
			Results: results,
		})
	}
}

func searchCheatsheets(siteRoot string, query string) ([]wikiSearchResult, error) {
	files, err := collectCheatsheetFiles(siteRoot)
	if err != nil {
		return nil, err
	}
	return searchMarkdownIndex(siteRoot, query, files, defaultApp.Caches.Search.Cheatsheets)
}

func collectDotfileFiles(siteRoot string) ([]markdownIndexEntry, error) {
	return collectDotfileFilesWithCache(siteRoot, defaultApp.Caches.Dotfiles)
}

func collectDotfileFilesWithCache(siteRoot string, cache *markdownIndexCache) ([]markdownIndexEntry, error) {
	root := filepath.Join(siteRoot, dotfilesDir)
	files := []markdownIndexEntry{}
	seen := map[string]struct{}{}

	if _, err := os.Stat(root); os.IsNotExist(err) {
		cache.prune(seen)
		return files, nil
	}

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		if !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		relativePath, err := filepath.Rel(siteRoot, path)
		if err != nil {
			return err
		}

		absolutePath, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		seen[absolutePath] = struct{}{}
		cache.markSeen(absolutePath, info)

		files = append(files, markdownIndexEntry{
			Path: filepath.ToSlash(relativePath),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	cache.prune(seen)

	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Path) < strings.ToLower(files[j].Path)
	})

	return files, nil
}

func writeDotfilesIndex(siteRoot string) (bool, error) {
	files, err := collectDotfileFiles(siteRoot)
	if err != nil {
		return false, err
	}

	nextJSON, err := json.MarshalIndent(files, "", "  ")
	if err != nil {
		return false, err
	}

	nextJSON = append(nextJSON, '\n')

	indexPath := filepath.Join(siteRoot, dotfilesIndexFile)
	previousJSON, err := os.ReadFile(indexPath)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	if bytes.Equal(previousJSON, nextJSON) {
		return false, nil
	}

	return true, os.WriteFile(indexPath, nextJSON, 0o644)
}

func dotfilesIndexHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		files, err := collectDotfileFiles(siteRoot)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, files)
	}
}

func dotfilesDocHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		docPath, path, err := resolveDotfileDoc(siteRoot, r.URL.Query().Get("path"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		content, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "dotfile document not found", http.StatusNotFound)
				return
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var rendered bytes.Buffer
		if err := wikiMarkdown.Convert(content, &rendered); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := wikiDocResponse{
			Path: docPath,
			HTML: rendered.String(),
		}

		if info, err := os.Stat(path); err == nil {
			response.LastEdited = info.ModTime().Format(time.RFC3339)
		}

		writeJSON(w, response)
	}
}

func resolveDotfileDoc(siteRoot string, docPath string) (string, string, error) {
	normalized := filepath.ToSlash(
		filepath.Clean(
			strings.ReplaceAll(docPath, "\\", "/"),
		),
	)
	normalized = strings.TrimPrefix(normalized, "/")

	if normalized == "." || normalized == "" || strings.Contains(normalized, "\x00") {
		return "", "", fmt.Errorf("dotfile document path is required")
	}

	if !strings.HasPrefix(normalized, dotfilesDir+"/") {
		return "", "", fmt.Errorf("dotfile document path must start with %s/", dotfilesDir)
	}

	if !strings.EqualFold(filepath.Ext(normalized), ".md") {
		return "", "", fmt.Errorf("dotfile document must be a .md file")
	}

	dotfilesRoot, err := filepath.Abs(filepath.Join(siteRoot, dotfilesDir))
	if err != nil {
		return "", "", err
	}

	target, err := filepath.Abs(filepath.Join(siteRoot, filepath.FromSlash(normalized)))
	if err != nil {
		return "", "", err
	}

	relativeTarget, err := filepath.Rel(dotfilesRoot, target)
	if err != nil {
		return "", "", err
	}

	if strings.HasPrefix(relativeTarget, "..") || relativeTarget == "." {
		return "", "", fmt.Errorf("dotfile document must stay inside %s", dotfilesDir)
	}

	return normalized, target, nil
}

func dotfilesSearchHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if query == "" {
			writeJSON(w, wikiSearchResponse{
				Results: []wikiSearchResult{},
			})
			return
		}

		results, err := searchDotfiles(siteRoot, query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, wikiSearchResponse{
			Results: results,
		})
	}
}

func searchDotfiles(siteRoot string, query string) ([]wikiSearchResult, error) {
	files, err := collectDotfileFiles(siteRoot)
	if err != nil {
		return nil, err
	}
	return searchMarkdownIndex(siteRoot, query, files, defaultApp.Caches.Search.Dotfiles)
}

func collectBookmarkFiles(siteRoot string) ([]markdownIndexEntry, error) {
	return collectBookmarkFilesWithCache(siteRoot, defaultApp.Caches.Bookmarks)
}

func collectBookmarkFilesWithCache(siteRoot string, cache *markdownIndexCache) ([]markdownIndexEntry, error) {
	root := filepath.Join(siteRoot, bookmarksDir)
	files := []markdownIndexEntry{}
	seen := map[string]struct{}{}

	if _, err := os.Stat(root); os.IsNotExist(err) {
		cache.prune(seen)
		return files, nil
	}

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		if !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		relativePath, err := filepath.Rel(siteRoot, path)
		if err != nil {
			return err
		}

		absolutePath, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		seen[absolutePath] = struct{}{}
		cache.markSeen(absolutePath, info)

		files = append(files, markdownIndexEntry{
			Path: filepath.ToSlash(relativePath),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	cache.prune(seen)

	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Path) < strings.ToLower(files[j].Path)
	})

	return files, nil
}

func writeBookmarksIndex(siteRoot string) (bool, error) {
	files, err := collectBookmarkFiles(siteRoot)
	if err != nil {
		return false, err
	}

	nextJSON, err := json.MarshalIndent(files, "", "  ")
	if err != nil {
		return false, err
	}

	nextJSON = append(nextJSON, '\n')

	indexPath := filepath.Join(siteRoot, bookmarksIndexFile)
	previousJSON, err := os.ReadFile(indexPath)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	if bytes.Equal(previousJSON, nextJSON) {
		return false, nil
	}

	return true, os.WriteFile(indexPath, nextJSON, 0o644)
}

func bookmarksIndexHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		files, err := collectBookmarkFiles(siteRoot)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, files)
	}
}

func bookmarksDocHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		docPath, path, err := resolveBookmarkDoc(siteRoot, r.URL.Query().Get("path"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		content, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "bookmark document not found", http.StatusNotFound)
				return
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var rendered bytes.Buffer
		if err := wikiMarkdown.Convert(content, &rendered); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := wikiDocResponse{
			Path: docPath,
			HTML: rendered.String(),
		}

		if info, err := os.Stat(path); err == nil {
			response.LastEdited = info.ModTime().Format(time.RFC3339)
		}

		writeJSON(w, response)
	}
}

func resolveBookmarkDoc(siteRoot string, docPath string) (string, string, error) {
	normalized := filepath.ToSlash(
		filepath.Clean(
			strings.ReplaceAll(docPath, "\\", "/"),
		),
	)
	normalized = strings.TrimPrefix(normalized, "/")

	if normalized == "." || normalized == "" || strings.Contains(normalized, "\x00") {
		return "", "", fmt.Errorf("bookmark document path is required")
	}

	if !strings.HasPrefix(normalized, bookmarksDir+"/") {
		return "", "", fmt.Errorf("bookmark document path must start with %s/", bookmarksDir)
	}

	if !strings.EqualFold(filepath.Ext(normalized), ".md") {
		return "", "", fmt.Errorf("bookmark document must be a .md file")
	}

	bookmarksRoot, err := filepath.Abs(filepath.Join(siteRoot, bookmarksDir))
	if err != nil {
		return "", "", err
	}

	target, err := filepath.Abs(filepath.Join(siteRoot, filepath.FromSlash(normalized)))
	if err != nil {
		return "", "", err
	}

	relativeTarget, err := filepath.Rel(bookmarksRoot, target)
	if err != nil {
		return "", "", err
	}

	if strings.HasPrefix(relativeTarget, "..") || relativeTarget == "." {
		return "", "", fmt.Errorf("bookmark document must stay inside %s", bookmarksDir)
	}

	return normalized, target, nil
}

func bookmarksSearchHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if query == "" {
			writeJSON(w, wikiSearchResponse{
				Results: []wikiSearchResult{},
			})
			return
		}

		results, err := searchBookmarks(siteRoot, query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, wikiSearchResponse{
			Results: results,
		})
	}
}

func searchBookmarks(siteRoot string, query string) ([]wikiSearchResult, error) {
	files, err := collectBookmarkFiles(siteRoot)
	if err != nil {
		return nil, err
	}
	return searchMarkdownIndex(siteRoot, query, files, defaultApp.Caches.Search.Bookmarks)
}

func dashboardsIndexHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		files, err := collectDashboardsFiles(siteRoot)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		files = filterDashboardFiles(files, r.URL.Query().Get("profile"))

		writeJSON(w, files)
	}
}

func dashboardsDocHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		docPath, path, err := resolveDashboardsDoc(siteRoot, r.URL.Query().Get("path"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		content, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "dashboard document not found", http.StatusNotFound)
				return
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var rendered bytes.Buffer
		if err := wikiMarkdown.Convert(content, &rendered); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := wikiDocResponse{
			Path: docPath,
			HTML: rendered.String(),
		}

		if info, err := os.Stat(path); err == nil {
			response.LastEdited = info.ModTime().Format(time.RFC3339)
		}

		writeJSON(w, response)
	}
}

func resolveDashboardsDoc(siteRoot string, docPath string) (string, string, error) {
	normalized := filepath.ToSlash(
		filepath.Clean(
			strings.ReplaceAll(docPath, "\\", "/"),
		),
	)
	normalized = strings.TrimPrefix(normalized, "/")

	if normalized == "." || normalized == "" || strings.Contains(normalized, "\x00") {
		return "", "", fmt.Errorf("dashboard document path is required")
	}

	if !strings.HasPrefix(normalized, dashboardsDir+"/") {
		return "", "", fmt.Errorf("dashboard document path must start with %s/", dashboardsDir)
	}

	if !strings.EqualFold(filepath.Ext(normalized), ".md") {
		return "", "", fmt.Errorf("dashboard document must be a .md file")
	}

	dashboardsRoot, err := filepath.Abs(filepath.Join(siteRoot, dashboardsDir))
	if err != nil {
		return "", "", err
	}

	target, err := filepath.Abs(filepath.Join(siteRoot, filepath.FromSlash(normalized)))
	if err != nil {
		return "", "", err
	}

	relativeTarget, err := filepath.Rel(dashboardsRoot, target)
	if err != nil {
		return "", "", err
	}

	if strings.HasPrefix(relativeTarget, "..") || relativeTarget == "." {
		return "", "", fmt.Errorf("dashboard document must stay inside %s", dashboardsDir)
	}

	return normalized, target, nil
}

func dashboardsSearchHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if query == "" {
			writeJSON(w, wikiSearchResponse{
				Results: []wikiSearchResult{},
			})
			return
		}

		results, err := searchDashboards(siteRoot, query, r.URL.Query().Get("profile"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, wikiSearchResponse{
			Results: results,
		})
	}
}

func searchDashboards(siteRoot string, query string, dashboard string) ([]wikiSearchResult, error) {
	files, err := collectDashboardsFiles(siteRoot)
	if err != nil {
		return nil, err
	}
	files = filterDashboardFiles(files, dashboard)
	return searchMarkdownIndex(siteRoot, query, files, defaultApp.Caches.Search.Dashboards)
}

func filterDashboardFiles(files []markdownIndexEntry, dashboard string) []markdownIndexEntry {
	dashboard = strings.Trim(strings.ReplaceAll(dashboard, "\\", "/"), "/")
	if dashboard == "" || strings.Contains(dashboard, "\x00") {
		if dashboard == "" {
			return files
		}
		return []markdownIndexEntry{}
	}
	for _, part := range strings.Split(dashboard, "/") {
		if part == "" || part == "." || part == ".." {
			return []markdownIndexEntry{}
		}
	}

	prefix := dashboardsDir + "/" + dashboard + "/"
	filtered := []markdownIndexEntry{}
	for _, file := range files {
		if strings.HasPrefix(file.Path, prefix) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

func writeDashboardsIndex(siteRoot string) (bool, error) {
	files, err := collectDashboardsFiles(siteRoot)
	if err != nil {
		return false, err
	}

	nextJSON, err := json.MarshalIndent(files, "", "  ")
	if err != nil {
		return false, err
	}

	nextJSON = append(nextJSON, '\n')

	indexPath := filepath.Join(siteRoot, dashboardsIndexFile)
	previousJSON, err := os.ReadFile(indexPath)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	if bytes.Equal(previousJSON, nextJSON) {
		return false, nil
	}

	return true, os.WriteFile(indexPath, nextJSON, 0o644)
}

func collectDashboardsFiles(siteRoot string) ([]markdownIndexEntry, error) {
	return collectMarkdownFilesWithCache(siteRoot, dashboardsDir, defaultApp.Caches.Dashboards)
}

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
	profileNamePattern                = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 _-]*$`)
)

var profileWorkspaceSections = map[string]bool{
	"bookmarks":   true,
	"bootstraps":  true,
	"cheatsheets": true,
	"dotfiles":    true,
	"scripts":     true,
	"wiki":        true,
}

var profileMarkdownSections = map[string]bool{
	"bookmarks":   true,
	"bootstraps":  true,
	"cheatsheets": true,
	"dotfiles":    true,
	"wiki":        true,
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

func normalizeProfileName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("profile is required")
	}
	if !profileNamePattern.MatchString(value) {
		return "", fmt.Errorf("profile contains unsupported characters")
	}
	return value, nil
}

func profileDashboardPath(profile string) string {
	return "Profiles/" + profile
}

func profileWorkspaceDir(profile string, section string) (string, error) {
	profile, err := normalizeProfileName(profile)
	if err != nil {
		return "", err
	}

	section = strings.ToLower(strings.TrimSpace(section))
	if !profileWorkspaceSections[section] {
		return "", fmt.Errorf("unknown profile workspace section: %s", section)
	}

	return profilesDir + "/" + profile + "/" + section, nil
}

func profileWorkspaceRequestProfile(w http.ResponseWriter, r *http.Request, siteRoot string) (string, bool) {
	profile, err := normalizeProfileName(r.URL.Query().Get("profile"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", false
	}

	if !dashboardSessionAllowsPath(siteRoot, profileDashboardPath(profile)) {
		http.Error(w, "profile is not available in the active Rock-OS session", http.StatusForbidden)
		return "", false
	}

	profileRoot := filepath.Join(siteRoot, filepath.FromSlash(profilesDir), profile)
	info, err := os.Stat(profileRoot)
	if err != nil || !info.IsDir() {
		http.Error(w, "profile not found", http.StatusNotFound)
		return "", false
	}

	return profile, true
}

func profileSectionCache(section string) *markdownIndexCache {
	switch section {
	case "bookmarks":
		return defaultApp.Caches.Bookmarks
	case "bootstraps":
		return defaultApp.Caches.Bootstraps
	case "cheatsheets":
		return defaultApp.Caches.Cheatsheets
	case "dotfiles":
		return defaultApp.Caches.Dotfiles
	default:
		return defaultApp.Caches.Wiki
	}
}

func profileSectionSearchCache(section string) *markdownSearchIndex {
	switch section {
	case "bookmarks":
		return defaultApp.Caches.Search.Bookmarks
	case "bootstraps":
		return defaultApp.Caches.Search.Bootstraps
	case "cheatsheets":
		return defaultApp.Caches.Search.Cheatsheets
	case "dotfiles":
		return defaultApp.Caches.Search.Dotfiles
	default:
		return defaultApp.Caches.Search.Wiki
	}
}

func profileMarkdownIndexHandler(siteRoot string, section string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		profile, ok := profileWorkspaceRequestProfile(w, r, siteRoot)
		if !ok {
			return
		}

		files, err := collectProfileMarkdownFiles(siteRoot, profile, section)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, files)
	}
}

func profileMarkdownDocHandler(siteRoot string, section string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		profile, ok := profileWorkspaceRequestProfile(w, r, siteRoot)
		if !ok {
			return
		}

		docPath, path, err := resolveProfileMarkdownDoc(siteRoot, profile, section, r.URL.Query().Get("path"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeRenderedMarkdownDoc(w, docPath, path)
	}
}

func profileMarkdownSearchHandler(siteRoot string, section string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		profile, ok := profileWorkspaceRequestProfile(w, r, siteRoot)
		if !ok {
			return
		}

		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if query == "" {
			writeJSON(w, wikiSearchResponse{Results: []wikiSearchResult{}})
			return
		}

		files, err := collectProfileMarkdownFiles(siteRoot, profile, section)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		results, err := searchMarkdownIndex(siteRoot, query, files, profileSectionSearchCache(section))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, wikiSearchResponse{Results: results})
	}
}

func writeRenderedMarkdownDoc(w http.ResponseWriter, docPath string, path string) {
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
		Text: string(content),
	}
	if info, err := os.Stat(path); err == nil {
		response.LastEdited = info.ModTime().Format(time.RFC3339)
	}

	writeJSON(w, response)
}

func resolveProfileMarkdownDoc(siteRoot string, profile string, section string, docPath string) (string, string, error) {
	rootDir, err := profileWorkspaceDir(profile, section)
	if err != nil {
		return "", "", err
	}

	normalized, err := normalizeMarkdownDocPath(docPath)
	if err != nil {
		return "", "", err
	}
	if !strings.HasPrefix(normalized, rootDir+"/") {
		return "", "", fmt.Errorf("markdown document path must start with %s/", rootDir)
	}

	root, err := filepath.Abs(filepath.Join(siteRoot, filepath.FromSlash(rootDir)))
	if err != nil {
		return "", "", err
	}
	target, err := filepath.Abs(filepath.Join(siteRoot, filepath.FromSlash(normalized)))
	if err != nil {
		return "", "", err
	}
	if !pathInsideRoot(root, target) {
		return "", "", fmt.Errorf("markdown document must stay inside %s", rootDir)
	}

	return normalized, target, nil
}

func normalizeMarkdownDocPath(docPath string) (string, error) {
	normalized := filepath.ToSlash(filepath.Clean(strings.ReplaceAll(docPath, "\\", "/")))
	normalized = strings.TrimPrefix(normalized, "/")

	if normalized == "." || normalized == "" || strings.Contains(normalized, "\x00") {
		return "", fmt.Errorf("markdown document path is required")
	}
	if !strings.EqualFold(filepath.Ext(normalized), ".md") {
		return "", fmt.Errorf("markdown document must be a .md file")
	}
	return normalized, nil
}

func pathInsideRoot(root string, target string) bool {
	relativeTarget, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return relativeTarget != "." &&
		relativeTarget != ".." &&
		!filepath.IsAbs(relativeTarget) &&
		!strings.HasPrefix(relativeTarget, ".."+string(os.PathSeparator))
}

func collectProfileMarkdownFiles(siteRoot string, profile string, section string) ([]markdownIndexEntry, error) {
	if !profileMarkdownSections[strings.ToLower(strings.TrimSpace(section))] {
		return nil, fmt.Errorf("unknown profile markdown section: %s", section)
	}

	dir, err := profileWorkspaceDir(profile, section)
	if err != nil {
		return nil, err
	}
	return collectMarkdownFilesWithCache(siteRoot, dir, profileSectionCache(section))
}

func collectAllowedProfileMarkdownFiles(siteRoot string, section string) ([]markdownIndexEntry, error) {
	profiles, err := allowedProfileNames(siteRoot)
	if err != nil {
		return nil, err
	}

	files := []markdownIndexEntry{}
	for _, profile := range profiles {
		profileFiles, err := collectProfileMarkdownFiles(siteRoot, profile, section)
		if err != nil {
			return nil, err
		}
		files = append(files, profileFiles...)
	}
	sortMarkdownIndexEntries(files)
	return files, nil
}

func allowedProfileNames(siteRoot string) ([]string, error) {
	root := filepath.Join(siteRoot, filepath.FromSlash(profilesDir))
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	profiles := []string{}
	for _, entry := range entries {
		if !entry.IsDir() || !dashboardSessionAllowsPath(siteRoot, profileDashboardPath(entry.Name())) {
			continue
		}
		profiles = append(profiles, entry.Name())
	}
	sort.Slice(profiles, func(i, j int) bool {
		return strings.ToLower(profiles[i]) < strings.ToLower(profiles[j])
	})
	return profiles, nil
}

func collectMarkdownFilesWithCache(siteRoot string, subDir string, cache *markdownIndexCache) ([]markdownIndexEntry, error) {
	root := filepath.Join(siteRoot, filepath.FromSlash(subDir))
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
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
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
		files = append(files, markdownIndexEntry{Path: filepath.ToSlash(relativePath)})
		return nil
	})
	if err != nil {
		return nil, err
	}

	cache.prune(seen)
	sortMarkdownIndexEntries(files)
	return files, nil
}

func sortMarkdownIndexEntries(files []markdownIndexEntry) {
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Path) < strings.ToLower(files[j].Path)
	})
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

		files = filterDashboardFilesForActiveSession(siteRoot, files)
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
		dashboard, ok := dashboardPathFromDocument(docPath)
		if !ok || !dashboardSessionAllowsPath(siteRoot, dashboard) {
			http.Error(w, "dashboard is not available in the active Rock-OS session", http.StatusForbidden)
			return
		}

		writeRenderedMarkdownDoc(w, docPath, path)
	}
}

func dashboardsSearchHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if query == "" {
			writeJSON(w, wikiSearchResponse{Results: []wikiSearchResult{}})
			return
		}

		results, err := searchDashboards(siteRoot, query, r.URL.Query().Get("profile"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, wikiSearchResponse{Results: results})
	}
}

func resolveDashboardsDoc(siteRoot string, docPath string) (string, string, error) {
	normalized, err := normalizeMarkdownDocPath(docPath)
	if err != nil {
		return "", "", err
	}
	if !strings.HasPrefix(normalized, dashboardsDir+"/") {
		return "", "", fmt.Errorf("dashboard document path must start with %s/", dashboardsDir)
	}

	root, err := filepath.Abs(filepath.Join(siteRoot, filepath.FromSlash(dashboardsDir)))
	if err != nil {
		return "", "", err
	}
	target, err := filepath.Abs(filepath.Join(siteRoot, filepath.FromSlash(normalized)))
	if err != nil {
		return "", "", err
	}
	if !pathInsideRoot(root, target) {
		return "", "", fmt.Errorf("dashboard document must stay inside %s", dashboardsDir)
	}
	return normalized, target, nil
}

func dashboardPathFromDocument(path string) (string, bool) {
	path = strings.TrimPrefix(filepath.ToSlash(path), dashboardsDir+"/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[0] == "" || parts[1] == "" {
		return "", false
	}
	return parts[0] + "/" + parts[1], true
}

func searchDashboards(siteRoot string, query string, dashboard string) ([]wikiSearchResult, error) {
	files, err := collectDashboardsFiles(siteRoot)
	if err != nil {
		return nil, err
	}
	files = filterDashboardFilesForActiveSession(siteRoot, files)
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
	files, err := collectMarkdownFilesWithCache(siteRoot, dashboardsDir, defaultApp.Caches.Dashboards)
	if err != nil {
		return nil, err
	}

	filtered := []markdownIndexEntry{}
	for _, file := range files {
		if !isProfileWorkspaceMarkdownPath(file.Path) {
			filtered = append(filtered, file)
		}
	}
	return filtered, nil
}

func isProfileWorkspaceMarkdownPath(path string) bool {
	path = strings.TrimPrefix(filepath.ToSlash(path), profilesDir+"/")
	parts := strings.Split(path, "/")
	return len(parts) >= 3 && profileWorkspaceSections[strings.ToLower(parts[1])]
}

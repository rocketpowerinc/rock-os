package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type markdownSearchIndex struct {
	mu      sync.Mutex
	entries map[string]markdownSearchEntry
}

type markdownSearchEntry struct {
	modTime      int64
	size         int64
	title        string
	lowerPath    string
	lowerTitle   string
	content      string
	lowerContent string
}

func newMarkdownSearchIndex() *markdownSearchIndex {
	return &markdownSearchIndex{
		entries: map[string]markdownSearchEntry{},
	}
}

func searchMarkdownIndex(siteRoot string, query string, files []markdownIndexEntry, index *markdownSearchIndex) ([]wikiSearchResult, error) {
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))
	if normalizedQuery == "" {
		return []wikiSearchResult{}, nil
	}

	entries, err := index.snapshot(siteRoot, files)
	if err != nil {
		return nil, err
	}

	results := []wikiSearchResult{}
	for _, file := range files {
		entry, ok := entries[file.Path]
		if !ok {
			continue
		}

		pathMatch := strings.Contains(entry.lowerPath, normalizedQuery) ||
			strings.Contains(entry.lowerTitle, normalizedQuery)
		contentMatch := strings.Contains(entry.lowerContent, normalizedQuery)
		if !pathMatch && !contentMatch {
			continue
		}

		result := wikiSearchResult{
			Path:  file.Path,
			Title: entry.title,
		}
		if contentMatch {
			result.Snippet = searchSnippet(entry.content, normalizedQuery)
		}

		results = append(results, result)
	}

	return results, nil
}

func (index *markdownSearchIndex) snapshot(siteRoot string, files []markdownIndexEntry) (map[string]markdownSearchEntry, error) {
	if index == nil {
		index = newMarkdownSearchIndex()
	}

	seen := map[string]struct{}{}
	index.mu.Lock()
	defer index.mu.Unlock()

	for _, file := range files {
		seen[file.Path] = struct{}{}
		fullPath := filepath.Join(siteRoot, filepath.FromSlash(file.Path))
		info, err := os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				delete(index.entries, file.Path)
				continue
			}
			return nil, err
		}

		entry, ok := index.entries[file.Path]
		modTime := info.ModTime().UnixNano()
		if ok && entry.modTime == modTime && entry.size == info.Size() {
			continue
		}

		content, err := os.ReadFile(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				delete(index.entries, file.Path)
				continue
			}
			return nil, err
		}

		title := fileTitle(file.Path)
		text := string(content)
		index.entries[file.Path] = markdownSearchEntry{
			modTime:      modTime,
			size:         info.Size(),
			title:        title,
			lowerPath:    strings.ToLower(file.Path),
			lowerTitle:   strings.ToLower(title),
			content:      text,
			lowerContent: strings.ToLower(text),
		}
	}

	for path := range index.entries {
		if _, ok := seen[path]; !ok {
			delete(index.entries, path)
		}
	}

	snapshot := make(map[string]markdownSearchEntry, len(index.entries))
	for path, entry := range index.entries {
		snapshot[path] = entry
	}

	return snapshot, nil
}

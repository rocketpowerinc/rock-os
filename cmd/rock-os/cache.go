package main

import (
	"os"
)

func (cache *markdownIndexCache) markSeen(path string, info os.FileInfo) {
	if cache == nil {
		return
	}
	cache.mu.Lock()
	if entry, ok := cache.entries[path]; ok &&
		entry.modTime.Equal(info.ModTime()) &&
		entry.size == info.Size() {
		cache.mu.Unlock()
		return
	}

	if cache.entries == nil {
		cache.entries = map[string]markdownIndexCacheEntry{}
	}
	cache.entries[path] = markdownIndexCacheEntry{
		modTime: info.ModTime(),
		size:    info.Size(),
	}
	cache.mu.Unlock()
}

func (cache *markdownIndexCache) prune(seen map[string]struct{}) {
	if cache == nil {
		return
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	for path := range cache.entries {
		if _, ok := seen[path]; !ok {
			delete(cache.entries, path)
		}
	}
}

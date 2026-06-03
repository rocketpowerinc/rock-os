package main

import (
	"regexp"
	"sync"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var startupTime = time.Now()

const (
	encryptedDir         = "ENCRYPTED"
	adminKeyFile         = "admin.key"
	launchPointsDir      = "launch-point-cards-locked"
	sessionsFile         = "Sessions/sessions.json"
	indexFile            = "wiki-index.json"
	markdownDir          = encryptedDir + "/menu/wiki"
	scriptsDir           = encryptedDir + "/menu/scripts"
	guidesDir            = encryptedDir + "/menu/guides"
	guidesIndexFile      = "guides-index.json"
	cheatsheetsDir       = encryptedDir + "/menu/cheatsheets"
	cheatsheetsIndexFile = "cheatsheets-index.json"
	dotfilesDir          = encryptedDir + "/menu/dotfiles"
	dotfilesIndexFile    = "dotfiles-index.json"
	bookmarksDir         = encryptedDir + "/menu/bookmarks"
	bookmarksIndexFile   = "bookmarks-index.json"
	dashboardsDir        = encryptedDir + "/dashboards"
	dashboardsIndexFile  = "dashboards-index.json"
)

const (
	defaultFeedLimit          = 5
	maxFeedLimit              = 20
	maxFeedURLParams          = 20
	maxRemoteFeedResponseSize = 5 * 1024 * 1024
	apiRateLimitBurst         = 120
	apiRateLimitRefill        = 2.0
)

const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiBlue   = "\033[34m"
	ansiCyan   = "\033[36m"
)

type markdownIndexEntry struct {
	Path string `json:"path"`
}

type scriptEntry struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Runnable bool   `json:"runnable"`
	Platform string `json:"platform"`
}

type scriptRunRequest struct {
	ID string `json:"id"`
}

type launchPoint struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Href        string `json:"href"`
	Path        string `json:"path"`
}

type serverRefreshResponse struct {
	Updated bool   `json:"updated"`
	Message string `json:"message"`
}

type scriptSearchResponse struct {
	Results []scriptSearchResult `json:"results"`
}

type scriptSearchResult struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Runnable bool   `json:"runnable"`
	Platform string `json:"platform"`
	Snippet  string `json:"snippet,omitempty"`
}

type linkHealthResponse struct {
	Checked  int              `json:"checked"`
	OK       int              `json:"ok"`
	Broken   int              `json:"broken"`
	External int              `json:"external"`
	Skipped  int              `json:"skipped"`
	Items    []linkHealthItem `json:"items"`
}

type linkHealthItem struct {
	Source string `json:"source"`
	Label  string `json:"label"`
	Href   string `json:"href"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
	Target string `json:"target,omitempty"`
}

type serverStatus struct {
	Mode         string   `json:"mode"`
	Host         string   `json:"host"`
	Description  string   `json:"description"`
	URLs         []string `json:"urls"`
	GitCrypt     string   `json:"gitCrypt"`
	WikiCount    int      `json:"wikiCount"`
	ScriptsCount int      `json:"scriptsCount"`
	Uptime       int64    `json:"uptime"`
	LastSync     int64    `json:"lastSync"`
	Commit       string   `json:"commit,omitempty"`
}

type wikiDocResponse struct {
	Path       string `json:"path"`
	HTML       string `json:"html"`
	LastEdited string `json:"lastEdited,omitempty"`
}

type wikiSearchResponse struct {
	Results []wikiSearchResult `json:"results"`
}

type wikiSearchResult struct {
	Path    string `json:"path"`
	Title   string `json:"title"`
	Snippet string `json:"snippet,omitempty"`
}

type markdownIndexCacheEntry struct {
	modTime time.Time
	size    int64
}

type markdownIndexCache struct {
	mu      sync.Mutex
	entries map[string]markdownIndexCacheEntry
}

type App struct {
	Caches  *CacheManager
	Flights *FlightManager
}

type FlightManager struct {
	Feeds *requestFlightGroup
}

type CacheManager struct {
	Markdown    *markdownIndexCache
	Guides      *markdownIndexCache
	Cheatsheets *markdownIndexCache
	Dotfiles    *markdownIndexCache
	Bookmarks   *markdownIndexCache
	Dashboards  *markdownIndexCache
	Search      *SearchCacheManager
}

type SearchCacheManager struct {
	Markdown    *markdownSearchIndex
	Guides      *markdownSearchIndex
	Cheatsheets *markdownSearchIndex
	Dotfiles    *markdownSearchIndex
	Bookmarks   *markdownSearchIndex
	Dashboards  *markdownSearchIndex
}

func newApp() *App {
	return &App{
		Caches: &CacheManager{
			Markdown:    newMarkdownIndexCache(),
			Guides:      newMarkdownIndexCache(),
			Cheatsheets: newMarkdownIndexCache(),
			Dotfiles:    newMarkdownIndexCache(),
			Bookmarks:   newMarkdownIndexCache(),
			Dashboards:  newMarkdownIndexCache(),
			Search: &SearchCacheManager{
				Markdown:    newMarkdownSearchIndex(),
				Guides:      newMarkdownSearchIndex(),
				Cheatsheets: newMarkdownSearchIndex(),
				Dotfiles:    newMarkdownSearchIndex(),
				Bookmarks:   newMarkdownSearchIndex(),
				Dashboards:  newMarkdownSearchIndex(),
			},
		},
		Flights: &FlightManager{
			Feeds: newRequestFlightGroup(),
		},
	}
}

func newMarkdownIndexCache() *markdownIndexCache {
	return &markdownIndexCache{
		entries: map[string]markdownIndexCacheEntry{},
	}
}

var defaultApp = newApp()

var safeScriptIDRegex = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 _./-]*\.(sh|ps1|cmd|bat)$`)

var wikiMarkdown = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.Linkify,
		extension.Typographer,
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
	goldmark.WithRendererOptions(
		html.WithHardWraps(),
	),
)

package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var startupTime = time.Now()

const (
	indexFile            = "wiki-index.json"
	markdownDir          = "menu/wiki"
	scriptsDir           = "menu/scripts"
	guidesDir            = "menu/guides"
	guidesIndexFile      = "guides-index.json"
	cheatsheetsDir       = "menu/cheatsheets"
	cheatsheetsIndexFile = "cheatsheets-index.json"
	dotfilesDir          = "menu/dotfiles"
	dotfilesIndexFile    = "dotfiles-index.json"
	bookmarksDir         = "menu/bookmarks"
	bookmarksIndexFile   = "bookmarks-index.json"
	profilesDir          = "profiles"
	profilesIndexFile    = "profiles-index.json"
	dashboardsDir        = "dashboards"
	dashboardsIndexFile  = "dashboards-index.json"
)

const (
	defaultFeedLimit          = 5
	maxFeedLimit              = 20
	maxFeedURLParams          = 20
	maxRemoteFeedResponseSize = 5 * 1024 * 1024
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

var globalMarkdownIndexCache = &markdownIndexCache{
	entries: map[string]markdownIndexCacheEntry{},
}

var globalGuidesIndexCache = &markdownIndexCache{
	entries: map[string]markdownIndexCacheEntry{},
}

var globalCheatsheetsIndexCache = &markdownIndexCache{
	entries: map[string]markdownIndexCacheEntry{},
}

var globalDotfilesIndexCache = &markdownIndexCache{
	entries: map[string]markdownIndexCacheEntry{},
}

var globalBookmarksIndexCache = &markdownIndexCache{
	entries: map[string]markdownIndexCacheEntry{},
}

var globalProfilesIndexCache = &markdownIndexCache{
	entries: map[string]markdownIndexCacheEntry{},
}

var globalDashboardsIndexCache = &markdownIndexCache{
	entries: map[string]markdownIndexCacheEntry{},
}

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

func main() {
	host := flag.String("host", "127.0.0.1", "host to bind: 127.0.0.1, local, lan, 0.0.0.0, or a specific IP")
	port := flag.Int("port", 8000, "port to listen on")
	open := flag.Bool("open", true, "open the site in your default browser")
	buildIndex := flag.Bool("build-index", false, "build wiki-index.json and exit")
	siteRootFlag := flag.String("site-root", "", "path to the Website folder; auto-detected when omitted")
	flag.Parse()

	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	siteRoot, err := resolveSiteRoot(workingDir, *siteRootFlag)
	if err != nil {
		log.Fatal(err)
	}

	if *buildIndex {
		if _, err := writeMarkdownIndex(siteRoot); err != nil {
			log.Fatal(err)
		}
		if _, err := writeGuidesIndex(siteRoot); err != nil {
			log.Fatal(err)
		}
		if _, err := writeCheatsheetsIndex(siteRoot); err != nil {
			log.Fatal(err)
		}
		if _, err := writeDotfilesIndex(siteRoot); err != nil {
			log.Fatal(err)
		}
		if _, err := writeBookmarksIndex(siteRoot); err != nil {
			log.Fatal(err)
		}
		if _, err := writeProfilesIndex(siteRoot); err != nil {
			log.Fatal(err)
		}
		if _, err := writeDashboardsIndex(siteRoot); err != nil {
			log.Fatal(err)
		}

		fmt.Println("Wrote all index.json files")
		return
	}

	bindHost, displayHosts, err := resolveHost(*host)
	if err != nil {
		log.Fatal(err)
	}

	fileServer := noCache(http.FileServer(http.Dir(siteRoot)))
	mux := http.NewServeMux()
	mux.HandleFunc("/api/scripts", scriptsListHandler(siteRoot))
	mux.HandleFunc("/api/scripts/content", scriptContentHandler(siteRoot))
	mux.HandleFunc("/api/scripts/search", scriptsSearchHandler(siteRoot))
	mux.HandleFunc("/api/scripts/run", scriptRunHandler(siteRoot))
	mux.HandleFunc("/api/server/status", serverStatusHandler(bindHost, displayHosts, *port, siteRoot))
	mux.HandleFunc("/api/health/links", linkHealthHandler(siteRoot))
	mux.HandleFunc("/api/wiki/doc", wikiDocHandler(siteRoot))
	mux.HandleFunc("/api/wiki/search", wikiSearchHandler(siteRoot))
	mux.HandleFunc("/wiki-index.json", markdownIndexHandler(siteRoot))
	mux.HandleFunc("/api/guides/doc", guidesDocHandler(siteRoot))
	mux.HandleFunc("/api/guides/search", guidesSearchHandler(siteRoot))
	mux.HandleFunc("/guides-index.json", guidesIndexHandler(siteRoot))
	mux.HandleFunc("/api/cheatsheets/doc", cheatsheetsDocHandler(siteRoot))
	mux.HandleFunc("/api/cheatsheets/search", cheatsheetsSearchHandler(siteRoot))
	mux.HandleFunc("/cheatsheets-index.json", cheatsheetsIndexHandler(siteRoot))
	mux.HandleFunc("/api/dotfiles/doc", dotfilesDocHandler(siteRoot))
	mux.HandleFunc("/api/dotfiles/search", dotfilesSearchHandler(siteRoot))
	mux.HandleFunc("/dotfiles-index.json", dotfilesIndexHandler(siteRoot))
	mux.HandleFunc("/api/bookmarks/doc", bookmarksDocHandler(siteRoot))
	mux.HandleFunc("/api/bookmarks/search", bookmarksSearchHandler(siteRoot))
	mux.HandleFunc("/bookmarks-index.json", bookmarksIndexHandler(siteRoot))
	mux.HandleFunc("/api/profiles/doc", profilesDocHandler(siteRoot))
	mux.HandleFunc("/api/profiles/search", profilesSearchHandler(siteRoot))
	mux.HandleFunc("/profiles-index.json", profilesIndexHandler(siteRoot))
	mux.HandleFunc("/api/dashboards/doc", dashboardsDocHandler(siteRoot))
	mux.HandleFunc("/api/dashboards/search", dashboardsSearchHandler(siteRoot))
	mux.HandleFunc("/dashboards-index.json", dashboardsIndexHandler(siteRoot))
	mux.HandleFunc("/api/feeds/reddit", feedRedditHandler(siteRoot))
	mux.HandleFunc("/api/feeds/youtube", feedYoutubeHandler(siteRoot))
	mux.HandleFunc("/api/feeds/podcast", feedPodcastHandler(siteRoot))
	mux.HandleFunc("/api/feeds/spotify", feedSpotifyHandler(siteRoot))
	mux.HandleFunc("/api/feeds/news", feedNewsHandler(siteRoot))
	mux.Handle("/", fileServer)
	address := fmt.Sprintf("%s:%d", bindHost, *port)
	url := fmt.Sprintf("http://%s:%d/", displayHosts[0], *port)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		if isAddressInUse(err) {
			printPortInUseMessage(address, displayHosts, *port)
			os.Exit(1)
		}

		log.Fatal(err)
	}
	defer listener.Close()

	fmt.Println()
	fmt.Println(colorize(ansiBold+ansiCyan, "[Rock-OS]"))
	printStartupStatus(siteRoot, bindHost, address)
	printStatus("OK", ansiGreen, "Open %s", url)
	if len(displayHosts) > 1 {
		fmt.Println("Other local URLs:")
		for _, displayHost := range displayHosts[1:] {
			fmt.Printf("  %s\n", colorize(ansiCyan, fmt.Sprintf("http://%s:%d/", displayHost, *port)))
		}
	}
	fmt.Println()

	if *open {
		if err := openBrowser(url); err != nil {
			log.Printf("Could not open browser automatically: %v", err)
		}
	}

	server := &http.Server{
		Handler: logRequests(compressResponses(mux)),
	}

	shutdownErrors := make(chan error, 1)
	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(signals)

		<-signals
		fmt.Println()
		fmt.Println("Shutting down Rock-OS...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		shutdownErrors <- server.Shutdown(ctx)
	}()

	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}

	select {
	case err := <-shutdownErrors:
		if err != nil {
			log.Fatal(err)
		}
	default:
	}
}

func printStartupStatus(siteRoot string, bindHost string, address string) {
	printStatus("OK", ansiGreen, "Serving %s", siteRoot)
	printStatus("OK", ansiGreen, "Listening on %s", address)

	if bindHost == "127.0.0.1" || bindHost == "localhost" {
		printStatus("OK", ansiGreen, "Server Mode: Host")
	} else {
		printStatus("WARN", ansiYellow, "Server Mode: LAN")
	}

	if _, err := os.Stat(filepath.Join(siteRoot, markdownDir)); err == nil {
		if files, err := collectMarkdownFiles(siteRoot); err == nil {
			printStatus("OK", ansiGreen, "Markdown docs indexed on demand: %d", len(files))
		} else {
			printStatus("WARN", ansiYellow, "Markdown docs could not be scanned: %v", err)
		}
	} else {
		printStatus("WARN", ansiYellow, "Markdown folder not found.")
	}

	if _, err := os.Stat(filepath.Join(siteRoot, scriptsDir)); err == nil {
		printStatus("OK", ansiGreen, "Scripts folder mounted.")
	} else {
		printStatus("WARN", ansiYellow, "Scripts folder not found.")
	}

	if _, err := os.Stat(filepath.Join(siteRoot, "media")); err == nil {
		printStatus("OK", ansiGreen, "Media folder mounted.")
	} else {
		printStatus("WARN", ansiYellow, "Media folder not found.")
	}

	if _, err := exec.LookPath("git-crypt"); err == nil {
		printStatus("OK", ansiGreen, "git-crypt installed.")
	} else {
		printStatus("WARN", ansiYellow, "git-crypt not found. Needed only for encrypted Profiles.")
	}

	if gitCryptKeyPresent(siteRoot) {
		printStatus("WARN", ansiYellow, "git-crypt .key file present in repo root. Keep it private and never commit it.")
	} else {
		printStatus("INFO", ansiCyan, "git-crypt .key file in repo root is not present.")
	}

	switch privateMarkdownStatus(siteRoot) {
	case "locked":
		printStatus("INFO", ansiCyan, "Profiles Folder Locked.")
	case "unlocked":
		printStatus("OK", ansiGreen, "Profiles Folder Unlocked.")
	default:
		printStatus("INFO", ansiCyan, "Profiles Folder not found.")
	}

	printStatus("OK", ansiGreen, "Request logging enabled.")
}

func printStatus(level string, color string, format string, args ...any) {
	fmt.Printf("%s %s\n", colorize(color, "["+level+"]"), fmt.Sprintf(format, args...))
}

func colorize(color string, value string) string {
	if os.Getenv("NO_COLOR") != "" {
		return value
	}

	return color + value + ansiReset
}

func privateMarkdownStatus(siteRoot string) string {
	privateRoot := filepath.Join(siteRoot, profilesDir)
	if info, err := os.Stat(privateRoot); err != nil || !info.IsDir() {
		return "missing"
	}

	locked := false
	err := filepath.WalkDir(privateRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		header := make([]byte, 32)
		count, _ := file.Read(header)
		if strings.Contains(string(header[:count]), "GITCRYPT") {
			locked = true
			return filepath.SkipAll
		}

		return nil
	})
	if err != nil {
		return "missing"
	}

	if locked {
		return "locked"
	}

	return "unlocked"
}

func gitCryptKeyPresent(siteRoot string) bool {
	repoRoot := filepath.Dir(siteRoot)
	matches, err := filepath.Glob(filepath.Join(repoRoot, "*.key"))
	if err != nil {
		return false
	}

	return len(matches) > 0
}

func lastCommitTime(siteRoot string) int64 {
	repoRoot := filepath.Dir(siteRoot)
	cmd := exec.Command("git", "log", "-1", "--format=%ct")
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	timestampStr := strings.TrimSpace(string(output))
	sec, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return 0
	}
	return sec
}

func serverStatusHandler(bindHost string, displayHosts []string, port int, siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		mode := "lan"
		description := "Rock-OS is listening on the local network. Trusted devices on this LAN can connect."
		if bindHost == "127.0.0.1" || bindHost == "localhost" {
			mode = "local"
			description = "Rock-OS is listening only on this computer. Other devices on the network cannot connect."
		}

		urls := make([]string, 0, len(displayHosts))
		for _, displayHost := range displayHosts {
			urls = append(urls, fmt.Sprintf("http://%s:%d/", displayHost, port))
		}

		gitCrypt := privateMarkdownStatus(siteRoot)
		markdownCount := 0
		if files, err := collectMarkdownFiles(siteRoot); err == nil {
			markdownCount = len(files)
		}
		scriptsCount := 0
		if scripts, err := collectScripts(siteRoot); err == nil {
			scriptsCount = len(scripts)
		}

		writeJSON(w, serverStatus{
			Mode:         mode,
			Host:         bindHost,
			Description:  description,
			URLs:         urls,
			GitCrypt:     gitCrypt,
			WikiCount:    markdownCount,
			ScriptsCount: scriptsCount,
			Uptime:       int64(time.Since(startupTime).Seconds()),
			LastSync:     lastCommitTime(siteRoot),
		})
	}
}

func linkHealthHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		report, err := scanLinkHealth(siteRoot)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, report)
	}
}

var markdownLinkRegex = regexp.MustCompile(`!?\[([^\]]*)\]\(([^)\r\n]+)\)`)

func scanLinkHealth(siteRoot string) (linkHealthResponse, error) {
	report := linkHealthResponse{
		Items: []linkHealthItem{},
	}

	sourceFiles, err := linkHealthSourceFiles(siteRoot)
	if err != nil {
		return report, err
	}

	for _, source := range sourceFiles {
		content, err := os.ReadFile(filepath.Join(siteRoot, filepath.FromSlash(source)))
		if err != nil {
			report.Skipped++
			report.Items = append(report.Items, linkHealthItem{
				Source: source,
				Status: "skipped",
				Reason: "could not read source file",
			})
			continue
		}

		for _, match := range markdownLinkRegex.FindAllStringSubmatch(string(content), -1) {
			label := strings.TrimSpace(match[1])
			href := cleanMarkdownHref(match[2])
			if href == "" {
				continue
			}

			item := checkLocalLink(siteRoot, source, label, href)
			report.Checked++
			switch item.Status {
			case "ok":
				report.OK++
			case "external":
				report.External++
			case "skipped":
				report.Skipped++
			default:
				report.Broken++
			}
			report.Items = append(report.Items, item)
		}
	}

	return report, nil
}

func linkHealthSourceFiles(siteRoot string) ([]string, error) {
	scanDirs := []string{
		markdownDir,
		guidesDir,
		cheatsheetsDir,
		dotfilesDir,
		bookmarksDir,
		dashboardsDir,
	}
	if privateMarkdownStatus(siteRoot) == "unlocked" {
		scanDirs = append(scanDirs, profilesDir)
	}

	files := []string{}
	for _, dir := range scanDirs {
		root := filepath.Join(siteRoot, dir)
		if _, err := os.Stat(root); os.IsNotExist(err) {
			continue
		}

		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
				return nil
			}

			relativePath, err := filepath.Rel(siteRoot, path)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(relativePath))
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	sort.Strings(files)
	return files, nil
}

func cleanMarkdownHref(rawHref string) string {
	href := strings.TrimSpace(rawHref)
	if strings.HasPrefix(href, "<") && strings.HasSuffix(href, ">") {
		href = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(href, "<"), ">"))
	}

	for _, quote := range []string{` "`, ` '`, "\t\""} {
		if idx := strings.Index(href, quote); idx >= 0 {
			href = strings.TrimSpace(href[:idx])
		}
	}

	return strings.TrimSpace(href)
}

func checkLocalLink(siteRoot string, source string, label string, href string) linkHealthItem {
	item := linkHealthItem{
		Source: source,
		Label:  label,
		Href:   href,
	}

	lowerHref := strings.ToLower(strings.TrimSpace(href))
	switch {
	case strings.HasPrefix(lowerHref, "http://") || strings.HasPrefix(lowerHref, "https://"):
		item.Status = "external"
		item.Reason = "external link not fetched"
		return item
	case strings.HasPrefix(lowerHref, "mailto:") ||
		strings.HasPrefix(lowerHref, "tel:") ||
		strings.HasPrefix(lowerHref, "data:"):
		item.Status = "skipped"
		item.Reason = "non-file link"
		return item
	case strings.HasPrefix(href, "#"):
		item.Status = "ok"
		item.Target = source
		return item
	}

	targetPath, reason := resolveLinkTargetPath(siteRoot, source, href)
	if reason != "" {
		item.Status = "broken"
		item.Reason = reason
		return item
	}

	relativeTarget, err := filepath.Rel(siteRoot, targetPath)
	if err == nil {
		item.Target = filepath.ToSlash(relativeTarget)
	}

	if linkTargetExists(targetPath) {
		item.Status = "ok"
		return item
	}

	item.Status = "broken"
	item.Reason = "target file does not exist"
	return item
}

func resolveLinkTargetPath(siteRoot string, source string, href string) (string, string) {
	href = strings.TrimSpace(href)
	if href == "" {
		return "", "empty link target"
	}

	if parsed, err := url.Parse(href); err == nil {
		href = parsed.Path
	}
	if href == "" {
		return "", ""
	}

	decodedHref, err := url.PathUnescape(href)
	if err == nil {
		href = decodedHref
	}
	href = filepath.FromSlash(strings.TrimPrefix(href, "/"))

	var target string
	if strings.HasPrefix(href, string(os.PathSeparator)) || strings.HasPrefix(href, "/") {
		target = filepath.Join(siteRoot, strings.TrimPrefix(href, string(os.PathSeparator)))
	} else if strings.HasPrefix(strings.TrimSpace(href), "index.html") ||
		strings.HasSuffix(strings.ToLower(href), ".html") {
		target = filepath.Join(siteRoot, href)
	} else if strings.HasPrefix(href, "assets"+string(os.PathSeparator)) ||
		strings.HasPrefix(href, "media"+string(os.PathSeparator)) ||
		strings.HasPrefix(href, "menu"+string(os.PathSeparator)) ||
		strings.HasPrefix(href, "profiles"+string(os.PathSeparator)) ||
		strings.HasPrefix(href, "dashboards"+string(os.PathSeparator)) {
		target = filepath.Join(siteRoot, href)
	} else {
		sourceDir := filepath.Dir(filepath.Join(siteRoot, filepath.FromSlash(source)))
		target = filepath.Join(sourceDir, href)
	}

	cleanSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return "", err.Error()
	}
	cleanTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err.Error()
	}
	if cleanTarget != cleanSiteRoot && !strings.HasPrefix(cleanTarget, cleanSiteRoot+string(os.PathSeparator)) {
		return "", "target escapes Website folder"
	}

	return cleanTarget, ""
}

func linkTargetExists(target string) bool {
	info, err := os.Stat(target)
	if err == nil {
		if info.IsDir() {
			if _, err := os.Stat(filepath.Join(target, "index.html")); err == nil {
				return true
			}
		}
		return true
	}

	if filepath.Ext(target) == "" {
		if _, err := os.Stat(target + ".md"); err == nil {
			return true
		}
		if _, err := os.Stat(filepath.Join(target, "index.html")); err == nil {
			return true
		}
	}

	return false
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

type gzipResponseWriter struct {
	http.ResponseWriter
	writer *gzip.Writer
}

func (writer *loggingResponseWriter) WriteHeader(status int) {
	writer.status = status
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *loggingResponseWriter) Write(data []byte) (int, error) {
	if writer.status == 0 {
		writer.status = http.StatusOK
	}

	return writer.ResponseWriter.Write(data)
}

func (writer *gzipResponseWriter) Write(data []byte) (int, error) {
	return writer.writer.Write(data)
}

func compressResponses(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requestAcceptsGzip(r) || !shouldCompressPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Add("Vary", "Accept-Encoding")
		w.Header().Del("Content-Length")
		w.Header().Set("Content-Encoding", "gzip")

		gzipWriter := gzip.NewWriter(w)
		defer gzipWriter.Close()

		next.ServeHTTP(&gzipResponseWriter{
			ResponseWriter: w,
			writer:         gzipWriter,
		}, r)
	})
}

func requestAcceptsGzip(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
}

func shouldCompressPath(path string) bool {
	if strings.HasPrefix(path, "/api/") ||
		path == "/wiki-index.json" ||
		path == "/guides-index.json" ||
		path == "/cheatsheets-index.json" ||
		path == "/dotfiles-index.json" ||
		path == "/bookmarks-index.json" ||
		path == "/profiles-index.json" ||
		path == "/dashboards-index.json" {
		return true
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case "", ".html", ".css", ".js", ".json", ".md", ".svg", ".txt", ".xml", ".webmanifest":
		return true
	default:
		return false
	}
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		writer := &loggingResponseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(writer, r)

		requestTime := colorize(ansiDim, time.Now().Format("15:04:05"))
		method := colorize(methodLogColor(r.Method), r.Method)
		path := colorize(ansiCyan, r.URL.Path)
		status := colorize(statusLogColor(writer.status), fmt.Sprintf("%d", writer.status))
		duration := colorize(ansiDim, time.Since(start).Round(time.Microsecond).String())

		fmt.Printf("%s %s %s %s %s\n", requestTime, method, path, status, duration)
	})
}

func methodLogColor(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead:
		return ansiGreen
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return ansiBlue
	case http.MethodDelete:
		return ansiRed
	default:
		return ansiCyan
	}
}

func statusLogColor(status int) string {
	switch {
	case status >= 500:
		return ansiRed
	case status >= 400:
		return ansiYellow
	case status >= 300:
		return ansiCyan
	default:
		return ansiGreen
	}
}

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

	normalizedQuery := strings.ToLower(query)
	results := []wikiSearchResult{}

	for _, file := range files {
		title := fileTitle(file.Path)
		searchablePath := strings.ToLower(file.Path)
		searchableTitle := strings.ToLower(title)

		pathMatch := strings.Contains(searchablePath, normalizedQuery) ||
			strings.Contains(searchableTitle, normalizedQuery)

		content, err := os.ReadFile(filepath.Join(siteRoot, filepath.FromSlash(file.Path)))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return nil, err
		}

		text := string(content)
		contentMatch := strings.Contains(strings.ToLower(text), normalizedQuery)
		if !pathMatch && !contentMatch {
			continue
		}

		result := wikiSearchResult{
			Path:  file.Path,
			Title: title,
		}
		if contentMatch {
			result.Snippet = searchSnippet(text, normalizedQuery)
		}

		results = append(results, result)
	}

	return results, nil
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

		trimmed := strings.TrimSpace(line)
		if len(trimmed) <= 120 {
			return trimmed
		}

		matchIndex := strings.Index(strings.ToLower(trimmed), normalizedQuery)
		start := max(0, matchIndex-45)
		end := min(len(trimmed), start+120)
		prefix := ""
		suffix := ""
		if start > 0 {
			prefix = "..."
		}
		if end < len(trimmed) {
			suffix = "..."
		}

		return prefix + trimmed[start:end] + suffix
	}

	return ""
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

func scriptsListHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		scripts, err := collectScripts(siteRoot)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, scripts)
	}
}

func scriptContentHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		script, path, err := resolveScript(siteRoot, r.URL.Query().Get("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		content, err := os.ReadFile(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, map[string]any{
			"script":  script,
			"content": string(content),
		})
	}
}

func scriptsSearchHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		results, err := searchScripts(siteRoot, r.URL.Query().Get("q"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, scriptSearchResponse{Results: results})
	}
}

func scriptRunHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if !scriptRunRequestAllowed(r) {
			http.Error(w, "unauthorized script request", http.StatusForbidden)
			return
		}

		var request scriptRunRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		script, path, err := resolveScript(siteRoot, request.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if !script.Runnable {
			http.Error(w, "script is not runnable on this operating system", http.StatusBadRequest)
			return
		}

		if err := launchScriptTerminal(siteRoot, path); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, map[string]string{"status": "launched"})
	}
}

func scriptRunRequestAllowed(r *http.Request) bool {
	if r.Header.Get("X-Rock-OS-Requested") != "true" {
		return false
	}

	return sameOriginHeaderAllowed(r, "Origin") &&
		sameOriginHeaderAllowed(r, "Referer")
}

func sameOriginHeaderAllowed(r *http.Request, header string) bool {
	value := strings.TrimSpace(r.Header.Get(header))
	if value == "" {
		return true
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return false
	}

	return strings.EqualFold(parsed.Host, r.Host)
}

func collectScripts(siteRoot string) ([]scriptEntry, error) {
	root := filepath.Join(siteRoot, scriptsDir)
	scripts := []scriptEntry{}

	if _, err := os.Stat(root); os.IsNotExist(err) {
		return scripts, nil
	}

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		id := filepath.ToSlash(relativePath)
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !scriptExtensionAllowed(ext) {
			return nil
		}

		scripts = append(scripts, scriptEntry{
			ID:       id,
			Name:     entry.Name(),
			Path:     filepath.ToSlash(filepath.Join(scriptsDir, relativePath)),
			Runnable: scriptRunnableOnCurrentOS(ext),
			Platform: scriptPlatformLabel(ext),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(scripts, func(i, j int) bool {
		return strings.ToLower(scripts[i].ID) < strings.ToLower(scripts[j].ID)
	})

	return scripts, nil
}

func searchScripts(siteRoot string, query string) ([]scriptSearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []scriptSearchResult{}, nil
	}

	normalizedQuery := strings.ToLower(query)
	scripts, err := collectScripts(siteRoot)
	if err != nil {
		return nil, err
	}

	results := []scriptSearchResult{}
	for _, script := range scripts {
		_, path, err := resolveScript(siteRoot, script.ID)
		if err != nil {
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		contentText := string(content)
		nameMatch := strings.Contains(strings.ToLower(script.Name), normalizedQuery)
		pathMatch := strings.Contains(strings.ToLower(script.ID), normalizedQuery) ||
			strings.Contains(strings.ToLower(script.Path), normalizedQuery)
		contentMatch := strings.Contains(strings.ToLower(contentText), normalizedQuery)

		if !nameMatch && !pathMatch && !contentMatch {
			continue
		}

		results = append(results, scriptSearchResult{
			ID:       script.ID,
			Name:     script.Name,
			Path:     script.Path,
			Runnable: script.Runnable,
			Platform: script.Platform,
			Snippet:  searchSnippet(contentText, normalizedQuery),
		})
	}

	return results, nil
}

func resolveScript(siteRoot string, id string) (scriptEntry, string, error) {
	id = filepath.ToSlash(strings.TrimSpace(id))
	if id == "" || strings.Contains(id, "..") || strings.HasPrefix(id, "/") {
		return scriptEntry{}, "", fmt.Errorf("invalid script id")
	}

	path := filepath.Join(siteRoot, scriptsDir, filepath.FromSlash(id))
	root := filepath.Join(siteRoot, scriptsDir)

	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return scriptEntry{}, "", err
	}

	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return scriptEntry{}, "", err
	}

	if cleanPath != cleanRoot && !strings.HasPrefix(cleanPath, cleanRoot+string(os.PathSeparator)) {
		return scriptEntry{}, "", fmt.Errorf("script must live inside %s", scriptsDir)
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return scriptEntry{}, "", err
	}
	if info.IsDir() {
		return scriptEntry{}, "", fmt.Errorf("script id points to a folder")
	}

	ext := strings.ToLower(filepath.Ext(info.Name()))
	if !scriptExtensionAllowed(ext) {
		return scriptEntry{}, "", fmt.Errorf("script type is not allowed")
	}

	return scriptEntry{
		ID:       id,
		Name:     info.Name(),
		Path:     filepath.ToSlash(filepath.Join(scriptsDir, id)),
		Runnable: scriptRunnableOnCurrentOS(ext),
		Platform: scriptPlatformLabel(ext),
	}, cleanPath, nil
}

func scriptExtensionAllowed(ext string) bool {
	switch ext {
	case ".cmd", ".bat", ".ps1", ".sh":
		return true
	default:
		return false
	}
}

func scriptRunnableOnCurrentOS(ext string) bool {
	if ext == ".ps1" {
		_, err := powershellCommand()
		return err == nil
	}

	switch runtime.GOOS {
	case "windows":
		return ext == ".cmd" || ext == ".bat"
	default:
		return ext == ".sh"
	}
}

func scriptPlatformLabel(ext string) string {
	switch ext {
	case ".cmd", ".bat":
		return "Windows"
	case ".ps1":
		return "PowerShell"
	case ".sh":
		return "Linux / macOS"
	default:
		return "Unknown"
	}
}

func launchScriptTerminal(siteRoot string, path string) error {
	commandName, args, err := terminalCommand(siteRoot, path)
	if err != nil {
		return err
	}

	cmd := exec.Command(commandName, args...)
	cmd.Dir = siteRoot

	return cmd.Start()
}

func scriptCommandLine(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))

	if ext == ".ps1" {
		command, err := powershellCommand()
		if err != nil {
			return "", err
		}

		args := []string{
			"-NoProfile",
			"-File",
			path,
		}
		if runtime.GOOS == "windows" {
			args = []string{
				"-NoProfile",
				"-ExecutionPolicy",
				"Bypass",
				"-File",
				path,
			}
		}

		return shellJoin(append([]string{command}, args...)), nil
	}

	switch runtime.GOOS {
	case "windows":
		return shellJoin([]string{"cmd", "/c", path}), nil
	default:
		return shellJoin([]string{"sh", path}), nil
	}
}

func terminalCommand(siteRoot string, path string) (string, []string, error) {
	switch runtime.GOOS {
	case "windows":
		return windowsTerminalCommand(path)
	case "darwin":
		commandLine, err := scriptCommandLine(path)
		if err != nil {
			return "", nil, err
		}
		runLine := fmt.Sprintf(
			"cd %s && %s; printf '\\n[Rock-OS] Script finished. Press Enter to close...'; read _",
			shellQuote(siteRoot),
			commandLine,
		)

		return "osascript", []string{
			"-e",
			fmt.Sprintf(
				`tell application "Terminal" to do script %q`,
				runLine,
			),
			"-e",
			`tell application "Terminal" to activate`,
		}, nil
	default:
		return linuxTerminalCommand(siteRoot, path)
	}
}

func windowsTerminalCommand(path string) (string, []string, error) {
	ext := strings.ToLower(filepath.Ext(path))

	if ext == ".ps1" {
		command, err := powershellCommand()
		if err != nil {
			return "", nil, err
		}

		args := []string{
			"/c",
			"start",
			"Rock-OS Script",
			command,
			"-NoExit",
			"-NoProfile",
			"-ExecutionPolicy",
			"Bypass",
			"-File",
			path,
		}

		return "cmd", args, nil
	}

	return "cmd", []string{"/c", "start", "Rock-OS Script", "cmd", "/k", windowsQuote(path)}, nil
}

func linuxTerminalCommand(siteRoot string, path string) (string, []string, error) {
	commandLine, err := scriptCommandLine(path)
	if err != nil {
		return "", nil, err
	}

	runLine := fmt.Sprintf(
		"cd %s && %s; printf '\\n[Rock-OS] Script finished. Press Enter to close...'; read _",
		shellQuote(siteRoot),
		commandLine,
	)

	candidates := []struct {
		name string
		args []string
	}{
		{"x-terminal-emulator", []string{"-e", "sh", "-c", runLine}},
		{"gnome-terminal", []string{"--", "sh", "-c", runLine}},
		{"konsole", []string{"-e", "sh", "-c", runLine}},
		{"xfce4-terminal", []string{"--command", "sh -c " + shellQuote(runLine)}},
		{"mate-terminal", []string{"--command", "sh -c " + shellQuote(runLine)}},
		{"alacritty", []string{"-e", "sh", "-c", runLine}},
		{"kitty", []string{"sh", "-c", runLine}},
		{"wezterm", []string{"start", "--", "sh", "-c", runLine}},
		{"xterm", []string{"-e", "sh", "-c", runLine}},
	}

	for _, candidate := range candidates {
		command, err := exec.LookPath(candidate.name)
		if err == nil {
			return command, candidate.args, nil
		}
	}

	return "", nil, fmt.Errorf("no supported terminal emulator was found")
}

func shellJoin(parts []string) string {
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		quoted = append(quoted, shellQuote(part))
	}

	return strings.Join(quoted, " ")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func windowsQuote(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}

func powershellCommand() (string, error) {
	if path, err := exec.LookPath("pwsh"); err == nil {
		return path, nil
	}

	if runtime.GOOS == "windows" {
		if path, err := exec.LookPath("powershell"); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("PowerShell was not found. Install PowerShell 7+ to run .ps1 scripts")
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func resolveSiteRoot(workingDir string, requestedSiteRoot string) (string, error) {
	if strings.TrimSpace(requestedSiteRoot) != "" {
		return cleanSiteRoot(requestedSiteRoot)
	}

	candidates := []string{
		workingDir,
		filepath.Join(workingDir, "Website"),
		filepath.Join(workingDir, "..", "Website"),
		filepath.Join(workingDir, "..", "..", "Website"),
	}

	seen := map[string]bool{}
	for _, candidate := range candidates {
		absoluteCandidate, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}

		if seen[absoluteCandidate] {
			continue
		}
		seen[absoluteCandidate] = true

		if siteRootLooksValid(absoluteCandidate) {
			return absoluteCandidate, nil
		}
	}

	return "", fmt.Errorf("could not find the Website folder; pass --site-root with the path to Website")
}

func cleanSiteRoot(siteRoot string) (string, error) {
	absoluteSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return "", err
	}

	if !siteRootLooksValid(absoluteSiteRoot) {
		return "", fmt.Errorf("%s does not look like the Rock-OS Website folder", absoluteSiteRoot)
	}

	return absoluteSiteRoot, nil
}

func siteRootLooksValid(siteRoot string) bool {
	requiredFiles := []string{
		"index.html",
		"wiki.html",
		"guides.html",
		"cheatsheets.html",
		"dotfiles.html",
		"bookmarks.html",
		"scripts.html",
		"profiles.html",
		"dashboards.html",
	}

	for _, file := range requiredFiles {
		info, err := os.Stat(filepath.Join(siteRoot, file))
		if err != nil || info.IsDir() {
			return false
		}
	}

	requiredDirs := []string{
		markdownDir,
		guidesDir,
		cheatsheetsDir,
		dotfilesDir,
		bookmarksDir,
		scriptsDir,
		profilesDir,
		dashboardsDir,
		"css",
		"js",
	}

	for _, dir := range requiredDirs {
		info, err := os.Stat(filepath.Join(siteRoot, dir))
		if err != nil || !info.IsDir() {
			return false
		}
	}

	return true
}

func resolveHost(host string) (string, []string, error) {
	switch strings.ToLower(strings.TrimSpace(host)) {
	case "", "local", "lan", "all", "0.0.0.0":
		localIPs, err := localIPv4s()
		if err != nil {
			return "", nil, err
		}

		return "0.0.0.0", localIPs, nil
	default:
		return host, []string{host}, nil
	}
}

type ipCandidate struct {
	ip    string
	score int
}

func localIPv4s() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	candidates := []ipCandidate{}
	for _, networkInterface := range interfaces {
		if networkInterface.Flags&net.FlagUp == 0 ||
			networkInterface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addresses, err := networkInterface.Addrs()
		if err != nil {
			continue
		}

		for _, address := range addresses {
			ip, _, err := net.ParseCIDR(address.String())
			if err != nil {
				continue
			}

			ip = ip.To4()
			if ip == nil || ip.IsLoopback() {
				continue
			}

			if ip.IsUnspecified() ||
				ip.IsMulticast() ||
				ip.IsLinkLocalUnicast() ||
				!ip.IsPrivate() {
				continue
			}

			candidates = append(candidates, ipCandidate{
				ip:    ip.String(),
				score: localIPScore(ip, networkInterface),
			})
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("could not find a private local IPv4 address")
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	seen := map[string]bool{}
	ips := []string{}
	for _, candidate := range candidates {
		if seen[candidate.ip] {
			continue
		}

		seen[candidate.ip] = true
		ips = append(ips, candidate.ip)
	}

	return ips, nil
}

func localIPScore(ip net.IP, networkInterface net.Interface) int {
	score := 0
	ip4 := ip.To4()

	switch {
	case ip4[0] == 192 && ip4[1] == 168:
		score += 300
	case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
		score += 200
	case ip4[0] == 10:
		score += 100
	}

	interfaceName := strings.ToLower(networkInterface.Name)
	virtualHints := []string{
		"bluetooth",
		"br-",
		"bridge",
		"docker",
		"hyper-v",
		"tailscale",
		"tap",
		"tun",
		"vbox",
		"vether",
		"virtual",
		"vmware",
		"vpn",
		"wsl",
		"zerotier",
	}
	for _, hint := range virtualHints {
		if strings.Contains(interfaceName, hint) {
			score -= 1000
		}
	}

	if networkInterface.Flags&net.FlagPointToPoint != 0 {
		score -= 500
	}

	return score
}

func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func isAddressInUse(err error) bool {
	message := strings.ToLower(err.Error())

	return strings.Contains(message, "address already in use") ||
		strings.Contains(message, "only one usage of each socket address")
}

func printPortInUseMessage(address string, displayHosts []string, port int) {
	fmt.Println()
	fmt.Println("[Rock-OS]")
	fmt.Printf("Could not listen on %s because port %d is already in use.\n", address, port)
	fmt.Println()
	fmt.Println("Rock-OS may already be running. Try opening:")
	for _, displayHost := range displayHosts {
		fmt.Printf("  http://%s:%d/\n", displayHost, port)
	}
	fmt.Println()
	fmt.Printf("If another app is using port %d, stop it or start Rock-OS on another port:\n", port)
	fmt.Printf("  go run . --port %d\n", port+1)
	fmt.Println()
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
	return collectMarkdownFilesWithCache(siteRoot, markdownDir, globalMarkdownIndexCache)
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

func openBrowser(url string) error {
	var command string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		command = "open"
		args = []string{url}
	case "linux":
		command = "xdg-open"
		args = []string{url}
	case "windows":
		command = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return exec.Command(command, args...).Start()
}

func collectGuideFiles(siteRoot string) ([]markdownIndexEntry, error) {
	return collectGuideFilesWithCache(siteRoot, globalGuidesIndexCache)
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

	normalizedQuery := strings.ToLower(query)
	results := []wikiSearchResult{}

	for _, file := range files {
		title := fileTitle(file.Path)
		searchablePath := strings.ToLower(file.Path)
		searchableTitle := strings.ToLower(title)

		pathMatch := strings.Contains(searchablePath, normalizedQuery) ||
			strings.Contains(searchableTitle, normalizedQuery)

		content, err := os.ReadFile(filepath.Join(siteRoot, filepath.FromSlash(file.Path)))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return nil, err
		}

		text := string(content)
		contentMatch := strings.Contains(strings.ToLower(text), normalizedQuery)
		if !pathMatch && !contentMatch {
			continue
		}

		result := wikiSearchResult{
			Path:  file.Path,
			Title: title,
		}
		if contentMatch {
			result.Snippet = searchSnippet(text, normalizedQuery)
		}

		results = append(results, result)
	}

	return results, nil
}

func collectCheatsheetFiles(siteRoot string) ([]markdownIndexEntry, error) {
	return collectCheatsheetFilesWithCache(siteRoot, globalCheatsheetsIndexCache)
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

	normalizedQuery := strings.ToLower(query)
	results := []wikiSearchResult{}

	for _, file := range files {
		title := fileTitle(file.Path)
		searchablePath := strings.ToLower(file.Path)
		searchableTitle := strings.ToLower(title)

		pathMatch := strings.Contains(searchablePath, normalizedQuery) ||
			strings.Contains(searchableTitle, normalizedQuery)

		content, err := os.ReadFile(filepath.Join(siteRoot, filepath.FromSlash(file.Path)))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return nil, err
		}

		text := string(content)
		contentMatch := strings.Contains(strings.ToLower(text), normalizedQuery)
		if !pathMatch && !contentMatch {
			continue
		}

		result := wikiSearchResult{
			Path:  file.Path,
			Title: title,
		}
		if contentMatch {
			result.Snippet = searchSnippet(text, normalizedQuery)
		}

		results = append(results, result)
	}

	return results, nil
}

func collectDotfileFiles(siteRoot string) ([]markdownIndexEntry, error) {
	return collectDotfileFilesWithCache(siteRoot, globalDotfilesIndexCache)
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

	normalizedQuery := strings.ToLower(query)
	results := []wikiSearchResult{}

	for _, file := range files {
		title := fileTitle(file.Path)
		searchablePath := strings.ToLower(file.Path)
		searchableTitle := strings.ToLower(title)

		pathMatch := strings.Contains(searchablePath, normalizedQuery) ||
			strings.Contains(searchableTitle, normalizedQuery)

		content, err := os.ReadFile(filepath.Join(siteRoot, filepath.FromSlash(file.Path)))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return nil, err
		}

		text := string(content)
		contentMatch := strings.Contains(strings.ToLower(text), normalizedQuery)
		if !pathMatch && !contentMatch {
			continue
		}

		result := wikiSearchResult{
			Path:  file.Path,
			Title: title,
		}
		if contentMatch {
			result.Snippet = searchSnippet(text, normalizedQuery)
		}

		results = append(results, result)
	}

	return results, nil
}

func collectBookmarkFiles(siteRoot string) ([]markdownIndexEntry, error) {
	return collectBookmarkFilesWithCache(siteRoot, globalBookmarksIndexCache)
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

	normalizedQuery := strings.ToLower(query)
	results := []wikiSearchResult{}

	for _, file := range files {
		title := fileTitle(file.Path)
		searchablePath := strings.ToLower(file.Path)
		searchableTitle := strings.ToLower(title)

		pathMatch := strings.Contains(searchablePath, normalizedQuery) ||
			strings.Contains(searchableTitle, normalizedQuery)

		content, err := os.ReadFile(filepath.Join(siteRoot, filepath.FromSlash(file.Path)))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return nil, err
		}

		text := string(content)
		contentMatch := strings.Contains(strings.ToLower(text), normalizedQuery)
		if !pathMatch && !contentMatch {
			continue
		}

		result := wikiSearchResult{
			Path:  file.Path,
			Title: title,
		}
		if contentMatch {
			result.Snippet = searchSnippet(text, normalizedQuery)
		}

		results = append(results, result)
	}

	return results, nil
}

func profilesIndexHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if privateMarkdownStatus(siteRoot) == "locked" {
			http.Error(w, "profiles are locked", http.StatusLocked)
			return
		}

		files, err := collectProfilesFiles(siteRoot)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		files = filterProfilesFiles(files, r.URL.Query().Get("profile"))

		writeJSON(w, files)
	}
}

func profilesDocHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if privateMarkdownStatus(siteRoot) == "locked" {
			http.Error(w, "profiles are locked", http.StatusLocked)
			return
		}

		docPath, path, err := resolveProfilesDoc(siteRoot, r.URL.Query().Get("path"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		content, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "profiles document not found", http.StatusNotFound)
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

func resolveProfilesDoc(siteRoot string, docPath string) (string, string, error) {
	normalized := filepath.ToSlash(
		filepath.Clean(
			strings.ReplaceAll(docPath, "\\", "/"),
		),
	)
	normalized = strings.TrimPrefix(normalized, "/")

	if normalized == "." || normalized == "" || strings.Contains(normalized, "\x00") {
		return "", "", fmt.Errorf("profiles document path is required")
	}

	if !strings.HasPrefix(normalized, profilesDir+"/") {
		return "", "", fmt.Errorf("profiles document path must start with %s/", profilesDir)
	}

	if !strings.EqualFold(filepath.Ext(normalized), ".md") {
		return "", "", fmt.Errorf("profiles document must be a .md file")
	}

	profilesRoot, err := filepath.Abs(filepath.Join(siteRoot, profilesDir))
	if err != nil {
		return "", "", err
	}

	target, err := filepath.Abs(filepath.Join(siteRoot, filepath.FromSlash(normalized)))
	if err != nil {
		return "", "", err
	}

	relativeTarget, err := filepath.Rel(profilesRoot, target)
	if err != nil {
		return "", "", err
	}

	if strings.HasPrefix(relativeTarget, "..") || relativeTarget == "." {
		return "", "", fmt.Errorf("profiles document must stay inside %s", profilesDir)
	}

	return normalized, target, nil
}

func profilesSearchHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if privateMarkdownStatus(siteRoot) == "locked" {
			http.Error(w, "profiles are locked", http.StatusLocked)
			return
		}

		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if query == "" {
			writeJSON(w, wikiSearchResponse{
				Results: []wikiSearchResult{},
			})
			return
		}

		results, err := searchProfiles(siteRoot, query, r.URL.Query().Get("profile"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, wikiSearchResponse{
			Results: results,
		})
	}
}

func searchProfiles(siteRoot string, query string, profile string) ([]wikiSearchResult, error) {
	files, err := collectProfilesFiles(siteRoot)
	if err != nil {
		return nil, err
	}
	files = filterProfilesFiles(files, profile)

	normalizedQuery := strings.ToLower(query)
	results := []wikiSearchResult{}

	for _, file := range files {
		title := fileTitle(file.Path)
		searchablePath := strings.ToLower(file.Path)
		searchableTitle := strings.ToLower(title)

		pathMatch := strings.Contains(searchablePath, normalizedQuery) ||
			strings.Contains(searchableTitle, normalizedQuery)

		content, err := os.ReadFile(filepath.Join(siteRoot, filepath.FromSlash(file.Path)))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return nil, err
		}

		text := string(content)
		contentMatch := strings.Contains(strings.ToLower(text), normalizedQuery)
		if !pathMatch && !contentMatch {
			continue
		}

		result := wikiSearchResult{
			Path:  file.Path,
			Title: title,
		}
		if contentMatch {
			result.Snippet = searchSnippet(text, normalizedQuery)
		}

		results = append(results, result)
	}

	return results, nil
}

func filterProfilesFiles(files []markdownIndexEntry, profile string) []markdownIndexEntry {
	profile = strings.Trim(strings.ReplaceAll(profile, "\\", "/"), "/")
	if profile == "" || strings.Contains(profile, "/") || strings.Contains(profile, "\x00") {
		if profile == "" {
			return files
		}
		return []markdownIndexEntry{}
	}

	prefix := profilesDir + "/" + profile + "/"
	filtered := []markdownIndexEntry{}
	for _, file := range files {
		if strings.HasPrefix(file.Path, prefix) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

func writeProfilesIndex(siteRoot string) (bool, error) {
	files, err := collectProfilesFiles(siteRoot)
	if err != nil {
		return false, err
	}

	nextJSON, err := json.MarshalIndent(files, "", "  ")
	if err != nil {
		return false, err
	}

	nextJSON = append(nextJSON, '\n')

	indexPath := filepath.Join(siteRoot, profilesIndexFile)
	previousJSON, err := os.ReadFile(indexPath)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	if bytes.Equal(previousJSON, nextJSON) {
		return false, nil
	}

	return true, os.WriteFile(indexPath, nextJSON, 0o644)
}

func collectProfilesFiles(siteRoot string) ([]markdownIndexEntry, error) {
	return collectMarkdownFilesWithCache(siteRoot, profilesDir, globalProfilesIndexCache)
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

	normalizedQuery := strings.ToLower(query)
	results := []wikiSearchResult{}

	for _, file := range files {
		title := fileTitle(file.Path)
		searchablePath := strings.ToLower(file.Path)
		searchableTitle := strings.ToLower(title)

		pathMatch := strings.Contains(searchablePath, normalizedQuery) ||
			strings.Contains(searchableTitle, normalizedQuery)

		content, err := os.ReadFile(filepath.Join(siteRoot, filepath.FromSlash(file.Path)))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return nil, err
		}

		text := string(content)
		contentMatch := strings.Contains(strings.ToLower(text), normalizedQuery)
		if !pathMatch && !contentMatch {
			continue
		}

		result := wikiSearchResult{
			Path:  file.Path,
			Title: title,
		}
		if contentMatch {
			result.Snippet = searchSnippet(text, normalizedQuery)
		}

		results = append(results, result)
	}

	return results, nil
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
	return collectMarkdownFilesWithCache(siteRoot, dashboardsDir, globalDashboardsIndexCache)
}

type feedItem struct {
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Created     string    `json:"created"`
	Author      string    `json:"author,omitempty"`
	Source      string    `json:"source,omitempty"`
	Thumbnail   string    `json:"thumbnail"`
	PublishTime time.Time `json:"-"`
}

var blockedServerFetchPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("::/128"),
	netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:2::/48"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("ff00::/8"),
}

func newPublicFetchClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: timeout}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}

			ips, err := resolvePublicFetchHost(ctx, host)
			if err != nil {
				return nil, err
			}
			if len(ips) == 0 {
				return nil, fmt.Errorf("could not resolve feed host %s", host)
			}

			return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
		},
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return http.ErrUseLastResponse
			}
			return validatePublicFetchURL(req.URL.String())
		},
	}
}

func clampFeedLimit(limit int) int {
	if limit <= 0 {
		return defaultFeedLimit
	}
	if limit > maxFeedLimit {
		return maxFeedLimit
	}
	return limit
}

func remoteResponseBodyReader(resp *http.Response) (io.Reader, error) {
	if resp.ContentLength > maxRemoteFeedResponseSize {
		return nil, fmt.Errorf("remote response too large: %d bytes", resp.ContentLength)
	}
	return io.LimitReader(resp.Body, maxRemoteFeedResponseSize), nil
}

func readRemoteResponseBody(resp *http.Response) ([]byte, error) {
	if resp.ContentLength > maxRemoteFeedResponseSize {
		return nil, fmt.Errorf("remote response too large: %d bytes", resp.ContentLength)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxRemoteFeedResponseSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxRemoteFeedResponseSize {
		return nil, fmt.Errorf("remote response exceeded %d bytes", maxRemoteFeedResponseSize)
	}
	return body, nil
}

func validatePublicFetchURL(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("url is required")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("invalid feed URL: %s", rawURL)
	}
	if parsed.User != nil {
		return fmt.Errorf("feed URL userinfo is not allowed")
	}

	_, err = resolvePublicFetchHost(context.Background(), parsed.Hostname())
	return err
}

func resolvePublicFetchHost(ctx context.Context, host string) ([]net.IP, error) {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	if host == "" {
		return nil, fmt.Errorf("feed URL host is required")
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return nil, fmt.Errorf("access to local network is restricted: %s", host)
	}

	if addr, err := netip.ParseAddr(host); err == nil {
		if err := validatePublicFetchAddr(addr, host); err != nil {
			return nil, err
		}
		return []net.IP{net.ParseIP(host)}, nil
	}

	resolved, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("could not resolve feed host %s: %w", host, err)
	}
	if len(resolved) == 0 {
		return nil, fmt.Errorf("could not resolve feed host %s", host)
	}

	ips := make([]net.IP, 0, len(resolved))
	for _, resolvedIP := range resolved {
		addr, ok := netip.AddrFromSlice(resolvedIP.IP)
		if !ok {
			return nil, fmt.Errorf("could not parse resolved IP for %s", host)
		}
		if err := validatePublicFetchAddr(addr, host); err != nil {
			return nil, err
		}
		ips = append(ips, resolvedIP.IP)
	}

	return ips, nil
}

func validatePublicFetchAddr(addr netip.Addr, host string) error {
	addr = addr.Unmap()
	for _, prefix := range blockedServerFetchPrefixes {
		if prefix.Contains(addr) {
			return fmt.Errorf("access to local network is restricted: %s", host)
		}
	}
	return nil
}

type redditChild struct {
	Data struct {
		Title      string  `json:"title"`
		Permalink  string  `json:"permalink"`
		CreatedUTC float64 `json:"created_utc"`
		Author     string  `json:"author"`
		Thumbnail  string  `json:"thumbnail"`
	} `json:"data"`
}

type redditResponse struct {
	Data struct {
		Children []redditChild `json:"children"`
	} `json:"data"`
}

type ytFeed struct {
	XMLName xml.Name  `xml:"feed"`
	Entries []ytEntry `xml:"entry"`
}

type ytEntry struct {
	Title     string `xml:"title"`
	VideoID   string `xml:"videoId"`
	Published string `xml:"published"`
	Link      ytLink `xml:"link"`
}

type ytLink struct {
	Href string `xml:"href,attr"`
}

func feedRedditHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		subreddit := r.URL.Query().Get("subreddit")
		redditURL := r.URL.Query().Get("url")

		if redditURL != "" {
			if u, err := url.Parse(redditURL); err == nil {
				path := strings.Trim(u.Path, "/")
				parts := strings.Split(path, "/")
				if len(parts) >= 2 && parts[0] == "r" {
					subreddit = parts[1]
				}
			}
		}

		if subreddit == "" {
			http.Error(w, "subreddit or url parameter is required", http.StatusBadRequest)
			return
		}

		// Sanitize subreddit: alphanumeric and underscores only, max 50 chars
		for _, char := range subreddit {
			if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_') {
				http.Error(w, "invalid subreddit parameter", http.StatusBadRequest)
				return
			}
		}
		if len(subreddit) > 50 {
			http.Error(w, "subreddit parameter is too long", http.StatusBadRequest)
			return
		}

		cacheDir := filepath.Join(siteRoot, ".gocache", "feeds")
		cachePath := filepath.Join(cacheDir, fmt.Sprintf("reddit_%s.json", subreddit))

		// Try to fetch live feed
		items, err := fetchLiveRedditFeed(subreddit)
		if err == nil {
			// Save to cache
			_ = os.MkdirAll(cacheDir, 0755)
			if data, err := json.Marshal(items); err == nil {
				_ = os.WriteFile(cachePath, data, 0644)
			}
			writeJSON(w, items)
			return
		}

		// Fallback to cache
		if data, err := os.ReadFile(cachePath); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(data)
			return
		}

		// No cache and offline/failed - return empty list so frontend fallback can load
		writeJSON(w, []feedItem{})
	}
}

func fetchLiveRedditFeed(subreddit string) ([]feedItem, error) {
	urlStr := fmt.Sprintf("https://www.reddit.com/r/%s/new.json?limit=5", subreddit)
	client := newPublicFetchClient(5 * time.Second)
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Rock-OS/1.0.0 (by rocketpowerinc)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reddit API returned HTTP %d", resp.StatusCode)
	}

	body, err := remoteResponseBodyReader(resp)
	if err != nil {
		return nil, err
	}

	var redditRes redditResponse
	if err := json.NewDecoder(body).Decode(&redditRes); err != nil {
		return nil, err
	}

	items := make([]feedItem, 0, len(redditRes.Data.Children))
	for _, child := range redditRes.Data.Children {
		createdStr := ""
		if child.Data.CreatedUTC > 0 {
			t := time.Unix(int64(child.Data.CreatedUTC), 0)
			createdStr = t.Format("2006-01-02")
		}

		thumb := child.Data.Thumbnail
		if thumb == "" || !strings.HasPrefix(thumb, "http") {
			thumb = ""
		}

		items = append(items, feedItem{
			Title:     child.Data.Title,
			URL:       "https://www.reddit.com" + child.Data.Permalink,
			Created:   createdStr,
			Author:    "u/" + child.Data.Author,
			Thumbnail: thumb,
		})
	}

	return items, nil
}

func feedYoutubeHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		channelIDs := r.URL.Query()["channel_id"]
		playlistIDs := r.URL.Query()["playlist_id"]
		urls := r.URL.Query()["url"]
		if len(urls)+len(channelIDs)+len(playlistIDs) > maxFeedURLParams {
			http.Error(w, "too many feed parameters", http.StatusBadRequest)
			return
		}

		for _, rawURL := range urls {
			t, val := resolveYoutubeURLToID(rawURL, siteRoot)
			if t == "channel_id" {
				channelIDs = append(channelIDs, val)
			} else if t == "playlist_id" {
				playlistIDs = append(playlistIDs, val)
			}
		}

		limitStr := r.URL.Query().Get("limit")

		if len(channelIDs) == 0 && len(playlistIDs) == 0 {
			http.Error(w, "at least one channel_id, playlist_id, or url parameter is required", http.StatusBadRequest)
			return
		}

		limit := defaultFeedLimit
		if limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
				limit = l
			}
		}
		limit = clampFeedLimit(limit)

		// Sanitize all IDs
		for _, id := range channelIDs {
			if !isValidFeedID(id) {
				http.Error(w, "invalid channel_id parameter", http.StatusBadRequest)
				return
			}
		}
		for _, id := range playlistIDs {
			if !isValidFeedID(id) {
				http.Error(w, "invalid playlist_id parameter", http.StatusBadRequest)
				return
			}
		}

		// Cache path based on hash of the sorted IDs to ensure consistent caching
		sortedIDs := make([]string, len(channelIDs)+len(playlistIDs))
		copy(sortedIDs, channelIDs)
		copy(sortedIDs[len(channelIDs):], playlistIDs)
		sort.Strings(sortedIDs)

		hasher := sha256.New()
		for _, id := range sortedIDs {
			hasher.Write([]byte(id))
		}
		cacheFilename := fmt.Sprintf("youtube_%x_l%d.json", hasher.Sum(nil), limit)
		cacheDir := filepath.Join(siteRoot, ".gocache", "feeds")
		cachePath := filepath.Join(cacheDir, cacheFilename)

		// Try to fetch live feed
		items, err := fetchCombinedYoutubeFeed(channelIDs, playlistIDs, limit)
		if err == nil {
			// Save to cache
			_ = os.MkdirAll(cacheDir, 0755)
			if data, err := json.Marshal(items); err == nil {
				_ = os.WriteFile(cachePath, data, 0644)
			}
			writeJSON(w, items)
			return
		}

		// Fallback to cache
		if data, err := os.ReadFile(cachePath); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(data)
			return
		}

		// Return empty list on failure
		writeJSON(w, []feedItem{})
	}
}

func isValidFeedID(id string) bool {
	if len(id) == 0 || len(id) > 100 {
		return false
	}
	for _, char := range id {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_' || char == '-') {
			return false
		}
	}
	return true
}

func fetchCombinedYoutubeFeed(channelIDs []string, playlistIDs []string, limit int) ([]feedItem, error) {
	type result struct {
		items []feedItem
		err   error
	}

	totalFeeds := len(channelIDs) + len(playlistIDs)
	ch := make(chan result, totalFeeds)

	for _, id := range channelIDs {
		go func(id string) {
			urlStr := fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?channel_id=%s", id)
			items, err := fetchSingleYoutubeFeed(urlStr)
			ch <- result{items, err}
		}(id)
	}

	for _, id := range playlistIDs {
		go func(id string) {
			urlStr := fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?playlist_id=%s", id)
			items, err := fetchSingleYoutubeFeed(urlStr)
			ch <- result{items, err}
		}(id)
	}

	var combined []feedItem
	var firstErr error

	for i := 0; i < totalFeeds; i++ {
		res := <-ch
		if res.err != nil {
			if firstErr == nil {
				firstErr = res.err
			}
		} else {
			combined = append(combined, res.items...)
		}
	}

	if len(combined) == 0 && firstErr != nil {
		return nil, firstErr
	}

	// Sort combined feed items by PublishTime descending (newest first)
	sort.Slice(combined, func(i, j int) bool {
		return combined[i].PublishTime.After(combined[j].PublishTime)
	})

	// Slice to limit
	if len(combined) > limit {
		combined = combined[:limit]
	}

	return combined, nil
}

func fetchSingleYoutubeFeed(urlStr string) ([]feedItem, error) {
	client := newPublicFetchClient(5 * time.Second)
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Rock-OS/1.0.0 (by rocketpowerinc)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("youtube API returned HTTP %d", resp.StatusCode)
	}

	body, err := remoteResponseBodyReader(resp)
	if err != nil {
		return nil, err
	}

	var yt ytFeed
	if err := xml.NewDecoder(body).Decode(&yt); err != nil {
		return nil, err
	}

	items := make([]feedItem, 0, len(yt.Entries))
	for _, entry := range yt.Entries {
		var pubTime time.Time
		dateStr := ""
		if entry.Published != "" {
			if t, err := time.Parse(time.RFC3339, entry.Published); err == nil {
				pubTime = t
				dateStr = t.Format("2006-01-02")
			} else {
				dateStr = entry.Published
			}
		}

		videoID := entry.VideoID
		if videoID == "" && entry.Link.Href != "" {
			videoID = extractYoutubeVideoID(entry.Link.Href)
		}

		thumb := ""
		if videoID != "" {
			thumb = fmt.Sprintf("https://i.ytimg.com/vi/%s/mqdefault.jpg", videoID)
		}

		linkURL := entry.Link.Href
		if linkURL == "" && videoID != "" {
			linkURL = fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
		}

		items = append(items, feedItem{
			Title:       entry.Title,
			URL:         linkURL,
			Created:     dateStr,
			Thumbnail:   thumb,
			PublishTime: pubTime,
		})
	}

	return items, nil
}

func extractYoutubeVideoID(urlStr string) string {
	if strings.Contains(urlStr, "v=") {
		parts := strings.Split(urlStr, "v=")
		if len(parts) > 1 {
			subparts := strings.Split(parts[1], "&")
			return subparts[0]
		}
	}
	if strings.Contains(urlStr, "youtu.be/") {
		parts := strings.Split(urlStr, "youtu.be/")
		if len(parts) > 1 {
			subparts := strings.Split(parts[1], "?")
			return subparts[0]
		}
	}
	if strings.Contains(urlStr, "embed/") {
		parts := strings.Split(urlStr, "embed/")
		if len(parts) > 1 {
			subparts := strings.Split(parts[1], "?")
			return subparts[0]
		}
	}
	return ""
}

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string      `xml:"title"`
	Image       rssImage    `xml:"image"`
	ItunesImage itunesImage `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd image"`
	Items       []rssItem   `xml:"item"`
}

type rssImage struct {
	URL string `xml:"url"`
}

type itunesImage struct {
	Href string `xml:"href,attr"`
}

type rssItem struct {
	Title          string       `xml:"title"`
	Link           string       `xml:"link"`
	PubDate        string       `xml:"pubDate"`
	Description    string       `xml:"description"`
	Image          rssImage     `xml:"image"`
	Enclosure      rssEnclosure `xml:"enclosure"`
	MediaThumbnail mediaImage   `xml:"http://search.yahoo.com/mrss/ thumbnail"`
	MediaContent   mediaImage   `xml:"http://search.yahoo.com/mrss/ content"`
}

type rssEnclosure struct {
	URL  string `xml:"url,attr"`
	Type string `xml:"type,attr"`
}

type mediaImage struct {
	URL    string `xml:"url,attr"`
	Medium string `xml:"medium,attr"`
	Type   string `xml:"type,attr"`
}

func feedPodcastHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		feedURL := r.URL.Query().Get("url")
		if feedURL == "" {
			http.Error(w, "url parameter is required", http.StatusBadRequest)
			return
		}

		feedURL = resolvePodcastURLToFeed(feedURL, siteRoot)
		if err := validatePublicFetchURL(feedURL); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		// Cache path based on hash of the feed URL
		hasher := sha256.New()
		hasher.Write([]byte(feedURL))
		cacheFilename := fmt.Sprintf("podcast_%x.json", hasher.Sum(nil))
		cacheDir := filepath.Join(siteRoot, ".gocache", "feeds")
		cachePath := filepath.Join(cacheDir, cacheFilename)

		// Try to fetch live feed
		items, err := fetchLivePodcastFeed(feedURL)
		if err == nil {
			// Save to cache
			_ = os.MkdirAll(cacheDir, 0755)
			if data, err := json.Marshal(items); err == nil {
				_ = os.WriteFile(cachePath, data, 0644)
			}
			writeJSON(w, items)
			return
		}

		// Fallback to cache
		if data, err := os.ReadFile(cachePath); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(data)
			return
		}

		// Return empty list on failure
		writeJSON(w, []feedItem{})
	}
}

func fetchLivePodcastFeed(feedURL string) ([]feedItem, error) {
	client := newPublicFetchClient(8 * time.Second)
	req, err := http.NewRequest("GET", feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Rock-OS/1.0.0 (by rocketpowerinc)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("podcast RSS returned HTTP %d", resp.StatusCode)
	}

	body, err := remoteResponseBodyReader(resp)
	if err != nil {
		return nil, err
	}

	var rss rssFeed
	if err := xml.NewDecoder(body).Decode(&rss); err != nil {
		return nil, err
	}

	channelThumb := rss.Channel.Image.URL
	if channelThumb == "" {
		channelThumb = rss.Channel.ItunesImage.Href
	}

	limit := len(rss.Channel.Items)
	if limit > 5 {
		limit = 5
	}

	items := make([]feedItem, 0, limit)
	for i := 0; i < limit; i++ {
		item := rss.Channel.Items[i]
		dateStr := ""
		var pubTime time.Time
		if item.PubDate != "" {
			pubTime = parseRssDate(item.PubDate)
			if !pubTime.IsZero() {
				dateStr = pubTime.Format("2006-01-02")
			} else {
				dateStr = item.PubDate
			}
		}

		items = append(items, feedItem{
			Title:       item.Title,
			URL:         item.Link,
			Created:     dateStr,
			Thumbnail:   channelThumb,
			PublishTime: pubTime,
		})
	}

	return items, nil
}

func parseRssDate(dateStr string) time.Time {
	dateStr = strings.TrimSpace(dateStr)
	// Try RFC1123
	t, err := time.Parse(time.RFC1123, dateStr)
	if err == nil {
		return t
	}
	// Try RFC1123Z
	t, err = time.Parse(time.RFC1123Z, dateStr)
	if err == nil {
		return t
	}
	// Try common layouts
	formats := []string{
		"Mon, _2 Jan 2006 15:04:05 MST",
		"Mon, _2 Jan 2006 15:04:05 -0700",
		"Mon, _2 Jan 2006 15:04:05 Z",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, format := range formats {
		t, err = time.Parse(format, dateStr)
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

var (
	channelIDRegex  = regexp.MustCompile(`/channel/(UC[a-zA-Z0-9_-]{22})`)
	itemPropRegex   = regexp.MustCompile(`<meta itemprop="channelId" content="(UC[a-zA-Z0-9_-]{22})">`)
	jsonChanIDRegex = regexp.MustCompile(`"channelId":"(UC[a-zA-Z0-9_-]{22})"`)
)

func resolveYoutubeURLToID(inputURL string, siteRoot string) (paramType string, paramVal string) {
	inputURL = strings.TrimSpace(inputURL)
	if inputURL == "" {
		return "", ""
	}

	// 1. Try to extract from local cache
	cachePath := filepath.Join(siteRoot, ".gocache", "resolved_urls.json")
	type CacheEntry struct {
		Type string `json:"type"`
		Val  string `json:"val"`
	}
	var cache map[string]CacheEntry

	if data, err := os.ReadFile(cachePath); err == nil {
		_ = json.Unmarshal(data, &cache)
	}
	if cache == nil {
		cache = make(map[string]CacheEntry)
	}

	if entry, ok := cache[inputURL]; ok {
		return entry.Type, entry.Val
	}

	// Helper to save to cache
	saveCache := func(t, v string) {
		cache[inputURL] = CacheEntry{Type: t, Val: v}
		_ = os.MkdirAll(filepath.Dir(cachePath), 0755)
		if data, err := json.Marshal(cache); err == nil {
			_ = os.WriteFile(cachePath, data, 0644)
		}
	}

	// 2. Check if it's already a raw ID
	if strings.HasPrefix(inputURL, "UC") && len(inputURL) == 24 {
		return "channel_id", inputURL
	}
	if strings.HasPrefix(inputURL, "PL") && len(inputURL) >= 18 {
		return "playlist_id", inputURL
	}

	// 3. Direct check for Playlist ID in query params of the URL
	if u, err := url.Parse(inputURL); err == nil {
		if playlistID := u.Query().Get("list"); playlistID != "" {
			saveCache("playlist_id", playlistID)
			return "playlist_id", playlistID
		}
	}

	// 4. Direct check for Channel ID in path
	if strings.Contains(inputURL, "/channel/") {
		parts := strings.Split(inputURL, "/channel/")
		if len(parts) > 1 {
			id := strings.Split(parts[1], "/")[0]
			id = strings.Split(id, "?")[0]
			if strings.HasPrefix(id, "UC") && len(id) == 24 {
				saveCache("channel_id", id)
				return "channel_id", id
			}
		}
	}

	// 5. Resolve handle/user or search query page
	if strings.Contains(inputURL, "youtube.com/@") || strings.Contains(inputURL, "youtube.com/user/") || strings.Contains(inputURL, "/results?search_query=") || strings.Contains(inputURL, "youtu.be/") {
		client := newPublicFetchClient(5 * time.Second)
		req, err := http.NewRequest("GET", inputURL, nil)
		if err != nil {
			return "", ""
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

		resp, err := client.Do(req)
		if err != nil {
			return "", ""
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			bodyBytes, err := readRemoteResponseBody(resp)
			if err == nil {
				bodyStr := string(bodyBytes)
				// Search meta tag first
				if match := itemPropRegex.FindStringSubmatch(bodyStr); len(match) > 1 {
					saveCache("channel_id", match[1])
					return "channel_id", match[1]
				}
				// Search JSON channelId
				if match := jsonChanIDRegex.FindStringSubmatch(bodyStr); len(match) > 1 {
					saveCache("channel_id", match[1])
					return "channel_id", match[1]
				}
				// Search generic channel URLs in body (especially useful for query search results)
				if match := channelIDRegex.FindStringSubmatch(bodyStr); len(match) > 1 {
					saveCache("channel_id", match[1])
					return "channel_id", match[1]
				}
			}
		}
	}

	return "", ""
}

func resolvePodcastURLToFeed(inputURL string, siteRoot string) string {
	inputURL = strings.TrimSpace(inputURL)
	if inputURL == "" {
		return ""
	}

	if !strings.Contains(inputURL, "podcasts.apple.com") {
		return inputURL // Direct RSS feed URL
	}

	// 1. Try cache
	cachePath := filepath.Join(siteRoot, ".gocache", "resolved_podcasts.json")
	var cache map[string]string
	if data, err := os.ReadFile(cachePath); err == nil {
		_ = json.Unmarshal(data, &cache)
	}
	if cache == nil {
		cache = make(map[string]string)
	}

	if resolved, ok := cache[inputURL]; ok {
		return resolved
	}

	// 2. Parse ID from URL e.g. /id284148583
	re := regexp.MustCompile(`/id(\d+)`)
	matches := re.FindStringSubmatch(inputURL)
	if len(matches) < 2 {
		return inputURL
	}
	podcastID := matches[1]

	// 3. Request iTunes Lookup API
	lookupURL := fmt.Sprintf("https://itunes.apple.com/lookup?id=%s", podcastID)
	client := newPublicFetchClient(5 * time.Second)
	resp, err := client.Get(lookupURL)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return inputURL
		}
		body, err := remoteResponseBodyReader(resp)
		if err != nil {
			return inputURL
		}
		type iTunesResult struct {
			Results []struct {
				FeedURL string `json:"feedUrl"`
			} `json:"results"`
		}
		var lookup iTunesResult
		if err := json.NewDecoder(body).Decode(&lookup); err == nil && len(lookup.Results) > 0 {
			feedURL := lookup.Results[0].FeedURL
			if feedURL != "" {
				// Save to cache
				cache[inputURL] = feedURL
				_ = os.MkdirAll(filepath.Dir(cachePath), 0755)
				if data, err := json.Marshal(cache); err == nil {
					_ = os.WriteFile(cachePath, data, 0644)
				}
				return feedURL
			}
		}
	}

	return inputURL
}

func fetchNewsFeedWithCache(feedURL string, siteRoot string, limit int) ([]feedItem, error) {
	if err := validatePublicFetchURL(feedURL); err != nil {
		return nil, err
	}

	resolvedFeedURL := resolveNewsURLToFeed(feedURL)
	if err := validatePublicFetchURL(resolvedFeedURL); err != nil {
		return nil, err
	}

	// Cache path based on hash of the resolved feed URL
	hasher := sha256.New()
	hasher.Write([]byte(resolvedFeedURL))
	cacheFilename := fmt.Sprintf("news_%x_l%d.json", hasher.Sum(nil), limit)
	cacheDir := filepath.Join(siteRoot, ".gocache", "feeds")
	cachePath := filepath.Join(cacheDir, cacheFilename)

	// Try to fetch live feed
	items, err := fetchLiveNewsFeed(resolvedFeedURL, limit)
	if err == nil {
		// Save to cache
		_ = os.MkdirAll(cacheDir, 0755)
		if data, err := json.Marshal(items); err == nil {
			_ = os.WriteFile(cachePath, data, 0644)
		}
		return items, nil
	}

	// Fallback to cache
	if data, err := os.ReadFile(cachePath); err == nil {
		var cachedItems []feedItem
		if err := json.Unmarshal(data, &cachedItems); err == nil {
			return cachedItems, nil
		}
	}

	return nil, err
}

func feedNewsHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		feedURLs := r.URL.Query()["url"]
		if len(feedURLs) == 0 {
			http.Error(w, "url parameter is required", http.StatusBadRequest)
			return
		}
		if len(feedURLs) > maxFeedURLParams {
			http.Error(w, "too many feed parameters", http.StatusBadRequest)
			return
		}

		limitStr := r.URL.Query().Get("limit")
		limit := defaultFeedLimit
		if limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
				limit = l
			}
		}
		limit = clampFeedLimit(limit)

		var allItems []feedItem
		for _, feedURL := range feedURLs {
			items, err := fetchNewsFeedWithCache(feedURL, siteRoot, limit)
			if err == nil {
				allItems = append(allItems, items...)
			}
		}

		// Sort by Created date descending (newest first)
		sort.Slice(allItems, func(i, j int) bool {
			return allItems[i].Created > allItems[j].Created
		})

		writeJSON(w, allItems)
	}
}

func fetchLiveNewsFeed(feedURL string, limit int) ([]feedItem, error) {
	client := newPublicFetchClient(8 * time.Second)
	req, err := http.NewRequest("GET", feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Rock-OS/1.0.0 (by rocketpowerinc)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("news RSS returned HTTP %d", resp.StatusCode)
	}

	body, err := remoteResponseBodyReader(resp)
	if err != nil {
		return nil, err
	}

	var rss rssFeed
	if err := xml.NewDecoder(body).Decode(&rss); err != nil {
		return nil, err
	}

	limit = clampFeedLimit(limit)
	if limit > len(rss.Channel.Items) {
		limit = len(rss.Channel.Items)
	}

	items := make([]feedItem, 0, limit)
	source := newsFeedSourceName(feedURL, rss.Channel.Title)
	for i := 0; i < limit; i++ {
		item := rss.Channel.Items[i]
		dateStr := ""
		var pubTime time.Time
		if item.PubDate != "" {
			pubTime = parseRssDate(item.PubDate)
			if !pubTime.IsZero() {
				dateStr = pubTime.Format("2006-01-02")
			} else {
				dateStr = item.PubDate
			}
		}

		items = append(items, feedItem{
			Title:     item.Title,
			URL:       item.Link,
			Created:   dateStr,
			Source:    source,
			Thumbnail: newsItemThumbnail(item),
		})
	}

	return items, nil
}

func newsFeedSourceName(feedURL string, channelTitle string) string {
	if parsed, err := url.Parse(feedURL); err == nil {
		host := strings.ToLower(parsed.Hostname())
		switch {
		case strings.Contains(host, "ign.com"):
			return "IGN"
		case strings.Contains(host, "news.google.com"):
			return "Google News"
		}
	}

	channelTitle = strings.TrimSpace(channelTitle)
	if channelTitle != "" {
		return channelTitle
	}

	return "News"
}

func newsItemThumbnail(item rssItem) string {
	candidates := []string{
		item.MediaThumbnail.URL,
		item.Image.URL,
	}

	if item.MediaContent.URL != "" && isImageMedia(item.MediaContent.Medium, item.MediaContent.Type, item.MediaContent.URL) {
		candidates = append(candidates, item.MediaContent.URL)
	}
	if item.Enclosure.URL != "" && isImageMedia("", item.Enclosure.Type, item.Enclosure.URL) {
		candidates = append(candidates, item.Enclosure.URL)
	}

	if descriptionImage := firstHTMLImageSrc(item.Description); descriptionImage != "" {
		candidates = append(candidates, descriptionImage)
	}

	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if strings.HasPrefix(candidate, "http://") || strings.HasPrefix(candidate, "https://") {
			return candidate
		}
	}

	return ""
}

func isImageMedia(medium string, mediaType string, mediaURL string) bool {
	medium = strings.ToLower(strings.TrimSpace(medium))
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	mediaURL = strings.ToLower(strings.TrimSpace(mediaURL))

	if medium == "image" || strings.HasPrefix(mediaType, "image/") {
		return true
	}

	return strings.HasSuffix(mediaURL, ".jpg") ||
		strings.HasSuffix(mediaURL, ".jpeg") ||
		strings.HasSuffix(mediaURL, ".png") ||
		strings.HasSuffix(mediaURL, ".webp") ||
		strings.HasSuffix(mediaURL, ".gif")
}

func firstHTMLImageSrc(html string) string {
	re := regexp.MustCompile(`(?is)<img[^>]+src=["']([^"']+)["']`)
	match := re.FindStringSubmatch(html)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func resolveNewsURLToFeed(inputURL string) string {
	u, err := url.Parse(inputURL)
	if err != nil {
		return inputURL
	}

	host := strings.ToLower(u.Hostname())

	// 1. Google News topics/sections translation
	if strings.Contains(host, "news.google.com") {
		path := u.Path
		if strings.HasPrefix(path, "/topics/") {
			u.Path = "/rss/topics/" + strings.TrimPrefix(path, "/topics/")
			return u.String()
		}
		if strings.HasPrefix(path, "/sections/") {
			u.Path = "/rss/sections/" + strings.TrimPrefix(path, "/sections/")
			return u.String()
		}
		if strings.HasPrefix(path, "/search") {
			u.Path = "/rss/search"
			return u.String()
		}
		if path == "" || path == "/" {
			u.Path = "/rss"
			return u.String()
		}
		return inputURL
	}

	if err := validatePublicFetchURL(inputURL); err != nil {
		return inputURL
	}

	// 3. General RSS Auto-Discovery
	client := newPublicFetchClient(5 * time.Second)
	resp, err := client.Get(inputURL)
	if err == nil {
		defer resp.Body.Close()
		body, err := readRemoteResponseBody(resp)
		if err == nil {
			re := regexp.MustCompile(`(?i)<link[^>]+type=["']application/rss\+xml["'][^>]+href=["']([^"']+)["']`)
			match := re.FindStringSubmatch(string(body))
			if len(match) > 1 {
				href := match[1]
				// Resolve relative URL
				if strings.HasPrefix(href, "/") {
					u.Path = href
					u.RawQuery = ""
					u.Fragment = ""
					return u.String()
				}
				if !strings.HasPrefix(href, "http") {
					// Prepend base schema/host
					return fmt.Sprintf("%s://%s/%s", u.Scheme, u.Hostname(), strings.TrimPrefix(href, "/"))
				}
				return href
			}
		}
	}

	return inputURL
}

func feedSpotifyHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		urls := r.URL.Query()["url"]
		if len(urls) == 0 {
			http.Error(w, "url parameter is required", http.StatusBadRequest)
			return
		}
		if len(urls) > maxFeedURLParams {
			http.Error(w, "too many feed parameters", http.StatusBadRequest)
			return
		}

		limit := defaultFeedLimit
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
				limit = l
			}
		}
		limit = clampFeedLimit(limit)

		// Cache path based on hash of the URLs combined
		hasher := sha256.New()
		for _, u := range urls {
			hasher.Write([]byte(u))
		}
		cacheFilename := fmt.Sprintf("spotify_%x.json", hasher.Sum(nil))
		cacheDir := filepath.Join(siteRoot, ".gocache", "feeds")
		cachePath := filepath.Join(cacheDir, cacheFilename)

		// Try to fetch live
		items := []feedItem{}
		var fetchErr error

		for i, spotifyURL := range urls {
			if i >= limit {
				break
			}
			item, err := fetchSpotifyOEmbed(spotifyURL)
			if err != nil {
				fetchErr = err
				break
			}
			items = append(items, item)
		}

		if fetchErr == nil && len(items) > 0 {
			// Save to cache
			_ = os.MkdirAll(cacheDir, 0755)
			if data, err := json.Marshal(items); err == nil {
				_ = os.WriteFile(cachePath, data, 0644)
			}
			writeJSON(w, items)
			return
		}

		// Fallback to cache
		if data, err := os.ReadFile(cachePath); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(data)
			return
		}

		// Return empty list on failure
		writeJSON(w, []feedItem{})
	}
}

func fetchSpotifyOEmbed(spotifyURL string) (feedItem, error) {
	apiURL := fmt.Sprintf("https://embed.spotify.com/oembed/?url=%s", url.QueryEscape(spotifyURL))

	client := newPublicFetchClient(5 * time.Second)
	resp, err := client.Get(apiURL)
	if err != nil {
		return feedItem{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return feedItem{}, fmt.Errorf("spotify oembed returned status %d", resp.StatusCode)
	}

	body, err := remoteResponseBodyReader(resp)
	if err != nil {
		return feedItem{}, err
	}

	var data struct {
		Title        string `json:"title"`
		ThumbnailURL string `json:"thumbnail_url"`
	}

	if err := json.NewDecoder(body).Decode(&data); err != nil {
		return feedItem{}, err
	}

	return feedItem{
		Title:     data.Title,
		URL:       spotifyURL,
		Created:   "Spotify",
		Thumbnail: data.ThumbnailURL,
	}, nil
}

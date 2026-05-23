package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

const (
	indexFile   = "markdown-index.json"
	markdownDir = "markdown"
	scriptsDir  = "scripts"
)

type markdownIndexEntry struct {
	Path   string `json:"path"`
	Pinned bool   `json:"pinned,omitempty"`
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

type serverStatus struct {
	Mode        string   `json:"mode"`
	Host        string   `json:"host"`
	Description string   `json:"description"`
	URLs        []string `json:"urls"`
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
	pinned  bool
}

type markdownIndexCache struct {
	mu      sync.Mutex
	entries map[string]markdownIndexCacheEntry
}

var globalMarkdownIndexCache = &markdownIndexCache{
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
	buildIndex := flag.Bool("build-index", false, "build markdown-index.json and exit")
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

	if _, err := writeMarkdownIndex(siteRoot); err != nil {
		log.Fatal(err)
	}

	if *buildIndex {
		fmt.Println("Wrote markdown-index.json")
		return
	}

	go watchMarkdownIndex(siteRoot, 2*time.Second)

	bindHost, displayHosts, err := resolveHost(*host)
	if err != nil {
		log.Fatal(err)
	}

	fileServer := noCache(http.FileServer(http.Dir(siteRoot)))
	mux := http.NewServeMux()
	mux.HandleFunc("/api/scripts", scriptsListHandler(siteRoot))
	mux.HandleFunc("/api/scripts/content", scriptContentHandler(siteRoot))
	mux.HandleFunc("/api/scripts/run", scriptRunHandler(siteRoot))
	mux.HandleFunc("/api/server/status", serverStatusHandler(bindHost, displayHosts, *port))
	mux.HandleFunc("/api/wiki/doc", wikiDocHandler(siteRoot))
	mux.HandleFunc("/api/wiki/search", wikiSearchHandler(siteRoot))
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
	fmt.Println("[Rock-OS Wiki]")
	fmt.Printf("Serving %s\n", siteRoot)
	fmt.Printf("Listening on %s\n", address)
	fmt.Printf("Open %s\n", url)
	if len(displayHosts) > 1 {
		fmt.Println("Other local URLs:")
		for _, displayHost := range displayHosts[1:] {
			fmt.Printf("  http://%s:%d/\n", displayHost, *port)
		}
	}
	fmt.Println()

	if *open {
		if err := openBrowser(url); err != nil {
			log.Printf("Could not open browser automatically: %v", err)
		}
	}

	log.Fatal(http.Serve(listener, mux))
}

func serverStatusHandler(bindHost string, displayHosts []string, port int) http.HandlerFunc {
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

		writeJSON(w, serverStatus{
			Mode:        mode,
			Host:        bindHost,
			Description: description,
			URLs:        urls,
		})
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
		"scripts.html",
	}

	for _, file := range requiredFiles {
		info, err := os.Stat(filepath.Join(siteRoot, file))
		if err != nil || info.IsDir() {
			return false
		}
	}

	requiredDirs := []string{
		markdownDir,
		scriptsDir,
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
	fmt.Println("[Rock-OS Wiki]")
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

func watchMarkdownIndex(siteRoot string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		changed, err := writeMarkdownIndex(siteRoot)
		if err != nil {
			log.Printf("Failed to update markdown index: %v", err)
			continue
		}

		if changed {
			log.Println("Updated markdown-index.json")
		}
	}
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
	return collectMarkdownFilesWithCache(siteRoot, globalMarkdownIndexCache)
}

func collectMarkdownFilesWithCache(siteRoot string, cache *markdownIndexCache) ([]markdownIndexEntry, error) {
	root := filepath.Join(siteRoot, markdownDir)
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

		files = append(files, markdownIndexEntry{
			Path:   filepath.ToSlash(relativePath),
			Pinned: cache.markdownFilePinned(absolutePath, info),
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

func (cache *markdownIndexCache) markdownFilePinned(path string, info os.FileInfo) bool {
	if cache == nil {
		return markdownFilePinned(path)
	}

	cache.mu.Lock()
	if entry, ok := cache.entries[path]; ok &&
		entry.modTime.Equal(info.ModTime()) &&
		entry.size == info.Size() {
		cache.mu.Unlock()
		return entry.pinned
	}
	cache.mu.Unlock()

	pinned := markdownFilePinned(path)

	cache.mu.Lock()
	if cache.entries == nil {
		cache.entries = map[string]markdownIndexCacheEntry{}
	}
	cache.entries[path] = markdownIndexCacheEntry{
		modTime: info.ModTime(),
		size:    info.Size(),
		pinned:  pinned,
	}
	cache.mu.Unlock()

	return pinned
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

func markdownFilePinned(path string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return false
	}

	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			return false
		}

		key, value, found := strings.Cut(trimmed, ":")
		if !found || !strings.EqualFold(strings.TrimSpace(key), "pinned") {
			continue
		}

		normalizedValue := strings.ToLower(
			strings.Trim(strings.TrimSpace(value), `"'`),
		)

		return normalizedValue == "true" ||
			normalizedValue == "yes" ||
			normalizedValue == "1"
	}

	return false
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

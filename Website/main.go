package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
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

type scriptInputRequest struct {
	Input string `json:"input"`
}

type scriptRunResponse struct {
	SessionID string `json:"sessionId"`
}

type scriptSession struct {
	id     string
	stdin  io.WriteCloser
	output chan string
	done   chan struct{}
}

var (
	scriptSessionsMu sync.Mutex
	scriptSessions   = map[string]*scriptSession{}
)

func main() {
	host := flag.String("host", "local", "host to bind: local, 127.0.0.1, 0.0.0.0, or a specific IP")
	port := flag.Int("port", 8000, "port to listen on")
	open := flag.Bool("open", true, "open the site in your default browser")
	buildIndex := flag.Bool("build-index", false, "build markdown-index.json and exit")
	flag.Parse()

	siteRoot, err := os.Getwd()
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
	mux.HandleFunc("/api/scripts/output/", scriptOutputHandler())
	mux.HandleFunc("/api/scripts/input/", scriptInputHandler())
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

		commandName, args, err := scriptCommand(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		cmd := exec.Command(commandName, args...)
		cmd.Dir = siteRoot

		stdin, err := cmd.StdinPipe()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		session := &scriptSession{
			id:     fmt.Sprintf("%d", time.Now().UnixNano()),
			stdin:  stdin,
			output: make(chan string, 200),
			done:   make(chan struct{}),
		}

		if err := cmd.Start(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		storeScriptSession(session)

		var streams sync.WaitGroup
		streams.Add(2)
		go streamScriptPipe(session, stdout, &streams)
		go streamScriptPipe(session, stderr, &streams)
		go func() {
			err := cmd.Wait()
			streams.Wait()
			if err != nil {
				session.output <- fmt.Sprintf("\n[exit] %v\n", err)
			} else {
				session.output <- "\n[exit] script completed successfully\n"
			}
			close(session.output)
			close(session.done)
			go deleteScriptSessionLater(session.id, 5*time.Minute)
		}()

		writeJSON(w, scriptRunResponse{SessionID: session.id})
	}
}

func scriptOutputHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		sessionID := strings.TrimPrefix(r.URL.Path, "/api/scripts/output/")
		session := getScriptSession(sessionID)
		if session == nil {
			http.Error(w, "script session not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming is not supported", http.StatusInternalServerError)
			return
		}

		for output := range session.output {
			payload, _ := json.Marshal(output)
			fmt.Fprintf(w, "data: %s\n\n", payload)
			flusher.Flush()
		}

		fmt.Fprint(w, "event: done\ndata: true\n\n")
		flusher.Flush()
	}
}

func scriptInputHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		sessionID := strings.TrimPrefix(r.URL.Path, "/api/scripts/input/")
		session := getScriptSession(sessionID)
		if session == nil {
			http.Error(w, "script session not found", http.StatusNotFound)
			return
		}

		var request scriptInputRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if _, err := io.WriteString(session.stdin, request.Input); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, map[string]string{"status": "sent"})
	}
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

func scriptCommand(path string) (string, []string, error) {
	ext := strings.ToLower(filepath.Ext(path))

	if ext == ".ps1" {
		command, err := powershellCommand()
		if err != nil {
			return "", nil, err
		}

		args := []string{
			"-NoProfile",
			"-Command",
			powershellScriptCommand(path),
		}
		if runtime.GOOS == "windows" {
			args = []string{
				"-NoProfile",
				"-ExecutionPolicy",
				"Bypass",
				"-Command",
				powershellScriptCommand(path),
			}
		}

		return command, args, nil
	}

	switch runtime.GOOS {
	case "windows":
		return "cmd", []string{"/c", path}, nil
	default:
		return "sh", []string{path}, nil
	}
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

func powershellScriptCommand(path string) string {
	quotedPath := "'" + strings.ReplaceAll(path, "'", "''") + "'"

	return strings.Join([]string{
		"function global:Read-Host {",
		"param([string]$Prompt)",
		"if ($Prompt) { [Console]::Out.Write($Prompt + ' ') }",
		"[Console]::In.ReadLine()",
		"}",
		"& " + quotedPath,
	}, " ")
}

func streamScriptPipe(session *scriptSession, pipe io.Reader, streams *sync.WaitGroup) {
	defer streams.Done()

	buffer := make([]byte, 1024)
	for {
		count, err := pipe.Read(buffer)
		if count > 0 {
			session.output <- string(buffer[:count])
		}

		if err == io.EOF {
			return
		}

		if err != nil {
			session.output <- fmt.Sprintf("\n[stream] %v\n", err)
			return
		}
	}
}

func storeScriptSession(session *scriptSession) {
	scriptSessionsMu.Lock()
	defer scriptSessionsMu.Unlock()
	scriptSessions[session.id] = session
}

func getScriptSession(id string) *scriptSession {
	scriptSessionsMu.Lock()
	defer scriptSessionsMu.Unlock()
	return scriptSessions[id]
}

func deleteScriptSessionLater(id string, delay time.Duration) {
	time.Sleep(delay)
	scriptSessionsMu.Lock()
	defer scriptSessionsMu.Unlock()
	delete(scriptSessions, id)
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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
	root := filepath.Join(siteRoot, markdownDir)
	files := []markdownIndexEntry{}

	if _, err := os.Stat(root); os.IsNotExist(err) {
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

		relativePath, err := filepath.Rel(siteRoot, path)
		if err != nil {
			return err
		}

		files = append(files, markdownIndexEntry{
			Path:   filepath.ToSlash(relativePath),
			Pinned: markdownFilePinned(path),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Path) < strings.ToLower(files[j].Path)
	})

	return files, nil
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

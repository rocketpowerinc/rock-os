package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

func scriptsListHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		profile, ok := profileWorkspaceRequestProfile(w, r, siteRoot)
		if !ok {
			return
		}

		scripts, err := collectScripts(siteRoot, profile)
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

		profile, ok := profileWorkspaceRequestProfile(w, r, siteRoot)
		if !ok {
			return
		}

		script, path, err := resolveScript(siteRoot, profile, r.URL.Query().Get("id"))
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

		profile, ok := profileWorkspaceRequestProfile(w, r, siteRoot)
		if !ok {
			return
		}

		results, err := searchScripts(siteRoot, profile, r.URL.Query().Get("q"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, scriptSearchResponse{Results: results})
	}
}

func scriptRunHandler(siteRoot string, allowLanScriptRuns bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if !scriptRunRequestAllowed(r, allowLanScriptRuns) {
			http.Error(w, "unauthorized script request", http.StatusForbidden)
			return
		}

		profile, ok := profileWorkspaceRequestProfile(w, r, siteRoot)
		if !ok {
			return
		}

		var request scriptRunRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		script, path, err := resolveScript(siteRoot, profile, request.ID)
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

func scriptRunRequestAllowed(r *http.Request, allowLanScriptRuns bool) bool {
	if r.Header.Get("X-Rock-OS-Requested") != "true" {
		return false
	}

	if !allowLanScriptRuns && !requestFromLoopback(r) {
		return false
	}

	return sameOriginHeaderAllowed(r, "Origin") &&
		sameOriginHeaderAllowed(r, "Referer")
}

func requestFromLoopback(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	ip := net.ParseIP(strings.TrimSpace(host))
	return ip != nil && ip.IsLoopback()
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

func collectScripts(siteRoot string, profile string) ([]scriptEntry, error) {
	scriptsDir, err := profileWorkspaceDir(profile, "scripts")
	if err != nil {
		return nil, err
	}

	root := filepath.Join(siteRoot, filepath.FromSlash(scriptsDir))
	scripts := []scriptEntry{}

	if _, err := os.Stat(root); os.IsNotExist(err) {
		return scripts, nil
	}

	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
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

func collectAllowedProfileScripts(siteRoot string) ([]scriptEntry, error) {
	profiles, err := allowedProfileNames(siteRoot)
	if err != nil {
		return nil, err
	}

	scripts := []scriptEntry{}
	for _, profile := range profiles {
		profileScripts, err := collectScripts(siteRoot, profile)
		if err != nil {
			return nil, err
		}
		scripts = append(scripts, profileScripts...)
	}
	return scripts, nil
}

func searchScripts(siteRoot string, profile string, query string) ([]scriptSearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []scriptSearchResult{}, nil
	}

	normalizedQuery := strings.ToLower(query)
	scripts, err := collectScripts(siteRoot, profile)
	if err != nil {
		return nil, err
	}

	results := []scriptSearchResult{}
	for _, script := range scripts {
		_, path, err := resolveScript(siteRoot, profile, script.ID)
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

func resolveScript(siteRoot string, profile string, id string) (scriptEntry, string, error) {
	id = filepath.ToSlash(strings.TrimSpace(id))
	if id == "" || strings.Contains(id, "..") || strings.HasPrefix(id, "/") {
		return scriptEntry{}, "", fmt.Errorf("invalid script id")
	}
	if !safeScriptIDRegex.MatchString(id) {
		return scriptEntry{}, "", fmt.Errorf("script id contains unsupported characters")
	}

	scriptsDir, err := profileWorkspaceDir(profile, "scripts")
	if err != nil {
		return scriptEntry{}, "", err
	}

	path := filepath.Join(siteRoot, filepath.FromSlash(scriptsDir), filepath.FromSlash(id))
	root := filepath.Join(siteRoot, filepath.FromSlash(scriptsDir))

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

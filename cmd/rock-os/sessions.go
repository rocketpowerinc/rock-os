package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type dashboardSession struct {
	Name        string `json:"name"`
	AllowedPath string `json:"path,omitempty"`
	Mode        string `json:"mode,omitempty"`
	Description string `json:"description,omitempty"`
}

type dashboardSessionsConfig struct {
	Active   string             `json:"active"`
	Notes    []string           `json:"notes,omitempty"`
	Sessions []dashboardSession `json:"sessions"`
}

type dashboardSessionUpdateRequest struct {
	Active string `json:"active"`
}

type activeDashboardSessionState struct {
	Active string `json:"active"`
}

func activeDashboardSession(siteRoot string) dashboardSession {
	config := readDashboardSessionsConfig(siteRoot)
	session := resolveDashboardSession(config)
	return normalizeDashboardSession(session)
}

func readDashboardSessionsConfig(siteRoot string) dashboardSessionsConfig {
	config := defaultDashboardSessionsConfig()
	content, err := os.ReadFile(filepath.Join(siteRoot, filepath.FromSlash(sessionsFile)))
	if err != nil {
		return applyActiveDashboardSessionState(siteRoot, config)
	}

	if err := json.Unmarshal(content, &config); err != nil {
		config = defaultDashboardSessionsConfig()
		return applyActiveDashboardSessionState(siteRoot, config)
	}

	config = sanitizeDashboardSessionsConfig(config)
	return applyActiveDashboardSessionState(siteRoot, config)
}

func defaultDashboardSessionsConfig() dashboardSessionsConfig {
	return sanitizeDashboardSessionsConfig(dashboardSessionsConfig{
		Active: "SysAdmin",
		Notes: []string{
			"Rock-OS uses active as the current session.",
			"Each session renders profiles from ENCRYPTED/Sessions/<SessionName>/Profiles/.",
		},
		Sessions: []dashboardSession{
			{
				Name:        "SysAdmin",
				AllowedPath: "SysAdmin",
				Description: "Shows SysAdmin profiles.",
			},
			{
				Name:        "Family",
				AllowedPath: "Family",
				Description: "Shows Family profiles.",
			},
			{
				Name:        "Doomsday",
				AllowedPath: "Doomsday",
				Description: "Shows Doomsday profiles.",
			},
		},
	})
}

func sessionKeyUnlocked(siteRoot string, keyFile string) bool {
	info, err := os.Stat(filepath.Join(siteRoot, filepath.FromSlash(sessionKeysDir), keyFile))
	return err == nil && !info.IsDir()
}

func rocketProfileUnlocked(siteRoot string) bool {
	return sessionKeyUnlocked(siteRoot, rocketKeyFile)
}

func applyActiveDashboardSessionState(siteRoot string, config dashboardSessionsConfig) dashboardSessionsConfig {
	active, ok := readActiveDashboardSessionState(siteRoot)
	if ok && dashboardSessionExists(config.Sessions, active) {
		config.Active = active
	}
	if !dashboardSessionExists(config.Sessions, config.Active) {
		config.Active = "SysAdmin"
	}
	if !dashboardSessionExists(config.Sessions, config.Active) && len(config.Sessions) > 0 {
		config.Active = config.Sessions[0].Name
	}
	return config
}

func readActiveDashboardSessionState(siteRoot string) (string, bool) {
	content, err := os.ReadFile(filepath.Join(siteRoot, filepath.FromSlash(activeSessionFile)))
	if err != nil {
		return "", false
	}

	var state activeDashboardSessionState
	if err := json.Unmarshal(content, &state); err != nil {
		return "", false
	}

	active := strings.TrimSpace(state.Active)
	if active == "" {
		return "", false
	}
	return active, true
}

func sanitizeDashboardSessionsConfig(config dashboardSessionsConfig) dashboardSessionsConfig {
	sessions := []dashboardSession{}
	seen := map[string]bool{}

	for _, session := range config.Sessions {
		session.Name = strings.TrimSpace(session.Name)
		if session.Name == "" || seen[strings.ToLower(session.Name)] {
			continue
		}
		seen[strings.ToLower(session.Name)] = true
		sessions = append(sessions, normalizeDashboardSession(session))
	}

	if len(sessions) == 0 {
		return defaultDashboardSessionsConfig()
	}

	config.Active = strings.TrimSpace(config.Active)
	if !dashboardSessionExists(sessions, config.Active) {
		config.Active = sessions[0].Name
	}
	config.Sessions = sessions
	return config
}

func dashboardSessionExists(sessions []dashboardSession, name string) bool {
	for _, session := range sessions {
		if strings.EqualFold(session.Name, name) {
			return true
		}
	}
	return false
}

func resolveDashboardSession(config dashboardSessionsConfig) dashboardSession {
	for _, session := range config.Sessions {
		if strings.EqualFold(session.Name, config.Active) {
			return session
		}
	}

	return dashboardSession{
		Name:        "SysAdmin",
		AllowedPath: "SysAdmin",
	}
}

func normalizeDashboardSession(session dashboardSession) dashboardSession {
	session.Mode = strings.ToLower(strings.TrimSpace(session.Mode))
	if session.AllowedPath == "" {
		session.AllowedPath = session.Name
	}
	session.AllowedPath = normalizeDashboardSessionPath(session.AllowedPath)
	return session
}

func normalizeDashboardSessionPath(value string) string {
	normalized := strings.Trim(strings.ReplaceAll(value, "\\", "/"), "/")
	if normalized == "" || strings.Contains(normalized, "\x00") {
		return ""
	}

	parts := []string{}
	for _, part := range strings.Split(normalized, "/") {
		part = strings.TrimSpace(part)
		if part == "" || part == "." || part == ".." {
			return ""
		}
		parts = append(parts, part)
	}

	return strings.Join(parts, "/")
}

func filterDashboardFilesForActiveSession(siteRoot string, files []markdownIndexEntry) []markdownIndexEntry {
	return filterDashboardFilesForSession(siteRoot, files, activeDashboardSession(siteRoot))
}

func dashboardSessionAllowsPath(siteRoot string, dashboard string) bool {
	dashboard = normalizeDashboardSessionPath(dashboard)
	if dashboard == "" {
		return false
	}

	probe := []markdownIndexEntry{{
		Path: profilesDir + "/" + dashboard + "/__access__.md",
	}}

	return len(filterDashboardFilesForSession(siteRoot, probe, activeDashboardSession(siteRoot))) == 1
}

func filterDashboardFilesForSession(siteRoot string, files []markdownIndexEntry, session dashboardSession) []markdownIndexEntry {
	if session.AllowedPath == "" {
		return []markdownIndexEntry{}
	}

	filtered := filterDashboardFilesInsideProfilePath(files, session.AllowedPath)
	if !rocketProfileUnlocked(siteRoot) {
		filtered = filterDashboardFilesOutsidePath(filtered, "SysAdmin/Profiles/Rocket")
	}
	return filtered
}

func filterDashboardFilesInsideProfilePath(files []markdownIndexEntry, dashboard string) []markdownIndexEntry {
	dashboard = normalizeDashboardSessionPath(dashboard)
	if dashboard == "" {
		return []markdownIndexEntry{}
	}

	prefix := profilesDir + "/" + dashboard + "/"
	filtered := []markdownIndexEntry{}
	for _, file := range files {
		if strings.HasPrefix(file.Path, prefix) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

func filterDashboardFilesOutsidePath(files []markdownIndexEntry, dashboard string) []markdownIndexEntry {
	dashboard = normalizeDashboardSessionPath(dashboard)
	if dashboard == "" {
		return files
	}

	prefix := profilesDir + "/" + dashboard + "/"
	filtered := []markdownIndexEntry{}
	for _, file := range files {
		if !strings.HasPrefix(file.Path, prefix) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

func sessionsHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, readDashboardSessionsConfig(siteRoot))
		case http.MethodPost:
			if !serverRefreshRequestAllowed(r) {
				http.Error(w, "unauthorized session update request", http.StatusForbidden)
				return
			}

			var request dashboardSessionUpdateRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, "invalid session update request", http.StatusBadRequest)
				return
			}

			config := readDashboardSessionsConfig(siteRoot)
			request.Active = strings.TrimSpace(request.Active)
			if !dashboardSessionExists(config.Sessions, request.Active) {
				http.Error(w, fmt.Sprintf("unknown dashboard session: %s", request.Active), http.StatusBadRequest)
				return
			}

			config.Active = request.Active
			if err := writeActiveDashboardSessionState(siteRoot, config.Active); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			writeJSON(w, config)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func writeActiveDashboardSessionState(siteRoot string, active string) error {
	state := activeDashboardSessionState{
		Active: strings.TrimSpace(active),
	}
	content, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')

	path := filepath.Join(siteRoot, filepath.FromSlash(activeSessionFile))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

func writeDashboardSessionsConfig(siteRoot string, config dashboardSessionsConfig) error {
	config = sanitizeDashboardSessionsConfig(config)
	content, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')

	path := filepath.Join(siteRoot, filepath.FromSlash(sessionsFile))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

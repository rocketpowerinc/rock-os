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
	Admin       bool   `json:"-"`
	Public      bool   `json:"-"`
	Rocket      bool   `json:"-"`
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
		config = applyLocalKeySessionAvailability(siteRoot, config)
		return applyActiveDashboardSessionState(siteRoot, config)
	}

	if err := json.Unmarshal(content, &config); err != nil {
		config = applyLocalKeySessionAvailability(siteRoot, defaultDashboardSessionsConfig())
		return applyActiveDashboardSessionState(siteRoot, config)
	}

	if localKeySessionUnlocked(siteRoot, adminKeyFile) &&
		localKeySessionRequested(config, "Admin") {
		config.Sessions = append(config.Sessions, adminDashboardSession())
	}
	if localKeySessionUnlocked(siteRoot, rocketKeyFile) &&
		localKeySessionRequested(config, "Rocket") {
		config.Sessions = append(config.Sessions, rocketDashboardSession())
	}

	config = applyLocalKeySessionAvailability(siteRoot, sanitizeDashboardSessionsConfig(config))
	return applyActiveDashboardSessionState(siteRoot, config)
}

func defaultDashboardSessionsConfig() dashboardSessionsConfig {
	return sanitizeDashboardSessionsConfig(dashboardSessionsConfig{
		Active: "Public",
		Notes: []string{
			"Rock-OS uses active as the current dashboard session.",
			"Public shows dashboards but hides Profiles.",
			"Path sessions show only one profile workspace, such as Profiles/Kids.",
			"Add future sessions to the sessions list so they appear in the home-page dropdown.",
		},
		Sessions: []dashboardSession{
			{
				Name:        "Public",
				Mode:        "public",
				Description: "Shows normal dashboard sections, but hides Profiles.",
			},
			{
				Name:        "Kids",
				AllowedPath: "Profiles/Kids",
				Description: "Shows only the Kids profile dashboard.",
			},
		},
	})
}

func applyLocalKeySessionAvailability(siteRoot string, config dashboardSessionsConfig) dashboardSessionsConfig {
	sessions := []dashboardSession{}
	for _, session := range config.Sessions {
		if isLocalKeySession(session) {
			continue
		}
		sessions = append(sessions, session)
	}

	if localKeySessionUnlocked(siteRoot, adminKeyFile) {
		insertAt := min(1, len(sessions))
		sessions = append(sessions[:insertAt], append([]dashboardSession{adminDashboardSession()}, sessions[insertAt:]...)...)
	} else if strings.EqualFold(config.Active, "Admin") {
		config.Active = "Public"
	}

	if localKeySessionUnlocked(siteRoot, rocketKeyFile) {
		insertAt := min(2, len(sessions))
		sessions = append(sessions[:insertAt], append([]dashboardSession{rocketDashboardSession()}, sessions[insertAt:]...)...)
	} else if strings.EqualFold(config.Active, "Rocket") {
		config.Active = "Public"
	}

	config.Sessions = sessions
	if !dashboardSessionExists(config.Sessions, config.Active) {
		config.Active = "Public"
	}
	if !dashboardSessionExists(config.Sessions, config.Active) && len(config.Sessions) > 0 {
		config.Active = config.Sessions[0].Name
	}

	return config
}

func adminDashboardSession() dashboardSession {
	return dashboardSession{
		Name:        "Admin",
		Mode:        "admin",
		Description: "Shows dashboard sections except the Rocket profile.",
		Admin:       true,
	}
}

func rocketDashboardSession() dashboardSession {
	return dashboardSession{
		Name:        "Rocket",
		Mode:        "rocket",
		Description: "Shows every dashboard section.",
		Rocket:      true,
	}
}

func localKeySessionUnlocked(siteRoot string, keyFile string) bool {
	info, err := os.Stat(filepath.Join(filepath.Dir(siteRoot), keyFile))
	return err == nil && !info.IsDir()
}

func localKeySessionRequested(config dashboardSessionsConfig, name string) bool {
	return strings.EqualFold(strings.TrimSpace(config.Active), name) &&
		!dashboardSessionExists(config.Sessions, name)
}

func applyActiveDashboardSessionState(siteRoot string, config dashboardSessionsConfig) dashboardSessionsConfig {
	active, ok := readActiveDashboardSessionState(siteRoot)
	if ok && dashboardSessionExists(config.Sessions, active) {
		config.Active = active
	}
	if !dashboardSessionExists(config.Sessions, config.Active) {
		config.Active = "Public"
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
		Name:   "Public",
		Mode:   "public",
		Public: true,
	}
}

func normalizeDashboardSession(session dashboardSession) dashboardSession {
	session.Mode = strings.ToLower(strings.TrimSpace(session.Mode))
	if strings.EqualFold(session.Name, "Public") || session.Mode == "public" {
		session.Mode = "public"
		session.Public = true
		session.Admin = false
		session.Rocket = false
		session.AllowedPath = ""
		return session
	}
	if strings.EqualFold(session.Name, "Admin") || session.Mode == "admin" {
		session.Mode = "admin"
		session.Admin = true
		session.Public = false
		session.Rocket = false
		session.AllowedPath = ""
		return session
	}
	if strings.EqualFold(session.Name, "Rocket") || session.Mode == "rocket" {
		session.Mode = "rocket"
		session.Rocket = true
		session.Admin = false
		session.Public = false
		session.AllowedPath = ""
		return session
	}

	session.Public = false
	session.Admin = false
	session.Rocket = false
	if session.AllowedPath == "" {
		session.AllowedPath = "Profiles/" + session.Name
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

	if len(parts) == 1 {
		parts = append([]string{"Profiles"}, parts...)
	}

	return strings.Join(parts, "/")
}

func isLocalKeySession(session dashboardSession) bool {
	return session.Admin || session.Rocket ||
		strings.EqualFold(session.Name, "Admin") ||
		strings.EqualFold(session.Name, "Rocket") ||
		strings.EqualFold(session.Mode, "admin") ||
		strings.EqualFold(session.Mode, "rocket")
}

func filterDashboardFilesForActiveSession(siteRoot string, files []markdownIndexEntry) []markdownIndexEntry {
	return filterDashboardFilesForSession(files, activeDashboardSession(siteRoot))
}

func dashboardSessionAllowsPath(siteRoot string, dashboard string) bool {
	dashboard = normalizeDashboardSessionPath(dashboard)
	if dashboard == "" {
		return false
	}

	probe := []markdownIndexEntry{{
		Path: profilesDir + "/" + strings.TrimPrefix(dashboard, "Profiles/") + "/__access__.md",
	}}

	return len(filterDashboardFilesForSession(probe, activeDashboardSession(siteRoot))) == 1
}

func filterDashboardFilesForSession(files []markdownIndexEntry, session dashboardSession) []markdownIndexEntry {
	if session.Admin {
		return filterDashboardFilesOutsidePath(files, "Profiles/Rocket")
	}

	if session.Rocket {
		return files
	}

	if session.Public {
		filtered := []markdownIndexEntry{}
		profilesPrefix := profilesDir + "/"
		for _, file := range files {
			if !strings.HasPrefix(file.Path, profilesPrefix) {
				filtered = append(filtered, file)
			}
		}
		return filtered
	}

	if session.AllowedPath == "" {
		return []markdownIndexEntry{}
	}

	return filterDashboardFilesInsideProfilePath(files, session.AllowedPath)
}

func filterDashboardFilesInsideProfilePath(files []markdownIndexEntry, dashboard string) []markdownIndexEntry {
	dashboard = normalizeDashboardSessionPath(dashboard)
	if dashboard == "" {
		return []markdownIndexEntry{}
	}

	prefix := profilesDir + "/" + strings.TrimPrefix(dashboard, "Profiles/") + "/"
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

	prefix := profilesDir + "/" + strings.TrimPrefix(dashboard, "Profiles/") + "/"
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
	if localKeySessionUnlocked(siteRoot, adminKeyFile) &&
		localKeySessionRequested(config, "Admin") {
		config.Sessions = append(config.Sessions, adminDashboardSession())
	}
	if localKeySessionUnlocked(siteRoot, rocketKeyFile) &&
		localKeySessionRequested(config, "Rocket") {
		config.Sessions = append(config.Sessions, rocketDashboardSession())
	}
	config = sanitizeDashboardSessionsConfig(config)
	config.Sessions = stripLocalKeySessions(config.Sessions)
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

func stripLocalKeySessions(sessions []dashboardSession) []dashboardSession {
	filtered := []dashboardSession{}
	for _, session := range sessions {
		if isLocalKeySession(session) {
			continue
		}
		filtered = append(filtered, session)
	}
	return filtered
}

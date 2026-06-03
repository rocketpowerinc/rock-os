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
}

type dashboardSessionsConfig struct {
	Active   string             `json:"active"`
	Notes    []string           `json:"notes,omitempty"`
	Sessions []dashboardSession `json:"sessions"`
}

type dashboardSessionUpdateRequest struct {
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
		return config
	}

	if err := json.Unmarshal(content, &config); err != nil {
		return defaultDashboardSessionsConfig()
	}

	return sanitizeDashboardSessionsConfig(config)
}

func defaultDashboardSessionsConfig() dashboardSessionsConfig {
	return sanitizeDashboardSessionsConfig(dashboardSessionsConfig{
		Active: "Public",
		Notes: []string{
			"Rock-OS uses active as the current dashboard session.",
			"Public shows dashboards but hides Profiles.",
			"Admin shows every dashboard section, including Profiles.",
			"Path sessions show only one dashboard folder, such as Profiles/Kids.",
			"Add future sessions to the sessions list so they appear in the home-page dropdown.",
		},
		Sessions: []dashboardSession{
			{
				Name:        "Public",
				Mode:        "public",
				Description: "Shows normal dashboard sections, but hides Profiles.",
			},
			{
				Name:        "Admin",
				Mode:        "admin",
				Description: "Shows every dashboard section, including Profiles.",
			},
			{
				Name:        "Kids",
				AllowedPath: "Profiles/Kids",
				Description: "Shows only the Kids profile dashboard.",
			},
		},
	})
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
		session.AllowedPath = ""
		return session
	}
	if strings.EqualFold(session.Name, "Admin") || session.Mode == "admin" {
		session.Mode = "admin"
		session.Admin = true
		session.Public = false
		session.AllowedPath = ""
		return session
	}

	session.Public = false
	session.Admin = false
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

func filterDashboardFilesForActiveSession(siteRoot string, files []markdownIndexEntry) []markdownIndexEntry {
	return filterDashboardFilesForSession(files, activeDashboardSession(siteRoot))
}

func filterDashboardFilesForSession(files []markdownIndexEntry, session dashboardSession) []markdownIndexEntry {
	if session.Admin {
		return files
	}

	if session.Public {
		filtered := []markdownIndexEntry{}
		profilesPrefix := dashboardsDir + "/Profiles/"
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

	return filterDashboardFiles(files, session.AllowedPath)
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
			if err := writeDashboardSessionsConfig(siteRoot, config); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			writeJSON(w, config)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
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

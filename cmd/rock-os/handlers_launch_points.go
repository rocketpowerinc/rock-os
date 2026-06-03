package main

import (
	"bytes"
	"fmt"
	"html"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var launchPointLinkPattern = regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)

func launchPointMarkdownPageHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		cleaned := path.Clean(r.URL.Path)
		fileName := path.Base(cleaned)
		if cleaned != "/"+launchPointsDir+"/"+fileName ||
			fileName == "." ||
			!strings.EqualFold(path.Ext(fileName), ".md") {
			http.NotFound(w, r)
			return
		}

		title := strings.TrimSuffix(fileName, path.Ext(fileName))
		filePath := filepath.Join(siteRoot, launchPointsDir, filepath.FromSlash(fileName))
		content, err := os.ReadFile(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var rendered bytes.Buffer
		if len(strings.TrimSpace(string(content))) == 0 {
			rendered.WriteString("<p>This locked-mode launch card is empty.</p>")
		} else if err := wikiMarkdown.Convert(content, &rendered); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		fmt.Fprintf(w, launchPointMarkdownPageTemplate, html.EscapeString(title), rendered.String())
	}
}

func launchPointsHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		points, err := collectLaunchPoints(siteRoot)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, points)
	}
}

const launchPointMarkdownPageTemplate = `<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s - Rock-OS</title>
<link rel="icon" href="/assets/favicon.ico" sizes="any">
<link rel="icon" type="image/png" href="/assets/favicon-32.png" sizes="32x32">
<link rel="apple-touch-icon" href="/assets/apple-touch-icon.png">
<link rel="manifest" href="/site.webmanifest">
<meta name="theme-color" content="#4f6475">
<script src="/js/theme-init.js"></script>
<link rel="stylesheet" href="/css/style.css">
</head>
<body>
<div class="grid-overlay"></div>
<nav class="navbar">
<div class="logo-group">
<img src="/assets/Rock-OS-Hero-Steel.png" class="logo theme-logo">
<div class="brand">Rock-OS</div>
</div>
<div class="nav-links">
<a href="/index.html">HOME</a>
<a href="/dashboards.html">DASHBOARDS</a>
<div class="nav-menu">
<button class="nav-menu-trigger" type="button" aria-haspopup="true" aria-expanded="false">MENU</button>
<div class="nav-menu-list" role="menu">
<a href="/bookmarks.html" role="menuitem">Bookmarks</a>
<a href="/cheatsheets.html" role="menuitem">Cheatsheets</a>
<a href="/dotfiles.html" role="menuitem">Dotfiles</a>
<a href="/guides.html" role="menuitem">Guides</a>
<a href="/scripts.html" role="menuitem">Scripts</a>
<a href="/wiki.html" role="menuitem">Wiki</a>
</div>
</div>
<select id="themeSelect" class="theme-select" aria-label="Theme">
<option value="steel">Steel</option>
<option value="rugged">Rugged</option>
<option value="cyberpunk">Cyberpunk</option>
<option value="blue-grass">Blue-Grass</option>
</select>
</div>
</nav>
<main class="content fullwidth launch-point-markdown-page">
%s
</main>
<script src="/js/theme.js"></script>
</body>
</html>
`

func collectLaunchPoints(siteRoot string) ([]launchPoint, error) {
	root := filepath.Join(siteRoot, launchPointsDir)
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return []launchPoint{}, nil
	}
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	points := []launchPoint{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(root, entry.Name()))
		if err != nil {
			return nil, err
		}

		point, err := parseLaunchPoint(string(content))
		if err != nil {
			point = fallbackLaunchPoint(entry.Name())
		}
		point.Title = strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		point.Path = "/" + filepath.ToSlash(filepath.Join(launchPointsDir, entry.Name()))
		points = append(points, point)
	}

	return points, nil
}

func parseLaunchPoint(markdown string) (launchPoint, error) {
	point := launchPoint{}
	descriptionLines := []string{}

	for _, rawLine := range strings.Split(markdown, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "<!--") {
			continue
		}

		if point.Title == "" && strings.HasPrefix(line, "# ") {
			point.Title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
			continue
		}

		if point.Href == "" {
			match := launchPointLinkPattern.FindStringSubmatch(line)
			if len(match) == 2 {
				point.Href = strings.TrimSpace(match[1])
				continue
			}
		}

		if !strings.HasPrefix(line, "#") {
			descriptionLines = append(descriptionLines, line)
		}
	}

	point.Description = strings.Join(descriptionLines, " ")
	if point.Title == "" {
		return launchPoint{}, fmt.Errorf("add a '# Title' heading")
	}
	if point.Description == "" {
		return launchPoint{}, fmt.Errorf("add a description paragraph")
	}
	if point.Href == "" {
		return launchPoint{}, fmt.Errorf("add a Markdown link destination")
	}

	return point, nil
}

func fallbackLaunchPoint(fileName string) launchPoint {
	title := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	return launchPoint{
		Title:       title,
		Description: "Open this locked-mode launch card.",
		Href:        "/" + filepath.ToSlash(filepath.Join(launchPointsDir, fileName)),
	}
}

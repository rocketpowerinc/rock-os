package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var launchPointLinkPattern = regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)

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

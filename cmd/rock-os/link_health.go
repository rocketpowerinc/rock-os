package main

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

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

const linkHealthIgnoreMarker = "rock-os-ignore-link"

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

		text := string(content)
		for _, match := range markdownLinkRegex.FindAllStringSubmatchIndex(text, -1) {
			if linkHealthIgnoredOnLine(text, match[1]) {
				continue
			}

			label := strings.TrimSpace(text[match[2]:match[3]])
			href := cleanMarkdownHref(text[match[4]:match[5]])
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

func linkHealthIgnoredOnLine(content string, linkEnd int) bool {
	if linkEnd < 0 || linkEnd > len(content) {
		return false
	}

	lineEnd := strings.IndexAny(content[linkEnd:], "\r\n")
	if lineEnd < 0 {
		lineEnd = len(content)
	} else {
		lineEnd += linkEnd
	}

	return strings.Contains(content[linkEnd:lineEnd], linkHealthIgnoreMarker)
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

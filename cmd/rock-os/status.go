package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// privateMarkdownStatus walks the entire encrypted tree, so it is cached for a
// short window to avoid repeating that walk on every gated request. The lock /
// unlock state only changes when the user runs git-crypt, so a few seconds of
// staleness is harmless. The cache is also cleared after a successful refresh.
const privateMarkdownStatusTTL = 3 * time.Second

var privateMarkdownStatusCache = struct {
	mu       sync.Mutex
	value    string
	siteRoot string
	expires  time.Time
}{}

func invalidatePrivateMarkdownStatus() {
	privateMarkdownStatusCache.mu.Lock()
	privateMarkdownStatusCache.expires = time.Time{}
	privateMarkdownStatusCache.mu.Unlock()
}

func serverRefreshHandler(siteRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if !serverRefreshRequestAllowed(r) {
			http.Error(w, "unauthorized refresh request", http.StatusForbidden)
			return
		}

		repoRoot := filepath.Dir(siteRoot)
		if _, err := os.Stat(filepath.Join(repoRoot, ".git")); err != nil {
			http.Error(w, "Rock-OS is not running from a Git clone", http.StatusConflict)
			return
		}

		beforeHead, err := gitHead(repoRoot)
		if err != nil {
			http.Error(w, "Git is required for live updates", http.StatusServiceUnavailable)
			return
		}

		cmd := exec.Command("git", "pull", "--ff-only")
		cmd.Dir = repoRoot
		output, err := cmd.CombinedOutput()
		if err != nil {
			message := strings.TrimSpace(string(output))
			if message == "" {
				message = err.Error()
			}
			http.Error(w, "Could not update from GitHub: "+message, http.StatusConflict)
			return
		}

		afterHead, err := gitHead(repoRoot)
		if err != nil {
			http.Error(w, "Updated files but could not read the current Git commit", http.StatusInternalServerError)
			return
		}

		// The pull may have changed which files are encrypted; drop the cache
		// so the next status check re-walks the tree.
		invalidatePrivateMarkdownStatus()

		updated := beforeHead != afterHead
		message := "Rock-OS is already up to date."
		if updated {
			message = "Rock-OS updated. Reloading the website."
		}

		writeJSON(w, serverRefreshResponse{
			Updated: updated,
			Message: message,
		})
	}
}

func serverRefreshRequestAllowed(r *http.Request) bool {
	return r.Header.Get("X-Rock-OS-Requested") == "true" &&
		requestFromLoopback(r) &&
		sameOriginHeaderAllowed(r, "Origin") &&
		sameOriginHeaderAllowed(r, "Referer")
}

func gitHead(repoRoot string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func printStartupStatus(siteRoot string, bindHost string, address string, allowLanScriptRuns bool) {
	printStatus("OK", ansiGreen, "Serving %s", siteRoot)
	printStatus("OK", ansiGreen, "Listening on %s", address)

	if bindHost == "127.0.0.1" || bindHost == "localhost" {
		printStatus("OK", ansiGreen, "Server Mode: Host")
	} else {
		printStatus("WARN", ansiYellow, "Server Mode: LAN")
	}

	if allowLanScriptRuns {
		printStatus("WARN", ansiYellow, "LAN script runs enabled. Trusted clients on this network can launch scripts.")
	} else {
		printStatus("OK", ansiGreen, "Script runs restricted to this computer.")
	}

	if files, err := collectAllowedProfileMarkdownFiles(siteRoot, "wiki"); err == nil {
		printStatus("OK", ansiGreen, "Profile wiki docs indexed on demand: %d", len(files))
	} else {
		printStatus("WARN", ansiYellow, "Profile wiki docs could not be scanned: %v", err)
	}

	if scripts, err := collectAllowedProfileScripts(siteRoot); err == nil {
		printStatus("OK", ansiGreen, "Profile scripts available: %d", len(scripts))
	} else {
		printStatus("WARN", ansiYellow, "Profile scripts could not be scanned: %v", err)
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
		printStatus("INFO", ansiCyan, "Encrypted content locked.")
	case "unlocked":
		printStatus("OK", ansiGreen, "Encrypted content unlocked.")
	default:
		printStatus("INFO", ansiCyan, "Encrypted content folder not found.")
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
	now := time.Now()

	privateMarkdownStatusCache.mu.Lock()
	if privateMarkdownStatusCache.siteRoot == siteRoot &&
		now.Before(privateMarkdownStatusCache.expires) {
		value := privateMarkdownStatusCache.value
		privateMarkdownStatusCache.mu.Unlock()
		return value
	}
	privateMarkdownStatusCache.mu.Unlock()

	value := computePrivateMarkdownStatus(siteRoot)

	privateMarkdownStatusCache.mu.Lock()
	privateMarkdownStatusCache.value = value
	privateMarkdownStatusCache.siteRoot = siteRoot
	privateMarkdownStatusCache.expires = now.Add(privateMarkdownStatusTTL)
	privateMarkdownStatusCache.mu.Unlock()

	return value
}

func computePrivateMarkdownStatus(siteRoot string) string {
	privateRoot := filepath.Join(siteRoot, encryptedDir)
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

	for _, match := range matches {
		name := filepath.Base(match)
		if !strings.EqualFold(name, adminKeyFile) &&
			!strings.EqualFold(name, rocketKeyFile) {
			return true
		}
	}

	return false
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
		if files, err := collectAllowedProfileMarkdownFiles(siteRoot, "wiki"); err == nil {
			markdownCount = len(files)
		}
		scriptsCount := 0
		if scripts, err := collectAllowedProfileScripts(siteRoot); err == nil {
			scriptsCount = len(scripts)
		}
		commit, _ := gitHead(filepath.Dir(siteRoot))

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
			Commit:       commit,
		})
	}
}

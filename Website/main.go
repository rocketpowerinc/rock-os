package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

const (
	indexFile   = "markdown-index.json"
	markdownDir = "markdown"
)

func main() {
	host := flag.String("host", "127.0.0.1", "host address to bind")
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

	fileServer := noCache(http.FileServer(http.Dir(siteRoot)))
	address := fmt.Sprintf("%s:%d", *host, *port)
	url := fmt.Sprintf("http://%s:%d/", *host, *port)

	fmt.Println()
	fmt.Println("[Rock-OS Wiki]")
	fmt.Printf("Serving %s\n", siteRoot)
	fmt.Printf("Open %s\n", url)
	fmt.Println()

	if *open {
		if err := openBrowser(url); err != nil {
			log.Printf("Could not open browser automatically: %v", err)
		}
	}

	log.Fatal(http.ListenAndServe(address, fileServer))
}

func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
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

func collectMarkdownFiles(siteRoot string) ([]string, error) {
	root := filepath.Join(siteRoot, markdownDir)
	files := []string{}

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

		files = append(files, filepath.ToSlash(relativePath))
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i]) < strings.ToLower(files[j])
	})

	return files, nil
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

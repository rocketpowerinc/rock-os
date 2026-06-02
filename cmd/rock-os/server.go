package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type apiRateLimiter struct {
	mu      sync.Mutex
	clients map[string]*apiRateLimitClient
}

type apiRateLimitClient struct {
	tokens float64
	last   time.Time
}

func newAPIRateLimiter() *apiRateLimiter {
	return &apiRateLimiter{
		clients: map[string]*apiRateLimitClient{},
	}
}

func rateLimitAPI(next http.Handler) http.Handler {
	limiter := newAPIRateLimiter()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		if !limiter.allow(r) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (limiter *apiRateLimiter) allow(r *http.Request) bool {
	clientKey := requestClientKey(r)
	if clientKey == "" {
		clientKey = "unknown"
	}

	now := time.Now()
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	client := limiter.clients[clientKey]
	if client == nil {
		limiter.clients[clientKey] = &apiRateLimitClient{
			tokens: apiRateLimitBurst - 1,
			last:   now,
		}
		return true
	}

	elapsed := now.Sub(client.last).Seconds()
	client.tokens += elapsed * apiRateLimitRefill
	if client.tokens > apiRateLimitBurst {
		client.tokens = apiRateLimitBurst
	}
	client.last = now

	if len(limiter.clients) > 512 {
		for key, value := range limiter.clients {
			if now.Sub(value.last) > 10*time.Minute {
				delete(limiter.clients, key)
			}
		}
	}

	if client.tokens < 1 {
		return false
	}

	client.tokens--
	return true
}

func requestClientKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

type gzipResponseWriter struct {
	http.ResponseWriter
	writer *gzip.Writer
}

func (writer *loggingResponseWriter) WriteHeader(status int) {
	writer.status = status
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *loggingResponseWriter) Write(data []byte) (int, error) {
	if writer.status == 0 {
		writer.status = http.StatusOK
	}

	return writer.ResponseWriter.Write(data)
}

func (writer *gzipResponseWriter) Write(data []byte) (int, error) {
	return writer.writer.Write(data)
}

func compressResponses(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requestAcceptsGzip(r) || !shouldCompressPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Add("Vary", "Accept-Encoding")
		w.Header().Del("Content-Length")
		w.Header().Set("Content-Encoding", "gzip")

		gzipWriter := gzip.NewWriter(w)
		defer gzipWriter.Close()

		next.ServeHTTP(&gzipResponseWriter{
			ResponseWriter: w,
			writer:         gzipWriter,
		}, r)
	})
}

func requestAcceptsGzip(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
}

func shouldCompressPath(path string) bool {
	if strings.HasPrefix(path, "/api/") ||
		path == "/wiki-index.json" ||
		path == "/guides-index.json" ||
		path == "/cheatsheets-index.json" ||
		path == "/dotfiles-index.json" ||
		path == "/bookmarks-index.json" ||
		path == "/dashboards-index.json" {
		return true
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case "", ".html", ".css", ".js", ".json", ".md", ".svg", ".txt", ".xml", ".webmanifest":
		return true
	default:
		return false
	}
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		writer := &loggingResponseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(writer, r)

		requestTime := colorize(ansiDim, time.Now().Format("15:04:05"))
		method := colorize(methodLogColor(r.Method), r.Method)
		path := colorize(ansiCyan, r.URL.Path)
		status := colorize(statusLogColor(writer.status), fmt.Sprintf("%d", writer.status))
		duration := colorize(ansiDim, time.Since(start).Round(time.Microsecond).String())

		fmt.Printf("%s %s %s %s %s\n", requestTime, method, path, status, duration)
	})
}

func methodLogColor(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead:
		return ansiGreen
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return ansiBlue
	case http.MethodDelete:
		return ansiRed
	default:
		return ansiCyan
	}
}

func statusLogColor(status int) string {
	switch {
	case status >= 500:
		return ansiRed
	case status >= 400:
		return ansiYellow
	case status >= 300:
		return ansiCyan
	default:
		return ansiGreen
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func requireUnlockedContent(siteRoot string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if privateMarkdownStatus(siteRoot) != "unlocked" {
			http.Error(w, "encrypted Rock-OS content is locked", http.StatusLocked)
			return
		}

		next(w, r)
	}
}

func isAddressInUse(err error) bool {
	message := strings.ToLower(err.Error())

	return strings.Contains(message, "address already in use") ||
		strings.Contains(message, "only one usage of each socket address")
}

func printPortInUseMessage(address string, displayHosts []string, port int) {
	fmt.Println()
	fmt.Println("[Rock-OS]")
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

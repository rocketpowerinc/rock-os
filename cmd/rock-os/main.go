package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	host := flag.String("host", "127.0.0.1", "host to bind: 127.0.0.1, local, lan, 0.0.0.0, or a specific IP")
	port := flag.Int("port", 8000, "port to listen on")
	open := flag.Bool("open", true, "open the site in your default browser")
	buildIndex := flag.Bool("build-index", false, "build wiki-index.json and exit")
	siteRootFlag := flag.String("site-root", "", "path to the Website folder; auto-detected when omitted")
	allowLanScriptRuns := flag.Bool("enable-lan-script-runs", false, "allow script execution requests from non-loopback LAN clients")
	flag.Parse()

	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	siteRoot, err := resolveSiteRoot(workingDir, *siteRootFlag)
	if err != nil {
		log.Fatal(err)
	}

	if *buildIndex {
		if privateMarkdownStatus(siteRoot) != "unlocked" {
			log.Fatal("encrypted Rock-OS content is locked")
		}

		if _, err := writeMarkdownIndex(siteRoot); err != nil {
			log.Fatal(err)
		}
		if _, err := writeGuidesIndex(siteRoot); err != nil {
			log.Fatal(err)
		}
		if _, err := writeCheatsheetsIndex(siteRoot); err != nil {
			log.Fatal(err)
		}
		if _, err := writeDotfilesIndex(siteRoot); err != nil {
			log.Fatal(err)
		}
		if _, err := writeBookmarksIndex(siteRoot); err != nil {
			log.Fatal(err)
		}
		if _, err := writeProfilesIndex(siteRoot); err != nil {
			log.Fatal(err)
		}
		if _, err := writeDashboardsIndex(siteRoot); err != nil {
			log.Fatal(err)
		}

		fmt.Println("Wrote all index.json files")
		return
	}

	bindHost, displayHosts, err := resolveHost(*host)
	if err != nil {
		log.Fatal(err)
	}

	fileServer := noCache(http.FileServer(http.Dir(siteRoot)))
	mux := http.NewServeMux()
	mux.HandleFunc("/api/scripts", requireUnlockedContent(siteRoot, scriptsListHandler(siteRoot)))
	mux.HandleFunc("/api/scripts/content", requireUnlockedContent(siteRoot, scriptContentHandler(siteRoot)))
	mux.HandleFunc("/api/scripts/search", requireUnlockedContent(siteRoot, scriptsSearchHandler(siteRoot)))
	mux.HandleFunc("/api/scripts/run", requireUnlockedContent(siteRoot, scriptRunHandler(siteRoot, *allowLanScriptRuns)))
	mux.HandleFunc("/api/server/status", serverStatusHandler(bindHost, displayHosts, *port, siteRoot))
	mux.HandleFunc("/api/server/refresh", serverRefreshHandler(siteRoot))
	mux.HandleFunc("/api/health/links", linkHealthHandler(siteRoot))
	mux.HandleFunc("/api/wiki/doc", requireUnlockedContent(siteRoot, wikiDocHandler(siteRoot)))
	mux.HandleFunc("/api/wiki/search", requireUnlockedContent(siteRoot, wikiSearchHandler(siteRoot)))
	mux.HandleFunc("/wiki-index.json", requireUnlockedContent(siteRoot, markdownIndexHandler(siteRoot)))
	mux.HandleFunc("/api/guides/doc", requireUnlockedContent(siteRoot, guidesDocHandler(siteRoot)))
	mux.HandleFunc("/api/guides/search", requireUnlockedContent(siteRoot, guidesSearchHandler(siteRoot)))
	mux.HandleFunc("/guides-index.json", requireUnlockedContent(siteRoot, guidesIndexHandler(siteRoot)))
	mux.HandleFunc("/api/cheatsheets/doc", requireUnlockedContent(siteRoot, cheatsheetsDocHandler(siteRoot)))
	mux.HandleFunc("/api/cheatsheets/search", requireUnlockedContent(siteRoot, cheatsheetsSearchHandler(siteRoot)))
	mux.HandleFunc("/cheatsheets-index.json", requireUnlockedContent(siteRoot, cheatsheetsIndexHandler(siteRoot)))
	mux.HandleFunc("/api/dotfiles/doc", requireUnlockedContent(siteRoot, dotfilesDocHandler(siteRoot)))
	mux.HandleFunc("/api/dotfiles/search", requireUnlockedContent(siteRoot, dotfilesSearchHandler(siteRoot)))
	mux.HandleFunc("/dotfiles-index.json", requireUnlockedContent(siteRoot, dotfilesIndexHandler(siteRoot)))
	mux.HandleFunc("/api/bookmarks/doc", requireUnlockedContent(siteRoot, bookmarksDocHandler(siteRoot)))
	mux.HandleFunc("/api/bookmarks/search", requireUnlockedContent(siteRoot, bookmarksSearchHandler(siteRoot)))
	mux.HandleFunc("/bookmarks-index.json", requireUnlockedContent(siteRoot, bookmarksIndexHandler(siteRoot)))
	mux.HandleFunc("/api/profiles/doc", requireUnlockedContent(siteRoot, profilesDocHandler(siteRoot)))
	mux.HandleFunc("/api/profiles/search", requireUnlockedContent(siteRoot, profilesSearchHandler(siteRoot)))
	mux.HandleFunc("/profiles-index.json", requireUnlockedContent(siteRoot, profilesIndexHandler(siteRoot)))
	mux.HandleFunc("/api/dashboards/doc", requireUnlockedContent(siteRoot, dashboardsDocHandler(siteRoot)))
	mux.HandleFunc("/api/dashboards/search", requireUnlockedContent(siteRoot, dashboardsSearchHandler(siteRoot)))
	mux.HandleFunc("/dashboards-index.json", requireUnlockedContent(siteRoot, dashboardsIndexHandler(siteRoot)))
	mux.HandleFunc("/api/feeds/reddit", feedRedditHandler(siteRoot))
	mux.HandleFunc("/api/feeds/youtube", feedYoutubeHandler(siteRoot))
	mux.HandleFunc("/api/feeds/podcast", feedPodcastHandler(siteRoot))
	mux.HandleFunc("/api/feeds/spotify", feedSpotifyHandler(siteRoot))
	mux.HandleFunc("/api/feeds/news", feedNewsHandler(siteRoot))
	mux.Handle("/", fileServer)
	address := fmt.Sprintf("%s:%d", bindHost, *port)
	url := fmt.Sprintf("http://%s:%d/", displayHosts[0], *port)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		if isAddressInUse(err) {
			printPortInUseMessage(address, displayHosts, *port)
			os.Exit(1)
		}

		log.Fatal(err)
	}
	defer listener.Close()

	fmt.Println()
	fmt.Println(colorize(ansiBold+ansiCyan, "[Rock-OS]"))
	printStartupStatus(siteRoot, bindHost, address, *allowLanScriptRuns)
	printStatus("OK", ansiGreen, "Open %s", url)
	if len(displayHosts) > 1 {
		fmt.Println("Other local URLs:")
		for _, displayHost := range displayHosts[1:] {
			fmt.Printf("  %s\n", colorize(ansiCyan, fmt.Sprintf("http://%s:%d/", displayHost, *port)))
		}
	}
	fmt.Println()

	if *open {
		if err := openBrowser(url); err != nil {
			log.Printf("Could not open browser automatically: %v", err)
		}
	}

	server := &http.Server{
		Handler: logRequests(compressResponses(rateLimitAPI(mux))),
	}

	shutdownErrors := make(chan error, 1)
	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(signals)

		<-signals
		fmt.Println()
		fmt.Println("Shutting down Rock-OS...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		shutdownErrors <- server.Shutdown(ctx)
	}()

	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}

	select {
	case err := <-shutdownErrors:
		if err != nil {
			log.Fatal(err)
		}
	default:
	}
}

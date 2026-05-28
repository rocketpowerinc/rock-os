package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

func resolveSiteRoot(workingDir string, requestedSiteRoot string) (string, error) {
	if strings.TrimSpace(requestedSiteRoot) != "" {
		return cleanSiteRoot(requestedSiteRoot)
	}

	candidates := []string{
		workingDir,
		filepath.Join(workingDir, "Website"),
		filepath.Join(workingDir, "..", "Website"),
		filepath.Join(workingDir, "..", "..", "Website"),
	}

	seen := map[string]bool{}
	for _, candidate := range candidates {
		absoluteCandidate, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}

		if seen[absoluteCandidate] {
			continue
		}
		seen[absoluteCandidate] = true

		if siteRootLooksValid(absoluteCandidate) {
			return absoluteCandidate, nil
		}
	}

	return "", fmt.Errorf("could not find the Website folder; pass --site-root with the path to Website")
}

func cleanSiteRoot(siteRoot string) (string, error) {
	absoluteSiteRoot, err := filepath.Abs(siteRoot)
	if err != nil {
		return "", err
	}

	if !siteRootLooksValid(absoluteSiteRoot) {
		return "", fmt.Errorf("%s does not look like the Rock-OS Website folder", absoluteSiteRoot)
	}

	return absoluteSiteRoot, nil
}

func siteRootLooksValid(siteRoot string) bool {
	requiredFiles := []string{
		"index.html",
		"wiki.html",
		"guides.html",
		"cheatsheets.html",
		"dotfiles.html",
		"bookmarks.html",
		"scripts.html",
		"profiles.html",
		"dashboards.html",
	}

	for _, file := range requiredFiles {
		info, err := os.Stat(filepath.Join(siteRoot, file))
		if err != nil || info.IsDir() {
			return false
		}
	}

	requiredDirs := []string{
		markdownDir,
		guidesDir,
		cheatsheetsDir,
		dotfilesDir,
		bookmarksDir,
		scriptsDir,
		profilesDir,
		dashboardsDir,
		"css",
		"js",
	}

	for _, dir := range requiredDirs {
		info, err := os.Stat(filepath.Join(siteRoot, dir))
		if err != nil || !info.IsDir() {
			return false
		}
	}

	return true
}

func resolveHost(host string) (string, []string, error) {
	switch strings.ToLower(strings.TrimSpace(host)) {
	case "", "local", "lan", "all", "0.0.0.0":
		localIPs, err := localIPv4s()
		if err != nil {
			return "", nil, err
		}

		return "0.0.0.0", localIPs, nil
	default:
		return host, []string{host}, nil
	}
}

type ipCandidate struct {
	ip    string
	score int
}

func localIPv4s() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	candidates := []ipCandidate{}
	for _, networkInterface := range interfaces {
		if networkInterface.Flags&net.FlagUp == 0 ||
			networkInterface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addresses, err := networkInterface.Addrs()
		if err != nil {
			continue
		}

		for _, address := range addresses {
			ip, _, err := net.ParseCIDR(address.String())
			if err != nil {
				continue
			}

			ip = ip.To4()
			if ip == nil || ip.IsLoopback() {
				continue
			}

			if ip.IsUnspecified() ||
				ip.IsMulticast() ||
				ip.IsLinkLocalUnicast() ||
				!ip.IsPrivate() {
				continue
			}

			candidates = append(candidates, ipCandidate{
				ip:    ip.String(),
				score: localIPScore(ip, networkInterface),
			})
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("could not find a private local IPv4 address")
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	seen := map[string]bool{}
	ips := []string{}
	for _, candidate := range candidates {
		if seen[candidate.ip] {
			continue
		}

		seen[candidate.ip] = true
		ips = append(ips, candidate.ip)
	}

	return ips, nil
}

func localIPScore(ip net.IP, networkInterface net.Interface) int {
	score := 0
	ip4 := ip.To4()

	switch {
	case ip4[0] == 192 && ip4[1] == 168:
		score += 300
	case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
		score += 200
	case ip4[0] == 10:
		score += 100
	}

	interfaceName := strings.ToLower(networkInterface.Name)
	virtualHints := []string{
		"bluetooth",
		"br-",
		"bridge",
		"docker",
		"hyper-v",
		"tailscale",
		"tap",
		"tun",
		"vbox",
		"vether",
		"virtual",
		"vmware",
		"vpn",
		"wsl",
		"zerotier",
	}
	for _, hint := range virtualHints {
		if strings.Contains(interfaceName, hint) {
			score -= 1000
		}
	}

	if networkInterface.Flags&net.FlagPointToPoint != 0 {
		score -= 500
	}

	return score
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

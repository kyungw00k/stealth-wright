// Package browserutil provides browser detection and installation utilities.
package browserutil

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// BrowserInfo contains information about a browser.
type BrowserInfo struct {
	Name    string
	Path    string
	Version string
}

// DetectSystemBrowsers returns available system browsers.
func DetectSystemBrowsers() []BrowserInfo {
	var browsers []BrowserInfo

	// Browser candidates in priority order
	candidates := getBrowserCandidates()

	for _, name := range candidates {
		if path, version := findBrowser(name); path != "" {
			browsers = append(browsers, BrowserInfo{
				Name:    name,
				Path:    path,
				Version: version,
			})
		}
	}

	return browsers
}

// getBrowserCandidates returns browser names to search for.
func getBrowserCandidates() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{
			"chrome",
			"chromium",
			"msedge",
			"firefox",
			"webkit",
		}
	case "linux":
		return []string{
			"chromium",
			"chrome",
			"chromium-browser",
			"msedge",
			"firefox",
			"webkit",
		}
	case "windows":
		return []string{
			"chrome",
			"msedge",
			"chromium",
			"firefox",
			"webkit",
		}
	default:
		return []string{"chromium", "chrome", "firefox", "webkit"}
	}
}

// findBrowser locates a browser and returns its path and version.
func findBrowser(name string) (path string, version string) {
	switch name {
	case "chrome":
		return findChrome()
	case "chromium", "chromium-browser":
		return findChromium()
	case "msedge":
		return findEdge()
	case "firefox":
		return findFirefox()
	case "webkit":
		return findWebKit()
	default:
		return "", ""
	}
}

func findChrome() (string, string) {
	var paths []string
	switch runtime.GOOS {
	case "darwin":
		paths = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		}
	case "linux":
		paths = []string{
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
		}
	case "windows":
		paths = []string{
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		}
	}

	for _, p := range paths {
		if version := getBrowserVersion(p, "--version"); version != "" {
			return p, version
		}
	}

	// Try `which` or `where`
	if path, _ := exec.LookPath("google-chrome"); path != "" {
		if version := getBrowserVersion(path, "--version"); version != "" {
			return path, version
		}
	}

	return "", ""
}

func findChromium() (string, string) {
	var paths []string
	switch runtime.GOOS {
	case "darwin":
		paths = []string{
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
	case "linux":
		paths = []string{
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/usr/bin/chrome",
		}
	case "windows":
		paths = []string{
			`C:\Program Files\Chromium\Application\chrome.exe`,
		}
	}

	for _, p := range paths {
		if version := getBrowserVersion(p, "--version"); version != "" {
			return p, version
		}
	}

	// Try `which`
	for _, cmd := range []string{"chromium", "chromium-browser"} {
		if path, _ := exec.LookPath(cmd); path != "" {
			if version := getBrowserVersion(path, "--version"); version != "" {
				return path, version
			}
		}
	}

	return "", ""
}

func findEdge() (string, string) {
	var paths []string
	switch runtime.GOOS {
	case "darwin":
		paths = []string{
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		}
	case "linux":
		paths = []string{
			"/usr/bin/microsoft-edge",
			"/usr/bin/microsoft-edge-stable",
		}
	case "windows":
		paths = []string{
			`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
			`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		}
	}

	for _, p := range paths {
		if version := getBrowserVersion(p, "--version"); version != "" {
			return p, version
		}
	}

	if path, _ := exec.LookPath("microsoft-edge"); path != "" {
		if version := getBrowserVersion(path, "--version"); version != "" {
			return path, version
		}
	}

	return "", ""
}

func findFirefox() (string, string) {
	var paths []string
	switch runtime.GOOS {
	case "darwin":
		paths = []string{
			"/Applications/Firefox.app/Contents/MacOS/firefox",
		}
	case "linux":
		paths = []string{
			"/usr/bin/firefox",
			"/usr/bin/firefox-esr",
		}
	case "windows":
		paths = []string{
			`C:\Program Files\Mozilla Firefox\firefox.exe`,
			`C:\Program Files (x86)\Mozilla Firefox\firefox.exe`,
		}
	}

	for _, p := range paths {
		if version := getBrowserVersion(p, "--version"); version != "" {
			return p, version
		}
	}

	if path, _ := exec.LookPath("firefox"); path != "" {
		if version := getBrowserVersion(path, "--version"); version != "" {
			return path, version
		}
	}

	return "", ""
}

func findWebKit() (string, string) {
	// WebKit/Safari is typically only available on macOS
	if runtime.GOOS == "darwin" {
		path := "/Applications/Safari.app/Contents/MacOS/Safari"
		if _, err := exec.Command(path, "--version").CombinedOutput(); err == nil {
			return path, "system"
		}
	}

	return "", ""
}

// getBrowserVersion executes the browser with --version flag.
func getBrowserVersion(path string, flag string) string {
	cmd := exec.Command(path, flag)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// GetBestBrowser returns the best available browser for the requested type.
// If browserType is empty, returns the first available browser.
// If no system browser is found, returns empty string (caller should install).
func GetBestBrowser(browserType string) (string, bool) {
	browsers := DetectSystemBrowsers()

	if browserType == "" {
		// Return first available
		if len(browsers) > 0 {
			return mapBrowserType(browsers[0].Name), true
		}
		return "", false
	}

	// Search for requested type
	for _, b := range browsers {
		if mapBrowserType(b.Name) == browserType {
			return browserType, true
		}
	}

	return "", false
}

// mapBrowserType normalizes browser names to playwright types.
func mapBrowserType(name string) string {
	switch name {
	case "chrome", "chromium", "chromium-browser":
		return "chromium"
	case "msedge":
		return "msedge"
	case "firefox", "firefox-esr":
		return "firefox"
	case "webkit", "safari":
		return "webkit"
	default:
		return "chromium"
	}
}

// NeedsInstall checks if playwright browsers need to be installed.
func NeedsInstall(browserType string) bool {
	// Check if system browser is available
	if _, found := GetBestBrowser(browserType); found {
		return false
	}
	return true
}

// InstallCommand returns the command to install playwright browsers.
func InstallCommand(browserType string) string {
	if browserType == "" {
		browserType = "chromium"
	}
	return fmt.Sprintf("go run github.com/playwright-community/playwright-go/cmd/playwright@latest install %s", browserType)
}

// PrintBrowserInfo prints detected browsers for debugging.
func PrintBrowserInfo() {
	browsers := DetectSystemBrowsers()
	if len(browsers) == 0 {
		fmt.Println("No system browsers detected.")
		fmt.Println("Playwright will download browsers automatically on first run.")
		return
	}

	fmt.Println("Detected system browsers:")
	for _, b := range browsers {
		fmt.Printf("  - %s: %s (%s)\n", b.Name, b.Version, b.Path)
	}
}

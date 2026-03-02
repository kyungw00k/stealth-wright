// Package browserutil provides browser detection and installation utilities.
package browserutil

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

// EnsurePlaywrightDriver ensures the Playwright driver is installed.
// It auto-installs if not present.
func EnsurePlaywrightDriver() error {
	// Try to install (it will skip if already installed)
	err := playwright.Install()
	if err == nil {
		return nil
	}

	// Install driver manually
	fmt.Fprintln(os.Stderr, "Installing Playwright driver...")
	return installDriver()
}

func installDriver() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run", "github.com/playwright-community/playwright-go/cmd/playwright@latest", "install-driver")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install Playwright driver: %w", err)
	}

	return nil
}

// EnsureBrowser ensures a browser is available.
// First checks for system browsers, then installs via Playwright if needed.
// Returns the browser type to use.
func EnsureBrowser(browserType string) (string, error) {
	// First, ensure driver is installed
	if err := EnsurePlaywrightDriver(); err != nil {
		return "", err
	}

	// Normalize browser type
	browserType = normalizeBrowserType(browserType)

	// Check for system browser
	if b, found := GetBestBrowser(browserType); found {
		return b, nil
	}

	// No system browser found, install via Playwright
	fmt.Fprintf(os.Stderr, "No %s browser found. Installing via Playwright...\n", browserType)
	if err := InstallPlaywrightBrowser(browserType); err != nil {
		return "", err
	}

	return browserType, nil
}

// InstallPlaywrightBrowser installs a browser via Playwright.
func InstallPlaywrightBrowser(browserType string) error {
	if browserType == "" {
		browserType = "chromium"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	args := []string{"run", "github.com/playwright-community/playwright-go/cmd/playwright@latest", "install"}
	if browserType != "all" {
		args = append(args, browserType)
	}

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install %s: %w", browserType, err)
	}

	return nil
}

// normalizeBrowserType normalizes browser type names.
func normalizeBrowserType(browserType string) string {
	browserType = strings.ToLower(strings.TrimSpace(browserType))

	switch browserType {
	case "chrome", "google-chrome", "googlechrome":
		return "chromium"
	case "edge", "msedge", "microsoft-edge":
		return "msedge"
	case "safari":
		return "webkit"
	case "":
		return "chromium"
	default:
		return browserType
	}
}

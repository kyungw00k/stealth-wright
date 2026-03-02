package browserutil

import (
	"testing"
)

func TestNormalizeBrowserType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"chrome", "chromium"},
		{"google-chrome", "chromium"},
		{"googlechrome", "chromium"},
		{"chromium", "chromium"},
		{"CHROMIUM", "chromium"},
		{"  chromium  ", "chromium"},
		{"edge", "msedge"},
		{"msedge", "msedge"},
		{"microsoft-edge", "msedge"},
		{"safari", "webkit"},
		{"firefox", "firefox"},
		{"", "chromium"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeBrowserType(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeBrowserType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMapBrowserType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"chrome", "chromium"},
		{"chromium", "chromium"},
		{"chromium-browser", "chromium"},
		{"msedge", "msedge"},
		{"firefox", "firefox"},
		{"firefox-esr", "firefox"},
		{"webkit", "webkit"},
		{"safari", "webkit"},
		{"unknown", "chromium"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapBrowserType(tt.input)
			if result != tt.expected {
				t.Errorf("mapBrowserType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetBrowserCandidates(t *testing.T) {
	candidates := getBrowserCandidates()

	if len(candidates) == 0 {
		t.Error("getBrowserCandidates() returned empty list")
	}

	// Should always include chromium
	hasChromium := false
	for _, c := range candidates {
		if c == "chromium" || c == "chrome" {
			hasChromium = true
			break
		}
	}
	if !hasChromium {
		t.Error("getBrowserCandidates() should include chromium or chrome")
	}
}

func TestNeedsInstall(t *testing.T) {
	t.Skip("Skipping - browser detection can be slow")
}


func TestInstallCommand(t *testing.T) {
	cmd := InstallCommand("chromium")
	if cmd == "" {
		t.Error("InstallCommand() returned empty string")
	}

	cmd = InstallCommand("")
	if cmd == "" {
		t.Error("InstallCommand('') returned empty string")
	}
}

func TestGetBestBrowser(t *testing.T) {
	t.Skip("Skipping - browser detection can be slow")
}

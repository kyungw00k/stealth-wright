// Package seleniumbase provides a seleniumbase-go implementation of the browser interface.
package seleniumbase

import (
	"fmt"

	"github.com/kyungw00k/seleniumbase-go/sb"
	"github.com/kyungw00k/sw/internal/browser"
)

// Driver implements browser.Browser using seleniumbase-go.
type Driver struct {
	page    *sb.Page
	cleanup func()
	config  *Config
}

// Config holds driver configuration.
type Config struct {
	Browser        string
	Channel        string
	Headless       bool
	Stealth        bool
	Proxy          string
	UserAgent      string
	ViewportWidth  int
	ViewportHeight int
	UserDataDir    string
	RecordVideo    string
	Device         string
}

// NewDriver creates a new seleniumbase-go driver.
func NewDriver(opts ...Option) (browser.Browser, error) {
	cfg := defaultConfig()
	for _, o := range opts {
		o(cfg)
	}

	// seleniumbase-go handles browser/driver installation internally
	sbOpts := cfg.toSBOptions()

	page, cleanup, err := sb.NewPage(sbOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser: %w", err)
	}

	return &Driver{
		page:    page,
		cleanup: cleanup,
		config:  cfg,
	}, nil
}

// Close closes the browser.
func (d *Driver) Close() error {
	if d.cleanup != nil {
		d.cleanup()
	}
	return nil
}

// NewPage creates a new page.
func (d *Driver) NewPage(opts ...browser.PageOption) (browser.Page, error) {
	// Return the existing page wrapped
	return NewPage(d.page.Playwright()), nil
}

// NewContext creates a new browser context.
func (d *Driver) NewContext(opts ...browser.ContextOption) (browser.Context, error) {
	// For now, return a simple context wrapper
	return &Context{
		pwCtx: d.page.Context(),
	}, nil
}

// Version returns the browser version.
func (d *Driver) Version() string {
	return "chromium"
}

// IsConnected returns whether the browser is connected.
func (d *Driver) IsConnected() bool {
	return d.page != nil
}

// Default config
func defaultConfig() *Config {
	return &Config{
		Browser:        "chromium",
		Headless:       true,
		Stealth:        true,
		ViewportWidth:  1280,
		ViewportHeight: 720,
	}
}

// channelBrowsers are browser values that are actually Playwright channels of chromium.
var channelBrowsers = map[string]bool{
	"chrome": true, "chrome-beta": true, "chrome-dev": true, "chrome-canary": true,
	"msedge": true, "msedge-beta": true, "msedge-dev": true, "msedge-canary": true,
}

// Convert to seleniumbase-go options
func (c *Config) toSBOptions() []sb.Option {
	browser := c.Browser
	channel := c.Channel

	// Map channel names (e.g. "chrome") to browser=chromium + channel=<name>
	if channelBrowsers[browser] {
		channel = browser
		browser = "chromium"
	}

	opts := []sb.Option{
		sb.WithBrowser(browser),
		sb.WithHeadless(c.Headless),
		sb.WithStealth(c.Stealth),
		sb.WithViewportSize(c.ViewportWidth, c.ViewportHeight),
	}

	if channel != "" {
		opts = append(opts, sb.WithChannel(channel))
	}
	if c.Proxy != "" {
		opts = append(opts, sb.WithProxy(c.Proxy))
	}
	if c.UserAgent != "" {
		opts = append(opts, sb.WithUserAgent(c.UserAgent))
	}
	if c.UserDataDir != "" {
		opts = append(opts, sb.WithUserDataDir(c.UserDataDir))
	}
	if c.RecordVideo != "" {
		opts = append(opts, sb.WithRecordVideo(c.RecordVideo))
	}
	if c.Device != "" {
		opts = append(opts, sb.WithDevice(c.Device))
	}

	return opts
}

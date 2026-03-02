package seleniumbase

import (
	"github.com/kyungw00k/sw/internal/browser"
	"github.com/playwright-community/playwright-go"
)

// Option configures the driver.
type Option func(*Config)

// WithBrowser sets the browser type.
func WithBrowser(b string) Option {
	return func(c *Config) { c.Browser = b }
}

// WithChannel sets the browser channel.
func WithChannel(ch string) Option {
	return func(c *Config) { c.Channel = ch }
}

// WithHeadless sets headless mode.
func WithHeadless(h bool) Option {
	return func(c *Config) { c.Headless = h }
}

// WithStealth sets stealth mode.
func WithStealth(s bool) Option {
	return func(c *Config) { c.Stealth = s }
}

// WithProxy sets the proxy server.
func WithProxy(p string) Option {
	return func(c *Config) { c.Proxy = p }
}

// WithUserAgent sets the user agent.
func WithUserAgent(ua string) Option {
	return func(c *Config) { c.UserAgent = ua }
}

// WithViewportSize sets the viewport size.
func WithViewportSize(w, h int) Option {
	return func(c *Config) {
		c.ViewportWidth = w
		c.ViewportHeight = h
	}
}

// WithUserDataDir sets the user data directory.
func WithUserDataDir(dir string) Option {
	return func(c *Config) { c.UserDataDir = dir }
}

// WithRecordVideo sets the video recording directory.
func WithRecordVideo(dir string) Option {
	return func(c *Config) { c.RecordVideo = dir }
}

// Context wraps playwright.BrowserContext to implement browser.Context.
type Context struct {
	pwCtx playwright.BrowserContext
}

// NewPage creates a new page in this context.
func (c *Context) NewPage(opts ...browser.PageOption) (browser.Page, error) {
	page, err := c.pwCtx.NewPage()
	if err != nil {
		return nil, err
	}
	return NewPage(page), nil
}

// Pages returns all pages in this context.
func (c *Context) Pages() []browser.Page {
	pages := c.pwCtx.Pages()
	result := make([]browser.Page, len(pages))
	for i, p := range pages {
		result[i] = NewPage(p)
	}
	return result
}

// Close closes the context.
func (c *Context) Close() error {
	if c.pwCtx != nil {
		c.pwCtx.Close()
	}
	return nil
}

// StorageState returns the current storage state.
func (c *Context) StorageState() (*browser.StorageState, error) {
	state, err := c.pwCtx.StorageState()
	if err != nil {
		return nil, err
	}
	if state == nil {
		return &browser.StorageState{}, nil
	}
	return convertStorageState(*state), nil
}

// SetStorageState sets the storage state.
func (c *Context) SetStorageState(state *browser.StorageState) error {
	if len(state.Cookies) > 0 {
		cookies := make([]playwright.OptionalCookie, len(state.Cookies))
		for i, c := range state.Cookies {
			cookies[i] = playwright.OptionalCookie{
				Name:     c.Name,
				Value:    c.Value,
				Domain:   &c.Domain,
				Path:     &c.Path,
				HttpOnly: &c.HTTPOnly,
				Secure:   &c.Secure,
			}
		}
		c.pwCtx.AddCookies(cookies)
	}
	return nil
}

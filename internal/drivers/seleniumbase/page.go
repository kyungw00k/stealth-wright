package seleniumbase

import (
	"time"

	"github.com/kyungw00k/sw/internal/browser"
	"github.com/playwright-community/playwright-go"
)

// ErrNotImplemented is returned for unimplemented features.
var ErrNotImplemented = browser.ErrNotImplemented

// Page wraps playwright.Page to implement browser.Page.
type Page struct {
	pwPage playwright.Page
}

// NewPage creates a new Page wrapper.
func NewPage(pwPage playwright.Page) *Page {
	return &Page{pwPage: pwPage}
}

// Goto navigates to a URL.
func (p *Page) Goto(url string, opts ...browser.GotoOption) error {
	waitUntil := playwright.WaitUntilStateLoad
	timeout := 30 * time.Second

	for _, o := range opts {
		_ = o
	}

	_, err := p.pwPage.Goto(url, playwright.PageGotoOptions{
		WaitUntil: waitUntil,
		Timeout:   playwright.Float(float64(timeout.Milliseconds())),
	})
	return err
}

// GoBack navigates back.
func (p *Page) GoBack() error {
	_, err := p.pwPage.GoBack()
	return err
}

// GoForward navigates forward.
func (p *Page) GoForward() error {
	_, err := p.pwPage.GoForward()
	return err
}

// Refresh refreshes the page.
func (p *Page) Refresh() error {
	_, err := p.pwPage.Reload()
	return err
}

// URL returns the current URL.
func (p *Page) URL() string {
	return p.pwPage.URL()
}

// Title returns the page title.
func (p *Page) Title() string {
	title, _ := p.pwPage.Title()
	return title
}

// Content returns the page HTML content.
func (p *Page) Content() (string, error) {
	return p.pwPage.Content()
}

// Click clicks an element.
func (p *Page) Click(selector string, opts ...browser.ClickOption) error {
	return p.pwPage.Click(selector)
}

// DblClick double-clicks an element.
func (p *Page) DblClick(selector string, opts ...browser.ClickOption) error {
	return p.pwPage.Dblclick(selector)
}

// Hover hovers over an element.
func (p *Page) Hover(selector string) error {
	return p.pwPage.Hover(selector)
}

// Type types text into an element.
func (p *Page) Type(selector, text string, opts ...browser.TypeOption) error {
	return p.pwPage.Fill(selector, text)
}

// Press presses a key on an element.
func (p *Page) Press(selector, key string) error {
	return p.pwPage.Press(selector, key)
}

// Fill fills text into an element (clears first).
func (p *Page) Fill(selector, text string) error {
	return p.pwPage.Fill(selector, text)
}

// Query finds a single element.
func (p *Page) Query(selector string) (browser.Element, error) {
	loc := p.pwPage.Locator(selector)
	return &Element{locator: loc}, nil
}

// QueryAll finds all matching elements.
func (p *Page) QueryAll(selector string) ([]browser.Element, error) {
	locs, err := p.pwPage.Locator(selector).All()
	if err != nil {
		return nil, err
	}

	elements := make([]browser.Element, len(locs))
	for i, loc := range locs {
		elements[i] = &Element{locator: loc}
	}
	return elements, nil
}

// WaitForSelector waits for an element to appear.
func (p *Page) WaitForSelector(selector string, opts ...browser.WaitOption) (browser.Element, error) {
	state := playwright.WaitForSelectorStateVisible
	for _, o := range opts {
		_ = o
	}

	handle, err := p.pwPage.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{
		State: state,
	})
	if err != nil {
		return nil, err
	}

	return &Element{handle: handle}, nil
}

// Screenshot takes a screenshot.
func (p *Page) Screenshot(opts ...browser.ScreenshotOption) ([]byte, error) {
	fullPage := false
	for _, o := range opts {
		_ = o
	}

	buf, err := p.pwPage.Screenshot(playwright.PageScreenshotOptions{
		FullPage: playwright.Bool(fullPage),
	})
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// Evaluate executes JavaScript.
func (p *Page) Evaluate(script string, args ...any) (any, error) {
	return p.pwPage.Evaluate(script, args...)
}

// Close closes the page.
func (p *Page) Close() error {
	return p.pwPage.Close()
}

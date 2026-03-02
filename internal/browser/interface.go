package browser

import (
	"context"
	"errors"
	"time"
)

// Common errors
var ErrNotImplemented = errors.New("feature not implemented")

// Context for operations
type Ctx = context.Context

// Interfaces

// Browser is the interface for a browser instance
type Browser interface {
	Close() error
	NewPage(opts ...PageOption) (Page, error)
	NewContext(opts ...ContextOption) (Context, error)
	Version() string
	IsConnected() bool
}

// Context represents an isolated browser context
type Context interface {
	NewPage(opts ...PageOption) (Page, error)
	Pages() []Page
	Close() error
	StorageState() (*StorageState, error)
	SetStorageState(state *StorageState) error
}

// Page represents a browser page/tab
type Page interface {
	Goto(url string, opts ...GotoOption) error
	GoBack() error
	GoForward() error
	Refresh() error
	URL() string
	Title() string
	Content() (string, error)
	Click(selector string, opts ...ClickOption) error
	DblClick(selector string, opts ...ClickOption) error
	Hover(selector string) error
	Type(selector, text string, opts ...TypeOption) error
	Press(selector, key string) error
	Fill(selector, text string) error
	Query(selector string) (Element, error)
	QueryAll(selector string) ([]Element, error)
	WaitForSelector(selector string, opts ...WaitOption) (Element, error)
	Screenshot(opts ...ScreenshotOption) ([]byte, error)
	Evaluate(script string, args ...any) (any, error)
	AriaSnapshot() (string, error)
	PDF(opts ...PDFOption) ([]byte, error)
	EvaluateOnElement(selector, script string) (any, error)
	MouseWheel(dx, dy float64) error
	StartTracing(opts ...TracingOption) error
	StopTracing(filename string) error
	Close() error
}

// Element represents a DOM element
type Element interface {
	Click(opts ...ClickOption) error
	Hover() error
	Type(text string, opts ...TypeOption) error
	Fill(text string) error
	TextContent() (string, error)
	InnerText() (string, error)
	GetAttribute(name string) (string, error)
	IsVisible() bool
	IsEnabled() bool
	IsChecked() bool
	BoundingBox() (*Rect, error)
	Screenshot(opts ...ScreenshotOption) ([]byte, error)
}

// Types

type StorageState struct {
	Cookies []Cookie
	Origins []OriginStorage
}

type Cookie struct {
	Name     string
	Value    string
	Domain   string
	Path     string
	Expires  time.Time
	HTTPOnly bool
	Secure   bool
	SameSite string
}

type OriginStorage struct {
	Origin         string
	LocalStorage   []StorageEntry
	SessionStorage []StorageEntry
}

type StorageEntry struct {
	Name  string
	Value string
}

type Rect struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

type Viewport struct {
	Width  int
	Height int
}

type Geolocation struct {
	Latitude  float64
	Longitude float64
	Accuracy  float64
}

// Option types

type PageOption func(*PageOptions)
type ContextOption func(*ContextOptions)
type GotoOption func(*GotoOptions)
type ClickOption func(*ClickOptions)
type TypeOption func(*TypeOptions)
type WaitOption func(*WaitOptions)
type ScreenshotOption func(*ScreenshotOptions)
type PDFOption func(*PDFOptions)
type TracingOption func(*TracingOptions)

// Option structs

type PageOptions struct {
	Viewport    *Viewport
	UserAgent   string
	Locale      string
	Timezone    string
	Geolocation *Geolocation
}

type ContextOptions struct {
	UserAgent   string
	Viewport    *Viewport
	Locale      string
	Timezone    string
	Geolocation *Geolocation
	Permissions []string
}

type GotoOptions struct {
	Timeout   time.Duration
	WaitUntil string
	Referer   string
}

type ClickOptions struct {
	Button     string
	ClickCount int
	Delay      time.Duration
	Timeout    time.Duration
	Force      bool
	Modifiers  []string
}

type TypeOptions struct {
	Delay   time.Duration
	Timeout time.Duration
}

type WaitOptions struct {
	Timeout time.Duration
	State   string
}

type ScreenshotOptions struct {
	Type           string
	Quality        int
	FullPage       bool
	Clip           *Rect
	OmitBackground bool
}

type PDFOptions struct {
	Path            string
	Format          string
	PrintBackground bool
}

type TracingOptions struct {
	Screenshots bool
	Snapshots   bool
}

// Option constructors

func WithViewport(width, height int) PageOption {
	return func(o *PageOptions) {
		o.Viewport = &Viewport{Width: width, Height: height}
	}
}

func WithUserAgent(ua string) PageOption {
	return func(o *PageOptions) {
		o.UserAgent = ua
	}
}

func WithTimeout(d time.Duration) GotoOption {
	return func(o *GotoOptions) {
		o.Timeout = d
	}
}

func WithWaitUntil(state string) GotoOption {
	return func(o *GotoOptions) {
		o.WaitUntil = state
	}
}

func WithButton(button string) ClickOption {
	return func(o *ClickOptions) {
		o.Button = button
	}
}

func WithClickDelay(d time.Duration) ClickOption {
	return func(o *ClickOptions) {
		o.Delay = d
	}
}

func WithTypeDelay(d time.Duration) TypeOption {
	return func(o *TypeOptions) {
		o.Delay = d
	}
}

func WithFullPage() ScreenshotOption {
	return func(o *ScreenshotOptions) {
		o.FullPage = true
	}
}

func WithTracingScreenshots() TracingOption {
	return func(o *TracingOptions) {
		o.Screenshots = true
	}
}

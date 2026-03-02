// Package stealth provides bot detection evasion capabilities.
package stealth

import (
	"math/rand"
	"time"

	"github.com/kyungw00k/sw/internal/browser"
)

// Config holds stealth configuration.
type Config struct {
	// Fingerprint randomization
	UserAgent    string
	Viewport     Viewport
	Screen       Screen
	Timezone     string
	Locale       string
	WebRTC       WebRTCConfig
	Canvas       CanvasConfig
	AudioContext AudioConfig

	// Behavior humanization
	TypingSpeed   time.Duration
	MouseMovement bool
	RandomDelays  bool

	// Detection evasion
	HideWebDriver   bool
	ModifyNavigator bool
	WebGLRenderer   string
}

// Viewport represents browser viewport dimensions.
type Viewport struct {
	Width  int
	Height int
}

// Screen represents screen dimensions.
type Screen struct {
	Width  int
	Height int
}

// WebRTCConfig controls WebRTC behavior.
type WebRTCConfig struct {
	Enabled     bool
	LocalIPMask string
}

// CanvasConfig controls canvas fingerprint noise.
type CanvasConfig struct {
	Noise float64
}

// AudioConfig controls audio fingerprint noise.
type AudioConfig struct {
	Noise float64
}

// Option is a functional option for Config.
type Option func(*Config)

// Module provides stealth capabilities.
type Module struct {
	config Config
	rand   *rand.Rand
}

// NewModule creates a new stealth module.
func NewModule(opts ...Option) *Module {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Module{
		config: cfg,
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// DefaultConfig returns default stealth configuration.
func DefaultConfig() Config {
	return Config{
		UserAgent:       "", // Use browser default
		Viewport:        Viewport{Width: 1920, Height: 1080},
		Screen:          Screen{Width: 1920, Height: 1080},
		Timezone:        "America/New_York",
		Locale:          "en-US",
		WebRTC:          WebRTCConfig{Enabled: false},
		Canvas:          CanvasConfig{Noise: 0.0001},
		AudioContext:    AudioConfig{Noise: 0.0001},
		TypingSpeed:     50 * time.Millisecond,
		MouseMovement:   true,
		RandomDelays:    true,
		HideWebDriver:   true,
		ModifyNavigator: true,
		WebGLRenderer:   "",
	}
}

// WithUserAgent sets custom user agent.
func WithUserAgent(ua string) Option {
	return func(c *Config) {
		c.UserAgent = ua
	}
}

// WithViewport sets viewport dimensions.
func WithViewport(width, height int) Option {
	return func(c *Config) {
		c.Viewport = Viewport{Width: width, Height: height}
	}
}

// WithScreen sets screen dimensions.
func WithScreen(width, height int) Option {
	return func(c *Config) {
		c.Screen = Screen{Width: width, Height: height}
	}
}

// WithTimezone sets timezone.
func WithTimezone(tz string) Option {
	return func(c *Config) {
		c.Timezone = tz
	}
}

// WithLocale sets locale.
func WithLocale(locale string) Option {
	return func(c *Config) {
		c.Locale = locale
	}
}

// WithHideWebDriver sets whether to hide webdriver flag.
func WithHideWebDriver(hide bool) Option {
	return func(c *Config) {
		c.HideWebDriver = hide
	}
}

// WithMouseMovement sets whether to humanize mouse movement.
func WithMouseMovement(enabled bool) Option {
	return func(c *Config) {
		c.MouseMovement = enabled
	}
}

// Apply applies stealth settings to a browser page.
func (m *Module) Apply(page browser.Page) error {
	// Apply WebDriver hiding
	if m.config.HideWebDriver {
		if err := m.hideWebDriver(page); err != nil {
			return err
		}
	}

	// Apply Navigator modifications
	if m.config.ModifyNavigator {
		if err := m.modifyNavigator(page); err != nil {
			return err
		}
	}

	// Apply WebGL modifications
	if m.config.WebGLRenderer != "" {
		if err := m.modifyWebGL(page); err != nil {
			return err
		}
	}

	// Apply Chrome runtime patching
	if err := m.patchChromeRuntime(page); err != nil {
		return err
	}

	return nil
}

// GetConfig returns the current configuration.
func (m *Module) GetConfig() Config {
	return m.config
}

// RandomDelay returns a random delay based on config.
func (m *Module) RandomDelay() time.Duration {
	if !m.config.RandomDelays {
		return 0
	}
	// Random delay between 10-100ms
	return time.Duration(m.rand.Intn(90)+10) * time.Millisecond
}

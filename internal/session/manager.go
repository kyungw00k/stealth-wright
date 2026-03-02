// Package session manages browser sessions.
package session

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/kyungw00k/sw/internal/browser"
	"github.com/kyungw00k/sw/internal/drivers/seleniumbase"
)

// Config represents session configuration.
type Config struct {
	Name        string
	Browser     string
	Headed      bool
	Stealth     bool
	Persistent  bool
	Profile     string
	UserDataDir string
	RecordVideo string
}

// Instance represents a browser session instance.
type Instance struct {
	Config    *Config
	Browser   browser.Browser
	Page      browser.Page
	CreatedAt time.Time
	LastUsed  time.Time
}

// Manager manages browser sessions.
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Instance
	registry *Registry
	baseDir  string
}

// NewManager creates a new session manager.
func NewManager(baseDir string) *Manager {
	return &Manager{
		sessions: make(map[string]*Instance),
		registry: NewRegistry(baseDir),
		baseDir:  baseDir,
	}
}

// GetOrCreate gets an existing session or creates a new one.
func (m *Manager) GetOrCreate(cfg *Config) (*Instance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check existing session
	if inst, ok := m.sessions[cfg.Name]; ok {
		inst.LastUsed = time.Now()
		return inst, nil
	}

	// Create new browser instance
	driverOpts := []seleniumbase.Option{
		seleniumbase.WithBrowser(cfg.Browser),
		seleniumbase.WithHeadless(!cfg.Headed),
		seleniumbase.WithStealth(cfg.Stealth),
	}

	if cfg.Profile != "" {
		driverOpts = append(driverOpts, seleniumbase.WithUserDataDir(cfg.Profile))
	} else if cfg.Persistent {
		profileDir := filepath.Join(m.baseDir, "profiles", fmt.Sprintf("ud-%s", cfg.Name))
		driverOpts = append(driverOpts, seleniumbase.WithUserDataDir(profileDir))
	}
	if cfg.RecordVideo != "" {
		driverOpts = append(driverOpts, seleniumbase.WithRecordVideo(cfg.RecordVideo))
	}

	b, err := seleniumbase.NewDriver(driverOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser: %w", err)
	}

	page, err := b.NewPage()
	if err != nil {
		b.Close()
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	inst := &Instance{
		Config:    cfg,
		Browser:   b,
		Page:      page,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}

	m.sessions[cfg.Name] = inst

	// Save to registry
	m.registry.Save(cfg.Name, cfg)

	return inst, nil
}

// Get gets an existing session.
func (m *Manager) Get(name string) (*Instance, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	inst, ok := m.sessions[name]
	if ok {
		inst.LastUsed = time.Now()
	}
	return inst, ok
}

// Close closes a session.
func (m *Manager) Close(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inst, ok := m.sessions[name]
	if !ok {
		return nil
	}

	if err := inst.Browser.Close(); err != nil {
		return err
	}

	delete(m.sessions, name)
	m.registry.Remove(name)

	return nil
}

// CloseAll closes all sessions.
func (m *Manager) CloseAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for name, inst := range m.sessions {
		if err := inst.Browser.Close(); err != nil {
			lastErr = err
		}
		delete(m.sessions, name)
		m.registry.Remove(name)
	}

	return lastErr
}

// List returns all session names.
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.sessions))
	for name := range m.sessions {
		names = append(names, name)
	}
	return names
}

// ListAll returns all registered sessions (including closed persistent ones).
func (m *Manager) ListAll() []*Config {
	return m.registry.List()
}

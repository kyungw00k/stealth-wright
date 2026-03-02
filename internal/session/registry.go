package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Registry persists session configurations.
type Registry struct {
	mu       sync.RWMutex
	baseDir  string
	sessions map[string]*Config
}

// NewRegistry creates a new registry.
func NewRegistry(baseDir string) *Registry {
	r := &Registry{
		baseDir:  baseDir,
		sessions: make(map[string]*Config),
	}
	r.load()
	return r
}

// Save saves a session configuration.
func (r *Registry) Save(name string, cfg *Config) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.sessions[name] = cfg

	// Ensure directory exists
	sessionsDir := filepath.Join(r.baseDir, "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return err
	}

	// Write to file
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(sessionsDir, name+".json"), data, 0644)
}

// Remove removes a session configuration.
func (r *Registry) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.sessions, name)

	// Remove file
	filename := filepath.Join(r.baseDir, "sessions", name+".json")
	os.Remove(filename)
}

// Get gets a session configuration.
func (r *Registry) Get(name string) (*Config, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cfg, ok := r.sessions[name]
	return cfg, ok
}

// List returns all session configurations.
func (r *Registry) List() []*Config {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cfgs := make([]*Config, 0, len(r.sessions))
	for _, cfg := range r.sessions {
		cfgs = append(cfgs, cfg)
	}
	return cfgs
}

// load loads all session configurations from disk.
func (r *Registry) load() {
	sessionsDir := filepath.Join(r.baseDir, "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if filepath.Ext(name) != ".json" {
			continue
		}

		// Read file
		data, err := os.ReadFile(filepath.Join(sessionsDir, name))
		if err != nil {
			continue
		}

		var cfg Config
		if err := json.Unmarshal(data, &cfg); err != nil {
			continue
		}

		sessionName := name[:len(name)-5] // remove .json
		r.sessions[sessionName] = &cfg
	}
}

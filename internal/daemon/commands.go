package daemon

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/kyungw00k/sw/internal/session"
	"github.com/kyungw00k/sw/internal/snapshot"
	"github.com/kyungw00k/sw/pkg/protocol"
)

// Error definitions
var (
	ErrBrowserNotOpen = errors.New("no browser session open")
	ErrNoSnapshot     = errors.New("no snapshot available")
)

// CommandHandler is a function that handles a command.
type CommandHandler func(params json.RawMessage) (interface{}, error)

// CommandRegistry stores command handlers.
type CommandRegistry struct {
	commands map[string]CommandHandler
}

// NewCommandRegistry creates a new command registry.
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]CommandHandler),
	}
}

// Register registers a command handler.
func (r *CommandRegistry) Register(name string, handler CommandHandler) {
	r.commands[name] = handler
}

// Get gets a command handler.
func (r *CommandRegistry) Get(name string) (CommandHandler, bool) {
	h, ok := r.commands[name]
	return h, ok
}

// registerCommands registers all commands.
func (s *Server) registerCommands() {
	s.commands.Register("open", s.cmdOpen)
	s.commands.Register("close", s.cmdClose)
	s.commands.Register("goto", s.cmdGoto)
	s.commands.Register("go-back", s.cmdGoBack)
	s.commands.Register("go-forward", s.cmdGoForward)
	s.commands.Register("reload", s.cmdReload)
	s.commands.Register("snapshot", s.cmdSnapshot)
	s.commands.Register("click", s.cmdClick)
	s.commands.Register("fill", s.cmdFill)
	s.commands.Register("type", s.cmdType)
	s.commands.Register("press", s.cmdPress)
	s.commands.Register("hover", s.cmdHover)
	s.commands.Register("screenshot", s.cmdScreenshot)
	s.commands.Register("eval", s.cmdEval)
	s.commands.Register("list", s.cmdList)
	s.commands.Register("close-all", s.cmdCloseAll)
	s.commands.Register("ping", s.cmdPing)
}

// requireSession ensures a session is active.
func (s *Server) requireSession() (*session.Instance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.currentSession == nil {
		return nil, ErrBrowserNotOpen
	}
	return s.currentSession, nil
}

// resolveSelector resolves a ref to a selector.
func (s *Server) resolveSelector(ref string) (string, error) {
	// If it starts with 'e' and looks like a ref, resolve it
	if len(ref) > 1 && ref[0] == 'e' {
		s.mu.Lock()
		snap := s.currentSnapshot
		s.mu.Unlock()

		if snap == nil {
			return "", ErrNoSnapshot
		}

		selector, err := snapshot.ResolveRef(snap, ref)
		if err != nil {
			return "", err
		}
		return selector, nil
	}

	// Otherwise treat as a direct selector
	return ref, nil
}

// Command implementations

func (s *Server) cmdOpen(params json.RawMessage) (interface{}, error) {
	var p protocol.OpenParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Close existing session if any
	if s.currentSession != nil {
		s.sessions.Close(s.currentSession.Config.Name)
		s.currentSession = nil
	}

	// Create new session
	cfg := &session.Config{
		Name:    "default",
		Browser: "chromium",
		Headed:  true,
		Stealth: true,
	}

	inst, err := s.sessions.GetOrCreate(cfg)
	if err != nil {
		return nil, err
	}
	s.currentSession = inst

	// Navigate if URL provided
	if p.URL != "" {
		if err := inst.Page.Goto(p.URL); err != nil {
			return nil, err
		}
	}

	// Generate initial snapshot
	snap, err := s.snapshots.Generate(inst.Page)
	if err == nil {
		s.currentSnapshot = snap
	}

	return &protocol.CommandResult{
		Success: true,
		Page: &protocol.PageResult{
			URL:   inst.Page.URL(),
			Title: inst.Page.Title(),
		},
	}, nil
}

func (s *Server) cmdClose(params json.RawMessage) (interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.currentSession == nil {
		return &protocol.CommandResult{Success: true, Message: "no session to close"}, nil
	}

	s.sessions.Close(s.currentSession.Config.Name)
	s.currentSession = nil
	s.currentSnapshot = nil

	return &protocol.CommandResult{Success: true, Message: "session closed"}, nil
}

func (s *Server) cmdGoto(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.GotoParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	if err := inst.Page.Goto(p.URL); err != nil {
		return nil, err
	}

	// Update snapshot
	snap, err := s.snapshots.Generate(inst.Page)
	if err == nil {
		s.mu.Lock()
		s.currentSnapshot = snap
		s.mu.Unlock()
	}

	return &protocol.CommandResult{
		Success: true,
		Page: &protocol.PageResult{
			URL:   inst.Page.URL(),
			Title: inst.Page.Title(),
		},
	}, nil
}

func (s *Server) cmdGoBack(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	if err := inst.Page.GoBack(); err != nil {
		return nil, err
	}

	return &protocol.CommandResult{
		Success: true,
		Page: &protocol.PageResult{
			URL:   inst.Page.URL(),
			Title: inst.Page.Title(),
		},
	}, nil
}

func (s *Server) cmdGoForward(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	if err := inst.Page.GoForward(); err != nil {
		return nil, err
	}

	return &protocol.CommandResult{
		Success: true,
		Page: &protocol.PageResult{
			URL:   inst.Page.URL(),
			Title: inst.Page.Title(),
		},
	}, nil
}

func (s *Server) cmdReload(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	if err := inst.Page.Refresh(); err != nil {
		return nil, err
	}

	return &protocol.CommandResult{
		Success: true,
		Page: &protocol.PageResult{
			URL:   inst.Page.URL(),
			Title: inst.Page.Title(),
		},
	}, nil
}

func (s *Server) cmdSnapshot(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	snap, err := s.snapshots.Generate(inst.Page)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.currentSnapshot = snap
	s.mu.Unlock()

	return snap, nil
}

func (s *Server) cmdClick(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.ClickParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	selector, err := s.resolveSelector(p.Ref)
	if err != nil {
		return nil, err
	}

	if err := inst.Page.Click(selector); err != nil {
		return nil, err
	}

	return &protocol.CommandResult{
		Success: true,
		Page: &protocol.PageResult{
			URL:   inst.Page.URL(),
			Title: inst.Page.Title(),
		},
	}, nil
}

func (s *Server) cmdFill(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.FillParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	selector, err := s.resolveSelector(p.Ref)
	if err != nil {
		return nil, err
	}

	if err := inst.Page.Fill(selector, p.Text); err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true}, nil
}

func (s *Server) cmdType(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.TypeParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	// Type into focused element (use body as fallback)
	if err := inst.Page.Type("body", p.Text); err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true}, nil
}

func (s *Server) cmdPress(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.PressParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	if err := inst.Page.Press("body", p.Key); err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true}, nil
}

func (s *Server) cmdHover(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.HoverParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	selector, err := s.resolveSelector(p.Ref)
	if err != nil {
		return nil, err
	}

	if err := inst.Page.Hover(selector); err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true}, nil
}

func (s *Server) cmdScreenshot(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	_, err = inst.Page.Screenshot()
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{
		Success: true,
		Message: "screenshot saved",
	}, nil
}

func (s *Server) cmdEval(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p struct {
		Script string `json:"script"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	result, err := inst.Page.Evaluate(p.Script)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{
		Success: true,
		Message: fmt.Sprintf("%v", result),
	}, nil
}

func (s *Server) cmdList(params json.RawMessage) (interface{}, error) {
	names := s.sessions.List()
	return names, nil
}

func (s *Server) cmdCloseAll(params json.RawMessage) (interface{}, error) {
	s.mu.Lock()
	s.currentSession = nil
	s.currentSnapshot = nil
	s.mu.Unlock()

	s.sessions.CloseAll()
	return &protocol.CommandResult{Success: true, Message: "all sessions closed"}, nil
}

func (s *Server) cmdPing(params json.RawMessage) (interface{}, error) {
	return map[string]string{"status": "ok"}, nil
}

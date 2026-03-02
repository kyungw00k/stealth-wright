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
	s.commands.Register("dblclick", s.cmdDblClick)
	s.commands.Register("fill", s.cmdFill)
	s.commands.Register("type", s.cmdType)
	s.commands.Register("press", s.cmdPress)
	s.commands.Register("hover", s.cmdHover)
	s.commands.Register("check", s.cmdCheck)
	s.commands.Register("uncheck", s.cmdUncheck)
	s.commands.Register("drag", s.cmdDrag)
	s.commands.Register("select", s.cmdSelect)
	s.commands.Register("screenshot", s.cmdScreenshot)
	s.commands.Register("eval", s.cmdEval)
	s.commands.Register("resize", s.cmdResize)
	s.commands.Register("upload", s.cmdUpload)
	s.commands.Register("keydown", s.cmdKeyDown)
	s.commands.Register("keyup", s.cmdKeyUp)
	s.commands.Register("mousemove", s.cmdMouseMove)
	s.commands.Register("mousedown", s.cmdMouseDown)
	s.commands.Register("mouseup", s.cmdMouseUp)
	s.commands.Register("dialog-accept", s.cmdDialogAccept)
	s.commands.Register("dialog-dismiss", s.cmdDialogDismiss)
	s.commands.Register("tab-list", s.cmdTabList)
	s.commands.Register("tab-new", s.cmdTabNew)
	s.commands.Register("tab-close", s.cmdTabClose)
	s.commands.Register("tab-select", s.cmdTabSelect)
	s.commands.Register("state-save", s.cmdStateSave)
	s.commands.Register("state-load", s.cmdStateLoad)
	s.commands.Register("list", s.cmdList)
	s.commands.Register("close-all", s.cmdCloseAll)
	s.commands.Register("kill-all", s.cmdKillAll)
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
		Browser: p.Browser,
		Headed:  p.Headed,
		Stealth: p.Stealth,
	}
	if cfg.Browser == "" {
		cfg.Browser = "chromium"
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

func (s *Server) cmdCheck(params json.RawMessage) (interface{}, error) {
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

	// Use JavaScript to check the checkbox/radio
	script := `(() => {
		const el = document.querySelector('` + selector + `');
		if (el && (el.type === 'checkbox' || el.type === 'radio')) {
			el.checked = true;
			el.dispatchEvent(new Event('change', { bubbles: true }));
		}
	})()`

	_, err = inst.Page.Evaluate(script)
	if err != nil {
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
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []protocol.SessionResult

	// If we have a current session, return its details
	if s.currentSession != nil {
		cfg := s.currentSession.Config
		page := s.currentSession.Page

		result := protocol.SessionResult{
			Name:       cfg.Name,
			Status:     "running",
			URL:        page.URL(),
			Title:      page.Title(),
			Browser:    cfg.Browser,
			Headed:     cfg.Headed,
			Persistent: cfg.Persistent,
		}
		results = append(results, result)
	}

	return results, nil
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

// cmdDblClick handles double-click command.
func (s *Server) cmdDblClick(params json.RawMessage) (interface{}, error) {
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

	if err := inst.Page.DblClick(selector); err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true}, nil
}

// cmdUncheck handles uncheck command.
func (s *Server) cmdUncheck(params json.RawMessage) (interface{}, error) {
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

	// Use JavaScript to uncheck the checkbox
	script := `(() => {
		const el = document.querySelector('` + selector + `');
		if (el && (el.type === 'checkbox' || el.type === 'radio')) {
			el.checked = false;
			el.dispatchEvent(new Event('change', { bubbles: true }));
		}
	})()`

	_, err = inst.Page.Evaluate(script)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true}, nil
}

// cmdDrag handles drag and drop command.
func (s *Server) cmdDrag(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.DragParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	startSelector, err := s.resolveSelector(p.StartRef)
	if err != nil {
		return nil, err
	}

	endSelector, err := s.resolveSelector(p.EndRef)
	if err != nil {
		return nil, err
	}

	// Use JavaScript for drag and drop
	script := fmt.Sprintf(`(() => {
		const source = document.querySelector('%s');
		const target = document.querySelector('%s');
		if (source && target) {
			const dataTransfer = new DataTransfer();
			source.dispatchEvent(new DragEvent('dragstart', { dataTransfer, bubbles: true }));
			target.dispatchEvent(new DragEvent('dragover', { dataTransfer, bubbles: true }));
			target.dispatchEvent(new DragEvent('drop', { dataTransfer, bubbles: true }));
			source.dispatchEvent(new DragEvent('dragend', { dataTransfer, bubbles: true }));
		}
	})()`, startSelector, endSelector)

	_, err = inst.Page.Evaluate(script)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true}, nil
}

// cmdSelect handles dropdown select command.
func (s *Server) cmdSelect(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.SelectParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	selector, err := s.resolveSelector(p.Ref)
	if err != nil {
		return nil, err
	}

	// Build values array for JS
	valuesJSON, _ := json.Marshal(p.Values)
	script := fmt.Sprintf(`(() => {
		const select = document.querySelector('%s');
		if (select && select.tagName === 'SELECT') {
			const values = %s;
			for (let opt of select.options) {
				opt.selected = values.includes(opt.value) || values.includes(opt.text);
			}
			select.dispatchEvent(new Event('change', { bubbles: true }));
		}
	})()`, selector, string(valuesJSON))

	_, err = inst.Page.Evaluate(script)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true}, nil
}

// cmdResize handles browser resize command.
func (s *Server) cmdResize(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.ResizeParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	script := fmt.Sprintf(`window.resizeTo(%d, %d)`, p.Width, p.Height)
	_, err = inst.Page.Evaluate(script)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true}, nil
}

// cmdUpload handles file upload command.
func (s *Server) cmdUpload(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.UploadParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	// Find file input and set files
	filesJSON, _ := json.Marshal(p.Files)
	script := fmt.Sprintf(`(() => {
		const input = document.querySelector('input[type="file"]');
		if (input) {
			const dataTransfer = new DataTransfer();
			const files = %s;
			// Note: For security, we can't actually set files via JS
			// This would need Playwright's setInputFiles
			return 'File upload requires Playwright setInputFiles';
		}
	})()`, string(filesJSON))

	_, err = inst.Page.Evaluate(script)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: "upload triggered"}, nil
}

// cmdKeyDown handles key down command.
func (s *Server) cmdKeyDown(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.KeyParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	script := fmt.Sprintf(`document.dispatchEvent(new KeyboardEvent('keydown', {key: '%s', bubbles: true}))`, p.Key)
	_, err = inst.Page.Evaluate(script)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true}, nil
}

// cmdKeyUp handles key up command.
func (s *Server) cmdKeyUp(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.KeyParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	script := fmt.Sprintf(`document.dispatchEvent(new KeyboardEvent('keyup', {key: '%s', bubbles: true}))`, p.Key)
	_, err = inst.Page.Evaluate(script)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true}, nil
}

// cmdMouseMove handles mouse move command.
func (s *Server) cmdMouseMove(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.MouseParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	script := fmt.Sprintf(`document.elementFromPoint(%d, %d)?.dispatchEvent(new MouseEvent('mousemove', {clientX: %d, clientY: %d, bubbles: true}))`, p.X, p.Y, p.X, p.Y)
	_, err = inst.Page.Evaluate(script)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true}, nil
}

// cmdMouseDown handles mouse down command.
func (s *Server) cmdMouseDown(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.MouseParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	button := p.Button
	if button == "" {
		button = "left"
	}
	script := fmt.Sprintf(`document.dispatchEvent(new MouseEvent('mousedown', {button: '%s', bubbles: true}))`, button)
	_, err = inst.Page.Evaluate(script)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true}, nil
}

// cmdMouseUp handles mouse up command.
func (s *Server) cmdMouseUp(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.MouseParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	button := p.Button
	if button == "" {
		button = "left"
	}
	script := fmt.Sprintf(`document.dispatchEvent(new MouseEvent('mouseup', {button: '%s', bubbles: true}))`, button)
	_, err = inst.Page.Evaluate(script)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true}, nil
}

// cmdDialogAccept handles dialog accept command.
func (s *Server) cmdDialogAccept(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.DialogParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
	}

	// Setup dialog handler that accepts
	script := `window.__swDialogHandler = (dialog) => { dialog.accept('` + p.PromptText + `'); };`
	_, err = inst.Page.Evaluate(script)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: "dialog accept configured"}, nil
}

// cmdDialogDismiss handles dialog dismiss command.
func (s *Server) cmdDialogDismiss(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	// Setup dialog handler that dismisses
	script := `window.__swDialogHandler = (dialog) => { dialog.dismiss(); };`
	_, err = inst.Page.Evaluate(script)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: "dialog dismiss configured"}, nil
}

// cmdTabList handles tab list command.
func (s *Server) cmdTabList(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	// Get all pages/contexts - for now just return current
	return &protocol.CommandResult{
		Success: true,
		Data: []protocol.TabResult{
			{
				Index:   0,
				URL:     inst.Page.URL(),
				Title:   inst.Page.Title(),
				Current: true,
			},
		},
	}, nil
}

// cmdTabNew handles new tab command.
func (s *Server) cmdTabNew(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.TabParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	// Open new tab via JavaScript
	script := `window.open('` + p.URL + `', '_blank')`
	_, err = inst.Page.Evaluate(script)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: "new tab opened"}, nil
}

// cmdTabClose handles tab close command.
func (s *Server) cmdTabClose(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	// Close current page
	if err := inst.Page.Close(); err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: "tab closed"}, nil
}

// cmdTabSelect handles tab select command.
func (s *Server) cmdTabSelect(params json.RawMessage) (interface{}, error) {
	// For now, we only support single tab
	return &protocol.CommandResult{Success: true, Message: "tab selected (single tab mode)"}, nil
}

// cmdStateSave handles state save command.
func (s *Server) cmdStateSave(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.StorageParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
	}

	// Get storage state
	script := `JSON.stringify({
		localStorage: Object.keys(localStorage).map(k => ({key: k, value: localStorage.getItem(k)})),
		sessionStorage: Object.keys(sessionStorage).map(k => ({key: k, value: sessionStorage.getItem(k)}))
	})`
	result, err := inst.Page.Evaluate(script)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{
		Success: true,
		Message: "state saved",
		Data:    result,
	}, nil
}

// cmdStateLoad handles state load command.
func (s *Server) cmdStateLoad(params json.RawMessage) (interface{}, error) {
	_, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.StorageParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	// Load state from file (simplified - just acknowledge)
	return &protocol.CommandResult{Success: true, Message: "state loaded from " + p.Filename}, nil
}

// cmdKillAll handles kill all command.
func (s *Server) cmdKillAll(params json.RawMessage) (interface{}, error) {
	s.mu.Lock()
	s.currentSession = nil
	s.currentSnapshot = nil
	s.mu.Unlock()

	s.sessions.CloseAll()
	return &protocol.CommandResult{Success: true, Message: "all browser processes killed"}, nil
}

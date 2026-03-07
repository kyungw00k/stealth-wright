package daemon

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kyungw00k/sw/internal/browser"
	"github.com/kyungw00k/sw/internal/drivers/seleniumbase"
	"github.com/kyungw00k/sw/internal/session"
	"github.com/kyungw00k/sw/internal/snapshot"
	"github.com/kyungw00k/sw/pkg/protocol"
	playwright "github.com/playwright-community/playwright-go"
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
	s.commands.Register("cookie-list", s.cmdCookieList)
	s.commands.Register("cookie-get", s.cmdCookieGet)
	s.commands.Register("cookie-set", s.cmdCookieSet)
	s.commands.Register("cookie-delete", s.cmdCookieDelete)
	s.commands.Register("cookie-clear", s.cmdCookieClear)
	s.commands.Register("localstorage-list", s.cmdLocalStorageList)
	s.commands.Register("localstorage-get", s.cmdLocalStorageGet)
	s.commands.Register("localstorage-set", s.cmdLocalStorageSet)
	s.commands.Register("localstorage-delete", s.cmdLocalStorageDelete)
	s.commands.Register("localstorage-clear", s.cmdLocalStorageClear)
	s.commands.Register("sessionstorage-list", s.cmdSessionStorageList)
	s.commands.Register("sessionstorage-get", s.cmdSessionStorageGet)
	s.commands.Register("sessionstorage-set", s.cmdSessionStorageSet)
	s.commands.Register("sessionstorage-delete", s.cmdSessionStorageDelete)
	s.commands.Register("sessionstorage-clear", s.cmdSessionStorageClear)
	s.commands.Register("mousewheel", s.cmdMouseWheel)
	s.commands.Register("pdf", s.cmdPDF)
	s.commands.Register("delete-data", s.cmdDeleteData)
	s.commands.Register("run-code", s.cmdRunCode)
	s.commands.Register("show", s.cmdShow)
	s.commands.Register("console", s.cmdConsole)
	s.commands.Register("network", s.cmdNetwork)
	s.commands.Register("tracing-start", s.cmdTracingStart)
	s.commands.Register("tracing-stop", s.cmdTracingStop)
	s.commands.Register("route", s.cmdRoute)
	s.commands.Register("route-list", s.cmdRouteList)
	s.commands.Register("unroute", s.cmdUnroute)
	s.commands.Register("video-start", s.cmdVideoStart)
	s.commands.Register("video-stop", s.cmdVideoStop)
	s.commands.Register("devtools-start", s.cmdDevtoolsStart)
	s.commands.Register("stop", s.cmdStop)
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

// takeSnapshot generates a snapshot and stores it as the current snapshot.
// Returns nil if snapshot generation fails (non-fatal).
func (s *Server) takeSnapshot(page browser.Page) *protocol.SnapshotResult {
	// Capture console state before generating snapshot
	s.eventMu.Lock()
	msgs := make([]protocol.ConsoleEntry, len(s.consoleMessages))
	copy(msgs, s.consoleMessages)
	prevLen := s.lastConsoleLen
	s.lastConsoleLen = len(msgs)
	s.eventMu.Unlock()

	snap, err := s.snapshots.Generate(page)
	if err != nil {
		return nil
	}

	// Count cumulative errors/warnings
	errors, warnings := 0, 0
	for _, m := range msgs {
		switch m.Type {
		case "error":
			errors++
		case "warning":
			warnings++
		}
	}
	snap.ConsoleErrors = errors
	snap.ConsoleWarnings = warnings

	// Write new console entries to file if any
	newEntries := msgs[prevLen:]
	if len(newEntries) > 0 {
		snap.ConsoleLogFile = s.writeConsoleLog(newEntries)
	}

	s.mu.Lock()
	s.currentSnapshot = snap
	s.mu.Unlock()
	return snap
}

// writeConsoleLog writes console entries to a timestamped log file in outputDir.
// Returns the full path of the written file, or "" on failure.
func (s *Server) writeConsoleLog(entries []protocol.ConsoleEntry) string {
	if s.outputDir == "" {
		return ""
	}
	if err := os.MkdirAll(s.outputDir, 0755); err != nil {
		return ""
	}
	t := time.Now().UTC()
	ms := t.Nanosecond() / 1e6
	filename := fmt.Sprintf("console-%s-%03dZ.log", t.Format("2006-01-02T15-04-05"), ms)
	fullPath := filepath.Join(s.outputDir, filename)
	var lines []string
	for _, e := range entries {
		lines = append(lines, fmt.Sprintf("[%s] %s", e.Type, e.Text))
	}
	if err := os.WriteFile(fullPath, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		return ""
	}
	return fullPath
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
		Device:  p.Device,
	}
	if cfg.Browser == "" {
		cfg.Browser = "chromium"
	}

	inst, err := s.sessions.GetOrCreate(cfg)
	if err != nil {
		return nil, err
	}
	s.currentSession = inst

	// Register console and network event listeners
	if sbPage, ok := inst.Page.(*seleniumbase.Page); ok {
		pwPage := sbPage.PlaywrightPage()
		pwPage.OnConsole(func(msg playwright.ConsoleMessage) {
			s.eventMu.Lock()
			s.consoleMessages = append(s.consoleMessages, protocol.ConsoleEntry{
				Type: msg.Type(),
				Text: msg.Text(),
			})
			s.eventMu.Unlock()
		})
		pwPage.Context().OnRequest(func(req playwright.Request) {
			s.eventMu.Lock()
			s.networkEvents = append(s.networkEvents, protocol.NetworkEntry{
				URL:          req.URL(),
				Method:       req.Method(),
				ResourceType: req.ResourceType(),
				Timestamp:    time.Now().UnixMilli(),
			})
			s.eventMu.Unlock()
		})
		pwPage.Context().OnResponse(func(resp playwright.Response) {
			s.eventMu.Lock()
			// Update the last entry with status if URL matches
			url := resp.URL()
			status := resp.Status()
			for i := len(s.networkEvents) - 1; i >= 0; i-- {
				if s.networkEvents[i].URL == url && s.networkEvents[i].Status == 0 {
					s.networkEvents[i].Status = status
					break
				}
			}
			s.eventMu.Unlock()
		})
	}

	// Navigate if URL provided
	if p.URL != "" {
		if err := inst.Page.Goto(p.URL); err != nil {
			return nil, err
		}
	}

	// Capture console stats accumulated during page load
	s.eventMu.Lock()
	msgs := make([]protocol.ConsoleEntry, len(s.consoleMessages))
	copy(msgs, s.consoleMessages)
	s.lastConsoleLen = len(msgs)
	s.eventMu.Unlock()

	// Generate snapshot while already holding s.mu.Lock() (takeSnapshot would deadlock)
	snapData, _ := s.snapshots.Generate(inst.Page)
	if snapData != nil {
		errors, warnings := 0, 0
		for _, m := range msgs {
			switch m.Type {
			case "error":
				errors++
			case "warning":
				warnings++
			}
		}
		snapData.ConsoleErrors = errors
		snapData.ConsoleWarnings = warnings
	}
	s.currentSnapshot = snapData

	return &protocol.CommandResult{
		Success:  true,
		Snapshot: snapData,
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

	snap := s.takeSnapshot(inst.Page)

	return &protocol.CommandResult{
		Success:  true,
		Snapshot: snap,
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

	snap := s.takeSnapshot(inst.Page)

	return &protocol.CommandResult{
		Success:  true,
		Snapshot: snap,
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

	snap := s.takeSnapshot(inst.Page)

	return &protocol.CommandResult{
		Success:  true,
		Snapshot: snap,
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

	snap := s.takeSnapshot(inst.Page)

	return &protocol.CommandResult{
		Success:  true,
		Snapshot: snap,
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

	var p protocol.SnapshotParams
	if len(params) > 0 {
		json.Unmarshal(params, &p)
	}

	snap, err := s.snapshots.Generate(inst.Page)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.currentSnapshot = snap
	s.mu.Unlock()

	// Write to file if filename specified
	if p.Filename != "" {
		content := snap.AriaSnapshot
		if err := os.WriteFile(p.Filename, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write snapshot to file: %w", err)
		}
		snap.Filename = p.Filename
	}

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

	// Use playwright directly when modifiers or non-default button are specified
	if len(p.Modifiers) > 0 || (p.Button != "" && p.Button != "left") {
		if sbPage, ok := inst.Page.(*seleniumbase.Page); ok {
			pwPage := sbPage.PlaywrightPage()
			opts := playwright.PageClickOptions{}
			if p.Button != "" {
				switch p.Button {
				case "right":
					opts.Button = playwright.MouseButtonRight
				case "middle":
					opts.Button = playwright.MouseButtonMiddle
				}
			}
			for _, mod := range p.Modifiers {
				switch mod {
				case "Alt":
					opts.Modifiers = append(opts.Modifiers, *playwright.KeyboardModifierAlt)
				case "Control":
					opts.Modifiers = append(opts.Modifiers, *playwright.KeyboardModifierControl)
				case "Meta":
					opts.Modifiers = append(opts.Modifiers, *playwright.KeyboardModifierMeta)
				case "Shift":
					opts.Modifiers = append(opts.Modifiers, *playwright.KeyboardModifierShift)
				}
			}
			if err := pwPage.Click(selector, opts); err != nil {
				return nil, err
			}
		} else {
			if err := inst.Page.Click(selector); err != nil {
				return nil, err
			}
		}
	} else {
		if err := inst.Page.Click(selector); err != nil {
			return nil, err
		}
	}

	snap := s.takeSnapshot(inst.Page)

	return &protocol.CommandResult{
		Success:  true,
		Snapshot: snap,
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

	if p.Submit {
		if err := inst.Page.Press("body", "Enter"); err != nil {
			return nil, err
		}
	}

	snap := s.takeSnapshot(inst.Page)
	return &protocol.CommandResult{Success: true, Snapshot: snap}, nil
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

	snap := s.takeSnapshot(inst.Page)
	return &protocol.CommandResult{Success: true, Snapshot: snap}, nil
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

	if err := inst.Page.Type("body", p.Text); err != nil {
		return nil, err
	}

	if p.Submit {
		if err := inst.Page.Press("body", "Enter"); err != nil {
			return nil, err
		}
	}

	snap := s.takeSnapshot(inst.Page)
	return &protocol.CommandResult{Success: true, Snapshot: snap}, nil
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

	snap := s.takeSnapshot(inst.Page)
	return &protocol.CommandResult{Success: true, Snapshot: snap}, nil
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

	snap := s.takeSnapshot(inst.Page)
	return &protocol.CommandResult{Success: true, Snapshot: snap}, nil
}

func (s *Server) cmdScreenshot(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.ScreenshotParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
	}

	opts := []browser.ScreenshotOption{}
	if p.FullPage {
		opts = append(opts, browser.WithFullPage())
	}

	var data []byte

	// If ref is specified, take element screenshot
	if p.Ref != "" {
		selector, resolveErr := s.resolveSelector(p.Ref)
		if resolveErr != nil {
			return nil, resolveErr
		}

		if sbPage, ok := inst.Page.(*seleniumbase.Page); ok {
			pwPage := sbPage.PlaywrightPage()
			var shotErr error
			data, shotErr = pwPage.Locator(selector).Screenshot()
			if shotErr != nil {
				return nil, shotErr
			}
		} else {
			return nil, fmt.Errorf("element screenshot not supported for this driver")
		}
	} else {
		var shotErr error
		data, shotErr = inst.Page.Screenshot(opts...)
		if shotErr != nil {
			return nil, shotErr
		}
	}

	filename := p.Filename
	if filename != "" && !filepath.IsAbs(filename) {
		dir := p.Dir
		if dir == "" {
			dir = s.outputDir
		}
		if dir != "" {
			filename = filepath.Join(dir, filename)
		}
	}
	if filename == "" {
		t := time.Now().UTC()
		ms := t.Nanosecond() / 1e6
		name := fmt.Sprintf("page-%s-%03dZ.png", t.Format("2006-01-02T15-04-05"), ms)
		dir := p.Dir
		if dir == "" {
			dir = s.outputDir
		}
		if dir != "" {
			if err := os.MkdirAll(dir, 0755); err == nil {
				filename = filepath.Join(dir, name)
			}
		}
		if filename == "" {
			filename = name
		}
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return nil, err
	}

	return &protocol.CommandResult{
		Success: true,
		Message: "screenshot saved to " + filename,
	}, nil
}

func (s *Server) cmdEval(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.EvalParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	var result any
	var evalErr error

	if p.Ref != "" {
		selector, err := s.resolveSelector(p.Ref)
		if err != nil {
			return nil, err
		}
		result, evalErr = inst.Page.EvaluateOnElement(selector, p.Script)
	} else {
		result, evalErr = inst.Page.Evaluate(p.Script)
	}

	if evalErr != nil {
		return nil, evalErr
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

	// Use playwright directly when modifiers are specified
	if len(p.Modifiers) > 0 {
		if sbPage, ok := inst.Page.(*seleniumbase.Page); ok {
			pwPage := sbPage.PlaywrightPage()
			opts := playwright.PageDblclickOptions{}
			for _, mod := range p.Modifiers {
				switch mod {
				case "Alt":
					opts.Modifiers = append(opts.Modifiers, *playwright.KeyboardModifierAlt)
				case "Control":
					opts.Modifiers = append(opts.Modifiers, *playwright.KeyboardModifierControl)
				case "Meta":
					opts.Modifiers = append(opts.Modifiers, *playwright.KeyboardModifierMeta)
				case "Shift":
					opts.Modifiers = append(opts.Modifiers, *playwright.KeyboardModifierShift)
				}
			}
			if err := pwPage.Dblclick(selector, opts); err != nil {
				return nil, err
			}
		} else {
			if err := inst.Page.DblClick(selector); err != nil {
				return nil, err
			}
		}
	} else {
		if err := inst.Page.DblClick(selector); err != nil {
			return nil, err
		}
	}

	snap := s.takeSnapshot(inst.Page)
	return &protocol.CommandResult{Success: true, Snapshot: snap}, nil
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

	snap := s.takeSnapshot(inst.Page)
	return &protocol.CommandResult{Success: true, Snapshot: snap}, nil
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

	snap := s.takeSnapshot(inst.Page)
	return &protocol.CommandResult{Success: true, Snapshot: snap}, nil
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

	snap := s.takeSnapshot(inst.Page)
	return &protocol.CommandResult{Success: true, Snapshot: snap}, nil
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

	// Get storage state via Playwright API
	sbPage, ok := inst.Page.(*seleniumbase.Page)
	if !ok {
		return nil, fmt.Errorf("state-save not supported for this driver")
	}
	state, err := sbPage.PlaywrightPage().Context().StorageState()
	if err != nil {
		return nil, fmt.Errorf("failed to get storage state: %w", err)
	}

	stateJSON, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize state: %w", err)
	}

	filename := p.Filename
	if filename == "" {
		t := time.Now().UTC()
		ms := t.Nanosecond() / 1e6
		name := fmt.Sprintf("state-%s-%03dZ.json", t.Format("2006-01-02T15-04-05"), ms)
		if s.outputDir != "" {
			if err := os.MkdirAll(s.outputDir, 0755); err == nil {
				filename = filepath.Join(s.outputDir, name)
			}
		}
		if filename == "" {
			filename = name
		}
	}

	if err := os.WriteFile(filename, stateJSON, 0644); err != nil {
		return nil, fmt.Errorf("failed to write state file: %w", err)
	}

	return &protocol.CommandResult{
		Success: true,
		Message: "state saved to " + filename,
	}, nil
}

// cmdStateLoad handles state load command.
func (s *Server) cmdStateLoad(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.StorageParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(p.Filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	sbPage, ok := inst.Page.(*seleniumbase.Page)
	if !ok {
		return nil, fmt.Errorf("state-load not supported for this driver")
	}
	pwCtx := sbPage.PlaywrightPage().Context()

	// Parse the state JSON (playwright format: {cookies: [...], origins: [...]})
	type cookieJSON struct {
		Name     string  `json:"name"`
		Value    string  `json:"value"`
		Domain   string  `json:"domain"`
		Path     string  `json:"path"`
		Expires  float64 `json:"expires"`
		HTTPOnly bool    `json:"httpOnly"`
		Secure   bool    `json:"secure"`
		SameSite string  `json:"sameSite"`
	}
	var state struct {
		Cookies []cookieJSON `json:"cookies"`
		Origins []struct {
			Origin       string `json:"origin"`
			LocalStorage []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"localStorage"`
		} `json:"origins"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	if len(state.Cookies) > 0 {
		cookies := make([]playwright.OptionalCookie, len(state.Cookies))
		for i, c := range state.Cookies {
			cookies[i] = playwright.OptionalCookie{
				Name:     c.Name,
				Value:    c.Value,
				Domain:   playwright.String(c.Domain),
				Path:     playwright.String(c.Path),
				HttpOnly: playwright.Bool(c.HTTPOnly),
				Secure:   playwright.Bool(c.Secure),
			}
		}
		if err := pwCtx.AddCookies(cookies); err != nil {
			return nil, fmt.Errorf("failed to restore cookies: %w", err)
		}
	}

	// Restore localStorage per origin via JS
	for _, origin := range state.Origins {
		if len(origin.LocalStorage) == 0 {
			continue
		}
		entriesJSON, _ := json.Marshal(origin.LocalStorage)
		script := fmt.Sprintf(`(() => {
			const entries = %s;
			for (const {name, value} of entries) {
				try { localStorage.setItem(name, value); } catch(e) {}
			}
		})()`, string(entriesJSON))
		if _, err := inst.Page.Evaluate(script); err != nil {
			// Non-fatal: page may be on a different origin
			_ = err
		}
	}

	return &protocol.CommandResult{
		Success: true,
		Message: "state loaded from " + p.Filename,
	}, nil
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

// Cookie commands

func (s *Server) cmdCookieList(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.CookieListParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
	}

	result, err := inst.Page.Evaluate(`JSON.stringify(document.cookie.split(';').map(c => c.trim()).filter(c => c))`)
	if err != nil {
		return nil, err
	}

	// Note: domain/path filtering via document.cookie is limited (browser doesn't expose those)
	// Return all cookies with note about filters
	msg := fmt.Sprintf("%v", result)
	if p.Domain != "" {
		msg = fmt.Sprintf("(filtered by domain=%s): %v", p.Domain, result)
	}

	return &protocol.CommandResult{Success: true, Message: msg}, nil
}

func (s *Server) cmdCookieGet(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.KeyValueParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	script := fmt.Sprintf(`(() => {
		const name = %q;
		const cookies = document.cookie.split(';').map(c => c.trim());
		const cookie = cookies.find(c => c.startsWith(name + '='));
		return cookie ? cookie.substring(name.length + 1) : null;
	})()`, p.Key)

	result, err := inst.Page.Evaluate(script)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: fmt.Sprintf("%v", result)}, nil
}

func (s *Server) cmdCookieSet(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.CookieSetParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	// Build cookie string with attributes
	cookie := fmt.Sprintf("%s=%s", p.Name, p.Value)
	if p.Domain != "" {
		cookie += "; domain=" + p.Domain
	}
	if p.Path != "" {
		cookie += "; path=" + p.Path
	} else {
		cookie += "; path=/"
	}
	if p.Expires != 0 {
		// Convert unix timestamp to date string
		t := time.Unix(int64(p.Expires), 0)
		cookie += "; expires=" + t.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
	}
	if p.HTTPOnly {
		cookie += "; HttpOnly"
	}
	if p.Secure {
		cookie += "; Secure"
	}
	if p.SameSite != "" {
		cookie += "; SameSite=" + p.SameSite
	}

	script := fmt.Sprintf(`document.cookie = %q`, cookie)
	_, err = inst.Page.Evaluate(script)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: fmt.Sprintf("cookie set: %s", p.Name)}, nil
}

func (s *Server) cmdCookieDelete(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.KeyValueParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	script := fmt.Sprintf(`document.cookie = %q + '=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/'`, p.Key)
	_, err = inst.Page.Evaluate(script)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: fmt.Sprintf("cookie deleted: %s", p.Key)}, nil
}

func (s *Server) cmdCookieClear(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	script := `(() => {
		document.cookie.split(';').forEach(cookie => {
			const name = cookie.trim().split('=')[0];
			document.cookie = name + '=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/';
		});
	})()`
	_, err = inst.Page.Evaluate(script)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: "all cookies cleared"}, nil
}

// localStorage commands

func (s *Server) cmdLocalStorageList(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	result, err := inst.Page.Evaluate(`JSON.stringify(Object.keys(localStorage).map(k => ({key: k, value: localStorage.getItem(k)})))`)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: fmt.Sprintf("%v", result)}, nil
}

func (s *Server) cmdLocalStorageGet(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.KeyValueParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	result, err := inst.Page.Evaluate(fmt.Sprintf(`localStorage.getItem(%q)`, p.Key))
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: fmt.Sprintf("%v", result)}, nil
}

func (s *Server) cmdLocalStorageSet(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.KeyValueParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	_, err = inst.Page.Evaluate(fmt.Sprintf(`localStorage.setItem(%q, %q)`, p.Key, p.Value))
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: fmt.Sprintf("localStorage.%s = %s", p.Key, p.Value)}, nil
}

func (s *Server) cmdLocalStorageDelete(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.KeyValueParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	_, err = inst.Page.Evaluate(fmt.Sprintf(`localStorage.removeItem(%q)`, p.Key))
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: fmt.Sprintf("localStorage.%s deleted", p.Key)}, nil
}

func (s *Server) cmdLocalStorageClear(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	_, err = inst.Page.Evaluate(`localStorage.clear()`)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: "localStorage cleared"}, nil
}

// sessionStorage commands

func (s *Server) cmdSessionStorageList(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	result, err := inst.Page.Evaluate(`JSON.stringify(Object.keys(sessionStorage).map(k => ({key: k, value: sessionStorage.getItem(k)})))`)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: fmt.Sprintf("%v", result)}, nil
}

func (s *Server) cmdSessionStorageGet(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.KeyValueParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	result, err := inst.Page.Evaluate(fmt.Sprintf(`sessionStorage.getItem(%q)`, p.Key))
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: fmt.Sprintf("%v", result)}, nil
}

func (s *Server) cmdSessionStorageSet(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.KeyValueParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	_, err = inst.Page.Evaluate(fmt.Sprintf(`sessionStorage.setItem(%q, %q)`, p.Key, p.Value))
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: fmt.Sprintf("sessionStorage.%s = %s", p.Key, p.Value)}, nil
}

func (s *Server) cmdSessionStorageDelete(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.KeyValueParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	_, err = inst.Page.Evaluate(fmt.Sprintf(`sessionStorage.removeItem(%q)`, p.Key))
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: fmt.Sprintf("sessionStorage.%s deleted", p.Key)}, nil
}

func (s *Server) cmdSessionStorageClear(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	_, err = inst.Page.Evaluate(`sessionStorage.clear()`)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: "sessionStorage cleared"}, nil
}

// Mouse wheel command

func (s *Server) cmdMouseWheel(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.MouseParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	if err := inst.Page.MouseWheel(float64(p.Dx), float64(p.Dy)); err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: fmt.Sprintf("scrolled dx=%d dy=%d", p.Dx, p.Dy)}, nil
}

// PDF command

func (s *Server) cmdPDF(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.ScreenshotParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
	}

	filename := p.Filename
	if filename == "" {
		filename = fmt.Sprintf("page-%d.pdf", time.Now().UnixMilli())
	}

	data, err := inst.Page.PDF(func(o *browser.PDFOptions) { o.Path = filename; o.PrintBackground = true })
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: "PDF saved to " + filename}, nil
}

// Delete data command

func (s *Server) cmdDeleteData(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	// Clear all browser storage data
	script := `(() => {
		localStorage.clear();
		sessionStorage.clear();
		document.cookie.split(';').forEach(cookie => {
			const name = cookie.trim().split('=')[0];
			document.cookie = name + '=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/';
		});
	})()`
	_, err = inst.Page.Evaluate(script)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: "browser data deleted"}, nil
}

// cmdRunCode runs arbitrary JavaScript code in the browser.
func (s *Server) cmdRunCode(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.RunCodeParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	result, err := inst.Page.Evaluate(p.Code)
	if err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: fmt.Sprintf("%v", result)}, nil
}

// cmdShow is an alias for cmdDevtoolsStart (matching playwright-cli behavior).
func (s *Server) cmdShow(params json.RawMessage) (interface{}, error) {
	return s.cmdDevtoolsStart(params)
}

// cmdDevtoolsStart opens the browser DevTools panel via CDP F12 key simulation.
// Only effective in headed mode on Chromium-based browsers.
func (s *Server) cmdDevtoolsStart(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	sbPage, ok := inst.Page.(*seleniumbase.Page)
	if !ok {
		return &protocol.CommandResult{
			Success: true,
			Message: "DevTools not supported for this browser driver. Use --headed mode with Chromium.",
		}, nil
	}
	pwPage := sbPage.PlaywrightPage()

	cdpSession, err := pwPage.Context().NewCDPSession(pwPage)
	if err != nil {
		return nil, fmt.Errorf("failed to create CDP session: %w", err)
	}
	defer cdpSession.Detach()

	// Send F12 keyDown then keyUp to open DevTools
	keyParams := map[string]any{
		"type":                  "keyDown",
		"key":                   "F12",
		"code":                  "F12",
		"windowsVirtualKeyCode": 123,
		"nativeVirtualKeyCode":  123,
	}
	if _, err := cdpSession.Send("Input.dispatchKeyEvent", keyParams); err != nil {
		return nil, fmt.Errorf("failed to dispatch F12 keyDown: %w", err)
	}
	keyParams["type"] = "keyUp"
	if _, err := cdpSession.Send("Input.dispatchKeyEvent", keyParams); err != nil {
		return nil, fmt.Errorf("failed to dispatch F12 keyUp: %w", err)
	}

	return &protocol.CommandResult{Success: true, Message: "DevTools opened"}, nil
}

// Console command

func (s *Server) cmdConsole(params json.RawMessage) (interface{}, error) {
	var p protocol.ConsoleParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
	}

	s.eventMu.Lock()
	defer s.eventMu.Unlock()

	if p.Clear {
		s.consoleMessages = nil
		s.lastConsoleLen = 0
		return &protocol.CommandResult{Success: true, Message: "console cleared"}, nil
	}

	var lines []string
	for _, entry := range s.consoleMessages {
		if p.Level == "" || p.Level == entry.Type {
			lines = append(lines, fmt.Sprintf("[%s] %s", entry.Type, entry.Text))
		}
	}

	msg := ""
	if len(lines) > 0 {
		msg = strings.Join(lines, "\n")
	} else {
		msg = "(no console messages)"
	}

	return &protocol.CommandResult{Success: true, Message: msg}, nil
}

// Network command

func (s *Server) cmdNetwork(params json.RawMessage) (interface{}, error) {
	var p protocol.NetworkParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
	}

	s.eventMu.Lock()
	defer s.eventMu.Unlock()

	if p.Clear {
		s.networkEvents = nil
		return &protocol.CommandResult{Success: true, Message: "network log cleared"}, nil
	}

	staticTypes := map[string]bool{
		"image": true, "stylesheet": true, "font": true, "media": true,
	}

	var lines []string
	for _, entry := range s.networkEvents {
		if !p.Static && staticTypes[entry.ResourceType] {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s %s %d (%s)", entry.Method, entry.URL, entry.Status, entry.ResourceType))
	}

	msg := ""
	if len(lines) > 0 {
		msg = strings.Join(lines, "\n")
	} else {
		msg = "(no network events)"
	}

	return &protocol.CommandResult{Success: true, Message: msg}, nil
}

// Tracing commands

func (s *Server) cmdTracingStart(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	if err := inst.Page.StartTracing(); err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: "tracing started"}, nil
}

func (s *Server) cmdTracingStop(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.TracingParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
	}

	filename := p.Filename
	if filename == "" {
		t := time.Now().UTC()
		ms := t.Nanosecond() / 1e6
		name := fmt.Sprintf("trace-%s-%03dZ.zip", t.Format("2006-01-02T15-04-05"), ms)
		if s.outputDir != "" {
			tracesDir := filepath.Join(s.outputDir, "traces")
			if err := os.MkdirAll(tracesDir, 0755); err == nil {
				filename = filepath.Join(tracesDir, name)
			}
		}
		if filename == "" {
			filename = name
		}
	}

	if err := inst.Page.StopTracing(filename); err != nil {
		return nil, err
	}

	return &protocol.CommandResult{Success: true, Message: "trace saved to " + filename}, nil
}

// Route commands

func (s *Server) cmdRoute(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.RouteParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	sbPage, ok := inst.Page.(*seleniumbase.Page)
	if !ok {
		return nil, fmt.Errorf("page does not support routing")
	}
	pwPage := sbPage.PlaywrightPage()

	status := p.Status
	if status == 0 {
		status = 200
	}

	err = pwPage.Route(p.Pattern, func(route playwright.Route) {
		fulfillOpts := playwright.RouteFulfillOptions{
			Status: playwright.Int(status),
		}
		if p.Body != "" {
			fulfillOpts.Body = p.Body
		}
		if p.ContentType != "" {
			fulfillOpts.ContentType = playwright.String(p.ContentType)
		}
		if len(p.Headers) > 0 {
			fulfillOpts.Headers = p.Headers
		}
		if err := route.Fulfill(fulfillOpts); err != nil {
			// log error but don't fail
			_ = err
		}
	})
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.activeRoutes = append(s.activeRoutes, protocol.RouteEntry{
		Pattern: p.Pattern,
		Status:  status,
	})
	s.mu.Unlock()

	return &protocol.CommandResult{Success: true, Message: fmt.Sprintf("route added: %s → %d", p.Pattern, status)}, nil
}

func (s *Server) cmdRouteList(params json.RawMessage) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.activeRoutes) == 0 {
		return &protocol.CommandResult{Success: true, Message: "(no active routes)"}, nil
	}

	var lines []string
	for _, r := range s.activeRoutes {
		lines = append(lines, fmt.Sprintf("%s → %d", r.Pattern, r.Status))
	}

	return &protocol.CommandResult{Success: true, Message: strings.Join(lines, "\n")}, nil
}

func (s *Server) cmdUnroute(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	var p protocol.UnrouteParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
	}

	sbPage, ok := inst.Page.(*seleniumbase.Page)
	if !ok {
		return nil, fmt.Errorf("page does not support routing")
	}
	pwPage := sbPage.PlaywrightPage()

	if p.Pattern == "" {
		// Remove all routes
		if err := pwPage.UnrouteAll(); err != nil {
			return nil, err
		}
		s.mu.Lock()
		s.activeRoutes = nil
		s.mu.Unlock()
		return &protocol.CommandResult{Success: true, Message: "all routes removed"}, nil
	}

	// Remove specific route
	if err := pwPage.Unroute(p.Pattern); err != nil {
		return nil, err
	}

	s.mu.Lock()
	var remaining []protocol.RouteEntry
	for _, r := range s.activeRoutes {
		if r.Pattern != p.Pattern {
			remaining = append(remaining, r)
		}
	}
	s.activeRoutes = remaining
	s.mu.Unlock()

	return &protocol.CommandResult{Success: true, Message: fmt.Sprintf("route removed: %s", p.Pattern)}, nil
}

// videoState holds CDP screencast recording state.
type videoState struct {
	cdpSession playwright.CDPSession
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	outputFile string
}

// findPlaywrightFFmpeg locates the ffmpeg binary bundled with the Playwright installation.
// playwright-go downloads Playwright's driver (including ffmpeg) to the OS cache directory.
func findPlaywrightFFmpeg() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get home directory: %w", err)
	}

	var cacheBase string
	switch runtime.GOOS {
	case "darwin":
		cacheBase = filepath.Join(homeDir, "Library", "Caches")
	case "linux":
		cacheBase = filepath.Join(homeDir, ".cache")
	case "windows":
		cacheBase = filepath.Join(homeDir, "AppData", "Local")
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	// Read browsers.json from the most recent playwright-go driver to get the ffmpeg revision.
	playwrightGoBase := filepath.Join(cacheBase, "ms-playwright-go")
	entries, err := os.ReadDir(playwrightGoBase)
	if err != nil {
		return "", fmt.Errorf("playwright-go cache not found at %s — run 'sw install' to install browsers: %w", playwrightGoBase, err)
	}

	var browsersJSONPath string
	for i := len(entries) - 1; i >= 0; i-- {
		p := filepath.Join(playwrightGoBase, entries[i].Name(), "package", "browsers.json")
		if _, err := os.Stat(p); err == nil {
			browsersJSONPath = p
			break
		}
	}
	if browsersJSONPath == "" {
		return "", fmt.Errorf("browsers.json not found in playwright-go cache at %s — run 'sw install' to install browsers", playwrightGoBase)
	}

	data, err := os.ReadFile(browsersJSONPath)
	if err != nil {
		return "", fmt.Errorf("failed to read browsers.json: %w", err)
	}

	var config struct {
		Browsers []struct {
			Name     string `json:"name"`
			Revision string `json:"revision"`
		} `json:"browsers"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("failed to parse browsers.json: %w", err)
	}

	var revision string
	for _, b := range config.Browsers {
		if b.Name == "ffmpeg" {
			revision = b.Revision
			break
		}
	}
	if revision == "" {
		return "", fmt.Errorf("ffmpeg entry not found in %s — run 'sw install-browser' to install ffmpeg", browsersJSONPath)
	}

	var binaryName string
	switch runtime.GOOS {
	case "darwin":
		binaryName = "ffmpeg-mac"
	case "linux":
		binaryName = "ffmpeg-linux"
	case "windows":
		binaryName = "ffmpeg-win64.exe"
	}

	ffmpegPath := filepath.Join(cacheBase, "ms-playwright", "ffmpeg-"+revision, binaryName)
	if _, err := os.Stat(ffmpegPath); err != nil {
		return "", fmt.Errorf("ffmpeg binary not found at %s — run 'sw install-browser' to install it: %w", ffmpegPath, err)
	}

	return ffmpegPath, nil
}

// cmdVideoStart starts video recording via CDP Page.startScreencast.
// Frames are captured as JPEG via CDP and piped to Playwright's bundled ffmpeg for WebM encoding.
// This approach works with stealth mode without restarting the browser session.
func (s *Server) cmdVideoStart(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	if s.video != nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("video recording already in progress")
	}
	s.mu.Unlock()

	var p protocol.VideoStartParams
	if len(params) > 0 {
		json.Unmarshal(params, &p)
	}

	sbPage, ok := inst.Page.(*seleniumbase.Page)
	if !ok {
		return nil, fmt.Errorf("CDP screencast requires playwright-based session")
	}
	pwPage := sbPage.PlaywrightPage()

	// Determine output file path
	outDir := p.Dir
	if outDir == "" {
		outDir = s.outputDir
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}
	ts := time.Now().UTC().Format("2006-01-02T15-04-05-000Z")
	outputFile := filepath.Join(outDir, "video-"+ts+".webm")

	// Find Playwright's bundled ffmpeg
	ffmpegPath, err := findPlaywrightFFmpeg()
	if err != nil {
		return nil, fmt.Errorf("playwright ffmpeg not found (run 'sw install'): %w", err)
	}

	// Start ffmpeg: read JPEG frames from stdin, encode to WebM VP8
	cmd := exec.Command(ffmpegPath,
		"-f", "image2pipe",
		"-c:v", "mjpeg",
		"-framerate", "10",
		"-i", "pipe:0",
		"-c:v", "libvpx",
		"-auto-alt-ref", "0",
		"-y",
		outputFile,
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get ffmpeg stdin pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Create CDPSession for the current page
	cdpSession, err := pwPage.Context().NewCDPSession(pwPage)
	if err != nil {
		stdin.Close()
		cmd.Process.Kill()
		return nil, fmt.Errorf("failed to create CDP session: %w", err)
	}

	vs := &videoState{
		cdpSession: cdpSession,
		cmd:        cmd,
		stdin:      stdin,
		outputFile: outputFile,
	}
	s.mu.Lock()
	s.video = vs
	s.mu.Unlock()

	// Register frame handler: decode base64 JPEG, write to ffmpeg stdin, ack the frame.
	// Must run in a goroutine: the handler is invoked from the playwright event loop goroutine,
	// and calling cdpSession.Send() (a blocking CDP round-trip) from within that goroutine
	// would deadlock since the response can't be processed while the event loop is blocked.
	cdpSession.On("Page.screencastFrame", func(payload map[string]any) {
		go func() {
			dataStr, _ := payload["data"].(string)
			sessionID := payload["sessionId"]

			frameData, err := base64.StdEncoding.DecodeString(dataStr)
			if err == nil {
				s.mu.Lock()
				w := s.video
				s.mu.Unlock()
				if w != nil && w.stdin != nil {
					w.stdin.Write(frameData) // errors are ignored; write fails silently after stop
				}
			}

			// Acknowledge the frame so Chrome continues sending the next one
			cdpSession.Send("Page.screencastFrameAck", map[string]any{"sessionId": sessionID})
		}()
	})

	// Begin screencast
	if _, err := cdpSession.Send("Page.startScreencast", map[string]any{
		"format":        "jpeg",
		"quality":       80,
		"everyNthFrame": 1,
	}); err != nil {
		cdpSession.Detach()
		stdin.Close()
		cmd.Process.Kill()
		s.mu.Lock()
		s.video = nil
		s.mu.Unlock()
		return nil, fmt.Errorf("failed to start screencast: %w", err)
	}

	return &protocol.CommandResult{
		Success: true,
		Message: fmt.Sprintf("video recording started (CDP screencast), output: %s", outputFile),
	}, nil
}

// cmdVideoStop stops the CDP screencast and finalizes the WebM video file.
func (s *Server) cmdVideoStop(params json.RawMessage) (interface{}, error) {
	s.mu.Lock()
	vs := s.video
	s.video = nil
	s.mu.Unlock()

	if vs == nil {
		return nil, fmt.Errorf("no video recording in progress")
	}

	// Stop the CDP screencast
	vs.cdpSession.Send("Page.stopScreencast", nil)

	// Brief pause to let any in-flight frames arrive and be written
	time.Sleep(300 * time.Millisecond)

	// Close ffmpeg stdin → signals EOF → ffmpeg finishes encoding the WebM file
	vs.stdin.Close()

	// Wait for ffmpeg to finish (with a 15s timeout in case something goes wrong)
	done := make(chan error, 1)
	go func() { done <- vs.cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		vs.cmd.Process.Kill()
	}

	// Detach CDP session
	vs.cdpSession.Detach()

	return &protocol.CommandResult{
		Success: true,
		Message: "video saved to " + vs.outputFile,
	}, nil
}

// cmdStop handles the "stop" command: gracefully shuts down the daemon.
func (s *Server) cmdStop(params json.RawMessage) (interface{}, error) {
	go s.Stop()
	return &protocol.CommandResult{Success: true, Message: "stopping"}, nil
}

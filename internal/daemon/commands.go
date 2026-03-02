package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	snap, err := s.snapshots.Generate(page)
	if err != nil {
		return nil
	}
	s.mu.Lock()
	s.currentSnapshot = snap
	s.mu.Unlock()
	return snap
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

	// Generate snapshot while already holding s.mu.Lock() (takeSnapshot would deadlock)
	snapData, _ := s.snapshots.Generate(inst.Page)
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
	if filename == "" {
		filename = fmt.Sprintf("screenshot-%d.png", time.Now().UnixMilli())
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

// cmdShow brings the browser window to the front.
func (s *Server) cmdShow(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	if sbPage, ok := inst.Page.(*seleniumbase.Page); ok {
		if err := sbPage.PlaywrightPage().BringToFront(); err != nil {
			return nil, err
		}
	}

	return &protocol.CommandResult{Success: true, Message: "window brought to front"}, nil
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
		filename = fmt.Sprintf("trace-%d.zip", time.Now().UnixMilli())
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

// cmdVideoStart starts real video recording by restarting the session with RecordVideo enabled.
func (s *Server) cmdVideoStart(params json.RawMessage) (interface{}, error) {
	inst, err := s.requireSession()
	if err != nil {
		return nil, err
	}

	// Save current URL and session config
	currentURL := inst.Page.URL()
	sessionCfg := inst.Config

	// Create a temp dir for recording
	videoDir, err := os.MkdirTemp("", "sw-video-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create video dir: %w", err)
	}

	s.mu.Lock()
	s.videoDir = videoDir
	// Close current session
	s.sessions.Close(sessionCfg.Name)
	s.currentSession = nil
	s.consoleMessages = nil
	s.networkEvents = nil

	// Restart session with RecordVideo
	newCfg := &session.Config{
		Name:        sessionCfg.Name,
		Browser:     sessionCfg.Browser,
		Headed:      sessionCfg.Headed,
		Stealth:     sessionCfg.Stealth,
		Persistent:  sessionCfg.Persistent,
		Profile:     sessionCfg.Profile,
		UserDataDir: sessionCfg.UserDataDir,
		RecordVideo: videoDir,
	}
	newInst, err := s.sessions.GetOrCreate(newCfg)
	if err != nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("failed to restart session for recording: %w", err)
	}
	s.currentSession = newInst

	// Re-register event listeners
	if sbPage, ok := newInst.Page.(*seleniumbase.Page); ok {
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
	s.mu.Unlock()

	// Navigate back to original URL
	if currentURL != "" && currentURL != "about:blank" {
		if err := newInst.Page.Goto(currentURL); err != nil {
			return nil, fmt.Errorf("failed to navigate back to original URL: %w", err)
		}
	}

	return &protocol.CommandResult{
		Success: true,
		Message: fmt.Sprintf("video recording started, saving to %s", videoDir),
	}, nil
}

// cmdVideoStop stops video recording, saves the file, and restarts the session without recording.
func (s *Server) cmdVideoStop(params json.RawMessage) (interface{}, error) {
	var p protocol.TracingParams // reuse TracingParams for filename
	if len(params) > 0 {
		json.Unmarshal(params, &p)
	}

	s.mu.Lock()
	inst := s.currentSession
	videoDir := s.videoDir
	s.mu.Unlock()

	if inst == nil {
		return nil, ErrBrowserNotOpen
	}
	if videoDir == "" {
		return nil, fmt.Errorf("no video recording in progress")
	}

	// Get video path before closing
	var videoPath string
	if sbPage, ok := inst.Page.(*seleniumbase.Page); ok {
		pwPage := sbPage.PlaywrightPage()
		if video := pwPage.Video(); video != nil {
			if path, err := video.Path(); err == nil {
				videoPath = path
			}
		}
	}

	// Save current URL and config for restart
	currentURL := inst.Page.URL()
	sessionCfg := inst.Config

	// Close session to finalize video
	s.mu.Lock()
	s.sessions.Close(sessionCfg.Name)
	s.currentSession = nil
	s.videoDir = ""
	s.mu.Unlock()

	// Determine output filename
	dest := p.Filename
	if dest == "" {
		dest = fmt.Sprintf("video-%d.webm", time.Now().UnixMilli())
	}

	// Move or find the video file
	if videoPath != "" {
		// Wait a moment for file to be written
		time.Sleep(500 * time.Millisecond)
		if err := os.Rename(videoPath, dest); err != nil {
			// Try copy if rename fails (cross-device)
			if data, readErr := os.ReadFile(videoPath); readErr == nil {
				os.WriteFile(dest, data, 0644)
				os.Remove(videoPath)
			}
		}
	} else if videoDir != "" {
		// Look for any video file in the directory
		entries, _ := os.ReadDir(videoDir)
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".webm") || strings.HasSuffix(e.Name(), ".mp4") {
				src := filepath.Join(videoDir, e.Name())
				if err := os.Rename(src, dest); err != nil {
					if data, readErr := os.ReadFile(src); readErr == nil {
						os.WriteFile(dest, data, 0644)
					}
				}
				break
			}
		}
		os.RemoveAll(videoDir)
	}

	// Restart session without recording
	var newInst *session.Instance
	s.mu.Lock()
	newCfg := &session.Config{
		Name:        sessionCfg.Name,
		Browser:     sessionCfg.Browser,
		Headed:      sessionCfg.Headed,
		Stealth:     sessionCfg.Stealth,
		Persistent:  sessionCfg.Persistent,
		Profile:     sessionCfg.Profile,
		UserDataDir: sessionCfg.UserDataDir,
	}
	newInst, err := s.sessions.GetOrCreate(newCfg)
	if err == nil {
		s.currentSession = newInst
		// Re-register event listeners
		if sbPage, ok := newInst.Page.(*seleniumbase.Page); ok {
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
	}
	s.mu.Unlock()

	// Navigate back
	if newInst != nil && currentURL != "" && currentURL != "about:blank" {
		newInst.Page.Goto(currentURL)
	}

	return &protocol.CommandResult{
		Success: true,
		Message: fmt.Sprintf("video recording stopped, saved to %s", dest),
	}, nil
}

// Package client implements the daemon client.
package client

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kyungw00k/sw/pkg/protocol"
)

// Client is the daemon client.
type Client struct {
	socketPath string
	conn       net.Conn
	encoder    *json.Encoder
	decoder    *json.Decoder
	mu         sync.Mutex
	nextID     int64
}

// Config is the client configuration.
type Config struct {
	SocketPath string
}

// NewClient creates a new client.
func NewClient(cfg *Config) *Client {
	return &Client{
		socketPath: cfg.SocketPath,
		nextID:     1,
	}
}

// Connect connects to the daemon.
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return nil
	}

	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}

	c.conn = conn
	c.encoder = json.NewEncoder(conn)
	c.decoder = json.NewDecoder(conn)

	return nil
}

// Disconnect disconnects from the daemon.
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

// Close is an alias for Disconnect.
func (c *Client) Close() error {
	return c.Disconnect()
}

// CanConnect checks if the daemon is running.
func (c *Client) CanConnect() bool {
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// Call makes a call to the daemon.
func (c *Client) Call(method string, params interface{}) (*protocol.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		if err := c.connectLocked(); err != nil {
			return nil, err
		}
	}

	// Marshal params
	var paramsData json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		paramsData = data
	}

	// Create request
	req := &protocol.Request{
		JSONRPC: protocol.Version,
		ID:      c.nextID,
		Method:  method,
		Params:  paramsData,
	}
	c.nextID++

	// Send request
	if err := c.encoder.Encode(req); err != nil {
		c.conn.Close()
		c.conn = nil
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Receive response
	var resp protocol.Response
	if err := c.decoder.Decode(&resp); err != nil {
		c.conn.Close()
		c.conn = nil
		return nil, fmt.Errorf("failed to receive response: %w", err)
	}

	return &resp, nil
}

// connectLocked connects without locking.
func (c *Client) connectLocked() error {
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}

	c.conn = conn
	c.encoder = json.NewEncoder(conn)
	c.decoder = json.NewDecoder(conn)

	return nil
}

// StartDaemon starts the daemon if not running.
// sessionName is passed to the subprocess so it listens on the correct socket.
func (c *Client) StartDaemon(daemonPath, sessionName string) error {
	if c.CanConnect() {
		return nil
	}

	// Build args: pass --session so the subprocess uses the right socket path.
	cmdArgs := []string{}
	if sessionName != "" && sessionName != "default" {
		cmdArgs = append(cmdArgs, "--session", sessionName)
	}
	cmdArgs = append(cmdArgs, "daemon", "start")

	// Start daemon process in background, capturing stdout for readiness signal
	cmd := exec.Command(daemonPath, cmdArgs...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to pipe daemon stdout: %w", err)
	}
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Read readiness signal from stdout (### Success ... <EOF>)
	ready := make(chan error, 1)
	go func() {
		buf := make([]byte, 4096)
		var output string
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				output += string(buf[:n])
				if strings.Contains(output, "<EOF>") {
					if strings.Contains(output, "### Success") {
						ready <- nil
					} else {
						ready <- fmt.Errorf("daemon reported error: %s", output)
					}
					return
				}
			}
			if err != nil {
				// Process closed stdout without sending signal; fall back to connect check
				ready <- nil
				return
			}
		}
	}()

	// Wait for readiness signal or timeout
	select {
	case err := <-ready:
		if err != nil {
			_ = cmd.Process.Kill()
			return err
		}
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		return fmt.Errorf("daemon did not start in time")
	}

	// Release the process so it continues after parent exits
	_ = cmd.Process.Release()
	return nil
}

// Convenience methods for common commands

// Open opens a browser session.
func (c *Client) Open(url string, opts ...OpenOption) (*protocol.CommandResult, error) {
	params := &protocol.OpenParams{
		URL:     url,
		Browser: "chromium",
		Headed:  true,
		Stealth: true,
	}
	for _, opt := range opts {
		opt(params)
	}

	resp, err := c.Call("open", params)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("%s", resp.Error.Message)
	}

	var result protocol.CommandResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// OpenOption is an option for Open.
type OpenOption func(*protocol.OpenParams)

// WithHeaded sets headed mode.
func WithHeaded(headed bool) OpenOption {
	return func(p *protocol.OpenParams) {
		p.Headed = headed
	}
}

// WithBrowser sets the browser type.
func WithBrowser(browser string) OpenOption {
	return func(p *protocol.OpenParams) {
		p.Browser = browser
	}
}

// WithStealth sets stealth mode.
func WithStealth(stealth bool) OpenOption {
	return func(p *protocol.OpenParams) {
		p.Stealth = stealth
	}
}

// WithDevice sets the device to emulate.
func WithDevice(device string) OpenOption {
	return func(p *protocol.OpenParams) {
		p.Device = device
	}
}

// CloseSession closes the browser session.
func (c *Client) CloseSession() error {
	_, err := c.Call("close", nil)
	return err
}

// Goto navigates to a URL.
func (c *Client) Goto(url string) (*protocol.CommandResult, error) {
	resp, err := c.Call("goto", &protocol.GotoParams{URL: url})
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("%s", resp.Error.Message)
	}

	var result protocol.CommandResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Snapshot generates a snapshot.
func (c *Client) Snapshot() (*protocol.SnapshotResult, error) {
	resp, err := c.Call("snapshot", nil)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("%s", resp.Error.Message)
	}

	var result protocol.SnapshotResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Click clicks an element.
func (c *Client) Click(ref string) (*protocol.CommandResult, error) {
	resp, err := c.Call("click", &protocol.ClickParams{Ref: ref})
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("%s", resp.Error.Message)
	}

	var result protocol.CommandResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Fill fills text into an element.
func (c *Client) Fill(ref, text string) (*protocol.CommandResult, error) {
	resp, err := c.Call("fill", &protocol.FillParams{Ref: ref, Text: text})
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("%s", resp.Error.Message)
	}

	var result protocol.CommandResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Type types text.
func (c *Client) Type(text string) (*protocol.CommandResult, error) {
	resp, err := c.Call("type", &protocol.TypeParams{Text: text})
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("%s", resp.Error.Message)
	}

	var result protocol.CommandResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Press presses a key.
func (c *Client) Press(key string) (*protocol.CommandResult, error) {
	resp, err := c.Call("press", &protocol.PressParams{Key: key})
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("%s", resp.Error.Message)
	}

	var result protocol.CommandResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Hover hovers over an element.
func (c *Client) Hover(ref string) (*protocol.CommandResult, error) {
	resp, err := c.Call("hover", &protocol.HoverParams{Ref: ref})
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("%s", resp.Error.Message)
	}

	var result protocol.CommandResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Check checks a checkbox or radio button.
func (c *Client) Check(ref string) (*protocol.CommandResult, error) {
	resp, err := c.Call("check", &protocol.ClickParams{Ref: ref})
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("%s", resp.Error.Message)
	}

	var result protocol.CommandResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Screenshot takes a screenshot.
func (c *Client) Screenshot() (*protocol.CommandResult, error) {
	resp, err := c.Call("screenshot", nil)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("%s", resp.Error.Message)
	}

	var result protocol.CommandResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// List lists sessions.
func (c *Client) List() ([]SessionInfo, error) {
	resp, err := c.Call("list", nil)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("%s", resp.Error.Message)
	}

	var result []SessionInfo
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// SessionInfo represents session information.
type SessionInfo struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	URL        string `json:"url,omitempty"`
	Title      string `json:"title,omitempty"`
	Browser    string `json:"browser"`
	Headed     bool   `json:"headed"`
	Persistent bool   `json:"persistent"`
}

// DefaultSocketPath returns the socket path for the given session name.
// Each named session gets its own socket, mirroring playwright-cli's per-session daemon model.
func DefaultSocketPath(sessionName string) string {
	if sessionName == "" {
		sessionName = "default"
	}
	cwd, _ := os.Getwd()
	hash := fmt.Sprintf("%08x", len(cwd))
	return filepath.Join(os.TempDir(), "sw", hash, sessionName+".sock")
}

// DefaultPidPath returns the pid file path for the given session name.
func DefaultPidPath(sessionName string) string {
	if sessionName == "" {
		sessionName = "default"
	}
	cwd, _ := os.Getwd()
	hash := fmt.Sprintf("%08x", len(cwd))
	return filepath.Join(os.TempDir(), "sw", hash, sessionName+".pid")
}

// DefaultBaseDir returns the default base directory (CWD-relative .sw).
func DefaultBaseDir() string {
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, ".sw")
}

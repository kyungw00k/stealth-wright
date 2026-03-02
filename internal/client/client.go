// Package client implements the daemon client.
package client

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
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
func (c *Client) StartDaemon(daemonPath string) error {
	if c.CanConnect() {
		return nil
	}

	// Start daemon process
	cmd := exec.Command(daemonPath, "daemon", "start")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Wait for daemon to start
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		if c.CanConnect() {
			return nil
		}
	}

	return fmt.Errorf("daemon did not start in time")
}

// Convenience methods for common commands

// Open opens a browser session.
func (c *Client) Open(url string) (*protocol.CommandResult, error) {
	params := &protocol.OpenParams{URL: url}
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
func (c *Client) List() ([]string, error) {
	resp, err := c.Call("list", nil)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("%s", resp.Error.Message)
	}

	var result []string
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// DefaultSocketPath returns the default socket path.
func DefaultSocketPath() string {
	// Use hash of current directory for isolation
	cwd, _ := os.Getwd()
	hash := fmt.Sprintf("%08x", len(cwd))
	return filepath.Join(os.TempDir(), "sw", hash, "default.sock")
}

// DefaultBaseDir returns the default base directory.
func DefaultBaseDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".sw")
}

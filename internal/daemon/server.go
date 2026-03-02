// Package daemon implements the background daemon server.
package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/kyungw00k/sw/internal/session"
	"github.com/kyungw00k/sw/internal/snapshot"
	"github.com/kyungw00k/sw/pkg/protocol"
)

// Server is the daemon server.
type Server struct {
	socketPath string
	listener   net.Listener
	sessions   *session.Manager
	snapshots  *snapshot.Generator
	commands   *CommandRegistry
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.Mutex
	// Current session/page state
	currentSession  *session.Instance
	currentSnapshot *protocol.SnapshotResult
}

// Config is the server configuration.
type Config struct {
	SocketPath string
	BaseDir    string
}

// NewServer creates a new daemon server.
func NewServer(cfg *Config) (*Server, error) {
	// Ensure socket directory exists
	dir := filepath.Dir(cfg.SocketPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Remove existing socket
	os.Remove(cfg.SocketPath)

	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		socketPath: cfg.SocketPath,
		sessions:   session.NewManager(cfg.BaseDir),
		snapshots:  snapshot.NewGenerator(filepath.Join(cfg.BaseDir, "snapshots")),
		commands:   NewCommandRegistry(),
		ctx:        ctx,
		cancel:     cancel,
	}

	// Register commands
	s.registerCommands()

	return s, nil
}

// Start starts the daemon server.
func (s *Server) Start() error {
	// Create listener
	l, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.socketPath, err)
	}
	s.listener = l

	// Set socket permissions
	os.Chmod(s.socketPath, 0600)

	// Handle shutdown
	go s.handleSignals()

	// Accept connections
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.acceptLoop()
	}()

	return nil
}

// acceptLoop accepts incoming connections.
func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return
			default:
				continue
			}
		}

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handleConnection(conn)
		}()
	}
}

// handleConnection handles a single connection.
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		var req protocol.Request
		if err := decoder.Decode(&req); err != nil {
			// Connection closed or error
			return
		}

		// Execute command
		resp := s.executeCommand(&req)

		// Send response
		if err := encoder.Encode(resp); err != nil {
			return
		}
	}
}

// executeCommand executes a command and returns the response.
func (s *Server) executeCommand(req *protocol.Request) *protocol.Response {
	handler, ok := s.commands.Get(req.Method)
	if !ok {
		return protocol.NewErrorResponse(req.ID, protocol.CodeMethodNotFound,
			fmt.Sprintf("method not found: %s", req.Method))
	}

	result, err := handler(req.Params)
	if err != nil {
		return protocol.NewErrorResponse(req.ID, protocol.CodeInternalError, err.Error())
	}

	resp, err := protocol.NewResponse(req.ID, result)
	if err != nil {
		return protocol.NewErrorResponse(req.ID, protocol.CodeInternalError,
			fmt.Sprintf("failed to marshal result: %v", err))
	}

	return resp
}

// handleSignals handles OS signals.
func (s *Server) handleSignals() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	s.Stop()
}

// Stop stops the daemon server.
func (s *Server) Stop() {
	s.cancel()

	if s.listener != nil {
		s.listener.Close()
	}

	// Close all sessions
	s.sessions.CloseAll()

	// Wait for goroutines
	s.wg.Wait()

	// Remove socket
	os.Remove(s.socketPath)
}

// WaitForShutdown waits for the server to shutdown.
func (s *Server) WaitForShutdown() {
	<-s.ctx.Done()
	s.wg.Wait()
}

// IsRunning checks if the daemon is running.
func IsRunning(socketPath string) bool {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// Print prints to stdout (for daemon communication).
func Print(format string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}

// PrintError prints to stderr.
func PrintError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// Daemon protocol markers
const (
	MarkerSuccess = "### Success"
	MarkerError   = "### Error"
	MarkerEOF     = "<EOF>"
)

// WriteSuccess writes a success marker.
func WriteSuccess(w *bufio.Writer, message string) {
	w.WriteString(fmt.Sprintf("%s\n%s\n%s\n", MarkerSuccess, message, MarkerEOF))
	w.Flush()
}

// WriteError writes an error marker.
func WriteError(w *bufio.Writer, message string) {
	w.WriteString(fmt.Sprintf("%s\n%s\n%s\n", MarkerError, message, MarkerEOF))
	w.Flush()
}

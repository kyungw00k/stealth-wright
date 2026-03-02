// Package protocol defines JSON-RPC types for daemon communication.
package protocol

import "encoding/json"

// Version is the JSON-RPC version.
const Version = "2.0"

// Request represents a JSON-RPC request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error represents a JSON-RPC error.
type Error struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Standard JSON-RPC error codes.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603

	// Custom error codes (-32000 to -32099)
	CodeBrowserNotOpen   = -32001
	CodeElementNotFound  = -32002
	CodeNavigationFailed = -32003
	CodeTimeout          = -32004
	CodeSessionNotFound  = -32005
)

// NewResponse creates a new response.
func NewResponse(id int64, result interface{}) (*Response, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return &Response{
		JSONRPC: Version,
		ID:      id,
		Result:  data,
	}, nil
}

// NewErrorResponse creates a new error response.
func NewErrorResponse(id int64, code int, message string) *Response {
	return &Response{
		JSONRPC: Version,
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
		},
	}
}

// Command types

// OpenParams represents parameters for the open command.
type OpenParams struct {
	URL string `json:"url,omitempty"`
}

// GotoParams represents parameters for the goto command.
type GotoParams struct {
	URL string `json:"url"`
}

// ClickParams represents parameters for the click command.
type ClickParams struct {
	Ref    string `json:"ref"`
	Button string `json:"button,omitempty"` // left, right, middle
}

// FillParams represents parameters for the fill command.
type FillParams struct {
	Ref  string `json:"ref"`
	Text string `json:"text"`
}

// TypeParams represents parameters for the type command.
type TypeParams struct {
	Text   string `json:"text"`
	Submit bool   `json:"submit,omitempty"`
}

// PressParams represents parameters for the press command.
type PressParams struct {
	Key string `json:"key"`
}

// HoverParams represents parameters for the hover command.
type HoverParams struct {
	Ref string `json:"ref"`
}

// ScreenshotParams represents parameters for the screenshot command.
type ScreenshotParams struct {
	Ref      string `json:"ref,omitempty"`
	Filename string `json:"filename,omitempty"`
	FullPage bool   `json:"fullPage,omitempty"`
}

// Result types

// PageResult represents page information.
type PageResult struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	SnapshotRef string `json:"snapshotRef,omitempty"`
}

// SnapshotResult represents snapshot result.
type SnapshotResult struct {
	PageURL   string        `json:"pageUrl"`
	PageTitle string        `json:"pageTitle"`
	Elements  []ElementInfo `json:"elements"`
	Filename  string        `json:"filename,omitempty"`
}

// ElementInfo represents element information in a snapshot.
type ElementInfo struct {
	Ref        string            `json:"ref"`
	Selector   string            `json:"selector"`
	TagName    string            `json:"tagName"`
	Text       string            `json:"text,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// SessionResult represents session information.
type SessionResult struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	URL        string `json:"url,omitempty"`
	Title      string `json:"title,omitempty"`
	Browser    string `json:"browser"`
	Headed     bool   `json:"headed"`
	Persistent bool   `json:"persistent"`
}

// CommandResult represents a generic command result.
type CommandResult struct {
	Success  bool        `json:"success"`
	Page     *PageResult `json:"page,omitempty"`
	Snapshot string      `json:"snapshot,omitempty"`
	Message  string      `json:"message,omitempty"`
}

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
	URL     string `json:"url,omitempty"`
	Browser string `json:"browser,omitempty"`
	Headed  bool   `json:"headed,omitempty"`
	Stealth bool   `json:"stealth,omitempty"`
}

// GotoParams represents parameters for the goto command.
type GotoParams struct {
	URL string `json:"url"`
}

// ClickParams represents parameters for the click command.
type ClickParams struct {
	Ref       string   `json:"ref"`
	Button    string   `json:"button,omitempty"` // left, right, middle
	Modifiers []string `json:"modifiers,omitempty"`
}

// CookieSetParams represents parameters for the cookie-set command.
type CookieSetParams struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain,omitempty"`
	Path     string  `json:"path,omitempty"`
	Expires  float64 `json:"expires,omitempty"`
	HTTPOnly bool    `json:"httpOnly,omitempty"`
	Secure   bool    `json:"secure,omitempty"`
	SameSite string  `json:"sameSite,omitempty"`
}

// CookieListParams represents parameters for the cookie-list command.
type CookieListParams struct {
	Domain string `json:"domain,omitempty"`
	Path   string `json:"path,omitempty"`
}

// FillParams represents parameters for the fill command.
type FillParams struct {
	Ref    string `json:"ref"`
	Text   string `json:"text"`
	Submit bool   `json:"submit,omitempty"`
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

// DragParams represents parameters for the drag command.
type DragParams struct {
	StartRef string `json:"startRef"`
	EndRef   string `json:"endRef"`
}

// SelectParams represents parameters for the select command.
type SelectParams struct {
	Ref    string `json:"ref"`
	Values []string `json:"values"`
}

// UploadParams represents parameters for the upload command.
type UploadParams struct {
	Files []string `json:"files"`
}

// ResizeParams represents parameters for the resize command.
type ResizeParams struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// EvalParams represents parameters for the eval command.
type EvalParams struct {
	Script string `json:"script"`
	Ref    string `json:"ref,omitempty"`
}

// KeyParams represents parameters for key commands.
type KeyParams struct {
	Key string `json:"key"`
}

// MouseParams represents parameters for mouse commands.
type MouseParams struct {
	X      int    `json:"x,omitempty"`
	Y      int    `json:"y,omitempty"`
	Button string `json:"button,omitempty"`
	Dx     int    `json:"dx,omitempty"`
	Dy     int    `json:"dy,omitempty"`
}

// DialogParams represents parameters for dialog commands.
type DialogParams struct {
	PromptText string `json:"promptText,omitempty"`
}

// TabParams represents parameters for tab commands.
type TabParams struct {
	URL   string `json:"url,omitempty"`
	Index int    `json:"index,omitempty"`
}

// StorageParams represents parameters for storage commands.
type StorageParams struct {
	Filename string `json:"filename,omitempty"`
}

// KeyValueParams represents parameters for key-value storage commands.
type KeyValueParams struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

// RunCodeParams represents parameters for the run-code command.
type RunCodeParams struct {
	Code string `json:"code"`
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
	PageURL      string        `json:"pageUrl"`
	PageTitle    string        `json:"pageTitle"`
	AriaSnapshot string        `json:"ariaSnapshot,omitempty"` // Playwright native aria snapshot with refs
	Elements     []ElementInfo `json:"elements"`
	Filename     string        `json:"filename,omitempty"`
}

// SnapshotParams represents parameters for the snapshot command.
type SnapshotParams struct {
	Filename string `json:"filename,omitempty"`
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
	Success  bool            `json:"success"`
	Page     *PageResult     `json:"page,omitempty"`
	Snapshot *SnapshotResult `json:"snapshot,omitempty"`
	Message  string          `json:"message,omitempty"`
	Data     any             `json:"data,omitempty"`
}

// EvalResult represents the result of eval command.
type EvalResult struct {
	Value any `json:"value"`
}

// TabResult represents tab information.
type TabResult struct {
	Index   int    `json:"index"`
	URL     string `json:"url"`
	Title   string `json:"title"`
	Current bool   `json:"current"`
}

// TabsResult represents the result of tab-list command.
type TabsResult struct {
	Tabs []TabResult `json:"tabs"`
}

// CookieResult represents a cookie.
type CookieResult struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain,omitempty"`
	Path     string `json:"path,omitempty"`
	Expires  int64  `json:"expires,omitempty"`
	HTTPOnly bool   `json:"httpOnly"`
	Secure   bool   `json:"secure"`
}

// StorageEntryResult represents a storage entry.
type StorageEntryResult struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ConsoleEntry represents a browser console message.
type ConsoleEntry struct {
	Type string `json:"type"` // "log","error","warning","info","debug"
	Text string `json:"text"`
	URL  string `json:"url,omitempty"`
	Line int    `json:"line,omitempty"`
}

// ConsoleParams represents parameters for the console command.
type ConsoleParams struct {
	Level string `json:"level,omitempty"` // filter: "info","warning","error","debug"
	Clear bool   `json:"clear,omitempty"`
}

// NetworkEntry represents a network request/response.
type NetworkEntry struct {
	URL          string `json:"url"`
	Method       string `json:"method"`
	Status       int    `json:"status"`
	ResourceType string `json:"resourceType"`
	Timestamp    int64  `json:"timestamp"`
}

// NetworkParams represents parameters for the network command.
type NetworkParams struct {
	Static bool `json:"static,omitempty"` // include static resources (images, CSS, fonts)
	Clear  bool `json:"clear,omitempty"`
}

// TracingParams represents parameters for tracing commands.
type TracingParams struct {
	Filename string `json:"filename,omitempty"`
}

// RouteParams represents parameters for the route command.
type RouteParams struct {
	Pattern       string            `json:"pattern"`
	Status        int               `json:"status,omitempty"`
	Body          string            `json:"body,omitempty"`
	ContentType   string            `json:"contentType,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	RemoveHeaders []string          `json:"removeHeaders,omitempty"`
}

// RouteEntry represents an active route.
type RouteEntry struct {
	Pattern string `json:"pattern"`
	Status  int    `json:"status"`
}

// UnrouteParams represents parameters for the unroute command.
type UnrouteParams struct {
	Pattern string `json:"pattern,omitempty"`
}

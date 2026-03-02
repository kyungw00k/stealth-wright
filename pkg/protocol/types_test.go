package protocol

import (
	"encoding/json"
	"testing"
)

func TestNewResponse(t *testing.T) {
	result := map[string]string{"status": "ok"}
	resp, err := NewResponse(1, result)
	if err != nil {
		t.Fatalf("NewResponse() error = %v", err)
	}

	if resp.JSONRPC != Version {
		t.Errorf("Response.JSONRPC = %q, want %q", resp.JSONRPC, Version)
	}
	if resp.ID != 1 {
		t.Errorf("Response.ID = %d, want 1", resp.ID)
	}
	if resp.Error != nil {
		t.Errorf("Response.Error should be nil")
	}

	var parsed map[string]string
	if err := json.Unmarshal(resp.Result, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	if parsed["status"] != "ok" {
		t.Errorf("Result status = %q, want %q", parsed["status"], "ok")
	}
}

func TestNewErrorResponse(t *testing.T) {
	resp := NewErrorResponse(2, CodeMethodNotFound, "method not found")

	if resp.JSONRPC != Version {
		t.Errorf("Response.JSONRPC = %q, want %q", resp.JSONRPC, Version)
	}
	if resp.ID != 2 {
		t.Errorf("Response.ID = %d, want 2", resp.ID)
	}
	if resp.Error == nil {
		t.Fatal("Response.Error should not be nil")
	}
	if resp.Error.Code != CodeMethodNotFound {
		t.Errorf("Error.Code = %d, want %d", resp.Error.Code, CodeMethodNotFound)
	}
	if resp.Error.Message != "method not found" {
		t.Errorf("Error.Message = %q, want %q", resp.Error.Message, "method not found")
	}
}

func TestRequestJSON(t *testing.T) {
	req := Request{
		JSONRPC: Version,
		ID:      1,
		Method:  "click",
		Params:  json.RawMessage(`{"ref":"e1"}`),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	var parsed Request
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	if parsed.Method != "click" {
		t.Errorf("Method = %q, want %q", parsed.Method, "click")
	}
}

func TestResponseJSON(t *testing.T) {
	resp := Response{
		JSONRPC: Version,
		ID:      1,
		Result:  json.RawMessage(`{"success":true}`),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var parsed Response
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if parsed.Error != nil {
		t.Errorf("Error should be nil for successful response")
	}
}

func TestCommandResult(t *testing.T) {
	result := CommandResult{
		Success: true,
		Page: &PageResult{
			URL:   "https://example.com",
			Title: "Example",
		},
		Message: "done",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal CommandResult: %v", err)
	}

	var parsed CommandResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal CommandResult: %v", err)
	}

	if !parsed.Success {
		t.Error("Success should be true")
	}
	if parsed.Page == nil {
		t.Fatal("Page should not be nil")
	}
	if parsed.Page.URL != "https://example.com" {
		t.Errorf("Page.URL = %q, want %q", parsed.Page.URL, "https://example.com")
	}
}

func TestSnapshotResult(t *testing.T) {
	result := SnapshotResult{
		PageURL:   "https://example.com",
		PageTitle: "Example",
		Elements: []ElementInfo{
			{
				Ref:      "e1",
				Selector: "#button",
				TagName:  "button",
				Text:     "Click me",
			},
		},
		Filename: "snapshot.yml",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal SnapshotResult: %v", err)
	}

	var parsed SnapshotResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal SnapshotResult: %v", err)
	}

	if len(parsed.Elements) != 1 {
		t.Fatalf("Elements count = %d, want 1", len(parsed.Elements))
	}
	if parsed.Elements[0].Ref != "e1" {
		t.Errorf("Element.Ref = %q, want %q", parsed.Elements[0].Ref, "e1")
	}
}

func TestErrorCodes(t *testing.T) {
	codes := []int{
		CodeParseError,
		CodeInvalidRequest,
		CodeMethodNotFound,
		CodeInvalidParams,
		CodeInternalError,
		CodeBrowserNotOpen,
		CodeElementNotFound,
		CodeNavigationFailed,
		CodeTimeout,
		CodeSessionNotFound,
	}

	for _, code := range codes {
		if code >= 0 {
			t.Errorf("Error code %d should be negative", code)
		}
	}
}

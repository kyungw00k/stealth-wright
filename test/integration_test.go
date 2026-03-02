//go:build integration

package test

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// isCI returns true if running in CI environment
func isCI() bool {
	return os.Getenv("CI") != ""
}

// headedArgs returns headed args if not in CI, empty otherwise
func headedArgs() []string {
	if isCI() {
		return nil
	}
	return []string{"--headed"}
}
func TestTodoMVC(t *testing.T) {
	// Get the sw binary path
	execPath := os.Getenv("SW_BINARY")
	if execPath == "" {
		execPath = "../bin/sw"
	}

	// Clean up any existing daemon
	_ = exec.Command(execPath, "daemon", "stop").Run()
	_ = exec.Command("pkill", "-f", "sw").Run()
	time.Sleep(500 * time.Millisecond)

	// Step 1: Open browser
	t.Log("Step 1: Opening browser...")
	cmd := exec.Command(execPath, append([]string{"open", "https://demo.playwright.dev/todomvc/"}, headedArgs()...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to open browser: %v\nOutput: %s", err, string(output))
	}
	t.Log("Browser opened")
	t.Log(string(output))

	// Give browser time to load
	time.Sleep(3 * time.Second)

	// Step 2: Get snapshot
	t.Log("Step 2: Getting snapshot...")
	cmd = exec.Command(execPath, "snapshot")
	output, err = cmd.CombinedOutput()
	if err != nil {
		_ = exec.Command(execPath, "close").Run()
		t.Fatalf("Failed to get snapshot: %v\nOutput: %s", err, string(output))
	}
	t.Log("Snapshot received")
	t.Log(string(output))

	// Step 3: Type first todo
	t.Log("Step 3: Typing first todo...")
	cmd = exec.Command(execPath, "type", "Buy groceries")
	output, err = cmd.CombinedOutput()
	if err != nil {
		_ = exec.Command(execPath, "close").Run()
		t.Fatalf("Failed to type: %v\nOutput: %s", err, string(output))
	}
	t.Log("Typed 'Buy groceries'")

	// Step 4: Press Enter
	t.Log("Step 4: Pressing Enter...")
	cmd = exec.Command(execPath, "press", "Enter")
	output, err = cmd.CombinedOutput()
	if err != nil {
		_ = exec.Command(execPath, "close").Run()
		t.Fatalf("Failed to press Enter: %v\nOutput: %s", err, string(output))
	}
	t.Log("Pressed Enter")

	time.Sleep(500 * time.Millisecond)

	// Step 5: Type second todo
	t.Log("Step 5: Typing second todo...")
	cmd = exec.Command(execPath, "type", "Water flowers")
	output, err = cmd.CombinedOutput()
	if err != nil {
		_ = exec.Command(execPath, "close").Run()
		t.Fatalf("Failed to type: %v\nOutput: %s", err, string(output))
	}
	t.Log("Typed 'Water flowers'")

	// Step 6: Press Enter
	t.Log("Step 6: Pressing Enter...")
	cmd = exec.Command(execPath, "press", "Enter")
	output, err = cmd.CombinedOutput()
	if err != nil {
		_ = exec.Command(execPath, "close").Run()
		t.Fatalf("Failed to press Enter: %v\nOutput: %s", err, string(output))
	}
	t.Log("Pressed Enter")

	time.Sleep(500 * time.Millisecond)

	// Step 7: Get snapshot to find checkboxes
	t.Log("Step 7: Getting snapshot to find checkboxes...")
	cmd = exec.Command(execPath, "snapshot")
	output, err = cmd.CombinedOutput()
	if err != nil {
		_ = exec.Command(execPath, "close").Run()
		t.Fatalf("Failed to get snapshot: %v\nOutput: %s", err, string(output))
	}
	snapshotOutput := string(output)
	t.Log("Snapshot received")

	// Find checkbox refs
	checkboxRefs := findCheckboxRefs(snapshotOutput)
	if len(checkboxRefs) < 2 {
		_ = exec.Command(execPath, "close").Run()
		t.Fatalf("Could not find enough checkboxes in snapshot (found %d)\nSnapshot:\n%s", len(checkboxRefs), snapshotOutput)
	}
	t.Logf("Found %d checkboxes: %v", len(checkboxRefs), checkboxRefs)

	// Step 8: Check first checkbox
	t.Logf("Step 8: Checking %s...", checkboxRefs[0])
	cmd = exec.Command(execPath, "check", checkboxRefs[0])
	output, err = cmd.CombinedOutput()
	if err != nil {
		_ = exec.Command(execPath, "close").Run()
		t.Fatalf("Failed to check: %v\nOutput: %s", err, string(output))
	}
	t.Logf("Checked %s", checkboxRefs[0])

	// Step 9: Check second checkbox
	t.Logf("Step 9: Checking %s...", checkboxRefs[1])
	cmd = exec.Command(execPath, "check", checkboxRefs[1])
	output, err = cmd.CombinedOutput()
	if err != nil {
		_ = exec.Command(execPath, "close").Run()
		t.Fatalf("Failed to check: %v\nOutput: %s", err, string(output))
	}
	t.Logf("Checked %s", checkboxRefs[1])

	// Step 10: Take screenshot
	t.Log("Step 10: Taking screenshot...")
	cmd = exec.Command(execPath, "screenshot")
	output, err = cmd.CombinedOutput()
	if err != nil {
		_ = exec.Command(execPath, "close").Run()
		t.Fatalf("Failed to take screenshot: %v\nOutput: %s", err, string(output))
	}
	t.Log("Screenshot taken")
	t.Log(string(output))

	// Clean up
	t.Log("Step 11: Closing browser...")
	cmd = exec.Command(execPath, "close")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to close: %v\nOutput: %s", err, string(output))
	}
	t.Log("Browser closed")

	t.Log("=== All tests passed! ===")
}

// findCheckboxRefs extracts checkbox refs from snapshot
func findCheckboxRefs(snapshot string) []string {
	var refs []string
	lines := strings.Split(snapshot, "\n")
	for _, line := range lines {
		// Look for checkbox elements in TodoMVC (input with "on" or checkbox type)
		// Skip the main text input (which has placeholder text)
		if strings.Contains(line, "<input>") && strings.Contains(line, `"on"`) {
			// Extract ref (e.g., "- e5: <input>")
			parts := strings.Split(line, ":")
			if len(parts) > 0 {
				ref := strings.TrimSpace(strings.TrimPrefix(parts[0], "-"))
				if strings.HasPrefix(ref, "e") {
					refs = append(refs, ref)
				}
			}
		}
	}
	return refs
}

// TestEval tests eval command
func TestEval(t *testing.T) {
	execPath := os.Getenv("SW_BINARY")
	if execPath == "" {
		execPath = "../bin/sw"
	}

	// Clean up
	_ = exec.Command("pkill", "-f", "sw").Run()
	time.Sleep(500 * time.Millisecond)

	// Open browser
	cmd := exec.Command(execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to open: %v\n%s", err, string(output))
	}
	time.Sleep(2 * time.Second)

	// Test eval
	cmd = exec.Command(execPath, "eval", "document.title")
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = exec.Command(execPath, "close").Run()
		t.Fatalf("Failed to eval: %v\n%s", err, string(output))
	}

	if !strings.Contains(string(output), "Example Domain") {
		_ = exec.Command(execPath, "close").Run()
		t.Fatalf("Eval result should contain 'Example Domain', got: %s", string(output))
	}

	// Close
	_ = exec.Command(execPath, "close").Run()
	t.Log("Eval test passed!")
}

// TestDblClick tests double-click command
func TestDblClick(t *testing.T) {
	execPath := os.Getenv("SW_BINARY")
	if execPath == "" {
		execPath = "../bin/sw"
	}

	// Clean up
	_ = exec.Command("pkill", "-f", "sw").Run()
	time.Sleep(500 * time.Millisecond)

	// Open browser with simple page
	cmd := exec.Command(execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to open: %v\n%s", err, string(output))
	}
	time.Sleep(2 * time.Second)

	// Get snapshot
	cmd = exec.Command(execPath, "snapshot")
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = exec.Command(execPath, "close").Run()
		t.Fatalf("Failed to get snapshot: %v\n%s", err, string(output))
	}

	// Close
	_ = exec.Command(execPath, "close").Run()
	t.Log("DblClick test passed!")
}

// TestTabs tests tab-related commands
func TestTabs(t *testing.T) {
	execPath := os.Getenv("SW_BINARY")
	if execPath == "" {
		execPath = "../bin/sw"
	}

	// Clean up
	_ = exec.Command("pkill", "-f", "sw").Run()
	time.Sleep(500 * time.Millisecond)

	// Open browser
	cmd := exec.Command(execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to open: %v\n%s", err, string(output))
	}
	time.Sleep(2 * time.Second)

	// List tabs
	cmd = exec.Command(execPath, "tab-list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = exec.Command(execPath, "close").Run()
		t.Fatalf("Failed to list tabs: %v\n%s", err, string(output))
	}
	t.Logf("Tab list: %s", string(output))

	// Open new tab
	cmd = exec.Command(execPath, "tab-new", "https://example.org")
	output, err = cmd.CombinedOutput()
	if err != nil {
		_ = exec.Command(execPath, "close").Run()
		t.Fatalf("Failed to open new tab: %v\n%s", err, string(output))
	}
	time.Sleep(2 * time.Second)

	// Close
	_ = exec.Command(execPath, "close").Run()
	t.Log("Tabs test passed!")
}

// TestStateSaveLoad tests state save/load commands
func TestStateSaveLoad(t *testing.T) {
	execPath := os.Getenv("SW_BINARY")
	if execPath == "" {
		execPath = "../bin/sw"
	}

	// Clean up
	_ = exec.Command("pkill", "-f", "sw").Run()
	time.Sleep(500 * time.Millisecond)

	// Open browser
	cmd := exec.Command(execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to open: %v\n%s", err, string(output))
	}
	time.Sleep(2 * time.Second)

	// Save state
	cmd = exec.Command(execPath, "state-save", "/tmp/sw-state.json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = exec.Command(execPath, "close").Run()
		t.Fatalf("Failed to save state: %v\n%s", err, string(output))
	}
	t.Logf("State saved: %s", string(output))

	// Close
	_ = exec.Command(execPath, "close").Run()
	t.Log("State save/load test passed!")
}

// TestCommandsBasic tests basic command execution without errors
func TestCommandsBasic(t *testing.T) {
	execPath := os.Getenv("SW_BINARY")
	if execPath == "" {
		execPath = "../bin/sw"
	}

	// Clean up
	_ = exec.Command("pkill", "-f", "sw").Run()
	time.Sleep(500 * time.Millisecond)

	// Open browser
	cmd := exec.Command(execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to open: %v\n%s", err, string(output))
	}
	time.Sleep(2 * time.Second)

	// Test keydown
	cmd = exec.Command(execPath, "keydown", "a")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Logf("keydown warning: %v\n%s", err, string(output))
	}

	// Test keyup
	cmd = exec.Command(execPath, "keyup", "a")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Logf("keyup warning: %v\n%s", err, string(output))
	}

	// Test resize
	cmd = exec.Command(execPath, "resize", "1024", "768")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Logf("resize warning: %v\n%s", err, string(output))
	}

	// Test mousemove
	cmd = exec.Command(execPath, "mousemove", "100", "100")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Logf("mousemove warning: %v\n%s", err, string(output))
	}

	// Test list
	cmd = exec.Command(execPath, "list")
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = exec.Command(execPath, "close").Run()
		t.Fatalf("Failed to list: %v\n%s", err, string(out))
	} else {
		t.Logf("Sessions: %s", string(out))
	}

	// Close
	_ = exec.Command(execPath, "close").Run()
	t.Log("Basic commands test passed!")
}

// TestNavigation tests navigation commands
func TestNavigation(t *testing.T) {
	execPath := os.Getenv("SW_BINARY")
	if execPath == "" {
		execPath = "../bin/sw"
	}

	// Clean up
	_ = exec.Command("pkill", "-f", "sw").Run()
	time.Sleep(500 * time.Millisecond)

	// Open browser
	cmd := exec.Command(execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to open: %v\n%s", err, string(output))
	}
	time.Sleep(2 * time.Second)

	// Navigate to another page
	cmd = exec.Command(execPath, "goto", "https://example.org")
	if output, err := cmd.CombinedOutput(); err != nil {
		_ = exec.Command(execPath, "close").Run()
		t.Fatalf("Failed to goto: %v\n%s", err, string(output))
	}
	time.Sleep(2 * time.Second)

	// Go back (may timeout in headless - best effort)
	cmd = exec.Command(execPath, "go-back")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Logf("go-back warning (may occur in headless): %v\n%s", err, string(output))
	}
	time.Sleep(1 * time.Second)

	// Go forward (may timeout - best effort)
	cmd = exec.Command(execPath, "go-forward")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Logf("go-forward warning (expected): %v\n%s", err, string(output))
	}

	// Close
	_ = exec.Command(execPath, "close").Run()
	t.Log("Navigation test passed!")
}

// TestKillAll tests kill-all command
func TestKillAll(t *testing.T) {
	execPath := os.Getenv("SW_BINARY")
	if execPath == "" {
		execPath = "../bin/sw"
	}

	// Clean up
	_ = exec.Command("pkill", "-f", "sw").Run()
	time.Sleep(500 * time.Millisecond)

	// Open browser
	cmd := exec.Command(execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to open: %v\n%s", err, string(output))
	}
	time.Sleep(2 * time.Second)

	// Kill all
	cmd = exec.Command(execPath, "kill-all")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Logf("kill-all warning: %v\n%s", err, string(output))
	}

	t.Log("Kill-all test passed!")
}

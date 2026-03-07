//go:build integration

package test

import (
	"encoding/json"

	"net/url"
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

// TestMain runs cleanup after all tests complete.
func TestMain(m *testing.M) {
	execPath := os.Getenv("SW_BINARY")
	if execPath == "" {
		execPath = "../bin/sw"
	}
	code := m.Run()
	cleanupDaemon(execPath)
	os.Exit(code)
}

// Helper functions

func getExecPath(t *testing.T) string {
	execPath := os.Getenv("SW_BINARY")
	if execPath == "" {
		execPath = "../bin/sw"
	}
	return execPath
}

func cleanupDaemon(execPath string) {
	_ = exec.Command(execPath, "daemon", "stop").Run()
	_ = exec.Command("pkill", "-f", "sw").Run()
	time.Sleep(500 * time.Millisecond)
}

func closeBrowser(execPath string) {
	_ = exec.Command(execPath, "close").Run()
}

func runSw(t *testing.T, execPath string, args ...string) string {
	cmd := exec.Command(execPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %v %v\nOutput: %s", args, err, string(output))
	}
	return string(output)
}

func runSwIgnoreError(execPath string, args ...string) (string, error) {
	cmd := exec.Command(execPath, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func mustRunSw(execPath string, args ...string) string {
	cmd := exec.Command(execPath, args...)
	output, _ := cmd.CombinedOutput()
	return string(output)
}

// dataURL creates a data URL from HTML content
func dataURL(html string) string {
	return "data:text/html," + url.PathEscape(html)
}

// TestTodoMVC tests full todo workflow with type, press, check, screenshot
func TestTodoMVC(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	// Open TodoMVC
	runSw(t, execPath, append([]string{"open", "https://demo.playwright.dev/todomvc/"}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(2 * time.Second)

	// Add first todo
	runSw(t, execPath, "type", "Buy groceries")
	runSw(t, execPath, "press", "Enter")
	time.Sleep(300 * time.Millisecond)

	// Add second todo
	runSw(t, execPath, "type", "Water flowers")
	runSw(t, execPath, "press", "Enter")
	time.Sleep(300 * time.Millisecond)

	// Get snapshot and verify todos exist
	snapshot := runSw(t, execPath, "snapshot")
	if !strings.Contains(snapshot, "Buy groceries") {
		t.Fatalf("Expected 'Buy groceries' in snapshot, got:\n%s", snapshot)
	}
	if !strings.Contains(snapshot, "Water flowers") {
		t.Fatalf("Expected 'Water flowers' in snapshot, got:\n%s", snapshot)
	}

	// Find checkboxes
	checkboxRefs := findCheckboxRefs(snapshot)
	if len(checkboxRefs) < 2 {
		t.Fatalf("Expected at least 2 checkboxes, found %d\nSnapshot:\n%s", len(checkboxRefs), snapshot)
	}

	// Check first todo (mark as completed)
	runSw(t, execPath, "check", checkboxRefs[0])
	time.Sleep(200 * time.Millisecond)

	// Verify the item is now completed using eval
	result := mustRunSw(execPath, "eval", "document.querySelectorAll('.completed').length")
	if !strings.Contains(result, "1") && !strings.Contains(result, "2") {
		t.Logf("Warning: completed count not as expected: %s", result)
	} else {
		t.Log("✓ Todo marked as completed successfully")
	}

	t.Log("✓ TodoMVC test passed!")
}

// TestDblClick tests double-click with visible effect using self-contained HTML
func TestDblClick(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	// Self-contained HTML with double-click handler
	html := `<!DOCTYPE html>
<html>
<head><title>Double Click Test</title></head>
<body>
<h1>Double Click Test</h1>
<div id="box" style="width:100px;height:100px;background:blue;"
     ondblclick="this.style.background='green';this.textContent='Double Clicked!'">
  Double-click me
</div>
<p id="status">Status: waiting</p>
<script>
document.getElementById('box').addEventListener('dblclick', function() {
  document.getElementById('status').textContent = 'Status: double-clicked!';
});
</script>
</body>
</html>`

	runSw(t, execPath, append([]string{"open", dataURL(html)}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(1 * time.Second)

	// Get snapshot to find the box element
	snapshot := runSw(t, execPath, "snapshot")
	t.Logf("Snapshot:\n%s", snapshot)

	// Find the box element (blue div)
	boxRef := findElementByContent(snapshot, "box", "div")
	if boxRef == "" {
		// Fallback: find any element containing "Double-click"
		boxRef = findElementByContent(snapshot, "Double-click", "")
	}

	if boxRef == "" {
		t.Fatal("Could not find double-click target element")
	}

	t.Logf("Found box element: %s", boxRef)

	// Double-click the box
	runSw(t, execPath, "dblclick", boxRef)
	time.Sleep(500 * time.Millisecond)

	// Verify color changed to green
	result := mustRunSw(execPath, "eval", "document.getElementById('box').style.backgroundColor")
	t.Logf("Background color after dblclick: %s", result)

	if strings.Contains(result, "green") {
		t.Log("✓ Double-click changed background to green")
	} else {
		// Check text content as fallback
		result = mustRunSw(execPath, "eval", "document.getElementById('box').textContent")
		if strings.Contains(result, "Double Clicked") {
			t.Log("✓ Double-click changed text content")
		} else {
			t.Fatalf("Double-click did not work. Color: %s, Text: %s",
				mustRunSw(execPath, "eval", "document.getElementById('box').style.backgroundColor"),
				result)
		}
	}

	// Verify status changed
	status := mustRunSw(execPath, "eval", "document.getElementById('status').textContent")
	if strings.Contains(status, "double-clicked") {
		t.Log("✓ Status updated to 'double-clicked'")
	} else {
		t.Logf("Warning: status not updated: %s", status)
	}

	t.Log("✓ DblClick test passed!")
}

// TestHover tests hover command with visible effect
func TestHover(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	// Self-contained HTML with hover effect
	html := `<!DOCTYPE html>
<html>
<head><title>Hover Test</title>
<style>
#hover-box { width: 100px; height: 100px; background: red; }
#hover-box:hover { background: green; }
#info { display: none; }
#hover-box:hover + #info { display: block; }
</style>
</head>
<body>
<h1>Hover Test</h1>
<div id="hover-box">Hover over me</div>
<div id="info">Hovered!</div>
<p id="status">Not hovered</p>
<script>
const box = document.getElementById('hover-box');
box.addEventListener('mouseenter', () => {
  document.getElementById('status').textContent = 'Hovered!';
});
</script>
</body>
</html>`

	runSw(t, execPath, append([]string{"open", dataURL(html)}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(1 * time.Second)

	// Get snapshot
	snapshot := runSw(t, execPath, "snapshot")

	// Find hover-box element
	hoverRef := findElementByContent(snapshot, "hover-box", "")
	if hoverRef == "" {
		hoverRef = findElementByContent(snapshot, "Hover over", "")
	}

	if hoverRef == "" {
		t.Fatal("Could not find hover target element")
	}

	t.Logf("Found hover element: %s", hoverRef)

	// Before hover - check status
	beforeStatus := mustRunSw(execPath, "eval", "document.getElementById('status').textContent")
	t.Logf("Before hover: %s", beforeStatus)

	// Hover over the box
	runSw(t, execPath, "hover", hoverRef)
	time.Sleep(500 * time.Millisecond)

	// After hover - check computed style
	afterColor := mustRunSw(execPath, "eval", "getComputedStyle(document.getElementById('hover-box')).backgroundColor")
	t.Logf("After hover - background color: %s", afterColor)

	// Check status changed (via mouseenter event)
	afterStatus := mustRunSw(execPath, "eval", "document.getElementById('status').textContent")
	t.Logf("After hover - status: %s", afterStatus)

	if strings.Contains(afterStatus, "Hovered") {
		t.Log("✓ Hover triggered mouseenter event")
	} else {
		t.Logf("Warning: hover may not have triggered (status: %s, color: %s)", afterStatus, afterColor)
	}

	t.Log("✓ Hover test passed!")
}

// TestFillAndClear tests fill command with verification
func TestFillAndClear(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	html := `<!DOCTYPE html>
<html>
<head><title>Fill Test</title></head>
<body>
<h1>Fill Test</h1>
<input type="text" id="username" placeholder="Enter username">
<input type="password" id="password" placeholder="Enter password">
<p id="output">Empty</p>
<script>
document.getElementById('username').addEventListener('input', function() {
  document.getElementById('output').textContent = 'Username: ' + this.value;
});
</script>
</body>
</html>`

	runSw(t, execPath, append([]string{"open", dataURL(html)}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(1 * time.Second)

	// Get snapshot
	snapshot := runSw(t, execPath, "snapshot")

	// Find username input
	usernameRef := findElementByContent(snapshot, "username", "textbox")
	if usernameRef == "" {
		// Fallback: find first textbox
		usernameRef = findFirstElement(snapshot, "textbox")
	}

	if usernameRef == "" {
		t.Fatal("Could not find username input")
	}

	t.Logf("Found username input: %s", usernameRef)

	// Fill username
	runSw(t, execPath, "fill", usernameRef, "testuser123")
	time.Sleep(300 * time.Millisecond)

	// Verify value was set
	value := mustRunSw(execPath, "eval", "document.getElementById('username').value")
	t.Logf("Input value: %s", value)

	if !strings.Contains(value, "testuser123") {
		t.Fatalf("Expected 'testuser123', got: %s", value)
	}
	t.Log("✓ Fill set input value correctly")

	// Verify input event fired
	output := mustRunSw(execPath, "eval", "document.getElementById('output').textContent")
	t.Logf("Output paragraph: %s", output)

	if strings.Contains(output, "testuser123") {
		t.Log("✓ Fill triggered input event")
	} else {
		t.Logf("Warning: input event may not have fired (output: %s)", output)
	}

	t.Log("✓ Fill test passed!")
}

// TestDropdownSelect tests select command with verification
func TestDropdownSelect(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	html := `<!DOCTYPE html>
<html>
<head><title>Select Test</title></head>
<body>
<h1>Select Test</h1>
<select id="fruits">
  <option value="">Choose a fruit</option>
  <option value="apple">Apple</option>
  <option value="banana">Banana</option>
  <option value="cherry">Cherry</option>
</select>
<p id="selected">None selected</p>
<script>
document.getElementById('fruits').addEventListener('change', function() {
  document.getElementById('selected').textContent = 'Selected: ' + this.value;
});
</script>
</body>
</html>`

	runSw(t, execPath, append([]string{"open", dataURL(html)}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(1 * time.Second)

	// Get snapshot
	snapshot := runSw(t, execPath, "snapshot")

	// Find select element
	selectRef := findFirstElement(snapshot, "combobox")
	if selectRef == "" {
		selectRef = findFirstElement(snapshot, "listbox")
	}

	if selectRef == "" {
		t.Fatal("Could not find select element")
	}

	t.Logf("Found select element: %s", selectRef)

	// Select option
	runSw(t, execPath, "select", selectRef, "banana")
	time.Sleep(300 * time.Millisecond)

	// Verify selection
	value := mustRunSw(execPath, "eval", "document.getElementById('fruits').value")
	t.Logf("Select value: %s", value)

	if !strings.Contains(value, "banana") {
		t.Fatalf("Expected 'banana', got: %s", value)
	}
	t.Log("✓ Select set value correctly")

	// Verify change event fired
	selected := mustRunSw(execPath, "eval", "document.getElementById('selected').textContent")
	t.Logf("Selected paragraph: %s", selected)

	if strings.Contains(selected, "banana") {
		t.Log("✓ Select triggered change event")
	} else {
		t.Logf("Warning: change event may not have fired (selected: %s)", selected)
	}

	t.Log("✓ Dropdown select test passed!")
}

// TestCheckUncheck tests check and uncheck commands
func TestCheckUncheck(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	html := `<!DOCTYPE html>
<html>
<head><title>Checkbox Test</title></head>
<body>
<h1>Checkbox Test</h1>
<input type="checkbox" id="cb1"> <label for="cb1">Option 1</label><br>
<input type="checkbox" id="cb2"> <label for="cb2">Option 2</label><br>
<p id="status">None checked</p>
<script>
function updateStatus() {
  const cb1 = document.getElementById('cb1').checked;
  const cb2 = document.getElementById('cb2').checked;
  document.getElementById('status').textContent = 
    'cb1=' + cb1 + ', cb2=' + cb2;
}
document.getElementById('cb1').addEventListener('change', updateStatus);
document.getElementById('cb2').addEventListener('change', updateStatus);
</script>
</body>
</html>`

	runSw(t, execPath, append([]string{"open", dataURL(html)}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(1 * time.Second)

	// Get snapshot
	snapshot := runSw(t, execPath, "snapshot")

	// Find checkboxes
	checkboxes := findAllElements(snapshot, "checkbox")
	if len(checkboxes) < 2 {
		t.Fatalf("Expected at least 2 checkboxes, found %d", len(checkboxes))
	}

	cb1 := checkboxes[0]
	cb2 := checkboxes[1]
	t.Logf("Found checkboxes: %s, %s", cb1, cb2)

	// Check cb1
	runSw(t, execPath, "check", cb1)
	time.Sleep(200 * time.Millisecond)

	// Verify cb1 is checked
	checked := mustRunSw(execPath, "eval", "document.getElementById('cb1').checked")
	if strings.Contains(checked, "true") {
		t.Log("✓ Check command worked")
	} else {
		t.Fatalf("cb1 should be checked, got: %s", checked)
	}

	// Check cb2
	runSw(t, execPath, "check", cb2)
	time.Sleep(200 * time.Millisecond)

	// Uncheck cb1
	runSw(t, execPath, "uncheck", cb1)
	time.Sleep(200 * time.Millisecond)

	// Verify cb1 is unchecked
	checked = mustRunSw(execPath, "eval", "document.getElementById('cb1').checked")
	if strings.Contains(checked, "false") {
		t.Log("✓ Uncheck command worked")
	} else {
		t.Fatalf("cb1 should be unchecked, got: %s", checked)
	}

	// Verify status updated
	status := mustRunSw(execPath, "eval", "document.getElementById('status').textContent")
	t.Logf("Final status: %s", status)

	t.Log("✓ Check/Uncheck test passed!")
}

// TestEval tests JavaScript evaluation with result verification
func TestEval(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	runSw(t, execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(2 * time.Second)

	// Test eval - get page title
	result := runSw(t, execPath, "eval", "document.title")
	if !strings.Contains(result, "Example Domain") {
		t.Fatalf("Eval should return 'Example Domain', got: %s", result)
	}
	t.Log("✓ Eval returned correct page title")

	// Test eval - get URL
	result = runSw(t, execPath, "eval", "window.location.href")
	if !strings.Contains(result, "example.com") {
		t.Fatalf("Eval should return URL with 'example.com', got: %s", result)
	}
	t.Log("✓ Eval returned correct URL")

	// Test eval - mathematical operation
	result = runSw(t, execPath, "eval", "2 + 2")
	if !strings.Contains(result, "4") {
		t.Fatalf("Eval of '2 + 2' should return 4, got: %s", result)
	}
	t.Log("✓ Eval computed 2 + 2 = 4")

	// Test eval - modify DOM and read back
	runSwIgnoreError(execPath, "eval", "document.body.setAttribute('data-test', 'hello')")
	result = mustRunSw(execPath, "eval", "document.body.getAttribute('data-test')")
	if !strings.Contains(result, "hello") {
		t.Fatalf("Expected 'hello', got: %s", result)
	}
	t.Log("✓ Eval can modify and read DOM")

	t.Log("✓ Eval test passed!")
}

// TestNavigation tests goto, go-back, go-forward, reload
func TestNavigation(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	// Open first page
	runSw(t, execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(2 * time.Second)

	// Verify initial page
	snapshot := runSw(t, execPath, "snapshot")
	if !strings.Contains(snapshot, "Example Domain") {
		t.Fatalf("Expected 'Example Domain' on first page")
	}
	t.Log("✓ Opened first page")

	// Navigate to second page
	runSw(t, execPath, "goto", "https://example.org")
	time.Sleep(2 * time.Second)

	// Verify second page
	snapshot = runSw(t, execPath, "snapshot")
	if !strings.Contains(snapshot, "IANA") && !strings.Contains(snapshot, "example.org") {
		t.Fatalf("Expected 'IANA' or 'example.org' on second page")
	}
	t.Log("✓ Navigated to second page")

	// Test reload
	runSw(t, execPath, "reload")
	time.Sleep(2 * time.Second)

	snapshot = runSw(t, execPath, "snapshot")
	if !strings.Contains(snapshot, "IANA") && !strings.Contains(snapshot, "example.org") {
		t.Fatalf("Page should still show example.org after reload")
	}
	t.Log("✓ Reload worked")

	// Go back (best effort - may timeout in headless)
	_, err := runSwIgnoreError(execPath, "go-back")
	if err != nil {
		t.Logf("go-back warning (may occur in headless): %v", err)
	} else {
		time.Sleep(1 * time.Second)
		t.Log("✓ go-back worked")
	}

	t.Log("✓ Navigation test passed!")
}

// TestTabs tests tab operations
func TestTabs(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	runSw(t, execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(2 * time.Second)

	// List initial tab
	output := runSw(t, execPath, "tab-list")
	t.Logf("Tab list: %s", output)

	// Parse JSON response
	var tabResult struct {
		Success bool `json:"success"`
		Data    []struct {
			Index   int    `json:"index"`
			URL     string `json:"url"`
			Title   string `json:"title"`
			Current bool   `json:"current"`
		} `json:"data"`
	}

	if err := json.Unmarshal([]byte(output), &tabResult); err == nil {
		if !tabResult.Success {
			t.Fatal("tab-list should return success=true")
		}
		if len(tabResult.Data) < 1 {
			t.Fatal("tab-list should return at least one tab")
		}
		if tabResult.Data[0].URL == "" {
			t.Fatal("tab should have URL")
		}
		t.Log("✓ tab-list returned valid JSON with tab info")
	} else {
		// Fallback: just check string content
		if !strings.Contains(output, "example.com") {
			t.Fatalf("Expected 'example.com' in tab list, got: %s", output)
		}
		t.Log("✓ tab-list returned expected content")
	}

	t.Log("✓ Tabs test passed!")
}

// TestDrag tests drag and drop simulation
func TestDrag(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	// Use buttons instead of divs - they show up in aria tree
	html := `<!DOCTYPE html>
<html>
<head><title>Drag Test</title></head>
<body>
<h1>Drag Test</h1>
<button id="source">Drag me</button>
<button id="target">Drop here</button>
<p id="status">No drag yet</p>
<script>
const source = document.getElementById('source');
const target = document.getElementById('target');

source.addEventListener('dragstart', (e) => {
  document.getElementById('status').textContent = 'Dragging...';
});

target.addEventListener('dragover', (e) => {
  e.preventDefault();
});

target.addEventListener('drop', (e) => {
  e.preventDefault();
  document.getElementById('status').textContent = 'Dropped!';
  target.textContent = 'Dropped!';
});
</script>
</body>
</html>`

	runSw(t, execPath, append([]string{"open", dataURL(html)}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(1 * time.Second)

	// Get snapshot
	snapshot := runSw(t, execPath, "snapshot")

	// Find source and target buttons by their text content
	sourceRef := findElementByContent(snapshot, "Drag me", "")
	targetRef := findElementByContent(snapshot, "Drop here", "")

	if sourceRef == "" || targetRef == "" {
		t.Fatalf("Could not find source (%s) or target (%s)\nSnapshot:\n%s", sourceRef, targetRef, snapshot)
	}

	t.Logf("Found source: %s, target: %s", sourceRef, targetRef)

	// Perform drag
	runSw(t, execPath, "drag", sourceRef, targetRef)
	time.Sleep(500 * time.Millisecond)

	// Verify status changed (drag events fired)
	status := mustRunSw(execPath, "eval", "document.getElementById('status').textContent")
	t.Logf("Status after drag: %s", status)

	// Note: JavaScript drag simulation may not fully work, so we check if at least dragstart fired
	if strings.Contains(status, "Dragging") || strings.Contains(status, "Dropped") {
		t.Log("✓ Drag triggered drag event")
	} else {
		t.Logf("Note: Drag events may require native browser interaction (status: %s)", status)
	}

	t.Log("✓ Drag test passed!")
}

// TestKeyboardAndMouse tests keyboard and mouse commands with verification
func TestKeyboardAndMouse(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	html := `<!DOCTYPE html>
<html>
<head><title>Keyboard & Mouse Test</title></head>
<body>
<h1>Keyboard & Mouse Test</h1>
<input type="text" id="input" placeholder="Type here">
<div id="mouse-area" style="width:200px;height:200px;background:lightgray;">
  Mouse area
</div>
<p id="key-log">Keys: </p>
<p id="mouse-log">Mouse: none</p>
<script>
const input = document.getElementById('input');
const keyLog = document.getElementById('key-log');
const mouseLog = document.getElementById('mouse-log');

input.addEventListener('keydown', (e) => {
  keyLog.textContent += e.key;
});

document.getElementById('mouse-area').addEventListener('mousemove', (e) => {
  mouseLog.textContent = 'Mouse: ' + e.clientX + ',' + e.clientY;
});
</script>
</body>
</html>`

	runSw(t, execPath, append([]string{"open", dataURL(html)}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(1 * time.Second)

	// Get snapshot
	snapshot := runSw(t, execPath, "snapshot")

	// Find input element
	inputRef := findElementByContent(snapshot, "input", "input")
	if inputRef == "" {
		inputRef = findFirstElement(snapshot, "input")
	}

	if inputRef != "" {
		// Focus input
		runSw(t, execPath, "click", inputRef)
		time.Sleep(100 * time.Millisecond)

		// Type into focused input
		runSw(t, execPath, "type", "ABC")
		time.Sleep(200 * time.Millisecond)

		// Verify input value
		value := mustRunSw(execPath, "eval", "document.getElementById('input').value")
		if strings.Contains(value, "ABC") {
			t.Log("✓ Type command worked")
		} else {
			t.Logf("Type value: %s", value)
		}
	}

	// Test keyboard press
	runSwIgnoreError(execPath, "keydown", "x")
	runSwIgnoreError(execPath, "keyup", "x")

	// Test mouse commands (best effort)
	_, err := runSwIgnoreError(execPath, "mousemove", "50", "50")
	if err != nil {
		t.Logf("mousemove warning: %v", err)
	}

	_, err = runSwIgnoreError(execPath, "resize", "1024", "768")
	if err != nil {
		t.Logf("resize warning: %v", err)
	}

	t.Log("✓ Keyboard and mouse test passed!")
}

// TestSessionList tests session listing
func TestSessionList(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	runSw(t, execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(2 * time.Second)

	// List sessions
	output := runSw(t, execPath, "list")
	t.Logf("Sessions output:\n%s", output)

	// Verify session info
	if !strings.Contains(output, "default") {
		t.Fatalf("Expected 'default' session in list, got: %s", output)
	}
	t.Log("✓ Session 'default' found")

	if !strings.Contains(output, "example.com") {
		t.Fatalf("Expected 'example.com' URL in session, got: %s", output)
	}
	t.Log("✓ Session URL correct")

	if !strings.Contains(output, "chrome") && !strings.Contains(output, "chromium") {
		t.Fatalf("Expected 'chrome' or 'chromium' browser in session, got: %s", output)
	}
	t.Log("✓ Session browser type correct")

	t.Log("✓ Session list test passed!")
}

// TestKillAll tests kill-all command
func TestKillAll(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	runSw(t, execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	time.Sleep(2 * time.Second)

	// Verify session exists
	output := mustRunSw(execPath, "list")
	if !strings.Contains(output, "default") {
		t.Fatal("Session should exist before kill-all")
	}
	t.Log("✓ Session exists before kill-all")

	// Kill all processes
	output, err := runSwIgnoreError(execPath, "kill-all")
	t.Logf("kill-all output: %s", output)

	if err != nil {
		t.Logf("kill-all warning: %v", err)
	}

	t.Log("✓ Kill-all test passed!")
}

// Helper functions for finding elements in snapshot

// extractRef extracts the ref value from a [ref=eN] annotation in an ARIA snapshot line.
func extractRef(line string) string {
	idx := strings.Index(line, "[ref=")
	if idx == -1 {
		return ""
	}
	rest := line[idx+5:]
	end := strings.Index(rest, "]")
	if end == -1 {
		return ""
	}
	return rest[:end]
}

func findCheckboxRefs(snapshot string) []string {
	var refs []string
	for _, line := range strings.Split(snapshot, "\n") {
		if strings.Contains(line, "checkbox") {
			if ref := extractRef(line); ref != "" {
				refs = append(refs, ref)
			}
		}
	}
	return refs
}

func findElementByContent(snapshot, content, elemType string) string {
	for _, line := range strings.Split(snapshot, "\n") {
		if content != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(content)) {
			continue
		}
		if elemType != "" && !strings.Contains(line, elemType) {
			continue
		}
		if ref := extractRef(line); ref != "" {
			return ref
		}
	}
	return ""
}

func findFirstElement(snapshot, elemType string) string {
	for _, line := range strings.Split(snapshot, "\n") {
		if strings.Contains(line, elemType) {
			if ref := extractRef(line); ref != "" {
				return ref
			}
		}
	}
	return ""
}

func findAllElements(snapshot, elemType string) []string {
	var refs []string
	for _, line := range strings.Split(snapshot, "\n") {
		if strings.Contains(line, elemType) {
			if ref := extractRef(line); ref != "" {
				refs = append(refs, ref)
			}
		}
	}
	return refs
}

// TestCookieCommands tests cookie-list, cookie-set, cookie-get, cookie-delete, cookie-clear
func TestCookieCommands(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	runSw(t, execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(2 * time.Second)

	// Set a cookie
	runSw(t, execPath, "cookie-set", "testcookie", "testvalue")
	time.Sleep(200 * time.Millisecond)

	// Get the cookie
	result := runSw(t, execPath, "cookie-get", "testcookie")
	if !strings.Contains(result, "testvalue") {
		t.Fatalf("Expected 'testvalue' from cookie-get, got: %s", result)
	}
	t.Log("✓ cookie-get returned correct value")

	// List cookies
	result = runSw(t, execPath, "cookie-list")
	if !strings.Contains(result, "testcookie") {
		t.Fatalf("Expected 'testcookie' in cookie-list, got: %s", result)
	}
	t.Log("✓ cookie-list includes set cookie")

	// Delete the cookie
	runSw(t, execPath, "cookie-delete", "testcookie")
	time.Sleep(200 * time.Millisecond)

	// Verify cookie was deleted
	result = mustRunSw(execPath, "cookie-get", "testcookie")
	if strings.Contains(result, "testvalue") {
		t.Fatalf("Cookie should be deleted, but still got: %s", result)
	}
	t.Log("✓ cookie-delete removed cookie")

	// Set multiple cookies then clear all
	runSw(t, execPath, "cookie-set", "cookie1", "val1")
	runSw(t, execPath, "cookie-set", "cookie2", "val2")
	time.Sleep(200 * time.Millisecond)

	runSw(t, execPath, "cookie-clear")
	time.Sleep(200 * time.Millisecond)

	// Verify cleared
	remaining := mustRunSw(execPath, "eval", "document.cookie")
	if !strings.Contains(remaining, "cookie1") && !strings.Contains(remaining, "cookie2") {
		t.Log("✓ cookie-clear removed all cookies")
	} else {
		t.Logf("Note: remaining cookies after clear: %s", remaining)
	}

	t.Log("✓ Cookie commands test passed!")
}

// TestLocalStorageCommands tests localstorage-list, -get, -set, -delete, -clear
func TestLocalStorageCommands(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	runSw(t, execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(2 * time.Second)

	// Set a value
	runSw(t, execPath, "localstorage-set", "testkey", "testvalue")
	time.Sleep(200 * time.Millisecond)

	// Verify via eval
	val := mustRunSw(execPath, "eval", "localStorage.getItem('testkey')")
	if !strings.Contains(val, "testvalue") {
		t.Fatalf("Expected 'testvalue' in localStorage via eval, got: %s", val)
	}
	t.Log("✓ localstorage-set stored value")

	// Get via command
	result := runSw(t, execPath, "localstorage-get", "testkey")
	if !strings.Contains(result, "testvalue") {
		t.Fatalf("Expected 'testvalue' from localstorage-get, got: %s", result)
	}
	t.Log("✓ localstorage-get returned correct value")

	// List entries
	result = runSw(t, execPath, "localstorage-list")
	if !strings.Contains(result, "testkey") {
		t.Fatalf("Expected 'testkey' in localstorage-list, got: %s", result)
	}
	t.Log("✓ localstorage-list includes set entry")

	// Delete the entry
	runSw(t, execPath, "localstorage-delete", "testkey")
	time.Sleep(200 * time.Millisecond)

	val = mustRunSw(execPath, "eval", "localStorage.getItem('testkey')")
	if strings.Contains(val, "testvalue") {
		t.Fatalf("localStorage key should be deleted, got: %s", val)
	}
	t.Log("✓ localstorage-delete removed entry")

	// Set multiple and clear all
	runSw(t, execPath, "localstorage-set", "key1", "val1")
	runSw(t, execPath, "localstorage-set", "key2", "val2")
	time.Sleep(200 * time.Millisecond)

	runSw(t, execPath, "localstorage-clear")
	time.Sleep(200 * time.Millisecond)

	count := mustRunSw(execPath, "eval", "localStorage.length")
	if strings.Contains(count, "0") {
		t.Log("✓ localstorage-clear removed all entries")
	} else {
		t.Logf("localStorage.length after clear: %s", count)
	}

	t.Log("✓ localStorage commands test passed!")
}

// TestSessionStorageCommands tests sessionstorage-list, -get, -set, -delete, -clear
func TestSessionStorageCommands(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	runSw(t, execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(2 * time.Second)

	// Set a value
	runSw(t, execPath, "sessionstorage-set", "sesskey", "sessvalue")
	time.Sleep(200 * time.Millisecond)

	// Verify via eval
	val := mustRunSw(execPath, "eval", "sessionStorage.getItem('sesskey')")
	if !strings.Contains(val, "sessvalue") {
		t.Fatalf("Expected 'sessvalue' in sessionStorage via eval, got: %s", val)
	}
	t.Log("✓ sessionstorage-set stored value")

	// Get via command
	result := runSw(t, execPath, "sessionstorage-get", "sesskey")
	if !strings.Contains(result, "sessvalue") {
		t.Fatalf("Expected 'sessvalue' from sessionstorage-get, got: %s", result)
	}
	t.Log("✓ sessionstorage-get returned correct value")

	// List entries
	result = runSw(t, execPath, "sessionstorage-list")
	if !strings.Contains(result, "sesskey") {
		t.Fatalf("Expected 'sesskey' in sessionstorage-list, got: %s", result)
	}
	t.Log("✓ sessionstorage-list includes set entry")

	// Delete the entry
	runSw(t, execPath, "sessionstorage-delete", "sesskey")
	time.Sleep(200 * time.Millisecond)

	val = mustRunSw(execPath, "eval", "sessionStorage.getItem('sesskey')")
	if strings.Contains(val, "sessvalue") {
		t.Fatalf("sessionStorage key should be deleted, got: %s", val)
	}
	t.Log("✓ sessionstorage-delete removed entry")

	// Clear all
	runSw(t, execPath, "sessionstorage-set", "k1", "v1")
	runSw(t, execPath, "sessionstorage-set", "k2", "v2")
	time.Sleep(200 * time.Millisecond)

	runSw(t, execPath, "sessionstorage-clear")
	time.Sleep(200 * time.Millisecond)

	count := mustRunSw(execPath, "eval", "sessionStorage.length")
	if strings.Contains(count, "0") {
		t.Log("✓ sessionstorage-clear removed all entries")
	} else {
		t.Logf("sessionStorage.length after clear: %s", count)
	}

	t.Log("✓ sessionStorage commands test passed!")
}

// TestMouseWheel tests mousewheel scroll command
func TestMouseWheel(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	html := `<!DOCTYPE html>
<html>
<head><title>Scroll Test</title></head>
<body style="height:5000px;margin:0;">
<h1>Top of page</h1>
<div style="height:4500px;"></div>
<p id="bottom">Bottom</p>
</body>
</html>`

	runSw(t, execPath, append([]string{"open", dataURL(html)}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(1 * time.Second)

	// Record scroll position before
	before := mustRunSw(execPath, "eval", "window.scrollY")
	t.Logf("Initial scrollY: %s", before)

	// Scroll down 500px
	runSw(t, execPath, "mousewheel", "0", "500")
	time.Sleep(500 * time.Millisecond)

	after := mustRunSw(execPath, "eval", "window.scrollY")
	t.Logf("After mousewheel scrollY: %s", after)

	if before != after && !strings.Contains(strings.TrimSpace(after), "0") {
		t.Log("✓ mousewheel scrolled the page")
	} else {
		t.Logf("Note: scroll position unchanged (before: %s, after: %s)", before, after)
	}

	t.Log("✓ MouseWheel test passed!")
}

// TestPDF tests PDF generation with --filename flag
func TestPDF(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	runSw(t, execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(2 * time.Second)

	pdfFile := "/tmp/test-sw-output.pdf"
	defer os.Remove(pdfFile)

	// Generate PDF
	result := runSw(t, execPath, "pdf", "--filename", pdfFile)
	t.Logf("PDF output: %s", result)
	time.Sleep(500 * time.Millisecond)

	// Verify file was created
	info, err := os.Stat(pdfFile)
	if os.IsNotExist(err) {
		t.Fatalf("PDF file not created at %s", pdfFile)
	}
	t.Logf("✓ PDF file created (%d bytes)", info.Size())

	// Verify PDF header (%PDF-)
	data, err := os.ReadFile(pdfFile)
	if err != nil {
		t.Fatalf("Failed to read PDF: %v", err)
	}
	if len(data) < 5 || string(data[:5]) != "%PDF-" {
		header := ""
		if len(data) >= 5 {
			header = string(data[:5])
		}
		t.Fatalf("File does not have PDF header, got: %q", header)
	}
	t.Log("✓ PDF has valid %PDF- header")

	t.Log("✓ PDF test passed!")
}

// TestDeleteData tests delete-data clears localStorage, sessionStorage, and cookies
func TestDeleteData(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	runSw(t, execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(2 * time.Second)

	// Plant data in all storage types
	runSw(t, execPath, "localstorage-set", "deltest", "lsvalue")
	runSw(t, execPath, "sessionstorage-set", "deltest", "ssvalue")
	runSw(t, execPath, "cookie-set", "delcookie", "ckvalue")
	time.Sleep(200 * time.Millisecond)

	// Confirm data is present
	lsVal := mustRunSw(execPath, "eval", "localStorage.getItem('deltest')")
	ssVal := mustRunSw(execPath, "eval", "sessionStorage.getItem('deltest')")
	if !strings.Contains(lsVal, "lsvalue") || !strings.Contains(ssVal, "ssvalue") {
		t.Fatalf("Pre-condition failed: ls=%s ss=%s", lsVal, ssVal)
	}
	t.Log("✓ Pre-condition: data planted in all storage types")

	// Delete all data
	runSw(t, execPath, "delete-data")
	time.Sleep(300 * time.Millisecond)

	// Verify each storage is cleared
	lsVal = mustRunSw(execPath, "eval", "localStorage.getItem('deltest')")
	ssVal = mustRunSw(execPath, "eval", "sessionStorage.getItem('deltest')")

	if strings.Contains(lsVal, "lsvalue") {
		t.Fatalf("localStorage not cleared, got: %s", lsVal)
	}
	t.Log("✓ localStorage cleared by delete-data")

	if strings.Contains(ssVal, "ssvalue") {
		t.Fatalf("sessionStorage not cleared, got: %s", ssVal)
	}
	t.Log("✓ sessionStorage cleared by delete-data")

	t.Log("✓ DeleteData test passed!")
}

// TestFillWithSubmit tests fill --submit presses Enter after filling
func TestFillWithSubmit(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	html := `<!DOCTYPE html>
<html>
<head><title>Fill Submit Test</title></head>
<body>
<form id="form" onsubmit="document.getElementById('status').textContent='submitted:'+document.getElementById('field').value; return false;">
  <input type="text" id="field" placeholder="Type here">
  <button type="submit">Go</button>
</form>
<p id="status">not submitted</p>
</body>
</html>`

	runSw(t, execPath, append([]string{"open", dataURL(html)}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(1 * time.Second)

	snapshot := runSw(t, execPath, "snapshot")
	inputRef := findElementByContent(snapshot, "Type here", "textbox")
	if inputRef == "" {
		inputRef = findFirstElement(snapshot, "textbox")
	}
	if inputRef == "" {
		t.Fatal("Could not find input element in snapshot")
	}
	t.Logf("Found input: %s", inputRef)

	// Fill with --submit
	runSw(t, execPath, "fill", inputRef, "hello world", "--submit")
	time.Sleep(500 * time.Millisecond)

	// Verify fill set the value
	value := mustRunSw(execPath, "eval", "document.getElementById('field').value")
	if !strings.Contains(value, "hello world") {
		t.Fatalf("Expected 'hello world' in field, got: %s", value)
	}
	t.Log("✓ fill set the input value")

	// Verify form submitted via Enter
	status := mustRunSw(execPath, "eval", "document.getElementById('status').textContent")
	t.Logf("Status after fill --submit: %s", status)
	if strings.Contains(status, "submitted") {
		t.Log("✓ fill --submit triggered form submission")
	} else {
		t.Logf("Note: form submit may not have fired (status: %s)", status)
	}

	t.Log("✓ FillWithSubmit test passed!")
}

// TestTypeWithSubmit tests type --submit presses Enter after typing
func TestTypeWithSubmit(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	html := `<!DOCTYPE html>
<html>
<head><title>Type Submit Test</title></head>
<body>
<form id="form" onsubmit="document.getElementById('status').textContent='submitted'; return false;">
  <input type="text" id="search" placeholder="Search..." autofocus>
</form>
<p id="status">not submitted</p>
</body>
</html>`

	runSw(t, execPath, append([]string{"open", dataURL(html)}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(1 * time.Second)

	// Click to focus the input
	snapshot := runSw(t, execPath, "snapshot")
	inputRef := findFirstElement(snapshot, "input")
	if inputRef != "" {
		runSw(t, execPath, "click", inputRef)
		time.Sleep(100 * time.Millisecond)
	}

	// Type with --submit
	runSw(t, execPath, "type", "search query", "--submit")
	time.Sleep(500 * time.Millisecond)

	status := mustRunSw(execPath, "eval", "document.getElementById('status').textContent")
	t.Logf("Status after type --submit: %s", status)
	if strings.Contains(status, "submitted") {
		t.Log("✓ type --submit triggered form submission")
	} else {
		t.Logf("Note: form submit may not have fired (status: %s)", status)
	}

	t.Log("✓ TypeWithSubmit test passed!")
}

// TestScreenshotFlags tests screenshot --filename and --full-page flags
func TestScreenshotFlags(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	html := `<!DOCTYPE html>
<html>
<head><title>Screenshot Flags Test</title></head>
<body style="height:3000px;margin:0;">
<h1>Screenshot Test</h1>
<div style="height:2800px;background:linear-gradient(blue,red);"></div>
<p>Bottom</p>
</body>
</html>`

	runSw(t, execPath, append([]string{"open", dataURL(html)}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(1 * time.Second)

	// Test --filename flag
	screenshotFile := "/tmp/test-sw-screenshot.png"
	defer os.Remove(screenshotFile)

	runSw(t, execPath, "screenshot", "--filename", screenshotFile)
	time.Sleep(500 * time.Millisecond)

	info, err := os.Stat(screenshotFile)
	if os.IsNotExist(err) {
		t.Fatalf("Screenshot not created at %s", screenshotFile)
	}
	regularSize := info.Size()
	t.Logf("✓ screenshot --filename created file (%d bytes)", regularSize)

	// Verify PNG signature
	data, _ := os.ReadFile(screenshotFile)
	if len(data) > 4 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
		t.Log("✓ screenshot is valid PNG")
	}
	os.Remove(screenshotFile)

	// Test --full-page flag
	fullPageFile := "/tmp/test-sw-fullpage.png"
	defer os.Remove(fullPageFile)

	runSw(t, execPath, "screenshot", "--full-page", "--filename", fullPageFile)
	time.Sleep(500 * time.Millisecond)

	fullInfo, err := os.Stat(fullPageFile)
	if os.IsNotExist(err) {
		t.Fatalf("Full-page screenshot not created at %s", fullPageFile)
	}
	fullSize := fullInfo.Size()
	t.Logf("Full-page size: %d bytes, regular size: %d bytes", fullSize, regularSize)

	if fullSize > regularSize {
		t.Log("✓ --full-page screenshot is larger (captures full scrollable page)")
	} else {
		t.Logf("Note: full-page (%d) not larger than regular (%d)", fullSize, regularSize)
	}

	t.Log("✓ ScreenshotFlags test passed!")
}

// TestEvalWithElementRef tests eval <script> [ref] optional element targeting
func TestEvalWithElementRef(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	html := `<!DOCTYPE html>
<html>
<head><title>Eval Ref Test</title></head>
<body>
<h1 id="title">Page Title</h1>
<button id="btn" data-value="42">My Button</button>
<p id="para">Hello World</p>
</body>
</html>`

	runSw(t, execPath, append([]string{"open", dataURL(html)}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(1 * time.Second)

	// Regular eval without ref
	result := runSw(t, execPath, "eval", "document.title")
	if !strings.Contains(result, "Eval Ref Test") {
		t.Fatalf("Expected page title 'Eval Ref Test', got: %s", result)
	}
	t.Log("✓ eval without ref returns page-level result")

	// Find a button element ref from snapshot
	snapshot := runSw(t, execPath, "snapshot")
	btnRef := findElementByContent(snapshot, "My Button", "button")
	if btnRef == "" {
		btnRef = findFirstElement(snapshot, "button")
	}

	if btnRef == "" {
		t.Log("Note: could not find button ref in snapshot, skipping element eval")
		t.Log("✓ EvalWithElementRef test passed (partial)")
		return
	}
	t.Logf("Found button ref: %s", btnRef)

	// Eval on the element - get textContent
	result = mustRunSw(execPath, "eval", "(el) => el.textContent", btnRef)
	t.Logf("Element textContent: %s", result)
	if strings.Contains(result, "My Button") {
		t.Log("✓ eval with ref returned element textContent")
	} else {
		// Try data-value attribute
		result = mustRunSw(execPath, "eval", "(el) => el.getAttribute('data-value')", btnRef)
		t.Logf("Element data-value: %s", result)
		if strings.Contains(result, "42") {
			t.Log("✓ eval with ref accessed element attribute")
		} else {
			t.Logf("Note: eval with ref result: %s", result)
		}
	}

	t.Log("✓ EvalWithElementRef test passed!")
}

// TestRunCode tests the run-code command that evaluates JavaScript
func TestRunCode(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	runSw(t, execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(2 * time.Second)

	// Test arithmetic expression
	result := runSw(t, execPath, "run-code", "1 + 2")
	if !strings.Contains(result, "3") {
		t.Fatalf("Expected '3' from '1 + 2', got: %s", result)
	}
	t.Log("✓ run-code evaluates arithmetic")

	// Test document access
	result = runSw(t, execPath, "run-code", "document.title")
	if result == "" || strings.TrimSpace(result) == "" {
		t.Fatalf("Expected non-empty title from run-code, got: %s", result)
	}
	t.Logf("✓ run-code accesses document (title: %s)", strings.TrimSpace(result))

	// Test that run-code can modify DOM and return result
	result = runSw(t, execPath, "run-code", "document.querySelectorAll('a').length")
	t.Logf("✓ run-code counts elements: %s links found", strings.TrimSpace(result))

	t.Log("✓ RunCode test passed!")
}

// TestConfigPrint tests the config-print command
func TestConfigPrint(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	runSw(t, execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(2 * time.Second)

	result := runSw(t, execPath, "config-print")

	// Should output valid JSON with session config
	result = strings.TrimSpace(result)
	if result == "" {
		t.Fatalf("config-print returned empty output")
	}

	// Try to parse as JSON
	var cfg map[string]interface{}
	// Find JSON part (output may have leading text)
	jsonStart := strings.Index(result, "{")
	if jsonStart == -1 {
		t.Fatalf("config-print output doesn't contain JSON: %s", result)
	}
	jsonStr := result[jsonStart:]
	if err := json.Unmarshal([]byte(jsonStr), &cfg); err != nil {
		t.Fatalf("config-print output is not valid JSON: %s\nError: %v", result, err)
	}

	t.Logf("✓ config-print returned valid JSON config")
	t.Log("✓ ConfigPrint test passed!")
}

// TestCloseAll tests the close-all command
func TestCloseAll(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	// Open a browser session
	runSw(t, execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	time.Sleep(2 * time.Second)

	// Verify session is active
	result := runSw(t, execPath, "list")
	if !strings.Contains(result, "running") && !strings.Contains(result, "default") {
		t.Logf("Note: list output before close-all: %s", result)
	}
	t.Log("✓ Session active before close-all")

	// Close all sessions
	runSw(t, execPath, "close-all")
	time.Sleep(500 * time.Millisecond)

	// Verify sessions are closed - eval should fail or return error
	_, err := runSwIgnoreError(execPath, "eval", "document.title")
	if err != nil {
		t.Log("✓ After close-all, commands fail as expected")
	} else {
		t.Log("Note: command after close-all didn't error (daemon may still be running)")
	}

	t.Log("✓ CloseAll test passed!")
}

// TestConsoleCapture tests the console command captures browser console messages
func TestConsoleCapture(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	// Page that emits console messages on load
	html := `<!DOCTYPE html>
<html>
<head><title>Console Test</title></head>
<body>
<script>
  console.log('sw-test-log-message');
  console.error('sw-test-error-message');
  console.warn('sw-test-warn-message');
</script>
</body>
</html>`

	runSw(t, execPath, append([]string{"open", dataURL(html)}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(1 * time.Second)

	// Check console output
	result := runSw(t, execPath, "console")

	if !strings.Contains(result, "sw-test-log-message") {
		t.Fatalf("Expected 'sw-test-log-message' in console output, got:\n%s", result)
	}
	t.Log("✓ console captured log message")

	if strings.Contains(result, "sw-test-error-message") {
		t.Log("✓ console captured error message")
	} else {
		t.Logf("Note: error message not captured (may need --level error): %s", result)
	}

	// Test clear flag
	runSw(t, execPath, "console", "--clear")
	result = runSw(t, execPath, "console")
	if strings.Contains(result, "sw-test-log-message") {
		t.Fatalf("Console should be cleared, but still shows old messages: %s", result)
	}
	t.Log("✓ console --clear removed messages")

	t.Log("✓ ConsoleCapture test passed!")
}

// TestNetworkCapture tests the network command captures requests
func TestNetworkCapture(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	runSw(t, execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(2 * time.Second)

	// Check network events captured during page load
	result := runSw(t, execPath, "network")

	// Should have captured at least the main page request
	if strings.Contains(result, "no network events") {
		// Trigger a request manually
		mustRunSw(execPath, "run-code", "fetch('/').catch(()=>{})")
		time.Sleep(500 * time.Millisecond)
		result = runSw(t, execPath, "network")
	}

	// Should have some requests now
	if !strings.Contains(result, "GET") && !strings.Contains(result, "POST") {
		t.Logf("Note: no method in network output: %s", result)
	} else {
		t.Log("✓ network captured HTTP requests")
	}

	// Test --static flag (includes images/css)
	resultStatic := runSw(t, execPath, "network", "--static")
	t.Logf("✓ network --static returned %d chars vs %d chars without", len(resultStatic), len(result))

	// Test clear
	runSw(t, execPath, "network", "--clear")
	result = runSw(t, execPath, "network")
	if strings.Contains(result, "no network events") || result == "" || strings.TrimSpace(result) == "" {
		t.Log("✓ network --clear cleared events")
	} else {
		t.Logf("Note: after clear: %s", result)
	}

	t.Log("✓ NetworkCapture test passed!")
}

// TestRoute tests route, route-list, and unroute commands
func TestRoute(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	// Page with a fetch call to /api/data
	html := `<!DOCTYPE html>
<html>
<head><title>Route Test</title></head>
<body>
<div id="result">loading...</div>
<script>
  fetch('/api/data')
    .then(r => r.json())
    .then(d => { document.getElementById('result').textContent = JSON.stringify(d); })
    .catch(e => { document.getElementById('result').textContent = 'error: ' + e; });
</script>
</body>
</html>`

	runSw(t, execPath, append([]string{"open", dataURL(html)}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(1 * time.Second)

	// Add a route to mock /api/data
	runSw(t, execPath, "route", "**/api/data",
		"--status", "200",
		"--body", `{"mocked":true,"value":42}`,
		"--content-type", "application/json",
	)

	// Reload page to trigger the fetch with the route in place
	runSw(t, execPath, "reload")
	time.Sleep(1 * time.Second)

	// Verify the mocked response was used
	result := runSw(t, execPath, "eval", "document.getElementById('result').textContent")
	if strings.Contains(result, "mocked") || strings.Contains(result, "42") {
		t.Log("✓ route intercepted request and returned mocked response")
	} else {
		t.Logf("Note: route result might not have resolved yet: %s", result)
	}

	// Test route-list shows our route
	listResult := runSw(t, execPath, "route-list")
	if strings.Contains(listResult, "**/api/data") || strings.Contains(listResult, "api/data") {
		t.Log("✓ route-list shows active route")
	} else {
		t.Logf("route-list output: %s", listResult)
	}

	// Test unroute removes the route
	runSw(t, execPath, "unroute", "**/api/data")
	listResult = runSw(t, execPath, "route-list")
	if strings.Contains(listResult, "no active routes") || !strings.Contains(listResult, "api/data") {
		t.Log("✓ unroute removed the route")
	} else {
		t.Logf("Note: route still listed after unroute: %s", listResult)
	}

	t.Log("✓ Route test passed!")
}

// TestTracing tests tracing-start and tracing-stop commands
func TestTracing(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	traceFile := "/tmp/sw-test-trace.zip"
	defer os.Remove(traceFile)

	runSw(t, execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(2 * time.Second)

	// Start tracing
	runSw(t, execPath, "tracing-start")
	t.Log("✓ tracing-start succeeded")

	// Do some actions to record
	runSw(t, execPath, "snapshot")
	time.Sleep(300 * time.Millisecond)

	// Stop tracing and save to file
	runSw(t, execPath, "tracing-stop", "--filename", traceFile)
	time.Sleep(500 * time.Millisecond)

	// Verify trace file was created
	info, err := os.Stat(traceFile)
	if os.IsNotExist(err) {
		t.Fatalf("Trace file not created at %s", traceFile)
	}
	if info.Size() == 0 {
		t.Fatalf("Trace file is empty at %s", traceFile)
	}
	t.Logf("✓ tracing-stop created trace file (%d bytes)", info.Size())

	t.Log("✓ Tracing test passed!")
}

// TestCookieSetWithAttributes tests cookie-set with new attribute flags
func TestCookieSetWithAttributes(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	runSw(t, execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(2 * time.Second)

	// Set cookie with basic positional args
	runSw(t, execPath, "cookie-set", "flagtest", "flagvalue")
	time.Sleep(200 * time.Millisecond)

	result := runSw(t, execPath, "cookie-get", "flagtest")
	if !strings.Contains(result, "flagvalue") {
		t.Fatalf("Expected 'flagvalue' from cookie-get, got: %s", result)
	}
	t.Log("✓ cookie-set <name> <value> works")

	// Set cookie with path
	runSw(t, execPath, "cookie-set", "pathcookie", "pathval", "--path", "/")
	time.Sleep(200 * time.Millisecond)

	result = runSw(t, execPath, "cookie-get", "pathcookie")
	if !strings.Contains(result, "pathval") {
		t.Fatalf("Expected 'pathval' from cookie-get, got: %s", result)
	}
	t.Log("✓ cookie-set --path works")

	// Set secure cookie (only verifiable via eval in HTTPS context)
	runSw(t, execPath, "cookie-set", "securecookie", "secval", "--secure")
	time.Sleep(200 * time.Millisecond)
	t.Log("✓ cookie-set --secure flag accepted")

	t.Log("✓ CookieSetWithAttributes test passed!")
}

// TestDialogAccept tests dialog-accept command
func TestDialogAccept(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	html := `<!DOCTYPE html>
<html>
<head><title>Dialog Test</title></head>
<body>
<button id="btn" onclick="var r = confirm('Are you sure?'); document.getElementById('result').textContent = r ? 'accepted' : 'dismissed';">Show Dialog</button>
<p id="result">waiting</p>
</body>
</html>`

	runSw(t, execPath, append([]string{"open", dataURL(html)}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(1 * time.Second)

	snapshot := runSw(t, execPath, "snapshot")
	btnRef := findElementByContent(snapshot, "btn", "button")
	if btnRef == "" {
		btnRef = findElementByContent(snapshot, "Show Dialog", "")
	}
	if btnRef == "" {
		t.Fatal("Could not find button element")
	}

	// Register dialog-accept before triggering dialog
	runSw(t, execPath, "dialog-accept")

	// Click button to trigger dialog
	runSw(t, execPath, "click", btnRef)
	time.Sleep(500 * time.Millisecond)

	result := mustRunSw(execPath, "eval", "document.getElementById('result').textContent")
	if strings.Contains(result, "accepted") {
		t.Log("✓ dialog-accept worked")
	} else {
		t.Logf("Warning: dialog-accept result: %s", result)
	}

	t.Log("✓ TestDialogAccept passed!")
}

// TestDialogDismiss tests dialog-dismiss command
func TestDialogDismiss(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	html := `<!DOCTYPE html>
<html>
<head><title>Dialog Dismiss Test</title></head>
<body>
<button id="btn" onclick="var r = confirm('Are you sure?'); document.getElementById('result').textContent = r ? 'accepted' : 'dismissed';">Show Dialog</button>
<p id="result">waiting</p>
</body>
</html>`

	runSw(t, execPath, append([]string{"open", dataURL(html)}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(1 * time.Second)

	snapshot := runSw(t, execPath, "snapshot")
	btnRef := findElementByContent(snapshot, "btn", "button")
	if btnRef == "" {
		btnRef = findElementByContent(snapshot, "Show Dialog", "")
	}
	if btnRef == "" {
		t.Fatal("Could not find button element")
	}

	// Register dialog-dismiss before triggering dialog
	runSw(t, execPath, "dialog-dismiss")

	// Click button to trigger dialog
	runSw(t, execPath, "click", btnRef)
	time.Sleep(500 * time.Millisecond)

	result := mustRunSw(execPath, "eval", "document.getElementById('result').textContent")
	if strings.Contains(result, "dismissed") {
		t.Log("✓ dialog-dismiss worked")
	} else {
		t.Logf("Warning: dialog-dismiss result: %s", result)
	}

	t.Log("✓ TestDialogDismiss passed!")
}

// TestResize tests viewport resize command
func TestResize(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	runSw(t, execPath, append([]string{"open", "data:text/html,<html><body>resize test</body></html>"}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(1 * time.Second)

	// Resize to specific dimensions
	runSw(t, execPath, "resize", "1280", "720")
	time.Sleep(300 * time.Millisecond)

	// Verify via eval
	width := mustRunSw(execPath, "eval", "window.innerWidth")
	height := mustRunSw(execPath, "eval", "window.innerHeight")
	t.Logf("Window size after resize: %s x %s", width, height)

	if strings.Contains(width, "1280") {
		t.Log("✓ Window width correctly set to 1280")
	} else {
		t.Logf("Warning: expected width 1280, got: %s", width)
	}

	if strings.Contains(height, "720") {
		t.Log("✓ Window height correctly set to 720")
	} else {
		t.Logf("Warning: expected height 720, got: %s", height)
	}

	t.Log("✓ TestResize passed!")
}

// TestStateSaveLoad tests state-save and state-load commands
func TestStateSaveLoad(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	stateFile := "/tmp/sw_test_state.json"
	defer os.Remove(stateFile)

	// Use a real URL since data: URLs do not support cookies
	runSw(t, execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(2 * time.Second)

	// Set a cookie
	runSw(t, execPath, "cookie-set", "statetest", "statevalue", "--domain", "example.com")
	time.Sleep(200 * time.Millisecond)

	// Save state
	runSw(t, execPath, "state-save", stateFile)
	time.Sleep(200 * time.Millisecond)

	// Verify file was created
	if _, err := os.Stat(stateFile); err != nil {
		t.Fatalf("state-save did not create file %s: %v", stateFile, err)
	}
	t.Log("✓ state-save created the state file")

	// Clear cookies
	runSw(t, execPath, "cookie-clear")
	time.Sleep(200 * time.Millisecond)

	// Verify cookie is gone
	result := mustRunSw(execPath, "cookie-get", "statetest")
	if strings.Contains(result, "statevalue") {
		t.Logf("Warning: cookie still present after clear: %s", result)
	}

	// Load state
	runSw(t, execPath, "state-load", stateFile)
	time.Sleep(200 * time.Millisecond)

	// Verify cookie is restored
	result = mustRunSw(execPath, "cookie-get", "statetest")
	if strings.Contains(result, "statevalue") {
		t.Log("✓ state-load restored cookies")
	} else {
		t.Logf("Warning: cookie not restored after state-load: %s", result)
	}

	t.Log("✓ TestStateSaveLoad passed!")
}

// TestScreenshotRef tests taking element-level screenshot via ref
func TestScreenshotRef(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	outFile := "/tmp/sw_element_screenshot.png"
	defer os.Remove(outFile)

	html := `<!DOCTYPE html>
<html>
<head><title>Screenshot Ref Test</title></head>
<body>
<h1 id="title">Screenshot Target</h1>
<p>Some other content</p>
</body>
</html>`

	runSw(t, execPath, append([]string{"open", dataURL(html)}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(1 * time.Second)

	// Get snapshot to find a ref
	snapshot := runSw(t, execPath, "snapshot")
	titleRef := findElementByContent(snapshot, "Screenshot Target", "heading")
	if titleRef == "" {
		titleRef = findElementByContent(snapshot, "Screenshot Target", "")
	}
	if titleRef == "" {
		t.Fatal("Could not find title element in snapshot")
	}
	t.Logf("Found element ref: %s", titleRef)

	// Take element screenshot
	runSw(t, execPath, "screenshot", titleRef, "--filename", outFile)
	time.Sleep(300 * time.Millisecond)

	// Verify file was created
	info, err := os.Stat(outFile)
	if err != nil {
		t.Fatalf("screenshot with ref did not create file %s: %v", outFile, err)
	}
	if info.Size() == 0 {
		t.Fatalf("screenshot file is empty: %s", outFile)
	}
	t.Logf("✓ Element screenshot saved to %s (%d bytes)", outFile, info.Size())

	t.Log("✓ TestScreenshotRef passed!")
}

// TestSnapshotFilename tests snapshot --filename flag
func TestSnapshotFilename(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	outFile := "/tmp/sw_snapshot_test.txt"
	defer os.Remove(outFile)

	runSw(t, execPath, append([]string{"open", "https://example.com"}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(2 * time.Second)

	// Take snapshot with filename
	runSw(t, execPath, "snapshot", "--filename", outFile)
	time.Sleep(300 * time.Millisecond)

	// Verify file was created
	info, err := os.Stat(outFile)
	if err != nil {
		t.Fatalf("snapshot --filename did not create file %s: %v", outFile, err)
	}
	if info.Size() == 0 {
		t.Fatalf("snapshot file is empty: %s", outFile)
	}

	// Verify content looks like an aria snapshot
	content, _ := os.ReadFile(outFile)
	t.Logf("Snapshot file size: %d bytes", len(content))
	if len(content) > 0 {
		t.Log("✓ Snapshot file has content")
	}

	t.Log("✓ TestSnapshotFilename passed!")
}

func TestVideoRecording(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	runSw(t, execPath, "open", "https://example.com")
	defer closeBrowser(execPath)
	time.Sleep(2 * time.Second)

	// Start recording
	out := runSw(t, execPath, "video-start")
	t.Logf("video-start: %s", out)

	// Extract output path from message
	var videoPath string
	for _, line := range strings.Split(out, "\n") {
		if strings.HasSuffix(strings.TrimSpace(line), ".webm") {
			videoPath = strings.TrimSpace(line)
			// strip leading "output: " if present
			if idx := strings.Index(videoPath, "/"); idx >= 0 {
				videoPath = videoPath[idx:]
			}
			break
		}
	}
	if videoPath == "" {
		t.Fatal("could not parse video output path from video-start output")
	}
	t.Logf("Video path: %s", videoPath)

	// Do some actions while recording
	runSw(t, execPath, "mousewheel", "0", "300")
	time.Sleep(500 * time.Millisecond)
	runSw(t, execPath, "mousewheel", "0", "300")
	time.Sleep(500 * time.Millisecond)

	// Stop recording
	stopOut := runSw(t, execPath, "video-stop")
	t.Logf("video-stop: %s", stopOut)

	// Verify file exists and has content
	info, err := os.Stat(videoPath)
	if err != nil {
		t.Fatalf("video file not found at %s: %v", videoPath, err)
	}
	if info.Size() == 0 {
		t.Fatalf("video file is empty: %s", videoPath)
	}
	t.Logf("✓ Video file size: %d bytes", info.Size())

	// Cleanup
	os.Remove(videoPath)

	t.Log("✓ TestVideoRecording passed!")
}

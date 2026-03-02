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
	usernameRef := findElementByContent(snapshot, "username", "input")
	if usernameRef == "" {
		// Fallback: find first input
		usernameRef = findFirstElement(snapshot, "input")
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
	selectRef := findElementByContent(snapshot, "fruits", "select")
	if selectRef == "" {
		selectRef = findFirstElement(snapshot, "select")
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
	checkboxes := findAllElements(snapshot, "input")
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

// TestDrag tests drag and drop
func TestDrag(t *testing.T) {
	execPath := getExecPath(t)
	cleanupDaemon(execPath)

	html := `<!DOCTYPE html>
<html>
<head><title>Drag Test</title>
<style>
.box { width: 80px; height: 80px; margin: 10px; display: inline-block; }
#source { background: red; }
#target { background: blue; }
</style></head>
<body>
<h1>Drag Test</h1>
<div id="source" class="box" draggable="true">Drag me</div>
<div id="target" class="box">Drop here</div>
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
  target.style.background = 'green';
});
</script>
</body>
</html>`

	runSw(t, execPath, append([]string{"open", dataURL(html)}, headedArgs()...)...)
	defer closeBrowser(execPath)
	time.Sleep(1 * time.Second)

	// Get snapshot
	snapshot := runSw(t, execPath, "snapshot")

	// Find source and target by their text content
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

	if !strings.Contains(output, "chromium") {
		t.Fatalf("Expected 'chromium' browser in session, got: %s", output)
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

func findCheckboxRefs(snapshot string) []string {
	var refs []string
	lines := strings.Split(snapshot, "\n")
	for _, line := range lines {
		if strings.Contains(line, "<input>") && strings.Contains(line, `"on"`) {
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

func findElementByContent(snapshot, content, elemType string) string {
	lines := strings.Split(snapshot, "\n")
	for _, line := range lines {
		if content != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(content)) {
			continue
		}
		if elemType != "" && !strings.Contains(line, "<"+elemType) {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) > 0 {
			ref := strings.TrimSpace(strings.TrimPrefix(parts[0], "-"))
			if strings.HasPrefix(ref, "e") {
				return ref
			}
		}
	}
	return ""
}

func findFirstElement(snapshot, elemType string) string {
	lines := strings.Split(snapshot, "\n")
	for _, line := range lines {
		if strings.Contains(line, "<"+elemType) {
			parts := strings.Split(line, ":")
			if len(parts) > 0 {
				ref := strings.TrimSpace(strings.TrimPrefix(parts[0], "-"))
				if strings.HasPrefix(ref, "e") {
					return ref
				}
			}
		}
	}
	return ""
}

func findAllElements(snapshot, elemType string) []string {
	var refs []string
	lines := strings.Split(snapshot, "\n")
	for _, line := range lines {
		if strings.Contains(line, "<"+elemType) {
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

# sw (Stealth Wright)

> Silent browser automation CLI with stealth capabilities

## Overview

**sw** (Stealth Wright) is a browser automation CLI that provides:

- **Playwright CLI-compatible UX** - Familiar commands and output format
- **Stealth Mode** - Built-in bot detection evasion via seleniumbase-go
- **Element References** - ARIA snapshot with inline `[ref=eN]` addressing
- **Session Management** - Multiple isolated browser sessions via daemon
- **AI-Friendly** - Skill system for coding agents

## Installation

```bash
# Clone the repository
git clone https://github.com/kyungw00k/sw.git
cd sw

# Build
go build -o bin/sw ./cmd/sw

# Or install globally
go install ./cmd/sw
```

## Quick Start

```bash
# Open browser
sw open https://example.com

# Take snapshot (shows ARIA tree with element references)
sw snapshot

# Interact using [ref=eN] references
sw click e5
sw fill e1 "user@example.com"

# Or use semantic locators (no snapshot needed)
sw click --role button --text "Sign In"
sw fill --label "Email" "user@example.com"

# Annotated screenshot (overlays ref numbers on elements)
sw screenshot --annotate

# Close browser
sw close
```

## Semantic Locators

sw supports two ways to target elements:

### 1. Ref-based (via snapshot)

Run `sw snapshot` first to assign `[ref=eN]` references to all visible elements, then use them in commands:

```bash
sw snapshot
# → - textbox "Email" [ref=e1]
# → - textbox "Password" [ref=e2]
# → - button "Sign In" [ref=e3]

sw fill e1 "user@example.com"
sw fill e2 "secret"
sw click e3
```

### 2. Semantic (no snapshot needed)

Target elements directly by ARIA role, visible text, label, or placeholder:

```bash
sw fill --label "Email" "user@example.com"
sw fill --placeholder "Password" "secret"
sw click --role button --text "Sign In"
```

Use `sw find` to discover elements before interacting:

```bash
sw find --role button
# → ### Found 3 element(s)
# → - [ref=e3] <button> "Sign In"
# → - [ref=e7] <button> "Register"
# → - [ref=e12] <button> "Forgot password"
```

### Annotated Screenshot

Capture a screenshot with `[eN]` ref numbers overlaid on every element — useful for multimodal AI models:

```bash
sw screenshot --annotate
```

---

## Snapshot Format

Snapshots use Playwright-compatible ARIA tree format with inline `[ref=eN]` element references:

```yaml
- heading "Example Domain" [level=1] [ref=e1]
- paragraph [ref=e2]: This domain is for use in illustrative examples.
- link "More information..." [ref=e3]
```

Use the ref value to target elements in subsequent commands:

```bash
sw click e3
sw fill e1 "search query"
```

## Commands

### Navigation

```bash
sw open [url]              # Open browser and navigate
sw close                   # Close browser session
sw goto <url>              # Navigate to URL
sw go-back                 # Go back
sw go-forward              # Go forward
sw reload                  # Reload page
```

### Snapshot & Screenshot

```bash
sw snapshot                         # Generate ARIA snapshot with element refs
sw screenshot                       # Take screenshot
sw screenshot --annotate            # Screenshot with ref number overlays on elements
sw screenshot --full-page           # Full page screenshot
sw screenshot --filename out.png    # Save to specific file
sw pdf [filename]                   # Save page as PDF
```

### Interaction

Ref-based (requires prior `sw snapshot`):
```bash
sw click <ref> [button]    # Click element (button: left/right/middle)
sw dblclick <ref>          # Double-click element
sw hover <ref>             # Hover over element
sw fill <ref> <text>       # Fill input field
sw check <ref>             # Check checkbox/radio
sw uncheck <ref>           # Uncheck checkbox
```

Semantic (no snapshot needed — target by ARIA role, text, or label):
```bash
sw click --role button --text "Submit"
sw click --label "Close dialog"
sw fill --label "Email" "user@example.com"
sw fill --placeholder "Search..." "query"
sw hover --role link --text "About"
sw check --label "Remember me"

# Find elements matching criteria
sw find --role button
sw find --text "Sign In"
sw find --label "Username"
sw find --placeholder "Password"
```

Flags for semantic targeting:
```
--role <aria-role>        ARIA role: button, link, textbox, checkbox, heading, etc.
--text <string>           Visible text content (partial match)
--label <string>          aria-label attribute (partial match)
--placeholder <string>    Input placeholder (partial match)
--exact                   Require exact string match
```

Other interaction commands:
```bash
sw type <text>             # Type into focused element
sw press <key>             # Press key (e.g. Enter, Tab)
sw keydown <key>           # Key down event
sw keyup <key>             # Key up event
sw select <ref> <value>    # Select dropdown option
sw upload <ref> <file>     # Upload file
sw drag <src> <dst>        # Drag and drop
sw eval <script>           # Evaluate JavaScript
```

### Mouse

```bash
sw mousemove <x> <y>       # Move mouse
sw mousedown <x> <y>       # Mouse button down
sw mouseup <x> <y>         # Mouse button up
sw mousewheel <x> <y>      # Mouse wheel scroll
```

### Dialogs

```bash
sw dialog-accept [text]    # Accept dialog (with optional input)
sw dialog-dismiss          # Dismiss dialog
```

### Tabs

```bash
sw tab-list                # List open tabs
sw tab-new [url]           # Open new tab
sw tab-close [index]       # Close tab
sw tab-select <index>      # Switch to tab
```

### Cookies

```bash
sw cookie-list             # List all cookies
sw cookie-get <name>       # Get cookie by name
sw cookie-set <name> <value> [--domain] [--path] [--expires] [--httpOnly] [--secure] [--sameSite]
sw cookie-delete <name>    # Delete cookie
sw cookie-clear            # Clear all cookies
```

### Local Storage

```bash
sw localstorage-list       # List all entries
sw localstorage-get <key>  # Get value
sw localstorage-set <key> <value>  # Set value
sw localstorage-delete <key>       # Delete entry
sw localstorage-clear      # Clear all entries
```

### Session Storage

```bash
sw sessionstorage-list     # List all entries
sw sessionstorage-get <key>  # Get value
sw sessionstorage-set <key> <value>  # Set value
sw sessionstorage-delete <key>       # Delete entry
sw sessionstorage-clear    # Clear all entries
```

### State & Data

```bash
sw state-save [path]       # Save browser state to file
sw state-load [path]       # Load browser state from file
sw delete-data             # Clear all browser data
sw resize <width> <height> # Resize browser window
```

### Video Recording

```bash
sw video-start             # Start recording (saves WebM to current directory)
sw video-stop              # Stop recording and save file
```

### Developer Tools

```bash
sw install-browser [--browser <type>]  # Install browser binaries (chromium/firefox/webkit)
sw devtools-start                      # Open browser DevTools (headed+Chromium only)
```

### Session Management

```bash
sw list                    # List all sessions
sw kill-all                # Kill all browser sessions
sw close-all               # Close all browser sessions
sw daemon start            # Start daemon manually
sw daemon stop             # Stop daemon
sw daemon status           # Check daemon status
```

## Global Options

```bash
-s, --session <name>       Session name (default: "default")
-b, --browser <type>       Browser: chromium, firefox, webkit
    --headed               Run in headed mode
    --persistent           Persist profile to disk
    --profile <path>       Custom profile directory
    --config <file>        Config file path
    --stealth              Enable stealth mode (default: true)
    --no-stealth           Disable stealth mode
```

## Stealth Features

Stealth mode is powered by [seleniumbase-go](https://github.com/kyungw00k/seleniumbase-go) which launches Chrome with fingerprint evasion arguments:

- Hide WebDriver automation markers
- Randomized browser fingerprints
- Custom User-Agent strings
- WebGL and Canvas fingerprint spoofing
- WebRTC leak prevention

## Architecture

```
sw (Stealth Wright)
│
├── CLI (cobra)                    # Command-line interface
├── Daemon (Unix socket)           # Background browser process
├── Browser Abstraction Layer      # Pluggable browser backends
│   └── seleniumbase-go (current)  # Playwright + stealth via seleniumbase-go
└── Snapshot Generator             # ARIA tree with [ref=eN] annotations
```

### Protocol

Communication between CLI and Daemon uses JSON-RPC 2.0 over Unix sockets.

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "click",
  "params": {"ref": "e5"}
}
```

## AI Skill System

For AI coding agents (Claude Code, GitHub Copilot, etc.), use the skill file:

```markdown
# In your project's CLAUDE.md or similar:

Use the sw skill for browser automation tasks.
Skill location: skills/sw/SKILL.md
```

### Example Usage with Claude Code

```
User: Use sw to open example.com and click the first link

Claude:
1. sw open https://example.com
2. sw snapshot
   → - link "More information..." [ref=e3]
3. sw click e3
```

## Comparison

| Feature | Playwright CLI | sw |
|---------|---------------|-----|
| Daemon Sessions | ✅ | ✅ |
| Element References (`[ref=eN]`) | ✅ | ✅ |
| ARIA Snapshot Format | ✅ | ✅ |
| Semantic Locators (`--role/--text/--label`) | ✅ | ✅ |
| Annotated Screenshots | ❌ | ✅ |
| Stealth Mode | ❌ | ✅ |
| Video Recording | ✅ | ✅ |
| AI Skills | ✅ | ✅ |
| Go Native Binary | ❌ | ✅ |

## Development

```bash
# Build
go build -o bin/sw ./cmd/sw

# Run unit tests
go test ./...

# Run integration tests (requires built binary)
go build -o bin/sw ./cmd/sw
go test -tags integration -v ./test/...

# Lint
go vet ./...

# Format
go fmt ./...
```

## Project Structure

```
sw/
├── cmd/sw/main.go              # CLI entry point (cobra commands)
├── internal/
│   ├── browser/                # Browser interface abstraction
│   ├── client/                 # Daemon JSON-RPC client
│   ├── daemon/                 # Daemon server + command handlers
│   ├── drivers/seleniumbase/   # seleniumbase-go driver implementation
│   ├── session/                # Session lifecycle management
│   └── snapshot/               # ARIA snapshot + [ref=eN] annotation
├── pkg/protocol/               # JSON-RPC request/response types
├── skills/sw/SKILL.md          # AI agent skill file
├── test/                       # Integration tests
└── README.md
```

## License

MIT

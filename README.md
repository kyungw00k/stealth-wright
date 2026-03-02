# sw (Stealth Wright)

> Silent browser automation CLI with stealth capabilities

## Overview

**sw** (Stealth Wright) is a browser automation CLI that provides:

- **Playwright CLI-like UX** - Familiar commands and workflows
- **Stealth Mode** - Built-in bot detection evasion
- **Element References** - Snapshot-based element addressing (e1, e2...)
- **Session Management** - Multiple isolated browser sessions
- **AI-Friendly** - Skill system for coding agents

## Prerequisites

- Go 1.22+
- [seleniumbase-go](https://github.com/kyungw00k/seleniumbase-go) (sibling directory)

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

# Take snapshot (shows element references)
sw snapshot

# Interact with elements
sw click e5
sw fill e1 "user@example.com"

# Screenshot
sw screenshot

# Close browser
sw close
```

## Commands

### Core

```bash
sw open [url]              # Open browser
sw close                   # Close browser
sw goto <url>              # Navigate to URL
sw snapshot                # Generate element references
sw click <ref>             # Click element
sw fill <ref> <text>       # Fill text
sw type <text>             # Type into focused element
sw press <key>             # Press key
sw hover <ref>             # Hover over element
sw screenshot [filename]   # Take screenshot
```

### Navigation

```bash
sw go-back                 # Go back
sw go-forward              # Go forward
sw reload                  # Reload page
```

### Session Management

```bash
sw list                    # List sessions
sw -s=auth open <url>      # Named session
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

- **Fingerprint Randomization** - Browser fingerprint spoofing
- **User-Agent Spoofing** - Custom UA strings
- **WebGL Renderer Spoofing** - GPU info spoofing
- **Canvas Fingerprint** - Noise injection
- **WebRTC Leak Prevention** - IP leak protection
- **Hide WebDriver** - Remove automation markers
- **Human-like Typing** - Realistic typing behavior
- **Random Delays** - Natural timing

## Architecture

```
sw (Stealth Wright)
│
├── CLI (cobra)                    # Command-line interface
├── Daemon (Unix socket)           # Background browser process
├── Browser Abstraction Layer      # Pluggable browser backends
│   ├── seleniumbase-go (current)
│   ├── playwright-go (planned)
│   └── rod (planned)
└── Stealth Module                 # Bot detection evasion
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
3. Parse snapshot to find first link (e.g., e3)
4. sw click e3
```

## Comparison

| Feature | Playwright CLI | sw |
|---------|---------------|-----|
| Daemon Sessions | ✅ | ✅ |
| Element References | ✅ | ✅ |
| Stealth Mode | ❌ | ✅ |
| AI Skills | ✅ | ✅ |
| Go Native | ❌ | ✅ |
| Plugin Backends | ❌ | ✅ |

## Development

```bash
# Run tests
go test ./...

# Build
go build -o bin/sw ./cmd/sw

# Run linter
go vet ./...

# Format
go fmt ./...
```

## Project Structure

```
sw/
├── cmd/sw/main.go              # CLI entry point
├── internal/
│   ├── browser/                # Browser interface (abstraction)
│   ├── client/                 # Daemon client
│   ├── daemon/                 # Daemon server
│   ├── drivers/seleniumbase/   # seleniumbase-go implementation
│   ├── session/                # Session management
│   ├── snapshot/               # Element reference generator
│   └── stealth/                # Stealth module
├── pkg/protocol/               # JSON-RPC types
├── skills/sw/SKILL.md          # AI skill file
├── ARCHITECTURE.md             # Architecture documentation
└── README.md
```

## License

MIT

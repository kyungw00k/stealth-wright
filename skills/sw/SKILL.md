---
name: sw
description: Browser automation CLI with stealth capabilities. Use when the user needs to navigate websites, interact with web pages, fill forms, take screenshots, or extract information from web pages without being detected as a bot.
allowed-tools: Bash(sw:*)
---

# Browser Automation with sw (Stealth Wright)

## Quick start

```bash
# open new browser
sw open
# navigate to a page
sw goto https://example.com
# interact with the page using refs from the snapshot
sw click e15
sw type "search query"
sw press Enter
# take a screenshot
sw screenshot
# close the browser
sw close
```

## Commands

### Core

```bash
sw open [url]              # open browser, optionally navigate
sw goto <url>              # navigate to URL
sw close                   # close browser
sw snapshot                # capture page snapshot with element refs
sw click <ref> [button]    # click element (button: left/right/middle)
sw fill <ref> <text>       # fill text into element
sw type <text>             # type into focused element
sw press <key>             # press a key (Enter, Tab, ArrowDown, etc.)
sw hover <ref>             # hover over element
sw screenshot [ref]        # screenshot page or element
```

### Navigation

```bash
sw go-back                 # go back in history
sw go-forward              # go forward in history
sw reload                  # reload current page
```

### Stealth Mode

```bash
sw stealth on              # enable stealth mode
sw stealth off             # disable stealth mode
sw stealth status          # show stealth configuration
sw stealth fingerprint     # display current fingerprint
```

### Sessions

```bash
sw list                    # list active sessions
sw -s=name open [url]      # open named session
sw -s=name close           # close named session
sw close-all               # close all sessions
sw kill-all                # kill zombie processes
```

### Storage

```bash
# Cookies
sw cookie-list
sw cookie-set <name> <value>
sw cookie-delete <name>
sw cookie-clear

# State
sw state-save [file]
sw state-load <file>
```

## Element References

After running `sw snapshot`, elements are assigned references:

```
- e1: <input> placeholder="Email"
- e2: <input> type="password"
- e3: <button> "Sign In"
- e4: <a> href="/forgot"
```

Use these refs in commands:

```bash
sw fill e1 "user@example.com"
sw fill e2 "password123"
sw click e3
```

## Sessions

Named sessions allow parallel browser contexts:

```bash
# Multiple sessions
sw -s=auth open https://login.example.com
sw -s=scrape open https://data.example.com

# Commands are isolated by session
sw -s=auth fill e1 "user"
sw -s=scrape snapshot
```

## Stealth Mode

Stealth mode is enabled by default. It:

- Randomizes browser fingerprint
- Hides automation markers
- Adds human-like behavior
- Prevents WebRTC leaks

```bash
# Check status
sw stealth status

# Disable if needed
sw stealth off

# Re-enable
sw stealth on
```

## Global Options

```bash
-s, --session <name>    Session name (default: "default")
-b, --browser <type>    Browser: chromium, firefox, webkit
    --headed            Run in headed mode (visible)
    --persistent        Save profile to disk
    --profile <path>    Custom profile directory
    --config <file>     Configuration file
    --no-stealth        Disable stealth mode
```

## Example Workflows

### Form Submission

```bash
sw open https://example.com/login
sw snapshot
sw fill e1 "user@example.com"
sw fill e2 "password123"
sw click e3
sw snapshot
sw close
```

### Multi-Step with Stealth

```bash
sw open https://example.com --stealth
sw snapshot
sw click e5
sw fill e2 "search term"
sw press Enter
sw screenshot result.png
sw close
```

### Parallel Sessions

```bash
sw -s=site1 open https://site1.com &
sw -s=site2 open https://site2.com &
wait
sw -s=site1 screenshot site1.png
sw -s=site2 screenshot site2.png
sw close-all
```

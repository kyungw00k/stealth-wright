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

### Core Navigation

```bash
sw open [url]              # open browser, optionally navigate
sw goto <url>              # navigate to URL
sw close                   # close browser
sw close-all               # close all browser sessions
sw reload                  # reload current page
sw go-back                 # go back in history
sw go-forward              # go forward in history
```

### Page Snapshots & Screenshots

```bash
sw snapshot                # capture page snapshot with element refs
sw screenshot [ref]        # screenshot page or specific element
sw screenshot --filename <file> [ref]  # save screenshot to file
sw screenshot --full-page  # full page screenshot
```

### Interaction Commands

```bash
sw click <ref> [button]    # click element (button: left/right/middle, default: left)
sw dblclick <ref>          # double-click element
sw fill <ref> <text>       # fill text into element
sw type <text>             # type into focused element
sw press <key>             # press key (Enter, Tab, ArrowDown, etc.)
sw hover <ref>             # hover over element
sw check <ref>             # check checkbox
sw uncheck <ref>           # uncheck checkbox
sw select <ref> <value>    # select option from dropdown
```

### Mouse & Keyboard

```bash
sw mouse-move <x> <y>      # move mouse to coordinates
sw mouse-down              # mouse button down
sw mouse-up                # mouse button up
sw mousewheel <x> <y>      # scroll with mouse wheel
sw key-down <key>          # key down
sw key-up <key>            # key up
sw drag <ref> <x> <y>      # drag element to coordinates
```

### Forms & Upload

```bash
sw upload <ref> <file>     # upload file to element
```

### Dialogs

```bash
sw dialog-accept           # accept dialog (OK, yes, etc.)
sw dialog-dismiss          # dismiss dialog (Cancel, no, etc.)
```

### Tabs

```bash
sw tab-new [url]           # open new tab
sw tab-close               # close current tab
sw tab-switch <index>      # switch to tab by index
sw tab-list                # list all tabs
```

### Page Data

```bash
sw eval <code> [--ref]     # evaluate JavaScript and return result
sw pdf [--filename <file>] # generate PDF of page
```

### Browser Size

```bash
sw resize <width> <height> # resize browser window
```

### Cookies

```bash
sw cookie-list [--domain <domain>] [--path <path>]
sw cookie-get <name>
sw cookie-set <name> <value> [--domain <domain>] [--path <path>] [--expires <timestamp>] [--httpOnly] [--secure] [--sameSite strict|lax|none]
sw cookie-delete <name>
sw cookie-clear
```

### Local & Session Storage

```bash
# LocalStorage
sw localstorage-list
sw localstorage-get <key>
sw localstorage-set <key> <value>
sw localstorage-remove <key>
sw localstorage-clear

# SessionStorage
sw sessionstorage-list
sw sessionstorage-get <key>
sw sessionstorage-set <key> <value>
sw sessionstorage-remove <key>
sw sessionstorage-clear
```

### State & Data

```bash
sw state-save [file]       # save browser state to file
sw state-load <file>       # load browser state from file
sw delete-data             # clear all browser data
```

### Sessions

```bash
sw list                    # list active sessions
sw kill-all                # kill zombie processes
```

### Network & Monitoring

```bash
sw console                 # show console messages
sw network                 # show network activity
sw route <glob> <url>      # intercept and route requests
sw route-list              # list active routes
sw unroute <glob>          # remove request interception
sw tracing-start           # start performance tracing
sw tracing-stop            # stop tracing and save file
```

### Video

```bash
sw video-start             # start recording video
sw video-stop              # stop video recording
```

## Global Options

```bash
-s, --session <name>       # Session name (default: "default")
-b, --browser <type>       # Browser: chromium, firefox, webkit (default: chromium)
    --headed               # Run in headed mode (visible)
    --persistent           # Save profile to disk
    --profile <path>       # Custom profile directory
    --config <file>        # Configuration file
    --stealth              # Enable stealth mode (default: true)
    --no-stealth           # Disable stealth mode
```

## Element References

After running `sw snapshot`, elements are assigned references in the format `eN` where N is a number:

```
- e1: <input> placeholder="Email"
- e2: <input> type="password"
- e3: <button> "Sign In"
- e4: <a> href="/forgot"
```

Use these refs in interaction commands:

```bash
sw fill e1 "user@example.com"
sw fill e2 "password123"
sw click e3
```

## Stealth Mode

Stealth mode is **enabled by default**. It:
- Randomizes browser fingerprint
- Hides automation markers
- Adds human-like behavior
- Prevents WebRTC leaks

Disable stealth if needed:

```bash
sw --no-stealth open https://example.com
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

### Search with Stealth

```bash
sw open https://example.com
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

### State Persistence

```bash
# Save state
sw open https://example.com
sw fill e1 "user"
sw press Enter
sw state-save session.json

# Later: restore state
sw state-load session.json
sw snapshot
```

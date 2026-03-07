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
sw devices                 # list all available devices for emulation
```

### Page Snapshots & Screenshots

```bash
sw snapshot                          # capture page snapshot with element refs
sw screenshot [ref]                  # screenshot page or specific element
sw screenshot --filename <file>      # save screenshot to file
sw screenshot --full-page            # full page screenshot
sw screenshot --annotate             # screenshot with ref number overlays on each element
```

### Interaction Commands

Ref-based (requires prior `sw snapshot`):
```bash
sw click <ref> [button]    # click element (button: left/right/middle, default: left)
sw dblclick <ref>          # double-click element
sw fill <ref> <text>       # fill text into element
sw hover <ref>             # hover over element
sw check <ref>             # check checkbox
sw uncheck <ref>           # uncheck checkbox
```

Semantic (no snapshot needed):
```bash
sw click --role button --text "Submit"       # click by ARIA role + text
sw click --label "Close"                     # click by aria-label
sw fill --label "Email" "user@example.com"   # fill by label
sw fill --placeholder "Search" "query"       # fill by placeholder
sw hover --role link --text "About"          # hover by role + text
sw check --label "Remember me"               # check by label
sw uncheck --label "Subscribe"               # uncheck by label

# Find elements matching semantic criteria
sw find --role button                        # list all buttons
sw find --text "Sign In"                     # find by text
sw find --label "Username"                   # find by aria-label
sw find --placeholder "Password"             # find by placeholder
```

Semantic flags (work on click/fill/hover/check/uncheck/find):
```
--role <aria-role>    ARIA role: button, link, textbox, checkbox, heading, combobox, etc.
--text <string>       Visible text content (partial match by default)
--label <string>      aria-label attribute (partial match by default)
--placeholder <string> Input placeholder (partial match by default)
--exact               Require exact string matches
```

Other interaction commands:
```bash
sw type <text>             # type into focused element
sw press <key>             # press key (Enter, Tab, ArrowDown, etc.)
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

### Video Recording

```bash
sw video-start             # start recording (saves WebM to current directory)
sw video-stop              # stop recording and save file
```

### Developer Tools

```bash
sw install-browser [--browser <type>]  # install browser binaries (chromium/firefox/webkit)
sw devtools-start                      # open browser DevTools (headed+Chromium only)
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
-s, --session <name>       # Session name (env: SW_SESSION, default: "default")
-b, --browser <type>       # Browser: chrome, chromium, firefox, webkit (env: SW_BROWSER, default: "chrome")
    --headed               # Run in headed mode (env: SW_HEADED)
    --persistent           # Save profile to disk
    --profile <path>       # Custom profile directory (env: SW_PROFILE)
    --config <file>        # Configuration file
    --stealth              # Enable stealth mode (env: SW_STEALTH, default: true)
    --no-stealth           # Disable stealth mode
```

### open-only Options

```bash
    --device <name>        # Emulate a device, e.g. "iPhone 15", "Pixel 7" (env: SW_DEVICE)
```

## Environment Variables

All key options can be set via environment variables. CLI flags always take precedence.

| Variable | Flag | Description |
|---|---|---|
| `SW_SESSION` | `-s` / `--session` | Session name |
| `SW_BROWSER` | `-b` / `--browser` | Browser type |
| `SW_HEADED` | `--headed` | Run in headed mode (`true`/`false`) |
| `SW_STEALTH` | `--stealth` | Stealth mode (`true`/`false`, default `true`) |
| `SW_PROFILE` | `--profile` | Profile directory path |
| `SW_DEVICE` | `--device` | Device to emulate |

Legacy alias: `PLAYWRIGHT_CLI_SESSION` is also accepted for `SW_SESSION`.

## Targeting Elements

### Method 1: Ref-based (via snapshot)

Run `sw snapshot` to assign `[ref=eN]` to all visible elements. The snapshot shows the ARIA tree:

```
- textbox "Email" [ref=e1]
- textbox "Password" [ref=e2]
- button "Sign In" [ref=e3]
- link "Forgot password" [ref=e4]
```

Use the ref in commands:
```bash
sw fill e1 "user@example.com"
sw fill e2 "password123"
sw click e3
```

### Method 2: Semantic locators (no snapshot needed)

Target elements by their ARIA role, visible text, label, or placeholder — without running `sw snapshot` first:

```bash
sw fill --label "Email" "user@example.com"
sw fill --label "Password" "secret"
sw click --role button --text "Sign In"
```

Use `sw find` to discover what elements are available:
```bash
sw find --role button
# → ### Found 2 element(s)
# → - [ref=e3] <button> "Sign In"
# → - [ref=e8] <button> "Register"
```

### Method 3: Annotated screenshot

For multimodal AI workflows, capture a screenshot with ref numbers overlaid on every element:

```bash
sw screenshot --annotate
# → saves page-2026-...png with red [eN] labels on each element
```

This lets a vision model identify elements by ref without reading the ARIA snapshot text.

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

## Mobile Device Emulation

Emulate a mobile device to test mobile-specific layouts and behaviors.

```bash
# List all available devices
sw devices

# Open browser with a specific device
sw open --device "iPhone 15" https://example.com
sw open --device "Pixel 7"
sw open --device "iPad Pro 11"

# Combine with headed mode to see the emulation
sw open --headed --device "Galaxy S8" https://example.com
```

The `--device` flag sets the user agent, viewport size, device scale factor, touch events, and mobile flag to match the real device. Use `sw devices` to see the full list of supported device names.

## Example Workflows

### Form Submission (ref-based)

```bash
sw open https://example.com/login
sw snapshot
# → - textbox "Email" [ref=e1]
# → - textbox "Password" [ref=e2]
# → - button "Sign In" [ref=e3]
sw fill e1 "user@example.com"
sw fill e2 "password123"
sw click e3
sw snapshot
sw close
```

### Form Submission (semantic — no snapshot needed)

```bash
sw open https://example.com/login
sw fill --label "Email" "user@example.com"
sw fill --label "Password" "password123"
sw click --role button --text "Sign In"
sw snapshot
sw close
```

### Search with Stealth

```bash
sw open https://example.com
sw fill --placeholder "Search" "query"
sw press Enter
sw screenshot result.png
sw close
```

### Annotated Screenshot for AI Vision

```bash
sw open https://example.com
sw screenshot --annotate
# → saves screenshot with [e1], [e2], ... overlaid on each element
# AI model can now reference elements by their visible label
```

### Video Recording

```bash
sw open https://example.com
sw video-start
sw snapshot
sw click e3
sw fill e1 "hello"
sw video-stop
# → saves video-2026-...webm in current directory
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
sw fill --label "Username" "user"
sw click --role button --text "Login"
sw state-save session.json

# Later: restore state
sw state-load session.json
sw snapshot
```

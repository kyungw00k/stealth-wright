package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/kyungw00k/sw/internal/client"
	"github.com/kyungw00k/sw/internal/daemon"
	"github.com/kyungw00k/sw/pkg/protocol"
	"github.com/kyungw00k/sw/skills"
	playwright "github.com/playwright-community/playwright-go"
	"github.com/spf13/cobra")

var (
	// Global options
	sessionName   string
	browserType   string
	headed        bool
	persistent    bool
	profile       string
	configFile    string
	stealthMode   bool
	noStealthMode bool

	// Client
	cli *client.Client
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "sw",
		Short: "Stealth Wright - Silent browser automation CLI",
		Long: `sw (Stealth Wright) is a browser automation CLI with stealth capabilities.

It provides Playwright CLI-like UX with built-in stealth mode for 
undetected browser automation.`,
		Version: "0.1.0",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Initialize client — each session routes to its own socket
			cli = client.NewClient(&client.Config{
				SocketPath: client.DefaultSocketPath(sessionName),
			})

			if noStealthMode {
				stealthMode = false
			}
		},
	}

	// Global flags — defaults fall back to SW_* env vars
	rootCmd.PersistentFlags().StringVarP(&sessionName, "session", "s", envStr("SW_SESSION", envStr("PLAYWRIGHT_CLI_SESSION", "default")), "Session name (env: SW_SESSION)")
	rootCmd.PersistentFlags().StringVarP(&browserType, "browser", "b", envStr("SW_BROWSER", "chrome"), "Browser type: chrome, chromium, firefox, webkit (env: SW_BROWSER)")
	rootCmd.PersistentFlags().BoolVar(&headed, "headed", envBool("SW_HEADED", false), "Run in headed mode (env: SW_HEADED)")
	rootCmd.PersistentFlags().BoolVar(&persistent, "persistent", false, "Persist browser profile to disk")
	rootCmd.PersistentFlags().StringVar(&profile, "profile", envStr("SW_PROFILE", ""), "Custom profile directory (env: SW_PROFILE)")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Config file path")
	rootCmd.PersistentFlags().BoolVar(&stealthMode, "stealth", envBool("SW_STEALTH", true), "Enable stealth mode (env: SW_STEALTH)")
	rootCmd.PersistentFlags().BoolVar(&noStealthMode, "no-stealth", false, "Disable stealth mode")

	// Add commands
	rootCmd.AddCommand(
		newOpenCmd(),
		newCloseCmd(),
		newGotoCmd(),
		newGoBackCmd(),
		newGoForwardCmd(),
		newReloadCmd(),
		newSnapshotCmd(),
		newClickCmd(),
		newCheckCmd(),
		newFillCmd(),
		newTypeCmd(),
		newPressCmd(),
		newHoverCmd(),
		newScreenshotCmd(),
		newListCmd(),
		newDblClickCmd(),
		newUncheckCmd(),
		newDragCmd(),
		newSelectCmd(),
		newEvalCmd(),
		newResizeCmd(),
		newUploadCmd(),
		newKeyDownCmd(),
		newKeyUpCmd(),
		newMouseMoveCmd(),
		newMouseDownCmd(),
		newMouseUpCmd(),
		newDialogAcceptCmd(),
		newDialogDismissCmd(),
		newTabNewCmd(),
		newTabCloseCmd(),
		newTabSelectCmd(),
		newTabListCmd(),
		newStateSaveCmd(),
		newStateLoadCmd(),
		newKillAllCmd(),
		newCookieListCmd(),
		newCookieGetCmd(),
		newCookieSetCmd(),
		newCookieDeleteCmd(),
		newCookieClearCmd(),
		newLocalStorageListCmd(),
		newLocalStorageGetCmd(),
		newLocalStorageSetCmd(),
		newLocalStorageDeleteCmd(),
		newLocalStorageClearCmd(),
		newSessionStorageListCmd(),
		newSessionStorageGetCmd(),
		newSessionStorageSetCmd(),
		newSessionStorageDeleteCmd(),
		newSessionStorageClearCmd(),
		newMouseWheelCmd(),
		newPDFCmd(),
		newDeleteDataCmd(),
		newCloseAllCmd(),
		newShowCmd(),
		newRunCodeCmd(),
		newConsoleCmd(),
		newNetworkCmd(),
		newTracingStartCmd(),
		newTracingStopCmd(),
		newRouteCmd(),
		newRouteListCmd(),
		newUnrouteCmd(),
		newInstallCmd(),
		newDevicesCmd(),
		newVideoStartCmd(),
		newVideoStopCmd(),
		newDaemonCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// envStr returns the value of an environment variable, or defaultVal if unset/empty.
func envStr(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// envBool parses a boolean environment variable.
// Truthy: "1", "true", "yes". Falsy: "0", "false", "no".
// Returns defaultVal if the variable is unset or empty.
func envBool(key string, defaultVal bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes":
		return true
	case "0", "false", "no":
		return false
	}
	return defaultVal
}

// ensureDaemon ensures the daemon is running
func ensureDaemon() error {
	if cli.CanConnect() {
		return nil
	}

	// Start daemon
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	return cli.StartDaemon(execPath, sessionName)
}

// newDaemonCmd returns the daemon command group.
func newDaemonCmd() *cobra.Command {
	daemonCmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage the background daemon",
	}
	daemonCmd.AddCommand(newDaemonStartCmd(), newDaemonStopCmd(), newDaemonStatusCmd())
	return daemonCmd
}

func newDaemonStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the daemon",
		Run: func(cmd *cobra.Command, args []string) {
			socketPath := client.DefaultSocketPath(sessionName)
			if daemon.IsRunning(socketPath) {
				w := bufio.NewWriter(os.Stdout)
				daemon.WriteSuccess(w, fmt.Sprintf("Daemon already listening on %s", socketPath))
				return
			}
			srv, err := daemon.NewServer(&daemon.Config{
				SocketPath: socketPath,
				BaseDir:    client.DefaultBaseDir(),
			})
			if err != nil {
				fmt.Fprintln(os.Stderr, "failed to create daemon:", err)
				os.Exit(1)
			}
			if err := srv.Start(); err != nil {
				fmt.Fprintln(os.Stderr, "failed to start daemon:", err)
				os.Exit(1)
			}
			w := bufio.NewWriter(os.Stdout)
			daemon.WriteSuccess(w, fmt.Sprintf("Daemon listening on %s", socketPath))
			srv.WaitForShutdown()
		},
	}
}

func newDaemonStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the daemon",
		Run: func(cmd *cobra.Command, args []string) {
			socketPath := client.DefaultSocketPath(sessionName)
			if !daemon.IsRunning(socketPath) {
				fmt.Println("Daemon is not running.")
				return
			}
			c := client.NewClient(&client.Config{SocketPath: socketPath})
			if err := c.Connect(); err != nil {
				fmt.Fprintln(os.Stderr, "failed to connect to daemon:", err)
				os.Exit(1)
			}
			defer c.Disconnect()
			if _, err := c.Call("stop", nil); err != nil {
				fmt.Fprintln(os.Stderr, "stop error:", err)
				os.Exit(1)
			}
			fmt.Println("Daemon stopped.")
		},
	}
}

func newDaemonStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show daemon status",
		Run: func(cmd *cobra.Command, args []string) {
			if daemon.IsRunning(client.DefaultSocketPath(sessionName)) {
				fmt.Println("Daemon is running.")
			} else {
				fmt.Println("Daemon is not running.")
			}
		},
	}
}

// printResult prints a command result
func printResult(result *protocol.CommandResult) {
	if result.Page != nil {
		fmt.Println("### Page")
		fmt.Printf("- Page URL: %s\n", result.Page.URL)
		fmt.Printf("- Page Title: %s\n", result.Page.Title)
	}
	if result.Message != "" {
		fmt.Println(result.Message)
	}
}

// printSnapshot prints a snapshot result in playwright-cli format (file link only).
func printSnapshot(result *protocol.SnapshotResult) {
	cwd, _ := os.Getwd()

	fmt.Println("### Page")
	fmt.Printf("- Page URL: %s\n", result.PageURL)
	fmt.Printf("- Page Title: %s\n", result.PageTitle)
	fmt.Printf("- Console: %d errors, %d warnings\n", result.ConsoleErrors, result.ConsoleWarnings)

	if result.Filename != "" {
		relPath := toRelPath(cwd, result.Filename)
		fmt.Println("\n### Snapshot")
		fmt.Printf("- [Snapshot](%s)\n", relPath)
	}

	if result.ConsoleLogFile != "" {
		relLogPath := toRelPath(cwd, result.ConsoleLogFile)
		fmt.Println("\n### Events")
		fmt.Printf("- New console entries: %s#L1\n", relLogPath)
	}
}

// toRelPath converts an absolute path to a relative path from base.
// Falls back to the original path if conversion fails.
func toRelPath(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return rel
}

// printSnapshotVerbose prints a snapshot with inline ARIA tree content.
func printSnapshotVerbose(result *protocol.SnapshotResult) {
	printSnapshot(result)
	if result.AriaSnapshot != "" {
		fmt.Println()
		fmt.Println(result.AriaSnapshot)
	}
}

// printRanCode prints the "Ran Playwright code" section
func printRanCode(jsCode string) {
	fmt.Println("### Ran Playwright code")
	fmt.Printf("```js\n%s\n```\n", jsCode)
}

// printBrowserOpened prints the browser opened message (playwright-cli format)
func printBrowserOpened(sessionName, browserType, userDataDir string, headed bool) {
	fmt.Printf("### Browser `%s` opened.\n", sessionName)
	fmt.Printf("- %s:\n", sessionName)
	fmt.Printf("  - browser-type: %s\n", browserType)
	fmt.Printf("  - user-data-dir: %s\n", userDataDir)
	fmt.Printf("  - headed: %v\n", headed)
	fmt.Println("---")
}

func newOpenCmd() *cobra.Command {
	device := envStr("SW_DEVICE", "")
	cmd := &cobra.Command{
		Use:   "open [url]",
		Short: "Open browser and optionally navigate to URL",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to start daemon:", err)
				os.Exit(1)
			}

			url := ""
			if len(args) > 0 {
				url = args[0]
			}

			openOpts := []client.OpenOption{
				client.WithHeaded(headed),
				client.WithBrowser(browserType),
				client.WithStealth(stealthMode),
			}
			if device != "" {
				openOpts = append(openOpts, client.WithDevice(device))
			}

			result, err := cli.Open(url, openOpts...)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			userDataDir := "<in-memory>"
			if profile != "" {
				userDataDir = profile
			}
			printBrowserOpened(sessionName, browserType, userDataDir, headed)

			if url != "" {
				printRanCode(fmt.Sprintf("await page.goto('%s');", url))
			}
			if result != nil && result.Snapshot != nil {
				printSnapshot(result.Snapshot)
			}
		},
	}
	cmd.Flags().StringVar(&device, "device", device, `device to emulate, e.g. "iPhone 15", "Pixel 7" (env: SW_DEVICE)`)
	return cmd
}

func newCloseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "close",
		Short: "Close browser",
		Run: func(cmd *cobra.Command, args []string) {
			if !cli.CanConnect() {
				fmt.Println("Browser is not open.")
				return
			}

			if err := cli.Close(); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			fmt.Println("Browser closed.")
		},
	}
}

func newGotoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "goto <url>",
		Short: "Navigate to URL",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			result, err := cli.Goto(args[0])
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			printRanCode(fmt.Sprintf("await page.goto('%s');", args[0]))
			if result != nil && result.Snapshot != nil {
				printSnapshot(result.Snapshot)
			}
		},
	}
}

func newGoBackCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "go-back",
		Short: "Go back in history",
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			resp, err := cli.Call("go-back", nil)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			var result protocol.CommandResult
			json.Unmarshal(resp.Result, &result)
			printRanCode("await page.goBack();")
			if result.Snapshot != nil {
				printSnapshot(result.Snapshot)
			}
		},
	}
}

func newGoForwardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "go-forward",
		Short: "Go forward in history",
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			resp, err := cli.Call("go-forward", nil)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			var result protocol.CommandResult
			json.Unmarshal(resp.Result, &result)
			printRanCode("await page.goForward();")
			if result.Snapshot != nil {
				printSnapshot(result.Snapshot)
			}
		},
	}
}

func newReloadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reload",
		Short: "Reload current page",
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			resp, err := cli.Call("reload", nil)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			var result protocol.CommandResult
			json.Unmarshal(resp.Result, &result)
			printRanCode("await page.reload();")
			if result.Snapshot != nil {
				printSnapshot(result.Snapshot)
			}
		},
	}
}

func newSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Generate page snapshot with element references",
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			filename, _ := cmd.Flags().GetString("filename")
			params := map[string]interface{}{}
			if filename != "" {
				params["filename"] = filename
			}

			resp, err := cli.Call("snapshot", params)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			var result protocol.SnapshotResult
			json.Unmarshal(resp.Result, &result)
			printSnapshotVerbose(&result)
		},
	}
	cmd.Flags().String("filename", "", "Save snapshot to file")
	return cmd
}

func newClickCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "click <ref>",
		Short: "Click element",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			modifiers, _ := cmd.Flags().GetStringArray("modifiers")
			params := map[string]interface{}{"ref": args[0]}
			if len(modifiers) > 0 {
				params["modifiers"] = modifiers
			}

			resp, err := cli.Call("click", params)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			var result protocol.CommandResult
			json.Unmarshal(resp.Result, &result)
			if len(modifiers) > 0 {
				printRanCode(fmt.Sprintf("await page.locator('[ref=%s]').click({ button: '%s' });", args[0], strings.Join(modifiers, ",")))
			} else {
				printRanCode(fmt.Sprintf("await page.locator('[ref=%s]').click();", args[0]))
			}
			if result.Snapshot != nil {
				printSnapshot(result.Snapshot)
			}
		},
	}
	cmd.Flags().StringArray("modifiers", nil, "Keyboard modifiers (Alt, Control, Meta, Shift)")
	return cmd
}

func newFillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fill <ref> <text>",
		Short: "Fill text into element",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			submit, _ := cmd.Flags().GetBool("submit")
			result, err := cli.Fill(args[0], args[1])
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			// If --submit flag, send Enter after fill
			if submit {
				cli.Call("press", map[string]string{"key": "Enter"})
			}
			printRanCode(fmt.Sprintf("await page.locator('[ref=%s]').fill('%s');", args[0], args[1]))
			if result != nil && result.Snapshot != nil {
				printSnapshot(result.Snapshot)
			}
		},
	}
	cmd.Flags().Bool("submit", false, "Press Enter after filling")
	return cmd
}

func newCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check <ref>",
		Short: "Check checkbox or radio button",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			result, err := cli.Check(args[0])
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			printRanCode(fmt.Sprintf("await page.locator('[ref=%s]').check();", args[0]))
			if result != nil && result.Snapshot != nil {
				printSnapshot(result.Snapshot)
			}
		},
	}
}

func newTypeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "type <text>",
		Short: "Type text into focused element",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			submit, _ := cmd.Flags().GetBool("submit")
			result, err := cli.Type(args[0])
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if submit {
				cli.Call("press", map[string]string{"key": "Enter"})
			}
			printRanCode(fmt.Sprintf("await page.keyboard.type('%s');", args[0]))
			if result != nil && result.Snapshot != nil {
				printSnapshot(result.Snapshot)
			}
		},
	}
	cmd.Flags().Bool("submit", false, "Press Enter after typing")
	return cmd
}

func newPressCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "press <key>",
		Short: "Press a key",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			result, err := cli.Press(args[0])
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			printRanCode(fmt.Sprintf("await page.keyboard.press('%s');", args[0]))
			if result != nil && result.Snapshot != nil {
				printSnapshot(result.Snapshot)
			}
		},
	}
}

func newHoverCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hover <ref>",
		Short: "Hover over element",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			result, err := cli.Hover(args[0])
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			printRanCode(fmt.Sprintf("await page.locator('[ref=%s]').hover();", args[0]))
			if result != nil && result.Snapshot != nil {
				printSnapshot(result.Snapshot)
			}
		},
	}
}

func newScreenshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "screenshot [ref]",
		Short: "Take screenshot",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			filename, _ := cmd.Flags().GetString("filename")
			fullPage, _ := cmd.Flags().GetBool("full-page")
			ref := ""
			if len(args) > 0 {
				ref = args[0]
			}
			params := map[string]interface{}{
				"filename": filename,
				"fullPage": fullPage,
			}
			if ref != "" {
				params["ref"] = ref
			}
			resp, err := cli.Call("screenshot", params)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			var result protocol.CommandResult
			json.Unmarshal(resp.Result, &result)
			printRanCode("await page.screenshot();")
			fmt.Println(result.Message)
		},
	}
	cmd.Flags().String("filename", "", "Output filename")
	cmd.Flags().Bool("full-page", false, "Capture full page")
	return cmd
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List active sessions",
		Run: func(cmd *cobra.Command, args []string) {
			if !cli.CanConnect() {
				fmt.Println("No active sessions.")
				return
			}

			sessions, err := cli.List()
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if len(sessions) == 0 {
				fmt.Println("No active sessions.")
				return
			}

			fmt.Println("### Sessions")
			for _, s := range sessions {
				fmt.Printf("- %s:\n", s.Name)
				fmt.Printf("  URL: %s\n", s.URL)
				fmt.Printf("  Title: %s\n", s.Title)
				fmt.Printf("  Browser: %s\n", s.Browser)
			}
		},
	}
}

func newDblClickCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dblclick <ref>",
		Short: "Double-click element",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			modifiers, _ := cmd.Flags().GetStringArray("modifiers")
			params := map[string]interface{}{"ref": args[0]}
			if len(modifiers) > 0 {
				params["modifiers"] = modifiers
			}

			resp, err := cli.Call("dblclick", params)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			var result protocol.CommandResult
			json.Unmarshal(resp.Result, &result)
			printRanCode(fmt.Sprintf("await page.locator('[ref=%s]').dblclick();", args[0]))
			if result.Snapshot != nil {
				printSnapshot(result.Snapshot)
			}
		},
	}
	cmd.Flags().StringArray("modifiers", nil, "Keyboard modifiers (Alt, Control, Meta, Shift)")
	return cmd
}

func newUncheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uncheck <ref>",
		Short: "Uncheck checkbox or radio button",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			resp, err := cli.Call("uncheck", map[string]string{"ref": args[0]})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			var result protocol.CommandResult
			json.Unmarshal(resp.Result, &result)
			printRanCode(fmt.Sprintf("await page.locator('[ref=%s]').uncheck();", args[0]))
			if result.Snapshot != nil {
				printSnapshot(result.Snapshot)
			}
		},
	}
}

func newDragCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "drag <startRef> <endRef>",
		Short: "Drag and drop between elements",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			resp, err := cli.Call("drag", map[string]string{"startRef": args[0], "endRef": args[1]})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			fmt.Printf("Dragged from %s to %s\n", args[0], args[1])
		},
	}
}

func newSelectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "select <ref> <value>",
		Short: "Select option in dropdown",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			resp, err := cli.Call("select", map[string]interface{}{"ref": args[0], "values": args[1:]})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			var result protocol.CommandResult
			json.Unmarshal(resp.Result, &result)
			printRanCode(fmt.Sprintf("await page.locator('[ref=%s]').selectOption('%s');", args[0], strings.Join(args[1:], ",")))
			if result.Snapshot != nil {
				printSnapshot(result.Snapshot)
			}
		},
	}
}

func newEvalCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "eval <script> [ref]",
		Short: "Evaluate JavaScript, optionally on an element",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			params := map[string]string{"script": args[0]}
			if len(args) > 1 {
				params["ref"] = args[1]
			}
			resp, err := cli.Call("eval", params)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			var result protocol.CommandResult
			json.Unmarshal(resp.Result, &result)
			printRanCode(fmt.Sprintf("await page.evaluate(%s);", args[0]))
			fmt.Println(result.Message)
		},
	}
}

func newResizeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resize <width> <height>",
		Short: "Resize browser window",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			width, _ := strconv.Atoi(args[0])
			height, _ := strconv.Atoi(args[1])

			resp, err := cli.Call("resize", map[string]int{"width": width, "height": height})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			fmt.Printf("Resized to %sx%s\n", args[0], args[1])
		},
	}
}

func newUploadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upload <file...>",
		Short: "Upload files",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			resp, err := cli.Call("upload", map[string]interface{}{"files": args})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			fmt.Println("Uploaded", args)
		},
	}
}

func newKeyDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "keydown <key>",
		Short: "Press key down",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			resp, err := cli.Call("keydown", map[string]string{"key": args[0]})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			fmt.Println("Key down:", args[0])
		},
	}
}

func newKeyUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "keyup <key>",
		Short: "Release key",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			resp, err := cli.Call("keyup", map[string]string{"key": args[0]})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			fmt.Println("Key up:", args[0])
		},
	}
}

func newMouseMoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mousemove <x> <y>",
		Short: "Move mouse to position",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			x, _ := strconv.Atoi(args[0])
			y, _ := strconv.Atoi(args[1])

			resp, err := cli.Call("mousemove", map[string]int{"x": x, "y": y})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			fmt.Printf("Mouse moved to %s,%s\n", args[0], args[1])
		},
	}
}

func newMouseDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mousedown [button]",
		Short: "Press mouse button down",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			button := "left"
			if len(args) > 0 {
				button = args[0]
			}

			resp, err := cli.Call("mousedown", map[string]string{"button": button})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			fmt.Println("Mouse down:", button)
		},
	}
}

func newMouseUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mouseup [button]",
		Short: "Release mouse button",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			button := "left"
			if len(args) > 0 {
				button = args[0]
			}

			resp, err := cli.Call("mouseup", map[string]string{"button": button})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			fmt.Println("Mouse up:", button)
		},
	}
}

func newDialogAcceptCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dialog-accept [promptText]",
		Short: "Accept dialog",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			promptText := ""
			if len(args) > 0 {
				promptText = args[0]
			}

			resp, err := cli.Call("dialog-accept", map[string]string{"promptText": promptText})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			fmt.Println("Dialog accept configured")
		},
	}
}

func newDialogDismissCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dialog-dismiss",
		Short: "Dismiss dialog",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			resp, err := cli.Call("dialog-dismiss", nil)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			fmt.Println("Dialog dismiss configured")
		},
	}
}

func newTabNewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tab-new [url]",
		Short: "Open new tab",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			url := ""
			if len(args) > 0 {
				url = args[0]
			}

			resp, err := cli.Call("tab-new", map[string]string{"url": url})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			fmt.Println("New tab opened")
		},
	}
}

func newTabCloseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tab-close [index]",
		Short: "Close tab",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			index := 0
			if len(args) > 0 {
				index, _ = strconv.Atoi(args[0])
			}

			resp, err := cli.Call("tab-close", map[string]int{"index": index})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			fmt.Println("Tab closed")
		},
	}
}

func newTabSelectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tab-select <index>",
		Short: "Select tab by index",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			index, _ := strconv.Atoi(args[0])

			resp, err := cli.Call("tab-select", map[string]int{"index": index})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			fmt.Println("Tab selected:", args[0])
		},
	}
}

func newTabListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tab-list",
		Short: "List all tabs",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			resp, err := cli.Call("tab-list", nil)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			fmt.Println(string(resp.Result))
		},
	}
}

func newStateSaveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "state-save [filename]",
		Short: "Save browser state",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			filename := ""
			if len(args) > 0 {
				filename = args[0]
			}

			resp, err := cli.Call("state-save", map[string]string{"filename": filename})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			fmt.Println("State saved")
		},
	}
}

func newStateLoadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "state-load <filename>",
		Short: "Load browser state",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			resp, err := cli.Call("state-load", map[string]string{"filename": args[0]})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			fmt.Println("State loaded from", args[0])
		},
	}
}

func newKillAllCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "kill-all",
		Short: "Kill all browser processes",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			resp, err := cli.Call("kill-all", nil)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			fmt.Println("All browser processes killed")
		},
	}
}

func newCookieListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cookie-list",
		Short: "List all cookies",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			domain, _ := cmd.Flags().GetString("domain")
			path, _ := cmd.Flags().GetString("path")
			params := map[string]string{}
			if domain != "" {
				params["domain"] = domain
			}
			if path != "" {
				params["path"] = path
			}
			var callParams interface{}
			if len(params) > 0 {
				callParams = params
			}
			resp, err := cli.Call("cookie-list", callParams)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			var result protocol.CommandResult
			json.Unmarshal(resp.Result, &result)
			fmt.Println(result.Message)
		},
	}
	cmd.Flags().String("domain", "", "Filter cookies by domain")
	cmd.Flags().String("path", "", "Filter cookies by path")
	return cmd
}

func newCookieGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cookie-get <name>",
		Short: "Get cookie value by name",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			resp, err := cli.Call("cookie-get", map[string]string{"key": args[0]})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			var result protocol.CommandResult
			json.Unmarshal(resp.Result, &result)
			fmt.Println(result.Message)
		},
	}
}

func newCookieSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cookie-set <name> <value>",
		Short: "Set a cookie with optional flags",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			name := args[0]
			value := args[1]
			domain, _ := cmd.Flags().GetString("domain")
			path, _ := cmd.Flags().GetString("path")
			expires, _ := cmd.Flags().GetFloat64("expires")
			httpOnly, _ := cmd.Flags().GetBool("httpOnly")
			secure, _ := cmd.Flags().GetBool("secure")
			sameSite, _ := cmd.Flags().GetString("sameSite")

			params := map[string]interface{}{
				"name":  name,
				"value": value,
			}
			if domain != "" {
				params["domain"] = domain
			}
			if path != "" {
				params["path"] = path
			}
			if expires != 0 {
				params["expires"] = expires
			}
			if httpOnly {
				params["httpOnly"] = httpOnly
			}
			if secure {
				params["secure"] = secure
			}
			if sameSite != "" {
				params["sameSite"] = sameSite
			}

			resp, err := cli.Call("cookie-set", params)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			fmt.Printf("Cookie set: %s = %s\n", name, value)
		},
	}
	cmd.Flags().String("domain", "", "Cookie domain")
	cmd.Flags().String("path", "", "Cookie path")
	cmd.Flags().Float64("expires", 0, "Cookie expiry as Unix timestamp")
	cmd.Flags().Bool("httpOnly", false, "Set HttpOnly flag")
	cmd.Flags().Bool("secure", false, "Set Secure flag")
	cmd.Flags().String("sameSite", "", "SameSite attribute (Strict, Lax, None)")
	return cmd
}

func newCookieDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cookie-delete <name>",
		Short: "Delete a cookie by name",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			resp, err := cli.Call("cookie-delete", map[string]string{"key": args[0]})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			fmt.Println("Cookie deleted:", args[0])
		},
	}
}

func newCookieClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cookie-clear",
		Short: "Clear all cookies",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			resp, err := cli.Call("cookie-clear", nil)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			fmt.Println("All cookies cleared")
		},
	}
}

func newLocalStorageListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "localstorage-list",
		Short: "List all localStorage entries",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			resp, err := cli.Call("localstorage-list", nil)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			var result protocol.CommandResult
			json.Unmarshal(resp.Result, &result)
			fmt.Println(result.Message)
		},
	}
}

func newLocalStorageGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "localstorage-get <key>",
		Short: "Get localStorage value by key",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			resp, err := cli.Call("localstorage-get", map[string]string{"key": args[0]})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			var result protocol.CommandResult
			json.Unmarshal(resp.Result, &result)
			fmt.Println(result.Message)
		},
	}
}

func newLocalStorageSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "localstorage-set <key> <value>",
		Short: "Set localStorage value",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			resp, err := cli.Call("localstorage-set", map[string]string{"key": args[0], "value": args[1]})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			fmt.Printf("localStorage.%s = %s\n", args[0], args[1])
		},
	}
}

func newLocalStorageDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "localstorage-delete <key>",
		Short: "Delete localStorage entry by key",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			resp, err := cli.Call("localstorage-delete", map[string]string{"key": args[0]})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			fmt.Println("localStorage." + args[0] + " deleted")
		},
	}
}

func newLocalStorageClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "localstorage-clear",
		Short: "Clear all localStorage entries",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			resp, err := cli.Call("localstorage-clear", nil)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			fmt.Println("localStorage cleared")
		},
	}
}

func newSessionStorageListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sessionstorage-list",
		Short: "List all sessionStorage entries",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			resp, err := cli.Call("sessionstorage-list", nil)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			var result protocol.CommandResult
			json.Unmarshal(resp.Result, &result)
			fmt.Println(result.Message)
		},
	}
}

func newSessionStorageGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sessionstorage-get <key>",
		Short: "Get sessionStorage value by key",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			resp, err := cli.Call("sessionstorage-get", map[string]string{"key": args[0]})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			var result protocol.CommandResult
			json.Unmarshal(resp.Result, &result)
			fmt.Println(result.Message)
		},
	}
}

func newSessionStorageSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sessionstorage-set <key> <value>",
		Short: "Set sessionStorage value",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			resp, err := cli.Call("sessionstorage-set", map[string]string{"key": args[0], "value": args[1]})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			fmt.Printf("sessionStorage.%s = %s\n", args[0], args[1])
		},
	}
}

func newSessionStorageDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sessionstorage-delete <key>",
		Short: "Delete sessionStorage entry by key",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			resp, err := cli.Call("sessionstorage-delete", map[string]string{"key": args[0]})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			fmt.Println("sessionStorage." + args[0] + " deleted")
		},
	}
}

func newSessionStorageClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sessionstorage-clear",
		Short: "Clear all sessionStorage entries",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			resp, err := cli.Call("sessionstorage-clear", nil)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			fmt.Println("sessionStorage cleared")
		},
	}
}

func newMouseWheelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mousewheel <dx> <dy>",
		Short: "Scroll using mouse wheel",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			dx, _ := strconv.Atoi(args[0])
			dy, _ := strconv.Atoi(args[1])
			resp, err := cli.Call("mousewheel", map[string]int{"dx": dx, "dy": dy})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			fmt.Printf("Scrolled dx=%s dy=%s\n", args[0], args[1])
		},
	}
}

func newPDFCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pdf",
		Short: "Generate PDF of current page",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			filename, _ := cmd.Flags().GetString("filename")
			resp, err := cli.Call("pdf", map[string]string{"filename": filename})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			var result protocol.CommandResult
			json.Unmarshal(resp.Result, &result)
			fmt.Println(result.Message)
		},
	}
	cmd.Flags().String("filename", "", "Output PDF filename")
	return cmd
}

func newDeleteDataCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete-data",
		Short: "Delete all browser data (cookies, localStorage, sessionStorage)",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			resp, err := cli.Call("delete-data", nil)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			fmt.Println("Browser data deleted")
		},
	}
}

func newCloseAllCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "close-all",
		Short: "Close all browser sessions",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			resp, err := cli.Call("close-all", nil)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			var cmdResult protocol.CommandResult
			json.Unmarshal(resp.Result, &cmdResult)
			fmt.Println(cmdResult.Message)
		},
	}
}

func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Bring browser window to front",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			resp, err := cli.Call("show", nil)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			var cmdResult protocol.CommandResult
			json.Unmarshal(resp.Result, &cmdResult)
			fmt.Println(cmdResult.Message)
		},
	}
}

func newRunCodeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run-code <code>",
		Short: "Run JavaScript code in the browser",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			resp, err := cli.Call("run-code", protocol.RunCodeParams{Code: args[0]})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			var cmdResult protocol.CommandResult
			json.Unmarshal(resp.Result, &cmdResult)
			fmt.Println(cmdResult.Message)
		},
	}
}

func newConsoleCmd() *cobra.Command {
	var level string
	var clear bool
	cmd := &cobra.Command{
		Use:   "console [level]",
		Short: "Show browser console messages",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			if len(args) > 0 {
				level = args[0]
			}
			params := protocol.ConsoleParams{Level: level, Clear: clear}
			resp, err := cli.Call("console", params)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			var cmdResult protocol.CommandResult
			if err := json.Unmarshal(resp.Result, &cmdResult); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			printResult(&cmdResult)
		},
	}
	cmd.Flags().StringVar(&level, "level", "", "Filter by level (info, warning, error, debug)")
	cmd.Flags().BoolVar(&clear, "clear", false, "Clear console messages")
	return cmd
}

func newNetworkCmd() *cobra.Command {
	var static bool
	var clear bool
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Show network requests",
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			params := protocol.NetworkParams{Static: static, Clear: clear}
			resp, err := cli.Call("network", params)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			var cmdResult protocol.CommandResult
			if err := json.Unmarshal(resp.Result, &cmdResult); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			printResult(&cmdResult)
		},
	}
	cmd.Flags().BoolVar(&static, "static", false, "Include static resources")
	cmd.Flags().BoolVar(&clear, "clear", false, "Clear network log")
	return cmd
}

func newTracingStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tracing-start",
		Short: "Start browser tracing",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			resp, err := cli.Call("tracing-start", nil)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			var result protocol.CommandResult
			json.Unmarshal(resp.Result, &result)
			fmt.Println(result.Message)
		},
	}
}

func newTracingStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tracing-stop",
		Short: "Stop browser tracing and save trace file",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			filename, _ := cmd.Flags().GetString("filename")
			resp, err := cli.Call("tracing-stop", protocol.TracingParams{Filename: filename})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			var result protocol.CommandResult
			json.Unmarshal(resp.Result, &result)
			fmt.Println(result.Message)
		},
	}
	cmd.Flags().String("filename", "", "Output filename for trace (default: trace-<timestamp>.zip)")
	return cmd
}

func newRouteCmd() *cobra.Command {
	var status int
	var body string
	var contentType string
	var headers []string
	var removeHeaders []string
	cmd := &cobra.Command{
		Use:   "route <pattern>",
		Short: "Mock network requests matching pattern",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			headerMap := make(map[string]string)
			for _, h := range headers {
				parts := strings.SplitN(h, ":", 2)
				if len(parts) == 2 {
					headerMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
				}
			}
			params := protocol.RouteParams{
				Pattern:       args[0],
				Status:        status,
				Body:          body,
				ContentType:   contentType,
				Headers:       headerMap,
				RemoveHeaders: removeHeaders,
			}
			resp, err := cli.Call("route", params)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			var cmdResult protocol.CommandResult
			if err := json.Unmarshal(resp.Result, &cmdResult); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			printResult(&cmdResult)
		},
	}
	cmd.Flags().IntVar(&status, "status", 200, "HTTP status code")
	cmd.Flags().StringVar(&body, "body", "", "Response body")
	cmd.Flags().StringVar(&contentType, "content-type", "", "Content-Type header")
	cmd.Flags().StringArrayVar(&headers, "header", nil, "Additional headers (key:value)")
	cmd.Flags().StringArrayVar(&removeHeaders, "remove-header", nil, "Headers to remove")
	return cmd
}

func newRouteListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "route-list",
		Short: "List active network routes",
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			resp, err := cli.Call("route-list", nil)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			var cmdResult protocol.CommandResult
			if err := json.Unmarshal(resp.Result, &cmdResult); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			printResult(&cmdResult)
		},
	}
}

func newUnrouteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unroute [pattern]",
		Short: "Remove network route(s)",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			pattern := ""
			if len(args) > 0 {
				pattern = args[0]
			}
			params := protocol.UnrouteParams{Pattern: pattern}
			resp, err := cli.Call("unroute", params)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			var cmdResult protocol.CommandResult
			if err := json.Unmarshal(resp.Result, &cmdResult); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			printResult(&cmdResult)
		},
	}
}

func newInstallCmd() *cobra.Command {
	var installSkills bool
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Initialize workspace",
		Run: func(cmd *cobra.Command, args []string) {
			if installSkills {
				// Install sw skill files to .claude/skills/sw/
				destDir := filepath.Join(".claude", "skills", "sw")
				if err := os.MkdirAll(destDir, 0755); err != nil {
					fmt.Fprintln(os.Stderr, "Error creating skills directory:", err)
					os.Exit(1)
				}
				entries, err := skills.Files.ReadDir("sw")
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error reading embedded skills:", err)
					os.Exit(1)
				}
				for _, e := range entries {
					if e.IsDir() {
						continue
					}
					content, err := skills.Files.ReadFile("sw/" + e.Name())
					if err != nil {
						fmt.Fprintln(os.Stderr, "Error reading skill file:", err)
						os.Exit(1)
					}
					dest := filepath.Join(destDir, e.Name())
					if err := os.WriteFile(dest, content, 0644); err != nil {
						fmt.Fprintln(os.Stderr, "Error writing skill file:", err)
						os.Exit(1)
					}
					fmt.Println("Installed:", dest)
				}
				fmt.Println("Skills installed to", destDir)
			} else {
				fmt.Println("workspace initialized")
			}
		},
	}
	cmd.Flags().BoolVar(&installSkills, "skills", false, "install skills for claude / github copilot")
	return cmd
}

func newDevicesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "devices",
		Short: "List available devices for emulation",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			pw, err := playwright.Run()
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			defer pw.Stop()

			names := make([]string, 0, len(pw.Devices))
			for name := range pw.Devices {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				fmt.Println(name)
			}
		},
	}
}

func newVideoStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "video-start",
		Short: "Start video recording",
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			resp, err := cli.Call("video-start", nil)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			var result protocol.CommandResult
			if err := json.Unmarshal(resp.Result, &result); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			printResult(&result)
		},
	}
}

func newVideoStopCmd() *cobra.Command {
	var filename string
	cmd := &cobra.Command{
		Use:   "video-stop",
		Short: "Stop video recording",
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			params := protocol.TracingParams{Filename: filename}
			resp, err := cli.Call("video-stop", params)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}
			var result protocol.CommandResult
			if err := json.Unmarshal(resp.Result, &result); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			printResult(&result)
		},
	}
	cmd.Flags().StringVar(&filename, "filename", "", "filename to save the video")
	return cmd
}

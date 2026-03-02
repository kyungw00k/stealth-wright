package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kyungw00k/sw/internal/client"
	"github.com/kyungw00k/sw/internal/daemon"
	"github.com/kyungw00k/sw/pkg/protocol"
	"github.com/spf13/cobra")

var (
	// Global options
	sessionName string
	browserType string
	headed      bool
	persistent  bool
	profile     string
	configFile  string
	stealthMode bool

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
			// Initialize client
			cli = client.NewClient(&client.Config{
				SocketPath: client.DefaultSocketPath(),
			})
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&sessionName, "session", "s", "default", "Session name")
	rootCmd.PersistentFlags().StringVarP(&browserType, "browser", "b", "chromium", "Browser type (chromium, firefox, webkit)")
	rootCmd.PersistentFlags().BoolVar(&headed, "headed", false, "Run in headed mode")
	rootCmd.PersistentFlags().BoolVar(&persistent, "persistent", false, "Persist browser profile to disk")
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "", "Custom profile directory")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Config file path")
	rootCmd.PersistentFlags().BoolVar(&stealthMode, "stealth", true, "Enable stealth mode")

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
		newTabsCmd(),
		newStorageCmd(),
		newCookiesCmd(),
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
		newDaemonCmd(),
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
		newConfigPrintCmd(),
		newRunCodeCmd(),
		newConsoleCmd(),
		newNetworkCmd(),
		newTracingStartCmd(),
		newTracingStopCmd(),
		newRouteCmd(),
		newRouteListCmd(),
		newUnrouteCmd(),
		newDevtoolsStartCmd(),
		newInstallCmd(),
		newInstallBrowserCmd(),
		newVideoStartCmd(),
		newVideoStopCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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

	return cli.StartDaemon(execPath)
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
	fmt.Println("### Page")
	fmt.Printf("- Page URL: %s\n", result.PageURL)
	fmt.Printf("- Page Title: %s\n", result.PageTitle)

	if result.Filename != "" {
		fmt.Println("\n### Snapshot")
		fmt.Printf("- [Snapshot](%s)\n", result.Filename)
	}
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
	return &cobra.Command{
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

			result, err := cli.Open(url,
				client.WithHeaded(headed),
				client.WithBrowser(browserType),
				client.WithStealth(stealthMode),
			)
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
	return &cobra.Command{
		Use:   "snapshot",
		Short: "Generate page snapshot with element references",
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			result, err := cli.Snapshot()
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			printSnapshotVerbose(result)
		},
	}
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
		Use:   "screenshot",
		Short: "Take screenshot",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			filename, _ := cmd.Flags().GetString("filename")
			fullPage, _ := cmd.Flags().GetBool("full-page")
			resp, err := cli.Call("screenshot", map[string]interface{}{
				"filename": filename,
				"fullPage": fullPage,
			})
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

func newDaemonCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "daemon",
		Short:  "Daemon management",
		Hidden: true,
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "start",
			Short: "Start daemon",
			Run: func(cmd *cobra.Command, args []string) {
				socketPath := client.DefaultSocketPath()
				baseDir := client.DefaultBaseDir()

				// Ensure directory exists
				os.MkdirAll(filepath.Dir(socketPath), 0755)
				os.MkdirAll(baseDir, 0755)

				srv, err := daemon.NewServer(&daemon.Config{
					SocketPath: socketPath,
					BaseDir:    baseDir,
				})
				if err != nil {
					fmt.Fprintln(os.Stderr, "Failed to create daemon:", err)
					os.Exit(1)
				}

				if err := srv.Start(); err != nil {
					fmt.Fprintln(os.Stderr, "Failed to start daemon:", err)
					os.Exit(1)
				}

				fmt.Printf("Daemon started on %s\n", socketPath)
				srv.WaitForShutdown()
			},
		},
		&cobra.Command{
			Use:   "stop",
			Short: "Stop daemon",
			Run: func(cmd *cobra.Command, args []string) {
				if !cli.CanConnect() {
					fmt.Println("Daemon is not running.")
					return
				}

				// TODO: Implement proper shutdown via command
				fmt.Println("Use 'sw close' to close the browser.")
			},
		},
		&cobra.Command{
			Use:   "status",
			Short: "Check daemon status",
			Run: func(cmd *cobra.Command, args []string) {
				if cli.CanConnect() {
					fmt.Println("Daemon is running.")
				} else {
					fmt.Println("Daemon is not running.")
				}
			},
		},
	)

	return cmd
}

func newTabsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tabs",
		Short: "Tab management",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List all tabs",
			Run: func(cmd *cobra.Command, args []string) {
				if err := ensureDaemon(); err != nil {
					fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
					os.Exit(1)
				}

				result, err := cli.Call("tabs-list", nil)
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error:", err)
					os.Exit(1)
				}

				fmt.Println("### Tabs")
				fmt.Println(result)
				},
			},
			&cobra.Command{
				Use:   "new <url>",
				Short: "Open new tab",
				Args:  cobra.ExactArgs(1),
				Run: func(cmd *cobra.Command, args []string) {
					if err := ensureDaemon(); err != nil {
						fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
						os.Exit(1)
					}

					_, err := cli.Call("tabs-new", map[string]string{"url": args[0]})
					if err != nil {
						fmt.Fprintln(os.Stderr, "Error:", err)
						os.Exit(1)
					}

					fmt.Println("Tab opened:", args[0])
					},
				},
			&cobra.Command{
				Use:   "close [index]",
				Short: "Close tab",
				Args:  cobra.MaximumNArgs(1),
				Run: func(cmd *cobra.Command, args []string) {
					if err := ensureDaemon(); err != nil {
						fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
						os.Exit(1)
					}

					index := -1
					if len(args) > 0 {
						fmt.Sscanf(args[0], "%d", &index)
					}

					_, err := cli.Call("tabs-close", map[string]int{"index": index})
					if err != nil {
						fmt.Fprintln(os.Stderr, "Error:", err)
						os.Exit(1)
					}

					fmt.Println("Tab closed.")
					},
				},
			&cobra.Command{
				Use:   "switch <index>",
				Short: "Switch to tab",
				Args:  cobra.ExactArgs(1),
				Run: func(cmd *cobra.Command, args []string) {
					if err := ensureDaemon(); err != nil {
						fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
						os.Exit(1)
					}

					var index int
					fmt.Sscanf(args[0], "%d", &index)

					_, err := cli.Call("tabs-switch", map[string]int{"index": index})
					if err != nil {
						fmt.Fprintln(os.Stderr, "Error:", err)
						os.Exit(1)
					}

					fmt.Println("Switched to tab", index)
					},
				},
		)

	return cmd
}

func newStorageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "storage",
		Short: "Storage management",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "get <key>",
			Short: "Get storage value",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				if err := ensureDaemon(); err != nil {
					fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
					os.Exit(1)
				}

				localStorage, _ := cmd.Flags().GetBool("local")
				storageType := "session"
				if localStorage {
					storageType = "local"
					}

				result, err := cli.Call("storage-get", map[string]string{"type": storageType, "key": args[0]})
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error:", err)
					os.Exit(1)
				}

				fmt.Println(result)
				},
			},
			&cobra.Command{
				Use:   "set <key> <value>",
				Short: "Set storage value",
				Args:  cobra.ExactArgs(2),
				Run: func(cmd *cobra.Command, args []string) {
					if err := ensureDaemon(); err != nil {
						fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
						os.Exit(1)
					}

					localStorage, _ := cmd.Flags().GetBool("local")
					storageType := "session"
					if localStorage {
						storageType = "local"
					}

					_, err := cli.Call("storage-set", map[string]string{"type": storageType, "key": args[0], "value": args[1]})
					if err != nil {
						fmt.Fprintln(os.Stderr, "Error:", err)
						os.Exit(1)
					}

					fmt.Printf("Set %s = %s\n", args[0], args[1])
					},
				},
			&cobra.Command{
				Use:   "clear",
				Short: "Clear storage",
				Run: func(cmd *cobra.Command, args []string) {
					if err := ensureDaemon(); err != nil {
						fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
						os.Exit(1)
					}

					localStorage, _ := cmd.Flags().GetBool("local")
					storageType := "session"
					if localStorage {
						storageType = "local"
					}

					_, err := cli.Call("storage-clear", map[string]string{"type": storageType})
					if err != nil {
						fmt.Fprintln(os.Stderr, "Error:", err)
						os.Exit(1)
					}

					fmt.Println("Storage cleared.")
					},
				},
		)

	// Add --local flag to storage commands
	for _, subCmd := range cmd.Commands() {
		subCmd.Flags().BoolP("local", "l", false, "Use localStorage instead of sessionStorage")
		}

	return cmd
}

func newCookiesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cookies",
		Short: "Cookie management",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List all cookies",
			Run: func(cmd *cobra.Command, args []string) {
				if err := ensureDaemon(); err != nil {
					fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
					os.Exit(1)
				}

				result, err := cli.Call("cookies-list", nil)
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error:", err)
					os.Exit(1)
				}

				fmt.Println("### Cookies")
				fmt.Println(result)
				},
			},
			&cobra.Command{
				Use:   "get <name>",
				Short: "Get cookie",
				Args:  cobra.ExactArgs(1),
				Run: func(cmd *cobra.Command, args []string) {
					if err := ensureDaemon(); err != nil {
						fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
						os.Exit(1)
					}

					result, err := cli.Call("cookies-get", map[string]string{"name": args[0]})
					if err != nil {
						fmt.Fprintln(os.Stderr, "Error:", err)
						os.Exit(1)
					}

					fmt.Println(result)
					},
				},
			&cobra.Command{
				Use:   "set <name> <value>",
				Short: "Set cookie",
				Args:  cobra.ExactArgs(2),
				Run: func(cmd *cobra.Command, args []string) {
					if err := ensureDaemon(); err != nil {
						fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
						os.Exit(1)
					}

					_, err := cli.Call("cookies-set", map[string]string{"name": args[0], "value": args[1]})
					if err != nil {
						fmt.Fprintln(os.Stderr, "Error:", err)
						os.Exit(1)
					}

					fmt.Printf("Cookie set: %s = %s\n", args[0], args[1])
					},
				},
			&cobra.Command{
				Use:   "delete <name>",
				Short: "Delete cookie",
				Args:  cobra.ExactArgs(1),
				Run: func(cmd *cobra.Command, args []string) {
					if err := ensureDaemon(); err != nil {
						fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
						os.Exit(1)
					}

					_, err := cli.Call("cookies-delete", map[string]string{"name": args[0]})
					if err != nil {
						fmt.Fprintln(os.Stderr, "Error:", err)
						os.Exit(1)
					}

					fmt.Println("Cookie deleted:", args[0])
					},
				},
			&cobra.Command{
				Use:   "clear",
				Short: "Clear all cookies",
				Run: func(cmd *cobra.Command, args []string) {
					if err := ensureDaemon(); err != nil {
						fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
						os.Exit(1)
					}

					_, err := cli.Call("cookies-clear", nil)
					if err != nil {
						fmt.Fprintln(os.Stderr, "Error:", err)
						os.Exit(1)
					}

					fmt.Println("All cookies cleared.")
					},
				},
	)

	return cmd
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

func newConfigPrintCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config-print",
		Short: "Print current session configuration",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			resp, err := cli.Call("config-print", nil)
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

func newDevtoolsStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "devtools-start",
		Short: "Show browser DevTools",
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}
			resp, err := cli.Call("devtools-start", nil)
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

func newInstallCmd() *cobra.Command {
	var skills bool
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Initialize workspace",
		Run: func(cmd *cobra.Command, args []string) {
			if skills {
				if err := os.MkdirAll("skills", 0755); err != nil {
					fmt.Fprintln(os.Stderr, "Error:", err)
					os.Exit(1)
				}
				fmt.Println("skills directory created")
			} else {
				fmt.Println("workspace initialized")
			}
		},
	}
	cmd.Flags().BoolVar(&skills, "skills", false, "install skills for claude / github copilot")
	return cmd
}

func newInstallBrowserCmd() *cobra.Command {
	var browser string
	cmd := &cobra.Command{
		Use:   "install-browser",
		Short: "Install browser",
		Run: func(cmd *cobra.Command, args []string) {
			if browser == "" {
				browser = "chromium"
			}
			fmt.Printf("Installing %s browser...\n", browser)
			fmt.Println("Run: npx playwright install " + browser)
		},
	}
	cmd.Flags().StringVar(&browser, "browser", "", "browser or chrome channel to use (chrome, firefox, webkit, msedge)")
	return cmd
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

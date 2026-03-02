package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

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
		fmt.Printf("- URL: %s\n", result.Page.URL)
		fmt.Printf("- Title: %s\n", result.Page.Title)
	}
	if result.Message != "" {
		fmt.Println(result.Message)
	}
}

// printSnapshot prints a snapshot result
func printSnapshot(result *protocol.SnapshotResult) {
	fmt.Println("### Page")
	fmt.Printf("- URL: %s\n", result.PageURL)
	fmt.Printf("- Title: %s\n", result.PageTitle)

	if result.Filename != "" {
		fmt.Println("\n### Snapshot")
		fmt.Printf("- [Snapshot](%s)\n", result.Filename)
	}

	if len(result.Elements) > 0 {
		fmt.Println("\n### Elements")
		for _, el := range result.Elements {
			text := el.Text
			if len(text) > 40 {
				text = text[:37] + "..."
			}
			if text != "" {
				fmt.Printf("- %s: <%s> %q\n", el.Ref, el.TagName, text)
			} else {
				fmt.Printf("- %s: <%s>\n", el.Ref, el.TagName)
			}
		}
	}
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

			printResult(result)

			// Auto snapshot
			snap, err := cli.Snapshot()
			if err == nil {
				printSnapshot(snap)
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

			printResult(result)

			// Auto snapshot
			snap, err := cli.Snapshot()
			if err == nil {
				printSnapshot(snap)
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
			printResult(&result)
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
			printResult(&result)
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
			printResult(&result)
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

			printSnapshot(result)
		},
	}
}

func newClickCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "click <ref>",
		Short: "Click element",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			result, err := cli.Click(args[0])
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			printResult(result)
		},
	}
}

func newFillCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fill <ref> <text>",
		Short: "Fill text into element",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			result, err := cli.Fill(args[0], args[1])
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			printResult(result)
		},
	}
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

			printResult(result)
		},
	}
}

func newTypeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "type <text>",
		Short: "Type text into focused element",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			result, err := cli.Type(args[0])
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			printResult(result)
		},
	}
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

			printResult(result)
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

			printResult(result)
		},
	}
}

func newScreenshotCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "screenshot [filename]",
		Short: "Take screenshot",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			result, err := cli.Screenshot()
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			printResult(result)
		},
	}
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
		Use:   "daemon",
		Short: "Daemon management",
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
	return &cobra.Command{
		Use:   "dblclick <ref>",
		Short: "Double-click element",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			resp, err := cli.Call("dblclick", map[string]string{"ref": args[0]})
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}

			if resp.Error != nil {
				fmt.Fprintln(os.Stderr, "Error:", resp.Error.Message)
				os.Exit(1)
			}

			fmt.Println("Double-clicked", args[0])
		},
	}
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

			fmt.Println("Unchecked", args[0])
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

			fmt.Println("Selected", args[1:], "in", args[0])
		},
	}
}

func newEvalCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "eval <script>",
		Short: "Evaluate JavaScript",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureDaemon(); err != nil {
				fmt.Fprintln(os.Stderr, "Failed to connect to daemon:", err)
				os.Exit(1)
			}

			resp, err := cli.Call("eval", map[string]string{"script": args[0]})
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

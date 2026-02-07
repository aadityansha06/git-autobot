package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aadityansha/autogit/internal/ai"
	"github.com/aadityansha/autogit/internal/config"
	"github.com/aadityansha/autogit/internal/daemon"
	"github.com/aadityansha/autogit/internal/git"
	"github.com/aadityansha/autogit/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

const Version = "1.0.0"

var rootCmd = &cobra.Command{
	Use:   "autogit",
	Short: "Automated Git version control with AI-generated commit messages",
	Long:  "Autogit is a CLI tool that automatically commits and pushes your changes using AI-generated commit messages.",
	Version: Version,
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize autogit daemon for the current repository",
	Long:  "Detects the Git root directory and starts a background daemon that monitors for changes.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Detect Git root
		rootPath, err := git.GetRootPath()
		if err != nil {
			return fmt.Errorf("failed to detect Git root: %w", err)
		}
		
		fmt.Printf("Detected Git root: %s\n", rootPath)
		
		// Check if daemon already exists for this repo
		daemonInfo, _ := config.LoadDaemonInfo()
		if daemonInfo != nil && daemonInfo.RepoPath == rootPath {
			// Check if process is still running
			if isProcessRunning(daemonInfo.PID) {
				return fmt.Errorf("daemon is already running for this repository (PID: %d)", daemonInfo.PID)
			}
		}
		
		// Load config
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		
		// Validate API key before starting daemon
		if err := ai.ValidateAPIKey(cfg.AIProvider, cfg.APIKey, cfg.BaseURL); err != nil {
			return fmt.Errorf("API key validation failed: %w\nPlease configure your API key using 'autogit --menu'", err)
		}
		
		fmt.Printf("✓ API key validated successfully\n")
		
		// Update root path in config
		cfg.RootPath = rootPath
		if err := config.SaveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		
		// Start daemon process
		if err := daemon.StartDaemonProcess(rootPath); err != nil {
			return fmt.Errorf("failed to start daemon: %w", err)
		}
		
		fmt.Printf("✓ Daemon started successfully\n")
		fmt.Printf("Repository: %s\n", rootPath)
		fmt.Printf("Use 'autogit --menu' to view the dashboard\n")
		
		return nil
	},
}

var menuCmd = &cobra.Command{
	Use:   "menu",
	Short: "Open interactive TUI dashboard",
	Long:  "Opens a terminal UI with dashboard, logs, and settings tabs.",
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := tui.NewModel()
		if err != nil {
			return fmt.Errorf("failed to initialize TUI: %w", err)
		}
		
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}
		
		return nil
	},
}

var startDaemonCmd = &cobra.Command{
	Use:    "start-daemon",
	Short:  "Internal command to start daemon (do not call directly)",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("root path required")
		}
		
		rootPath := args[0]
		
		// Load config
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		
		// Create daemon
		d, err := daemon.NewDaemon(cfg, rootPath)
		if err != nil {
			return fmt.Errorf("failed to create daemon: %w", err)
		}
		
		// Setup signal handling
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		
		// Start daemon
		d.Start()
		
		// Wait for signal
		<-sigChan
		
		// Stop daemon
		d.Stop()
		
		// Clean up daemon info
		config.DeleteDaemonInfo()
		
		return nil
	},
}

var pauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Pause the running daemon",
	Long:  "Stops the background daemon for the current repository.",
	RunE: func(cmd *cobra.Command, args []string) error {
		daemonInfo, err := config.LoadDaemonInfo()
		if err != nil || daemonInfo == nil {
			return fmt.Errorf("no daemon is running")
		}
		
		// Check if process is running
		if !isProcessRunning(daemonInfo.PID) {
			config.DeleteDaemonInfo()
			return fmt.Errorf("daemon process not found (may have crashed)")
		}
		
		// Kill the process
		process, err := os.FindProcess(daemonInfo.PID)
		if err != nil {
			return fmt.Errorf("failed to find process: %w", err)
		}
		
		if err := process.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to stop daemon: %w", err)
		}
		
		// Clean up daemon info
		config.DeleteDaemonInfo()
		
		fmt.Printf("✓ Daemon stopped successfully\n")
		
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status",
	RunE: func(cmd *cobra.Command, args []string) error {
		daemonInfo, err := config.LoadDaemonInfo()
		if err != nil || daemonInfo == nil {
			fmt.Println("Status: Not running")
			return nil
		}
		
		running := isProcessRunning(daemonInfo.PID)
		if !running {
			fmt.Println("Status: Process not found (may have crashed)")
			return nil
		}
		
		fmt.Printf("Status: %s\n", daemonInfo.Status)
		fmt.Printf("PID: %d\n", daemonInfo.PID)
		fmt.Printf("Repository: %s\n", daemonInfo.RepoPath)
		
		return nil
	},
}

func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	
	// On Unix, sending signal 0 checks if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(menuCmd)
	rootCmd.AddCommand(startDaemonCmd)
	rootCmd.AddCommand(pauseCmd)
	rootCmd.AddCommand(statusCmd)
	
	// Enable version flag
	rootCmd.SetVersionTemplate("autogit version {{.Version}}\n")
	
	// Alias --menu for menu command
	rootCmd.PersistentFlags().BoolP("menu", "m", false, "Open interactive TUI dashboard")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if menu, _ := cmd.Flags().GetBool("menu"); menu {
			// Execute menu command
			menuCmd.RunE(cmd, args)
			os.Exit(0)
		}
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}


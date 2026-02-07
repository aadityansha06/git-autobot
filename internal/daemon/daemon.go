package daemon

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/aadityansha/autogit/internal/ai"
	"github.com/aadityansha/autogit/internal/config"
	"github.com/aadityansha/autogit/internal/git"
	"github.com/aadityansha/autogit/internal/notify"
)

const (
	StatusRunning = "running"
	StatusError   = "error"
	StatusPaused  = "paused"
)

type Daemon struct {
	config     *config.Config
	aiProvider ai.AIProvider
	ticker     *time.Ticker
	stopChan   chan bool
	status     string
	rootPath   string
	repoName   string
	logFile    *os.File
	logger     *log.Logger
}

func NewDaemon(cfg *config.Config, rootPath string) (*Daemon, error) {
	// Import AI provider
	ai, err := importAIProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create AI provider: %w", err)
	}
	
	repoName := git.GetRepoName(rootPath)
	
	// Setup logging
	logDir := filepath.Join(config.GetConfigDir(), "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}
	
	logPath := filepath.Join(logDir, fmt.Sprintf("%s.log", repoName))
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	
	logger := log.New(logFile, "", log.LstdFlags)
	
	return &Daemon{
		config:     cfg,
		aiProvider: ai,
		status:     StatusRunning,
		rootPath:   rootPath,
		repoName:   repoName,
		logFile:    logFile,
		logger:     logger,
		stopChan:   make(chan bool),
	}, nil
}

// Import AI provider
func importAIProvider(cfg *config.Config) (ai.AIProvider, error) {
	return ai.NewProvider(cfg.AIProvider, cfg.APIKey, cfg.BaseURL)
}

func (d *Daemon) Start() {
	d.logger.Printf("Daemon started for repository: %s", d.rootPath)
	
	// Change to root directory
	if err := git.ChangeToRoot(d.rootPath); err != nil {
		d.logger.Printf("ERROR: Failed to change to root directory: %v", err)
		d.status = StatusError
		return
	}
	
	interval := d.config.GetCheckInterval()
	d.ticker = time.NewTicker(interval)
	
	go d.runLoop()
}

func (d *Daemon) runLoop() {
	// Run initial check
	d.checkAndCommit()
	
	for {
		select {
		case <-d.ticker.C:
			d.checkAndCommit()
		case <-d.stopChan:
			d.ticker.Stop()
			d.logger.Printf("Daemon stopped")
			return
		}
	}
}

func (d *Daemon) checkAndCommit() {
	d.logger.Printf("Checking for changes...")
	
	hasChanges, err := git.HasChanges()
	if err != nil {
		d.logger.Printf("ERROR: Failed to check changes: %v", err)
		return
	}
	
	if !hasChanges {
		d.logger.Printf("No changes detected")
		return
	}
	
	d.logger.Printf("Changes detected, generating commit message...")
	
	// Get diff
	diff, err := git.GetDiff()
	if err != nil {
		d.logger.Printf("ERROR: Failed to get diff: %v", err)
		return
	}
	
	// Generate commit message
	commitMsg, err := d.aiProvider.GenerateCommitMsg(diff)
	if err != nil {
		d.logger.Printf("ERROR: Failed to generate commit message: %v", err)
		// Don't change status to error, just log and retry next cycle
		return
	}
	
	d.logger.Printf("Generated commit message: %s", commitMsg)
	
	// Stage changes
	if err := git.AddAll(); err != nil {
		d.logger.Printf("ERROR: Failed to stage changes: %v", err)
		return
	}
	
	// Commit
	if err := git.Commit(commitMsg); err != nil {
		d.logger.Printf("ERROR: Failed to commit: %v", err)
		return
	}
	
	d.logger.Printf("Committed successfully")
	
	// Push
	if err := git.Push(); err != nil {
		d.logger.Printf("ERROR: Failed to push: %v", err)
		d.status = StatusError
		
		// Notify user
		notify.NotifyError(d.repoName, err.Error())
		
		// Stop the ticker
		if d.ticker != nil {
			d.ticker.Stop()
		}
		
		return
	}
	
	d.logger.Printf("Pushed successfully")
	d.status = StatusRunning
	
	// Notify success
	notify.NotifySuccess(d.repoName, commitMsg)
}

func (d *Daemon) Stop() {
	if d.ticker != nil {
		d.ticker.Stop()
	}
	d.stopChan <- true
	d.logFile.Close()
}

func (d *Daemon) GetStatus() string {
	return d.status
}

// StartDaemonProcess starts a new daemon process in the background
func StartDaemonProcess(rootPath string) error {
	// Get the current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	
	// Resolve absolute path
	absExecPath, err := filepath.Abs(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}
	
	// Create command
	cmd := exec.Command(absExecPath, "start-daemon", rootPath)
	
	// Detach from terminal
	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setsid: true,
		}
	}
	// On Windows, the process will be detached by default when started
	
	// Redirect output to null
	nullFile, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err == nil {
		cmd.Stdout = nullFile
		cmd.Stderr = nullFile
	}
	
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}
	
	// Save daemon info
	daemonInfo := &config.DaemonInfo{
		PID:      cmd.Process.Pid,
		RepoPath: rootPath,
		Status:   StatusRunning,
	}
	
	if err := config.SaveDaemonInfo(daemonInfo); err != nil {
		return fmt.Errorf("failed to save daemon info: %w", err)
	}
	
	return nil
}


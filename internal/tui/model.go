package tui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aadityansha/autogit/internal/ai"
	"github.com/aadityansha/autogit/internal/config"
	"github.com/aadityansha/autogit/internal/daemon"
	"github.com/aadityansha/autogit/internal/git"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	tabDashboard = iota
	tabLogs
	tabSettings
)

type model struct {
	width      int
	height     int
	activeTab  int
	config     *config.Config
	daemonInfo *config.DaemonInfo
	
	// Dashboard
	dashboardViewport viewport.Model
	
	// Logs
	logsViewport viewport.Model
	logLines     []string
	
	// Settings
	settingsList     list.Model
	apiKeyInput      textinput.Model
	baseURLInput     textinput.Model
	intervalInput    textinput.Model
	selectedProvider string
	showAPIKey       bool
	showBaseURL      bool
	focusedInput     int // 0: provider, 1: apiKey, 2: baseURL, 3: interval
	saveMessage      string // Message to show after saving
	
	// Common
	quitting bool
}

type tickMsg time.Time
type clearSaveMsg struct{}

func NewModel() (*model, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}
	
	daemonInfo, _ := config.LoadDaemonInfo()
	
	m := &model{
		activeTab:  tabDashboard,
		config:     cfg,
		daemonInfo: daemonInfo,
		selectedProvider: cfg.AIProvider,
		showAPIKey: false,
		showBaseURL: false,
		focusedInput: 0,
	}
	
	// Initialize viewports
	m.dashboardViewport = viewport.New(0, 0)
	m.logsViewport = viewport.New(0, 0)
	
	// Initialize settings inputs
	m.apiKeyInput = textinput.New()
	m.apiKeyInput.Placeholder = "Enter API key"
	m.apiKeyInput.CharLimit = 200
	m.apiKeyInput.Width = 50
	
	m.baseURLInput = textinput.New()
	m.baseURLInput.Placeholder = "Enter base URL (optional)"
	m.baseURLInput.CharLimit = 200
	m.baseURLInput.Width = 50
	
	m.intervalInput = textinput.New()
	m.intervalInput.Placeholder = "10"
	m.intervalInput.CharLimit = 10
	m.intervalInput.Width = 20
	
	// Load existing values
	if cfg.APIKey != "" {
		m.apiKeyInput.SetValue(cfg.APIKey)
	}
	if cfg.BaseURL != "" {
		m.baseURLInput.SetValue(cfg.BaseURL)
	}
	m.intervalInput.SetValue(fmt.Sprintf("%d", cfg.CheckIntervalMinutes))
	
	// Initialize settings list
	items := []list.Item{
		item{title: "AI Provider", desc: fmt.Sprintf("Current: %s", cfg.AIProvider)},
		item{title: "API Key", desc: "Click to edit"},
		item{title: "Base URL", desc: "Click to edit (for OpenRouter)"},
		item{title: "Check Interval", desc: fmt.Sprintf("Current: %d minutes", cfg.CheckIntervalMinutes)},
		item{title: "Save", desc: "Save settings"},
	}
	
	m.settingsList = list.New(items, itemDelegate{}, 50, 20)
	m.settingsList.Title = "Settings"
	m.settingsList.SetShowStatusBar(false)
	m.settingsList.SetFilteringEnabled(false)
	
	m.loadLogs()
	m.updateDashboard()
	
	return m, nil
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		tick(),
	)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.dashboardViewport.Width = msg.Width - 4
		m.dashboardViewport.Height = msg.Height - 8
		m.logsViewport.Width = msg.Width - 4
		m.logsViewport.Height = msg.Height - 8
		m.settingsList.SetWidth(msg.Width - 4)
		m.settingsList.SetHeight(msg.Height - 8)
		return m, nil
		
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "1":
			m.activeTab = tabDashboard
			m.updateDashboard()
			return m, nil
		case "2":
			m.activeTab = tabLogs
			m.loadLogs()
			return m, nil
		case "3":
			m.activeTab = tabSettings
			return m, nil
		}
		
		// Tab-specific key handling
		switch m.activeTab {
		case tabDashboard:
			return m.handleDashboardKeys(msg)
		case tabLogs:
			return m.updateLogs(msg)
		case tabSettings:
			return m.updateSettings(msg)
		}
		
	case tickMsg:
		m.updateDashboard()
		m.loadLogs()
		return m, tick()
	case clearSaveMsg:
		m.saveMessage = ""
		return m, nil
	}
	
	return m, nil
}

func (m *model) View() string {
	if m.quitting {
		return ""
	}
	
	var content string
	switch m.activeTab {
	case tabDashboard:
		content = m.dashboardViewport.View()
	case tabLogs:
		content = m.logsViewport.View()
	case tabSettings:
		content = m.settingsList.View()
		if m.focusedInput == 1 {
			content += "\n\n" + m.apiKeyInput.View()
		} else if m.focusedInput == 2 {
			content += "\n\n" + m.baseURLInput.View()
		} else if m.focusedInput == 3 {
			content += "\n\n" + m.intervalInput.View()
		}
		if m.saveMessage != "" {
			var style lipgloss.Style
			if strings.HasPrefix(m.saveMessage, "✓") {
				style = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
			} else {
				style = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
			}
			content += "\n\n" + style.Render(m.saveMessage)
		}
	}
	
	return lipgloss.JoinVertical(
		lipgloss.Left,
		renderTabs(m.activeTab),
		content,
		renderHelp(),
	)
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *model) updateDashboard() {
	daemonInfo, _ := config.LoadDaemonInfo()
	m.daemonInfo = daemonInfo
	
	var status string
	var statusColor lipgloss.Color
	if daemonInfo == nil {
		status = "● Stopped"
		statusColor = lipgloss.Color("9")
	} else if daemonInfo.Status == daemon.StatusRunning {
		status = "● Running"
		statusColor = lipgloss.Color("2")
	} else {
		status = "● Error"
		statusColor = lipgloss.Color("9")
	}
	
	statusStyle := lipgloss.NewStyle().Foreground(statusColor).Bold(true)
	
	var repoPath string
	if daemonInfo != nil {
		repoPath = daemonInfo.RepoPath
	} else {
		repoPath = "Not initialized"
	}
	
	var nextCheck string
	if daemonInfo != nil && m.config != nil {
		interval := m.config.GetCheckInterval()
		nextCheck = fmt.Sprintf("Next check in: %s", interval.String())
	} else {
		nextCheck = "N/A"
	}
	
	content := fmt.Sprintf(
		"\n%s\n\nRepository: %s\n%s\n\nPress 'r' to run check now\n",
		statusStyle.Render(status),
		repoPath,
		nextCheck,
	)
	
	m.dashboardViewport.SetContent(content)
}

func (m *model) handleDashboardKeys(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			// Run check now
			if m.daemonInfo != nil {
				// Trigger immediate check (this would need daemon integration)
				m.updateDashboard()
			}
		}
	}
	return m, nil
}

func (m *model) loadLogs() {
	if m.daemonInfo == nil {
		m.logsViewport.SetContent("No daemon running. No logs available.")
		return
	}
	
	logDir := filepath.Join(config.GetConfigDir(), "logs")
	repoName := git.GetRepoName(m.daemonInfo.RepoPath)
	logPath := filepath.Join(logDir, fmt.Sprintf("%s.log", repoName))
	
	data, err := os.ReadFile(logPath)
	if err != nil {
		m.logsViewport.SetContent("No log file found.")
		return
	}
	
	lines := strings.Split(string(data), "\n")
	// Get last 50 lines
	start := len(lines) - 50
	if start < 0 {
		start = 0
	}
	m.logLines = lines[start:]
	
	// Style the log lines
	var styledLines []string
	for _, line := range m.logLines {
		if strings.Contains(line, "ERROR") {
			styledLines = append(styledLines, lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(line))
		} else if strings.Contains(line, "successfully") || strings.Contains(line, "Committed") {
			styledLines = append(styledLines, lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(line))
		} else {
			styledLines = append(styledLines, line)
		}
	}
	
	m.logsViewport.SetContent(strings.Join(styledLines, "\n"))
	m.logsViewport.GotoBottom()
}

func (m *model) updateLogs(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.logsViewport, cmd = m.logsViewport.Update(msg)
	return m, cmd
}

func (m *model) updateSettings(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle input fields first
	if m.focusedInput == 1 {
		var cmd tea.Cmd
		m.apiKeyInput, cmd = m.apiKeyInput.Update(msg)
		// If Enter is pressed in input, blur it and return to list
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
			m.focusedInput = 0
			m.apiKeyInput.Blur()
			m.updateSettingsList()
			return m, nil
		}
		return m, cmd
	} else if m.focusedInput == 2 {
		var cmd tea.Cmd
		m.baseURLInput, cmd = m.baseURLInput.Update(msg)
		// If Enter is pressed in input, blur it and return to list
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
			m.focusedInput = 0
			m.baseURLInput.Blur()
			m.updateSettingsList()
			return m, nil
		}
		return m, cmd
	} else if m.focusedInput == 3 {
		var cmd tea.Cmd
		m.intervalInput, cmd = m.intervalInput.Update(msg)
		// If Enter is pressed in input, blur it and return to list
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
			m.focusedInput = 0
			m.intervalInput.Blur()
			m.updateSettingsList()
			return m, nil
		}
		return m, cmd
	}
	
	// Handle list navigation and actions
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			selected := m.settingsList.SelectedItem().(item)
			switch selected.title {
			case "AI Provider":
				// Cycle through providers
				providers := []string{"gemini", "openai", "openrouter", "anthropic"}
				currentIdx := -1
				for i, p := range providers {
					if p == m.selectedProvider {
						currentIdx = i
						break
					}
				}
				nextIdx := (currentIdx + 1) % len(providers)
				m.selectedProvider = providers[nextIdx]
				m.config.AIProvider = m.selectedProvider
				m.updateSettingsList()
			case "API Key":
				m.focusedInput = 1
				m.apiKeyInput.Focus()
			case "Base URL":
				m.focusedInput = 2
				m.baseURLInput.Focus()
			case "Check Interval":
				m.focusedInput = 3
				m.intervalInput.Focus()
			case "Save":
				// Validate and save settings
				m.config.AIProvider = m.selectedProvider
				m.config.APIKey = m.apiKeyInput.Value()
				m.config.BaseURL = m.baseURLInput.Value()
				
				// Parse interval
				var interval int
				if _, err := fmt.Sscanf(m.intervalInput.Value(), "%d", &interval); err != nil || interval <= 0 {
					m.saveMessage = "Error: Check interval must be a positive number"
					m.focusedInput = 0
					m.updateSettingsList()
					return m, nil
				}
				m.config.CheckIntervalMinutes = interval
				
				// Validate API key
				if err := ai.ValidateAPIKey(m.config.AIProvider, m.config.APIKey, m.config.BaseURL); err != nil {
					m.saveMessage = fmt.Sprintf("Error: %v", err)
					m.focusedInput = 0
					m.updateSettingsList()
					return m, nil
				}
				
				// Save config
				if err := config.SaveConfig(m.config); err != nil {
					m.saveMessage = fmt.Sprintf("Error saving config: %v", err)
					m.focusedInput = 0
					m.updateSettingsList()
					return m, nil
				}
				
				// Success
				m.saveMessage = "✓ Settings saved successfully!"
				m.focusedInput = 0
				m.updateSettingsList()
				
				// Clear message after 3 seconds
				return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
					return clearSaveMsg{}
				})
			}
		case "esc":
			m.focusedInput = 0
			m.apiKeyInput.Blur()
			m.baseURLInput.Blur()
			m.intervalInput.Blur()
		}
	}
	
	var cmd tea.Cmd
	m.settingsList, cmd = m.settingsList.Update(msg)
	return m, cmd
}

func (m *model) updateSettingsList() {
	// Update settings list items to reflect current values
	apiKeyDisplay := m.apiKeyInput.Value()
	if apiKeyDisplay != "" {
		// Mask API key for display
		if len(apiKeyDisplay) > 8 {
			apiKeyDisplay = apiKeyDisplay[:4] + "..." + apiKeyDisplay[len(apiKeyDisplay)-4:]
		} else {
			apiKeyDisplay = "***"
		}
	} else {
		apiKeyDisplay = "Not set"
	}
	
	baseURLDisplay := m.baseURLInput.Value()
	if baseURLDisplay == "" {
		baseURLDisplay = "Not set"
	}
	
	items := []list.Item{
		item{title: "AI Provider", desc: fmt.Sprintf("Current: %s", m.selectedProvider)},
		item{title: "API Key", desc: fmt.Sprintf("Current: %s", apiKeyDisplay)},
		item{title: "Base URL", desc: fmt.Sprintf("Current: %s", baseURLDisplay)},
		item{title: "Check Interval", desc: fmt.Sprintf("Current: %d minutes", m.config.CheckIntervalMinutes)},
		item{title: "Save", desc: "Save settings"},
	}
	m.settingsList.SetItems(items)
}

func renderTabs(activeTab int) string {
	tabs := []string{"Dashboard", "Logs", "Settings"}
	var rendered []string
	
	for i, tab := range tabs {
		if i == activeTab {
			rendered = append(rendered, lipgloss.NewStyle().
				Foreground(lipgloss.Color("6")).
				Bold(true).
				Render(fmt.Sprintf("[%d] %s", i+1, tab)))
		} else {
			rendered = append(rendered, lipgloss.NewStyle().
				Foreground(lipgloss.Color("8")).
				Render(fmt.Sprintf("[%d] %s", i+1, tab)))
		}
	}
	
	return lipgloss.JoinHorizontal(lipgloss.Left, rendered...)
}

func renderHelp() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("Press [1-3] to switch tabs | [q] to quit")
}

// List items for settings
type item struct {
	title, desc string
}

func (i item) FilterValue() string { return i.title }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}
	
	var style lipgloss.Style
	if index == m.Index() {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	} else {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	}
	
	fmt.Fprint(w, style.Render(fmt.Sprintf("%s - %s", i.title, i.desc)))
}


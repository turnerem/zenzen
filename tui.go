package main

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/turnerem/zenzen/core"
)

// Model represents the TUI state
type Model struct {
	logs          map[string]core.Entry
	orderedIDs    []string // Ordered list of IDs for navigation
	selectedIndex int      // Index in orderedIDs
	view          string   // "list" or "detail"
	renderer      *UIRenderer
	width         int
	height        int
	// err      error
}

// NewModel creates a new TUI model
func NewModel(logs map[string]core.Entry) *Model {
	// Extract and sort IDs for consistent ordering
	orderedIDs := make([]string, 0, len(logs))
	for id := range logs {
		orderedIDs = append(orderedIDs, id)
	}
	// You could sort here if desired: sort.Strings(orderedIDs)

	return &Model{
		logs:          logs,
		orderedIDs:    orderedIDs,
		selectedIndex: 0,
		view:          "list",
		renderer:      NewUIRenderer(NewMinimalUI()),
		width:         80,
		height:        24,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}
	return m, nil
}

// handleKey processes keyboard input
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.view == "list" && m.selectedIndex > 0 {
			m.selectedIndex--
		}
	case "down", "j":
		if m.view == "list" && m.selectedIndex < len(m.orderedIDs)-1 {
			m.selectedIndex++
		}
	case "enter", " ":
		if m.view == "list" && len(m.logs) > 0 {
			m.view = "detail"
		}
	case "d": // delete log
		selectedID := m.orderedIDs[m.selectedIndex]
		delete(m.logs, selectedID)
		// Remove from orderedIDs
		m.orderedIDs = append(m.orderedIDs[:m.selectedIndex], m.orderedIDs[m.selectedIndex+1:]...)
		// Adjust selectedIndex if needed
		if m.selectedIndex >= len(m.orderedIDs) && m.selectedIndex > 0 {
			m.selectedIndex--
		}
		if m.view == "detail" && len(m.orderedIDs) > 0 {
			m.view = "list"
		}
	case "esc", "l":
		if m.view == "detail" {
			m.view = "list"
		}
	case "n":
		// TODO: Create new log
	}
	return m, nil
}

// View renders the UI
func (m Model) View() string {
	if len(m.logs) == 0 {
		return "No logs found. Run 'go run . setup' to create test data.\n\nPress 'q' to quit.\n"
	}

	switch m.view {
	case "list":
		return m.renderListView()
	case "detail":
		return m.renderDetailView()
	}
	return ""
}

// generateFigletHeader generates a large ASCII art header using figlet
func generateFigletHeader(text string) string {
	cmd := exec.Command("figlet", "-f", "slant", text)
	output, err := cmd.Output()
	if err != nil {
		// Fallback if figlet fails
		return text
	}
	return string(output)
}

// applyBorder applies a bright pink rounded border to content
func (m Model) applyBorder(content []string) string {
	innerContent := lipgloss.JoinVertical(lipgloss.Left, content...)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF1493")).
		Padding(1, 1).
		Width(m.width - 4).
		Height(m.height - 2).
		Render(innerContent)
}

// renderListView renders the list of logs
func (m Model) renderListView() string {
	headerText := generateFigletHeader("LOGS")
	headerText = lipgloss.NewStyle().
		Foreground(lipgloss.Color("4")).
		Render(headerText)
	// TODO: center the header horizontally?
	// headerText = lipgloss.Place(m.width-4, lipgloss.Height(headerText), lipgloss.Center, lipgloss.Top, headerText)
	headerLines := strings.Split(headerText, "\n")

	// List items
	var listItems []string
	for i, id := range m.orderedIDs {
		log := m.logs[id]
		selected := i == m.selectedIndex

		var line string
		if selected {
			// Highlight selected item
			line = lipgloss.NewStyle().
				Foreground(lipgloss.Color("11")).
				Bold(true).
				Background(lipgloss.Color("4")).
				Padding(0, 1).
				Render(fmt.Sprintf("▶ %s", log.Title))
		} else {
			// Normal item
			line = fmt.Sprintf("  %s", log.Title)
		}
		listItems = append(listItems, line)
	}

	// Footer help
	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("↑/↓ (j/k) navigate | enter select | n new | q quit")

	// Build content with header and items
	var content []string
	content = append(content, headerLines...)
	content = append(content, "")

	// Limit items shown based on available height
	availableHeight := m.height - 6 // Account for borders, padding, header, and footer
	visibleItems := listItems
	if len(listItems) > availableHeight {
		visibleItems = listItems[:availableHeight]
	}
	content = append(content, visibleItems...)

	content = append(content, "")
	content = append(content, help)

	return m.applyBorder(content)
}

// renderDetailView renders the detail view of selected log
func (m Model) renderDetailView() string {
	if len(m.orderedIDs) == 0 || m.selectedIndex >= len(m.orderedIDs) {
		return "Error: No log selected\n"
	}

	selectedID := m.orderedIDs[m.selectedIndex]
	log := m.logs[selectedID]
	var content []string

	// Header with back instruction
	// header := lipgloss.NewStyle().
	// 	Foreground(lipgloss.Color("4")).
	// 	Bold(true).
	// 	Render("📋 core.Entry Details")

	// content = append(content, header)
	// content = append(content, "")

	// Use the renderer to display the log
	logRendered := m.renderer.RenderEntry(log)

	// Limit log content height
	availableHeight := m.height - 6
	logLines := strings.Split(logRendered, "\n")
	if len(logLines) > availableHeight {
		logLines = logLines[:availableHeight]
	}
	trimmedLog := strings.Join(logLines, "\n")

	content = append(content, trimmedLog)
	content = append(content, "")

	// Footer
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("esc go back | q quit")

	content = append(content, footer)

	return m.applyBorder(content)
}

// StartTUI starts the interactive TUI
func StartTUI(entries map[string]core.Entry) error {
	model := NewModel(entries)
	p := tea.NewProgram(model, tea.WithAltScreen())

	_, err := p.Run()
	return err
}

package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/turnerem/zenzen/core"
)

// SaveFunc is a function that saves entries to storage
type SaveFunc func(entries map[string]core.Entry) error

// Model represents the TUI state
type Model struct {
	entries       map[string]core.Entry
	orderedIDs    []string
	saveFn        SaveFunc
	selectedIndex int // Index in OrderedIDs
	view          string // "list", "detail", or "edit"
	textarea      textarea.Model
	renderer      *UIRenderer
	width         int
	height        int
}

// NewModel creates a new TUI model
func NewModel(entries map[string]core.Entry, orderedIDs []string, saveFn SaveFunc) *Model {
	// Initialize textarea
	ta := textarea.New()
	ta.Placeholder = "Enter log body..."
	ta.Focus()

	return &Model{
		entries:       entries,
		orderedIDs:    orderedIDs,
		saveFn:        saveFn,
		selectedIndex: 0,
		view:          "list",
		textarea:      ta,
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
	var cmd tea.Cmd

	// When in edit mode, let textarea handle its updates
	if m.view == "edit" {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			// Handle our escape/save logic first
			if msg.String() == "esc" {
				selectedID := m.orderedIDs[m.selectedIndex]
				entry := m.entries[selectedID]
				entry.Body = m.textarea.Value()
				m.entries[selectedID] = entry
				// Save to disk
				if err := m.saveFn(m.entries); err != nil {
					log.Printf("Error saving notes: %v", err)
				}
				m.view = "list"
				return m, nil
			}
			// Otherwise let textarea handle the key
		}
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	}

	// Handle other messages
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
		if m.view == "list" && len(m.entries) > 0 {
			// Load current entry body into textarea
			selectedID := m.orderedIDs[m.selectedIndex]
			entry := m.entries[selectedID]
			m.textarea.SetValue(entry.Body)
			m.view = "edit"
		}
	case "d": // delete log
		if m.view == "list" && len(m.orderedIDs) > 0 {
			selectedID := m.orderedIDs[m.selectedIndex]
			delete(m.entries, selectedID)
			// Remove from orderedIDs
			m.orderedIDs = append(m.orderedIDs[:m.selectedIndex], m.orderedIDs[m.selectedIndex+1:]...)
			// Save to disk
			if err := m.saveFn(m.entries); err != nil {
				log.Printf("Error saving notes: %v", err)
			}
			// Adjust selectedIndex if needed
			if m.selectedIndex >= len(m.orderedIDs) && m.selectedIndex > 0 {
				m.selectedIndex--
			}
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
	if len(m.entries) == 0 {
		return "No logs found. Run 'go run . setup' to create test data.\n\nPress 'q' to quit.\n"
	}

	switch m.view {
	case "list":
		return m.renderListView()
	case "detail":
		return m.renderDetailView()
	case "edit":
		return m.renderEditView()
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
		log := m.entries[id]
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
		Render("↑/↓ (j/k) navigate | enter edit | d delete | n new | q quit")

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
	log := m.entries[selectedID]
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
		Render("e edit | esc go back | q quit")

	content = append(content, footer)

	return m.applyBorder(content)
}

// renderEditView renders the edit view with metadata and textarea
func (m Model) renderEditView() string {
	if len(m.orderedIDs) == 0 || m.selectedIndex >= len(m.orderedIDs) {
		return "Error: No log selected\n"
	}

	selectedID := m.orderedIDs[m.selectedIndex]
	log := m.entries[selectedID]
	var content []string

	// Display metadata (read-only)
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("4")).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	content = append(content, titleStyle.Render("Editing: "+log.Title))
	content = append(content, "")

	// Tags
	if len(log.Tags) > 0 {
		content = append(content, labelStyle.Render("Tags: ")+strings.Join(log.Tags, ", "))
	}

	// Duration info
	if log.EstimatedDuration > 0 {
		content = append(content, labelStyle.Render(fmt.Sprintf("Estimated: %v", log.EstimatedDuration)))
	}

	content = append(content, "")
	content = append(content, labelStyle.Render("Body:"))

	// Textarea for editing body
	content = append(content, m.textarea.View())
	content = append(content, "")

	// Footer
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("esc save & exit | ctrl+c quit without saving")

	content = append(content, footer)

	return m.applyBorder(content)
}

// StartTUI starts the interactive TUI
func StartTUI(entries map[string]core.Entry, orderedIDs []string, saveFn SaveFunc) error {
	model := NewModel(entries, orderedIDs, saveFn)
	p := tea.NewProgram(model, tea.WithAltScreen())

	_, err := p.Run()
	return err
}

package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/turnerem/zenzen/core"
	"github.com/turnerem/zenzen/logger"
	"golang.org/x/term"
)

// SaveEntryFunc is a function that saves a single entry to storage
type SaveEntryFunc func(entry core.Entry) error

// DeleteEntryFunc is a function that deletes a single entry from storage
type DeleteEntryFunc func(id string) error

// Model represents the TUI state
type Model struct {
	entries            map[string]core.Entry
	orderedIDs         []string
	saveEntryFn        SaveEntryFunc
	deleteEntryFn      DeleteEntryFunc
	selectedIndex      int    // Index in OrderedIDs
	view               string // "list", "detail", or "edit"
	titleInput         textinput.Model
	tagsInput          textinput.Model
	estimatedInput     textinput.Model
	bodyTextarea       textarea.Model
	focusIndex         int      // 0=title, 1=tags, 2=estimated, 3=body
	availableTags      []string // All unique tags from all entries
	tagSuggestions     []string // Filtered suggestions based on input
	selectedSuggest    int      // Index of selected suggestion
	showTagSuggestions bool     // Whether to show tag suggestions
	renderer           *UIRenderer
	width              int
	height             int
}

// NewModel creates a new TUI model
func NewModel(entries map[string]core.Entry, saveEntryFn SaveEntryFunc, deleteEntryFn DeleteEntryFunc, width, height int) *Model {
	// Initialize title input
	titleInput := textinput.New()
	titleInput.Placeholder = "Entry Title"
	titleInput.CharLimit = 100

	// Initialize tags input
	tagsInput := textinput.New()
	tagsInput.Placeholder = "tag1, tag2, tag3"
	tagsInput.CharLimit = 200

	// Initialize estimated duration input
	estimatedInput := textinput.New()
	estimatedInput.Placeholder = "e.g. 5d, 1h30m, 2d5h (d/h/m/w)"
	estimatedInput.CharLimit = 20

	// Initialize body textarea
	bodyTextarea := textarea.New()
	bodyTextarea.Placeholder = "enter body..."

	// Build initial ordering from entries sorted by StartedAtTimestamp (most recent first)
	orderedIDs := make([]string, 0, len(entries))
	for id := range entries {
		orderedIDs = append(orderedIDs, id)
	}

	// Sort by StartedAtTimestamp, most recent first
	sort.Slice(orderedIDs, func(i, j int) bool {
		entryI := entries[orderedIDs[i]]
		entryJ := entries[orderedIDs[j]]

		// Handle zero timestamps - put entries without timestamps at the end
		if entryI.StartedAtTimestamp.IsZero() && !entryJ.StartedAtTimestamp.IsZero() {
			return false
		}
		if !entryI.StartedAtTimestamp.IsZero() && entryJ.StartedAtTimestamp.IsZero() {
			return true
		}

		// Both have timestamps (or both are zero) - most recent first (descending order)
		return entryI.StartedAtTimestamp.After(entryJ.StartedAtTimestamp)
	})

	// Collect all unique tags from all entries
	tagSet := make(map[string]bool)
	for _, entry := range entries {
		for _, tag := range entry.Tags {
			tagSet[tag] = true
		}
	}
	availableTags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		availableTags = append(availableTags, tag)
	}

	return &Model{
		entries:            entries,
		orderedIDs:         orderedIDs,
		saveEntryFn:        saveEntryFn,
		deleteEntryFn:      deleteEntryFn,
		selectedIndex:      0,
		view:               "list",
		titleInput:         titleInput,
		tagsInput:          tagsInput,
		estimatedInput:     estimatedInput,
		bodyTextarea:       bodyTextarea,
		focusIndex:         0,
		availableTags:      availableTags,
		tagSuggestions:     []string{},
		selectedSuggest:    0,
		showTagSuggestions: false,
		renderer:           NewUIRenderer(NewMinimalUI()),
		width:              width,
		height:             height,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// When in edit mode, handle input updates
	if m.view == "edit" {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			// Handle tag suggestions navigation when shown
			if m.focusIndex == 1 && m.showTagSuggestions && len(m.tagSuggestions) > 0 {
				switch msg.String() {
				case "down":
					if m.selectedSuggest < len(m.tagSuggestions)-1 {
						m.selectedSuggest++
					}
					return m, nil
				case "up":
					if m.selectedSuggest > 0 {
						m.selectedSuggest--
					}
					return m, nil
				case "enter":
					// Apply selected suggestion
					selectedTag := m.tagSuggestions[m.selectedSuggest]
					currentVal := m.tagsInput.Value()
					cursorPos := m.tagsInput.Position()

					// Find the tag boundaries around the cursor
					startPos, endPos := m.findTagBoundaries(currentVal, cursorPos)

					// Build new value: before tag + selected tag + after tag
					newVal := currentVal[:startPos] + selectedTag + currentVal[endPos:]
					m.tagsInput.SetValue(newVal)

					// Move cursor to end of the inserted tag
					newCursorPos := startPos + len(selectedTag)
					m.tagsInput.SetCursor(newCursorPos)

					m.showTagSuggestions = false
					m.tagSuggestions = []string{}
					return m, nil
				}
			}

			switch msg.String() {
			case "esc":
				// Save all fields
				selectedID := m.orderedIDs[m.selectedIndex]
				entry := m.entries[selectedID]

				// Save title
				entry.Title = m.titleInput.Value()
				if entry.Title == "" {
					entry.Title = "Untitled"
				}

				// Parse tags from comma-separated input
				tagsStr := m.tagsInput.Value()
				if tagsStr != "" {
					tags := strings.Split(tagsStr, ",")
					for i := range tags {
						tags[i] = strings.TrimSpace(tags[i])
					}
					entry.Tags = tags
				} else {
					entry.Tags = []string{}
				}

				// Parse estimated duration
				estimatedStr := m.estimatedInput.Value()
				if estimatedStr != "" {
					entry.EstimatedDuration = parseDuration(estimatedStr)
				}

				// Save body
				entry.Body = m.bodyTextarea.Value()

				m.entries[selectedID] = entry
				if err := m.saveEntryFn(entry); err != nil {
					logger.Error("entry_save_failed", "error", err.Error())
				}

				// Rebuild available tags after save
				m.availableTags = m.collectAllTags()

				m.view = "list"
				m.showTagSuggestions = false
				return m, nil
			case "tab":
				// Cycle through inputs
				m.focusIndex = (m.focusIndex + 1) % 4
				if m.focusIndex == 0 {
					m.titleInput.Focus()
					m.tagsInput.Blur()
					m.estimatedInput.Blur()
					m.bodyTextarea.Blur()
					m.showTagSuggestions = false
				} else if m.focusIndex == 1 {
					m.titleInput.Blur()
					m.tagsInput.Focus()
					m.estimatedInput.Blur()
					m.bodyTextarea.Blur()
					// Update suggestions when entering tags field
					m.updateTagSuggestions()
				} else if m.focusIndex == 2 {
					m.titleInput.Blur()
					m.tagsInput.Blur()
					m.estimatedInput.Focus()
					m.bodyTextarea.Blur()
					m.showTagSuggestions = false
				} else {
					m.titleInput.Blur()
					m.tagsInput.Blur()
					m.estimatedInput.Blur()
					m.bodyTextarea.Focus()
					m.showTagSuggestions = false
				}
				return m, nil
			}
		}

		// Update the focused input
		if m.focusIndex == 0 {
			m.titleInput, cmd = m.titleInput.Update(msg)
		} else if m.focusIndex == 1 {
			oldVal := m.tagsInput.Value()
			m.tagsInput, cmd = m.tagsInput.Update(msg)
			newVal := m.tagsInput.Value()

			// Update suggestions if value changed
			if oldVal != newVal {
				m.updateTagSuggestions()
			}
		} else if m.focusIndex == 2 {
			oldVal := m.estimatedInput.Value()
			m.estimatedInput, cmd = m.estimatedInput.Update(msg)
			newVal := m.estimatedInput.Value()

			// Validate estimated duration input
			if newVal != oldVal && !m.isValidDurationInput(newVal) {
				// Revert to old value if invalid
				m.estimatedInput.SetValue(oldVal)
			}
		} else {
			m.bodyTextarea, cmd = m.bodyTextarea.Update(msg)
		}
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
			// Load current entry into all inputs
			selectedID := m.orderedIDs[m.selectedIndex]
			entry := m.entries[selectedID]

			// Load title
			m.titleInput.SetValue(entry.Title)

			// Load tags
			m.tagsInput.SetValue(strings.Join(entry.Tags, ", "))

			// Load estimated duration
			if entry.EstimatedDuration > 0 {
				m.estimatedInput.SetValue(formatDuration(entry.EstimatedDuration))
			} else {
				m.estimatedInput.SetValue("")
			}

			// Load body
			m.bodyTextarea.SetValue(entry.Body)

			// Focus on title first
			m.focusIndex = 0
			m.titleInput.Focus()
			m.tagsInput.Blur()
			m.estimatedInput.Blur()
			m.bodyTextarea.Blur()

			// Initialize tag suggestions
			m.updateTagSuggestions()

			m.view = "edit"
		}
	case "d": // delete log
		if m.view == "list" && len(m.orderedIDs) > 0 {
			selectedID := m.orderedIDs[m.selectedIndex]
			delete(m.entries, selectedID)
			// Remove from orderedIDs
			m.orderedIDs = append(m.orderedIDs[:m.selectedIndex], m.orderedIDs[m.selectedIndex+1:]...)
			// Delete from storage
			if err := m.deleteEntryFn(selectedID); err != nil {
				logger.Error("entry_delete_failed", "error", err.Error())
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
		if m.view == "list" {
			// Create new entry
			newID := fmt.Sprintf("%d", time.Now().UnixNano())
			newEntry := core.Entry{
				ID:                    newID,
				Title:                 "New Log Entry",
				Tags:                  []string{},
				StartedAtTimestamp:    time.Now(),
				EndedAtTimestamp:      time.Time{}, // Zero value = in progress
				LastModifiedTimestamp: time.Now(),
				EstimatedDuration:     0,
				Body:                  "",
			}

			// Add to entries map
			m.entries[newID] = newEntry

			// Add to ordered list at the beginning (most recent)
			m.orderedIDs = append([]string{newID}, m.orderedIDs...)

			// Save to storage
			if err := m.saveEntryFn(newEntry); err != nil {
				logger.Error("entry_create_failed", "error", err.Error())
			}

			// Select the new entry and switch to edit mode
			m.selectedIndex = 0
			m.titleInput.SetValue("New Log Entry")
			m.tagsInput.SetValue("")
			m.estimatedInput.SetValue("")
			m.bodyTextarea.SetValue("")
			m.focusIndex = 0
			m.titleInput.Focus()
			m.tagsInput.Blur()
			m.estimatedInput.Blur()
			m.bodyTextarea.Blur()
			m.updateTagSuggestions()
			m.view = "edit"
		}
	}
	return m, nil
}

// View renders the UI
func (m Model) View() string {
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
		// BorderForeground(lipgloss.Color("#32a852")).
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
	if len(m.orderedIDs) == 0 {
		// Show empty state message
		emptyMsg := lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Italic(true).
			Render("No logs yet. Press 'n' to create your first log entry.")
		listItems = append(listItems, emptyMsg)
	} else {
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

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	// Timestamps at the top (read-only)
	timestampStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	if !log.StartedAtTimestamp.IsZero() {
		content = append(content, timestampStyle.Render(fmt.Sprintf("%s: %s",
			core.FieldDisplayNames["StartedAtTimestamp"],
			log.StartedAtTimestamp.Format("2006-01-02 15:04"))))
	}
	if !log.EndedAtTimestamp.IsZero() {
		content = append(content, timestampStyle.Render(fmt.Sprintf("%s: %s",
			core.FieldDisplayNames["EndedAtTimestamp"],
			log.EndedAtTimestamp.Format("2006-01-02 15:04"))))
	}
	if !log.LastModifiedTimestamp.IsZero() {
		content = append(content, timestampStyle.Render(fmt.Sprintf("%s: %s",
			core.FieldDisplayNames["LastModifiedTimestamp"],
			log.LastModifiedTimestamp.Format("2006-01-02 15:04"))))
	}

	content = append(content, "")

	// Editable fields
	content = append(content, labelStyle.Render("title:"))
	content = append(content, m.titleInput.View())
	content = append(content, "")

	content = append(content, labelStyle.Render("tags:"))
	content = append(content, m.tagsInput.View())

	// Show tag suggestions if available
	if m.showTagSuggestions && len(m.tagSuggestions) > 0 {
		suggestionStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("7")).
			Background(lipgloss.Color("8"))

		selectedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("6"))

		for i, suggestion := range m.tagSuggestions {
			if i >= 5 { // Limit to 5 suggestions
				break
			}
			if i == m.selectedSuggest {
				content = append(content, selectedStyle.Render("  > "+suggestion))
			} else {
				content = append(content, suggestionStyle.Render("    "+suggestion))
			}
		}
	}

	content = append(content, "")

	content = append(content, labelStyle.Render("estimated:"))
	content = append(content, m.estimatedInput.View())
	content = append(content, "")

	content = append(content, labelStyle.Render("body:"))
	content = append(content, m.bodyTextarea.View())
	content = append(content, "")

	// Footer
	var footerText string
	if m.showTagSuggestions && len(m.tagSuggestions) > 0 {
		footerText = "↑/↓ select tag | enter apply | tab switch field | esc save & exit"
	} else {
		footerText = "tab: title→tags→estimated→body | esc save & exit | ctrl+c quit"
	}
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render(footerText)

	content = append(content, footer)

	return m.applyBorder(content)
}

// StartTUI starts the interactive TUI
func StartTUI(entries map[string]core.Entry, saveEntryFn SaveEntryFunc, deleteEntryFn DeleteEntryFunc) error {
	// Get initial terminal size
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// Fallback to default values if we can't get terminal size
		width = 80
		height = 24
	}

	model := NewModel(entries, saveEntryFn, deleteEntryFn, width, height)
	p := tea.NewProgram(model, tea.WithAltScreen())

	_, err = p.Run()
	return err
}

// formatDuration converts time.Duration to a human-readable string like "5d", "1h30m"
func formatDuration(d time.Duration) string {
	if d == 0 {
		return ""
	}

	var result string

	// Weeks
	weeks := d / core.WEEK
	if weeks > 0 {
		result += fmt.Sprintf("%dw", weeks)
		d -= weeks * core.WEEK
	}

	// Days
	days := d / core.DAY
	if days > 0 {
		result += fmt.Sprintf("%dd", days)
		d -= days * core.DAY
	}

	// Hours
	hours := d / time.Hour
	if hours > 0 {
		result += fmt.Sprintf("%dh", hours)
		d -= hours * time.Hour
	}

	// Minutes
	minutes := d / time.Minute
	if minutes > 0 {
		result += fmt.Sprintf("%dm", minutes)
	}

	return result
}

// collectAllTags gathers all unique tags from all entries
func (m *Model) collectAllTags() []string {
	tagSet := make(map[string]bool)
	for _, entry := range m.entries {
		for _, tag := range entry.Tags {
			if tag != "" {
				tagSet[tag] = true
			}
		}
	}

	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	return tags
}

// updateTagSuggestions updates the tag suggestions based on current input
func (m *Model) updateTagSuggestions() {
	input := m.tagsInput.Value()
	cursorPos := m.tagsInput.Position()

	// Get the tag at the cursor position
	startPos, endPos := m.findTagBoundaries(input, cursorPos)
	currentTag := strings.TrimSpace(input[startPos:endPos])

	// Filter available tags based on current input
	suggestions := []string{}

	if currentTag == "" {
		// Show all available tags when cursor is in an empty spot
		for _, tag := range m.availableTags {
			if !m.tagAlreadyInInput(tag, input) {
				suggestions = append(suggestions, tag)
			}
		}
	} else {
		// Filter tags based on what's being typed
		currentTagLower := strings.ToLower(currentTag)
		for _, tag := range m.availableTags {
			if strings.HasPrefix(strings.ToLower(tag), currentTagLower) {
				// Don't suggest tags that are already in the input
				if !m.tagAlreadyInInput(tag, input) {
					suggestions = append(suggestions, tag)
				}
			}
		}
	}

	m.tagSuggestions = suggestions
	m.showTagSuggestions = len(suggestions) > 0
	m.selectedSuggest = 0
}

// findTagBoundaries finds the start and end position of the tag at the cursor
func (m *Model) findTagBoundaries(input string, cursorPos int) (start, end int) {
	// Find the comma before the cursor
	start = 0
	for i := cursorPos - 1; i >= 0; i-- {
		if input[i] == ',' {
			start = i + 1
			break
		}
	}

	// Find the comma after the cursor
	end = len(input)
	for i := cursorPos; i < len(input); i++ {
		if input[i] == ',' {
			end = i
			break
		}
	}

	// Trim leading spaces from start position
	for start < len(input) && input[start] == ' ' {
		start++
	}

	return start, end
}

// tagAlreadyInInput checks if a tag is already in the comma-separated input
func (m *Model) tagAlreadyInInput(tag, input string) bool {
	tags := strings.Split(input, ",")
	for _, t := range tags {
		if strings.TrimSpace(t) == tag {
			return true
		}
	}
	return false
}

// isValidDurationInput validates that the duration input contains only valid units
// Allows formats like: 5d, 1h30m, 2d5h, 1w2d3h30m
func (m *Model) isValidDurationInput(input string) bool {
	input = strings.TrimSpace(input)

	// Empty input is valid
	if input == "" {
		return true
	}

	// Parse character by character
	// Valid pattern: (digit+)(unit)(digit+)(unit)...
	i := 0

	for i < len(input) {
		// Expect at least one digit
		if i >= len(input) || input[i] < '0' || input[i] > '9' {
			return false
		}

		// Consume all consecutive digits
		for i < len(input) && input[i] >= '0' && input[i] <= '9' {
			i++
		}

		// If we reached the end, that's OK (still typing the number)
		if i >= len(input) {
			return true
		}

		// Next character must be a valid unit
		if input[i] != 'd' && input[i] != 'h' && input[i] != 'm' && input[i] != 'w' {
			return false
		}

		// Move past the unit
		i++

		// Continue to next segment (if any)
	}

	return true
}

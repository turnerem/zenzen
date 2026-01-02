package main

import (
	"fmt"
	"os"
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
	// Filter and sort state
	sortBy             string // "started_at", "ended_at", "last_modified"
	sortDescending     bool   // true = newest first, false = oldest first
	filterText         string // Search in title and body
	filterTag          string // Filter by specific tag
	filterTextInput    textinput.Model
	filterTagInput     textinput.Model
	filterInputMode    string // "", "text", "tag" - which filter input is active
	pendingKeySequence string // Track multi-character key sequences like "cf", "ct"
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

	// Initialize filter inputs
	filterTextInput := textinput.New()
	filterTextInput.Placeholder = "search text..."
	filterTextInput.CharLimit = 100

	filterTagInput := textinput.New()
	filterTagInput.Placeholder = "tag name..."
	filterTagInput.CharLimit = 50

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
		sortBy:             "started_at",
		sortDescending:     true, // Newest first by default
		filterText:         "",
		filterTag:          "",
		filterTextInput:    filterTextInput,
		filterTagInput:     filterTagInput,
		filterInputMode:    "",
		pendingKeySequence: "",
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle filter input mode
	if m.filterInputMode != "" {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			// Handle tag suggestions navigation when in tag filter mode
			if m.filterInputMode == "tag" && m.showTagSuggestions && len(m.tagSuggestions) > 0 {
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
					// Apply selected tag suggestion as filter
					selectedTag := m.tagSuggestions[m.selectedSuggest]
					m.filterTag = selectedTag
					m.filterInputMode = ""
					m.filterTagInput.Blur()
					m.showTagSuggestions = false
					m.tagSuggestions = []string{}
					m.selectedIndex = 0
					return m, nil
				}
			}

			switch msg.String() {
			case "enter":
				// Apply filter (if no suggestions showing)
				if m.filterInputMode == "text" {
					m.filterText = m.filterTextInput.Value()
				} else if m.filterInputMode == "tag" {
					m.filterTag = m.filterTagInput.Value()
				}
				m.filterInputMode = ""
				m.filterTextInput.Blur()
				m.filterTagInput.Blur()
				m.showTagSuggestions = false
				m.tagSuggestions = []string{}
				m.selectedIndex = 0 // Reset selection
				return m, nil
			case "esc":
				// Cancel filter input
				m.filterInputMode = ""
				m.filterTextInput.Blur()
				m.filterTagInput.Blur()
				m.showTagSuggestions = false
				m.tagSuggestions = []string{}
				return m, nil
			}
		}

		// Update the active filter input
		if m.filterInputMode == "text" {
			m.filterTextInput, cmd = m.filterTextInput.Update(msg)
		} else if m.filterInputMode == "tag" {
			oldVal := m.filterTagInput.Value()
			m.filterTagInput, cmd = m.filterTagInput.Update(msg)
			newVal := m.filterTagInput.Value()

			// Update suggestions if value changed
			if oldVal != newVal {
				m.updateFilterTagSuggestions()
			}
		}
		return m, cmd
	}

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
	key := msg.String()

	// Handle pending key sequences (like "cf", "ct", "cc")
	if m.pendingKeySequence != "" {
		if m.pendingKeySequence == "c" && m.view == "list" {
			switch key {
			case "f": // cf - clear text filter
				m.filterText = ""
				m.selectedIndex = 0
				m.pendingKeySequence = ""
				return m, nil
			case "t": // ct - clear tag filter
				m.filterTag = ""
				m.selectedIndex = 0
				m.pendingKeySequence = ""
				return m, nil
			case "c": // cc - clear all filters
				m.filterText = ""
				m.filterTag = ""
				m.selectedIndex = 0
				m.pendingKeySequence = ""
				return m, nil
			}
		}
		// Invalid second key, clear pending
		m.pendingKeySequence = ""
	}

	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "s": // Cycle sort field
		if m.view == "list" {
			switch m.sortBy {
			case "started_at":
				m.sortBy = "ended_at"
			case "ended_at":
				m.sortBy = "last_modified"
			case "last_modified":
				m.sortBy = "started_at"
			}
			m.selectedIndex = 0 // Reset selection when sorting changes
		}
	case "r": // Reverse sort direction
		if m.view == "list" {
			m.sortDescending = !m.sortDescending
			m.selectedIndex = 0
		}
	case "c": // Start clear filter sequence (c + f/t) or clear all
		if m.view == "list" {
			// Set pending to wait for second character
			m.pendingKeySequence = "c"
			// But also support single "c" to clear all after a brief moment
			// For now, just set pending and user can press c twice or cf/ct
		}
	case "f": // Enter text filter input mode
		if m.view == "list" && m.pendingKeySequence == "" {
			m.filterInputMode = "text"
			m.filterTextInput.SetValue(m.filterText)
			m.filterTextInput.Focus()
		}
	case "t": // Enter tag filter input mode
		if m.view == "list" && m.pendingKeySequence == "" {
			m.filterInputMode = "tag"
			m.filterTagInput.SetValue(m.filterTag)
			m.filterTagInput.Focus()
			m.updateFilterTagSuggestions() // Show tag suggestions
		}
	case "up", "k":
		if m.view == "list" {
			displayIDs := m.getFilteredAndSortedIDs()
			if m.selectedIndex > 0 && len(displayIDs) > 0 {
				m.selectedIndex--
			}
		}
	case "down", "j":
		if m.view == "list" {
			displayIDs := m.getFilteredAndSortedIDs()
			if m.selectedIndex < len(displayIDs)-1 {
				m.selectedIndex++
			}
		}
	case "enter", " ":
		if m.view == "list" && len(m.entries) > 0 {
			// Load current entry into all inputs
			displayIDs := m.getFilteredAndSortedIDs()
			if len(displayIDs) == 0 {
				return m, nil
			}
			selectedID := displayIDs[m.selectedIndex]
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
		if m.view == "list" {
			displayIDs := m.getFilteredAndSortedIDs()
			if len(displayIDs) == 0 {
				return m, nil
			}

			selectedID := displayIDs[m.selectedIndex]

			// Delete from entries map
			delete(m.entries, selectedID)

			// Remove from orderedIDs
			for i, id := range m.orderedIDs {
				if id == selectedID {
					m.orderedIDs = append(m.orderedIDs[:i], m.orderedIDs[i+1:]...)
					break
				}
			}

			// Delete from storage
			if err := m.deleteEntryFn(selectedID); err != nil {
				logger.Error("entry_delete_failed", "error", err.Error())
			}

			// Adjust selectedIndex if needed
			newDisplayIDs := m.getFilteredAndSortedIDs()
			if m.selectedIndex >= len(newDisplayIDs) && m.selectedIndex > 0 {
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

// applyBorder applies a rounded border to content
// Preserved for future use in bordered subspaces
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

// getFilteredAndSortedIDs returns entry IDs filtered and sorted according to current settings
func (m Model) getFilteredAndSortedIDs() []string {
	// Start with all IDs
	ids := make([]string, 0, len(m.entries))
	for id := range m.entries {
		ids = append(ids, id)
	}

	// Apply filters
	var filteredIDs []string
	for _, id := range ids {
		entry := m.entries[id]

		// Filter by text (search in title and body)
		if m.filterText != "" {
			searchText := strings.ToLower(m.filterText)
			title := strings.ToLower(entry.Title)
			body := strings.ToLower(entry.Body)
			if !strings.Contains(title, searchText) && !strings.Contains(body, searchText) {
				continue // Skip this entry
			}
		}

		// Filter by tag
		if m.filterTag != "" {
			hasTag := false
			for _, tag := range entry.Tags {
				if tag == m.filterTag {
					hasTag = true
					break
				}
			}
			if !hasTag {
				continue // Skip this entry
			}
		}

		filteredIDs = append(filteredIDs, id)
	}

	// Sort filtered IDs with stable secondary sort by ID
	sort.Slice(filteredIDs, func(i, j int) bool {
		entryI := m.entries[filteredIDs[i]]
		entryJ := m.entries[filteredIDs[j]]
		idI := filteredIDs[i]
		idJ := filteredIDs[j]

		var timeI, timeJ time.Time
		switch m.sortBy {
		case "started_at":
			timeI = entryI.StartedAtTimestamp
			timeJ = entryJ.StartedAtTimestamp
		case "ended_at":
			timeI = entryI.EndedAtTimestamp
			timeJ = entryJ.EndedAtTimestamp
		case "last_modified":
			timeI = entryI.LastModifiedTimestamp
			timeJ = entryJ.LastModifiedTimestamp
		default:
			timeI = entryI.StartedAtTimestamp
			timeJ = entryJ.StartedAtTimestamp
		}

		// Handle zero timestamps - put entries without timestamps at the end
		if timeI.IsZero() && !timeJ.IsZero() {
			return false
		}
		if !timeI.IsZero() && timeJ.IsZero() {
			return true
		}

		// If both zero or both non-zero, compare timestamps
		if !timeI.Equal(timeJ) {
			if m.sortDescending {
				return timeI.After(timeJ)
			}
			return timeI.Before(timeJ)
		}

		// Timestamps are equal - use ID as stable secondary sort
		// IDs are timestamps, so numeric comparison
		if m.sortDescending {
			return idI > idJ
		}
		return idI < idJ
	})

	return filteredIDs
}

// renderFilterSortSection renders the filter and sort controls
func (m Model) renderFilterSortSection() []string {
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Bold(true)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("7"))

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	var lines []string

	// Sort line
	sortDirection := "↓ newest"
	if !m.sortDescending {
		sortDirection = "↑ oldest"
	}
	sortFieldDisplay := map[string]string{
		"started_at":    "Started",
		"ended_at":      "Ended",
		"last_modified": "Modified",
	}[m.sortBy]

	sortLine := labelStyle.Render("Sort: ") +
		valueStyle.Render(sortFieldDisplay+" "+sortDirection) +
		dimStyle.Render("  [s: field, r: reverse]")
	lines = append(lines, sortLine)

	// Filter line
	filterParts := []string{}
	if m.filterText != "" {
		filterParts = append(filterParts, "text:"+m.filterText)
	}
	if m.filterTag != "" {
		filterParts = append(filterParts, "tag:"+m.filterTag)
	}

	var filterLine string
	if m.pendingKeySequence == "c" {
		// Show pending command hint
		filterLine = labelStyle.Render("Filter: ") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true).Render("c") +
			dimStyle.Render(" + [f: clear text, t: clear tag, c: clear all]")
	} else if len(filterParts) > 0 {
		filterLine = labelStyle.Render("Filter: ") +
			valueStyle.Render(strings.Join(filterParts, ", ")) +
			dimStyle.Render("  [f: text, t: tag, cf: clear text, ct: clear tag, cc: clear all]")
	} else {
		filterLine = labelStyle.Render("Filter: ") +
			dimStyle.Render("none  [f: text, t: tag]")
	}
	lines = append(lines, filterLine)

	return lines
}

// layoutListView arranges all visual elements vertically:
// top border, filter/sort section, list content, spacer, navigation instructions, bottom border
func (m Model) layoutListView(listItems []string, helpText string) string {
	borderStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#32a852"))

	// Build top border
	label := "<< ZENZEN >>"
	slashesNeeded := m.width - len(label)
	if slashesNeeded < 0 {
		slashesNeeded = 0
	}
	leftSlashes := slashesNeeded / 5
	rightSlashes := slashesNeeded - leftSlashes
	topBorder := strings.Repeat("/", leftSlashes) + label + strings.Repeat("/", rightSlashes)

	// Build bottom border
	bottomBorder := strings.Repeat("/", m.width)

	// Get filter/sort section
	filterSortLines := m.renderFilterSortSection()
	filterSortCount := len(filterSortLines)

	// Calculate available space for list items
	// Layout: top border (1) + filter/sort (2) + empty (1) + items + spacer + empty (1) + help (1) + bottom (1)
	reservedLines := 4 + filterSortCount + 1 // top, filter/sort, empty after filter, empty before help, help, bottom
	availableForItems := m.height - reservedLines
	if availableForItems < 0 {
		availableForItems = 0
	}

	// Limit visible items
	visibleItems := listItems
	if len(listItems) > availableForItems {
		visibleItems = listItems[:availableForItems]
	}

	// Calculate spacer to push help and bottom border down
	usedLines := 1 + filterSortCount + 1 + len(visibleItems) + 1 + 1 + 1 // top + filter/sort + empty + items + empty + help + bottom
	spacerLines := m.height - usedLines
	if spacerLines < 0 {
		spacerLines = 0
	}

	// Build final layout
	var result []string
	result = append(result, borderStyle.Render(topBorder))
	result = append(result, filterSortLines...)

	// Show filter input prompt if in filter input mode
	if m.filterInputMode != "" {
		promptStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")).
			Bold(true)

		var promptText string
		var inputView string
		if m.filterInputMode == "text" {
			promptText = "Filter by text: "
			inputView = m.filterTextInput.View()
		} else if m.filterInputMode == "tag" {
			promptText = "Filter by tag: "
			inputView = m.filterTagInput.View()
		}

		result = append(result, "")
		result = append(result, promptStyle.Render(promptText)+inputView)

		// Show tag suggestions if in tag filter mode
		if m.filterInputMode == "tag" && m.showTagSuggestions && len(m.tagSuggestions) > 0 {
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
					result = append(result, selectedStyle.Render("  > "+suggestion))
				} else {
					result = append(result, suggestionStyle.Render("    "+suggestion))
				}
			}
		}

		result = append(result, lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("enter: apply | esc: cancel"))
	} else {
		result = append(result, "") // Empty line after filter/sort
		result = append(result, visibleItems...)

		// Add spacer
		for i := 0; i < spacerLines; i++ {
			result = append(result, "")
		}

		// Add help text (2 lines above bottom: empty line + help line)
		result = append(result, "")
		result = append(result, helpText)
	}

	result = append(result, borderStyle.Render(bottomBorder))

	return strings.Join(result, "\n")
}

// renderListView renders the list of logs
func (m Model) renderListView() string {
	// Get filtered and sorted IDs
	displayIDs := m.getFilteredAndSortedIDs()

	// Build list items
	var listItems []string
	if len(displayIDs) == 0 {
		// Show empty state message
		if len(m.entries) == 0 {
			emptyMsg := lipgloss.NewStyle().
				Foreground(lipgloss.Color("8")).
				Italic(true).
				Render("No logs yet. Press 'n' to create your first log entry.")
			listItems = append(listItems, emptyMsg)
		} else {
			// Has entries but all filtered out
			emptyMsg := lipgloss.NewStyle().
				Foreground(lipgloss.Color("8")).
				Italic(true).
				Render("No entries match current filters. Press 'c' to clear filters.")
			listItems = append(listItems, emptyMsg)
		}
	} else {
		for i, id := range displayIDs {
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

	// Build help text
	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("↑/↓ (j/k) navigate | enter edit | d delete | n new | q quit")

	// Layout everything
	return m.layoutListView(listItems, help)
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

// renderMetadataSection renders the left metadata section (timestamps, title, tags, estimated)
func (m Model) renderMetadataSection() []string {
	displayIDs := m.getFilteredAndSortedIDs()
	if len(displayIDs) == 0 || m.selectedIndex >= len(displayIDs) {
		return []string{"Error: No entry selected"}
	}

	selectedID := displayIDs[m.selectedIndex]
	log := m.entries[selectedID]

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	timestampStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	var lines []string

	// Timestamps (read-only) - always show, even if not set
	if !log.StartedAtTimestamp.IsZero() {
		lines = append(lines, timestampStyle.Render(fmt.Sprintf("%s: %s",
			core.FieldDisplayNames["StartedAtTimestamp"],
			log.StartedAtTimestamp.Format("2006-01-02 15:04"))))
	} else {
		lines = append(lines, timestampStyle.Render(fmt.Sprintf("%s: (not set)",
			core.FieldDisplayNames["StartedAtTimestamp"])))
	}

	if !log.EndedAtTimestamp.IsZero() {
		lines = append(lines, timestampStyle.Render(fmt.Sprintf("%s: %s",
			core.FieldDisplayNames["EndedAtTimestamp"],
			log.EndedAtTimestamp.Format("2006-01-02 15:04"))))
	} else {
		lines = append(lines, timestampStyle.Render(fmt.Sprintf("%s: (not set)",
			core.FieldDisplayNames["EndedAtTimestamp"])))
	}

	if !log.LastModifiedTimestamp.IsZero() {
		lines = append(lines, timestampStyle.Render(fmt.Sprintf("%s: %s",
			core.FieldDisplayNames["LastModifiedTimestamp"],
			log.LastModifiedTimestamp.Format("2006-01-02 15:04"))))
	} else {
		lines = append(lines, timestampStyle.Render(fmt.Sprintf("%s: (not set)",
			core.FieldDisplayNames["LastModifiedTimestamp"])))
	}

	lines = append(lines, "")

	// Title
	lines = append(lines, labelStyle.Render("title:"))
	lines = append(lines, m.titleInput.View())
	lines = append(lines, "")

	// Tags
	lines = append(lines, labelStyle.Render("tags:"))
	lines = append(lines, m.tagsInput.View())

	// Tag suggestions
	if m.showTagSuggestions && len(m.tagSuggestions) > 0 {
		suggestionStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("7")).
			Background(lipgloss.Color("8"))

		selectedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("6"))

		for i, suggestion := range m.tagSuggestions {
			if i >= 5 {
				break
			}
			if i == m.selectedSuggest {
				lines = append(lines, selectedStyle.Render("  > "+suggestion))
			} else {
				lines = append(lines, suggestionStyle.Render("    "+suggestion))
			}
		}
	}

	lines = append(lines, "")

	// Estimated duration
	lines = append(lines, labelStyle.Render("estimated:"))
	lines = append(lines, m.estimatedInput.View())

	return lines
}

// layoutEditView arranges the edit view with two columns: left (metadata) and right (body)
func (m Model) layoutEditView() string {
	borderStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#32a852"))

	// Build borders
	label := "<< ZENZEN >>"
	slashesNeeded := m.width - len(label)
	if slashesNeeded < 0 {
		slashesNeeded = 0
	}
	leftSlashes := slashesNeeded / 5
	rightSlashes := slashesNeeded - leftSlashes
	topBorder := strings.Repeat("/", leftSlashes) + label + strings.Repeat("/", rightSlashes)
	bottomBorder := strings.Repeat("/", m.width)

	// Get metadata (left column)
	metadataLines := m.renderMetadataSection()

	// Calculate widths: metadata gets what it needs, body gets the rest
	// Find the longest line in metadata to determine left column width
	maxMetadataWidth := 0
	for _, line := range metadataLines {
		// Strip ANSI codes to get actual width
		width := lipgloss.Width(line)
		if width > maxMetadataWidth {
			maxMetadataWidth = width
		}
	}

	// Add some padding
	leftWidth := maxMetadataWidth + 4
	if leftWidth > m.width/2 {
		leftWidth = m.width / 2
	}
	rightWidth := m.width - leftWidth - 2 // -2 for spacing

	// Body (right column) - create bordered box
	bodyLabel := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("body:")

	// Calculate body box dimensions (accounting for border)
	bodyBoxWidth := rightWidth
	bodyBoxHeight := m.height - 6

	// Set textarea dimensions (inside the border)
	m.bodyTextarea.SetWidth(bodyBoxWidth - 4)   // -4 for border + padding
	m.bodyTextarea.SetHeight(bodyBoxHeight - 3) // -3 for border + label

	// Create body content with label and textarea
	bodyContent := bodyLabel + "\n" + m.bodyTextarea.View()

	// Apply rounded border to body
	bodyBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#32a852")).
		Padding(0, 1).
		Width(bodyBoxWidth - 2).
		Height(bodyBoxHeight).
		Render(bodyContent)

	// Split the bordered box into lines
	bodyBoxLines := strings.Split(bodyBox, "\n")

	// Calculate available height
	availableHeight := m.height - 4 // top, bottom, footer, empty line

	// Build two-column layout
	maxLines := availableHeight
	var result []string
	result = append(result, borderStyle.Render(topBorder))

	// Render lines side by side
	for i := 0; i < maxLines; i++ {
		var leftLine, rightLine string

		if i < len(metadataLines) {
			leftLine = metadataLines[i]
		} else {
			leftLine = ""
		}

		if i < len(bodyBoxLines) {
			rightLine = bodyBoxLines[i]
		} else {
			rightLine = ""
		}

		// Pad left to fixed width
		leftPadded := lipgloss.NewStyle().
			Width(leftWidth).
			Render(leftLine)

		// Combine (no separator, just spacing)
		combined := leftPadded + rightLine
		result = append(result, combined)
	}

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

	result = append(result, "")
	result = append(result, footer)
	result = append(result, borderStyle.Render(bottomBorder))

	return strings.Join(result, "\n")
}

// renderEditView renders the edit view with metadata and textarea
func (m Model) renderEditView() string {
	return m.layoutEditView()
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

// updateFilterTagSuggestions updates tag suggestions for filter input
func (m *Model) updateFilterTagSuggestions() {
	input := strings.TrimSpace(m.filterTagInput.Value())

	// Filter available tags based on current input
	suggestions := []string{}

	if input == "" {
		// Show all available tags when input is empty
		suggestions = append(suggestions, m.availableTags...)
	} else {
		// Filter tags based on what's being typed
		inputLower := strings.ToLower(input)
		for _, tag := range m.availableTags {
			if strings.HasPrefix(strings.ToLower(tag), inputLower) {
				suggestions = append(suggestions, tag)
			}
		}
	}

	m.tagSuggestions = suggestions
	m.showTagSuggestions = len(suggestions) > 0
	m.selectedSuggest = 0
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

package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/turnerem/zenzen/core"
)

// MinimalUI is a clean, minimal terminal UI
type MinimalUI struct{}

// NewMinimalUI creates a new minimal UI renderer
func NewMinimalUI() *MinimalUI {
	return &MinimalUI{}
}

// Name returns the name of this UI
func (m *MinimalUI) Name() string {
	return "minimal"
}

// RenderEntrysList renders entrys in a simple vertical list
func (m *MinimalUI) RenderEntrysList(entrys []string) string {
	if len(entrys) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Render("No entrys yet. Start with 'gli add <title>'")
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("4")).
		Bold(true)

	var result []string
	result = append(result, "\n📋 Your Notes\n")

	for i, entry := range entrys {
		result = append(result, fmt.Sprintf("  %d. %s", i+1, titleStyle.Render(entry)))
	}

	return strings.Join(result, "\n") + "\n"
}

// RenderEntry renders a single entry entry
func (m *MinimalUI) RenderEntry(entry core.Entry) string {
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Bold(true)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("7"))

	timestampStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	var parts []string

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("4")).
		Bold(true)
	parts = append(parts, titleStyle.Render(entry.Title))

	parts = append(parts, "")

	// Timestamps at the top
	if !entry.StartedAtTimestamp.IsZero() {
		parts = append(parts, timestampStyle.Render(fmt.Sprintf("%s: %s",
			core.FieldDisplayNames["StartedAtTimestamp"],
			entry.StartedAtTimestamp.Format("2006-01-02 15:04"))))
	}
	if !entry.EndedAtTimestamp.IsZero() {
		parts = append(parts, timestampStyle.Render(fmt.Sprintf("%s: %s",
			core.FieldDisplayNames["EndedAtTimestamp"],
			entry.EndedAtTimestamp.Format("2006-01-02 15:04"))))
	}
	if !entry.LastModifiedTimestamp.IsZero() {
		parts = append(parts, timestampStyle.Render(fmt.Sprintf("%s: %s",
			core.FieldDisplayNames["LastModifiedTimestamp"],
			entry.LastModifiedTimestamp.Format("2006-01-02 15:04"))))
	}

	parts = append(parts, "")

	// Tags
	if len(entry.Tags) > 0 {
		tagsStr := strings.Join(entry.Tags, ", ")
		parts = append(parts, labelStyle.Render("🏷 tags:"), valueStyle.Render("  "+tagsStr))
	} else {
		parts = append(parts, labelStyle.Render("🏷 tags:"), valueStyle.Render(""))
	}

	// TTC Prediction
	if entry.EstimatedDuration.String() != "" {
		parts = append(parts, labelStyle.Render("⏱ estimated:"), valueStyle.Render("  "+entry.EstimatedDuration.String()))
	} else {
		parts = append(parts, labelStyle.Render("⏱ estimated:"), valueStyle.Render(""))
	}

	// TTC Actual
	if !entry.EndedAtTimestamp.IsZero() && !entry.StartedAtTimestamp.IsZero() {
		duration := entry.EndedAtTimestamp.Sub(entry.StartedAtTimestamp)
		durFmt := fmt.Sprintf("%v", duration)
		parts = append(parts, labelStyle.Render("✓ actual:"), valueStyle.Render("  "+durFmt))
	} else {
		parts = append(parts, labelStyle.Render("✓ actual:"), valueStyle.Render(""))
	}

	parts = append(parts, "")

	// Body
	parts = append(parts, labelStyle.Render("📝 body:"))
	if entry.Body != "" {
		parts = append(parts, valueStyle.Render(entry.Body))
	} else {
		parts = append(parts, valueStyle.Render(""))
	}

	return strings.Join(parts, "\n")
}

// RenderTags renders tags with counts
func (m *MinimalUI) RenderTags(tags map[string]int) string {
	if len(tags) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Render("No tags found")
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("4")).
		Bold(true)

	tagStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6"))

	countStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("2")).
		Bold(true)

	var result []string
	result = append(result, "\n"+headerStyle.Render("🏷 Tags")+"\n")

	for tag, count := range tags {
		result = append(result, fmt.Sprintf("  %s %s", tagStyle.Render(tag), countStyle.Render(fmt.Sprintf("(%d)", count))))
	}

	return strings.Join(result, "\n") + "\n"
}

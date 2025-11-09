package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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

// RenderLogsList renders logs in a simple vertical list
func (m *MinimalUI) RenderLogsList(logs []string) string {
	if len(logs) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Render("No logs yet. Start with 'gli add <title>'")
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("4")).
		Bold(true)

	var result []string
	result = append(result, "\n📋 Your Logs\n")

	for i, log := range logs {
		result = append(result, fmt.Sprintf("  %d. %s", i+1, titleStyle.Render(log)))
	}

	return strings.Join(result, "\n") + "\n"
}

// RenderLog renders a single log entry
func (m *MinimalUI) RenderLog(log Log) string {
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Bold(true)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("7"))

	var parts []string

	// Title and start time on same line
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("4")).
		Bold(true)
	titleLine := log.Title
	if log.Start != "" {
		titleLine = log.Start + " " + log.Title
	}
	parts = append(parts, titleStyle.Render(titleLine))

	parts = append(parts, "")

	// Tags
	if len(log.Tags) > 0 {
		tagsStr := strings.Join(log.Tags, ", ")
		parts = append(parts, labelStyle.Render("🏷 Tags:"), valueStyle.Render("  "+tagsStr))
	} else {
		parts = append(parts, labelStyle.Render("🏷 Tags:"), valueStyle.Render(""))
	}

	// TTC Prediction
	if log.TTCPrediction != "" {
		parts = append(parts, labelStyle.Render("⏱ Predicted:"), valueStyle.Render("  "+log.TTCPrediction))
	} else {
		parts = append(parts, labelStyle.Render("⏱ Predicted:"), valueStyle.Render(""))
	}

	// TTC Actual
	if log.TTCActual != "" {
		parts = append(parts, labelStyle.Render("✓ Actual:"), valueStyle.Render("  "+log.TTCActual))
	} else {
		parts = append(parts, labelStyle.Render("✓ Actual:"), valueStyle.Render(""))
	}

	parts = append(parts, "")

	// Body
	parts = append(parts, labelStyle.Render("📝 Notes:"))
	if log.Body != "" {
		parts = append(parts, valueStyle.Render(log.Body))
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

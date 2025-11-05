package main

import "fmt"

// UI defines the interface for different UI renderers
type UI interface {
	// RenderLogsList renders a list of log titles
	RenderLogsList(logs []string) string

	// RenderLog renders a single log entry
	RenderLog(log Log) string

	// RenderTags renders tags with counts
	RenderTags(tags map[string]int) string

	// Name returns the name of this UI renderer
	Name() string
}

// UIRenderer holds the current UI implementation
type UIRenderer struct {
	current UI
}

// NewUIRenderer creates a new UI renderer with a default prototype
func NewUIRenderer(prototype UI) *UIRenderer {
	return &UIRenderer{current: prototype}
}

// Switch changes to a different UI renderer
func (r *UIRenderer) Switch(ui UI) {
	r.current = ui
}

// RenderLogsList renders a list of logs
func (r *UIRenderer) RenderLogsList(logs []string) string {
	return r.current.RenderLogsList(logs)
}

// RenderLog renders a single log
func (r *UIRenderer) RenderLog(log Log) string {
	return r.current.RenderLog(log)
}

// RenderTags renders tags
func (r *UIRenderer) RenderTags(tags map[string]int) string {
	return r.current.RenderTags(tags)
}

// CurrentName returns the current UI renderer's name
func (r *UIRenderer) CurrentName() string {
	return r.current.Name()
}

// ListAvailableUIs returns a list of available UI prototypes
func ListAvailableUIs(uiMap map[string]UI) string {
	if len(uiMap) == 0 {
		return "No UI prototypes available"
	}

	result := "Available UI Prototypes:\n"
	for name := range uiMap {
		result += fmt.Sprintf("  - %s\n", name)
	}
	return result
}

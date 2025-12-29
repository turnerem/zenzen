package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/turnerem/zenzen/core"
)

// EntryResponse represents an entry in API responses
type EntryResponse struct {
	ID                    string        `json:"id"`
	Title                 string        `json:"title"`
	Tags                  []string      `json:"tags"`
	StartedAt             string        `json:"started_at"`
	EndedAt               string        `json:"ended_at,omitempty"`
	LastModified          string        `json:"last_modified"`
	EstimatedDuration     string        `json:"estimated_duration,omitempty"`
	ActualDuration        string        `json:"actual_duration,omitempty"`
	Body                  string        `json:"body"`
	InProgress            bool          `json:"in_progress"`
	EstimationBias        string        `json:"estimation_bias,omitempty"`
}

// EntriesResponse represents a list of entries
type EntriesResponse struct {
	Entries []EntryResponse `json:"entries"`
	Total   int             `json:"total"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// handleHealth handles GET /health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	}
	writeJSON(w, http.StatusOK, response)
}

// handleGetEntries handles GET /api/v1/entries
func (s *Server) handleGetEntries(w http.ResponseWriter, r *http.Request) {
	entries, err := s.store.GetAll()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch entries", err.Error())
		return
	}

	// Convert to response format and sort by StartedAt (most recent first)
	entryList := make([]EntryResponse, 0, len(entries))
	for _, entry := range entries {
		entryList = append(entryList, toEntryResponse(entry))
	}

	// Sort by StartedAt timestamp, most recent first
	sort.Slice(entryList, func(i, j int) bool {
		// Parse timestamps for comparison
		timeI, errI := time.Parse(time.RFC3339, entryList[i].StartedAt)
		timeJ, errJ := time.Parse(time.RFC3339, entryList[j].StartedAt)

		if errI != nil || errJ != nil {
			return false
		}

		return timeI.After(timeJ)
	})

	response := EntriesResponse{
		Entries: entryList,
		Total:   len(entryList),
	}

	writeJSON(w, http.StatusOK, response)
}

// handleGetEntry handles GET /api/v1/entries/{id}
func (s *Server) handleGetEntry(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Missing entry ID", "")
		return
	}

	entries, err := s.store.GetAll()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch entry", err.Error())
		return
	}

	entry, exists := entries[id]
	if !exists {
		writeError(w, http.StatusNotFound, "Entry not found", "")
		return
	}

	response := toEntryResponse(entry)
	writeJSON(w, http.StatusOK, response)
}

// toEntryResponse converts core.Entry to EntryResponse
func toEntryResponse(entry core.Entry) EntryResponse {
	resp := EntryResponse{
		ID:           entry.ID,
		Title:        entry.Title,
		Tags:         entry.Tags,
		Body:         entry.Body,
		InProgress:   entry.InProgress(),
	}

	// Format timestamps
	if !entry.StartedAtTimestamp.IsZero() {
		resp.StartedAt = entry.StartedAtTimestamp.Format(time.RFC3339)
	}
	if !entry.EndedAtTimestamp.IsZero() {
		resp.EndedAt = entry.EndedAtTimestamp.Format(time.RFC3339)
	}
	if !entry.LastModifiedTimestamp.IsZero() {
		resp.LastModified = entry.LastModifiedTimestamp.Format(time.RFC3339)
	}

	// Format durations
	if entry.EstimatedDuration > 0 {
		resp.EstimatedDuration = formatDuration(entry.EstimatedDuration)
	}

	// Calculate actual duration if entry is complete
	if !entry.InProgress() && !entry.StartedAtTimestamp.IsZero() && !entry.EndedAtTimestamp.IsZero() {
		actualDuration := entry.EndedAtTimestamp.Sub(entry.StartedAtTimestamp)
		resp.ActualDuration = formatDuration(actualDuration)

		// Calculate estimation bias
		if entry.EstimatedDuration > 0 {
			bias, err := entry.EstimationBias()
			if err == nil {
				if bias > 0 {
					resp.EstimationBias = "over"
				} else if bias < 0 {
					resp.EstimationBias = "under"
				} else {
					resp.EstimationBias = "accurate"
				}
			}
		}
	}

	return resp
}

// formatDuration formats a duration in human-readable format
func formatDuration(d time.Duration) string {
	if d == 0 {
		return ""
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 && minutes > 0 {
		return formatWithUnits(hours, "h") + formatWithUnits(minutes, "m")
	} else if hours > 0 {
		return formatWithUnits(hours, "h")
	} else if minutes > 0 {
		return formatWithUnits(minutes, "m")
	}

	return formatWithUnits(int(d.Seconds()), "s")
}

func formatWithUnits(value int, unit string) string {
	if value == 0 {
		return ""
	}
	return fmt.Sprintf("%d%s", value, unit)
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, error string, message string) {
	response := ErrorResponse{
		Error:   error,
		Message: message,
	}
	writeJSON(w, status, response)
}

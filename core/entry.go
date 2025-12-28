package core

import (
	"time"
)

const (
	DAY  = 24 * time.Hour
	WEEK = 7 * DAY
)

// Split out metadata from note body?
type Entry struct {
	ID                    string        `json:"ID"`
	Title                 string        `json:"Title"`
	Tags                  []string      `json:"Tags"`
	StartedAtTimestamp    time.Time     `json:"StartAt"`
	EndedAtTimestamp      time.Time     `json:"End"`
	LastModifiedTimestamp time.Time     `json:"LastModified"`
	EstimatedDuration     time.Duration `json:"Estimated Duration"`
	Body                  string        `json:"Body"`
}

// FieldDisplayNames maps struct field names to human-readable display names
var FieldDisplayNames = map[string]string{
	"StartedAtTimestamp":    "started at",
	"EndedAtTimestamp":      "ended at",
	"LastModifiedTimestamp": "last modified",
}

func (l *Entry) InProgress() bool {
	return l.EndedAtTimestamp.IsZero()
}

// return difference between estimated and actual duration
// RFC3339 format: 2006-01-02T15:04:05Z
func (l *Entry) EstimationBias() (time.Duration, error) {
	if l.EndedAtTimestamp.IsZero() || l.StartedAtTimestamp.IsZero() {
		return 0, nil
	}
	actualDuration := l.EndedAtTimestamp.Sub(l.StartedAtTimestamp)
	return l.EstimatedDuration - actualDuration, nil
}

// func (d *Duration) Print() {
// 	dur :=
// }

func (l *Entry) SetDuration(weeks, days, hours time.Duration) {
	l.EstimatedDuration = weeks*WEEK + days*DAY + hours*time.Hour
}

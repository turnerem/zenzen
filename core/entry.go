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
	ID                int           `json:"ID"`
	Title             string        `json:"Title"`
	Tags              []string      `json:"Tags"`
	StartedAt         time.Time     `json:"StartAt"`
	EndedAt           time.Time     `json:"End"`
	EstimatedDuration time.Duration `json:"Estimated Duration"`
	Body              string        `json:"Body"`
}

func (l *Entry) InProgress() bool {
	return l.EndedAt.IsZero()
}

// return difference between estimated and actual duration
// RFC3339 format: 2006-01-02T15:04:05Z
func (l *Entry) EstimationBias() (time.Duration, error) {
	if l.EndedAt.IsZero() || l.StartedAt.IsZero() {
		return 0, nil
	}
	actualDuration := l.EndedAt.Sub(l.StartedAt)
	return l.EstimatedDuration - actualDuration, nil
}

// func (d *Duration) Print() {
// 	dur :=
// }

func (l *Entry) SetDuration(weeks, days, hours time.Duration) {
	l.EstimatedDuration = weeks*WEEK + days*DAY + hours*time.Hour
}

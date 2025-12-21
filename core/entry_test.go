package core

import (
	"testing"
	"time"
)

var (
	K8s = Entry{
		ID:                "1",
		Title:             "K8s",
		Tags:              []string{"learning", "open-source"},
		StartedAt:         time.Date(2025, 12, 20, 9, 0, 0, 0, time.UTC),
		EndedAt:           time.Date(2025, 12, 20, 11, 30, 0, 0, time.UTC),
		EstimatedDuration: 1*time.Hour + 30*time.Minute,
		Body:              "The journey has just begun.",
	}
	SystemDesign = Entry{
		ID:                "2",
		Title:             "System Design",
		Tags:              []string{"interviews"},
		StartedAt:         time.Date(2025, 05, 20, 10, 0, 0, 0, time.UTC),
		EndedAt:           time.Time{},
		EstimatedDuration: 3 * time.Hour,
		Body:              "Books combined with youtube resources were very helpful.",
	}
)

func TestInProgress(t *testing.T) {
	cases := []struct {
		name string
		log  Entry
		want bool
	}{
		{
			name: "finished",
			log:  K8s,
			want: false,
		},
		{
			name: "unfinished",
			log:  SystemDesign,
			want: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.log.InProgress()
			if got != c.want {
				t.Errorf("InProgress() = %v; want %v", got, c.want)
			}
		})
	}
}

func TestEstimationBias(t *testing.T) {
	cases := []struct {
		name    string
		log     Entry
		want    time.Duration
		wantErr bool
	}{
		{
			name: "finished log",
			log:  K8s,
			want: -time.Hour,
		},
		{
			name: "unfinished log",
			log:  SystemDesign,
			want: 0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := c.log.EstimationBias()
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
				return
			}
			if got != c.want {
				t.Errorf("ActualPredictedDiff() = %v, want %v", got, c.want)
			}
		})
	}
}

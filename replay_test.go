package main

import "testing"

func TestPending(t *testing.T) {
	tests := []struct {
		name   string
		events []Event
		want   int
		bad    bool
	}{
		{"outage gap", []Event{{"a", "asset-1", "ingested", true}, {"b", "asset-1", "processed", false}, {"c", "asset-1", "creator_ready", false}}, 2, false},
		{"already delivered", []Event{{"a", "asset-2", "creator_ready", true}}, 0, false},
		{"duplicate identifier", []Event{{"a", "asset-1", "ingested", false}, {"a", "asset-1", "processed", false}}, 0, true},
		{"unknown stage", []Event{{"a", "asset-1", "deleted", false}}, 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := pending(tc.events)
			if (err != nil) != tc.bad || (!tc.bad && len(got) != tc.want) {
				t.Fatalf("pending() = %v, %v; want count %d, error %v", got, err, tc.want, tc.bad)
			}
		})
	}
}

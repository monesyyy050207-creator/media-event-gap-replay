package main

import "fmt"

type Event struct {
	ID        string `json:"id"`
	AssetID   string `json:"asset_id"`
	Stage     string `json:"stage"`
	Delivered bool   `json:"delivered"`
}

func pending(events []Event) ([]Event, error) {
	seen := make(map[string]bool)
	var out []Event
	for _, event := range events {
		if event.ID == "" || event.AssetID == "" {
			return nil, fmt.Errorf("event id and asset_id are required")
		}
		switch event.Stage {
		case "ingested", "processed", "creator_ready":
		default:
			return nil, fmt.Errorf("invalid media stage %q", event.Stage)
		}
		if seen[event.ID] {
			return nil, fmt.Errorf("duplicate event id %q", event.ID)
		}
		seen[event.ID] = true
		if !event.Delivered {
			out = append(out, event)
		}
	}
	return out, nil
}

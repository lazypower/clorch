package state

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
)

// TimelineEvent represents a single event in a session's timeline.
type TimelineEvent struct {
	Time    string `json:"t"`
	Event   string `json:"e"`
	Summary string `json:"s"`
}

// ReadEvents reads the event log for a session, returning the most recent maxEvents entries.
func ReadEvents(stateDir, sessionID string, maxEvents int) []TimelineEvent {
	path := filepath.Join(stateDir, sessionID+".events")
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var all []TimelineEvent
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var ev TimelineEvent
		if json.Unmarshal(scanner.Bytes(), &ev) == nil {
			all = append(all, ev)
		}
	}

	// Return tail
	if len(all) > maxEvents {
		return all[len(all)-maxEvents:]
	}
	return all
}

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

// ReadAllEvents reads and parses the entire event log for a session.
func ReadAllEvents(stateDir, sessionID string) []TimelineEvent {
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
	return all
}

// ReadEvents reads the event log for a session, returning the most recent maxEvents entries.
func ReadEvents(stateDir, sessionID string, maxEvents int) []TimelineEvent {
	tail, _ := ReadEventsTail(stateDir, sessionID, maxEvents)
	return tail
}

// ReadEventsTail returns the most recent maxEvents entries along with the total
// number of events in the log, so callers can show an accurate "N older" count
// without retaining the whole slice.
func ReadEventsTail(stateDir, sessionID string, maxEvents int) ([]TimelineEvent, int) {
	all := ReadAllEvents(stateDir, sessionID)
	total := len(all)
	if total > maxEvents {
		return all[total-maxEvents:], total
	}
	return all, total
}

package state

import "time"

const (
	StatusIdle              = "IDLE"
	StatusWorking           = "WORKING"
	StatusWaitingPermission = "WAITING_PERMISSION"
	StatusWaitingAnswer     = "WAITING_ANSWER"
	StatusError             = "ERROR"
)

// AgentState represents the current state of a Claude Code session.
type AgentState struct {
	SessionID           string   `json:"session_id"`
	Status              string   `json:"status"`
	CWD                 string   `json:"cwd"`
	ProjectName         string   `json:"project_name"`
	Model               string   `json:"model"`
	LastEvent           string   `json:"last_event"`
	LastEventTime       string   `json:"last_event_time"`
	LastTool            string   `json:"last_tool"`
	NotificationMessage *string  `json:"notification_message"`
	ToolRequestSummary  *string  `json:"tool_request_summary"`
	StartedAt           string   `json:"started_at"`
	ToolCount           int      `json:"tool_count"`
	ErrorCount          int      `json:"error_count"`
	SubagentCount       int      `json:"subagent_count"`
	CompactCount        int      `json:"compact_count"`
	LastCompactTime     string   `json:"last_compact_time"`
	TaskCompletedCount  int      `json:"task_completed_count"`
	ActivityHistory     []int    `json:"activity_history"`
	PID                 int      `json:"pid"`
	GitBranch           string   `json:"git_branch"`
	GitDirtyCount       int      `json:"git_dirty_count"`
	TmuxWindow          string   `json:"tmux_window"`
	TmuxPane            string   `json:"tmux_pane"`
	TmuxSession         string   `json:"tmux_session"`
	TmuxWindowIndex     string   `json:"tmux_window_index"`
	TermProgram         string           `json:"term_program"`
	RecentTools         []RecentToolCall `json:"recent_tools"`
	FilesModified       []string              `json:"files_modified,omitempty"`
	BranchedFrom        string                `json:"branched_from,omitempty"`
	BranchLabel         string                `json:"branch_label,omitempty"`
	Subagents           map[string]SubAgent   `json:"subagents,omitempty"`

	DisplayName   string        `json:"-"`
	StaleDuration time.Duration `json:"-"`
	StuckLoop     bool          `json:"-"`
}

// SubAgent tracks an individual subagent spawned by this session.
type SubAgent struct {
	AgentID   string `json:"agent_id"`
	AgentType string `json:"agent_type"`
	Status    string `json:"status"`
	StartedAt string `json:"started_at"`
}

// RunningSubagentCount returns the number of subagents currently running.
func (a AgentState) RunningSubagentCount() int {
	count := 0
	for _, sub := range a.Subagents {
		if sub.Status == "running" {
			count++
		}
	}
	return count
}

// RecentToolCall tracks a recent tool invocation for stuck-loop detection.
type RecentToolCall struct {
	Tool     string `json:"tool"`
	ArgsHash string `json:"args_hash"`
	Time     string `json:"time"`
}

// ActionItem represents a pending action requiring human attention.
type ActionItem struct {
	Agent  AgentState
	Letter string
}

// StatusSummary provides aggregate counts for the TUI header.
type StatusSummary struct {
	Total   int
	Working int
	Idle    int
	Waiting int
	Error   int
}

// ComputeSummary calculates a StatusSummary from a slice of agents.
func ComputeSummary(agents []AgentState) StatusSummary {
	s := StatusSummary{Total: len(agents)}
	for _, a := range agents {
		switch a.Status {
		case StatusWorking:
			s.Working++
		case StatusIdle:
			s.Idle++
		case StatusWaitingPermission, StatusWaitingAnswer:
			s.Waiting++
		case StatusError:
			s.Error++
		}
	}
	return s
}

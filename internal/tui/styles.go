package tui

import "github.com/charmbracelet/lipgloss"

// Nord palette
var (
	nordFg     = lipgloss.Color("#d8dee9")
	nordGreen  = lipgloss.Color("#a3be8c")
	nordYellow = lipgloss.Color("#ebcb8b")
	nordRed    = lipgloss.Color("#bf616a")
	nordBlue   = lipgloss.Color("#81a1c1")
	nordCyan   = lipgloss.Color("#88c0d0")
	nordDimmed = lipgloss.Color("#4c566a")
	nordOrange = lipgloss.Color("#d08770")
)

var (
	titleStyle         = lipgloss.NewStyle().Bold(true).Foreground(nordCyan).Padding(0, 1)
	headerStyle        = lipgloss.NewStyle().Foreground(nordFg).Padding(0, 1)
	agentWorkingStyle  = lipgloss.NewStyle().Foreground(nordGreen)
	agentIdleStyle     = lipgloss.NewStyle().Foreground(nordDimmed)
	agentWaitingStyle  = lipgloss.NewStyle().Foreground(nordYellow).Bold(true)
	agentErrorStyle    = lipgloss.NewStyle().Foreground(nordRed).Bold(true)
	agentSelectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("#3b4252")).Padding(0, 1)
	actionLetterStyle  = lipgloss.NewStyle().Foreground(nordCyan).Bold(true)
	actionSummaryStyle = lipgloss.NewStyle().Foreground(nordFg)
	footerStyle        = lipgloss.NewStyle().Foreground(nordDimmed).Padding(0, 1)
	costStyle          = lipgloss.NewStyle().Foreground(nordOrange)
	sectionTitleStyle  = lipgloss.NewStyle().Foreground(nordBlue).Bold(true).Padding(0, 1)
	staleWarnStyle     = lipgloss.NewStyle().Foreground(nordYellow)
	staleCritStyle     = lipgloss.NewStyle().Foreground(nordRed)
	yoloActiveStyle    = lipgloss.NewStyle().Foreground(nordRed).Bold(true)
	sparkStyle         = lipgloss.NewStyle().Foreground(nordGreen)
)

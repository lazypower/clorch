package tui

import "github.com/charmbracelet/lipgloss"

// Nord palette. AdaptiveColor lets lipgloss pick the variant matching the
// terminal background: the Dark values are the original Nord hues (tuned for a
// dark background); the Light values are darkened/desaturated so they keep
// enough contrast on a white background.
var (
	nordFg     = lipgloss.AdaptiveColor{Light: "#2e3440", Dark: "#d8dee9"}
	nordGreen  = lipgloss.AdaptiveColor{Light: "#4f7a3a", Dark: "#a3be8c"}
	nordYellow = lipgloss.AdaptiveColor{Light: "#8a6a2b", Dark: "#ebcb8b"}
	nordRed    = lipgloss.AdaptiveColor{Light: "#b02a37", Dark: "#bf616a"}
	nordBlue   = lipgloss.AdaptiveColor{Light: "#3b5a78", Dark: "#81a1c1"}
	nordCyan   = lipgloss.AdaptiveColor{Light: "#2a7585", Dark: "#88c0d0"}
	nordDimmed = lipgloss.AdaptiveColor{Light: "#7b88a1", Dark: "#4c566a"}
	nordOrange = lipgloss.AdaptiveColor{Light: "#b5552f", Dark: "#d08770"}

	// selectedBg is the highlight bar behind the selected agent row.
	selectedBg = lipgloss.AdaptiveColor{Light: "#fdf3c0", Dark: "#3b4252"}
	// idleFade dims agents inactive for a while: it recedes toward the
	// background on either theme.
	idleFade = lipgloss.AdaptiveColor{Light: "#aab2c0", Dark: "#3b4252"}
	// footerFg is plain black for the bottom status bar on a light theme, but
	// keeps the original dimmed Nord gray on a dark theme.
	footerFg = lipgloss.AdaptiveColor{Light: "#000000", Dark: "#4c566a"}
)

var (
	titleStyle         = lipgloss.NewStyle().Bold(true).Foreground(nordCyan).Padding(0, 1)
	headerStyle        = lipgloss.NewStyle().Foreground(nordFg).Padding(0, 1)
	agentWorkingStyle  = lipgloss.NewStyle().Foreground(nordGreen)
	agentIdleStyle     = lipgloss.NewStyle().Foreground(nordDimmed)
	agentWaitingStyle  = lipgloss.NewStyle().Foreground(nordYellow).Bold(true)
	agentErrorStyle    = lipgloss.NewStyle().Foreground(nordRed).Bold(true)
	agentSelectedStyle = lipgloss.NewStyle().Background(selectedBg).Padding(0, 1)
	actionLetterStyle  = lipgloss.NewStyle().Foreground(nordCyan).Bold(true)
	actionSummaryStyle = lipgloss.NewStyle().Foreground(nordFg)
	footerStyle        = lipgloss.NewStyle().Foreground(footerFg).Padding(0, 1)
	costStyle          = lipgloss.NewStyle().Foreground(nordOrange)
	sectionTitleStyle  = lipgloss.NewStyle().Foreground(nordBlue).Bold(true).Padding(0, 1)
	staleWarnStyle     = lipgloss.NewStyle().Foreground(nordYellow)
	staleCritStyle     = lipgloss.NewStyle().Foreground(nordRed)
	yoloActiveStyle    = lipgloss.NewStyle().Foreground(nordRed).Bold(true)
	sparkStyle         = lipgloss.NewStyle().Foreground(nordGreen)
	idleFadeStyle      = lipgloss.NewStyle().Foreground(idleFade)
	stuckLoopStyle     = lipgloss.NewStyle().Foreground(nordOrange).Bold(true)
	staleHookStyle     = lipgloss.NewStyle().Foreground(nordYellow)
)

package cli

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lazypower/clorch/internal/config"
	"github.com/lazypower/clorch/internal/hooks"
	"github.com/lazypower/clorch/internal/notify"
	"github.com/lazypower/clorch/internal/rules"
	"github.com/lazypower/clorch/internal/state"
	"github.com/lazypower/clorch/internal/tmux"
	tuipkg "github.com/lazypower/clorch/internal/tui"
	"github.com/lazypower/clorch/internal/usage"
	"github.com/spf13/cobra"
)

var version = "dev"

// SetVersion sets the version string from build flags.
func SetVersion(v string) {
	version = v
}

// Execute runs the root cobra command.
func Execute() {
	rootCmd := &cobra.Command{
		Use:   "clorch",
		Short: "Claude Code session orchestrator",
		RunE:  runDashboard,
	}

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Install hooks into ~/.claude/settings.json",
		RunE:  runInit,
	}
	initCmd.Flags().Bool("dry-run", false, "Preview changes without writing")

	uninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove clorch hooks from settings",
		RunE:  runUninstall,
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "One-line summary for scripting",
		RunE:  runStatus,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Table view of all agents",
		RunE:  runList,
	}

	widgetCmd := &cobra.Command{
		Use:   "tmux-widget",
		Short: "Output for tmux status-right",
		RunE:  runWidget,
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("clorch " + version)
		},
	}

	rootCmd.AddCommand(initCmd, uninstallCmd, statusCmd, listCmd, widgetCmd, versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runDashboard(cmd *cobra.Command, args []string) error {
	cfg := config.Load()

	mgr := state.NewManager(cfg.StateDir)
	if err := mgr.EnsureStateDir(); err != nil {
		return fmt.Errorf("creating state dir: %w", err)
	}

	rulesEngine, err := rules.NewEngine(cfg.RulesPath)
	if err != nil {
		return fmt.Errorf("loading rules: %w", err)
	}

	notifier := notify.NewNotifier()
	navigator := tmux.NewNavigator()

	model := tuipkg.NewModel(mgr, rulesEngine, notifier, navigator, version)
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Start watcher
	watcher := state.NewWatcher(mgr, p, cfg.PollMS)
	watcher.Start()
	defer watcher.Stop()

	// Start usage tracker
	parser := usage.NewParser(cfg.ProjectsDir())
	tracker := usage.NewTracker(parser, p)
	tracker.Start()
	defer tracker.Stop()

	// Start cleanup goroutine
	cleanup := state.NewCleanup(mgr)
	cleanup.Start()
	defer cleanup.Stop()

	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

func runInit(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	err := hooks.Install(cfg.StateDir, cfg.HooksDir, cfg.SettingsPath(), version, dryRun)
	if err != nil {
		return err
	}
	if !dryRun {
		fmt.Println("Hooks installed successfully.")
		fmt.Printf("  Scripts: %s\n", cfg.HooksDir)
		fmt.Printf("  Settings: %s\n", cfg.SettingsPath())
	}
	return nil
}

func runUninstall(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	if err := hooks.Uninstall(cfg.HooksDir, cfg.SettingsPath()); err != nil {
		return err
	}
	fmt.Println("Hooks uninstalled.")
	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	mgr := state.NewManager(cfg.StateDir)
	agents, summary, _ := mgr.Scan()
	_ = agents
	fmt.Printf("%d working, %d waiting, %d idle, %d error\n",
		summary.Working, summary.Waiting, summary.Idle, summary.Error)
	return nil
}

func runList(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	mgr := state.NewManager(cfg.StateDir)
	agents, _, _ := mgr.Scan()

	if len(agents) == 0 {
		fmt.Println("No active agents.")
		return nil
	}

	fmt.Printf("%-12s %-20s %-18s %-10s %-6s\n", "STATUS", "PROJECT", "BRANCH", "TOOLS", "PID")
	for _, a := range agents {
		name := a.ProjectName
		if a.DisplayName != "" {
			name = a.DisplayName
		}
		fmt.Printf("%-12s %-20s %-18s %-10d %-6d\n",
			a.Status, name, a.GitBranch, a.ToolCount, a.PID)
	}
	return nil
}

func runWidget(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	mgr := state.NewManager(cfg.StateDir)
	_, summary, _ := mgr.Scan()
	fmt.Print(tmux.Widget(summary))
	return nil
}

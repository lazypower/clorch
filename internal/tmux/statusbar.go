package tmux

import (
	"fmt"

	"github.com/lazypower/clorch/internal/state"
)

// Widget returns a tmux status-right formatted string with Nord palette colors.
func Widget(summary state.StatusSummary) string {
	return fmt.Sprintf("#[fg=#a3be8c]●%d #[fg=#ebcb8b]◉%d #[fg=#bf616a]✕%d",
		summary.Working, summary.Waiting, summary.Error)
}

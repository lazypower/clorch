package tmux

import "math/rand"

// WindowNames is a pool of fun names for tmux windows.
var WindowNames = []string{
	"cobra", "falcon", "phoenix", "dragon", "kraken",
	"hydra", "sphinx", "griffin", "titan", "raptor",
	"viper", "mantis", "panther", "jaguar", "osprey",
	"merlin", "raven", "condor", "hawk", "lynx",
	"wolf", "bear", "fox", "owl", "stag",
	"pike", "trout", "bass", "heron", "crane",
	"wren", "finch", "swift", "storm", "blaze",
	"frost",
}

// PickWindowName returns a random unused name from the pool.
// If all names are taken, returns a fallback like "win-N".
func PickWindowName(existing []string) string {
	used := make(map[string]bool, len(existing))
	for _, name := range existing {
		used[name] = true
	}

	var available []string
	for _, name := range WindowNames {
		if !used[name] {
			available = append(available, name)
		}
	}

	if len(available) == 0 {
		for i := 0; ; i++ {
			candidate := "win-" + itoa(i)
			if !used[candidate] {
				return candidate
			}
		}
	}

	return available[rand.Intn(len(available))]
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

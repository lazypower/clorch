package terminal

// Backend provides terminal-specific navigation capabilities.
type Backend interface {
	GetTTYMap() (map[string]string, error)
	ActivateTab(id string) error
	BringToFront() error
	CanResolveTabs() bool
}

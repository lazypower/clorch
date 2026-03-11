package state

import (
	"context"
	"time"
)

// Cleanup runs periodic stale state file cleanup in the background.
type Cleanup struct {
	manager *Manager
	cancel  context.CancelFunc
}

// NewCleanup creates a cleanup goroutine manager.
func NewCleanup(manager *Manager) *Cleanup {
	return &Cleanup{manager: manager}
}

// Start begins the background cleanup loop (every 60 seconds).
func (c *Cleanup) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	go c.loop(ctx)
}

// Stop terminates the background goroutine.
func (c *Cleanup) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
}

func (c *Cleanup) loop(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.manager.CleanupStale()
		}
	}
}

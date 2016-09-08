package main

import (
	"sync"
)

type ConfTrackerConfig struct {
}

type ConfTracker struct {
	config *ConfTrackerConfig

	shutdown     bool
	shutdownLock sync.Mutex
	shutdownCh   chan struct{}
}

/*
func NewConfTracker(config *ConfTrackerConfig) (*ConfTracker, error) {
}
*/

func (t *ConfTracker) Shutdown() error {
	t.shutdownLock.Lock()
	defer t.shutdownLock.Unlock()

	if t.shutdown {
		return nil
	}

	t.shutdown = true
	close(t.shutdownCh)
	return nil
}

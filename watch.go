package main

import (
	"fmt"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/watch"
	"sync"
)

type WatcherConfig struct {
	watchType string
	key       string
	host      string
}

type Watcher struct {
	config *WatcherConfig

	shutdown     bool
	shutdownLock sync.Mutex
	shutdownCh   chan struct{}

	eventCh chan interface{}
	plan    *watch.WatchPlan
}

func (w *Watcher) Shutdown() error {
	w.shutdownLock.Lock()
	defer w.shutdownLock.Unlock()

	if w.shutdown {
		return nil
	}

	w.shutdown = true
	w.plan.Stop()
	close(w.shutdownCh)

	return nil
}

func NewWatcher(config *WatcherConfig) (*Watcher, error) {

	params := make(map[string]interface{})

	params["type"] = config.watchType
	params[config.watchType] = config.key

	wp, err := watch.Parse(params)

	w := &Watcher{
		config:     config,
		shutdown:   false,
		shutdownCh: make(chan struct{}),
		eventCh:    make(chan interface{}),
		plan:       wp,
	}

	if err != nil {
		return nil, err
	}

	wp.Handler = func(idx uint64, data interface{}) {
		if data == nil {
			return
		}

		if w.config.watchType == "key" {
			v, ok := data.(*consulapi.KVPair)
			// TODO: better way?
			if !ok || v == nil {
				return
			} else {
				w.eventCh <- v
			}
		} else {
			v, ok := data.(consulapi.KVPairs)
			if !ok || v == nil {
				return
			} else {
				w.eventCh <- v
			}
		}

	}

	go func() {
		// wp.Run() is blocking
		if err := wp.Run(w.config.host); err != nil {
			fmt.Printf("Error quering Consul agent: %s", err)
			return
		}

		for {
			select {
			case <-w.shutdownCh:
				close(w.eventCh)
				return
			}
		}
	}()

	return w, nil
}

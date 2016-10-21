package main

import (
	"fmt"

	"sync"

	log "github.com/Sirupsen/logrus"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/watch"
)

// WatcherConfig contains configration for Watcher
type WatcherConfig struct {
	watchType string
	key       string
	host      string
}

// Watcher watches keyprefix
type Watcher struct {
	config *WatcherConfig

	shutdown     bool
	shutdownLock sync.Mutex
	shutdownCh   chan struct{}

	eventCh chan interface{}
	plan    *watch.WatchPlan
	log     *log.Entry
}

// Shutdown shutdowns watcher
func (w *Watcher) Shutdown() error {
	w.shutdownLock.Lock()
	defer w.shutdownLock.Unlock()

	if w.shutdown {
		return nil
	}
	log.Infof("Shutting down")
	w.shutdown = true

	w.plan.Stop()
	close(w.shutdownCh)
	return nil
}

// NewWatcher creates a new watcher
func NewWatcher(config *WatcherConfig) (*Watcher, error) {

	params := make(map[string]interface{})
	watchType := config.watchType

	prefix := fmt.Sprintf("fetcher[%s]", config.key)
	logEntry := configureLogger(prefix)

	if config.watchType == "prefix" {
		config.watchType = "keyprefix"
	}

	params["type"] = config.watchType
	params[watchType] = config.key

	wp, err := watch.Parse(params)
	if err != nil {
		return nil, err
	}

	log.Infof("Watcher starting...")
	w := &Watcher{
		config:     config,
		shutdown:   false,
		shutdownCh: make(chan struct{}),
		eventCh:    make(chan interface{}),
		plan:       wp,
		log:        logEntry,
	}

	wp.Handler = func(idx uint64, data interface{}) {
		if data == nil {
			return
		}
		//log.Printf("idx(%d)\n", idx)

		if w.config.watchType == "key" {
			v, ok := data.(*consulapi.KVPair)
			// TODO: what happens when key is deleted
			if !ok || v == nil {
				return
			}
			w.eventCh <- v
		} else { // keyprefix
			v, ok := data.(consulapi.KVPairs)
			//log.Printf("ok(%v) v(%v)\n", ok, v)
			if !ok {
				return
			}
			w.eventCh <- v
		}

	}

	go func() {
		// wp.Run() is blocking
		// after wp.Stop() is called, 'connection refused' error
		// will be silently supressed
		if err := wp.Run(w.config.host); err != nil {
			logEntry.Errorf("Error quering Consul agent: %s", err)
			return
		}
	}()

	go func() {
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

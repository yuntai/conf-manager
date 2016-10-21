package main

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	consulapi "github.com/hashicorp/consul/api"
)

// ConfTrackerConfig contains configuration for ConfTracker
type ConfTrackerConfig struct {
	keyPrefix  string
	consulAddr string
}

/* TODO: based on struct type? */

// AppConfEvent for configuration changes
type AppConfEvent struct {
	t int
	*AppConf
}

func (e AppConfEvent) String() string {
	var prefix string
	switch e.t {
	case appConfNew:
		prefix = "NEW"
	case appConfChanged:
		prefix = "CHANGED"
	case appConfRemoved:
		prefix = "REMOVED"
	default:
		panic("Unknown type")
	}
	return fmt.Sprintf("[%s] %v", prefix, *e.AppConf)
}

// AppConf helper structure for json parsing
type AppConf struct {
	ID     string
	Branch string
	Repo   string
	Rev    string
}

func (c *AppConf) String() string {
	return fmt.Sprintf("ID(%s) Branch(%s) Repo(%s) Rev(%s)", c.ID, c.Branch, c.Repo, c.Rev)
}

const (
	appConfNew = iota
	appConfChanged
	appConfRemoved
)

func (c *AppConf) isComplete() bool {
	ret := true
	elem := reflect.ValueOf(c).Elem()
	for i := 0; i < elem.NumField(); i++ {
		if elem.Field(i).String() == "" {
			ret = false
		}
	}
	return ret
}

// ConfTracker emits changes in app configuration
type ConfTracker struct {
	shutdown     bool
	shutdownLock sync.Mutex
	shutdownCh   chan struct{}
	C            chan interface{}
	appConfigs   map[string]*AppConf
	events       chan AppConfEvent
	newConfigApp map[string]bool
}

// NewConfTracker makes a new ConfTracker
func NewConfTracker(config *ConfTrackerConfig) (*ConfTracker, error) {
	watcherConfig := &WatcherConfig{watchType: "prefix", key: config.keyPrefix, host: config.consulAddr}
	watcher, err := NewWatcher(watcherConfig)
	if err != nil {
		return nil, err
	}

	tracker := &ConfTracker{
		shutdownCh: make(chan struct{}),

		appConfigs:   make(map[string]*AppConf),
		newConfigApp: make(map[string]bool),

		C:      watcher.eventCh,
		events: make(chan AppConfEvent),
	}

	go tracker.Run()

	return tracker, nil
}

// Run starts ConfTracker
func (t *ConfTracker) Run() {
	for {
		select {
		case <-t.shutdownCh:
			return
		case v := <-t.C:
			pairs, ok := v.(consulapi.KVPairs)
			if !ok {
				panic("invalid value from watcher")
			}
			t.emitConf(pairs, t.events)
			/*
				for _, p := range v {
					t.emitConf(p, t.confC)
					//fmt.Printf("got key(%s) v(%s)\n", p.Key, string(p.Value))
				}*/
		}
	}
}

// emtiConf process Consul's key value pairs
func (t *ConfTracker) emitConf(pairs consulapi.KVPairs, confChan chan AppConfEvent) {
	// use as a set
	tmp := make(map[string]bool)
	for _, pair := range pairs {
		appID, _ := parseKey(pair.Key)
		tmp[appID] = true
		t.emitConfPair(pair, confChan)
	}

	// removed app configs
	var appsRemoved []string
	for k := range t.appConfigs {
		_, ok := tmp[k]
		_, ok2 := t.newConfigApp[k]

		if !ok && ok2 {
			delete(t.newConfigApp, k)
			appsRemoved = append(appsRemoved, k)
		}
	}

	for _, id := range appsRemoved {
		e := AppConfEvent{
			t:       appConfRemoved,
			AppConf: &AppConf{ID: id},
		}
		//fmt.Printf("emitting evt(%#v)\n", e)
		confChan <- e
	}
}

// parseKey parses key to appId & field
func parseKey(key string) (appID string, field string) {
	// TODO: Fix - parse depends on key format
	// key = config/global/testapp/branch
	parts := strings.Split(key, "/")
	appID = parts[2]
	field = strings.Title(parts[3])
	if field == "Id" {
		field = "ID"
	}
	return
}

// emitConfPair process one KV pair
func (t *ConfTracker) emitConfPair(pair *consulapi.KVPair, confChan chan AppConfEvent) {
	appID, field := parseKey(pair.Key)
	val := string(pair.Value)

	appConf, ok := t.appConfigs[appID]

	if !ok {
		_, ok2 := t.newConfigApp[appID]
		if !ok2 {
			t.newConfigApp[appID] = false
		}
		t.appConfigs[appID] = &AppConf{ID: appID}
	}
	appConf = t.appConfigs[appID]

	changed := false

	fld := reflect.ValueOf(appConf).Elem().FieldByName(field)

	if fld.String() != val {
		changed = true
		fld.SetString(val)
	}

	if appConf.isComplete() && changed {
		evt := AppConfEvent{
			t:       appConfChanged,
			AppConf: appConf,
		}
		if !t.newConfigApp[appID] {
			evt.t = appConfNew
			t.newConfigApp[appID] = true
		}
		//fmt.Printf("emitting evt(%s)\n", evt)
		confChan <- evt
	}
}

// Shutdown shutdown global configuration tracker
func (t *ConfTracker) Shutdown() error {
	t.shutdownLock.Lock()
	defer t.shutdownLock.Unlock()

	if t.shutdown {
		return nil
	}
	t.shutdown = true

	close(t.shutdownCh)
	fmt.Printf("Closing conf channel\n")
	close(t.events)
	return nil
}

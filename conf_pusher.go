package main

import (
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"

	consulapi "github.com/hashicorp/consul/api"
)

// ConfChange contains KV changes
type ConfChange struct {
	appID string
	kvs   *map[string][]byte
}

// ConfPusherConfig contains Puser configuration
type ConfPusherConfig struct {
	kv        *consulapi.KV
	keyPrefix string
}

// ConfPusher pushes configuration changes to Consul KV storage
type ConfPusher struct {
	shutdown     bool
	shutdownLock sync.Mutex
	shutdownCh   chan struct{}

	config    *ConfPusherConfig
	changes   chan *ConfChange
	logger    *log.Entry
	kv        *consulapi.KV
	keyPrefix string
}

// NewConfPusher creates a new conf pusher
func NewConfPusher(conf *ConfPusherConfig) *ConfPusher {
	logger := configureLogger("pusher")

	prefix := conf.keyPrefix
	if prefix == "" {
		prefix = "appconfigs"
	}
	conf.keyPrefix = prefix

	f := &ConfPusher{
		shutdownCh: make(chan struct{}),
		config:     conf,
		// Not sure how much it would help when parallelizeing updating consul KV store
		// for now, use one thread with buffered channel
		changes:   make(chan *ConfChange, 5),
		logger:    logger,
		kv:        conf.kv,
		keyPrefix: conf.keyPrefix,
	}
	return f
}

// KVUpdate update kv storage
// use tranaction feature(https://www.consul.io/docs/agent/http/kv.html#txn)
func (p *ConfPusher) KVUpdate(change *ConfChange) error {
	ops := consulapi.KVTxnOps{}
	prefix := p.keyPrefix + "/" + change.appID

	// Remove whole prefix tree
	// TODO: Perf using cache or diff?
	ops = append(ops, &consulapi.KVTxnOp{
		Verb: string(consulapi.KVDeleteTree),
		Key:  prefix,
	})

	// append updates
	for k, v := range *change.kvs {
		key := prefix + "/" + k
		p.logger.Infof("[INFO] pushing k(%s) v(%s)\n", key, strings.TrimSpace(string(v)))
		op := &consulapi.KVTxnOp{
			Verb:  string(consulapi.KVSet),
			Key:   key,
			Value: []byte(v),
		}
		ops = append(ops, op)
	}

	p.logger.Infof("Txn len(%d) ops", len(ops))
	ok, response, _, err := p.kv.Txn(ops, nil)
	if err != nil {
		p.logger.Printf("ok(%v) response(%v)\n", ok, response)
		p.logger.Printf("Failed to update KV stroage: %v\n", err)
		return err
	}

	/*
		for _, kp := range response.Results {
			p.logger.Printf("Updated k(%s) value(%s)\n", kp.Key, string(kp.Value))
		}
	*/

	if !ok {
		panic("Failed to update KV Storage")
	}
	return nil
}

// Run starts ConfPusher
func (p *ConfPusher) Run() {
	go p.Loop()
}

// Loop is internal loop for ConfPusher
func (p *ConfPusher) Loop() {
Loop:
	for {
		select {
		case evt, ok := <-p.changes:
			if !ok { // f.events closed
				p.changes = nil
				continue
			}
			//TODO: error handling
			p.KVUpdate(evt)
		case _, ok := <-p.shutdownCh:
			if !ok {
				p.shutdownCh = nil // f.done closed
			}
			break Loop
		}
		if p.changes == nil && p.shutdownCh == nil {
			break Loop
		}
	}
}

// Shutdown shutdown ConfPusher
func (p *ConfPusher) Shutdown() {
	p.shutdownLock.Lock()
	defer p.shutdownLock.Unlock()

	if p.shutdown {
		return
	}
	p.shutdown = true

	close(p.shutdownCh)
	return
}

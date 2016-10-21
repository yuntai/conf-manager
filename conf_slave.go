package main

import (
	"time"

	consulapi "github.com/hashicorp/consul/api"
)

// SlaveConfig is configration for ConfSlave
type SlaveConfig struct {
	configKey       string
	monitorInterval int64
	gitRoot         string
}

// SlaveContext will be deprecated
type SlaveContext struct {
	config       *SlaveConfig
	consulClient *consulapi.Client
	kv           *consulapi.KV
	nodeName     string
	nodeType     string
}

func monitorCommit(context *SlaveContext) error {
	return nil
}

func slaveLoop(done chan struct{}, context *SlaveContext) {
	config := context.config
	monitorTicker := time.NewTicker(time.Millisecond * time.Duration(config.monitorInterval))

	context.consulClient.KV()

	go func() {
		for {
			select {
			case <-monitorTicker.C:
				monitorCommit(context)
			case <-done:
				return
			}
		}
	}()
}

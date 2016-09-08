package main

import (
	"time"

	consulapi "github.com/hashicorp/consul/api"
	git "github.com/yuntai/git2go"
)

type SlaveConfig struct {
	configKey       string
	monitorInterval int64
	gitRoot         string
}

type SlaveContext struct {
	config       *SlaveConfig
	consulClient *consulapi.Client
	kv           *consulapi.KV
	nodeName     string
	nodeType     string
	repos        []AppConfig
}

type AppConfig struct {
	repoName   string
	branchName string
	repo       *git.Repository
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

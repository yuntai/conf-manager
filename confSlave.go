package main

import (
	"flag"
	"fmt"
	consulapi "github.com/hashicorp/consul/api"
	"log"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"
)

type SlaveConfig struct {
	configKey       string
	monitorInterval int64
}

type SlaveContext struct {
	config   *SlaveConfig
	kv       *consulapi.KV
	nodeName string
}

func initializeSalve() *SlaveContext {

	var configKeyPrefix = flag.String("config key prefix", DEFAULT_CONFIG_KEY_PREFIX, "key prefix for config version")
	var monitorInterval = flag.Int64("monitor interval", DEFUALT_MONITOR_INTERVAL, "monitor interval in millisecond")
	var consulHost = flag.String("consul host", DEFAULT_CONSUL_HOST, "consul host")
	var localGitPathRoot = flag.String("local git path root", DEFAULT_LOCAL_GIT_PATH_ROOT, "local git path root")

	flag.Parse()

	if !flag.Parsed() {
		flag.PrintDefaults()
		os.Exit(1)
	}

	config := &SlaveConfig{
		configKey:       *configKeyPrefix,
		monitorInterval: *monitorInterval,
	}

	consulClient, err := GetConsulClient(*consulHost)
	if err != nil {
		log.Fatal(err)
	}
	nodeName, err := consulClient.Agent().NodeName()

	if err != nil {
		log.Fatal(err)
	}

	gitRoot := path.Join(*localGitPathRoot, nodeName)
	if err := os.MkdirAll(gitRoot, 0600); err != nil {
		panic(err)
	}

	return &SlaveContext{config: config, kv: consulClient.KV(), nodeName: nodeName}
}

func monitorCommit(context *SlaveContext) error {
	return nil
}

func slaveLoop(done chan struct{}, context *SlaveContext) {
	config := context.config
	monitorTicker := time.NewTicker(time.Millisecond * time.Duration(config.monitorInterval))

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

func run() {
	context := initializeSalve()

	done := make(chan struct{})
	var wg sync.WaitGroup

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	slaveLoop(done, context)
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer close(done)
		v := <-c
		//TODO: handle SIGHUP
		fmt.Printf("Get signal(%v)...\n", v)
	}()
	wg.Wait()
}

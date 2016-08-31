package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func parseParams() (interface{}, string) {
	var nodeType = flag.String("nodetype", "slave", "specify node type")
	var configKeyPrefix = flag.String("configkey", DEFAULT_CONFIG_KEY_PREFIX, "key prefix for config version")
	var consulHost = flag.String("consulhost", DEFAULT_CONSUL_HOST, "consul host")
	var updateInterval = flag.Int64("updateinterval", DEFAULT_UPDATE_INTERVAL, "update interval in millisecond")
	var monitorInterval = flag.Int64("monitorinterval", DEFUALT_MONITOR_INTERVAL, "monitor interval in millisecond")

	flag.Parse()

	if !flag.Parsed() {
		flag.PrintDefaults()
		os.Exit(1)
	}

	consulClient, err := GetConsulClient(*consulHost)
	if err != nil {
		log.Fatal(err)
	}

	nodeName, err := consulClient.Agent().NodeName()
	if err != nil {
		log.Fatal(err)
	}

	kv := consulClient.KV()

	if *nodeType == "master" {
		nodeName = nodeName + "-m"
		fmt.Printf("Initalizing master(%s)\n", nodeName)
		return &MasterContext{
			config: &MasterConfig{
				configKey:       *configKeyPrefix,
				updateInterval:  *updateInterval,
				monitorInterval: *monitorInterval,
			},
			kv:       kv,
			repos:    make(map[string]*Repo),
			nodeName: nodeName,
		}, "master"
	} else {
		fmt.Printf("Initalizing slave(%s)\n", nodeName)
		return &SlaveContext{
			config: &SlaveConfig{
				configKey:       *configKeyPrefix,
				monitorInterval: *monitorInterval,
			},
			kv:       kv,
			nodeName: nodeName,
		}, "slave"
	}
}

func main() {
	context, nodeType := parseParams()

	done := make(chan struct{})
	var wg sync.WaitGroup

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	if nodeType == "master" {
		masterContext := context.(*MasterContext)
		flushKV("", masterContext.kv)
		// add test file repo
		if err := AddFSRepo(masterContext, "/home/yuntai/git/testrepo", "master"); err != nil {
			log.Panic(err)
		}

		masterLoop(done, masterContext)
	} else {
		slaveContext := context.(*SlaveContext)
		slaveLoop(done, slaveContext)
	}

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

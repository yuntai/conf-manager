package main

// error handling/timeout/retry
// memcache backend
// config transportation type (consul, git pull?)
// git pull (packed file?)

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	git "gopkg.in/libgit2/git2go.v24"
)

type MasterConfig struct {
	configKey       string
	monitorInterval int64
	updateInterval  int64
}

type MasterContext struct {
	config   *MasterConfig
	kv       *consulapi.KV
	repos    map[string]*Repo
	nodeName string
}

type Repo struct {
	currentTip string          // latest commit
	name       string          // repo name
	branchName string          // branch name
	repo       *git.Repository // object cache
	//TODO: remember error
}

func AddFSRepo(context *MasterContext, pathName string, branchName string) error {
	repo, err := git.OpenRepository(pathName)
	if err != nil {
		return err
	}
	repoName := path.Base(pathName)
	context.repos["repoName/branchName"] = &Repo{"", repoName, branchName, repo}
	return nil
}

func parseParams() {
	var configKeyPrefix = flag.String("config key prefix", DEFAULT_CONFIG_KEY_PREFIX, "key prefix for config version")
	var updateInterval = flag.Int64("update interval", DEFAULT_UPDATE_INTERVAL, "update interval in millisecond")
	var monitorInterval = flag.Int64("monitor interval", DEFUALT_MONITOR_INTERVAL, "monitor interval in millisecond")
	var consulHost = flag.String("consul host", DEFAULT_CONSUL_HOST, "consul host")

	flag.Parse()

	if !flag.Parsed() {
		flag.PrintDefaults()
		os.Exit(1)
	}
}

func initializeMaster() *MasterContext {

	config := &MasterConfig{
		configKey:       *configKeyPrefix,
		updateInterval:  *updateInterval,
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

	kv := consulClient.KV()

	fmt.Printf("Initalizing node(%s)\n", nodeName)
	return &MasterContext{config: config, kv: kv, repos: make(map[string]*Repo), nodeName: nodeName}
}

func monitorWatch(context *MasterContext) {
	return
	config := context.config
	keys, m, err := context.kv.Keys(config.configKey, "/", nil)
	if err != nil {
		//TODO: error handling
		return
	}

	fmt.Printf("monitor key(%s) req(%v)\n", keys, m.RequestTime)
	// check for new key
	for _, k := range keys {
		// TODO: should be subkey
		fmt.Printf("key(%s)\n", k)
		// handle new key

		if _, ok := context.repos[k]; !ok {
			path := "dummy"
			repoName := "repoName"
			branchName := "branchName"
			repo, err := git.OpenRepository(path)
			if err != nil {
				// TODO: error handling
				context.repos[k] = &Repo{"", repoName, branchName, repo}
			} else {
				context.repos[k] = nil
			}
		}
	}
	//TODO: reverse check
}

func updateCommit(context *MasterContext) {
	config := context.config

	// TODO: maybe parallelize
	for _, repo := range context.repos {

		repoKey := strings.Join([]string{config.configKey, repo.name, repo.branchName, context.nodeName + "_master"}, "/")

		commit, err := getLastCommit(repo.repo, repo.branchName)
		if err != nil {
			panic(err)
		}

		fmt.Printf("update(%s) repoKey(%s) commit(%s)\n", context.nodeName, repoKey, commit)

		if commit != repo.currentTip {
			value := []byte(commit)

			// TODO: use user interface/OOP way?
			w, err := context.kv.Put(&consulapi.KVPair{Flags: FLAG0, Key: repoKey, Value: value}, nil)
			if err != nil {
				log.Fatal(err)
			}
			repo.currentTip = commit // cache
			fmt.Printf("Pushed key(%s) commit(%s) time(%v)\n", repoKey, commit, w.RequestTime)
		}
	}
}

func masterLoop(done chan struct{}, context *MasterContext) {
	config := context.config
	updateTicker := time.NewTicker(time.Millisecond * time.Duration(config.updateInterval))
	monitorTicker := time.NewTicker(time.Millisecond * time.Duration(config.monitorInterval))

	go func() {
		for {
			select {
			case <-updateTicker.C:
				monitorWatch(context)
			case <-monitorTicker.C:
				updateCommit(context)
			case <-done:
				return
			}
		}
	}()
}

func main() {
	context := initializeMaster()
	flushKV("", context.kv)

	// add test file repo
	if err := AddFSRepo(context, "/home/yuntai/git/testrepo", "master"); err != nil {
		log.Panic(err)
	}

	done := make(chan struct{})
	var wg sync.WaitGroup

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	masterLoop(done, context)
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer close(done)
		v := <-c
		//TODO: handle SIGHUP
		fmt.Printf("Get signal(%v)...\n", v)
	}()
	wg.Wait()

	/*
		repoName := "nomad"
		path := "/home/yuntai/git_pub/nomad"
		repo, err := git.OpenRepository(path)

		//branchName := "f-sort-summaries"
		branchName := "master"
		commit, err := getLastCommit(repo, branchName)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("branch(%s) commit(%s)\n", branchName, commit)
	*/
}

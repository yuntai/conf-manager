package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	git "gopkg.in/libgit2/git2go.v24"
)

func GetKVStorage(host string) (*consulapi.KV, error) {
	config := consulapi.DefaultConfig()
	config.Address = host + ":8500"

	if client, err := consulapi.NewClient(config); err != nil {
		return nil, err
	} else {
		return client.KV(), nil
	}
}

func getLastCommit(repo *git.Repository, branchName string) (string, error) {
	branch, err := repo.LookupBranch(branchName, git.BranchLocal)
	if err != nil {
		return "", err
	}
	//TODO: when branch need to be resolved?
	//ref, err := branch.Resolve()
	//if err != nil {
	//return nil, err
	//}
	currentTip, err := repo.LookupCommit(branch.Target())
	return currentTip.Id().String(), nil
}

type Config struct {
	configKey       string
	monitorKey      string
	monitorInterval int64
	updateInterval  int64
}

type Repo struct {
	currentTip string
	name       string
	branchName string
	repo       *git.Repository // object cache
	//TODO: remember error
}

type Context struct {
	config *Config
	kv     *consulapi.KV
	repos  map[string]*Repo
}

func initialize() *Context {

	const (
		DEFAULT_CONFIG_KEY_PREFIX       = "config/version"
		DEFAULT_CONFIG_WATCH_KEY_PREFIX = "config/watch"
		DEFAULT_UPDATE_INTERVAL         = 1000 // in millisecond
		DEFUALT_MONITOR_INTERVAL        = 3000
		DEFAULT_CONSUL_HOST             = "localhost"
	)

	var configKeyPrefix = flag.String("config key prefix", DEFAULT_CONFIG_KEY_PREFIX, "key prefix for config version")
	var configMonitorKeyPrefix = flag.String("config watch key prefix", DEFAULT_CONFIG_WATCH_KEY_PREFIX, "key prefix for watch")
	var updateInterval = flag.Int64("update interval", DEFAULT_UPDATE_INTERVAL, "update interval in millisecond")
	var monitorInterval = flag.Int64("update interval", DEFUALT_MONITOR_INTERVAL, "update interval in millisecond")
	var consulHost = flag.String("consul host", DEFAULT_CONSUL_HOST, "consul host")

	flag.Parse()

	if !flag.Parsed() {
		flag.PrintDefaults()
		os.Exit(1)
	}

	config := &Config{
		configKey:       *configKeyPrefix,
		monitorKey:      *configMonitorKeyPrefix,
		updateInterval:  *updateInterval,
		monitorInterval: *monitorInterval,
	}

	kv, err := GetKVStorage(*consulHost)
	if err != nil {
		log.Fatal(err)
	}

	return &Context{config: config, kv: kv}
}

func monitorWatch(context *Context) {
	keys, m, err := context.kv.Keys(context.config.monitorKey, "/", nil)
	if err != nil {
		//TODO: error handling
		return
	}

	fmt.Printf("req(%v)", m.RequestTime)
	// check for new key
	for _, k := range keys {
		// TODO: should be subkey
		fmt.Printf("key(%s)", k)
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

func updateCommits(context *Context) {
	config := context.config

	// TODO: parallelize
	for _, repo := range context.repos {

		key := strings.Join([]string{config.configKey, repo.name, repo.branchName}, "/")

		commit, err := getLastCommit(repo.repo, repo.branchName)

		value := []byte(commit)

		// TODO: user interface && OOP way
		w, err := context.kv.Put(&consulapi.KVPair{Flags: 31, Key: key, Value: value}, nil)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Pushing key(%s) commit(%s) time(%v)", key, commit, w.RequestTime)
	}
}

func loop(done chan struct{}, context *Context) {

	config := context.config
	updateTicker := time.NewTicker(time.Millisecond * time.Duration(config.updateInterval))
	monitorTicker := time.NewTicker(time.Millisecond * time.Duration(config.monitorInterval))

	go func() {
		for {
			select {
			case <-updateTicker.C:
				monitorWatch(context)
			case <-monitorTicker.C:
				updateCommits(context)
			case <-done:
				return
			}
		}
	}()
}

func main() {
	context := initialize()
	done := make(chan struct{})
	var wg sync.WaitGroup

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	loop(done, context)

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

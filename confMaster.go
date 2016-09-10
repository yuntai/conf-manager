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
	_ "time"

	consulapi "github.com/hashicorp/consul/api"
)

type ConfMasterConfig struct {
	globalConfigKeyPrefix string
	monitorInterval       int64
	updateInterval        int64
	consulHost            string
}

type ConfMaster struct {
	nodeName     string
	config       *ConfMasterConfig
	consulClient *consulapi.Client
	kv           *consulapi.KV
	watcher      *Watcher
}

func parseParams() (interface{}, string) {
	// TODO: ugly
	var nodeType = flag.String("nodetype", "slave", "specify node type")
	*nodeType = "master"

	/*
		var configKeyPrefix = flag.String("configkey", DEFAULT_CONFIG_KEY_PREFIX, "key prefix for config version")
		var consulHost = flag.String("consulhost", DEFAULT_CONSUL_HOST, "consul host")
		var updateInterval = flag.Int64("updateinterval", DEFAULT_UPDATE_INTERVAL, "update interval in millisecond")
		var monitorInterval = flag.Int64("monitorinterval", DEFUALT_MONITOR_INTERVAL, "monitor interval in millisecond")

		// slave only param
		var gitRoot = flag.String("gitroot", DEFAULT_LOCAL_GIT_PATH_ROOT, "local git path root")
	*/

	flag.Parse()

	if !flag.Parsed() {
		flag.PrintDefaults()
		os.Exit(1)
	}
	return nil, ""
}

func NewConfMaster(config *ConfMasterConfig) (*ConfMaster, error) {
	consulClient, err := GetConsulClient(config.consulHost)
	if err != nil {
		log.Fatal(err)
	}

	/*
		nodeName, err := consulClient.Agent().NodeName()
		if err != nil {
			log.Fatal(err)
		}
	*/

	kv := consulClient.KV()

	watcherConfig := &WatcherConfig{
		watchType: "prefix",
		key:       config.globalConfigKeyPrefix,
		host:      config.consulHost,
	}

	// global key watcher
	watcher, err := NewWatcher(watcherConfig)
	if err != nil {
		return nil, err
	}

	m := &ConfMaster{
		nodeName:     MASTER_NODE_NAME,
		config:       config,
		consulClient: consulClient,
		kv:           kv,
		watcher:      watcher,
	}

	/*
		updateTicker := time.NewTicker(time.Millisecond * time.Duration(config.updateInterval))
		monitorTicker := time.NewTicker(time.Millisecond * time.Duration(config.monitorInterval))
		globalKeyCh := watcher.eventCh
	*/

	/*
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
	*/

	return m, nil
}

func (c *ConfMaster) Run() {
	go func() {
		for {
			select {
			case v := <-c.watcher.eventCh:
				fmt.Printf("v(%#v)", v)
				//processGlobalKey(v)
				//case <-done:
				//	return
			}
		}
	}()
}

/*
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

func initialize(context *MasterContext) {
	fmt.Printf("Listing root key(%s)\n", context.config.configKey)
	pairs, meta, err := context.kv.List(context.config.configKey, nil)
	fmt.Printf("meta(%+v) err(%+v)\n", meta, err)

	for _, p := range pairs {
		fmt.Printf("key(%s) value(%s) flags(%d)", p.Key, p.Value, p.Flags)

		if path.Base(p.Key) == MASTER_NODE_NAME {
			switch p.Flags {
			case STABLE:
				if err := context.localRepo.sync(p.Value); err != nil {
					panic(err)
				}
			case TRANSITION:
				if err := context.localRepo.fallback(p.Value); err != nil {
					panic(err)
				}
			}
		}
	}

	/*
		fmt.Printf("keys(%v)\n", keys)

		for _, k := range pairs {
			base := path.Base(k)
			if base == MASTER_NODE_NAME {
				key := strings.Replace(k, context.config.configKey+"/", "", 1)
				s := strings.Split(key, "/")
				repoName, branchName := s[0], s[1]
				fmt.Printf("repo(%s) branch(%s)\n", repoName, branchName)

				pair, meta, err := context.kv.Get(key, nil)
			}
			//strings.Split(k, "/")
		}
}
*/

/*
func runMaster(done chan struct{}, context *MasterContext) {
	//flushKV("", context.kv)
	// add test file repo
	if err := AddFSRepo(context, "/home/yuntai/git/testrepo", "master"); err != nil {
		log.Panic(err)
	}

	initialize(context)
	masterLoop(done, context)
}

func fallbackCommit(context *MasterContext, repo *Repo, commit string) {
	repo.fallBack(commit)
}

func updateCommit(context *MasterContext) {
	config := context.config

	// TODO: maybe parallelize
	for _, conf := range context.appConfigs {

		localTip := conf.localRepo.getTip()

		globalTip := conf.globalRepo.getTip()

		res := CompareCommitTip(repo.globalRepo, localTip, globalTip)

		if res == 0 {
			continue
		} else if res < 0 {
			// fatal condition
		}

		nextCommit := repo.globalRepo.nextCommit(localTip)

		repo.localRepo.advance(nextCommit)

		w, err := context.kv.Put(&consulapi.KVPair{Flags: TRANSITION, Key: repo.key(), Value: nextCommit}, nil)

		if err != nil {
			log.Fatal(err)
		}
		// set timeout
		//SetTimeout()

		//repoKey := strings.Join([]string{config.configKey, repo.name, repo.branchName, context.nodeName}, "/")

		//commit, err := getLastCommit(repo.repo, repo.branchName)

		//if err != nil {
		//	panic(err)
		//}

		//fmt.Printf("update(%s) repoKey(%s) commit(%s)\n", context.nodeName, repoKey, commit)

		//if commit != repo.currentTip {
		//	value := []byte(commit)

		//	// TODO: use user interface/OOP way?
		//	w, err := context.kv.Put(&consulapi.KVPair{Flags: FLAG0, Key: repoKey, Value: value}, nil)
		//	if err != nil {
		//		log.Fatal(err)
		//	}
		//	repo.currentTip = commit // cache
		//	fmt.Printf("Pushed key(%s) commit(%s) time(%v)\n", repoKey, commit, w.RequestTime)
		//}
	}
}

func masterLoop(done chan struct{}, context *MasterContext) {
	config := context.config
}
*/

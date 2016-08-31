package main

// error handling/timeout/retry
// memcache backend
// config transportation type (consul, git pull?)
// git pull (packed file?)

import (
	"fmt"
	"log"
	"path"
	"strings"
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
	nodeName     string
	config       *MasterConfig
	consulClient *consulapi.Client
	kv           *consulapi.KV
	repos        map[string]*Repo
	nodeType     string
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

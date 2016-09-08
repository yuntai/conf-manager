package main

// error handling/timeout/retry
// memcache backend
// config transportation type (consul, git pull?)
// git pull (packed file?)

import (
	"fmt"
	"log"

	consulapi "github.com/hashicorp/consul/api"
	git "github.com/yuntai/git2go"
)

/*
func AddFSRepo(context *MasterContext, pathName string, branchName string) error {
	repo, err := git.OpenRepository(pathName)
	if err != nil {
		return err
	}
	repoName := path.Base(pathName)
	context.repos["repoName/branchName"] = &Repo{"", repoName, branchName, repo}
	return nil
}
*/

const (
	STABLE = iota
	TRANSITION
)

const (
	DEFAULT_CONFIG_KEY_PREFIX       = "config/revision"
	DEFAULT_CONFIG_WATCH_KEY_PREFIX = "config/watch"
	DEFAULT_UPDATE_INTERVAL         = 1000 // in millisecond
	DEFUALT_MONITOR_INTERVAL        = 3000
	DEFAULT_CONSUL_HOST             = "localhost"

	DEFAULT_LOCAL_GIT_PATH_ROOT = "/mnt/tmp/conf/gitroot"

	MASTER_NODE_NAME = "master"
)

func GetConsulClient(host string) (*consulapi.Client, error) {
	config := consulapi.DefaultConfig()
	config.Address = host + ":8500"

	if client, err := consulapi.NewClient(config); err != nil {
		return nil, err
	} else {
		return client, nil
	}
}

func flushKV(prefix string, kv *consulapi.KV) error {
	fmt.Printf("Flushing KV storage prefix(%s)\n", prefix)
	_, err := kv.DeleteTree(prefix, nil)
	if err != nil {
		fmt.Println("Failed to flush KV storage")
		log.Panic(err)
	}
	return err
}

func initGitRepo(path string) error {
	if repo, err := git.InitRepository(path, false); err != nil {
		return err
	} else {
		fmt.Printf("repo(%v)", repo)
		return nil
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

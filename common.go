package main

// error handling/timeout/retry
// memcache backend
// config transportation type (consul, git pull?)
// git pull (packed file?)

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/Sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"

	consulapi "github.com/hashicorp/consul/api"
	git "github.com/libgit2/git2go"

	testutil "bitbucket.org/cdnetworks/eos-conf/test"
)

var gitHTTPPort = 9000

func nextGitHTTPPort() int {
	defer func() {
		gitHTTPPort++
	}()
	return gitHTTPPort
}

// singleton logger
var logger = logrus.New()

// initialize logger
// TODO: use environment variable or cmd line args to configure log level
func init() {
	logger.Formatter = &prefixed.TextFormatter{
		ShortTimestamp:  false,
		TimestampFormat: "2006/01/02 15:04:05",
	}
	logger.Level = logrus.InfoLevel
	logger.Out = os.Stderr
}

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


const (
	STABLE = iota
	TRANSITION
)
*/

const (
	// DefaultGlobalConfigKeyPrefix is key prefix for global configuration
	DefaultGlobalConfigKeyPrefix = "config/global"
	// DefaultAppConfigKeyPrefix is key prefix for app configuration
	DefaultAppConfigKeyPrefix = "config/app"
	// DefaultConsulAddr specifies default consul host to contact
	DefaultConsulAddr = "localhost:8500"
	// DefaultCommitMonitorPeriod specifies montitor period in millisecond
	DefaultCommitMonitorPeriod = 3000
	// DefaultServiceKey for leader election
	DefaultServiceKey = "service/confmaster/leader"
	/*
		DefaultUpdateInterval     = 1000
		DefaultMonitorInterval    = 3000
		MasterNodeName    = "master"
	*/
)

// makeTempDir creates a temporary directory
func makeTempDir(t *testing.T) string {
	path, err := ioutil.TempDir("", "git2go")
	checkFatal(t, err)
	return path
}

// configureLogger configures a log.logger
func configureLogger(prefix string) *logrus.Entry {
	return logger.WithField("prefix", prefix)
}

// GetConsulClient allocaes a new consul client
func GetConsulClient(host string) (*consulapi.Client, error) {
	config := consulapi.DefaultConfig()
	config.Address = host + ":8500"

	client, err := consulapi.NewClient(config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// flushKV flushes KV storage rooted at prefix
func flushKV(prefix string, kv *consulapi.KV) error {
	fmt.Printf("Flushing KV storage prefix(%s)\n", prefix)
	_, err := kv.DeleteTree(prefix, nil)
	if err != nil {
		fmt.Println("Failed to flush KV storage")
		log.Panic(err)
	}
	return err
}

// initGitRepo initizlies a local git repository
func initGitRepo(path string) error {
	if _, err := git.InitRepository(path, false); err != nil {
		return err
	}
	return nil
}

// copied from libgit2/git_test.go
func checkFatal(t *testing.T, err error) {
	testutil.CheckFatal(t, err)
}

// getLastCommit get the latest commit from "branch"
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
	if err != nil {
		return "", nil
	}
	return currentTip.Id().String(), nil
}

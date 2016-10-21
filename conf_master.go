package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"

	lh "bitbucket.org/cdnetworks/eos-conf/leaderhandler"
	consulapi "github.com/hashicorp/consul/api"
)

// https://www.consul.io/docs/guides/leader-election.html

// MasterConfig - configuration for ConfMaster
type MasterConfig struct {
	tempPathRoot          string
	globalConfigKeyPrefix string
	appConfigKeyPrefix    string
	consulAddr            string
}

// ConfMaster is top-level module for configuration delivery for local cluster
type ConfMaster struct {
	config *MasterConfig

	pusher  *ConfPusher
	fetcher *ConfFetcher
	handler *lh.LeaderHandler

	consulClient *consulapi.Client

	logger     *log.Entry
	shutdownCh chan interface{}
	shutdown   bool
}

// makeConsulClient creates a new consul client with a given URL
func makeConsulClient(addr string) (*consulapi.Client, error) {
	conf := consulapi.DefaultConfig()
	conf.Address = addr
	client, err := consulapi.NewClient(conf)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// NewConfMaster creates a new ConfMaster
func NewConfMaster(config *MasterConfig) (*ConfMaster, error) {
	logEntry := configureLogger("master")

	globalConfigKeyPrefix := config.globalConfigKeyPrefix
	if globalConfigKeyPrefix == "" {
		globalConfigKeyPrefix = DefaultGlobalConfigKeyPrefix
	}

	consulAddr := config.consulAddr
	if consulAddr == "" {
		consulAddr = DefaultConsulAddr
	}

	tempPathRoot := config.tempPathRoot
	if tempPathRoot == "" {
		sum := md5.Sum([]byte(fmt.Sprintf("%d", time.Now().Nanosecond())))
		//TODO: timestamped directory
		sessionDir := hex.EncodeToString(sum[:])

		rootPath, err := ioutil.TempDir("", "confmaster")
		if err != nil {
			logEntry.Printf("Failed to create temporary directory\n")
			return nil, err
		}
		tempPathRoot = path.Join(rootPath, sessionDir)
	}
	logEntry.Infof("tempPathRoot(%s)", tempPathRoot)

	appConfigKeyPrefix := config.appConfigKeyPrefix
	if appConfigKeyPrefix == "" {
		appConfigKeyPrefix = DefaultAppConfigKeyPrefix
	}

	client, err := makeConsulClient(config.consulAddr)
	if err != nil {
		return nil, err
	}

	tracker, err := NewConfTracker(&ConfTrackerConfig{
		keyPrefix:  globalConfigKeyPrefix,
		consulAddr: consulAddr,
	})
	if err != nil {
		return nil, err
	}

	pusher := NewConfPusher(&ConfPusherConfig{
		kv:        client.KV(),
		keyPrefix: appConfigKeyPrefix,
	})

	handler, err := lh.NewLeaderHandler(&lh.Config{
		Logger:      logger,
		LeaderKey:   lh.DefaultLeaderKey,
		WatchPeriod: 1000,
		IsMaster:    true,
		Client:      client,
	})

	if err != nil {
		return nil, err
	}

	githttp := NewGitHTTPServer(tempPathRoot, nextGitHTTPPort())
	err = githttp.Run()
	if err != nil {
		logEntry.Errorf("Faield to start http(%+v)", githttp)
		return nil, err
	}
	logEntry.Infof("Git HTTP server started(%+v)", githttp)

	fetcher := NewConfFetcher(&ConfFetcherConfig{
		pathRoot:   tempPathRoot,
		done:       make(chan interface{}),
		events:     tracker.events,
		leaderC:    handler.LeaderCh(),
		changes:    pusher.changes,
		gitHTTPURL: githttp.url,
	})

	return &ConfMaster{
		config:       config,
		pusher:       pusher,
		fetcher:      fetcher,
		handler:      handler,
		consulClient: client,
		logger:       logEntry,
		shutdownCh:   make(chan interface{}),
	}, nil
}

// Run starts ConfMaster
func (m *ConfMaster) Run() {

	sigC := make(chan os.Signal, 2)
	signal.Notify(sigC, os.Interrupt, syscall.SIGTERM)

	m.pusher.Run()
	m.fetcher.Run()
	m.handler.Run()

	for {
		select {
		case sig := <-sigC:
			m.logger.Printf("===> Caught signal: %v\n", sig)
			go m.Shutdown()
		case <-m.shutdownCh:
			//TODO: cleanup sub components proper
			m.logger.Printf("Shutting down ConfMaster...\n")
			m.pusher.Shutdown()
			return
		}
	}
}

// Shutdown shutdowns ConfMaster
func (m *ConfMaster) Shutdown() error {
	if m.shutdown {
		return nil
	}
	m.shutdown = true
	close(m.shutdownCh)
	return nil
}

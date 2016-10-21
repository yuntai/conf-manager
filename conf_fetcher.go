package main

import (
	"os"
	"path"
	"strings"
	"time"

	lh "bitbucket.org/cdnetworks/eos-conf/leaderhandler"
	"github.com/Sirupsen/logrus"
)

// LatestCommit macro
const LatestCommit string = "latest"

// ConfFetcherConfig is
type ConfFetcherConfig struct {
	pathRoot      string
	done          chan interface{}
	events        chan AppConfEvent
	changes       chan *ConfChange
	monitorPeriod int
	leaderC       chan lh.LeaderEvent
	gitHTTPURL    string
}

// ConfFetcher get config from git
type ConfFetcher struct {
	config        *ConfFetcherConfig
	repos         map[string]*Repo
	done          chan interface{}
	events        chan AppConfEvent
	log           *logrus.Entry
	changes       chan *ConfChange
	monitorPeriod time.Duration
	leaderC       chan lh.LeaderEvent
	gitHTTPURL    string
}

// ConfEvent is used to deliver configuration changes event
type ConfEvent struct {
	evt        AppConfEvent
	isMaster   bool
	leaderNode string
}

// NewConfFetcher creates a new ConfFetcher
func NewConfFetcher(conf *ConfFetcherConfig) *ConfFetcher {

	monitorPeriod := conf.monitorPeriod
	if monitorPeriod == 0 {
		monitorPeriod = DefaultCommitMonitorPeriod
	}

	logEntry := configureLogger("fetcher")

	f := &ConfFetcher{
		config:        conf,
		repos:         make(map[string]*Repo),
		done:          conf.done,
		events:        conf.events,
		changes:       conf.changes,
		log:           logEntry,
		monitorPeriod: time.Duration(monitorPeriod) * time.Millisecond,
		leaderC:       conf.leaderC,
		gitHTTPURL:    conf.gitHTTPURL,
	}
	return f
}

// CreateLocalRepo clones a repo
func (f *ConfFetcher) CreateLocalRepo(event AppConfEvent) error {
	p := path.Join(f.config.pathRoot, event.ID)

	config := &RepoConfig{
		path:       p,
		remoteURL:  event.Repo,
		branchName: event.Branch,
	}

	repo, err := CloneRepo(config)
	if err != nil {
		f.log.Error("Failed clone repo")
		return err
	}
	f.repos[event.ID] = repo

	f.log.Infof("Clonning Repo ID(%s) url(%s) path(%s) branch(%s) rev(%s)",
		event.ID,
		event.Repo,
		p,
		event.Branch,
		event.Rev,
	)

	return nil
}

// RemoveLocalRepo removes a local repo
func (f *ConfFetcher) RemoveLocalRepo(repoID string) {
	f.log.Infof("RemoveLocalRepo id(%v)\n", repoID)
	r := f.repos[repoID]
	path := r.repo.Path()
	r.Close()
	delete(f.repos, repoID)
	os.RemoveAll(path)
}

func (f *ConfFetcher) processEvent(id string, confEvt ConfEvent, commitCached string) (string, error) {
	repo := f.repos[id]

	var commit string
	var err error

	evt := confEvt.evt

	f.log.Infof("processing evt(%d) ID(%s) branch(%s) rev(%s) repo(%s)", evt.t, evt.ID, evt.Branch, evt.Rev, repo.Path())

	// chaning branch
	if repo.BranchName() != evt.Branch {
		f.log.Infof("Changing branch from(%s) to(%s)\n", repo.BranchName(), evt.Branch)
		err = repo.SetBranch(evt.Branch)
		if err != nil {
			f.log.Errorf("Failed to change branch to (%s)\n", evt.Branch)
			return "", err
		}
	}

	// fetching repo
	err = repo.Fetch()
	if err != nil {
		return "", err
	}

	switch evt.Rev {
	case LatestCommit:
		commit, err = repo.GetLatestCommit()
		if err != nil {
			f.log.Printf("[Error] failed to latest commit from repo(%s)", repo.Path())
			return "", err
		}

		if commit == commitCached {
			return commit, nil
		}
	default:
		if evt.Rev[0] == 'v' {
			// TODO: add tag for fetch spec
			commit, err = repo.LookupTag(evt.Rev)
			if err != nil {
				f.log.Errorf("Failed to find tag(%s)", evt.Rev)
				return "", err
			}
		} else {
			commit = evt.Rev
		}
	}

	f.log.Infof("Snapshotting repo(%s)", evt.ID)

	snapshot, err := repo.GetSnapshot(commit)
	if err != nil {
		f.log.Errorf("Failed to get snapshot for commit(%s): %v", commit, err)
		return "", err
	}

	f.log.Infof("snapshot repo(%s) branch(%s) commit(%s)", evt.ID, repo.BranchName(), commit)

	// dump snapshot
	for k, v := range *snapshot {
		f.log.Infof("k(%s) v(%s)", k, strings.TrimSpace(string(v)))
	}

	// adding meta info
	metaKey := "_meta/"
	(*snapshot)[metaKey+"branch"] = []byte(evt.Branch)
	(*snapshot)[metaKey+"rev"] = []byte(evt.Rev)
	(*snapshot)[metaKey+"commit"] = []byte(commit)
	(*snapshot)[metaKey+"repo"] = []byte(f.gitHTTPURL + "/" + evt.ID)

	// push snapshot to Consul KV
	f.changes <- &ConfChange{
		appID: evt.ID,
		kvs:   snapshot,
	}

	return commit, nil
}

// Fetcher processes configuration changes
func (f *ConfFetcher) Fetcher(id string, events chan ConfEvent) error {
	var cachedEvent *ConfEvent
	var cachedCommit string
	var err error

	ticker := time.NewTicker(f.monitorPeriod)
	defer ticker.Stop()
Loop:
	for {
		select {
		case evt, ok := <-events:
			if !ok {
				// events channel closed?
				break Loop
			}

			cachedCommit, err = f.processEvent(id, evt, "")
			if err != nil {
				//TODO: fatal what to do?
				break Loo
			}
			cachedEvent = &evt
		case <-ticker.C:
			// replay event to force fetching latest for other operation should be no effect
			if cachedEvent != nil && cachedEvent.evt.Rev == "latest" {
				cachedCommit, err = f.processEvent(id, *cachedEvent, cachedCommit)
				if err != nil {
					//TODO: fatal what to do?
					break Loop
				}
			}
		}
	}
	f.log.Infof("Fetcher id(%s) terminating...", id)
	return err
}

// Loop contains a main processing loop
func (f *ConfFetcher) Loop() {
	mapa := make(map[string]chan ConfEvent)

	var isLeader bool
	var leaderNode string

Loop:
	for {
		select {
		case le, ok := <-f.leaderC:
			if !ok {
				f.leaderC = nil
				continue
			}
			isLeader = le.IsMaster
			leaderNode = le.LeaderNode

		case evt, ok := <-f.events:
			if !ok { // f.events closed
				f.events = nil
				continue
			}

			switch evt.t {
			case appConfNew:

				f.CreateLocalRepo(evt)
				events := make(chan ConfEvent)

				f.log.Infof("Creating channel for ID(%s)", evt.ID)
				mapa[evt.ID] = events

				go f.Fetcher(evt.ID, events)

				events <- ConfEvent{evt, isLeader, leaderNode}

			case appConfChanged:
				mapa[evt.ID] <- ConfEvent{evt, isLeader, leaderNode}

			case appConfRemoved:
				f.log.Info("Removing channel for ID(%s)", evt.ID)
				close(mapa[evt.ID])
				delete(mapa, evt.ID)

				//f.RemoveLocalRepo(evt.conf.ID)
			}
		case _, ok := <-f.done:
			if !ok {
				f.done = nil // f.done closed
			}

			go func(mapa map[string]chan ConfEvent) {
				for _, c := range mapa {
					close(c)
				}
			}(mapa)
			break Loop
		}

		if f.events == nil && f.done == nil {
			break Loop
		}
	}
}

// Run runs a main loop
func (f *ConfFetcher) Run() {
	go f.Loop()
}

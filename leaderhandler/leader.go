package leaderelection

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	consulapi "github.com/hashicorp/consul/api"
	"math/rand"
	"sync"
	"time"
)

const (
	DefaultLeaderKey = "service/confmaster/leader"
)

type LeaderEvent struct {
	LeaderNode string
	IsMaster   bool
}

type LeaderState uint32

// not sure why this is needed yet
const (
	Slave LeaderState = iota
	Master
)

func (s LeaderState) String() string {
	switch s {
	case Master:
		return "Master"
	case Slave:
		return "Slave"
	default:
		return "Unknown"
	}
}

type Config struct {
	Logger      *logrus.Logger
	LeaderKey   string
	WatchPeriod int // in millisecond
	IsMaster    bool
	Client      *consulapi.Client
}

type LeaderHandler struct {
	LeaderKey     string
	Client        *consulapi.Client
	log           *logrus.Entry
	NodeName      string // node name
	WatchPeriod   time.Duration
	IsMaster      bool // part of master group, slave otherwise
	Running       bool
	shutdownCh    chan struct{}
	shutdownWait  sync.WaitGroup
	currentLeader string // cache for event generation
	leaderCh      chan LeaderEvent
	state         LeaderState
}

// Not thread safe!!!
func NewLeaderHandler(config *Config) (*LeaderHandler, error) {

	name, err := GetNodeName(config.Client)
	if err != nil {
		return nil, err
	}

	var prefix string = fmt.Sprintf("LE %s", name)
	if config.IsMaster {
		prefix += "[M]"
	} else {
		prefix += "[S]"
	}
	logEntry := config.Logger.WithField("prefix", prefix)

	handler := &LeaderHandler{
		Client:       config.Client,
		NodeName:     name,
		LeaderKey:    config.LeaderKey,
		log:          logEntry,
		WatchPeriod:  time.Duration(config.WatchPeriod) * time.Millisecond,
		IsMaster:     config.IsMaster,
		shutdownCh:   make(chan struct{}),
		shutdownWait: sync.WaitGroup{},
		leaderCh:     make(chan LeaderEvent),
	}

	handler.state = Slave
	handler.currentLeader = ""

	return handler, nil
}

func GetNodeName(client *consulapi.Client) (string, error) {
	agent, err := client.Agent().Self()
	if err != nil {
		return "", err
	}

	name := agent["Config"]["NodeName"].(string)
	return name, nil
}

func (l *LeaderHandler) LeaderCh() chan LeaderEvent {
	return l.leaderCh
}

func (l *LeaderHandler) Cleanup() error {
	sessionID, err := l.GetSession()
	if err != nil {
		return err
	}

	_, err = l.Client.Session().Destroy(sessionID, nil)
	if err != nil {
		return err
	}

	l.log.Infof("node(%s) SessionID(%s) destroyed", l.NodeName, sessionID)
	return nil
}

/*
When a session is invalidated, it is destroyed and can no longer be used.

What happens to the associated locks depends on the behavior specified at creation time.
Consul supports a release and delete behavior.

The release behavior is the default if none is specified.
*/

func (l *LeaderHandler) GetSession() (string, error) {
	if !l.IsMaster {
		panic("Non-master doesn't need a session")
	}

	c := l.Client
	sessionName := l.LeaderKey

	// TODO: sessionID cache
	sessions, _, err := c.Session().List(nil)
	for _, s := range sessions {
		if s.Name == sessionName && s.Node == l.NodeName {
			return s.ID, nil
		}
	}

	sessionEntry := &consulapi.SessionEntry{Name: sessionName}
	sessionID, _, err := c.Session().Create(sessionEntry, nil)

	if err != nil {
		return "", nil
	}

	return sessionID, nil
}

func (l *LeaderHandler) IsLeader() (bool, error) {
	if !l.IsMaster {
		return false, nil
	}

	sessionID, err := l.GetSession()
	if err != nil {
		return false, err
	}

	kv, _, err := l.Client.KV().Get(l.LeaderKey, nil)
	if err != nil {
		return false, err
	}

	return kv != nil && l.NodeName == string(kv.Value) && sessionID == kv.Session, nil
}

func (l *LeaderHandler) StepDown() error {
	if !l.IsMaster {
		return nil
	}

	b, err := l.IsLeader()
	if err != nil {
		return err
	}

	if b {
		sessionID, err := l.GetSession()
		if err != nil {
			return err
		}

		key := &consulapi.KVPair{Key: l.LeaderKey, Value: []byte(l.NodeName), Session: sessionID}
		released, _, err := l.Client.KV().Release(key, nil)
		if !released || err != nil {
			l.log.Errorf("Failed to release leadership node(%s) sessionID(%s)", l.NodeName, sessionID)
			return err
		} else {
			l.log.Debugf("Released leadership node(%s) sessionID(%s)", l.NodeName, sessionID)
			time.Sleep(1 * time.Second)
		}
	}
	return nil
}

func (l *LeaderHandler) Shutdown() {
	//TODO: lock for safe shutdown
	if !l.Running {
		return
	}
	l.Running = false

	l.shutdownWait.Add(1)
	l.shutdownCh <- struct{}{}
	l.shutdownWait.Wait()

	close(l.leaderCh)

	if l.IsMaster {
		l.StepDown()
		l.Cleanup()
	}
}

func (l *LeaderHandler) Run() {
	go l.Loop()
}

//TODO: currently polling & cache
// should be changed to watch
func (l *LeaderHandler) Loop() {
	defer l.shutdownWait.Done()

	l.Running = true

	c := l.Client

	// randomize starting
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	time.Sleep(time.Millisecond * time.Duration((r.Int() % 1000)))

	ticker := time.NewTicker(l.WatchPeriod)
	defer ticker.Stop()

	for l.Running {
		select {
		case <-l.shutdownCh:
			return

		case <-ticker.C:
			// if no leader is found, participate in leader leection
			if l.IsMaster && l.currentLeader == "" {
				/*
					b, err := l.IsLeader()

					if err != nil {
						panic(err)
					}

					if !b {
				*/
				sessionID, err := l.GetSession()
				if err != nil {
					panic(err)
				}

				pair := &consulapi.KVPair{
					Key:     l.LeaderKey,
					Value:   []byte(l.NodeName),
					Session: sessionID,
				}

				acquired, _, err := c.KV().Acquire(pair, nil)
				if acquired {
					l.log.Debugf("Elected as leader node(%s) sessionID(%s)", l.NodeName, sessionID)
				} else {
					l.log.Debugf("Failed to acquire leadership")
				}
			}

			// master followers and client check current leader
			// TODO: should change to watch
			kv, _, err := c.KV().Get(l.LeaderKey, nil)
			if err != nil {
				//TODO: error handling
				panic(err)
			}

			if kv != nil && kv.Session != "" {
				newLeader := string(kv.Value)
				if newLeader != l.currentLeader {
					//var c EventCode
					if l.currentLeader == "" {
						l.log.Debugf("New Leadership (%s) found", newLeader)
					} else {
						// seems never reached here
						l.log.Debugf("Leadership Change (%s -> %s) found", l.currentLeader, newLeader)
					}
					l.currentLeader = newLeader
					if newLeader == l.NodeName {
						l.state = Master
						l.leaderCh <- LeaderEvent{newLeader, true}
					} else {
						l.state = Slave
						l.leaderCh <- LeaderEvent{newLeader, false}
					}
				} else { // leader == l.currentLeader
					if newLeader == l.NodeName {
						l.log.Debugf("Enjoying leadership....")
					}
				}
			} else { // leadership missing
				l.state = Slave
				if l.currentLeader != "" {
					if l.currentLeader == l.NodeName {
						l.leaderCh <- LeaderEvent{"", false}
					}
					l.currentLeader = ""
				} else {
					l.log.Debugf("Leader not yet elected")
				}
			}
		}
	}
}

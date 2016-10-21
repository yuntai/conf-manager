package leaderelection

import (
	"fmt"
	"testing"
	"time"

	testutil "bitbucket.org/cdnetworks/eos-conf/test"
	"github.com/Sirupsen/logrus"
	. "github.com/franela/goblin"
	consulapi "github.com/hashicorp/consul/api"
	consultestutil "github.com/hashicorp/consul/testutil"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

func WaitForLeader(t *testing.T, client *consulapi.Client) string {
	retries := 1000

	for retries > 0 {
		time.Sleep(10 * time.Millisecond)
		retries--

		leader, err := client.Status().Leader()
		testutil.CheckFatal(t, err)
		if leader != "" {
			return leader
		}
	}
	testutil.CheckFatal(t, fmt.Errorf("Failed to get leader"))
	return ""
}

func dumpLeaderEvents(handler *LeaderHandler) {
	c := handler.LeaderCh()
	for e := range c {
		if e.IsMaster {
			handler.log.Info("Elected as Leader")
		} else {
			handler.log.Infof("Following Leader(%s)", e.LeaderNode)
		}
	}
}

func GetClient(t *testing.T, svr *consultestutil.TestServer) *consulapi.Client {
	cli, err := consulapi.NewClient(&consulapi.Config{
		Address:    svr.HTTPAddr,
		HttpClient: svr.HttpClient,
	})
	testutil.CheckFatal(t, err)
	return cli
}

func setupClusters(t *testing.T, numServers int) ([]*consultestutil.TestServer, []*consulapi.Client) {
	var svrs []*consultestutil.TestServer
	var clis []*consulapi.Client

	for i := 0; i < numServers; i++ {
		var svr *consultestutil.TestServer

		if i == 0 {
			svr = consultestutil.NewTestServerConfig(t, func(c *consultestutil.TestServerConfig) {
				c.LogLevel = "INFO"
				c.Bootstrap = true
			})
		} else {
			svr = consultestutil.NewTestServerConfig(t, func(c *consultestutil.TestServerConfig) {
				c.LogLevel = "INFO"
				c.Bootstrap = false
			})
			svr.JoinLAN(svrs[0].LANAddr)
			svr.JoinLAN(svrs[0].LANAddr)
		}
		svrs = append(svrs, svr)
		clis = append(clis, GetClient(t, svr))
	}

	return svrs, clis
}

func getLeaderIndex(t *testing.T, handlers []*LeaderHandler) int {
	var leaderIx int = -1
	for i := 0; i < len(handlers); i++ {
		h := handlers[i]
		b, err := h.IsLeader()
		testutil.CheckFatal(t, err)
		if b {
			leaderIx = i
			break
		}
	}
	return leaderIx
}

func TestLeaderElection(t *testing.T) {
	const numNodes = 6
	const numServers = 3

	svrs, clis := setupClusters(t, numNodes)
	for i := 0; i < numNodes; i++ {
		defer svrs[i].Stop()
	}

	g := Goblin(t)
	g.Describe("Cluster Leader Election", func() {
		leader := WaitForLeader(t, clis[0])
		g.It("Should leader be elected ", func() {
			var b = leader == ""
			g.Assert(b).IsFalse()
		})
		g.It("Should match leader ", func() {
			for i := 1; i < numServers; i++ {
				l := WaitForLeader(t, clis[i])
				g.Assert(l).Equal(leader)
			}
		})
	})

	logger := getLogger()

	var handlers []*LeaderHandler
	for i := 0; i < numNodes; i++ {
		var err error
		handler, err := NewLeaderHandler(&Config{
			Logger:      logger,
			LeaderKey:   DefaultLeaderKey,
			WatchPeriod: 400,
			IsMaster:    i < numServers,
			Client:      clis[i],
		})
		testutil.CheckFatal(t, err)
		defer handler.Shutdown()
		handlers = append(handlers, handler)

		handler.Run()

		go dumpLeaderEvents(handler)
	}

	time.Sleep(2 * time.Second)

	leaderIx := getLeaderIndex(t, handlers)
	g.Describe("App Leader Election", func() {
		g.It("leader should have been chosen", func() {
			b := leaderIx == -1
			g.Assert(b).IsFalse()
		})
	})

	time.Sleep(2 * time.Second)

	handlers[leaderIx].Shutdown()

	leaderIx2 := getLeaderIndex(t, handlers)
	g.Describe("App Leader Election", func() {
		g.It("leader should have been changed", func() {
			b := leaderIx == leaderIx2
			g.Assert(b).IsFalse()
		})
	})

	logger.Println("session list:")
	for i := 0; i < numNodes; i++ {
		if i != leaderIx {
			entries, _, err := clis[i].Session().List(nil)
			testutil.CheckFatal(t, err)
			for _, e := range entries {
				logger.Printf("  %v", e)
			}
			break
		}
	}

	time.Sleep(5 * time.Second)
}

func getLogger() *logrus.Logger {
	logger := logrus.New()
	logger.Level = logrus.InfoLevel
	f := new(prefixed.TextFormatter)
	f.TimestampFormat = "2006/01/02 15:04:05"
	logger.Formatter = f

	return logger
}

func TestLeaderHandler(t *testing.T) {
	logger := getLogger()
	client, _ := testutil.MakeClient(t)

	handler, err := NewLeaderHandler(&Config{
		Logger:      logger,
		LeaderKey:   DefaultLeaderKey,
		WatchPeriod: 100,
		IsMaster:    true,
		Client:      client,
	})
	testutil.CheckFatal(t, err)

	handler.Run()

	go dumpLeaderEvents(handler)

	time.Sleep(10 * time.Second)
	handler.Shutdown()
}

package test

import (
	"runtime"
	"testing"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
)

// makeClient is copied from consul/api/api_test.go
func MakeClient(t *testing.T) (*consulapi.Client, *testutil.TestServer) {
	return MakeClientWithConfig(t, func(clientConfig *consulapi.Config) {
	}, func(serverConfig *testutil.TestServerConfig) {
		serverConfig.LogLevel = "info"
		serverConfig.Bootstrap = true
	})
}

type ConfigCallback func(c *consulapi.Config)

// makeClientWithConfig is copied from consul/api/api_test.go
func MakeClientWithConfig(
	t *testing.T,
	cb1 ConfigCallback,
	cb2 testutil.ServerConfigCallback) (*consulapi.Client, *testutil.TestServer) {

	// Make client config
	conf := consulapi.DefaultConfig()
	if cb1 != nil {
		cb1(conf)
	}

	// Create server
	server := testutil.NewTestServerConfig(t, cb2)
	conf.Address = server.HTTPAddr

	// Create client
	client, err := consulapi.NewClient(conf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	return client, server
}

// copied from libgit2/git_test.go
func CheckFatal(t *testing.T, err error) {
	if err == nil {
		return
	}

	// The failure happens at wherever we were called, not here
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		t.Fatalf("Unable to get caller")
	}
	t.Fatalf("Fail at %v:%v; %v", file, line, err)
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"

	"io/ioutil"

	testutil "bitbucket.org/cdnetworks/eos-conf/test"
	"github.com/davecgh/go-spew/spew"
	consulapi "github.com/hashicorp/consul/api"
	consul "github.com/hashicorp/consul/consul"
	_ "github.com/hashicorp/consul/watch"
)

var testKey string
var testKeyprefix string

var nextPort = 15000

func getPort() int {
	p := nextPort
	nextPort++
	return p
}

func init() {
	testKey = "foo/bar/baz"
	testKeyprefix = "foo/bar/"
}

func startConsulTestServer(t *testing.T) (*consul.Server, string) {
	tempDir, err := ioutil.TempDir("", "consul")
	checkFatal(t, err)

	config := consul.DefaultConfig()

	addr := &net.TCPAddr{
		IP:   []byte{127, 0, 0, 1},
		Port: getPort(),
	}

	config.DataDir = tempDir
	config.DevMode = true
	config.RPCAddr = addr

	spew.Dump(config)

	s, err := consul.NewServer(config)
	if err != nil {
		t.Fatalf("failed to start a consul server: %v", err)
	}
	return s, addr.String()
}

func getConsulClient(t *testing.T, addr string) (*consulapi.Client, error) {
	config := consulapi.DefaultConfig()
	config.Address = addr

	client, err := consulapi.NewClient(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func makeParams(t *testing.T, s string) map[string]interface{} {
	var out map[string]interface{}
	dec := json.NewDecoder(bytes.NewReader([]byte(s)))
	if err := dec.Decode(&out); err != nil {
		t.Fatalf("err: %v", err)
	}
	return out
}

func TestKeyWatch(t *testing.T) {
	client, s := testutil.MakeClient(t)
	defer s.Stop()
	addr := s.HTTPAddr

	config := &WatcherConfig{
		watchType: "key",
		key:       testKey,
		host:      addr,
	}

	w, err := NewWatcher(config)
	if err != nil {
		t.Fatalf("Failed to get watcher: %s", err)
	}
	fmt.Printf("Got watcher (%#v)\n", w)

	go func() {
		defer w.Shutdown()
		time.Sleep(20 * time.Millisecond)

		kv := client.KV()

		for i := 0; i < 3; i++ {
			pair := &consulapi.KVPair{
				Key:   testKey,
				Value: []byte(fmt.Sprintf("testValue%d", i)),
			}

			_, err = kv.Put(pair, nil)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			//time.Sleep(3 * time.Millisecond)
		}

		//opts := consulapi.QueryOptions{WaitIndex: 0}
		//p, _, err := kv.Get(pair.Key, &opts)
		//t.Logf("pair(%#v)", p)

		// Wait for the query to run
		time.Sleep(20 * time.Millisecond)
		w.Shutdown()

		// Delete the key
		_, err = kv.Delete(testKey, nil)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	}()

	var i = 0
	for evt := range w.eventCh {
		e := (evt).(*consulapi.KVPair)
		expectedValue := fmt.Sprintf("testValue%d", i)
		t.Logf("Key(%s) Value(%s) Expected(%s)", e.Key, e.Value, expectedValue)
		if string(e.Value) != expectedValue {
			t.Errorf("Key(%s) Value(%s) Expected(%s)", e.Key, e.Value, expectedValue)
		}
		i++
	}

	// delay server shutdown
	time.Sleep(20 * time.Millisecond)
}

func TestKeyPrefixWatch(t *testing.T) {
	client, server := testutil.MakeClient(t)
	defer server.Stop()
	addr := server.HTTPAddr

	config := &WatcherConfig{
		watchType: "prefix",
		key:       testKeyprefix,
		host:      addr,
	}

	w, err := NewWatcher(config)
	if err != nil {
		t.Fatalf("Failed to get watcher: %s", err)
	}
	//t.Logf("Got watcher (%#v)\n", w)

	go func() {
		defer w.Shutdown()
		time.Sleep(20 * time.Millisecond)

		kv := client.KV()

		testKeyBases := [...]string{"baz", "test", "foo", "holy", "yahoo"}
		for _, b := range testKeyBases {
			pair := &consulapi.KVPair{
				Key:   testKeyprefix + b,
				Value: []byte(fmt.Sprintf("testValue=%s", b)),
			}
			_, err = kv.Put(pair, nil)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
		}

		// Wait for the query to run
		time.Sleep(20 * time.Millisecond)

		// Delete the key
		for _, b := range testKeyBases {
			key := testKeyprefix + b
			_, err = kv.Delete(key, nil)
			if err != nil {
				t.Fatalf("Failed to delete key(%s): %v", key, err)
			} else {
				fmt.Printf("Successfully deleted key(%s)\n", key)
			}
		}

		// wait for delete events being consumed
		// delay server shutdown
		time.Sleep(20 * time.Millisecond)
	}()

	for evt := range w.eventCh {
		pairs := (evt).(consulapi.KVPairs)
		for _, pair := range pairs {
			fmt.Printf("\tGot KV(%#v)\n", pair)
		}
	}
}

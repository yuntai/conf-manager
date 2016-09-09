package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	_ "github.com/hashicorp/consul/watch"
)

var consulAddr string
var testKey string
var testKeyprefix string

func init() {
	consulAddr = "localhost:8500"
	testKey = "foo/bar/baz"
	testKeyprefix = "foo/bar/"
}

func getConsulClient(t *testing.T, host string) (*consulapi.Client, error) {
	config := consulapi.DefaultConfig()
	config.Address = host + ":8500"

	if client, err := consulapi.NewClient(config); err != nil {
		return nil, err
	} else {
		return client, nil
	}
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
	if consulAddr == "" {
		t.Skip()
	}

	config := &WatcherConfig{
		watchType: "key",
		key:       testKey,
		host:      consulAddr,
	}

	w, err := NewWatcher(config)
	if err != nil {
		t.Fatalf("Failed to get watcher: %s", err)
	}
	fmt.Printf("Got watcher (%#v)\n", w)

	go func() {
		defer w.Shutdown()
		time.Sleep(20 * time.Millisecond)

		client, err := getConsulClient(t, "localhost")
		if err != nil {
			t.Fatalf("Failed to get Consul client: %s", err)
		}

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
		i += 1
	}
}

func TestKeyPrefixWatch(t *testing.T) {
	if consulAddr == "" {
		t.Skip()
	}

	config := &WatcherConfig{
		watchType: "keyprefix",
		key:       testKeyprefix,
		host:      consulAddr,
	}

	w, err := NewWatcher(config)
	if err != nil {
		t.Fatalf("Failed to get watcher: %s", err)
	}
	t.Logf("Got watcher (%#v)\n", w)

	go func() {
		defer w.Shutdown()
		time.Sleep(20 * time.Millisecond)

		client, err := getConsulClient(t, "localhost")
		if err != nil {
			t.Fatalf("Failed to get Consul client: %s", err)
		}

		kv := client.KV()

		testKeyBases := [...]string{"baz", "test"}
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
		w.Shutdown()

		// Delete the key
		for _, b := range testKeyBases {
			_, err = kv.Delete(testKeyprefix+b, nil)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
		}
	}()

	for evt := range w.eventCh {
		pairs := (evt).(consulapi.KVPairs)
		fmt.Printf("len(%d)\n", len(pairs))
		for _, pair := range pairs {
			fmt.Printf("pair(%#v)\n", pair)
		}
	}
}

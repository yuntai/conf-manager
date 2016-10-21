package main

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	TT "bitbucket.org/cdnetworks/eos-conf/test"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	_ "github.com/hashicorp/consul/watch"
)

var confTestKeyPrefix string

func init() {
	confTestKeyPrefix = "conf"
}

type Entry struct {
	ID     string `json:"id"`
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
	Rev    string `json:"rev"`
}

type Fixture struct {
	Entries []Entry `json:"entries"`
}

func generateFixture(t *testing.T) []Entry {
	fixtureJSON := `{"entries": [
{"ID":"web2048","Branch":"master","Repo":"repo0","Rev":"latest"},
{"ID":"web4096","Branch":"feature","Repo":"repo1","Rev":"latest"},
{"ID":"web8192","Branch":"topic/test","Repo":"repo2","Rev":"latest"}
]}`

	f := Fixture{}

	if err := json.Unmarshal([]byte(fixtureJSON), &f); err != nil {
		t.Fatalf("Failed to deocde json fixture %v", err)
	}

	return f.Entries
}

func generateConfKeys(t *testing.T, client *consulapi.Client, entries []Entry) {

	time.Sleep(20 * time.Millisecond)

	kv := client.KV()

	for _, entry := range entries {
		// TODO: ugly => refactor
		// set branch key
		key := confTestKeyPrefix + "/" + entry.ID + "/" + "branch"
		pair := &consulapi.KVPair{Key: key, Value: []byte(entry.Branch)}
		if _, err := kv.Put(pair, nil); err != nil {
			t.Fatalf("err: %v", err)
		}

		// set repo key
		key = confTestKeyPrefix + "/" + entry.ID + "/" + "repo"
		pair = &consulapi.KVPair{Key: key, Value: []byte(entry.Repo)}

		if _, err := kv.Put(pair, nil); err != nil {
			t.Fatalf("err: %v", err)
		}

		// set revision key
		key = confTestKeyPrefix + "/" + entry.ID + "/" + "rev"
		pair = &consulapi.KVPair{Key: key, Value: []byte(entry.Rev)}

		if _, err := kv.Put(pair, nil); err != nil {
			t.Fatalf("err: %v", err)
		}
	}
	time.Sleep(2000 * time.Millisecond)

	// change event
	for ix, entry := range entries {
		// TODO: ugly => refactor
		// set revision key
		// Test switch branch
		if entry.Branch != "master" {
			fmt.Printf("repo(%s) chainging branch from '%s' to 'master'\n", entry.ID, entry.Branch)
			entries[ix].Branch = "master"
			key := confTestKeyPrefix + "/" + entry.ID + "/" + "branch"
			pair := &consulapi.KVPair{Key: key, Value: []byte(entries[ix].Branch)}
			if _, err := kv.Put(pair, nil); err != nil {
				t.Fatalf("err: %v", err)
			}

			/*
				c := "99c080a722ce799e1577dcb5a601dbf91053175d"
				key = confTestKeyPrefix + "/" + entry.ID + "/" + "rev"
				pair = &consulapi.KVPair{Key: key, Value: []byte(c)}
				if _, err := kv.Put(pair, nil); err != nil {
					t.Fatalf("err: %v", err)
				}
			*/
		}

		/*
			if ix%2 == 0 {
				key := confTestKeyPrefix + "/" + entry.ID + "/" + "rev"
				pair := &consulapi.KVPair{Key: key, Value: []byte(entry.Rev + "-new")}
				if _, err := kv.Put(pair, nil); err != nil {
					t.Fatalf("err: %v", err)
				}
			}
		*/
	}
	time.Sleep(20 * time.Millisecond)

	if _, err := kv.DeleteTree(confTestKeyPrefix, nil); err != nil {
		t.Fatalf("err: %v", err)
	}
	time.Sleep(20 * time.Millisecond)
}

func makeTestTracker(t *testing.T) (*consulapi.Client, *testutil.TestServer, *ConfTracker) {
	client, server := TT.MakeClient(t)
	addr := server.HTTPAddr

	config := &ConfTrackerConfig{keyPrefix: confTestKeyPrefix, consulAddr: addr}
	tracker, err := NewConfTracker(config)
	if err != nil {
		checkFatal(t, err)
	}
	return client, server, tracker
}

func TestConfTracker(t *testing.T) {
	client, server, tracker := makeTestTracker(t)
	defer server.Stop()

	go func() {
		defer tracker.Shutdown()
		entries := generateFixture(t)
		generateConfKeys(t, client, entries)
		time.Sleep(20 * time.Millisecond)
	}()

	//TODO: real test
	/*
		for _, entry := range entries {
			// set branch key
			key := confTestKeyPrefix + "/" + entry.ID + "/" + "branch"
			pair := &consulapi.KVPair{Key: key, Value: []byte(entry.Branch)}
			evt := <-tracker.confC

			// set repo key
			key = confTestKeyPrefix + "/" + entry.ID + "/" + "repo"
			pair = &consulapi.KVPair{Key: key, Value: []byte(entry.Repo)}

			// set revision key
			key = confTestKeyPrefix + "/" + entry.ID + "/" + "rev"
			pair = &consulapi.KVPair{Key: key, Value: []byte(entry.Rev)}

		}
	*/

	for evt := range tracker.events {
		fmt.Printf("Got evt(%s)\n", evt)
	}

	// delay server shutdown
	time.Sleep(100 * time.Millisecond)
}

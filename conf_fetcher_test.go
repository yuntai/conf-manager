package main

import (
	"fmt"
	"io/ioutil"
	_ "os"
	"path"
	"testing"
	"time"

	_ "github.com/hashicorp/consul/watch"
	git "github.com/libgit2/git2go"
)

func refToBranch(branchName string) string {
	return fmt.Sprintf("refs/heads/%s", branchName)
}

// TODO: test code using Repo
func createTestReposWithContents(t *testing.T, entries []Entry) map[string]*git.Repository {

	ret := make(map[string]*git.Repository)

	for ix, entry := range entries {
		// figure out where we can create the test repo
		pathRoot, err := ioutil.TempDir("", "git2go")
		checkFatal(t, err)

		repo, err := git.InitRepository(pathRoot, false)
		checkFatal(t, err)

		tmpfile := "README"
		err = ioutil.WriteFile(path.Join(pathRoot, tmpfile), []byte("foo\n"), 0644)
		checkFatal(t, err)

		seedTestRepo(t, repo)

		updateReadme(t, repo, "HELLO1")

		/*
			for pathPart, v := range contents {
				filePath := path.Join(pathRoot, pathPart)
				d := path.Dir(filePath)
				if err := os.MkdirAll(d, 0777); err != nil {
					t.Fatalf("Failed to create directory(%s)\n", d)
				}
				if err = ioutil.WriteFile(filePath, []byte(v), 0644); err != nil {
					t.Fatalf("Failed to create file(%s)", filePath)
				}
			}
		*/

		_, oid := getHeadTip(t, repo)
		if entry.Branch != "master" {
			ref := refToBranch(entry.Branch)
			createBranch(t, repo, ref, oid)
			checkoutBranch(t, repo, ref)
			updateReadme(t, repo, "HELLO4")
		}

		// modify url path
		entries[ix].Repo = "file://" + repo.Path()
		fmt.Printf("Created a test repo ID(%s) path(%s) repo(%s) rev(%s)\n", entry.ID, repo.Path(), entry.Repo, entry.Rev)
	}

	return ret
}

func TestConfFetcherIntegrated(t *testing.T) {
	client, server, tracker := makeTestTracker(t)
	defer func() {
		server.Stop()
		tracker.Shutdown()
	}()

	// get test fixture
	entries := generateFixture(t)

	// create test global repos
	repos := createTestReposWithContents(t, entries)

	// change keys accordingly
	tempDir, err := ioutil.TempDir("", "confFetch")
	checkFatal(t, err)

	pusher := NewConfPusher(&ConfPusherConfig{
		kv: client.KV(),
	})

	fetcher := NewConfFetcher(&ConfFetcherConfig{
		pathRoot: tempDir,
		done:     make(chan interface{}),
		events:   tracker.events,
		changes:  pusher.changes,
	})

	go func() {
		generateConfKeys(t, client, entries)
	}()

	pusher.Run()
	fetcher.Run()

	// delay server shutdown
	time.Sleep(20000 * time.Millisecond)
	for _, r := range repos {
		r.Free()
	}
}

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"testing"
	"time"

	git "gopkg.in/libgit2/git2go.v24"
)

// copied from libgit2/git_test.go
func pathInRepo(repo *git.Repository, name string) string {
	return path.Join(path.Dir(path.Dir(repo.Path())), name)
}

// copied from libgit2/git_test.go
func updateReadme(t *testing.T, repo *git.Repository, content string) (*git.Oid, *git.Oid) {
	loc, err := time.LoadLocation("Europe/Berlin")
	checkFatal(t, err)
	sig := &git.Signature{
		Name:  "Rand Om Hacker",
		Email: "random@hacker.com",
		When:  time.Date(2013, 03, 06, 14, 30, 0, 0, loc),
	}

	tmpfile := "README"
	err = ioutil.WriteFile(pathInRepo(repo, tmpfile), []byte(content), 0644)
	checkFatal(t, err)

	idx, err := repo.Index()
	checkFatal(t, err)
	err = idx.AddByPath("README")
	checkFatal(t, err)
	treeId, err := idx.WriteTree()
	checkFatal(t, err)

	currentBranch, err := repo.Head()
	checkFatal(t, err)
	currentTip, err := repo.LookupCommit(currentBranch.Target())
	checkFatal(t, err)

	message := "This is a commit\n"
	tree, err := repo.LookupTree(treeId)
	checkFatal(t, err)
	commitId, err := repo.CreateCommit("HEAD", sig, sig, message, tree, currentTip)
	checkFatal(t, err)

	return commitId, treeId
}

// copied from libgit2/git_test.go
func seedTestRepo(t *testing.T, repo *git.Repository) (*git.Oid, *git.Oid) {
	loc, err := time.LoadLocation("Europe/Berlin")
	checkFatal(t, err)
	sig := &git.Signature{
		Name:  "Rand Om Hacker",
		Email: "random@hacker.com",
		When:  time.Date(2013, 03, 06, 14, 30, 0, 0, loc),
	}

	idx, err := repo.Index()
	checkFatal(t, err)
	err = idx.AddByPath("README")
	checkFatal(t, err)
	treeId, err := idx.WriteTree()
	checkFatal(t, err)

	message := "This is a commit\n"
	tree, err := repo.LookupTree(treeId)
	checkFatal(t, err)
	commitId, err := repo.CreateCommit("HEAD", sig, sig, message, tree)
	checkFatal(t, err)

	return commitId, treeId
}

// copied from libgit2/git_test.go
func cleanupTestRepo(t *testing.T, r *git.Repository) {
	var err error
	if r.IsBare() {
		err = os.RemoveAll(r.Path())
	} else {
		err = os.RemoveAll(r.Workdir())
	}
	checkFatal(t, err)

	r.Free()
}

// copied from libgit2/git_test.go
func createTestRepo(t *testing.T) *git.Repository {
	// figure out where we can create the test repo
	path, err := ioutil.TempDir("", "git2go")
	checkFatal(t, err)
	repo, err := git.InitRepository(path, false)
	checkFatal(t, err)

	tmpfile := "README"
	err = ioutil.WriteFile(path+"/"+tmpfile, []byte("foo\n"), 0644)

	checkFatal(t, err)

	return repo
}

// copied from libgit2/git_test.go
func checkFatal(t *testing.T, err error) {
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

func makeTestGit(t *testing.T) *git.Repository {
	repo := createTestRepo(t)
	seedTestRepo(t, repo)
	updateReadme(t, repo, "HELLO1")
	updateReadme(t, repo, "HELLO2")
	updateReadme(t, repo, "HELLO3")
	return repo
}

func makeTempDir(t *testing.T) string {
	path, err := ioutil.TempDir("", "git2go")
	checkFatal(t, err)
	return path
}

func TestRepoOpen(t *testing.T) {
	r := makeTestGit(t)
	repoPath := r.Path()
	_, err := OpenRepo(repoPath)
	if err != nil {
		t.Fatalf("Failed to open existing repo: %v", err)
	}
}

func TestRepoInit(t *testing.T) {
	path := makeTempDir(t)
	_, err := InitRepo(path)
	if err != nil {
		t.Fatalf("Failed to initialize repo")
	}
}

func TestRepoClone(t *testing.T) {
	r := makeTestGit(t)
	url := fmt.Sprintf("file://%s", r.Path())

	remoteConfig := &RepoRemoteConfig{
		url:    url,
		branch: "master",
	}

	path := makeTempDir(t)
	config := &RepoConfig{
		path:         path,
		remoteConfig: remoteConfig,
	}

	repo, err := CloneRepo(config)
	if err != nil {
		t.Fatalf("Failed to clone repo")
	}
	fmt.Printf("repo(%#v)", repo)
}

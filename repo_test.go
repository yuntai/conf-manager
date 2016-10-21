package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	. "github.com/franela/goblin"
	git "github.com/libgit2/git2go"
)

func dumpSnapshot(snapshot *map[string][]byte) {
	for k, v := range *snapshot {
		fmt.Printf("k(%s)\n%s\n", k, string(v))
	}
}

// fileUrl composes url for filesystem (file:///mnt/..)
func fileURL(path string) string {
	return fmt.Sprintf("file:///%s", path)
}

func checkoutBranch(t *testing.T, repo *git.Repository, branchRefName string) {
	headRef, err := repo.References.Lookup("HEAD")
	if err != nil {
		t.Fatalf("Failed to get HeadRef: %v", headRef)
	}
	defer headRef.Free()

	// switching to feature1 branch
	newHeadRef, err := headRef.SetSymbolicTarget(branchRefName, "")
	if err != nil {
		t.Fatalf("Failed to set symbolic to branch(%s)", branchRefName)
	}
	defer newHeadRef.Free()

	resolvedRef, err := newHeadRef.Resolve()
	if err != nil {
		fmt.Printf("failed to resolve new head\n")
	}
	defer resolvedRef.Free()
	if resolvedRef.Name() != branchRefName {
		t.Fatalf("Failed to checkout to [%s]", branchRefName)
	}

	checkoutOpts := &git.CheckoutOpts{
		Strategy: git.CheckoutForce,
	}
	err = repo.CheckoutHead(checkoutOpts)
	if err != nil {
		t.Fatalf("Failed to checkout")
	}
}

func createBranch(t *testing.T, repo *git.Repository, branchRefName string, oid *git.Oid) {
	// creating feature1 branch
	branchRef, err := repo.References.Create(branchRefName, oid, true, "")
	if err != nil {
		t.Fatalf("Failed to create branch ref: %s", branchRefName)
	}
	defer branchRef.Free()
}

func getHeadTip(t *testing.T, repo *git.Repository) (string, *git.Oid) {
	headRef, err := repo.References.Lookup("HEAD")
	if err != nil {
		t.Fatalf("Failed to get HeadRef: %v", headRef)
	}
	defer headRef.Free()

	ref, err := headRef.Resolve()
	if err != nil {
		fmt.Printf("Failed to resolve HEAD")
	}
	defer ref.Free()
	return ref.Name(), ref.Target()
}

func printHeadTip(t *testing.T, repo *git.Repository) (string, *git.Oid) {
	branchName, tip := getHeadTip(t, repo)
	fmt.Printf("branch(%s) commit(%s)\n", branchName, tip.String())
	return branchName, tip
}

// copied from libgit2/git_test.go
func pathInRepo(repo *git.Repository, name string) string {
	return path.Join(path.Dir(path.Dir(repo.Path())), name)
}

// copied from libgit2/git_test.go
func updateReadme(t *testing.T, repo *git.Repository, content string) (*git.Oid, *git.Oid) {
	return updateFile(t, repo, "README", content)
}

func updateFile(t *testing.T, repo *git.Repository, filePath string, content string) (*git.Oid, *git.Oid) {
	loc, err := time.LoadLocation("Asia/Seoul")
	checkFatal(t, err)
	sig := &git.Signature{
		Name:  "Rand Om Hacker",
		Email: "random@hacker.com",
		When:  time.Date(2013, 03, 06, 14, 30, 0, 0, loc),
	}

	err = ioutil.WriteFile(pathInRepo(repo, filePath), []byte(content), 0644)
	checkFatal(t, err)

	idx, err := repo.Index()
	checkFatal(t, err)
	err = idx.AddByPath(filePath)
	checkFatal(t, err)
	treeID, err := idx.WriteTree()
	checkFatal(t, err)

	currentBranch, err := repo.Head()
	checkFatal(t, err)
	currentTip, err := repo.LookupCommit(currentBranch.Target())
	checkFatal(t, err)

	message := "This is a commit\n"
	tree, err := repo.LookupTree(treeID)
	checkFatal(t, err)
	commitID, err := repo.CreateCommit("HEAD", sig, sig, message, tree, currentTip)
	checkFatal(t, err)

	fmt.Printf("updateFile commit(%s) tree(%s)\n", commitID, treeID)

	return commitID, treeID
}

// copied from libgit2/git_test.go
func seedTestRepo(t *testing.T, repo *git.Repository) (*git.Oid, *git.Oid) {
	loc, err := time.LoadLocation("Asia/Seoul")
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
	treeID, err := idx.WriteTree()
	checkFatal(t, err)

	message := "This is a commit\n"
	tree, err := repo.LookupTree(treeID)
	checkFatal(t, err)
	commitID, err := repo.CreateCommit("HEAD", sig, sig, message, tree)
	checkFatal(t, err)

	return commitID, treeID
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
func createTestRepo(t *testing.T, rootPath string) *git.Repository {
	// figure out where we can create the test repo
	path, err := ioutil.TempDir(rootPath, "git2go")
	checkFatal(t, err)

	repo, err := git.InitRepository(path, false)
	checkFatal(t, err)

	tmpfile := "README"
	err = ioutil.WriteFile(path+"/"+tmpfile, []byte("foo\n"), 0644)

	checkFatal(t, err)

	return repo
}

func makeTestGit(t *testing.T) *git.Repository {
	repo := createTestRepo(t, "")
	seedTestRepo(t, repo)
	updateReadme(t, repo, "HELLO1")
	updateReadme(t, repo, "HELLO2")
	updateReadme(t, repo, "HELLO3")
	return repo
}

func makeTestRepoWithBranch(t *testing.T, branchName string, rootPath string) *git.Repository {
	repo := createTestRepo(t, rootPath)

	seedTestRepo(t, repo)

	printHeadTip(t, repo)

	// modify files in master branch
	updateReadme(t, repo, "HELLO1")
	updateReadme(t, repo, "HELLO2")
	updateReadme(t, repo, "HELLO3")

	_, tip := printHeadTip(t, repo)

	branchRefName := fmt.Sprintf("refs/heads/%s", branchName)

	createBranch(t, repo, branchRefName, tip)
	checkoutBranch(t, repo, branchRefName)

	// modify files in a new branch
	updateReadme(t, repo, "HELLO4")
	updateReadme(t, repo, "HELLO5")
	updateReadme(t, repo, "HELLO6")

	printHeadTip(t, repo)

	//checkoutBranch(t, repo, "refs/heads/master")
	//printHeadTip(t, repo)

	return repo
}

func TestRepoOpen(t *testing.T) {
	g := Goblin(t)
	g.Describe("Repo Open", func() {
		g.It("OpenRepo should work ", func() {
			r := makeTestGit(t)
			repoPath := r.Path()
			_, err := OpenRepo(repoPath)
			if err != nil {
				t.Fatalf("Failed to open existing repo: %v", err)
			}
		})
	})
}

func TestRepoBranchChange(t *testing.T) {
	// emulate central git
	branchName := "feature1"
	rootPath, err := ioutil.TempDir("", "git2go")
	checkFatal(t, err)

	r := makeTestRepoWithBranch(t, branchName, rootPath)
	log := configureLogger("test")

	globalURL := fmt.Sprintf("file://%s", r.Path())
	config := &RepoConfig{
		path:       makeTempDir(t),
		remoteName: "global",
		remoteURL:  globalURL,
		branchName: branchName,
		appID:      "master",
	}

	// cluster master repo
	masterRepo, err := CloneRepo(config)
	if err != nil {
		t.Fatalf("Failed to clone repo")
	}
	log.Infof("repo(%s)", masterRepo)
	masterRepo.DumpRemotesAll()

	err = masterRepo.Fetch()
	checkFatal(t, err)

	commit, err := masterRepo.GetLatestCommit()
	checkFatal(t, err)
	log.Infof("master tip(%s)", commit)

	/*
		err = masterRepo.SetBranch("master")
		checkFatal(t, err)

		err = masterRepo.Fetch()
		checkFatal(t, err)
		commit, err = masterRepo.GetLatestCommit()
		checkFatal(t, err)
		log.Infof("master tip(%s)", commit)
	*/
}

func extractRepoName(p string) string {
	return path.Base(path.Dir(filepath.Clean(p)))
}

func TestRepoSlave(t *testing.T) {
	log := configureLogger("test")

	// emulate central git
	branchName := "feature1"
	rootPath, err := ioutil.TempDir("", "git2go")

	r := makeTestRepoWithBranch(t, branchName, rootPath)

	repoName := extractRepoName(r.Path())
	log.Infof("repo name(%s)", repoName)

	s := NewGitHTTPServer(rootPath, nextGitHTTPPort())
	err = s.Run()
	checkFatal(t, err)
	log.Infof("started git http server url(%s) path(%s)", s.url, s.path)

	globalURL := s.url + "/" + repoName
	log.Infof("globalURL(%s)", globalURL)

	config := &RepoConfig{
		path:       makeTempDir(t),
		remoteName: "global",
		remoteURL:  globalURL,
		branchName: branchName,
		appID:      repoName,
	}

	// cluster master repo
	masterRepo, err := CloneRepo(config)
	if err != nil {
		t.Fatalf("Failed to clone repo")
	}
	log.Infof("repo(%s)", masterRepo)
	masterRepo.DumpRemotesAll()

	err = masterRepo.Fetch()
	checkFatal(t, err)

	commit, err := masterRepo.GetLatestCommit()
	checkFatal(t, err)
	log.Infof("master tip(%s)", commit)

	slaveConfig := &RepoConfig{
		path:       makeTempDir(t),
		remoteURL:  "file://" + masterRepo.Path(),
		remoteName: "master",
		branchName: branchName,
		appID:      "slave",
	}
	slaveRepo, err := CloneRepo(slaveConfig)
	log.Infof("slave remotes")
	slaveRepo.DumpRemotesAll()

	if err != nil {
		t.Fatalf("Failed to clone maste repo")
	}
	log.Printf("repo(%s)", slaveRepo)

	slaveRepo.AddRemote("global", globalURL)
	log.Printf("Slave's remotes after AddRemote()")
	slaveRepo.DumpRemotesAll()

	log.Printf("slave remote(%s) branch(%s)", slaveRepo.RemoteName(), slaveRepo.BranchName())

	slaveRepo.SetRemote("global")

	log.Printf("slave remote(%s) branch(%s)", slaveRepo.RemoteName(), slaveRepo.BranchName())

	err = slaveRepo.Fetch()
	checkFatal(t, err)

	commit, err = slaveRepo.GetLatestCommit()
	checkFatal(t, err)

	fmt.Printf("slave tip(%s)\n", commit)
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

	path := makeTempDir(t)
	config := &RepoConfig{
		path:       path,
		remoteURL:  url,
		branchName: "master",
	}

	repo, err := CloneRepo(config)
	if err != nil {
		t.Fatalf("Failed to clone repo")
	}
	fmt.Printf("repo(%#v)\n", repo)

	commit, err := repo.GetLatestCommit()
	if err != nil {
		t.Fatalf("Failed to get tip: %v", err)
	}
	fmt.Printf("cur tip(%s)\n", commit)
}

func TestRepoSnapshot(t *testing.T) {
	r := makeTestGit(t)
	url := fmt.Sprintf("file://%s", r.Path())

	path := makeTempDir(t)
	config := &RepoConfig{
		path:       path,
		remoteURL:  url,
		branchName: "master",
	}

	repo, err := CloneRepo(config)
	if err != nil {
		t.Fatalf("Failed to clone repo")
	}
	fmt.Printf("repo(%#v)\n", repo)

	commit, err := repo.GetLatestCommit()
	if err != nil {
		t.Fatalf("Failed to get tip: %v", err)
	}
	repo.GetSnapshot("")

	fmt.Printf("cur tip(%s)\n", commit)
}

func TestRepoAddRemote(t *testing.T) {
	r := makeTestRepoWithBranch(t, "test-branch", "")

	url := fmt.Sprintf("file://%s", r.Path())
	path := makeTempDir(t)

	config := &RepoConfig{
		remoteURL: url,
		path:      path,
	}

	repo, err := CloneRepo(config)
	fmt.Printf("path: %s\n", config.path)

	//defer repo.Close()

	if err != nil {
		t.Fatalf("err(%s)", err)
	}

	r2 := makeTestGit(t)

	repo.AddRemote("origin2", fileURL(r2.Path()))

	remotes, err := repo.repo.Remotes.List()
	if err != nil {
		t.Fatalf("Failed to list remotes")
	}

	for _, r := range remotes {
		fmt.Printf("remtoe (%s)\n", r)
	}

	if err := repo.Fetch(); err != nil {
		t.Fatalf("Failed to Fetch()")
	}
}

func TestRepoFetch(t *testing.T) {
	r := makeTestRepoWithBranch(t, "feature1", "")
	url := fmt.Sprintf("file://%s", r.Path())

	path := makeTempDir(t)
	config := &RepoConfig{
		path:       path,
		remoteURL:  url,
		branchName: "feature1",
	}

	repo, err := CloneRepo(config)
	if err != nil {
		t.Fatalf("Failed to clone repo")
	}
	fmt.Printf("repo(%#v) path(%s)\n", repo, repo.Path())

	snapshot, err := repo.GetSnapshot("")
	dumpSnapshot(snapshot)
	checkFatal(t, err)

	// update README file
	updateReadme(t, r, "HELLO_A")
	updateReadme(t, r, "HELLO_B")
	updateReadme(t, r, "HELLO_C")

	err = repo.Fetch()
	checkFatal(t, err)

	commit := "e491da11fa3ffc9eb80a58374cb99365c1f1aef8"
	snapshot, err = repo.GetSnapshot(commit)
	dumpSnapshot(snapshot)
	checkFatal(t, err)
}

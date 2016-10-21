package main

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/Sirupsen/logrus"

	git "github.com/libgit2/git2go"
)

// DefaultRemoteName = origin
const DefaultRemoteName string = "origin"

// RepoConfig contains configuration for Repo
type RepoConfig struct {
	// local path
	path string
	// initial branch name
	branchName string
	// initial remote name & url
	remoteName string
	remoteURL  string
	// unique id (for convenience since url uniquely idenfies)
	appID string
}

// Repo wraps up Git repository
type Repo struct {
	config     *RepoConfig
	repo       *git.Repository
	log        *logrus.Entry
	branchName string
	remoteName string
	appID      string
}

func (r *Repo) String() string {
	return fmt.Sprintf("pId(%s) remote(%s) branch(%s) lpath(%s)", r.appID, r.remoteName, r.branchName, r.repo.Path())
}

// DefaultPushOptions returns default push options populated
func DefaultPushOptions(log *logrus.Entry) *git.PushOptions {
	return &git.PushOptions{
		RemoteCallbacks: git.RemoteCallbacks{
			// showing Counting objects ..
			SidebandProgressCallback: func(str string) git.ErrorCode {
				log.Debugf("%s", str)
				return 0
			},
			TransferProgressCallback: func(stats git.TransferProgress) git.ErrorCode {
				log.Debugf("Tx progress: %d/%d/%d/%d objs %d B\r",
					stats.IndexedObjects,
					stats.LocalObjects,
					stats.ReceivedObjects,
					stats.TotalObjects,
					stats.ReceivedBytes,
				)
				return 0
			},
			CertificateCheckCallback: func(cert *git.Certificate, valid bool, hostname string) git.ErrorCode {
				log.Printf("Cert check host(%s)\n", hostname)
				return 0
				//return assertHostname(cert, valid, hostname, t)
			},
		},
	}
}

// DefaultFetchOptions returns default fetch options populated
func DefaultFetchOptions(log *logrus.Entry) *git.FetchOptions {
	return &git.FetchOptions{
		RemoteCallbacks: git.RemoteCallbacks{
			// showing Counting objects ..
			SidebandProgressCallback: func(str string) git.ErrorCode {
				log.Debugf("%s", str)
				return 0
			},
			TransferProgressCallback: func(stats git.TransferProgress) git.ErrorCode {
				log.Debugf("Tx progress: %d/%d/%d/%d objs %d B\r",
					stats.IndexedObjects,
					stats.LocalObjects,
					stats.ReceivedObjects,
					stats.TotalObjects,
					stats.ReceivedBytes,
				)
				return 0
			},
			CertificateCheckCallback: func(cert *git.Certificate, valid bool, hostname string) git.ErrorCode {
				log.Printf("Cert check host(%s)\n", hostname)
				return 0
				//return assertHostname(cert, valid, hostname, t)
			},
		},
		Prune: git.FetchNoPrune,
		// Don't ask for any tags beyond the refspecs
		DownloadTags: git.DownloadTagsNone,
	}
}

// InitRepo initailize local Git repository
func InitRepo(path string) (*Repo, error) {
	repo, err := git.InitRepository(path, false)
	if err != nil {
		return nil, err
	}

	config := &RepoConfig{
		path: repo.Path(),
	}

	return &Repo{config: config, repo: repo}, nil
}

// OpenRepo open an exisiting git repository
func OpenRepo(path string) (*Repo, error) {
	rootPath, err := git.Discover(path, false, nil)
	if err != nil {
		return nil, err
	}

	repo, err := git.OpenRepository(rootPath)

	if err != nil {
		fmt.Printf("Failed open root(%s): %v", rootPath, err)
		return nil, err
	}

	config := &RepoConfig{
		path:       repo.Path(),
		branchName: "master",
	}

	return &Repo{config: config, repo: repo}, nil
}

// AddRemoteBranch add remote fetch spec
func (r *Repo) AddRemoteBranch(remoteName, branchName string) error {
	remote, err := r.repo.Remotes.Lookup(remoteName)
	if err != nil {
		return err
	}
	defer remote.Free()

	specs, err := remote.FetchRefspecs()
	if err != nil {
		return err
	}
	fetchSpec := fmt.Sprintf("+refs/heads/%s:refs/remotes/%s/%s", branchName, remoteName, branchName)

	if contains(specs, fetchSpec) {
		r.log.Infof("remote(%s) already contains fetch spec(%s)", fetchSpec, specs)
		return nil
	}

	err = r.repo.Remotes.AddFetch(remoteName, fetchSpec)
	r.log.Infof("adding fetch spec(%s) to remote(%s)", fetchSpec, remoteName)
	if err != nil {
		r.log.Printf("Failed to add fetchspec(%s) to remote(%s)\n", fetchSpec, r.remoteName)
	}
	return nil
}

// CloneRepo clones a repo
func CloneRepo(config *RepoConfig) (*Repo, error) {
	appID := config.appID
	if appID != "" {
		appID = config.appID
	} else {
		url := config.remoteURL
		if url != "" {
			appID = path.Base(url)
		}
	}

	var logger *logrus.Entry
	if appID != "" {
		logger = configureLogger(fmt.Sprintf("repo(%s)", appID))
	} else {
		logger = configureLogger("repo")
	}

	logger.Infof("CloneRepo ID(%s) branch(%s) remote(%s) url(%s) lpath(%s)",
		appID,
		config.branchName,
		config.remoteName,
		config.remoteURL,
		config.path,
	)

	branchName := config.branchName
	if branchName == "" {
		branchName = "master"
	}

	remoteName := config.remoteName
	if remoteName == "" {
		remoteName = "origin"
	}

	// intialize bare repository (git init)
	repo, err := git.InitRepository(config.path, true)
	if err != nil {
		logger.Errorf("Failed to initilize git repository path(%s)\n", config.path)
		return nil, err
	}

	if config.remoteURL != "" {
		remote, err := repo.Remotes.CreateWithFetchspec(remoteName, config.remoteURL, createFetchSpec(remoteName, branchName))
		if err != nil {
			fmt.Printf("Failed to create remtoe(%s)", config.remoteName)
		}
		defer remote.Free()
	}

	r := &Repo{
		config:     config,
		repo:       repo,
		branchName: branchName,
		remoteName: remoteName,
		log:        logger,
		appID:      appID,
	}

	// set HEAD symbolic reference
	if err := r.setHead(branchName); err != nil {
		logger.Errorf("Failed to set head to branch(%s)", branchName)
		return nil, err
	}

	return r, nil
}

// DumpRemotesAll dump all remotes configured
func (r *Repo) DumpRemotesAll() error {
	return r.DumpRemotes("")
}

// DumpRemotes dumps fetchspecs for a remtoe
func (r *Repo) DumpRemotes(remoteName string) error {
	var dumpFetchSpecs = func(rn string) error {
		r.log.Infof("remote(%s)", rn)
		remote, err := r.repo.Remotes.Lookup(rn)
		if err != nil {
			r.log.Printf("Failed to get remote (%s)\n", rn)
			return err
		}

		specs, err := remote.FetchRefspecs()
		if err != nil {
			r.log.Printf("Failed to get fetch specs\n")
			return err
		}
		for _, s := range specs {
			r.log.Infof("  fetch spec(%s)", s)
		}
		return nil
	}

	if remoteName == "" {
		remoteNames, err := r.repo.Remotes.List()
		if err != nil {
			r.log.Errorf("Failed to get remote list")
			return err
		}

		for _, rn := range remoteNames {
			if err := dumpFetchSpecs(rn); err != nil {
				return err
			}
		}
	}

	return nil
}

// CloneRepoAndFetch make a local repository with upstream set to a remote repository
func CloneRepoAndFetch(config *RepoConfig) (*Repo, error) {
	logger := configureLogger("repo")
	logger.Infof("CloneRepo config(%+v)", config)

	branchName := config.branchName
	if branchName == "" {
		branchName = "master"
	}

	remoteName := config.remoteName
	if remoteName == "" {
		remoteName = "origin"
	}

	fetchOpts := DefaultFetchOptions(logger)

	opts := git.CloneOptions{
		FetchOptions:   fetchOpts,
		CheckoutBranch: branchName,
		//CheckoutBranch: config.branchName,
		Bare: true,
		RemoteCreateCallback: func(r *git.Repository, name, url string) (*git.Remote, git.ErrorCode) {
			// name = remote name
			fetchSpec := fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", branchName, branchName)
			logger.Infof("creating remote(%s) url(%s) spec(%s)", name, url, fetchSpec)
			//fetchSpec := "+refs/heads/master:refs/remotes/origin/master"
			remote, err := r.Remotes.CreateWithFetchspec(name, url, fetchSpec)
			if err != nil {
				return nil, git.ErrGeneric
			}
			return remote, git.ErrOk
		},
	}

	repo, err := git.Clone(config.remoteURL, config.path, &opts)
	if err != nil {
		fmt.Printf("Failed to Clone url(%s) path(%s)\n%v\n", config.remoteURL, config.path, err)
		return nil, err
	}

	r := &Repo{
		config:     config,
		repo:       repo,
		log:        logger,
		branchName: branchName,
		remoteName: remoteName,
		appID:      config.appID,
	}

	if err := r.Fetch(); err != nil {
		fmt.Printf("Failed to fetch\n")
		return nil, err
	}
	return r, nil
}

func (r *Repo) setHead(branchName string) error {
	headRef, err := r.repo.References.Lookup("HEAD")
	if err != nil {
		return err
	}
	defer headRef.Free()

	branchRefName := fmt.Sprintf("refs/heads/%s", branchName)
	r.log.Infof("Setting HEAD to ref(%s)", branchRefName)
	newHeadRef, err := headRef.SetSymbolicTarget(branchRefName, "")
	if err != nil {
		return err
	}
	defer newHeadRef.Free()
	return nil
}

func remoteBranchRef(remoteName, branchName string) string {
	return fmt.Sprintf("%s/%s", remoteName, branchName)
}

func createFetchSpec(remoteName, branchName string) string {
	return fmt.Sprintf("refs/heads/%s:refs/remotes/%s/%s", branchName, remoteName, branchName)
}

// getBranch gets remote git.Branch object
func (r *Repo) getBranch() (*git.Branch, error) {
	branch, err := r.repo.LookupBranch(remoteBranchRef(r.RemoteName(), r.BranchName()), git.BranchRemote)
	if err != nil {
		return nil, err
	}
	return branch, nil
}

// GetLatestCommit gets the latest commit form the repository
func (r *Repo) GetLatestCommit() (string, error) {
	branch, err := r.getBranch()
	if err != nil {
		return "", err
	}

	obj, err := r.repo.LookupCommit(branch.Target())
	if err != nil {
		return "", err
	}
	return obj.Id().String(), nil
}

// BranchName returns current branch name
func (r *Repo) BranchName() string {
	return r.branchName
}

func contains(names []string, name string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}

// AddRemote adds a new remote
func (r *Repo) AddRemote(remoteName string, remoteURL string) error {
	remoteNames, err := r.repo.Remotes.List()

	if contains(remoteNames, remoteName) {
		return nil
	}

	remote, err := r.repo.Remotes.CreateWithFetchspec(remoteName, remoteURL, createFetchSpec(remoteName, r.branchName))
	if err != nil {
		return err
	}
	defer remote.Free()

	return nil
}

// Path gives location of git repository
func (r *Repo) Path() string {
	return r.repo.Path()
}

// Push pushes to remote repository
func (r *Repo) Push() error {
	remoteName := r.RemoteName()
	branchName := r.BranchName()

	opt := DefaultPushOptions(r.log)
	r.log.Infof("Push options(%+v)", opt)
	remote, err := r.repo.Remotes.Lookup(remoteName)
	if err != nil {
		r.log.Errorf("Failed to get remote (%s)\n", remoteName)
		return err
	}

	refspecs := []string{fmt.Sprintf("refs/heads/%s", branchName)}
	err = remote.Push(refspecs, opt)
	if err != nil {
		r.log.Errorf("Failed to get remote (%s)\n", remoteName)
		return err
	}
	return nil
}

// Fetch fetches remote references
func (r *Repo) Fetch() error {
	remoteName := r.RemoteName()
	branchName := r.BranchName()

	refspecs := []string{fmt.Sprintf("refs/heads/%s", branchName)}

	remoteRef := fmt.Sprintf("refs/remotes/%s/%s", r.RemoteName(), r.BranchName())

	remote, err := r.repo.Remotes.Lookup(remoteName)
	if err != nil {
		fmt.Printf("Failed to get remote (%s)\n", remoteName)
		return nil
	}

	r.log.Infof("fetching remote ref(%s)", refspecs[0])

	options := DefaultFetchOptions(r.log)
	err = remote.Fetch(refspecs, options, "")
	if err != nil {
		r.log.Errorf("Failed to fetch remote ref(%s)\n", refspecs[0])
		return err
	}

	remoteRefRef, err := r.repo.References.Lookup(remoteRef)
	if err != nil {
		return err
	}
	defer remoteRefRef.Free()
	r.log.Infof("remote ref(%s) found", remoteRef)

	ref, err := remoteRefRef.Resolve()
	if err != nil {
		return err
	}
	defer ref.Free()

	oid := ref.Target()
	r.log.Infof("remote ref(%s) resolved to commit(%s)", remoteRef, oid)

	branchRefName := fmt.Sprintf("refs/heads/%s", branchName)
	_, err = r.repo.References.Lookup(branchRefName)
	if err != nil {
		gerr := err.(*git.GitError)
		if gerr.Code == -3 { // Not found error
			r.log.Debugf("branch ref(%s) not found", branchRefName)
			// create branch
			r.log.Infof("Creating branch ref(%s)", branchRefName)
			_, err0 := r.repo.References.Create(branchRefName, oid, true, "")
			if err0 != nil {
				return err0
			}
			//defer newRef.Free()
		} else {
			r.log.Errorf("Lookup failed: %v code(%v)", err, gerr.Code)
			return err
		}
	}

	//branchRef, err := repo.References.Create(branchName, oid, true, "")
	return nil
}

func (r *Repo) createBranch(t *testing.T, repo *git.Repository, branchRefName string, oid *git.Oid) {
	// creating feature1 branch
	branchRef, err := repo.References.Create(branchRefName, oid, true, "")
	if err != nil {
		t.Fatalf("Failed to create branch ref: %s", branchRefName)
	}
	defer branchRef.Free()
}

// RemoteName returns current remote name
func (r *Repo) RemoteName() string {
	return r.remoteName
}

// SetRemote changes current remote
func (r *Repo) SetRemote(remoteName string) error {
	if remoteName == r.RemoteName() {
		return nil
	}
	r.remoteName = remoteName
	return nil
}

// SetBranch changes remote branch to fetch
func (r *Repo) SetBranch(branchName string) error {
	if branchName == r.BranchName() {
		return nil
	}

	if err := r.AddRemoteBranch(r.RemoteName(), branchName); err != nil {
		return err
	}

	if err := r.setHead(branchName); err != nil {
		return err
	}

	r.branchName = branchName

	return nil
}

// LookupTag find tag in the repo
// http://ben.straub.cc/2013/06/03/refs-tags-and-branching/
func (r *Repo) LookupTag(tagName string) (string, error) {

	// lookup reference
	refName := fmt.Sprintf("refs/tags/%s", tagName)
	ref, err := r.repo.References.Lookup(refName)
	if err != nil {
		fmt.Printf("Failed to find ref(%s)\n", refName)
		return "", err
	}

	// Peel recursively peels an object
	// until an object of the specified type is met.
	tagObj, err := ref.Peel(git.ObjectTag)
	if err != nil {
		fmt.Printf("Failed peel for tag(%s)\n", tagName)
		return "", nil
	}

	tag, err := tagObj.AsTag()
	if err != nil {
		fmt.Printf("Failed to get tag object for tag(%s)\n", tagName)
		return "", nil
	}
	fmt.Printf("Tag found name(%s) msg(%s)\n", tag.Name(), tag.Message())

	commit, err := tag.AsCommit()
	if err != nil {
		fmt.Printf("Failed to get tag commit for tag(%s)\n", tagName)
		return "", nil
	}

	return commit.Id().String(), nil
}

// Close repository
func (r *Repo) Close() error {
	//TODO: Lock
	if r.repo != nil {
		repo := r.repo
		r.repo = nil
		var path string

		if repo.IsBare() {
			path = repo.Path()
		} else {
			path = repo.Workdir()
		}
		repo.Free()
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return err
		}
		return os.RemoveAll(path)
	}
	return nil
}

// GetSnapshot returns the snapshot of the repository for a given commit
func (r *Repo) GetSnapshot(commit string) (*map[string][]byte, error) {
	var oid *git.Oid
	var err error
	var branch *git.Branch

	kv := make(map[string][]byte)

	if commit == "" {
		branch, err = r.getBranch()
		if err != nil {
			fmt.Printf("Failed to get branch object: %v\n", err)
			return nil, err
		}
		oid = branch.Target()
	} else {
		if oid, err = git.NewOid(commit); err != nil {
			return nil, err
		}
	}

	// lookup commit for a given oid
	c, err := r.repo.LookupCommit(oid)
	if err != nil {
		r.log.Printf("Failed to find commit(%s)\n", commit)
		return nil, err
	}

	// get tree object form commit object
	treeObj, err := c.Peel(git.ObjectTree)
	if err != nil {
		r.log.Printf("Failed to peel commit object: %v\n", err)
		return nil, err
	}

	// convert obj to tree
	tree, err := treeObj.AsTree()
	if err != nil {
		fmt.Printf("Failed to get tree: %v\n", err)
		return nil, err
	}

	// walk the tree
	tree.Walk(func(dir string, entry *git.TreeEntry) int {
		name := path.Join(dir, entry.Name)
		switch entry.Type {
		case git.ObjectBlob:
			blob, err := r.repo.LookupBlob(entry.Id)
			if err != nil {
				fmt.Printf("failed to lookup blob")
				return 1
			}
			kv[name] = blob.Contents()
		}
		return 0
	})

	return &kv, nil
}

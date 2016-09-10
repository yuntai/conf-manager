package main

import (
	"fmt"
	"os"

	git "github.com/yuntai/git2go"
)

type RepoConfig struct {
	path       string
	bare       bool
	remoteUrl  string
	branchName string
}

type Repo struct {
	config       *RepoConfig
	repo         *git.Repository
	branch       *git.Branch
	remoteBranch *git.Branch
	remote       *git.Remote
}

func InitRepo(path string) (*Repo, error) {
	isbare := false
	repo, err := git.InitRepository(path, isbare)
	if err != nil {
		return nil, err
	}
	config := &RepoConfig{
		path: repo.Path(),
		bare: repo.IsBare(),
	}
	return &Repo{config: config, repo: repo}, nil
}

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
		path: repo.Path(),
		bare: repo.IsBare(),
	}
	return &Repo{config: config, repo: repo}, nil
}

// CloneRepo make a local repository with upstream set to a remote repository
func CloneRepo(config *RepoConfig) (*Repo, error) {
	opts := git.CloneOptions{
		CheckoutBranch: config.branchName,
		Bare:           config.bare,
		RemoteCreateCallback: func(r *git.Repository, name, url string) (*git.Remote, git.ErrorCode) {
			// name = branch name
			fmt.Printf("RemoteCreateCallback name(%s) url(%s)\n", name, url)

			// limit fetch spec to specific branch
			fetchSpec := fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", config.branchName, config.branchName)
			//fetchSpec := "+refs/heads/*:refs/remotes/origin/*"

			remote, err := r.Remotes.CreateWithFetchspec(name, url, fetchSpec)
			//remote, err := r.Remotes.Create(name, url)
			if err != nil {
				return nil, git.ErrGeneric
			}
			return remote, git.ErrOk
		},
	}

	repo, err := git.Clone(config.remoteUrl, config.path, &opts)
	if err != nil {
		return nil, err
	}

	r := &Repo{config: config, repo: repo}

	branch, err := r.repo.LookupBranch(config.branchName, git.BranchLocal)
	if err != nil {
		return nil, err
	}
	r.branch = branch
	fmt.Printf("branch(%v)\n", branch)

	remote, err := r.repo.Remotes.Lookup("origin")
	if err != nil {
		fmt.Printf("Failed to get remote(%v)", err)
	}
	r.remote = remote

	fetchOpt := git.FetchOptions{
		RemoteCallbacks: git.RemoteCallbacks{
			CertificateCheckCallback: func(cert *git.Certificate, valid bool, hostname string) git.ErrorCode {
				fmt.Printf("cert check host(%s)\n", hostname)
				return 0
				//return assertHostname(cert, valid, hostname, t)
			},
		},
	}

	specs, err := remote.FetchRefspecs()
	if err != nil {
		fmt.Printf("Failed to get fetch specs: %v\n", err)
		return nil, err
	}

	for _, spec := range specs {
		fmt.Printf("spec(%s)\n", spec)
	}

	err = remote.Fetch(specs, &fetchOpt, "")
	if err != nil {
		fmt.Printf("Failed to fetch: %v", err)
		return nil, err
	}

	/*
		bi, err := repo.NewBranchIterator(git.BranchAll)
		bi.ForEach(func(b *git.Branch, t git.BranchType) error {
			s, _ := b.Name()
			fmt.Printf("Branch(%v) Type(%v) n(%s) r(%v)\n", b, t, s, b.IsRemote())
			return nil
		})
	*/

	remoteBranchRef := fmt.Sprintf("%s/%s", remote.Name(), config.branchName)
	remoteBranch, err := r.repo.LookupBranch(remoteBranchRef, git.BranchRemote)
	if err != nil {
		return nil, err
	}
	r.remoteBranch = remoteBranch

	return r, nil
}

func (r *Repo) getBranch(branchName string) (*git.Branch, error) {
	branch, err := r.repo.LookupBranch(branchName, git.BranchLocal)
	if err != nil {
		return nil, err
	}
	return branch, nil
}

func (r *Repo) Fetch() error {
	fetchOpt := git.FetchOptions{
		RemoteCallbacks: git.RemoteCallbacks{
			CertificateCheckCallback: func(cert *git.Certificate, valid bool, hostname string) git.ErrorCode {
				fmt.Printf("cert check host(%s)\n", hostname)
				return 0
				//return assertHostname(cert, valid, hostname, t)
			},
		},
	}

	specs, err := r.remote.FetchRefspecs()
	if err != nil {
		fmt.Printf("Failed to get fetch specs: %v\n", err)
		return err
	}

	err = r.remote.Fetch(specs, &fetchOpt, "")
	if err != nil {
		fmt.Printf("Failed to fetch: %v", err)
		return err
	}
	return nil
}

func (r *Repo) getRemoteTip() (string, error) {
	currentTip, err := r.repo.LookupCommit(r.remoteBranch.Target())
	if err != nil {
		return "", err
	}
	return currentTip.Id().String(), nil
}

// GetTip get the latest commit form the repository
func (r *Repo) getTip() (string, error) {
	currentTip, err := r.repo.LookupCommit(r.branch.Target())
	if err != nil {
		return "", err
	}
	return currentTip.Id().String(), nil
}

func (r *Repo) BranchName() (string, error) {
	name, err := r.branch.Name()
	if err != nil {
		return "", err
	}
	return name, err
}

func (r *Repo) AddRemote(name string, remoteUrl string) error {
	_, err := r.repo.Remotes.Create(name, remoteUrl)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repo) ListRemotes() ([]string, error) {
	return r.repo.Remotes.List()
}

// Close & Cleanup repository
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
		} else {
			return os.RemoveAll(path)
		}
	}
	return nil
}

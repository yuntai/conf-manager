package main

import (
	"fmt"
	git "github.com/yuntai/git2go"
	"os"
)

type RepoRemoteConfig struct {
	url        string
	branchName string
}

type RepoConfig struct {
	path         string
	bare         bool
	remoteConfig *RepoRemoteConfig
}

type Repo struct {
	config *RepoConfig
	repo   *git.Repository
	branch *git.Branch
}

func InitRepo(path string) (*Repo, error) {
	repo, err := git.InitRepository(path, false)
	if err != nil {
		return nil, err
	}
	config := &RepoConfig{
		path:         repo.Path(),
		bare:         repo.IsBare(),
		remoteConfig: nil,
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
		path:         repo.Path(),
		bare:         repo.IsBare(),
		remoteConfig: nil,
	}
	return &Repo{config: config, repo: repo}, nil
}

// CloneRepo make a local repository with upstream set to a remote repository
func CloneRepo(config *RepoConfig) (*Repo, error) {
	remoteConfig := config.remoteConfig
	path := config.path

	// TODO: need to set only branches that are interested
	opts := git.CloneOptions{
		Bare: config.bare,
		RemoteCreateCallback: func(r *git.Repository, name, url string) (*git.Remote, git.ErrorCode) {
			// name = branch name
			fmt.Printf("RemoteCreateCallback name(%s) url(%s)\n", name, url)
			remote, err := r.Remotes.Create(name, url)
			if err != nil {
				return nil, git.ErrGeneric
			}
			return remote, git.ErrOk
		},
	}

	repo, err := git.Clone(remoteConfig.url, path, &opts)
	if err != nil {
		return nil, err
	}
	//repo.SetHead(remoteConfig.branch)

	r := &Repo{config: config, repo: repo}
	branch, err := r.getBranch(remoteConfig.branchName)
	if err != nil {
		return nil, err
	}
	r.branch = branch
	return r, nil
}

func (r *Repo) getBranch(branchName string) (*git.Branch, error) {
	branch, err := r.repo.LookupBranch(branchName, git.BranchLocal)
	if err != nil {
		return nil, err
	}
	return branch, nil
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
	remote, err := r.repo.Remotes.Create(name, remoteUrl)
	options := git.FetchOptions{
		RemoteCallbacks: git.RemoteCallbacks{
			CertificateCheckCallback: func(cert *git.Certificate, valid bool, hostname string) git.ErrorCode {
				return 0
				//return assertHostname(cert, valid, hostname, t)
			},
		},
	}
	err = remote.Fetch([]string{}, &options, "")
	if err != nil {
		return err
	}
	return nil
}

// Close repository
func (r *Repo) Close() error {
	if r.repo != nil {
		r.repo = nil
		repo := r.repo
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

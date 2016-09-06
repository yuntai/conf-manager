package main

import (
	"fmt"
	git "gopkg.in/libgit2/git2go.v24"
	"os"
)

type RepoRemoteConfig struct {
	url    string
	branch string
}

type RepoConfig struct {
	path         string
	bare         bool
	remoteConfig *RepoRemoteConfig
}

type Repo struct {
	config *RepoConfig
	repo   *git.Repository
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
	fmt.Printf("Found repo path(%s)", rootPath)
	repo, err := git.OpenRepository(rootPath)
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

// CloneRepo make a local repository with upstream set to a remote repository
func CloneRepo(config *RepoConfig) (*Repo, error) {
	remoteConfig := config.remoteConfig
	path := config.path

	opts := git.CloneOptions{
		Bare: config.bare,
		RemoteCreateCallback: func(r *git.Repository, name, url string) (*git.Remote, git.ErrorCode) {
			// name = branch name
			fmt.Printf("RemoteCreateCallback name(%s) url(%s)\n", name, remoteConfig.url)
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
	return &Repo{config: config, repo: repo}, nil
}

func (r *Repo) Close() {
	if r.repo != nil {
		repo := r.repo
		var path string

		if repo.IsBare() {
			path = repo.Path()
		} else {
			path = repo.Workdir()
		}
		repo.Free()
		err := os.RemoveAll(path)
		if err != nil {
			fmt.Printf("Failed to remove directory(%s)", path)
		}
	}
}

package main

import (
	"errors"
	"fmt"
	git "github.com/yuntai/git2go"
	"os"
	"strings"
	"sync"
)

type TreeEntry struct {
	oid  string
	name string
}

type stack struct {
	lock sync.Mutex // you don't have to do this if you don't want thread safety
	s    []*TreeEntry
}

func NewStack() *stack {
	return &stack{s: make([]*TreeEntry, 0)}
}

func (s *stack) Push(v *TreeEntry) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.s = append(s.s, v)
}

//TODO: perf cache?
func (s *stack) getKey(basename string) string {
	var names []string
	for _, entry := range s.s {
		names = append(names, entry.name)
	}
	names = append(names, basename)
	return strings.Join(names, "/")
}

func (s *stack) IsEmpty() bool {
	return len(s.s) == 0
}

func (s *stack) Pop() (*TreeEntry, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	l := len(s.s)
	if l == 0 {
		return nil, errors.New("Empty Stack")
	}

	res := s.s[l-1]
	s.s = s.s[:l-1]
	return res, nil
}

func (s *stack) Peep() (*TreeEntry, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	l := len(s.s)
	if l == 0 {
		return nil, errors.New("Empty Stack")
	}
	return s.s[l-1], nil
}

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

func isAncestor(l string, r string, postOrderListMap *map[string]int) bool {
	return (*postOrderListMap)[l] > (*postOrderListMap)[r]
}

func getPrefix(s *stack) {
}

func (r *Repo) getSnapshot() (*map[string][]byte, error) {
	if r.branch == nil {
		branch, err := r.getBranch("master")
		if err != nil {
			fmt.Printf("Failed get master branch\n")
			return nil, err
		}
		r.branch = branch
	}
	currentTip, err := r.repo.LookupCommit(r.branch.Target())
	if err != nil {
		return nil, err
	}

	// get tree object form commit object
	obj, err := currentTip.Peel(git.ObjectTree)
	if err != nil {
		fmt.Printf("Failed to peel commit object: %v\n", err)
		return nil, err
	}

	tree, err := obj.AsTree()
	if err != nil {
		fmt.Printf("Failed to get tree: %v\n", err)
		return nil, err
	}

	fmt.Printf("Tree(%#v)\n", tree)

	//var kv map[string][]byte

	var postOrderIndex int
	postOrderListMap := make(map[string]int)

	tree.WalkWithMode(func(name string, entry *git.TreeEntry) int {
		oid := entry.Id.String()
		fmt.Printf("name(%v) type(%v) oid(%v)\n", entry.Name, entry.Type, oid)
		postOrderListMap[entry.Id.String()] = postOrderIndex
		postOrderIndex += 1
		return 0
	}, git.TreeWalkModePost)

	fmt.Printf("Post order list(%v)\n", postOrderListMap)
	kv := make(map[string][]byte)
	stack := NewStack()
	tree.WalkWithMode(func(name string, entry *git.TreeEntry) int {
		oid := entry.Id.String()
		fmt.Printf("name(%v) type(%v) oid(%v)\n", entry.Name, entry.Type, oid)

		for {
			if stack.IsEmpty() {
				break
			}
			p, err := stack.Peep()
			if err != nil {
				panic(err)
			}

			if !isAncestor(p.oid, oid, &postOrderListMap) {
				_, err := stack.Pop()
				if err != nil {
					panic(err)
				}
			} else {
				break
			}
		}

		if entry.Type == git.ObjectTree {
			stack.Push(&TreeEntry{oid, entry.Name})
		} else if entry.Type == git.ObjectBlob {
			blobKey := stack.getKey(entry.Name)
			blob, err := r.repo.LookupBlob(entry.Id)
			if err != nil {
				panic(err)
			}
			kv[blobKey] = blob.Contents()
		} else {
			fmt.Printf("Object type(%v) not supported\n", entry.Type)
		}

		return 0
	}, git.TreeWalkModePre)
	return &kv, nil
}

func (r *Repo) BranchName() (string, error) {
	name, err := r.branch.Name()
	if err != nil {
		return "", err
	}
	return name, err
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

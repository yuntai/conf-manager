package main

import (
	"errors"
	"fmt"
	git "github.com/yuntai/git2go"
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

func isAncestor(l string, r string, postOrderListMap *map[string]int) bool {
	return (*postOrderListMap)[l] > (*postOrderListMap)[r]
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

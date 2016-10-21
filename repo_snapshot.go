package main

/*
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
*/

/*
seems useless
func (r *Repo) GetSnapshot(commit string) (*map[string][]byte, error) {
	var oid *git.Oid
	var err error

	if commit == "" {
		oid = r.branch.Target()
	} else {
		if oid, err = git.NewOid(commit); err != nil {
			return nil, err
		}
	}

	fmt.Printf("Looking up oid(%s)\n", oid)
	c, err := r.repo.LookupCommit(oid)
	if err != nil {
		fmt.Printf("Failed to find commit(%s)\n", commit)
		return nil, err
	}

	// get tree object form commit object
	obj, err := c.Peel(git.ObjectTree)
	if err != nil {
		fmt.Printf("Failed to peel commit object: %v\n", err)
		return nil, err
	}

	tree, err := obj.AsTree()
	if err != nil {
		fmt.Printf("Failed to get tree: %v\n", err)
		return nil, err
	}

	//fmt.Printf("Tree(%#v)\n", tree)

	var postOrderIndex int
	postOrderListMap := make(map[string]int)

	// walk with post-order
	tree.WalkWithMode(func(name string, entry *git.TreeEntry) int {
		// oid := entry.Id.String()
		//fmt.Printf("name(%v) type(%v) oid(%v)\n", entry.Name, entry.Type, oid)
		postOrderListMap[entry.Id.String()] = postOrderIndex
		postOrderIndex += 1
		return 0
	}, git.TreeWalkModePost)

	// walk with pre-order
	//fmt.Printf("Post order list(%v)\n", postOrderListMap)

	kv := make(map[string][]byte)

	stack := NewStack()
	tree.WalkWithMode(func(name string, entry *git.TreeEntry) int {
		oid := entry.Id.String()
		//fmt.Printf("name(%v) type(%v) oid(%v)\n", entry.Name, entry.Type, oid)

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
			panic(fmt.Sprintf("Object type(%v) not supported\n", entry.Type))
		} else {
		}

		return 0
	}, git.TreeWalkModePre)
	return &kv, nil
}
*/

package main

import (
	"fmt"
)

func main() {
	repo, err := OpenRepo("/mnt/tmp/repotest")
	if err != nil {
		fmt.Printf("Failed open repository(%v)\n", err)
		return
	}
	snapshot, err := repo.getSnapshot()
	if err != nil {
		panic(err)
	}
	for k, v := range *snapshot {
		fmt.Printf("k(%s) v:\n%s", k, string(v))
	}

	fmt.Printf("-------------------\n")

	repo, err = OpenRepo("/mnt/tmp/repotest2")
	if err != nil {
		fmt.Printf("Failed open repository(%v)\n", err)
		return
	}
	snapshot, err = repo.getSnapshot()
	if err != nil {
		panic(err)
	}
	for k, v := range *snapshot {
		fmt.Printf("k(%s) v:\n%s", k, string(v))
	}
}

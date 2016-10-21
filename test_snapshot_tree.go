package main

import (
	"fmt"
)

// http://ben.straub.cc/2013/06/03/refs-tags-and-branching/
func testTag() {
	repo, err := OpenRepo("/mnt/tmp/repotest")
	if err != nil {
		fmt.Printf("Failed open repository(%v)\n", err)
		return
	}
	repo.LookupTag("v1.4")
	//tag, err := repo.repo.LookupTag(tagId)
	if tags, err := repo.repo.Tags.List(); err != nil {
		fmt.Printf("Failed to get tag list\n")
	} else {
		for _, t := range tags {
			fmt.Printf("Tag(%v)\n", t)
		}
	}
}

func run() {
	repo, err := OpenRepo("/mnt/tmp/repotest2")
	if err != nil {
		fmt.Printf("Failed open repository(%v)\n", err)
		return
	}

	var commit string
	//commit := "f2e86f2a221e164b073598a744df9b77af86f6e2"
	commit = "86e561b54d903f499a3605d771234d75cf3e54e1"
	//commit := "bbfbda0d0a73d70018634079862b3d94220486cf"
	commit = "632d6dba0ddd7f501fbb21c814143d770c96133d"
	commit = ""

	snapshot, err := repo.GetSnapshot(commit)
	if err != nil {
		panic(err)
	}
	for k, v := range *snapshot {
		fmt.Printf("%s\n  %s\n", k, string(v))
	}

	/*
		snapshot, err = repo.GetSnapshot2(commit)
		if err != nil {
			panic(err)
		}
		for k, v := range *snapshot {
			fmt.Printf("%s\n  %s\n", k, string(v))
		}
	*/

	/*
		fmt.Printf("-------------------\n")

		repo, err = OpenRepo("/mnt/tmp/repotest2")
		if err != nil {
			fmt.Printf("Failed open repository(%v)\n", err)
			return
		}
		snapshot, err = repo.GetSnapshot("")
		if err != nil {
			panic(err)
		}
		for k, v := range *snapshot {
			fmt.Printf("k(%s) v:\n%s", k, string(v))
		}
	*/
}

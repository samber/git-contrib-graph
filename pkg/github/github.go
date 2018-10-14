package github

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/samber/git-contrib-graph/pkg/config"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

func Repo(gitPath string, gitRemote string) object.CommitIter {
	var err error
	var repo *git.Repository
	var path string

	if gitPath != "" {
		path = gitPath
		repo, err = git.PlainOpen(gitPath)
	} else if gitRemote != "" {
		path = gitRemote
		parts := strings.Split(path, "/")
		dir, err := ioutil.TempDir("", parts[len(parts)-1])
		if err != nil {
			log.Fatal(err)
		}
		repo, err = git.PlainClone(dir, false, &git.CloneOptions{
			URL:          gitRemote,
			SingleBranch: true,
		})
	} else {
		log.Fatal("Repository not found")
	}

	if err != nil {
		log.Fatalf("Failed to find repo %s: %s", path, err)
	}

	cIter, err := repo.Log(&git.LogOptions{
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		log.Fatalf("Error: %s", err)
	}

	if config.JSONOutput == false {
		fmt.Printf("Repo: %s\n\n", path)
		fmt.Printf("Contributions to master, excluding merge commits:\n\n")
	}

	return cIter
}

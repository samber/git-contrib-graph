package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/samber/git-contrib-graph/pkg/config"
	"github.com/samber/git-contrib-graph/pkg/github"
	graphStats "github.com/samber/git-contrib-graph/pkg/stats"
	datePkg "github.com/samber/git-contrib-graph/pkg/utils/date"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

func main() {
	contribs := map[string]map[string]graphStats.Stats{}

	cIter := github.Repo(config.GitPath, config.GitRemote)

	// scan history
	err := cIter.ForEach(func(c *object.Commit) error {
		author := c.Author.Email
		if config.AuthorEmail != "" && author != config.AuthorEmail {
			return nil
		}
		date := c.Author.When.Format(datePkg.DateFormat)

		d, err := time.Parse(datePkg.DateFormat, date)
		if err != nil {
			log.Fatalf("Error: %s", err)
		}

		if !datePkg.InTimeSpan(config.SinceDate, config.ToDate, d) {
			return nil
		}

		if _, ok := contribs[author]; ok == false {
			// init author
			contribs[author] = map[string]graphStats.Stats{}
		}
		if _, ok := contribs[author][date]; ok == false {
			// init date (grouped by author)
			contribs[author][date] = graphStats.Stats{
				Commits:  0,
				Files:    0,
				Addition: 0,
				Deletion: 0,
			}
		}

		day, _ := contribs[author][date]

		stats, err := c.Stats()
		if err == nil {
			// prevent merge to be counted as changes
			if len(c.ParentHashes) == 1 {
				for i := 0; i < len(stats); i++ {
					// fill stats
					day.Files++
					day.Addition += stats[i].Addition
					day.Deletion += stats[i].Deletion
				}
			}
		} else {
			// initial commit case
			if len(c.ParentHashes) == 0 {
				f, a, err := graphStats.InitialCommits(c)
				if err != nil {
					log.Fatalf("Failed to fetch initial commit stats: %s", err)
				}
				day.Files += f
				day.Addition += a
			} else {
				log.Fatal(err)
			}
		}

		day.Commits++
		contribs[author][date] = day
		return nil
	})

	if err != nil {
		log.Fatalf("Error: %s", err)
	}

	totals := graphStats.AggregateAuthors(contribs)

	if config.JSONOutput {
		enc := json.NewEncoder(os.Stdout)
		err := enc.Encode(totals)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Print(totals)
	}
}

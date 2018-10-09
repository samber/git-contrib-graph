package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

type Stats struct {
	Commits  int
	Files    int
	Addition int
	Deletion int
}

const (
	DATE_FORMAT = "2006-01-02"
)

var (
	NBR_COLUMN int
	INTERVAL   string
	FULL_GRAPH bool

	GREEN_COLOR = "\x1b[32m"
	RED_COLOR   = "\x1b[31m"
	BLUE_COLOR  = "\x1b[94m"
	RESET_COLOR = "\x1b[0m"
)

func getTotalsByAuthor(days map[string]Stats) (int, int, int, int) {
	commitCount := 0
	filesSum := 0
	additionSum := 0
	deletionSum := 0

	for _, v := range days {
		commitCount += v.Commits
		filesSum += v.Files
		additionSum += v.Addition
		deletionSum += v.Deletion
	}
	return commitCount, filesSum, additionSum, deletionSum
}

func getPlusMinusProgression(additions int, deletions int, maxChars int) string {
	changes := additions + deletions
	if changes > maxChars {
		rate := float64(maxChars) / float64(changes)
		additions = int(math.Round(float64(additions) * rate))
		deletions = int(math.Round(float64(deletions) * rate))
	}
	return GREEN_COLOR + strings.Repeat("+", additions) + RED_COLOR + strings.Repeat("-", deletions) + RESET_COLOR
}

func getDateLimits(contribs map[string]map[string]Stats) (time.Time, time.Time) {
	min := time.Time{}
	max := time.Time{}

	for _, v := range contribs {
		for k, _ := range v {
			if date, err := time.Parse(DATE_FORMAT, k); err == nil {
				if min.IsZero() || date.Before(min) {
					min = date
				}
				if max.IsZero() || date.After(max) {
					max = date
				}
			} else {
				log.Fatalf("Failed to parse date %s: %s", k, err)
			}
		}
	}
	return min, max
}

func getIntervalContribs(start time.Time, days map[string]Stats) (int, int, int) {
	addition := 0
	deletion := 0
	commits := 0

	// compute last day of stats collection, based on INTERVAL parameter
	end := start.AddDate(0, 0, 1)
	if INTERVAL == "week" {
		end = start.AddDate(0, 0, 7)
	} else if INTERVAL == "month" {
		end = start.AddDate(0, 1, 0)
	}

	// addition changes in range
	for start.Before(end) {
		strDate := start.Format(DATE_FORMAT)
		day, ok := days[strDate]
		if ok == true {
			addition += day.Addition
			deletion += day.Deletion
			commits += day.Commits
		}
		start = start.AddDate(0, 0, 1)
	}

	return addition, deletion, commits
}

func printAuthorContribGraph(minDate time.Time, maxDate time.Time, days map[string]Stats) {
	from := minDate
	to := maxDate.AddDate(0, 0, 1) // maxDate included

	// `from` must be on sunday if interval == "week" or at the begining of the month if interval == "month"
	if INTERVAL == "week" {
		from = from.AddDate(0, 0, -int(from.Weekday()))
	} else if INTERVAL == "month" {
		from = from.AddDate(0, 0, -from.Day()+1)
	}

	for from.Before(to) {
		addition, deletion, commits := getIntervalContribs(from, days)

		// display if FULL_GRAPH parameter is set or if current author commited something
		if commits > 0 || FULL_GRAPH == true {
			fmt.Printf(
				"   %s | %3d(+) %3d(-) %s\n",
				from.Format(DATE_FORMAT),
				addition,
				deletion,
				getPlusMinusProgression(addition, deletion, NBR_COLUMN-30),
			)
		}

		// next day
		if INTERVAL == "day" {
			from = from.AddDate(0, 0, 1)
		} else if INTERVAL == "week" {
			from = from.AddDate(0, 0, 7)
		} else {
			from = from.AddDate(0, 1, 0)
		}
	}
}

func printAuthors(contribs map[string]map[string]Stats) {
	minDate, maxDate := getDateLimits(contribs)

	for author, days := range contribs {
		commitCount, _, additionSum, deletionSum := getTotalsByAuthor(days)

		// author header
		fmt.Printf(
			"\n\n%s\n%s\n\n\nAuthor: %s%s%s\n\nTotal:\n   %d commits\n   Insertions: %4d %s\n   Deletions:  %4d %s\n\nPer day:\n",
			strings.Repeat("#", NBR_COLUMN),
			strings.Repeat("#", NBR_COLUMN),
			BLUE_COLOR,
			author,
			RESET_COLOR,
			commitCount,
			additionSum,
			getPlusMinusProgression(additionSum, 0, NBR_COLUMN-20),
			deletionSum,
			getPlusMinusProgression(0, deletionSum, NBR_COLUMN-20),
		)

		printAuthorContribGraph(minDate, maxDate, days)
	}
}

func getInitialCommitStats(c *object.Commit) (int, int, error) {
	files, err := c.Files()
	if err != nil {
		return 0, 0, err
	}

	file := 0
	addition := 0

	err = files.ForEach(func(f *object.File) error {
		file++

		// the following line is fucking slow !!
		lines, err := f.Lines()
		if err == nil {
			addition += len(lines)
		}

		return err
	})

	return file, addition, err
}

func getRepo(git_path string, git_remote string) object.CommitIter {
	var err error
	var repo *git.Repository
	var path string

	if git_path != "" {
		path = git_path
		repo, err = git.PlainOpen(git_path)
	} else if git_remote != "" {
		path = git_remote
		parts := strings.Split(path, "/")
		dir, err := ioutil.TempDir("", parts[len(parts)-1])
		if err != nil {
			log.Fatal(err)
		}
		repo, err = git.PlainClone(dir, false, &git.CloneOptions{
			URL:          git_remote,
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

	fmt.Printf("Repo: %s\n\n", path)
	fmt.Printf("Contributions to master, excluding merge commits:\n\n")

	return cIter
}

func getConfig() (string, string) {
	git_path := flag.String("git-path", "", "Fetch logs from local git repository (bare or normal)")
	git_remote := flag.String("git-remote", "", "Fetch logs from remote git repository Github, Gitlab...")
	no_colors := flag.Bool("no-colors", false, "Disabled colors in output")

	flag.IntVar(&NBR_COLUMN, "max-columns", 80, "Number of columns in your terminal or output")
	flag.StringVar(&INTERVAL, "interval", "day", "Display contributions per day, week or month")
	flag.BoolVar(&FULL_GRAPH, "full-graph", false, "Display days without contributions")
	flag.Parse()

	if *git_path == "" && *git_remote == "" {
		fmt.Println("Please provide a --git-path or --git-remote argument")
		flag.PrintDefaults()
		os.Exit(1)
	}
	if *no_colors == true {
		BLUE_COLOR = ""
		GREEN_COLOR = ""
		RED_COLOR = ""
		RESET_COLOR = ""
	}
	if INTERVAL != "day" && INTERVAL != "week" && INTERVAL != "month" {
		log.Fatalf("Invalid date range: %s", INTERVAL)
	}

	return *git_path, *git_remote
}

func main() {
	git_path, git_remote := getConfig()
	contribs := map[string]map[string]Stats{}

	cIter := getRepo(git_path, git_remote)

	// scan history
	err := cIter.ForEach(func(c *object.Commit) error {
		// id := c.Hash.String()
		date := c.Author.When.Format("2006-01-02")
		author := c.Author.Email

		if _, ok := contribs[author]; ok == false {
			// init author
			contribs[author] = map[string]Stats{}
		}
		if _, ok := contribs[author][date]; ok == false {
			// init date (grouped by author)
			contribs[author][date] = Stats{
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
				f, a, err := getInitialCommitStats(c)
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

	printAuthors(contribs)
}

package main

import (
	"sort"
	"encoding/json"
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

type stats struct {
	Commits  int `json:"commits"`
	Files    int `json:"-"`
	Addition int `json:"insertions"`
	Deletion int `json:"deletions"`
}

type totalStats struct {
	Interval     string             `json:"interval"`
	Contributors []contributorStats `json:"contributors"`
}

func (ts totalStats) String() string {
	b := &strings.Builder{}

	for _, cs := range ts.Contributors {
		b.WriteString(cs.String())
	}

	return b.String()
}

type contributorStats struct {
	Author string          `json:"author"`
	Totals stats           `json:"totals"`
	Graph  []intervalStats `json:"graph"`
}

func (cs contributorStats) String() string {
	b := &strings.Builder{}

	fmt.Fprintf(
		b,
		"\n\n%s\n%s\n\n\nAuthor: %s%s%s\n\nTotal:\n   %d commits\n   Insertions: %4d %s\n   Deletions:  %4d %s\n\nPer day:\n",
		strings.Repeat("#", nbrColumn),
		strings.Repeat("#", nbrColumn),
		blueColor,
		cs.Author,
		resetColor,
		cs.Totals.Commits,
		cs.Totals.Addition,
		getPlusMinusProgression(cs.Totals.Addition, 0, nbrColumn-20),
		cs.Totals.Deletion,
		getPlusMinusProgression(0, cs.Totals.Deletion, nbrColumn-20),
	)

	for _, is := range cs.Graph {
		b.WriteString(is.String())
	}

	return b.String()
}

type intervalStats struct {
	Date     time.Time
	Addition int
	Deletion int
}

func (is intervalStats) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`{"date": %q, "add": %d, "sub": %d}`,
		is.Date.Format(dateFormat),
		is.Addition,
		is.Deletion,
	)), nil
}

func (is intervalStats) String() string {
	return fmt.Sprintf(
		"   %s | %3d(+) %3d(-) %s\n",
		is.Date.Format(dateFormat),
		is.Addition,
		is.Deletion,
		getPlusMinusProgression(is.Addition, is.Deletion, nbrColumn-30),
	)
}

const (
	dateFormat = "2006-01-02"
)

var (
	nbrColumn   int
	interval    string
	fullGraph   bool
	jsonOutput  bool
	authorEmail string

	greenColor = "\x1b[32m"
	redColor   = "\x1b[31m"
	blueColor  = "\x1b[94m"
	resetColor = "\x1b[0m"
)

func getTotalsByAuthor(days map[string]stats) (int, int, int, int) {
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
	return greenColor + strings.Repeat("+", additions) + redColor + strings.Repeat("-", deletions) + resetColor
}

func getDateLimits(contribs map[string]map[string]stats) (time.Time, time.Time) {
	min := time.Time{}
	max := time.Time{}

	for _, v := range contribs {
		for k := range v {
			if date, err := time.Parse(dateFormat, k); err == nil {
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

func getIntervalContribs(start time.Time, days map[string]stats) (int, int, int) {
	addition := 0
	deletion := 0
	commits := 0

	// compute last day of stats collection, based on interval parameter
	end := start.AddDate(0, 0, 1)
	if interval == "week" {
		end = start.AddDate(0, 0, 7)
	} else if interval == "month" {
		end = start.AddDate(0, 1, 0)
	}

	// addition changes in range
	for start.Before(end) {
		strDate := start.Format(dateFormat)
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

func aggregateIntervalStatistics(minDate time.Time, maxDate time.Time, days map[string]stats) []intervalStats {
	from := minDate
	to := maxDate.AddDate(0, 0, 1) // maxDate included

	// `from` must be on sunday if interval == "week" or at the begining of the month if interval == "month"
	if interval == "week" {
		from = from.AddDate(0, 0, -int(from.Weekday()))
	} else if interval == "month" {
		from = from.AddDate(0, 0, -from.Day()+1)
	}

	var iss []intervalStats
	for from.Before(to) {
		addition, deletion, commits := getIntervalContribs(from, days)

		// display if fullGraph parameter is set or if current author commited something
		if commits > 0 || fullGraph == true {
			iss = append(iss, intervalStats{
				Date:     from,
				Addition: addition,
				Deletion: deletion,
			})
		}

		// next day
		if interval == "day" {
			from = from.AddDate(0, 0, 1)
		} else if interval == "week" {
			from = from.AddDate(0, 0, 7)
		} else {
			from = from.AddDate(0, 1, 0)
		}
	}

	return iss
}

func aggregateAuthors(contribs map[string]map[string]stats) totalStats {
	minDate, maxDate := getDateLimits(contribs)

	totalStats := totalStats{
		Interval: interval,
	}

	var authors []string
	for author, _ := range contribs {
		authors = append(authors, author)
	}

	sort.Strings(authors)

	for _, author := range authors {
		days := contribs[author]
		commitCount, _, additionSum, deletionSum := getTotalsByAuthor(days)

		contribStats := contributorStats{
			Author: author,
			Totals: stats{
				Commits:  commitCount,
				Addition: additionSum,
				Deletion: deletionSum,
			},
		}

		contribStats.Graph = aggregateIntervalStatistics(minDate, maxDate, days)
		totalStats.Contributors = append(totalStats.Contributors, contribStats)
	}

	return totalStats
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

func getRepo(gitPath string, gitRemote string) object.CommitIter {
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

	if jsonOutput == false {
		fmt.Printf("Repo: %s\n\n", path)
		fmt.Printf("Contributions to master, excluding merge commits:\n\n")
	}

	return cIter
}

func getConfig() (string, string) {
	gitPath := flag.String("git-path", "", "Fetch logs from local git repository (bare or normal)")
	gitRemote := flag.String("git-remote", "", "Fetch logs from remote git repository Github, Gitlab...")
	noColors := flag.Bool("no-colors", false, "Disabled colors in output")

	flag.IntVar(&nbrColumn, "max-columns", 80, "Number of columns in your terminal or output")
	flag.StringVar(&interval, "interval", "day", "Display contributions per day, week or month")
	flag.BoolVar(&fullGraph, "full-graph", false, "Display days without contributions")
	flag.BoolVar(&jsonOutput, "json", false, "Display json output contributions object")
	flag.StringVar(&authorEmail, "author-email", "", "Display graph for a single committer")
	flag.Parse()

	if *gitPath == "" && *gitRemote == "" {
		fmt.Println("Please provide a --git-path or --git-remote argument")
		flag.PrintDefaults()
		os.Exit(1)
	}
	if *noColors == true {
		blueColor = ""
		greenColor = ""
		redColor = ""
		resetColor = ""
	}
	if interval != "day" && interval != "week" && interval != "month" {
		log.Fatalf("Invalid date range: %s", interval)
	}

	return *gitPath, *gitRemote
}

func main() {
	gitPath, gitRemote := getConfig()
	contribs := map[string]map[string]stats{}

	cIter := getRepo(gitPath, gitRemote)

	// scan history
	err := cIter.ForEach(func(c *object.Commit) error {
		// id := c.Hash.String()
		author := c.Author.Email
		if authorEmail != "" && author != authorEmail {
			return nil
		}
		date := c.Author.When.Format(dateFormat)

		if _, ok := contribs[author]; ok == false {
			// init author
			contribs[author] = map[string]stats{}
		}
		if _, ok := contribs[author][date]; ok == false {
			// init date (grouped by author)
			contribs[author][date] = stats{
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

	totals := aggregateAuthors(contribs)

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		err := enc.Encode(totals)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Print(totals)
	}
}

package stats

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/samber/git-contrib-graph/pkg/utils/date"

	"github.com/samber/git-contrib-graph/pkg/config"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

type Stats struct {
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
	Totals Stats           `json:"totals"`
	Graph  []intervalStats `json:"graph"`
}

func (cs contributorStats) String() string {
	b := &strings.Builder{}

	fmt.Fprintf(
		b,
		"\n\n%s\n%s\n\n\nAuthor: %s%s%s\n\nTotal:\n   %d commits\n   Insertions: %4d %s\n   Deletions:  %4d %s\n\nPer day:\n",
		strings.Repeat("#", config.NbrColumn),
		strings.Repeat("#", config.NbrColumn),
		config.BlueColor,
		cs.Author,
		config.ResetColor,
		cs.Totals.Commits,
		cs.Totals.Addition,
		getPlusMinusProgression(cs.Totals.Addition, 0, config.NbrColumn-20),
		cs.Totals.Deletion,
		getPlusMinusProgression(0, cs.Totals.Deletion, config.NbrColumn-20),
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
		is.Date.Format(date.DateFormat),
		is.Addition,
		is.Deletion,
	)), nil
}

func (is intervalStats) String() string {
	return fmt.Sprintf(
		"   %s | %3d(+) %3d(-) %s\n",
		is.Date.Format(date.DateFormat),
		is.Addition,
		is.Deletion,
		getPlusMinusProgression(is.Addition, is.Deletion, config.NbrColumn-30),
	)
}

func AggregateAuthors(contribs map[string]map[string]Stats) totalStats {
	minDate, maxDate := getDateLimits(contribs)

	totalStats := totalStats{
		Interval: config.Interval,
	}

	for author, days := range contribs {
		commitCount, _, additionSum, deletionSum := getTotalsByAuthor(days)

		contribStats := contributorStats{
			Author: author,
			Totals: Stats{
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

func InitialCommits(c *object.Commit) (int, int, error) {
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

func getPlusMinusProgression(additions int, deletions int, maxChars int) string {
	changes := additions + deletions
	if changes > maxChars {
		rate := float64(maxChars) / float64(changes)
		additions = int(math.Round(float64(additions) * rate))
		deletions = int(math.Round(float64(deletions) * rate))
	}
	return config.GreenColor + strings.Repeat("+", additions) + config.RedColor + strings.Repeat("-", deletions) + config.ResetColor
}

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

func getDateLimits(contribs map[string]map[string]Stats) (time.Time, time.Time) {
	min := time.Time{}
	max := time.Time{}

	for _, v := range contribs {
		for k := range v {
			if date, err := time.Parse(date.DateFormat, k); err == nil {
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

	// compute last day of stats collection, based on interval parameter
	end := start.AddDate(0, 0, 1)
	if config.Interval == "week" {
		end = start.AddDate(0, 0, 7)
	} else if config.Interval == "month" {
		end = start.AddDate(0, 1, 0)
	}

	// addition changes in range
	for start.Before(end) {
		strDate := start.Format(date.DateFormat)
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

func aggregateIntervalStatistics(minDate time.Time, maxDate time.Time, days map[string]Stats) []intervalStats {
	from := minDate
	to := maxDate.AddDate(0, 0, 1) // maxDate included

	// `from` must be on sunday if interval == "week" or at the begining of the month if interval == "month"
	if config.Interval == "week" {
		from = from.AddDate(0, 0, -int(from.Weekday()))
	} else if config.Interval == "month" {
		from = from.AddDate(0, 0, -from.Day()+1)
	}

	var iss []intervalStats
	for from.Before(to) {
		addition, deletion, commits := getIntervalContribs(from, days)

		// display if fullGraph parameter is set or if current author commited something
		if commits > 0 || config.FullGraph == true {
			iss = append(iss, intervalStats{
				Date:     from,
				Addition: addition,
				Deletion: deletion,
			})
		}

		// next day
		if config.Interval == "day" {
			from = from.AddDate(0, 0, 1)
		} else if config.Interval == "week" {
			from = from.AddDate(0, 0, 7)
		} else {
			from = from.AddDate(0, 1, 0)
		}
	}

	return iss
}

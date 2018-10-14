package stats

import (
	"errors"
	"fmt"
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func TestCorrectSumTotalsChangesByAuthor(t *testing.T) {
	convey.Convey("Given two day with changes of a author", t, func() {
		days := make(map[string]Stats)
		changesFirstDay := Stats{Addition: 10, Commits: 2, Deletion: 3, Files: 2}
		changesSecondDay := Stats{Addition: 2, Commits: 1, Deletion: 4, Files: 1}

		days["2018-12-10"] = changesFirstDay
		days["2018-12-11"] = changesSecondDay
		convey.Convey("When get sum a changes of the days", func() {
			commitCount, filesSum, additionSum, deletionSum := getTotalsByAuthor(days)
			convey.Convey("Should return correct sum a changes of the days", func() {

				expectedCommitCount := 3
				expectedFileSum := 3
				expectedAdditionSum := 12
				expectedDeletionSum := 7

				var err error
				if commitCount != expectedCommitCount || filesSum != expectedFileSum || additionSum != expectedAdditionSum || deletionSum != expectedDeletionSum {
					err = errors.New(fmt.Sprintf("Doesn't returned correct value to sum changes, returned commit %v, file %v, addition %v and deletion %v but expected commit 3, file 3, addition 12 and deletion 7 ", commitCount, filesSum, additionSum, deletionSum))
				}
				convey.So(err, convey.ShouldBeNil)
			})

		})
	})

}

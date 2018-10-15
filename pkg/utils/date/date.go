package date

import (
	"fmt"
	"os"
	"time"
)

const (
	DateFormat = "2006-01-02"
)

func InTimeSpan(start, end, check time.Time) bool {
	return check.After(start) && check.Before(end)
}

func Parse(dateStr string) time.Time {
	d, err := time.Parse(DateFormat, dateStr)
	if err != nil {
		fmt.Printf("Wrong date format: %s | Please provide date in format: `%s`\n", dateStr, DateFormat)
		os.Exit(1)
	}
	return d
}

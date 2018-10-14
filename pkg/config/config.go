package config

import (
	"flag"
	"fmt"
	"log"
	"os"
)

// Flags
var (
	NbrColumn   int
	Interval    string
	FullGraph   bool
	JSONOutput  bool
	AuthorEmail string
	GitPath     string
	GitRemote   string
)

// Colors
var (
	GreenColor = "\x1b[32m"
	RedColor   = "\x1b[31m"
	BlueColor  = "\x1b[94m"
	ResetColor = "\x1b[0m"
)

func init() {
	noColors := flag.Bool("no-colors", false, "Disabled colors in output")

	flag.StringVar(&GitPath, "git-path", "", "Fetch logs from local git repository (bare or normal)")
	flag.StringVar(&GitRemote, "git-remote", "", "Fetch logs from remote git repository Github, Gitlab...")
	flag.IntVar(&NbrColumn, "max-columns", 80, "Number of columns in your terminal or output")
	flag.StringVar(&Interval, "interval", "day", "Display contributions per day, week or month")
	flag.BoolVar(&FullGraph, "full-graph", false, "Display days without contributions")
	flag.BoolVar(&JSONOutput, "json", false, "Display json output contributions object")
	flag.StringVar(&AuthorEmail, "author-email", "", "Display graph for a single committer")
	flag.Parse()

	if GitPath == "" && GitRemote == "" {
		fmt.Println("Please provide a --git-path or --git-remote argument")
		flag.PrintDefaults()
		os.Exit(1)
	}
	if *noColors == true {
		BlueColor = ""
		GreenColor = ""
		RedColor = ""
		ResetColor = ""
	}
	if Interval != "day" && Interval != "week" && Interval != "month" {
		log.Fatalf("Invalid date range: %s", Interval)
	}
}

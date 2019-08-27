package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/stamblerre/work-stats/github"
	"github.com/stamblerre/work-stats/golang"
	"golang.org/x/build/maintner"
	"golang.org/x/build/maintner/godata"
)

var (
	username = flag.String("username", "", "GitHub username")
	email    = flag.String("email", "", "Gerrit email or emails, comma-separated")
	since    = flag.String("since", "", "date since when to collect data")

	// Optional flags.
	goIssuesFlag      = flag.Bool("go_issues", true, "If false, do not collect data on Go issues")
	goChangelistsFlag = flag.Bool("go_cls", true, "If false, do not collect data on Go changelists")
	githubIssuesFlag  = flag.Bool("github_issues", true, "If false, do not collect data on GitHub issues")

	// Globals.
	corpus *maintner.Corpus
)

func main() {
	flag.Parse()

	// Username and email are required flags.
	// If since is omitted, results reflect all history.
	if *username == "" {
		log.Fatalf("please provide a Github username")
	}
	if *email == "" {
		log.Fatalf("please provide your Gerrit email")
	}
	emails := strings.Split(*email, ",")

	// Parse out the start date, if provided.
	var (
		start time.Time
		err   error
	)
	if *since != "" {
		start, err = time.Parse("2006-01-02", *since)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		start = time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC)
	}

	// Write output to a temporary directory.
	dir, err := ioutil.TempDir("", "work-stats")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Get the corpus data (very slow on first try, uses cache after).
	var initOnce sync.Once
	initCorpus := func() {
		corpus, err = godata.Get(ctx)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Write out data on the user's activity on the Go project's GitHub issues.
	if *goIssuesFlag {
		initOnce.Do(initCorpus)
		goIssues, err := golang.Issues(corpus.GitHub(), *username, start)
		if err != nil {
			log.Fatal(err)
		}
		if err := write(dir, goIssues); err != nil {
			log.Fatal(err)
		}
	}

	// Write out data on the user's activity on the Go project's Gerrit code reviews.
	if *goChangelistsFlag {
		initOnce.Do(initCorpus)
		goCLs, err := golang.Changelists(corpus.Gerrit(), emails, start)
		if err != nil {
			log.Fatal(err)
		}
		if err := write(dir, goCLs); err != nil {
			log.Fatal(err)
		}
	}

	// Write out data on the user's activity on GitHub issues outside of the Go project.
	if *githubIssuesFlag {
		githubIssues, err := github.Issues(ctx, *username, start)
		if err != nil {
			log.Fatal(err)
		}
		if err := write(dir, githubIssues); err != nil {
			log.Fatal(err)
		}
	}
}

func write(outputDir string, outputFns map[string]func(writer *csv.Writer) error) error {
	var filenames []string
	for filename, fn := range outputFns {
		fullpath := filepath.Join(outputDir, fmt.Sprintf("%s.csv", filename))
		file, err := os.Create(fullpath)
		if err != nil {
			return err
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		if err := fn(writer); err != nil {
			return err
		}
		filenames = append(filenames, fullpath)
	}
	for _, filename := range filenames {
		fmt.Printf("Wrote output to %s\n", filename)
	}
	return nil
}

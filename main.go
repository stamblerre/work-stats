package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/stamblerre/work-stats/issues"
	"github.com/stamblerre/work-stats/reviews"
	"golang.org/x/build/maintner/godata"
)

var (
	username = flag.String("username", "", "GitHub username")
	email    = flag.String("email", "", "Gerrit email")
	since    = flag.String("since", "", "date since when to collect data")
)

func main() {
	flag.Parse()

	// Only the username flag is required.
	// If since is omitted, results are all time.
	// If repo is omitted, results are for all repos.
	if *username == "" {
		log.Fatalf("please provide a Github username")
	}
	if *email == "" {
		log.Fatalf("please provide your Gerrit email")
	}

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

	// Get the corpus data (very slow on first try, uses cache after).
	corpus, err := godata.Get(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	// Write out data on the user's activity on GitHub issues.
	issueStats, err := issues.Data(corpus.GitHub(), *username, start)
	if err != nil {
		log.Fatal(err)
	}
	filename, err := issues.Write(dir, issueStats)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("GitHub issues: wrote output to %s\n", filename)

	// Write out data on the user's activity on Gerrit code reviews.
	reviewStats, err := reviews.Data(corpus.Gerrit(), *email, start)
	if err != nil {
		log.Fatal(err)
	}
	filenames, err := reviews.Write(dir, reviewStats)
	if err != nil {
		log.Fatal(err)
	}
	for _, filename := range filenames {
		fmt.Printf("Gerrit reviews: wrote output to %s\n", filename)
	}
}

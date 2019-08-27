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
	"time"

	"github.com/stamblerre/work-stats/issues"
	"github.com/stamblerre/work-stats/reviews"
	"golang.org/x/build/maintner/godata"
)

var (
	username = flag.String("username", "", "GitHub username")
	email    = flag.String("email", "", "Gerrit email or emails, comma-separated")
	since    = flag.String("since", "", "date since when to collect data")
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
	filenames, err := write(dir, issueStats)
	if err != nil {
		log.Fatal(err)
	}
	for _, filename := range filenames {
		fmt.Printf("GitHub issues: Wrote output to %s\n", filename)
	}

	// Write out data on the user's activity on Gerrit code reviews.
	reviewStats, err := reviews.Data(corpus.Gerrit(), emails, start)
	if err != nil {
		log.Fatal(err)
	}
	filenames, err = write(dir, reviewStats)
	if err != nil {
		log.Fatal(err)
	}
	for _, filename := range filenames {
		fmt.Printf("Gerrit reviews: Wrote output to %s\n", filename)
	}
}

func write(outputDir string, outputFns map[string]func(writer *csv.Writer) error) ([]string, error) {
	var filenames []string
	for filename, fn := range outputFns {
		fullpath := filepath.Join(outputDir, fmt.Sprintf("%s.csv", filename))
		file, err := os.Create(fullpath)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		if err := fn(writer); err != nil {
			return nil, err
		}
		filenames = append(filenames, fullpath)
	}
	return filenames, nil
}

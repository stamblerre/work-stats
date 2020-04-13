package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/stamblerre/work-stats/golang"
	"golang.org/x/build/maintner/godata"
)

var (
	username = flag.String("username", "", "GitHub username")
	email    = flag.String("email", "", "Gerrit email or emails, comma-separated")

	// Optional flags.
	gerritFlag = flag.Bool("gerrit", true, "collect data on Go issues or changelists")
	gitHubFlag = flag.Bool("github", true, "collect data on GitHub issues")
)

func main() {
	flag.Parse()

	ctx := context.Background()

	// Username and email are required flags.
	// If since is omitted, results reflect all history.
	if *username == "" && *gitHubFlag {
		log.Fatal("Please provide a GitHub username.")
	}
	if *email == "" && *gerritFlag {
		log.Fatal("Please provide your Gerrit email.")
	}
	emails := strings.Split(*email, ",")

	// Assume that users will run the command on Fri, Sat, Sun, or Mon.
	// Look for the previous week.
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start
	for start.Weekday() != time.Monday {
		start = start.AddDate(0, 0, -1)
	}
	for end.Weekday() != time.Sunday && end.After(start) {
		end = end.AddDate(0, 0, -1)
	}
	end = end.Add(24 * time.Hour)

	corpus, err := godata.Get(ctx)
	if err != nil {
		log.Fatal(err)
	}
	authored, reviewed, err := golang.Changelists(corpus.Gerrit(), emails, start, end)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("---------------------------------")
	var b strings.Builder
	b.WriteString("## CLs Merged\n\n")
	for _, cl := range authored {
		if cl.Status != "merged" {
			continue
		}
		b.WriteString(fmt.Sprintf("* [CL %d](https://%s): %s\n", cl.Number, cl.Link, cl.Description))
	}
	b.WriteString("\n## CLs In Progress\n\n")
	for _, cl := range authored {
		if cl.Status == "merged" {
			continue
		}
		b.WriteString(fmt.Sprintf(" * [CL %d](https://%s): %s\n", cl.Number, cl.Link, cl.Description))
	}
	b.WriteString("\n## CLs Reviewed\n\n")
	for _, cl := range reviewed {
		b.WriteString(fmt.Sprintf("* [CL %d](https://%s): %s\n", cl.Number, cl.Link, cl.Description))
	}
	fmt.Println(b.String())
}

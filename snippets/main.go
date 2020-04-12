package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/stamblerre/work-stats/generic"
	"golang.org/x/build/maintner"
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
	merged, _, _, err := changelists(corpus.Gerrit(), emails, start, end)
	if err != nil {
		log.Fatal(err)
	}
	for k, v := range merged {
		log.Printf("K: %v, V: %v", k, v)
	}
}

func changelists(gerrit *maintner.Gerrit, emails []string, start, end time.Time) (merged, inProgress, reviewed map[string]*generic.Changelist, err error) {
	emailset := make(map[string]bool)
	for _, e := range emails {
		emailset[e] = true
	}
	merged = make(map[string]*generic.Changelist)
	inProgress = make(map[string]*generic.Changelist)
	reviewed = make(map[string]*generic.Changelist)
	if err := gerrit.ForeachProjectUnsorted(func(project *maintner.GerritProject) error {
		if err := project.ForeachCLUnsorted(func(cl *maintner.GerritCL) error {
			if cl.Owner() == nil || !emailset[cl.Owner().Email()] {
				return nil
			}
			var thisWeek bool
			for _, meta := range cl.Metas {
				log.Printf("CL: %v", meta.Commit.Author.String())
				if !emailset[meta.Commit.Author.Email()] {
					continue
				}
				if t := meta.Commit.CommitTime; t.After(start) && t.Before(end) {
					thisWeek = true
					break
				}
			}
			if !thisWeek {
				return nil
			}
			l := link(cl)
			log.Printf("HELLO: %v", l)
			merged[l] = &generic.Changelist{
				Link:        l,
				Author:      cl.Owner().Email(),
				Description: cl.Subject(),
				Repo:        project.Project(),
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, nil, nil, err
	}
	return merged, inProgress, reviewed, nil
}

func link(cl *maintner.GerritCL) string {
	return fmt.Sprintf("go-review.googlesource.com/c/%s/+/%v", cl.Project.Project(), cl.Number)
}

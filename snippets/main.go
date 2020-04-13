package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/stamblerre/work-stats/generic"
	"github.com/stamblerre/work-stats/github"
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

	start, end := generic.SnippetTimeRange()
	log.Printf("Generating weekly snippets for dates %s to %s", start.Format("01-02-2006"), end.Format("01-02-2006"))

	corpus, err := godata.Get(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var b strings.Builder
	if *gerritFlag {
		authored, reviewed, err := golang.Changelists(corpus.Gerrit(), emails, start, end)
		if err != nil {
			log.Fatal(err)
		}
		issues, err := golang.Issues(corpus.GitHub(), *username, start, end)
		if err != nil {
			log.Fatal(err)
		}
		var merged, inProgress []*generic.Changelist
		for _, cl := range authored {
			if cl.Status == generic.Merged {
				merged = append(merged, cl)
			} else {
				inProgress = append(inProgress, cl)
			}
		}
		if len(merged) > 0 {
			b.WriteString("## CLs Merged\n\n")
			for _, cl := range merged {
				b.WriteString(formatCL(cl))
			}
		}
		if len(inProgress) > 0 {
			b.WriteString("\n## CLs In Progress\n\n")
			for _, cl := range inProgress {
				b.WriteString(formatCL(cl))
			}
		}
		if len(reviewed) > 0 {
			b.WriteString("\n## CLs Reviewed\n\n")
			for _, cl := range reviewed {
				b.WriteString(formatCL(cl))
			}
		}
		if len(issues) > 0 {
			b.WriteString(fmt.Sprintf("\n### Commented on %v golang/go issues\n\n", len(issues)))
		}
	}

	if *gitHubFlag {
		authored, reviewed, issues, err := github.IssuesAndPRs(ctx, *username, start, end)
		if err != nil {
			log.Fatal(err)
		}
		var merged, inProgress []*generic.Changelist
		for _, pr := range authored {
			if pr.Status == generic.Merged {
				merged = append(merged, pr)
			} else {
				inProgress = append(inProgress, pr)
			}
		}
		if len(merged) > 0 {
			b.WriteString("## PRs Merged\n\n")
			for _, pr := range merged {
				b.WriteString(formatPR(pr))
			}
		}
		if len(inProgress) > 0 {
			b.WriteString("\n## PRs In Progress\n\n")
			for _, pr := range inProgress {
				b.WriteString(formatPR(pr))
			}
		}
		if len(reviewed) > 0 {
			b.WriteString("\n## PRs Reviewed\n\n")
			for _, pr := range reviewed {
				b.WriteString(formatPR(pr))
			}
		}
		if len(issues) > 0 {
			b.WriteString(fmt.Sprintf("\n### Commented on %v GitHub issues\n\n", len(issues)))
		}
	}
	fmt.Println(b.String())
}

func formatCL(cl *generic.Changelist) string {
	return fmt.Sprintf(" * [%d](https://%s): %s\n", cl.Number, cl.Link, cl.Description)
}

func formatPR(pr *generic.Changelist) string {
	return fmt.Sprintf("* [%s#%d](%s): %s\n", pr.Repo, pr.Number, pr.Link, pr.Description)
}

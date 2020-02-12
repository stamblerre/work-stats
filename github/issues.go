// Package github reports data on GitHub PRs and issues.
package github

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v28/github"
	"github.com/stamblerre/work-stats/generic"
	"golang.org/x/oauth2"
)

func IssuesAndPRs(ctx context.Context, username string, since time.Time) (map[string][][]string, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable is not configured")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token,
	})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	var issues []*generic.Issue
	var authoredPRs, reviewedPRs []*generic.Changelist

	var current, total int
	for i := 0; ; i++ {
		result, _, err := client.Search.Issues(ctx, fmt.Sprintf("involves:%v updated:>=%v", username, since.Format("2006-01-02")), &github.SearchOptions{
			ListOptions: github.ListOptions{
				Page:    i,
				PerPage: 100,
			},
		})
		if err != nil {
			return nil, err
		}
		for _, issue := range result.Issues {
			trimmed := strings.TrimPrefix(issue.GetRepositoryURL(), "https://api.github.com/repos/")
			split := strings.SplitN(trimmed, "/", 2)
			org, repo := split[0], split[1]
			// golang issues are tracker via the golang package.
			if org == "golang" {
				continue
			}
			// Only mark issues as opened if the user opened them since the specified date.
			opened := issue.GetUser().GetLogin() == username && issue.GetCreatedAt().After(since)
			if issue.IsPullRequest() {
				gc := &generic.Changelist{
					Repo:        fmt.Sprintf("%s/%s", org, repo),
					Description: issue.GetTitle(),
					Link:        issue.GetHTMLURL(),
					Author:      issue.GetUser().GetLogin(),
				}
				if opened {
					authoredPRs = append(authoredPRs, gc)
				} else {
					reviewedPRs = append(reviewedPRs, gc)
				}
				continue
			}
			events, _, err := client.Issues.ListIssueEvents(ctx, org, repo, issue.GetNumber(), nil)
			if err != nil {
				return nil, err
			}
			var closed bool
			for _, e := range events {
				if e.GetActor().GetLogin() != username {
					continue
				}
				if e.GetCreatedAt().Before(since) {
					continue
				}
				switch e.GetEvent() {
				case "closed":
					closed = true
				}
			}
			comments, _, err := client.Issues.ListComments(ctx, org, repo, issue.GetNumber(), nil)
			if err != nil {
				return nil, err
			}
			var numComments int
			for _, c := range comments {
				if c.GetUser().GetLogin() != username {
					continue
				}
				if c.GetCreatedAt().Before(since) {
					continue
				}
				numComments++
			}
			issues = append(issues, &generic.Issue{
				Repo:     fmt.Sprintf("%s/%s", org, repo),
				Title:    issue.GetTitle(),
				Link:     issue.GetHTMLURL(),
				Opened:   opened,
				Closed:   closed,
				Comments: numComments,
			})
		}
		total = result.GetTotal()
		current += len(result.Issues)
		if current >= total {
			break
		}
	}

	return map[string][][]string{
		"github-issues":       generic.IssuesToCells(issues),
		"github-prs-authored": generic.AuthoredChangelistsToCells(authoredPRs),
		"github-prs-reviewed": generic.ReviewedChangelistsToCells(reviewedPRs),
	}, nil
}

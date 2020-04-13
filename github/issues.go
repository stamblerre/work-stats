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

func CategorizeIssuesAndPRs(ctx context.Context, username string, start, end time.Time) (map[string][][]string, error) {
	authored, reviewed, issues, err := IssuesAndPRs(ctx, username, start, end)
	if err != nil {
		return nil, err
	}
	var genericIssues []*generic.Issue
	for _, i := range issues {
		genericIssues = append(genericIssues, i)
	}
	var authoredPRs, reviewedPRs []*generic.Changelist
	for _, pr := range authored {
		authoredPRs = append(authoredPRs, pr)
	}
	for _, pr := range reviewed {
		reviewedPRs = append(reviewedPRs, pr)
	}
	return map[string][][]string{
		"github-issues":       generic.IssuesToCells(genericIssues),
		"github-prs-authored": generic.AuthoredChangelistsToCells(authoredPRs),
		"github-prs-reviewed": generic.ReviewedChangelistsToCells(reviewedPRs),
	}, nil
}

func IssuesAndPRs(ctx context.Context, username string, start, end time.Time) (authored, reviewed map[string]*generic.Changelist, issues map[string]*generic.Issue, err error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, nil, nil, fmt.Errorf("GITHUB_TOKEN environment variable is not configured")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token,
	})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	issues = make(map[string]*generic.Issue)
	authored = make(map[string]*generic.Changelist)
	reviewed = make(map[string]*generic.Changelist)
	seen := make(map[string]struct{})

	var mostRecentIssue time.Time
	last := start
outer:
	for {
		var current int
		for i := 0; i < 10; i++ {
			result, _, err := client.Search.Issues(ctx, fmt.Sprintf("involves:%v -user:golang updated:%s..%s", username, last.Format("2006-01-02"), end.Format("2006-01-02")), &github.SearchOptions{
				ListOptions: github.ListOptions{
					Page:    i,
					PerPage: 100,
				},
				Sort:  "updated",
				Order: "asc",
			})
			if err != nil {
				return nil, nil, nil, err
			}
			for _, issue := range result.Issues {
				if _, ok := seen[issue.GetHTMLURL()]; ok {
					continue
				}
				seen[issue.GetHTMLURL()] = struct{}{}
				mostRecentIssue = *issue.UpdatedAt
				trimmed := strings.TrimPrefix(issue.GetRepositoryURL(), "https://api.github.com/repos/")
				split := strings.SplitN(trimmed, "/", 2)
				org, repo := split[0], split[1]
				// golang issues are tracker via the golang package.
				if org == "golang" {
					continue
				}
				// Only mark issues as opened if the user opened them since the specified date.
				opened := issue.GetUser().GetLogin() == username
				closed := issue.GetClosedBy() != nil
				if issue.IsPullRequest() {
					status := generic.Unknown
					if closed {
						status = generic.Merged
					}
					gc := &generic.Changelist{
						Repo:        fmt.Sprintf("%s/%s", org, repo),
						Description: issue.GetTitle(),
						Link:        issue.GetHTMLURL(),
						Author:      issue.GetUser().GetLogin(),
						Number:      issue.GetNumber(),
						Status:      status,
					}
					if opened {
						authored[issue.GetHTMLURL()] = gc
					} else {
						reviewed[issue.GetHTMLURL()] = gc
					}
					continue
				}
				comments, _, err := client.Issues.ListComments(ctx, org, repo, issue.GetNumber(), nil)
				if err != nil {
					return nil, nil, nil, err
				}
				var numComments int
				for _, c := range comments {
					if c.GetUser().GetLogin() != username {
						continue
					}
					if !inScope(c.GetCreatedAt(), start, end) {
						continue
					}
					numComments++
				}
				issues[issue.GetHTMLURL()] = &generic.Issue{
					Repo:     fmt.Sprintf("%s/%s", org, repo),
					Title:    issue.GetTitle(),
					Link:     issue.GetHTMLURL(),
					Opened:   opened,
					Closed:   closed,
					Comments: numComments,
				}
			}
			current += len(result.Issues)
			if current >= result.GetTotal() {
				break outer
			}
		}
		last = mostRecentIssue
	}
	return authored, reviewed, issues, nil
}

func inScope(t, start, end time.Time) bool {
	return t.After(start) && t.Before(end)
}

package github

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v28/github"
	"golang.org/x/oauth2"
)

type issueData struct {
	org, repo      string
	number         int
	opened, closed bool
	comments       int
}

func Issues(ctx context.Context, username string, since time.Time) (map[string]func(*csv.Writer) error, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable is not configured")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token,
	})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	stats := make(map[string]*issueData)

	// Get all non-golang/go issues.
	var current, total int
	for i := 0; ; i++ {
		result, _, err := client.Search.Issues(ctx, fmt.Sprintf("commenter:%v updated:>=%v", username, since.Format("2006-01-02")), &github.SearchOptions{
			Sort: "updated-dsc",
			ListOptions: github.ListOptions{
				Page:    i,
				PerPage: 100,
			},
		})
		if err != nil {
			return nil, err
		}
		for _, issue := range result.Issues {
			// Skip PRs for now.
			if issue.IsPullRequest() {
				continue
			}
			trimmed := strings.TrimPrefix(issue.GetRepositoryURL(), "https://api.github.com/repos/")
			split := strings.SplitN(trimmed, "/", 2)
			org, repo := split[0], split[1]
			// golang/go issues are tracker via the golang package.
			if org == "golang" && repo == "go" {
				continue
			}
			stats[issue.GetHTMLURL()] = &issueData{
				org:    org,
				repo:   repo,
				number: issue.GetNumber(),
				// Only mark issues as opened if the user opened them since the specified date.
				opened: issue.GetUser().GetLogin() == username && issue.GetCreatedAt().After(since),
			}
		}
		total = result.GetTotal()
		current += len(result.Issues)
		if current >= total {
			break
		}
	}
	for _, issue := range stats {
		events, _, err := client.Issues.ListIssueEvents(ctx, issue.org, issue.repo, issue.number, nil)
		if err != nil {
			return nil, err
		}
		for _, e := range events {
			if e.GetActor().GetLogin() != username {
				continue
			}
			if e.GetCreatedAt().Before(since) {
				continue
			}
			switch e.GetEvent() {
			case "closed":
				issue.closed = true
			}
		}
		comments, _, err := client.Issues.ListComments(ctx, issue.org, issue.repo, issue.number, nil)
		if err != nil {
			return nil, err
		}
		for _, c := range comments {
			if c.GetUser().GetLogin() != username {
				continue
			}
			if c.GetCreatedAt().Before(since) {
				continue
			}
			issue.comments++
		}
	}
	sorted := make([]string, 0, len(stats))
	for url := range stats {
		sorted = append(sorted, url)
	}
	sort.Strings(sorted)

	// TODO(rstambler): Add per-repo totals.
	return map[string]func(*csv.Writer) error{
		"github-issues": func(writer *csv.Writer) error {
			if err := writer.Write([]string{"Issue", "Opened", "Closed", "Number of Comments"}); err != nil {
				return err
			}
			var opened, closed, comments int
			for _, url := range sorted {
				data := stats[url]
				if data.opened {
					opened++
				}
				if data.closed {
					closed++
				}
				comments += data.comments
				if err := writer.Write([]string{
					url,
					strconv.FormatBool(data.opened),
					strconv.FormatBool(data.closed),
					fmt.Sprintf("%v", data.comments),
				}); err != nil {
					return err
				}
			}
			return writer.Write([]string{
				fmt.Sprintf("%v", len(stats)),
				fmt.Sprintf("%v", opened),
				fmt.Sprintf("%v", closed),
				fmt.Sprintf("%v", comments),
			})
		},
	}, nil
}

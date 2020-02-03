package github

import (
	"context"
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
	isPR           bool
}

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

	stats := make(map[string]*issueData)

	// Get all non-golang/go issues.
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
				isPR:   issue.IsPullRequest(),
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

	sortedPRs := make([]string, 0, len(stats))
	for url, data := range stats {
		if !data.isPR {
			continue
		}
		sortedPRs = append(sortedPRs, url)
	}

	var issuesCells [][]string
	{
		sorted := make([]string, 0, len(stats))
		for url, data := range stats {
			if data.isPR {
				continue
			}
			sorted = append(sorted, url)
		}
		sort.Strings(sorted)
		issuesCells = append(issuesCells, []string{"Issue", "Opened", "Closed", "Number of Comments"})
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
			issuesCells = append(issuesCells, []string{
				url,
				strconv.FormatBool(data.opened),
				strconv.FormatBool(data.closed),
				fmt.Sprintf("%v", data.comments),
			})
		}
		issuesCells = append(issuesCells, []string{
			fmt.Sprintf("%v", len(stats)),
			fmt.Sprintf("%v", opened),
			fmt.Sprintf("%v", closed),
			fmt.Sprintf("%v", comments),
		})
	}

	var authoredCells [][]string
	{
		authoredCells = append(authoredCells, []string{"Repo", "URL"})
		var total int
		for _, url := range sortedPRs {
			data := stats[url]
			// Skip any CLs reviewed.
			if !data.opened {
				continue
			}
			total++
			authoredCells = append(authoredCells, []string{
				fmt.Sprintf("%v/%v", data.org, data.repo),
				url,
			})
		}
		authoredCells = append(authoredCells, []string{
			"Total",
			fmt.Sprintf("%v", total),
		})
	}

	var reviewedCells [][]string
	{
		reviewedCells = append(reviewedCells, []string{"Repo", "URL", "Closed", "Number of comments"})
		var total, closed, comments int
		for _, url := range sortedPRs {
			data := stats[url]
			// SKip any CLs authored.
			if data.opened {
				continue
			}
			if data.closed {
				closed++
			}
			comments += data.comments
			total++
			reviewedCells = append(reviewedCells, []string{
				fmt.Sprintf("%v/%v", data.org, data.repo),
				url,
				strconv.FormatBool(data.closed),
				fmt.Sprintf("%v", data.comments),
			})
		}
		reviewedCells = append(reviewedCells, []string{
			"Total",
			fmt.Sprintf("%v", total),
			fmt.Sprintf("%v", closed),
			fmt.Sprintf("%v", comments),
		})
	}

	// TODO(rstambler): Add per-repo totals.
	return map[string][][]string{
		"github-issues":       issuesCells,
		"github-prs-authored": authoredCells,
		"github-prs-reviewed": reviewedCells,
	}, nil
}

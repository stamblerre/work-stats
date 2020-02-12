package golang

import (
	"fmt"
	"strings"
	"time"

	"github.com/stamblerre/work-stats/generic"
	"golang.org/x/build/maintner"
)

// Get some statistics on issues opened, closed, and commented on.
func Issues(github *maintner.GitHub, username string, start time.Time) (map[string][][]string, error) {
	stats := make(map[*maintner.GitHubIssue]*generic.Issue)

	github.ForeachRepo(func(repo *maintner.GitHubRepo) error {
		return repo.ForeachIssue(func(issue *maintner.GitHubIssue) error {
			maybeAddIssue := func() {
				if _, ok := stats[issue]; !ok {
					r := fmt.Sprintf("%s/%s", repo.ID().Owner, repo.ID().Repo)
					stats[issue] = &generic.Issue{
						Title:    issue.Title,
						Repo:     r,
						Link:     fmt.Sprintf("github.com/%s/issues/%v", r, issue.Number),
						Category: extractCategory(issue.Title),
					}
				}
			}
			// Check if the user opened the given issue.
			if issue.User != nil && issue.User.Login == username {
				if issue.Created.After(start) {
					maybeAddIssue()
					stats[issue].Opened = true
				}
			}
			// Check if the user closed the issue.
			issue.ForeachEvent(func(event *maintner.GitHubIssueEvent) error {
				if event.Actor != nil && event.Actor.Login == username {
					if event.Created.After(start) {
						switch event.Type {
						case "closed":
							maybeAddIssue()
							stats[issue].Closed = true
						}
					}

				}
				return nil
			})
			issue.ForeachComment(func(comment *maintner.GitHubComment) error {
				if comment.User != nil && comment.User.Login == username {
					if comment.Created.After(start) {
						maybeAddIssue()
						stats[issue].Comments++
					}
				}
				return nil
			})
			return nil
		})
	})
	var issues []*generic.Issue
	for _, issue := range stats {
		issues = append(issues, issue)
	}
	return map[string][][]string{
		"golang-issues": generic.IssuesToCells(issues),
	}, nil
}

func extractCategory(description string) string {
	split := strings.Split(description, ":")
	if len(split) > 1 {
		if !strings.Contains(split[0], " ") {
			return split[0]
		}
	}
	return ""
}

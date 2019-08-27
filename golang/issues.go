package golang

import (
	"encoding/csv"
	"fmt"
	"sort"
	"strconv"
	"time"

	"golang.org/x/build/maintner"
)

type commentData struct {
	opened, closed bool
	numComments    int
}

// Get some statistics on issues opened, closed, and commented on.
func Issues(github *maintner.GitHub, username string, start time.Time) (map[string]func(writer *csv.Writer) error, error) {
	stats := make(map[*maintner.GitHubIssue]*commentData)

	// Only use the golang/go repo, since we don't file issues elsewhere.
	repo := github.Repo("golang", "go")
	repo.ForeachIssue(func(issue *maintner.GitHubIssue) error {
		maybeAddIssue := func() {
			if _, ok := stats[issue]; !ok {
				stats[issue] = &commentData{}
			}
		}
		// Check if the user opened the given issue.
		if issue.User != nil && issue.User.Login == username {
			if issue.Created.After(start) {
				maybeAddIssue()
				stats[issue].opened = true
			}
		}
		// Check if the user closed the issue.
		issue.ForeachEvent(func(event *maintner.GitHubIssueEvent) error {
			if event.Actor != nil && event.Actor.Login == username {
				if event.Created.After(start) {
					switch event.Type {
					case "closed":
						maybeAddIssue()
						stats[issue].closed = true
					}
				}

			}
			return nil
		})
		issue.ForeachComment(func(comment *maintner.GitHubComment) error {
			if comment.User != nil && comment.User.Login == username {
				if comment.Created.After(start) {
					maybeAddIssue()
					stats[issue].numComments++
				}
			}
			return nil
		})
		return nil
	})
	return map[string]func(*csv.Writer) error{
		"golang-issues": func(writer *csv.Writer) error {
			var opened, closed, comments int
			if err := writer.Write([]string{"issue number", "opened", "closed", "num comments"}); err != nil {
				return err
			}
			var sorted []*maintner.GitHubIssue
			for issue := range stats {
				sorted = append(sorted, issue)
			}
			sort.Slice(sorted, func(i, j int) bool {
				return sorted[i].Created.Before(sorted[j].Created)
			})
			for _, issue := range sorted {
				data := stats[issue]
				if data.opened {
					opened++
				}
				if data.closed {
					closed++
				}
				comments += data.numComments
				record := []string{
					fmt.Sprintf("github.com/%s/%s/issues/%v", repo.ID().Owner, repo.ID().Repo, issue.Number),
					strconv.FormatBool(data.opened),
					strconv.FormatBool(data.closed),
					fmt.Sprintf("%v", data.numComments),
				}
				if err := writer.Write(record); err != nil {
					return err
				}
			}
			// Write out the totals.
			return writer.Write([]string{
				"Total",
				fmt.Sprintf("%v", opened),
				fmt.Sprintf("%v", closed),
				fmt.Sprintf("%v", comments),
			})
		},
	}, nil
}

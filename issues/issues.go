package issues

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"golang.org/x/build/maintner"
)

type commentData struct {
	opened, closed bool
	numComments    int
}

type IssueData struct {
	repo   *maintner.GitHubRepo
	issues map[*maintner.GitHubIssue]*commentData
}

// Get some statistics on issues opened, closed, and commented on.
func Data(github *maintner.GitHub, username string, start time.Time) (*IssueData, error) {
	stats := &IssueData{
		repo:   github.Repo("golang", "go"),
		issues: make(map[*maintner.GitHubIssue]*commentData),
	}

	// Only use the golang/go repo, since we don't file issues elsewhere.
	stats.repo.ForeachIssue(func(issue *maintner.GitHubIssue) error {
		addIssue := func() {
			if _, ok := stats.issues[issue]; !ok {
				stats.issues[issue] = &commentData{}
			}
		}
		// Check if the user opened the given issue.
		if issue.User != nil && issue.User.Login == username {
			if issue.Created.After(start) {
				addIssue()
				stats.issues[issue].opened = true
			}
		}
		// Check if the user closed the issue.
		issue.ForeachEvent(func(event *maintner.GitHubIssueEvent) error {
			if event.Actor != nil && event.Actor.Login == username {
				if event.Created.After(start) {
					switch event.Type {
					case "closed":
						addIssue()
						stats.issues[issue].closed = true
					}
				}

			}
			return nil
		})
		issue.ForeachComment(func(comment *maintner.GitHubComment) error {
			if comment.User != nil && comment.User.Login == username {
				if comment.Created.After(start) {
					addIssue()
					stats.issues[issue].numComments++
				}
			}
			return nil
		})
		return nil
	})
	return stats, nil
}

func Write(outputDir string, stats *IssueData) (string, error) {
	filename := filepath.Join(outputDir, "issues.csv")
	file, err := os.Create(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	var opened, closed, comments int
	if err := writer.Write([]string{"issue number", "opened", "closed", "num comments"}); err != nil {
		return "", err
	}
	var sorted []*maintner.GitHubIssue
	for issue := range stats.issues {
		sorted = append(sorted, issue)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Created.Before(sorted[j].Created)
	})
	for _, issue := range sorted {
		data := stats.issues[issue]
		if data.opened {
			opened++
		}
		if data.closed {
			closed++
		}
		comments += data.numComments
		record := []string{
			fmt.Sprintf("github.com/%s/%s/issues/%v", stats.repo.ID().Owner, stats.repo.ID().Repo, issue.Number),
			strconv.FormatBool(data.opened),
			strconv.FormatBool(data.closed),
			fmt.Sprintf("%v", data.numComments),
		}
		if err := writer.Write(record); err != nil {
			return "", err
		}
	}
	// Write out the totals.
	writer.Write([]string{
		"Total",
		fmt.Sprintf("%v", opened),
		fmt.Sprintf("%v", closed),
		fmt.Sprintf("%v", comments),
	})
	return filename, err
}

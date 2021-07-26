package golang

import (
	"fmt"
	"sort"
	"time"

	"github.com/stamblerre/work-stats/generic"
	"golang.org/x/build/maintner"
)

func Issues(github *maintner.GitHub, repository, username string, start, end time.Time) ([]*generic.Issue, error) {
	issuesMap := make(map[*maintner.GitHubIssue]*generic.Issue)

	if err := github.ForeachRepo(func(repo *maintner.GitHubRepo) error {
		if repository != "" && repo.ID().Repo != repository {
			return nil
		}
		return repo.ForeachIssue(func(issue *maintner.GitHubIssue) error {
			if issue.PullRequest {
				return nil
			}
			if issue.NotExist {
				return nil
			}
			maybeAddIssue := func() {
				if _, ok := issuesMap[issue]; !ok {
					var labels []string
					for _, label := range issue.Labels {
						labels = append(labels, label.Name)
					}
					repository := fmt.Sprintf("%s/%s", repo.ID().Owner, repo.ID().Repo)
					link := fmt.Sprintf("github.com/%s/issues/%v", repository, issue.Number)
					issuesMap[issue] = &generic.Issue{
						Title:  issue.Title,
						Repo:   repository,
						Link:   link,
						Labels: labels,
					}
				}
			}
			// If there is no username given, add the issue unconditionally.
			if username == "" {
				maybeAddIssue()
			}
			// Check if the user opened the given issue.
			if username == "" || (issue.User != nil && issue.User.Login == username) {
				if inScope(issue.Created, start, end) {
					maybeAddIssue()

					issuesMap[issue].OpenedBy = username
					issuesMap[issue].DateOpened = issue.Created
				}
			}
			// Check if the user closed the issue.
			if err := issue.ForeachEvent(func(event *maintner.GitHubIssueEvent) error {
				if username == "" || (event.Actor != nil && event.Actor.Login == username) {
					if inScope(event.Created, start, end) {
						switch event.Type {
						case "closed":
							maybeAddIssue()
							issuesMap[issue].DateClosed = issue.ClosedAt
							issuesMap[issue].ClosedBy = issue.User.Login
						case "reopened":
							if _, ok := issuesMap[issue]; ok {
								issuesMap[issue].DateClosed = time.Time{}
								issuesMap[issue].ClosedBy = ""
							}
						}
					}
				}
				return nil
			}); err != nil {
				return err
			}
			return issue.ForeachComment(func(comment *maintner.GitHubComment) error {
				if comment.User != nil && comment.User.Login == username {
					if inScope(comment.Created, start, end) {
						maybeAddIssue()
						issuesMap[issue].Comments++
					}
				}
				return nil
			})
		})
	}); err != nil {
		return nil, err
	}
	var issues []*generic.Issue
	for _, issue := range issuesMap {
		issues = append(issues, issue)
	}
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Link < issues[j].Link
	})
	return issues, nil
}

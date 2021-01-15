// Package generic represents generic issues and changelists.
package generic

import (
	"fmt"
	"sort"
	"strconv"
	"time"
)

type Issue struct {
	Link                   string
	Repo                   string
	Title                  string
	OpenedBy               string
	DateOpened, DateClosed time.Time
	Comments               int
	Category               string
	Labels                 []string
}

func (issue Issue) OpenedByUser(username string) bool {
	return issue.OpenedBy == username
}

func (issue Issue) Closed() bool {
	return !issue.DateClosed.IsZero()
}

type issueTotal struct {
	issues, comments, opened, closed int
}

func (t1 *issueTotal) asCells() []string {
	return []string{
		fmt.Sprint(t1.opened),
		fmt.Sprint(t1.closed),
		fmt.Sprint(t1.comments),
		fmt.Sprint(t1.issues),
	}
}

func (t1 *issueTotal) add(t2 *issueTotal) {
	t1.issues += t2.issues
	t1.comments += t2.comments
	t1.opened += t2.opened
	t1.closed += t2.closed
}

func IssuesToCells(username string, issues []*Issue) [][]string {
	// First, categorize issues by repository.
	repos := make(map[string][]*Issue)
	for _, issue := range issues {
		repos[issue.Repo] = append(repos[issue.Repo], issue)
	}
	var sortedRepos []string
	for repo := range repos {
		sortedRepos = append(sortedRepos, repo)
	}
	sort.Strings(sortedRepos)

	cells := append([][]string{}, []string{"Issue Number", "Description", "Opened", "Closed", "Number of Comments", "Total Issues"})
	grandTotal := &issueTotal{}
	for _, repo := range sortedRepos {
		categories := make(map[string][]*Issue)
		for _, issue := range repos[repo] {
			categories[issue.Category] = append(categories[issue.Category], issue)
		}
		var sortedCategories []string
		for category := range categories {
			sortedCategories = append(sortedCategories, category)
		}
		sort.Strings(sortedCategories)

		repoTotal := &issueTotal{}
		for _, category := range sortedCategories {
			issues := categories[category]
			sort.Slice(issues, func(i, j int) bool {
				return issues[i].Link < issues[j].Link
			})
			categoryTotal := &issueTotal{
				issues: len(issues),
			}
			for _, issue := range issues {
				opened := issue.OpenedByUser(username)
				if opened {
					categoryTotal.opened++
				}
				if issue.Closed() {
					categoryTotal.closed++
				}
				categoryTotal.comments += issue.Comments
				cells = append(cells, []string{
					issue.Link,
					truncate(issue.Title),
					strconv.FormatBool(opened),
					strconv.FormatBool(issue.Closed()),
					strconv.FormatInt(int64(issue.Comments), 10),
				})
			}
			if len(sortedCategories) > 1 {
				cells = append(cells, append([]string{"", category}, categoryTotal.asCells()...))
			}
			repoTotal.add(categoryTotal)
		}
		// Only add the subtotal if there are multiple repos.
		if len(repos) > 1 {
			cells = append(cells, append([]string{"Subtotal", repo}, repoTotal.asCells()...))
		}
		grandTotal.add(repoTotal)
	}
	cells = append(cells, append([]string{"Total", ""}, grandTotal.asCells()...))
	return cells
}

// Package generic represents generic issues and changelists.
package generic

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	rsheets "github.com/stamblerre/sheets"
)

type Issue struct {
	Number                 int
	Link                   string
	Repo                   string
	Title                  string
	OpenedBy, ClosedBy     string
	DateOpened, DateClosed time.Time
	Comments               int
	Labels                 []string
	Transferred            bool
	Milestone              string
}

func (issue Issue) Category() string {
	return extractCategory(issue.Title)
}

func (issue Issue) OpenedByUser(username string) bool {
	return issue.OpenedBy == username
}

func (issue Issue) ClosedByUser(username string) bool {
	return issue.ClosedBy == username
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

func IssuesToCells(username string, issues []*Issue) []*rsheets.Row {
	if len(issues) == 0 {
		return nil
	}
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

	cells := append([]*rsheets.Row{}, &rsheets.Row{Cells: []*rsheets.Cell{
		{Text: "Issue Number"},
		{Text: "Description"},
		{Text: "Opened"},
		{Text: "Closed"},
		{Text: "Number of Comments"},
		{Text: "Total Issues"},
	}})
	grandTotal := &issueTotal{}
	for _, repo := range sortedRepos {
		categories := make(map[string][]*Issue)
		for _, issue := range repos[repo] {
			categories[issue.Category()] = append(categories[issue.Category()], issue)
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
				closed := issue.ClosedByUser(username)
				if closed {
					categoryTotal.closed++
				}
				categoryTotal.comments += issue.Comments
				cells = append(cells, &rsheets.Row{
					Cells: []*rsheets.Cell{
						{Text: issue.Link, Hyperlink: issue.Link},
						{Text: truncate(issue.Title)},
						{Text: strconv.FormatBool(opened)},
						{Text: strconv.FormatBool(closed)},
						{Text: strconv.FormatInt(int64(issue.Comments), 10)},
					}})
			}
			if len(sortedCategories) > 1 {
				cells = append(cells, rsheets.TotalRow(append([]string{"", category}, categoryTotal.asCells()...)...))
			}
			repoTotal.add(categoryTotal)
		}
		// Only add the subtotal if there are multiple repos.
		if len(repos) > 1 {
			cells = append(cells, rsheets.TotalRow(append([]string{"Subtotal", repo}, repoTotal.asCells()...)...))
		}
		grandTotal.add(repoTotal)
	}
	cells = append(cells, rsheets.TotalRow(append([]string{"Total", ""}, grandTotal.asCells()...)...))
	return cells
}

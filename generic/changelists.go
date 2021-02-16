package generic

import (
	"fmt"
	"sort"
	"time"
)

type Changelist struct {
	Number      int
	Link        string
	Description string
	Branch      string
	Author      string
	Repo        string
	Category    string
	Status      ChangelistStatus
	MergedAt    time.Time
}

type ChangelistStatus int

const (
	Merged = ChangelistStatus(iota)
	Abandoned
	New
	Draft
	Unknown
)

type category struct {
	branch string
	desc   string
}

func (c category) String() string {
	return c.branch + ": " + c.desc
}

func AuthoredChangelistsToCells(cls []*Changelist) [][]string {
	repos := make(map[string][]*Changelist)
	for _, cl := range cls {
		repos[cl.Repo] = append(repos[cl.Repo], cl)
	}
	var sortedRepos []string
	for repo := range repos {
		sortedRepos = append(sortedRepos, repo)
	}
	sort.Strings(sortedRepos)

	cells := [][]string{{"CL", "Description"}}
	for _, repo := range sortedRepos {
		cls := repos[repo]
		categories := make(map[category][]*Changelist)
		for _, cl := range cls {
			c := category{
				branch: cl.Branch,
				desc:   cl.Category,
			}
			categories[c] = append(categories[c], cl)
		}
		var sortedCategories []category
		for category := range categories {
			sortedCategories = append(sortedCategories, category)
		}
		sort.Slice(sortedCategories, func(i, j int) bool {
			if sortedCategories[i].branch == sortedCategories[j].branch {
				return sortedCategories[i].desc < sortedCategories[j].desc
			}
			return sortedCategories[i].branch < sortedCategories[j].branch
		})
		for _, category := range sortedCategories {
			cls := categories[category]
			sort.SliceStable(cls, func(i, j int) bool {
				return cls[i].Link < cls[j].Link
			})
			for _, cl := range cls {
				cells = append(cells, []string{cl.Link, truncate(cl.Description)})
			}
			// Only add subtotals for categories only if they are legitimate.
			if len(sortedCategories) > 1 {
				cells = append(cells, []string{"", category.String(), fmt.Sprint(len(cls))})
			}
		}
		cells = append(cells, []string{"Subtotal", repo, fmt.Sprint(len(repos[repo]))})
	}
	cells = append(cells, []string{"Total", "", fmt.Sprintf("%v", len(cls))})
	return cells
}

func ReviewedChangelistsToCells(cls []*Changelist) [][]string {
	repos := make(map[string][]*Changelist)
	for _, cl := range cls {
		repos[cl.Repo] = append(repos[cl.Repo], cl)
	}
	var sortedRepos []string
	for repo := range repos {
		sortedRepos = append(sortedRepos, repo)
	}
	sort.Strings(sortedRepos)

	cells := [][]string{{"CL", "Description"}}
	for _, repo := range sortedRepos {
		authors := make(map[string][]*Changelist)
		for _, cl := range repos[repo] {
			authors[cl.Author] = append(authors[cl.Author], cl)
		}
		var sortedAuthors []string
		for author := range authors {
			sortedAuthors = append(sortedAuthors, author)
		}
		sort.Strings(sortedAuthors)

		for _, author := range sortedAuthors {
			cls := authors[author]
			sort.SliceStable(cls, func(i, j int) bool {
				return cls[i].Link < cls[j].Link
			})
			for _, cl := range cls {
				cells = append(cells, []string{cl.Link, truncate(cl.Description)})
			}
			cells = append(cells, []string{"", author, fmt.Sprint(len(cls))})
		}
		if len(repos) > 1 {
			cells = append(cells, []string{"Subtotal", repo, fmt.Sprint(len(repos[repo]))})
		}
	}
	cells = append(cells, []string{"Total", "", fmt.Sprint(len(cls))})
	return cells
}

func truncate(x string) string {
	if len(x) > 80 {
		return x[:80]
	}
	return x
}

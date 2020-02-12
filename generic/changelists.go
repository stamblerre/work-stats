package generic

import (
	"fmt"
	"sort"
)

type Changelist struct {
	Link        string
	Description string
	Author      string
	Repo        string

	category string
}

func AuthoredChangelistsToCells(cls []*Changelist) [][]string {
	repos := make(map[string][]*Changelist)
	for _, cl := range cls {
		repos[cl.Repo] = append(repos[cl.Repo], cl)
		cl.category = extractCategory(cl.Repo, cl.Description)
	}
	var sortedRepos []string
	for repo := range repos {
		sortedRepos = append(sortedRepos, repo)
	}
	sort.Strings(sortedRepos)

	cells := [][]string{{"CL", "Description"}}
	for _, repo := range sortedRepos {
		cls := repos[repo]
		categories := make(map[string][]*Changelist)
		for _, cl := range cls {
			categories[cl.category] = append(categories[cl.category], cl)
		}
		var sortedCategories []string
		for category := range categories {
			sortedCategories = append(sortedCategories, category)
		}
		sort.Strings(sortedCategories)
		for _, category := range sortedCategories {
			cls := categories[category]
			sort.SliceStable(cls, func(i, j int) bool {
				return cls[i].Link < cls[j].Link
			})
			for _, cl := range cls {
				cells = append(cells, []string{cl.Link, cl.Description})
			}
			// Only add subtotals for categories only if they are legitimate.
			if len(sortedCategories) > 1 {
				cells = append(cells, []string{"Subtotal", "", fmt.Sprint(len(cls))})
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
		for _, cl := range cls {
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
				cells = append(cells, []string{cl.Link, cl.Description})
			}
			cells = append(cells, []string{"Subtotal", author, fmt.Sprint(len(cls))})
		}
		cells = append(cells, []string{"Subtotal", repo, fmt.Sprint(len(repos[repo]))})
	}
	if len(repos) > 1 {
		cells = append(cells, []string{"Total", "", fmt.Sprint(len(cls))})
	}
	return cells
}

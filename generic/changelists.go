package generic

import (
	"fmt"
	"image/color"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/stamblerre/sheets"
)

type Changelist struct {
	Number           int
	Link             string
	Subject          string
	Message          string
	Comments         []string
	Branch           string
	Author           string
	Repo             string
	Status           ChangelistStatus
	MergedAt         time.Time
	AssociatedIssues []*Issue
	AffectedFiles    []string
}

type ChangelistStatus int

const (
	Abandoned = ChangelistStatus(iota)
	Draft
	New
	Merged
	Unknown
)

func (status ChangelistStatus) String() string {
	switch status {
	case Merged:
		return "merged"
	case Abandoned:
		return "abandoned"
	case New:
		return "new"
	case Draft:
		return "draft"
	default:
		return "unknown"
	}
}

func (cl *Changelist) Category() string {
	if category := extractCategory(cl.Subject); category != "" {
		return category
	}
	// No category in the CL description. Check the affected files.
	// Determine the longest and most popular parent directory and choose that
	// as the category.
	directories := map[string]int{}
	for _, filename := range cl.AffectedFiles {
		dir := filepath.Dir(filename)
		for dir != filepath.Dir(dir) {
			directories[dir]++
			dir = filepath.Dir(dir)
		}
	}
	var popularDir string
	var popularCount int
	for dir, count := range directories {
		if count < popularCount {
			continue
		}
		if count > popularCount || len(dir) > len(popularDir) {
			popularCount = count
			popularDir = dir
		}
	}
	return popularDir
}

type category struct {
	branch string
	desc   string
}

func (c category) String() string {
	if c.branch == "" {
		return c.desc
	}
	return c.branch + ": " + c.desc
}

func AuthoredChangelistsToCells(cls []*Changelist) []*sheets.Row {
	if len(cls) == 0 {
		return nil
	}
	repos := make(map[string][]*Changelist)
	for _, cl := range cls {
		repos[cl.Repo] = append(repos[cl.Repo], cl)
	}
	var sortedRepos []string
	for repo := range repos {
		sortedRepos = append(sortedRepos, repo)
	}
	sort.Strings(sortedRepos)

	sheet := []*sheets.Row{{
		Cells: []*sheets.Cell{
			{Text: "CL"},
			{Text: "Description"},
		},
		BoldText: true,
	}}
	for _, repo := range sortedRepos {
		cls := repos[repo]
		categories := make(map[category][]*Changelist)
		for _, cl := range cls {
			c := category{
				branch: cl.Branch,
				desc:   cl.Category(),
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
				var yellow color.Color
				if cl.Status != Merged {
					yellow = sheets.PaleYellow()
				}
				sheet = append(sheet, &sheets.Row{
					Cells: []*sheets.Cell{
						{Text: cl.Link, Hyperlink: cl.Link},
						{Text: truncate(cl.Subject)},
						{Text: ""},
					},
					Color: yellow,
				})
			}
			// Only add subtotals for categories only if they are legitimate.
			if len(sortedCategories) > 1 {
				sheet = append(sheet, sheets.TotalRow("", category.String(), fmt.Sprint(len(cls))))
			}
		}
		sheet = append(sheet, sheets.TotalRow("Subtotal", repo, fmt.Sprint(len(repos[repo]))))
	}
	sheet = append(sheet, sheets.TotalRow("Total", "", fmt.Sprintf("%v", len(cls))))
	return sheet
}

func ReviewedChangelistsToCells(cls []*Changelist) []*sheets.Row {
	if len(cls) == 0 {
		return nil
	}
	repos := make(map[string][]*Changelist)
	for _, cl := range cls {
		repos[cl.Repo] = append(repos[cl.Repo], cl)
	}
	var sortedRepos []string
	for repo := range repos {
		sortedRepos = append(sortedRepos, repo)
	}
	sort.Strings(sortedRepos)

	cells := []*sheets.Row{{
		Cells: []*sheets.Cell{
			{Text: "CL"},
			{Text: "Description"},
		},
		BoldText: true,
	}}
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
				cells = append(cells, &sheets.Row{Cells: []*sheets.Cell{
					{Text: cl.Link, Hyperlink: cl.Link},
					{Text: truncate(cl.Subject)},
					{Text: ""},
				}})
			}
			cells = append(cells, sheets.TotalRow("", author, fmt.Sprint(len(cls))))
		}
		if len(repos) > 1 {
			cells = append(cells, sheets.TotalRow("Subtotal", repo, fmt.Sprint(len(repos[repo]))))
		}
	}
	cells = append(cells, sheets.TotalRow("Total", "", fmt.Sprint(len(cls))))
	return cells
}

func truncate(x string) string {
	if len(x) > 80 {
		return x[:80]
	}
	return x
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

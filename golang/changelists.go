package golang

import (
	"encoding/csv"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/build/maintner"
)

// Get some statistics on issues opened, closed, and commented on.
func Changelists(gerrit *maintner.Gerrit, emails []string, start time.Time) (map[string]func(writer *csv.Writer) error, error) {
	authored := make(map[*maintner.GerritCL]struct{})
	reviewed := make(map[*maintner.GerritCL]struct{})
	emailset := make(map[string]bool)
	for _, e := range emails {
		emailset[e] = true
	}
	ownerIDs := make(map[int]bool)
	if err := gerrit.ForeachProjectUnsorted(func(project *maintner.GerritProject) error {
		// First, collect all CLs authored by the user.
		project.ForeachCLUnsorted(func(cl *maintner.GerritCL) error {
			if cl.Owner() != nil && emailset[cl.Owner().Email()] {
				if cl.Branch() == "master" {
					ownerIDs[cl.OwnerID()] = true
				}
				if cl.Status == "merged" {
					if cl.Created.After(start) {
						authored[cl] = struct{}{}
					}
				}
			}
			return nil
		})
		return nil
	}); err != nil {
		return nil, err
	}
	if len(ownerIDs) == 0 {
		return nil, errors.Errorf("unable to collect review data, user has never authored a CL, so the reviewer ID cannot be matched")
	}
	if err := gerrit.ForeachProjectUnsorted(func(project *maintner.GerritProject) error {
		// We have to do this call separately, since we have to make sure that the owner ID has been set correctly.
		return project.ForeachCLUnsorted(func(cl *maintner.GerritCL) error {
			// If it was the user owns the CL, they cannot be its reviewer.
			if cl.Owner() != nil && emailset[cl.Owner().Email()] {
				return nil
			}
			// If the user reviewed the CL.
			for _, msg := range cl.Messages {
				// If the user's email is actually tracked.
				// Not sure why this happens for some people, but not others.
				if msg.Author != nil && emailset[msg.Author.Email()] {
					if msg.Date.After(start) {
						reviewed[cl] = struct{}{}
						return nil
					}
				}
				if strings.HasPrefix(msg.Author.Name(), "Gerrit User") {
					split := strings.Split(msg.Author.Name(), " ")
					if len(split) == 3 {
						id, err := strconv.ParseInt(split[2], 10, 64)
						if err != nil {
							log.Fatal(err)
						}
						if ownerIDs[int(id)] {
							if msg.Date.After(start) {
								reviewed[cl] = struct{}{}
								return nil
							}
						}
					}
				}
			}
			return nil
		})
	}); err != nil {
		return nil, err
	}
	return map[string]func(*csv.Writer) error{
		"golang-authored": func(writer *csv.Writer) error {
			var sorted []*maintner.GerritCL
			for cl := range authored {
				sorted = append(sorted, cl)
			}
			sort.Slice(sorted, func(i, j int) bool {
				return sorted[i].Created.Before(sorted[j].Created)
			})
			writer.Write([]string{"CL", "Description"})
			for _, cl := range sorted {
				writer.Write([]string{
					// TODO: Technically should insert the -review into cl.Project.Server().
					fmt.Sprintf("go-review.googlesource.com/c/%s/+/%v", cl.Project.Project(), cl.Number),
					cl.Subject(),
				})
			}
			return writer.Write([]string{"Total", fmt.Sprintf("%v", len(authored))})
		},
		"golang-reviewed": func(writer *csv.Writer) error {
			var sorted []*maintner.GerritCL
			for cl := range reviewed {
				sorted = append(sorted, cl)
			}
			sort.Slice(sorted, func(i, j int) bool {
				return sorted[i].Created.Before(sorted[j].Created)
			})
			writer.Write([]string{"CL", "Author", "Description"})
			for _, cl := range sorted {
				writer.Write([]string{
					// TODO: Technically should insert the -review into cl.Project.Server().
					fmt.Sprintf("go-review.googlesource.com/c/%s/+/%v", cl.Project.Project(), cl.Number),
					cl.Owner().Email(),
					cl.Subject(),
				})
			}
			return writer.Write([]string{"Total", fmt.Sprintf("%v", len(reviewed))})
		},
	}, nil
}

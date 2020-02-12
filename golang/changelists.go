// Package golang reports Go contributions and issues.
package golang

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stamblerre/work-stats/generic"
	"golang.org/x/build/maintner"
)

// Get some statistics on issues opened, closed, and commented on.
func Changelists(gerrit *maintner.Gerrit, emails []string, start time.Time) (map[string][][]string, error) {
	var authored, reviewed []*generic.Changelist
	emailset := make(map[string]bool)
	for _, e := range emails {
		emailset[e] = true
	}
	type ownerKey struct {
		project *maintner.GerritProject
		id      int
	}
	ownerIDs := make(map[ownerKey]bool)
	if err := gerrit.ForeachProjectUnsorted(func(project *maintner.GerritProject) error {
		// First, collect all CLs authored by the user.
		project.ForeachCLUnsorted(func(cl *maintner.GerritCL) error {
			if cl.Owner() != nil && emailset[cl.Owner().Email()] {
				if cl.Branch() == "master" && cl.OwnerID() != -1 {
					ownerIDs[ownerKey{project, cl.OwnerID()}] = true
				}
				if cl.Status == "merged" {
					if cl.Created.After(start) {
						gcl := &generic.Changelist{
							Link:        fmt.Sprintf("go-review.googlesource.com/c/%s/+/%v", cl.Project.Project(), cl.Number),
							Author:      cl.Owner().Email(),
							Description: cl.Subject(),
							Repo:        project.Project(),
						}
						authored = append(authored, gcl)
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
						gcl := &generic.Changelist{
							Link:        fmt.Sprintf("go-review.googlesource.com/c/%s/+/%v", cl.Project.Project(), cl.Number),
							Author:      cl.Owner().Email(),
							Description: cl.Subject(),
							Repo:        project.Project(),
						}
						reviewed = append(reviewed, gcl)
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
						if ownerIDs[ownerKey{project, int(id)}] {
							if msg.Date.After(start) {
								gcl := &generic.Changelist{
									Link:        fmt.Sprintf("go-review.googlesource.com/c/%s/+/%v", cl.Project.Project(), cl.Number),
									Author:      cl.Owner().Email(),
									Description: cl.Subject(),
									Repo:        project.Project(),
								}
								reviewed = append(reviewed, gcl)
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

	return map[string][][]string{
		"golang-authored": generic.AuthoredChangelistsToCells(authored),
		"golang-reviewed": generic.AuthoredChangelistsToCells(reviewed),
	}, nil
}

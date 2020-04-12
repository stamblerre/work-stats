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
	authored, _, reviewed, err := changelists(gerrit, emails, start, time.Now())
	if err != nil {
		return nil, err
	}
	var authoredCLs, reviewedCLs []*generic.Changelist
	for _, cl := range authored {
		authoredCLs = append(authoredCLs, cl)
	}
	for _, cl := range reviewed {
		reviewedCLs = append(reviewedCLs, cl)
	}
	return map[string][][]string{
		"golang-authored": generic.AuthoredChangelistsToCells(authoredCLs),
		"golang-reviewed": generic.ReviewedChangelistsToCells(reviewedCLs),
	}, nil
}

func ownerIDsForEmails(gerrit *maintner.Gerrit, emailset map[string]bool) (map[*maintner.GerritProject]int, error) {
	ownerIDs := make(map[*maintner.GerritProject]int)
	err := gerrit.ForeachProjectUnsorted(func(project *maintner.GerritProject) error {
		return project.ForeachCLUnsorted(func(cl *maintner.GerritCL) error {
			if cl.Owner() == nil || !emailset[cl.Owner().Email()] {
				return nil
			}
			if cl.Status != "merged" {
				return nil
			}
			// TODO(rstambler): Owner IDs change between branches. Support non-master branches.
			if cl.Branch() == "master" && cl.OwnerID() != -1 {
				if id, ok := ownerIDs[project]; ok && id != cl.OwnerID() {
					// The CL could be a cherry-pick from internal Gerrit. If so, skip it.
					if strings.HasPrefix(cl.Footer("Reviewed-on:"), "https://team-review.git.corp.google.com/") {
						return nil
					}
					log.Printf("Conflicting owner IDs (have %v, got %v) caused by %v. Ignoring that CL, please file an issue if you were involved in the CL.", id, cl.OwnerID(), link(cl))
				} else {
					ownerIDs[project] = cl.OwnerID()
				}
			}
		})
	})
	return ownerIDs, err
}

func changelists(gerrit *maintner.Gerrit, emails []string, start, end time.Time) (merged, inProgress, reviewed map[string]*generic.Changelist, err error) {
	emailset := make(map[string]bool)
	for _, e := range emails {
		emailset[e] = true
	}
	merged = make(map[string]*generic.Changelist)
	inProgress = make(map[string]*generic.Changelist)
	reviewed = make(map[string]*generic.Changelist)
	if err := gerrit.ForeachProjectUnsorted(func(project *maintner.GerritProject) error {
		// First, collect all CLs authored by the user.
		project.ForeachCLUnsorted(func(cl *maintner.GerritCL) error {
			if cl.Owner() == nil || !emailset[cl.Owner().Email()] {
				return nil
			}
			if cl.Status != "merged" {
				return nil
			}
			// TODO(rstambler): Owner IDs change between branches. Support non-master branches.
			if cl.Branch() == "master" && cl.OwnerID() != -1 {
				if id, ok := ownerIDs[project]; ok && id != cl.OwnerID() {
					// The CL could be a cherry-pick from internal Gerrit. If so, skip it.
					if strings.HasPrefix(cl.Footer("Reviewed-on:"), "https://team-review.git.corp.google.com/") {
						return nil
					}
					log.Printf("Conflicting owner IDs (have %v, got %v) caused by %v. Ignoring that CL, please file an issue if you were involved in the CL.", id, cl.OwnerID(), link(cl))
				} else {
					ownerIDs[project] = cl.OwnerID()
				}
			}
			if cl.Created.Before(start) {
				return nil
			}
			l := link(cl)
			merged[l] = &generic.Changelist{
				Link:        l,
				Author:      cl.Owner().Email(),
				Description: cl.Subject(),
				Repo:        project.Project(),
				Category:    extractCategory(cl.Subject()),
			}
			return nil
		})
		return nil
	}); err != nil {
		return nil, nil, nil, err
	}
	if len(ownerIDs) == 0 {
		return nil, nil, nil, errors.Errorf("unable to collect review data, user has never authored a CL, so the reviewer ID cannot be matched")
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
				if msg.Date.Before(start) {
					continue
				}
				if msg.Author == nil {
					continue
				}
				// If the user's email is not actually tracked.
				// Not sure why this happens for some people, but not others.
				if strings.HasPrefix(msg.Author.Name(), "Gerrit User") {
					split := strings.Split(msg.Author.Name(), " ")
					if len(split) != 3 {
						continue
					}
					id, err := strconv.ParseInt(split[2], 10, 64)
					if err != nil {
						log.Fatal(err)
					}
					if ownerIDs[project] != int(id) {
						continue
					}
				} else if !emailset[msg.Author.Email()] {
					continue
				}
				l := link(cl)
				reviewed[l] = &generic.Changelist{
					Link:        l,
					Author:      cl.Owner().Email(),
					Description: cl.Subject(),
					Repo:        project.Project(),
					Category:    extractCategory(cl.Subject()),
				}
				return nil
			}
			return nil
		})
	}); err != nil {
		return nil, nil, nil, err
	}
	return merged, inProgress, reviewed, nil
}

func link(cl *maintner.GerritCL) string {
	return fmt.Sprintf("go-review.googlesource.com/c/%s/+/%v", cl.Project.Project(), cl.Number)
}

// Package golang reports Go contributions and issues.
package golang

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/stamblerre/work-stats/generic"
	"golang.org/x/build/maintner"
)

// Get some statistics on issues opened, closed, and commented on.
func CategorizeChangelists(gerrit *maintner.Gerrit, emails []string, start, end time.Time) (map[string][][]string, error) {
	authored, reviewed, err := Changelists(gerrit, emails, start, end)
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

func Changelists(gerrit *maintner.Gerrit, emails []string, start, end time.Time) (authored, reviewed map[string]*generic.Changelist, err error) {
	emailset := make(map[string]bool)
	for _, e := range emails {
		emailset[e] = true
	}
	authored = make(map[string]*generic.Changelist)
	reviewed = make(map[string]*generic.Changelist)
	ownerIDs, err := OwnerIDs(gerrit, emailset)
	if err != nil {
		return nil, nil, err
	}
	if err := gerrit.ForeachProjectUnsorted(func(project *maintner.GerritProject) error {
		// First, collect all CLs authored by the user.
		project.ForeachCLUnsorted(func(cl *maintner.GerritCL) error {
			if cl.Owner() == nil || !emailset[cl.Owner().Email()] {
				return nil
			}
			key := key(cl)
			var match bool
			for _, meta := range cl.Metas {
				if cl.Status == "abandoned" {
					continue
				}
				if !inScope(cl.Commit.AuthorTime, start, end) {
					continue
				}
				id := personToId(meta.Commit.Author)
				if ownerID := ownerIDs[key]; ownerID == id {
					match = true
					break
				}
			}
			if !match {
				return nil
			}
			l := link(cl)
			authored[l] = &generic.Changelist{
				Number:      int(cl.Number),
				Link:        l,
				Author:      cl.Owner().Email(),
				Description: cl.Subject(),
				Repo:        project.Project(),
				Category:    extractCategory(cl.Subject()),
				Status:      cl.Status,
			}
			return nil
		})
		return nil
	}); err != nil {
		return nil, nil, err
	}
	if err := gerrit.ForeachProjectUnsorted(func(project *maintner.GerritProject) error {
		// We have to do this call separately, since we have to make sure that the owner ID has been set correctly.
		return project.ForeachCLUnsorted(func(cl *maintner.GerritCL) error {
			// If it was the user owns the CL, they cannot be its reviewer.
			if cl.Owner() != nil && emailset[cl.Owner().Email()] {
				return nil
			}
			if cl.Status == "abandoned" {
				return nil
			}
			key := key(cl)
			var match bool
			for _, msg := range cl.Messages {
				if !inScope(msg.Date, start, end) {
					continue
				}
				if msg.Author == nil {
					continue
				}
				// If the user's email is not actually tracked.
				// Not sure why this happens for some people, but not others.
				if id := personToId(msg.Author); ownerIDs[key] == int(id) {
					match = true
					break
				} else if emailset[msg.Author.Email()] {
					match = true
					break
				}
			}
			if !match {
				return nil
			}
			l := link(cl)
			reviewed[l] = &generic.Changelist{
				Number:      int(cl.Number),
				Link:        l,
				Author:      cl.Owner().Email(),
				Description: cl.Subject(),
				Repo:        project.Project(),
				Category:    extractCategory(cl.Subject()),
				Status:      cl.Status,
			}
			return nil
		})
	}); err != nil {
		return nil, nil, err
	}
	return authored, reviewed, nil
}

type GerritIdKey struct {
	project, branch, status string
}

func OwnerIDs(gerrit *maintner.Gerrit, emailset map[string]bool) (map[GerritIdKey]int, error) {
	ownerIDs := make(map[GerritIdKey]int)
	err := gerrit.ForeachProjectUnsorted(func(project *maintner.GerritProject) error {
		return project.ForeachCLUnsorted(func(cl *maintner.GerritCL) error {
			if cl.Owner() == nil || !emailset[cl.Owner().Email()] {
				return nil
			}
			// Skip PRs imported as CLs.
			// These are complicated and probably should be handled separately.
			if cl.OwnerID() == gerritbotID {
				return nil
			}
			k := key(cl)
			if id, ok := ownerIDs[k]; !ok {
				ownerIDs[k] = cl.OwnerID()
			} else if id != cl.OwnerID() {
				// The CL could be a cherry-pick from internal Gerrit. If so, skip it.
				if strings.HasPrefix(cl.Footer("Reviewed-on:"), "https://team-review.git.corp.google.com/") {
					return nil
				}
				log.Printf("Conflicting owner IDs (have %v, got %v) caused by %v with key %v. Ignoring that CL, please file an issue if you were involved in the CL.", id, cl.OwnerID(), link(cl), k)
			}
			return nil
		})
	})
	if len(ownerIDs) == 0 {
		return nil, errors.New("unable to collect review data, user has never authored a CL, so the reviewer ID cannot be matched")
	}
	return ownerIDs, err
}

// personToId returns the Gerrit ID for a given name of the form "Gerrit User 1234".
func personToId(person *maintner.GitPerson) int {
	if person == nil {
		return -1
	}
	if !strings.HasPrefix(person.Name(), "Gerrit User") {
		return -1
	}
	split := strings.Split(person.Name(), " ")
	if len(split) != 3 {
		return -1
	}
	id, err := strconv.ParseInt(split[2], 10, 64)
	if err != nil {
		return -1
	}
	return int(id)
}

const gerritbotID = 12446

func key(cl *maintner.GerritCL) GerritIdKey {
	return GerritIdKey{
		project: cl.Project.Project(),
		branch:  cl.Branch(),
		status:  cl.Status,
	}
}

func link(cl *maintner.GerritCL) string {
	return fmt.Sprintf("go-review.googlesource.com/c/%s/+/%v", cl.Project.Project(), cl.Number)
}

func inScope(t, start, end time.Time) bool {
	return t.After(start) && t.Before(end)
}

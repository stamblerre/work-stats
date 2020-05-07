// Package golang reports Go contributions and issues.
package golang

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/stamblerre/work-stats/generic"
	"golang.org/x/build/maintner"
)

func Changelists(gerrit *maintner.Gerrit, emails []string, start, end time.Time) (authored, reviewed []*generic.Changelist, err error) {
	emailset := make(map[string]bool)
	for _, e := range emails {
		emailset[e] = true
	}
	authoredMap := make(map[string]*generic.Changelist)
	reviewedMap := make(map[string]*generic.Changelist)
	ownerIDs, err := OwnerIDs(gerrit, emailset)
	if err != nil {
		return nil, nil, err
	}
	if err := gerrit.ForeachProjectUnsorted(func(project *maintner.GerritProject) error {
		// First, collect all CLs authored by the user.
		err := project.ForeachCLUnsorted(func(cl *maintner.GerritCL) error {
			if cl.Owner() == nil || !emailset[cl.Owner().Email()] {
				return nil
			}
			if cl.Status == "abandoned" {
				return nil
			}
			key := key(cl)
			var match bool
			for _, meta := range cl.Metas {
				if !inScope(cl.Commit.CommitTime, start, end) {
					continue
				}
				id := personToID(meta.Commit.Author)
				if ownerID := ownerIDs[key]; ownerID == id {
					match = true
					break
				}
			}
			if !match {
				return nil
			}
			l := link(cl)
			authoredMap[l] = &generic.Changelist{
				Number:      int(cl.Number),
				Link:        l,
				Author:      cl.Owner().Email(),
				Description: cl.Subject(),
				Repo:        project.Project(),
				Category:    extractCategory(cl.Subject()),
				Status:      toStatus(cl.Status),
				MergedAt:    toMergeTime(cl),
			}
			return nil
		})
		return err
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
				if id := personToID(msg.Author); ownerIDs[key] == int(id) {
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
			reviewedMap[l] = &generic.Changelist{
				Number:      int(cl.Number),
				Link:        l,
				Author:      cl.Owner().Email(),
				Description: cl.Subject(),
				Repo:        project.Project(),
				Category:    extractCategory(cl.Subject()),
				Status:      toStatus(cl.Status),
			}
			return nil
		})
	}); err != nil {
		return nil, nil, err
	}
	for _, cl := range authoredMap {
		authored = append(authored, cl)
	}
	for _, cl := range reviewedMap {
		reviewed = append(reviewed, cl)
	}
	sort.Slice(authored, func(i, j int) bool {
		return authored[i].Link < authored[j].Link
	})
	sort.Slice(reviewed, func(i, j int) bool {
		return reviewed[i].Link < reviewed[j].Link
	})
	return authored, reviewed, nil
}

type GerritIDKey struct {
	project, branch, status string
}

func OwnerIDs(gerrit *maintner.Gerrit, emailset map[string]bool) (map[GerritIDKey]int, error) {
	ownerIDs := make(map[GerritIDKey]int)
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

// personToID returns the Gerrit ID for a given name of the form "Gerrit User 1234".
func personToID(person *maintner.GitPerson) int {
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

func key(cl *maintner.GerritCL) GerritIDKey {
	return GerritIDKey{
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

func toStatus(s string) generic.ChangelistStatus {
	switch s {
	case "merged":
		return generic.Merged
	case "abandoned":
		return generic.Abandoned
	case "new":
		return generic.New
	case "draft":
		return generic.Draft
	}
	return generic.Unknown
}

func toMergeTime(cl *maintner.GerritCL) time.Time {
	if cl.Status != "merged" {
		return time.Time{}
	}
	for _, m := range cl.Metas {
		if m.ActionTag() == "autogenerated:gerrit:merged" {
			return m.Commit.CommitTime
		}
	}
	return time.Time{}
}

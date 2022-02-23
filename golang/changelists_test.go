package golang_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stamblerre/work-stats/generic"
	"github.com/stamblerre/work-stats/golang"
	"golang.org/x/build/maintner"
	"golang.org/x/build/maintner/godata"
)

var gerrit *maintner.Gerrit

func TestMain(m *testing.M) {
	corpus, err := godata.Get(context.Background())
	if err != nil {
		os.Exit(1)
	}
	gerrit = corpus.Gerrit()
	os.Exit(m.Run())
}

func TestOwnerIds(t *testing.T) {
	for _, tt := range []struct {
		email string
		ids   map[int]bool
	}{
		{
			email: "rstambler@golang.org",
			ids: map[int]bool{
				16140: true,
			},
		},
	} {
		ids, err := golang.OwnerIDs(gerrit, map[string]bool{
			tt.email: true,
		})
		if err != nil {
			t.Error(err)
		}
		idset := make(map[int]bool)
		for _, id := range ids {
			idset[id] = true
		}
		if len(idset) != len(tt.ids) {
			t.Errorf("got %v, expected %v", idset, tt.ids)
		}
		for id := range idset {
			if !tt.ids[id] {
				t.Errorf("unexpected ID %v", id)
			}
		}
	}
}

func TestGerritToGenericCL(t *testing.T) {
	for _, tt := range []struct {
		want *generic.Changelist
	}{{
		want: &generic.Changelist{
			Number:  352613,
			Link:    "go-review.googlesource.com/c/vscode-go/+/352613",
			Subject: "goSurvey: consolidate gopls survey and go developer survey logic",
			Message: `goSurvey: consolidate gopls survey and go developer survey logic

DO NOT REVIEW

Change-Id: I4409b7d41556a9a22f90c7c865ebdcd6acbc5df0
`,
			Branch:        "master",
			Author:        "rstambler@golang.org",
			Repo:          "vscode-go",
			Status:        generic.Abandoned,
			AffectedFiles: []string{"src/goDeveloperSurvey.ts", "src/goMain.ts", "src/goplsSurvey.ts", "test/gopls/survey.test.ts"},
		},
	}} {
		cl, err := fetchCL(gerrit, tt.want.Repo, int32(tt.want.Number))
		if err != nil {
			t.Fatal(err)
		}
		got := golang.GerritToGenericCL(cl)
		if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreFields(generic.Changelist{}, "Comments")); diff != "" {
			t.Errorf("got unexpected results: %s", diff)
		}
	}
}

func fetchCL(gerrit *maintner.Gerrit, repo string, number int32) (*maintner.GerritCL, error) {
	var result *maintner.GerritCL
	if err := gerrit.ForeachProjectUnsorted(func(project *maintner.GerritProject) error {
		if project.Project() != repo {
			return nil
		}
		return project.ForeachCLUnsorted(func(cl *maintner.GerritCL) error {
			if cl.Number != number {
				return nil
			}
			if result != nil {
				return fmt.Errorf("duplicate CL with the same repo and number: %s/%d", repo, number)
			}
			result = cl
			return nil
		})
	}); err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("unable to find CL for %s/%d", repo, number)
	}
	return result, nil
}

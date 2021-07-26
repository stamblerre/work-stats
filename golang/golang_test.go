package golang_test

import (
	"context"
	"os"
	"testing"

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

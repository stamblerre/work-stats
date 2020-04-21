package generic_test

import (
	"testing"
	"time"

	"github.com/stamblerre/work-stats/generic"
)

func TestInferTimeRange(t *testing.T) {
	for _, tt := range []struct {
		date, weekOf, start, end string
	}{
		{
			// Monday
			date:  "2020-04-13",
			start: "2020-04-06",
			end:   "2020-04-13",
		},
		{
			// Tuesday
			date:  "2020-04-14",
			start: "2020-04-06",
			end:   "2020-04-13",
		},
		{
			// Wednesday
			date:  "2020-04-15",
			start: "2020-04-06",
			end:   "2020-04-13",
		},
		{
			// Thursday
			date:  "2020-04-16",
			start: "2020-04-13",
			end:   "2020-04-20",
		},
		{
			// Friday
			date:  "2020-04-17",
			start: "2020-04-13",
			end:   "2020-04-20",
		},
		{
			// Saturday
			date:  "2020-04-18",
			start: "2020-04-13",
			end:   "2020-04-20",
		},
		{
			// Sunday
			date:  "2020-04-19",
			start: "2020-04-13",
			end:   "2020-04-20",
		},
		{
			// Tuesday, but week of is the current week
			date:   "2020-04-16",
			weekOf: "2020-04-16",
			start:  "2020-04-13",
			end:    "2020-04-20",
		},
	} {
		now, err := time.Parse("2006-01-02", tt.date)
		if err != nil {
			t.Fatal(err)
		}
		wantStart, err := time.Parse("2006-01-02", tt.start)
		if err != nil {
			t.Fatal(err)
		}
		wantEnd, err := time.Parse("2006-01-02", tt.end)
		if err != nil {
			t.Fatal(err)
		}
		start, end, err := generic.InferTimeRange(now, tt.weekOf)
		if err != nil {
			t.Fatal(err)
		}
		if !start.Equal(wantStart) {
			t.Errorf("%s: wanted start %s, got %s", now, wantStart, start)
		}
		if !end.Equal(wantEnd) {
			t.Errorf("%s: wanted end %s, got %s", now, wantEnd, end)
		}
	}
}

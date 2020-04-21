package generic

import (
	"time"
)

// InferTimeRange gets the start and end time for a weekly snippet report.
// If the optional weekOf parameter is provided, the time range is for the
// week in which the date falls, not inferred.
func InferTimeRange(now time.Time, weekOf string) (start, end time.Time, err error) {
	if weekOf != "" {
		t, err := time.Parse("2006-01-02", weekOf)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		now = t
	} else {
		// If this command is running Mon-Wed, assume that it's for the previous
		// week and look for the preceding Monday. If the command is running
		// Thurs-Sunday, assume that it is for the current week.
		switch now.Weekday() {
		case time.Monday:
			now = now.AddDate(0, 0, -1)
		case time.Tuesday:
			now = now.AddDate(0, 0, -2)
		case time.Wednesday:
			now = now.AddDate(0, 0, -3)
		}
	}
	start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	for start.Weekday() != time.Monday {
		start = start.AddDate(0, 0, -1)
	}
	end = start.AddDate(0, 0, 7)
	return start, end, nil
}

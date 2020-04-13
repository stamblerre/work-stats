package generic

import "time"

// SnippetTimeRange gets the start and end time for a weekly snippet report.
func SnippetTimeRange() (time.Time, time.Time) {
	// Assume that users will run the command on Fri, Sat, Sun, or Mon.
	// Look for the previous week.
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// If this command is running on a Monday, assume that it's for the pevious
	// week and look for the preceding Monday.
	if start.Weekday() == time.Monday {
		start = start.AddDate(0, 0, -1)
	}
	end := start
	for start.Weekday() != time.Monday {
		start = start.AddDate(0, 0, -1)
	}
	for end.Weekday() != time.Sunday && end.After(start) {
		end = end.AddDate(0, 0, -1)
	}
	end = end.Add(24 * time.Hour)
	return start, end
}

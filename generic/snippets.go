package generic

import "time"

// SnippetTimeRange gets the start and end time for a weekly snippet report.
func SnippetTimeRange() (time.Time, time.Time) {
	// Assume that users will run the command on Fri, Sat, Sun, or Mon.
	// Look for the previous week.
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// If this command is running Mon-Wed, assume that it's for the previous
	// week and look for the preceding Monday.
	// TODO(rstambler): This approach may need to be rethought.
	switch start.Weekday() {
	case time.Monday:
		start = start.AddDate(0, 0, -1)
	case time.Tuesday:
		start = start.AddDate(0, 0, -2)
	case time.Wednesday:
		start = start.AddDate(0, 0, -3)
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

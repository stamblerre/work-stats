package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/stamblerre/work-stats/generic"
	"github.com/stamblerre/work-stats/golang"
	"github.com/wcharczuk/go-chart"
	"golang.org/x/build/maintner/godata"
)

var (
	since = flag.String("since", "", "date from which to collect data")
)

func main() {
	flag.Parse()

	// Parse out the start date, if provided.
	var (
		start time.Time
		err   error
	)
	if *since != "" {
		start, err = time.Parse("2006-01-02", *since)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		start = time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC)
	}

	ctx := context.Background()

	end := time.Now()

	// Get the corpus data (very slow on first try, uses cache after).
	corpus, err := godata.Get(ctx)
	if err != nil {
		log.Fatal(err)
	}
	vscodeIssues, err := golang.Issues(corpus.GitHub(), "vscode-go", "", start, end)
	if err != nil {
		log.Fatal(err)
	}
	if err := issuesToGraph("vscode-go.png", vscodeIssues, start, end); err != nil {
		log.Fatal(err)
	}
	toolsIssues, err := golang.Issues(corpus.GitHub(), "go", "", start, end)
	if err != nil {
		log.Fatal(err)
	}
	var goplsIssues []*generic.Issue
	for _, issue := range toolsIssues {
		var hasGoplsLabel bool
		for _, label := range issue.Labels {
			if label == "gopls" {
				hasGoplsLabel = true
				break
			}
		}
		if hasGoplsLabel {
			goplsIssues = append(goplsIssues, issue)
		}
	}
	if err := issuesToGraph("gopls.png", goplsIssues, start, end); err != nil {
		log.Fatal(err)
	}
}

func issuesToGraph(filename string, issues []*generic.Issue, start, end time.Time) error {
	var dates []time.Time
	inclusiveEnd := end.Add(time.Hour * 24)
	for t := start; t.Before(inclusiveEnd); {
		dates = append(dates, time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC))
		t = t.Add(time.Hour * 24)
	}
	count := map[time.Time]float64{}
	for _, date := range dates {
		for _, issue := range issues {
			opened := time.Date(issue.DateOpened.Year(), issue.DateOpened.Month(), issue.DateOpened.Day(), 0, 0, 0, 0, time.UTC)
			closed := time.Date(issue.DateClosed.Year(), issue.DateClosed.Month(), issue.DateClosed.Day(), 0, 0, 0, 0, time.UTC)
			if inScope(date, opened, closed) {
				count[date]++
			}
		}
	}
	// for _, date := range dates {
	// 	fmt.Printf("ON %s COUNT IS %v\n", date, count[date])
	// }
	countSlice := make([]float64, len(dates))
	for i, date := range dates {
		countSlice[i] = float64(count[date])
	}
	graph := chart.Chart{
		Title:      filename,
		TitleStyle: chart.StyleShow(),
		YAxis: chart.YAxis{
			Name:  "Issues",
			Style: chart.StyleShow(),
		},
		XAxis: chart.XAxis{
			Name:           "Time",
			ValueFormatter: chart.TimeDateValueFormatter,
			NameStyle:      chart.StyleShow(),
			Style:          chart.StyleShow(),
		},
		Series: []chart.Series{
			chart.TimeSeries{
				XValues: dates,
				YValues: countSlice,
				Style:   chart.StyleShow(),
			},
		},
	}
	f, _ := os.Create(filename)
	defer f.Close()
	return graph.Render(chart.PNG, f)
}

func inScope(t, start, end time.Time) bool {
	afterStart := t.Equal(start) || t.After(start)
	if end.IsZero() {
		return afterStart
	}
	return afterStart && (t.Equal(end) || t.Before(end))
}

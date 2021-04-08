package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	gh "github.com/google/go-github/v28/github"
	"github.com/stamblerre/work-stats/generic"
	"github.com/stamblerre/work-stats/github"
	"github.com/stamblerre/work-stats/golang"
	"github.com/wcharczuk/go-chart/v2"
	"golang.org/x/build/maintner/godata"
	"golang.org/x/oauth2"
)

var (
	since = flag.String("since", "", "date from which to collect data")
	repos = flag.String("repos", "", "repositories to process, comma separated")
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

func issuesToGraph(filename string, incomingIssues []*generic.Issue, start, end time.Time) error {
	var dates []time.Time
	inclusiveEnd := end.Add(time.Hour * 24)
	for t := start; t.Before(inclusiveEnd); {
		dates = append(dates, time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC))
		t = t.Add(time.Hour * 24)
	}
	transfers := map[*generic.Issue]struct{}{}
	var issues []*generic.Issue
	for _, issue := range incomingIssues {
		owner, repo := strings.Split(issue.Repo, "/")[0], strings.Split(issue.Repo, "/")[1]
		split := strings.Split(issue.Link, "/")
		number, err := strconv.Atoi(split[len(split)-1])
		if err != nil {
			return err
		}
		transferred, err := wasTransferred(context.TODO(), owner, repo, int32(number))
		if err != nil {
			return err
		}
		if transferred {
			continue
		}
		issues = append(issues, issue)
	}
	total := 0
	var sum float64
	for _, issue := range issues {
		if !issue.Closed() {
			continue
		}
		sum += end.Sub(start).Hours()
		total++
	}
	timeToClose := sum / float64(total)
	parsed, err := time.ParseDuration(fmt.Sprintf("%vh", timeToClose))
	if err != nil {
		return err
	}
	log.Printf("Average time to close an issue is %s.", parsed)
	count := map[time.Time]float64{}
	fr := map[time.Time]float64{}
	for _, date := range dates {
		for _, issue := range issues {
			if _, ok := transfers[issue]; ok {
				continue
			}
			opened := time.Date(issue.DateOpened.Year(), issue.DateOpened.Month(), issue.DateOpened.Day(), 0, 0, 0, 0, time.UTC)
			closed := time.Date(issue.DateClosed.Year(), issue.DateClosed.Month(), issue.DateClosed.Day(), 0, 0, 0, 0, time.UTC)
			if !inScope(date, opened, closed) {
				continue
			}
			if isFeatureRequest(issue) {
				fr[date]++
			} else {
				count[date]++
			}
		}
	}
	countSlice := make([]float64, len(dates))
	frSlice := make([]float64, len(dates))
	for i, date := range dates {
		countSlice[i] = float64(count[date])
		frSlice[i] = float64(fr[date])
	}
	graph := chart.Chart{
		Title:      filename,
		TitleStyle: chart.Shown(),
		YAxis: chart.YAxis{
			Name:  "Issues",
			Style: chart.Shown(),
		},
		XAxis: chart.XAxis{
			Name:           "Time",
			ValueFormatter: chart.TimeDateValueFormatter,
			NameStyle:      chart.Shown(),
			Style:          chart.Shown(),
		},
		Series: []chart.Series{
			chart.TimeSeries{
				XValues: dates,
				YValues: frSlice,
				Style:   chart.Shown(),
				Name:    "Feature requests",
			},
			chart.TimeSeries{
				XValues: dates,
				YValues: countSlice,
				Style:   chart.Shown(),
				Name:    "Other issues",
			},
		},
	}
	graph.Elements = []chart.Renderable{
		chart.LegendLeft(&graph),
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

func isFeatureRequest(issue *generic.Issue) bool {
	for _, label := range issue.Labels {
		if label == "FeatureRequest" {
			return true
		}
	}
	return false
}

var once sync.Once
var client *gh.Client

func wasTransferred(ctx context.Context, owner, repo string, number int32) (bool, error) {
	once.Do(func() {
		token := os.Getenv("GITHUB_TOKEN")
		if token == "" {
			panic("GITHUB_TOKEN environment variable is not configured")
		}
		ts := oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: token,
		})
		tc := oauth2.NewClient(ctx, ts)
		client = gh.NewClient(tc)
	})
	return github.WasTransferred(ctx, client, owner, repo, number)
}

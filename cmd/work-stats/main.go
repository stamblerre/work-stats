package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/stamblerre/sheets"
	"github.com/stamblerre/work-stats/generic"
	"github.com/stamblerre/work-stats/github"
	"github.com/stamblerre/work-stats/golang"
	"golang.org/x/build/maintner/godata"
	gsheets "google.golang.org/api/sheets/v4"
)

var (
	username = flag.String("username", "", "GitHub username")
	email    = flag.String("email", "", "Gerrit email or emails, comma-separated")
	since    = flag.String("since", "", "date from which to collect data")

	// Optional flags.
	gerritFlag = flag.Bool("gerrit", true, "collect data on Go issues or changelists")
	gitHubFlag = flag.Bool("github", true, "collect data on GitHub issues")

	// Flags relating to Google sheets exporter.
	googleSheetsFlag = flag.String("sheets", "", "write or append output to a Google spreadsheet (either \"\", \"new\", or the URL of an existing sheet)")
	credentialsFile  = flag.String("credentials", "", "path to credentials file for Google Sheets")
	tokenFile        = flag.String("token", "", "path to token file for authentication in Google sheets")
)

func main() {
	flag.Parse()

	// Snippets are a summary of a user's contributions over the past week.
	snippets := flag.Arg(0) == "snippets"

	// Username and email are required flags.
	// If since is omitted, results reflect all history.
	if *username == "" && *gitHubFlag {
		log.Fatal("Please provide a GitHub username.")
	}
	if *email == "" && *gerritFlag {
		log.Fatal("Please provide your Gerrit email.")
	}
	emails := strings.Split(*email, ",")

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
	} else if snippets {
		// If we're generating a snippet report, the start date is a week ago.
		start = time.Now().AddDate(0, 0, -7)
	} else {
		start = time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC)
	}

	// Determine if the user has provided a valid Google Sheets URL.
	var spreadsheetID string
	if *googleSheetsFlag != "new" && *googleSheetsFlag != "" {
		var err error
		spreadsheetID, err = sheets.GetSpreadsheetID(*googleSheetsFlag)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Write output to a temporary directory.
	dir, err := ioutil.TempDir("", "work-stats")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	rowData := make(map[string][]*gsheets.RowData)

	end := time.Now()

	// Write out data on the user's activity on the Go project's GitHub issues
	// and the Go project's Gerrit code reviews.
	if *gerritFlag {
		// Get the corpus data (very slow on first try, uses cache after).
		corpus, err := godata.Get(ctx)
		if err != nil {
			log.Fatal(err)
		}
		issues, err := golang.Issues(corpus.GitHub(), "", *username, start, end)
		if err != nil {
			log.Fatal(err)
		}
		if err := sheets.Write(ctx, dir, map[string][]*sheets.Row{
			"golang-issues": generic.IssuesToCells(*username, issues),
		}, rowData); err != nil {
			log.Fatal(err)
		}
		authored, reviewed, err := golang.Changelists(corpus.Gerrit(), emails, start, end)
		if err != nil {
			log.Fatal(err)
		}
		if err := sheets.Write(ctx, dir, map[string][]*sheets.Row{
			"golang-authored": generic.AuthoredChangelistsToCells(authored),
			"golang-reviewed": generic.ReviewedChangelistsToCells(reviewed),
		}, rowData); err != nil {
			log.Fatal(err)
		}
	}

	// Write out data on the user's activity on GitHub issues outside of the Go project.
	if *gitHubFlag {
		authored, reviewed, issues, err := github.IssuesAndPRs(ctx, *username, start, end)
		if err != nil {
			log.Fatal(err)
		}
		if err := sheets.Write(ctx, dir, map[string][]*sheets.Row{
			"github-issues":       generic.IssuesToCells(*username, issues),
			"github-prs-authored": generic.AuthoredChangelistsToCells(authored),
			"github-prs-reviewed": generic.ReviewedChangelistsToCells(reviewed),
		}, rowData); err != nil {
			log.Fatal(err)
		}
	}

	// Optionally write output to Google Sheets.
	if *googleSheetsFlag == "" {
		return
	}
	if *tokenFile == "" {
		log.Fatal("please provide -token when using -sheets")
	}
	if *credentialsFile == "" {
		log.Fatal("please provide -credentials when using -sheets")
	}
	srv, err := sheets.GoogleSheetsService(ctx, *credentialsFile, *tokenFile)
	if err != nil {
		log.Fatal(err)
	}
	var spreadsheet *gsheets.Spreadsheet
	if *googleSheetsFlag == "new" {
		name := *username
		if name == "" {
			name = strings.Split(*email, "@")[0]
		}
		title := fmt.Sprintf("%s (as of %s)", name, start.Format("01-02-2006"))
		spreadsheet, err = sheets.CreateSheet(ctx, srv, title, rowData)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		spreadsheet, err = sheets.AppendToSheet(ctx, srv, spreadsheetID, rowData)
		if err != nil {
			log.Fatal(err)
		}
	}
	if err := sheets.ResizeColumns(ctx, srv, spreadsheet); err != nil {
		log.Fatal(err)
	}
	log.Printf("Wrote data to Google Sheet: %s\n", spreadsheet.SpreadsheetUrl)
}

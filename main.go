package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/stamblerre/work-stats/github"
	"github.com/stamblerre/work-stats/golang"
	"golang.org/x/build/maintner"
	"golang.org/x/build/maintner/godata"
	"google.golang.org/api/sheets/v4"
)

var (
	username = flag.String("username", "", "GitHub username")
	email    = flag.String("email", "", "Gerrit email or emails, comma-separated")
	since    = flag.String("since", "", "Date from which to collect data")

	// Optional flags.
	gerritFlag = flag.Bool("gerrit", true, "If false, do not collect data on Go issues or changelists")
	gitHubFlag = flag.Bool("github", true, "If false, do not collect data on GitHub issues")

	// Flags relating to Google sheets exporter.
	googleSheetsFlag = flag.Bool("sheets", true, "If false, do not write output to Google sheets")
	credentialsFile  = flag.String("credentials", "credentials.json", "Path to credentials file for Google Sheets")
	tokenFile        = flag.String("token", "token.json", "Path to token file for authentication in Google sheets")

	// Globals.
	corpus      *maintner.Corpus
	spreadsheet *sheets.Spreadsheet
)

func main() {
	flag.Parse()

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
	} else {
		start = time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC)
	}

	// Write output to a temporary directory.
	dir, err := ioutil.TempDir("", "work-stats")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Get the corpus data (very slow on first try, uses cache after).
	var initOnce sync.Once
	initCorpus := func() {
		corpus, err = godata.Get(ctx)
		if err != nil {
			log.Fatal(err)
		}
		spreadsheet = &sheets.Spreadsheet{
			Properties: &sheets.SpreadsheetProperties{
				Title: fmt.Sprintf("Work Stats (as of %s)", start.Format("01-02-2006")),
			},
		}
	}

	// Write out data on the user's activity on the Go project's GitHub issues
	// and the Go project's Gerrit code reviews.
	if *gerritFlag {
		initOnce.Do(initCorpus)
		goIssues, err := golang.Issues(corpus.GitHub(), *username, start)
		if err != nil {
			log.Fatal(err)
		}
		if err := write(ctx, dir, goIssues); err != nil {
			log.Fatal(err)
		}
		goCLs, err := golang.Changelists(corpus.Gerrit(), emails, start)
		if err != nil {
			log.Fatal(err)
		}
		if err := write(ctx, dir, goCLs); err != nil {
			log.Fatal(err)
		}
	}

	// Write out data on the user's activity on GitHub issues outside of the Go project.
	if *gitHubFlag {
		initOnce.Do(initCorpus)
		githubIssues, err := github.IssuesAndPRs(ctx, *username, start)
		if err != nil {
			log.Fatal(err)
		}
		if err := write(ctx, dir, githubIssues); err != nil {
			log.Fatal(err)
		}
	}
	if !*googleSheetsFlag {
		return
	}
	// Create the spreadsheet.
	srv, err := googleSheetsService(ctx)
	if err != nil {
		log.Fatal(err)
	}
	spreadsheet, err := srv.Spreadsheets.Create(spreadsheet).Context(ctx).Do()
	if err != nil {
		log.Fatal(err)
	}
	// Auto-resize the columns of the spreadsheet to fit.
	var requests []*sheets.Request
	for _, sheet := range spreadsheet.Sheets {
		requests = append(requests, &sheets.Request{
			AutoResizeDimensions: &sheets.AutoResizeDimensionsRequest{
				Dimensions: &sheets.DimensionRange{
					Dimension: "COLUMNS",
					SheetId:   sheet.Properties.SheetId,
				},
			},
		})
	}
	if _, err := srv.Spreadsheets.BatchUpdate(spreadsheet.SpreadsheetId, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Context(ctx).Do(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Wrote data to Google Sheet: %s\n", spreadsheet.SpreadsheetUrl)
}

func write(ctx context.Context, outputDir string, data map[string][][]string) error {
	// Write output to disk first.
	var filenames []string
	for filename, cells := range data {
		fullpath := filepath.Join(outputDir, fmt.Sprintf("%s.csv", filename))
		file, err := os.Create(fullpath)
		if err != nil {
			return err
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		for _, row := range cells {
			if err := writer.Write(row); err != nil {
				return err
			}
		}
		filenames = append(filenames, fullpath)
	}
	for _, filename := range filenames {
		fmt.Printf("Wrote output to %s.\n", filename)
	}
	// Add a new sheet and write output to it.
	for filename, cells := range data {
		sheet := &sheets.Sheet{
			Properties: &sheets.SheetProperties{
				Title: filename,
				GridProperties: &sheets.GridProperties{
					FrozenRowCount: 1,
				},
			},
		}
		gd := &sheets.GridData{}
		for i, row := range cells {
			rd := &sheets.RowData{}
			for _, cell := range row {
				var total, subtotal bool
				if len(row) >= 1 {
					total = row[0] == "Total"
					subtotal = row[0] == "Subtotal"
				}
				cd := &sheets.CellData{
					UserEnteredValue: &sheets.ExtendedValue{
						StringValue: cell,
					},
					UserEnteredFormat: &sheets.CellFormat{
						TextFormat: &sheets.TextFormat{
							Bold: i == 0 || total || subtotal,
						},
					},
				}
				if subtotal {
					cd.UserEnteredFormat.BackgroundColor = &sheets.Color{
						Blue:  0.96,
						Green: 0.96,
						Red:   0.96,
					}
				} else if total {
					cd.UserEnteredFormat.BackgroundColor = &sheets.Color{
						Blue:  0.92,
						Green: 0.92,
						Red:   0.92,
					}
				}
				rd.Values = append(rd.Values, cd)
			}
			gd.RowData = append(gd.RowData, rd)
		}
		sheet.Data = append(sheet.Data, gd)
		spreadsheet.Sheets = append(spreadsheet.Sheets, sheet)
	}
	return nil
}

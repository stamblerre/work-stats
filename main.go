package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/stamblerre/work-stats/github"
	"github.com/stamblerre/work-stats/golang"
	"golang.org/x/build/maintner/godata"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
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

	// Determine if the user has provided a valid Google Sheets URL.
	spreadsheetID, err := getSpreadsheetID()
	if err != nil {
		log.Fatal(err)
	}

	// Write output to a temporary directory.
	dir, err := ioutil.TempDir("", "work-stats")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	rowData := make(map[string][]*sheets.RowData)

	// Write out data on the user's activity on the Go project's GitHub issues
	// and the Go project's Gerrit code reviews.
	if *gerritFlag {
		// Get the corpus data (very slow on first try, uses cache after).
		corpus, err := godata.Get(ctx)
		if err != nil {
			log.Fatal(err)
		}
		goIssues, err := golang.Issues(corpus.GitHub(), *username, start)
		if err != nil {
			log.Fatal(err)
		}
		if err := write(ctx, dir, goIssues, rowData); err != nil {
			log.Fatal(err)
		}
		goCLs, err := golang.Changelists(corpus.Gerrit(), emails, start)
		if err != nil {
			log.Fatal(err)
		}
		if err := write(ctx, dir, goCLs, rowData); err != nil {
			log.Fatal(err)
		}
	}

	// Write out data on the user's activity on GitHub issues outside of the Go project.
	if *gitHubFlag {
		githubIssues, err := github.IssuesAndPRs(ctx, *username, start)
		if err != nil {
			log.Fatal(err)
		}
		if err := write(ctx, dir, githubIssues, rowData); err != nil {
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

	srv, err := googleSheetsService(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var spreadsheet *sheets.Spreadsheet
	if *googleSheetsFlag == "new" {
		spreadsheet, err = createSheet(ctx, srv, start, rowData)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		spreadsheet, err = appendToSheet(ctx, srv, spreadsheetID, rowData)
		if err != nil {
			log.Fatal(err)
		}
	}
	// Final sheet updates:
	// - Auto-resize the columns of the spreadsheet to fit.
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
	log.Printf("Wrote data to Google Sheet: %s\n", spreadsheet.SpreadsheetUrl)
}

func write(ctx context.Context, outputDir string, data map[string][][]string, rowData map[string][]*sheets.RowData) error {
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
		log.Printf("Wrote output to %s.\n", filename)
	}
	// Add a new sheet and write output to it.
	for title, cells := range data {
		var rd []*sheets.RowData
		for i, row := range cells {
			var values []*sheets.CellData
			for _, cell := range row {
				var total, subtotal, subsubtotal bool
				if len(row) >= 1 {
					total = row[0] == "Total"
					subtotal = row[0] == "Subtotal"
					subsubtotal = row[0] == ""
				}
				cd := &sheets.CellData{
					UserEnteredValue: &sheets.ExtendedValue{
						StringValue: cell,
					},
					UserEnteredFormat: &sheets.CellFormat{
						TextFormat: &sheets.TextFormat{
							Bold: i == 0 || total || subtotal || subsubtotal,
						},
					},
				}
				if subsubtotal {
					cd.UserEnteredFormat.BackgroundColor = &sheets.Color{
						Blue:  0.97,
						Green: 0.97,
						Red:   0.97,
					}
				} else if subtotal {
					cd.UserEnteredFormat.BackgroundColor = &sheets.Color{
						Blue:  0.94,
						Green: 0.94,
						Red:   0.94,
					}
				} else if total {
					cd.UserEnteredFormat.BackgroundColor = &sheets.Color{
						Blue:  0.91,
						Green: 0.91,
						Red:   0.91,
					}
				}
				values = append(values, cd)
			}
			rd = append(rd, &sheets.RowData{
				Values: values,
			})
		}
		rowData[title] = rd
	}
	return nil
}

func googleSheetsService(ctx context.Context) (*sheets.Service, error) {
	// Read the user's credentials file.
	b, err := ioutil.ReadFile(*credentialsFile)
	if err != nil {
		return nil, err
	}
	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		return nil, err
	}
	tok, err := getOauthToken(ctx, config)
	if err != nil {
		return nil, err
	}
	return sheets.New(config.Client(ctx, tok))
}

func getOauthToken(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	// token.json stores the user's access and refresh tokens, and is created
	// automatically when the authorization flow completes for the first time.
	f, err := os.Open(*tokenFile)
	if err == nil {
		defer f.Close()
		tok := &oauth2.Token{}
		if err := json.NewDecoder(f).Decode(tok); err != nil {
			return nil, err
		}
		return tok, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	// If the token file isn't available, create one.
	// Request a token from the web, then returns the retrieved token.
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	log.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, err
	}
	tok, err := config.Exchange(ctx, authCode)
	if err != nil {
		return nil, err
	}
	// Save the token for future use.
	log.Printf("Saving credential file to: %s\n", *tokenFile)
	f, err = os.OpenFile(*tokenFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(tok); err != nil {
		return nil, err
	}
	return tok, nil
}

func createSheet(ctx context.Context, srv *sheets.Service, start time.Time, rowData map[string][]*sheets.RowData) (*sheets.Spreadsheet, error) {
	var newSheets []*sheets.Sheet
	for title, data := range rowData {
		newSheets = append(newSheets, &sheets.Sheet{
			Properties: &sheets.SheetProperties{
				Title: title,
				GridProperties: &sheets.GridProperties{
					FrozenRowCount: 1,
				},
			},
			Data: []*sheets.GridData{{RowData: data}},
		})
	}
	spreadsheet := &sheets.Spreadsheet{
		Properties: &sheets.SpreadsheetProperties{
			Title: fmt.Sprintf("Work Stats (as of %s)", start.Format("01-02-2006")),
		},
		Sheets: newSheets,
	}
	return srv.Spreadsheets.Create(spreadsheet).Context(ctx).Do()
}

func appendToSheet(ctx context.Context, srv *sheets.Service, spreadsheetID string, rowData map[string][]*sheets.RowData) (*sheets.Spreadsheet, error) {
	// First, create the new sheets in spreadsheet.
	var createRequests []*sheets.Request
	for title := range rowData {
		createRequests = append(createRequests, &sheets.Request{
			AddSheet: &sheets.AddSheetRequest{
				Properties: &sheets.SheetProperties{
					Title: title,
					GridProperties: &sheets.GridProperties{
						FrozenRowCount: 1,
					},
				},
			},
		})
	}
	response, err := srv.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		IncludeSpreadsheetInResponse: true,
		Requests:                     createRequests,
	}).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	// Now, add the data to the spreadsheets.
	var dataRequests []*sheets.Request
	for _, sheet := range response.UpdatedSpreadsheet.Sheets {
		dataRequests = append(dataRequests, &sheets.Request{
			AppendCells: &sheets.AppendCellsRequest{
				SheetId: sheet.Properties.SheetId,
				Rows:    rowData[sheet.Properties.Title],
				Fields:  "*",
			},
		})
	}
	response, err = srv.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		IncludeSpreadsheetInResponse: true,
		Requests:                     dataRequests,
	}).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	return response.UpdatedSpreadsheet, nil
}

// Return the Google Sheets spreadsheet ID. If the googleSheetsFlag is an
// invalid format, an error will be returned. If the googleSheetsFlag is empty
// or "new", an empty ID will be returned.
func getSpreadsheetID() (string, error) {
	if *googleSheetsFlag == "new" || *googleSheetsFlag == "" {
		return "", nil
	}

	var spreadsheetID string
	// Trim the extra pieces that the URL may contain.
	trimmed := strings.TrimPrefix(*googleSheetsFlag, "https://docs.google.com")
	trimmed = strings.TrimSuffix(trimmed, "edit#gid=0")

	// Source: https://developers.google.com/sheets/api/guides/concepts.
	re, err := regexp.Compile("/spreadsheets/d/(?P<ID>([a-zA-Z0-9-_]+))")
	if err != nil {
		return "", err
	}
	match := re.FindStringSubmatch(trimmed)
	for i, name := range re.SubexpNames() {
		if name == "ID" {
			spreadsheetID = match[i]
		}
	}

	return spreadsheetID, nil
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
)

func addSheet(ctx context.Context, title string) error {
	req := &sheets.BatchUpdateSpreadsheetRequest{
		IncludeSpreadsheetInResponse: true,
		Requests: []*sheets.Request{
			{
				AddSheet: &sheets.AddSheetRequest{
					Properties: &sheets.SheetProperties{
						Title: title,
					},
				},
			},
		},
	}
	resp, err := srv.Spreadsheets.BatchUpdate(spreadsheet.SpreadsheetId, req).Context(ctx).Do()
	if err != nil {
		return err
	}
	spreadsheet = resp.UpdatedSpreadsheet
	return nil
}

func deleteSheet(ctx context.Context, id int64) error {
	req := &sheets.BatchUpdateSpreadsheetRequest{
		IncludeSpreadsheetInResponse: true,
		Requests: []*sheets.Request{
			{
				DeleteSheet: &sheets.DeleteSheetRequest{
					SheetId: id,
				},
			},
		},
	}
	resp, err := srv.Spreadsheets.BatchUpdate(spreadsheet.SpreadsheetId, req).Context(ctx).Do()
	if err != nil {
		return err
	}
	spreadsheet = resp.UpdatedSpreadsheet
	return nil
}

func writeToSheet(ctx context.Context, id, title string, values [][]interface{}) error {
	_, err := srv.Spreadsheets.Values.Update(id, title, &sheets.ValueRange{
		Values: values,
	}).Context(ctx).ValueInputOption("RAW").Do()
	return err
}

func googleSheetsService(ctx context.Context) (*sheets.Service, error) {
	// Return early if we aren't also writing to a Google Sheet.
	if !*googleSheetsFlag {
		return nil, nil
	}
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
	tok, err := getTokenFromWeb(ctx, config)
	if err != nil {
		return nil, err
	}
	// Save the token for future use.
	fmt.Printf("Saving credential file to: %s\n", *tokenFile)
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

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, err
	}
	return config.Exchange(ctx, authCode)
}

# work-stats

A command-line tool for getting data on a user's open-source work, specific to Go contributors. It uses [maintner](https://pkg.go.dev/golang.org/x/build/maintner?tab=doc) to get more detailed statistics about Go contributions, and it uses the GitHub API to get information on other GitHub contributions.

It exports CSV files for the following:

* Go issues opened, closed, and commented on
* CLs sent to any of the Go repos
* CLs reviewed in anym of the Go repos
* GitHub issues opened, closed, and commented on
* GitHub PRs opened
* GitHub PRs reviewed

## Installation

`go get github.com/stamblerre/work-stats`

## Run

Grab a token from https://github.com/settings/tokens.

It will need: `read:discussion, read:enterprise, read:gpg_key, read:org, read:packages, read:public_key, read:repo_hook, repo, user`.

```shell
$ export GITHUB_TOKEN=<your token>
$ work-stats --username=bob --email=bob@gmail.com,bob@golang.org --since=2019-01-01
```

## In progress

This tool can also generate a Google Sheet containing the CSV files as different sheets. To do this, the user will need to create a Google Cloud project with the Google Sheets API enabled. This is an obstacle to usage of this tool, so work still needs to be done to simplify this process. As of 02/02/2020, the [Go Google Sheets API Tutorial](https://developers.google.com/sheets/api/quickstart/go) details all of the necessary steps. In particular, the user must enable the Google Sheets API and download the `credentials.json` file. When the application first runs, it will request the user's authorization of the app and generate a `token.json`.
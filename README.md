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

### Go-specific contribution data

```shell
$ work-stats --email=bob@gmail.com,bob@golang.org --since=2019-01-01
```

### Other GitHub contributions
Grab a token from https://github.com/settings/tokens. It will need: `read:discussion, read:enterprise, read:gpg_key, read:org, read:packages, read:public_key, read:repo_hook, repo, user`. You will also need to pass in your GitHub username.

```shell
$ export GITHUB_TOKEN=<your token>
$ work-stats --username=bob --email=bob@gmail.com,bob@golang.org --since=2019-01-01
```

### Export data to Google Sheets

This is a bit more involved, but the output will be a formatted Google sheet with different tabs for each category. To use the Google Sheets API from a command-line tool, you will need to create a Google Cloud project with the Google Sheets API enabled. The easiest way to do this is by following the link on the [Google Sheets API tutorial](https://developers.google.com/sheets/api/quickstart/go). It will create a project with the name "Quickstart". It will prompt you to download a `credentials.json` file. The path to that file will need to be passed in through the `-credentials` flag. These credentials will be used to generate a token the first time you run the program. Pass the file path at which you would like the token to be created through the  `-token` flag.

Alternatively, if you do not follow the Quickstart link, you can go to https://pantheon.corp.google.com/apis and create a new project with any name. Click on "Enable APIs", select the Google Sheets API, and clicking "Enable". Then click "APIs & Services" -> "Credentials" -> "Create Credentials" -> "Oauth client ID". Once the credentials are created, click the download button on the right. The path to the credentials will be passed through the `-credentials` flag. These credentials will be used to generate a token the first time you run the program. Pass the file path at which you would like the token to be created through the  `-token` flag.

The command will then be:

```shell
$ work-stats --username=bob --email=bob@gmail.com,bob@golang.org --since=2019-01-01 --sheets=true --credentials=/path/to/credentials.json --token=/path/to/token.json
```

Make sure to add `-sheets` to turn on the Google Sheets feature. When you first create the token, you will be prompted to authorize your Google Clould project to access your Google account by following a link. The link to your Google sheet will be printed when the program exits.

# snippets

A command-line tool for getting weekly summary of a user's open-source work, specific to Go contributors.
It uses [maintner](https://pkg.go.dev/golang.org/x/build/maintner?tab=doc) to get more detailed statistics about Go
contributions, and it uses the GitHub API to get information on other GitHub contributions.

The tool infers the week for which to produce snippets based on the time at which the user runs the command.
Alternatively, a user can specify the week via a flag.

It produces data on the following:

* CLs merged in any of the Go repos
* CLs in-progress in any of the Go repos
* CLs reviewed in any of the Go repos
* Number of Go issues opened, closed, or commented on
* GitHub PRs merged
* GitHub PRs in-progress
* GitHub PRs reviewed
* Number of GitHub issues opened, closed, and commented on

## Installation

`go get github.com/stamblerre/work-stats/snippets`

## Usage

You will need to set the `GITHUB_TOKEN` environment variable to get GitHub contributions.
See [GitHub Token](#GitHub-Token) below on how to do this. The `-email` flag is a comma-separated list
that specifies a user's Gerrit email. The `-username` flag is a user's GitHub username.
Both of these are optional and can be omitted if the user only wants data on Gerrit contributions or GitHub contributions.

An optional `-week` flag can be optionally provided to specify the week for which to collect snippets.
The date provided to this flag can be any date in the intended week, in the format `2006-01-02`.
Without this flag, the tool will infer the week for which to generate snippets based on the date on which the command is
being executed.

```shell
$ snippets -email=bob@gmail.com -username=bob
```

### GitHub Token

Grab a token from https://github.com/settings/tokens. It will need: 

```
read:discussion
read:enterprise
read:gpg_key
read:org
read:packages
read:public_key
read:repo_hook
repo
user
```

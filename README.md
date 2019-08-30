# work-stats

Install:

`go get github.com/stamblerre/work-stats`

Run:

Grab a token from https://github.com/settings/tokens. It will need: `read:discussion, read:enterprise, read:gpg_key, read:org, read:packages, read:public_key, read:repo_hook, repo, user`.

```
export GITHUB_TOKEN=<your token>
work-stats --username=bob --email=bob@gmail.com,bob@golang.org --since=2019-01-01
```

### TODO
* Automatically generate Google Sheets

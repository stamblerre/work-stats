// Package github reports data on GitHub PRs and issues.
package github

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v28/github"
	"github.com/stamblerre/work-stats/generic"
	"golang.org/x/oauth2"
)

func IssuesAndPRs(ctx context.Context, username string, start, end time.Time) (authored, reviewed []*generic.Changelist, issues []*generic.Issue, err error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, nil, nil, fmt.Errorf("GITHUB_TOKEN environment variable is not configured")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token,
	})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	issuesMap := make(map[string]*generic.Issue)
	authoredMap := make(map[string]*generic.Changelist)
	reviewedMap := make(map[string]*generic.Changelist)
	seen := make(map[string]struct{})

	var mostRecentIssue time.Time
	last := start
outer:
	for {
		var current int
		for i := 1; i < 11; i++ {
			result, _, err := client.Search.Issues(ctx, fmt.Sprintf("is:issue involves:%v updated:%s..%s", username, last.Format(time.RFC3339), end.Format(time.RFC3339)), &github.SearchOptions{
				ListOptions: github.ListOptions{
					Page:    i,
					PerPage: 100,
				},
				Sort:  "updated",
				Order: "asc",
			})
			if err != nil {
				return nil, nil, nil, err
			}
			for _, issue := range result.Issues {
				if _, ok := seen[issue.GetHTMLURL()]; ok {
					continue
				}
				seen[issue.GetHTMLURL()] = struct{}{}
				mostRecentIssue = *issue.UpdatedAt
				trimmed := strings.TrimPrefix(issue.GetRepositoryURL(), "https://api.github.com/repos/")
				split := strings.SplitN(trimmed, "/", 2)
				org, repo := split[0], split[1]
				// golang issues are tracker via the golang package.
				if org == "golang" {
					continue
				}
				// Only mark issues as opened if the user opened them since the specified date.
				openedBy := issue.GetUser().GetLogin()
				closed := issue.GetClosedBy() != nil || !issue.GetClosedAt().Equal(time.Time{})
				if issue.IsPullRequest() {
					status := generic.Unknown
					if closed {
						// Check if the PR has been merged. (It may have been
						// closed without being merged.)
						merged, _, err := client.PullRequests.IsMerged(ctx, org, repo, issue.GetNumber())
						if err != nil {
							return nil, nil, nil, err
						}
						// Ignore issues that have been closed without being
						// merged. This will ignore merged PRs that are
						// mirrored from Gerrit because those are closed, even
						// though the CL has been merged.
						if !merged {
							continue
						}
						status = generic.Merged
					}
					gc := GitHubToGenericChangelist(issue, org, repo, status)
					if openedBy == username {
						authoredMap[issue.GetHTMLURL()] = gc
					} else {
						reviewedMap[issue.GetHTMLURL()] = gc
					}
					continue
				}
				comments, _, err := client.Issues.ListComments(ctx, org, repo, issue.GetNumber(), nil)
				if err != nil {
					return nil, nil, nil, err
				}
				var numComments int
				for _, c := range comments {
					if c.GetUser().GetLogin() != username {
						continue
					}
					if !inScope(c.GetCreatedAt(), start, end) {
						continue
					}
					numComments++
				}
				issuesMap[issue.GetHTMLURL()] = GitHubToGenericIssue(issue, org, repo, numComments)
			}
			current += len(result.Issues)
			if current >= result.GetTotal() {
				break outer
			}
		}
		last = mostRecentIssue
	}
	for _, i := range issuesMap {
		issues = append(issues, i)
	}
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Link < issues[j].Link
	})
	for _, pr := range authoredMap {
		authored = append(authored, pr)
	}
	for _, pr := range reviewedMap {
		reviewed = append(reviewed, pr)
	}
	sort.Slice(authored, func(i, j int) bool {
		return authored[i].Link < authored[j].Link
	})
	sort.Slice(reviewed, func(i, j int) bool {
		return reviewed[i].Link < reviewed[j].Link
	})
	return authored, reviewed, issues, nil
}

func inScope(t, start, end time.Time) bool {
	return t.After(start) && t.Before(end)
}

func WasTransferred(ctx context.Context, client *github.Client, owner, repo string, number int32) (bool, error) {
	issue, _, err := client.Issues.Get(ctx, owner, repo, int(number))
	if err != nil {
		return false, err
	}
	split := strings.Split(issue.GetRepositoryURL(), "/")
	actualRepo := split[len(split)-1]
	return actualRepo != repo, nil
}

func GitHubToGenericIssue(issue github.Issue, org, repo string, numComments int) *generic.Issue {
	var milestone string
	if issue.GetMilestone() != nil {
		milestone = *issue.GetMilestone().Title
	}
	return &generic.Issue{
		Number:     issue.GetNumber(),
		Repo:       fmt.Sprintf("%s/%s", org, repo),
		Title:      issue.GetTitle(),
		Link:       issue.GetHTMLURL(),
		OpenedBy:   issue.GetUser().GetLogin(),
		ClosedBy:   issue.GetClosedBy().GetLogin(),
		DateOpened: issue.GetCreatedAt(),
		DateClosed: issue.GetClosedAt(),
		Comments:   numComments,
		Milestone:  milestone,
	}
}

func GitHubToGenericChangelist(pr github.Issue, org, repo string, status generic.ChangelistStatus) *generic.Changelist {
	return &generic.Changelist{
		Repo:     fmt.Sprintf("%s/%s", org, repo),
		Subject:  pr.GetTitle(),
		Message:  pr.GetBody(),
		Link:     pr.GetHTMLURL(),
		Author:   pr.GetUser().GetLogin(),
		Number:   pr.GetNumber(),
		Status:   status,
		MergedAt: pr.GetClosedAt(),
	}
}

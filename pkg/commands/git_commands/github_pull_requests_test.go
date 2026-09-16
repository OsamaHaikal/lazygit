package git_commands

import (
	"testing"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/hosting_service"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stretchr/testify/assert"
)

var testGithubRepo = hosting_service.ServiceInfo{
	Provider:   "github",
	WebDomain:  "github.com",
	Owner:      "jesseduffield",
	Repository: "lazygit",
	RepoName:   "jesseduffield/lazygit",
}

func TestPullRequestSearchQuery(t *testing.T) {
	cases := []struct {
		filter   PullRequestFilter
		expected string
	}{
		{PullRequestFilterOpen, "repo:jesseduffield/lazygit is:pr is:open sort:updated-desc"},
		{PullRequestFilterMine, "repo:jesseduffield/lazygit is:pr is:open author:@me sort:updated-desc"},
		{PullRequestFilterReviewRequested, "repo:jesseduffield/lazygit is:pr is:open review-requested:@me sort:updated-desc"},
		{PullRequestFilterMerged, "repo:jesseduffield/lazygit is:pr is:merged sort:updated-desc"},
		{PullRequestFilterClosed, "repo:jesseduffield/lazygit is:pr is:closed is:unmerged sort:updated-desc"},
		{PullRequestFilterAll, "repo:jesseduffield/lazygit is:pr sort:updated-desc"},
	}

	for _, c := range cases {
		t.Run(c.expected, func(t *testing.T) {
			assert.Equal(t, c.expected, pullRequestSearchQuery(testGithubRepo, c.filter))
		})
	}
}

func TestListPullRequests(t *testing.T) {
	t.Setenv("GH_PATH", "gh")

	response := `{"data":{"search":{"nodes":[
		{
			"number": 12,
			"title": "Add a feature",
			"url": "https://github.com/jesseduffield/lazygit/pull/12",
			"state": "OPEN",
			"isDraft": false,
			"headRefName": "feature",
			"headRefOid": "abc123",
			"baseRefName": "master",
			"baseRefOid": "def456",
			"reviewDecision": "APPROVED",
			"additions": 10,
			"deletions": 2,
			"changedFiles": 3,
			"updatedAt": "2026-09-16T14:00:27Z",
			"author": {"login": "alice"},
			"headRepositoryOwner": {"login": "alice"},
			"comments": {"totalCount": 4},
			"commits": {"nodes": [{"commit": {"statusCheckRollup": {"state": "SUCCESS"}}}]}
		},
		{
			"number": 11,
			"title": "Work in progress",
			"url": "https://github.com/jesseduffield/lazygit/pull/11",
			"state": "OPEN",
			"isDraft": true,
			"reviewDecision": null,
			"author": {"login": "bob"},
			"headRepositoryOwner": {"login": "bob"},
			"comments": {"totalCount": 0},
			"commits": {"nodes": [{"commit": {"statusCheckRollup": null}}]}
		},
		{}
	]}}}`

	runner := oscommands.NewFakeRunner(t).ExpectArgs([]string{
		"gh", "api", "graphql",
		"--hostname", "github.com",
		"-f", "query=" + listPullRequestsQuery,
		"-f", "q=repo:jesseduffield/lazygit is:pr is:open author:@me sort:updated-desc",
	}, response, nil)
	instance := NewGitHubCommands(buildGitCommon(commonDeps{runner: runner}))

	prs, err := instance.ListPullRequests(testGithubRepo, PullRequestFilterMine)
	assert.NoError(t, err)
	runner.CheckForMissingCalls()

	assert.Equal(t, []*models.GithubPullRequest{
		{
			Number:              12,
			Title:               "Add a feature",
			Url:                 "https://github.com/jesseduffield/lazygit/pull/12",
			State:               "OPEN",
			ChecksState:         "SUCCESS",
			HeadRefName:         "feature",
			HeadRepositoryOwner: models.GithubRepositoryOwner{Login: "alice"},
			Author:              "alice",
			BaseRefName:         "master",
			HeadRefOid:          "abc123",
			BaseRefOid:          "def456",
			ReviewDecision:      "APPROVED",
			Additions:           10,
			Deletions:           2,
			ChangedFiles:        3,
			CommentCount:        4,
			UpdatedAt:           time.Date(2026, 9, 16, 14, 0, 27, 0, time.UTC),
		},
		{
			Number:              11,
			Title:               "Work in progress",
			Url:                 "https://github.com/jesseduffield/lazygit/pull/11",
			State:               "DRAFT",
			HeadRepositoryOwner: models.GithubRepositoryOwner{Login: "bob"},
			Author:              "bob",
		},
	}, prs)
}

func TestListPullRequestsWithoutGh(t *testing.T) {
	t.Setenv("GH_PATH", "")
	t.Setenv("PATH", "")

	instance := NewGitHubCommands(buildGitCommon(commonDeps{}))

	_, err := instance.ListPullRequests(testGithubRepo, PullRequestFilterOpen)
	assert.ErrorIs(t, err, ErrGhNotInstalled)
}

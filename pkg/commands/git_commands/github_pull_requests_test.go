package git_commands

import (
	"testing"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/hosting_service"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/commands/patch"
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
		{PullRequestFilterAssigned, "repo:jesseduffield/lazygit is:pr is:open assignee:@me sort:updated-desc"},
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
			"commits": {"nodes": [{"commit": {"statusCheckRollup": {"state": "SUCCESS"}}}]},
			"assignees": {"nodes": [{"login": "alice"}]},
			"reviewRequests": {"nodes": [{"requestedReviewer": {"login": "carol"}}, {"requestedReviewer": {"combinedSlug": "jesseduffield/maintainers"}}]},
			"labels": {"nodes": [{"name": "enhancement"}]}
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
			Assignees:           []string{"alice"},
			ReviewRequests:      []string{"carol", "jesseduffield/maintainers"},
			Labels:              []string{"enhancement"},
		},
		{
			Number:              11,
			Title:               "Work in progress",
			Url:                 "https://github.com/jesseduffield/lazygit/pull/11",
			State:               "DRAFT",
			HeadRepositoryOwner: models.GithubRepositoryOwner{Login: "bob"},
			Author:              "bob",
			Assignees:           []string{},
			ReviewRequests:      []string{},
			Labels:              []string{},
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

func TestPullRequestCommands(t *testing.T) {
	t.Setenv("GH_PATH", "gh")

	cases := []struct {
		name         string
		run          func(*GitHubCommands) error
		expectedArgs []string
	}{
		{
			name:         "checkout",
			run:          func(c *GitHubCommands) error { return c.CheckoutPullRequest(testGithubRepo, 12) },
			expectedArgs: []string{"gh", "pr", "checkout", "12", "--repo", "github.com/jesseduffield/lazygit"},
		},
		{
			name: "squash merge",
			run: func(c *GitHubCommands) error {
				return c.MergePullRequest(testGithubRepo, 12, PullRequestMergeMethodSquash, false)
			},
			expectedArgs: []string{"gh", "pr", "merge", "12", "--repo", "github.com/jesseduffield/lazygit", "--squash"},
		},
		{
			name: "auto-merge",
			run: func(c *GitHubCommands) error {
				return c.MergePullRequest(testGithubRepo, 12, PullRequestMergeMethodRebase, true)
			},
			expectedArgs: []string{"gh", "pr", "merge", "12", "--repo", "github.com/jesseduffield/lazygit", "--rebase", "--auto"},
		},
		{
			name:         "convert to draft",
			run:          func(c *GitHubCommands) error { return c.ConvertPullRequestToDraft(testGithubRepo, 12) },
			expectedArgs: []string{"gh", "pr", "ready", "12", "--repo", "github.com/jesseduffield/lazygit", "--undo"},
		},
		{
			name:         "comment",
			run:          func(c *GitHubCommands) error { return c.CommentOnPullRequest(testGithubRepo, 12, "Looks good") },
			expectedArgs: []string{"gh", "pr", "comment", "12", "--repo", "github.com/jesseduffield/lazygit", "--body", "Looks good"},
		},
		{
			name: "approve without a body",
			run: func(c *GitHubCommands) error {
				return c.ReviewPullRequest(testGithubRepo, 12, PullRequestReviewApprove, "")
			},
			expectedArgs: []string{"gh", "pr", "review", "12", "--repo", "github.com/jesseduffield/lazygit", "--approve"},
		},
		{
			name: "request changes",
			run: func(c *GitHubCommands) error {
				return c.ReviewPullRequest(testGithubRepo, 12, PullRequestReviewRequestChanges, "Please add tests")
			},
			expectedArgs: []string{"gh", "pr", "review", "12", "--repo", "github.com/jesseduffield/lazygit", "--request-changes", "--body", "Please add tests"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			runner := oscommands.NewFakeRunner(t).ExpectArgs(c.expectedArgs, "", nil)
			instance := NewGitHubCommands(buildGitCommon(commonDeps{runner: runner}))

			assert.NoError(t, c.run(instance))
			runner.CheckForMissingCalls()
		})
	}
}

func TestFetchPullRequest(t *testing.T) {
	runner := oscommands.NewFakeRunner(t).ExpectGitArgs(
		[]string{"fetch", "--no-tags", "--no-write-fetch-head", "upstream", "refs/pull/12/head", "refs/heads/main"}, "", nil)
	instance := NewGitHubCommands(buildGitCommon(commonDeps{runner: runner}))

	assert.NoError(t, instance.FetchPullRequest(nil, "upstream", 12, "main"))
	runner.CheckForMissingCalls()
}

func TestHasCommits(t *testing.T) {
	cases := []struct {
		name     string
		output   string
		expected bool
	}{
		{
			name:     "all present",
			output:   "abc commit 200\ndef commit 210\n",
			expected: true,
		},
		{
			name:     "one missing",
			output:   "abc commit 200\ndef^{commit} missing\n",
			expected: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			runner := oscommands.NewFakeRunner(t).ExpectGitArgs([]string{"cat-file", "--batch-check"}, c.output, nil)
			instance := NewGitHubCommands(buildGitCommon(commonDeps{runner: runner}))

			assert.Equal(t, c.expected, instance.HasCommits("abc", "def"))
			runner.CheckForMissingCalls()
		})
	}
}

func TestCommentOnPullRequestLines(t *testing.T) {
	t.Setenv("GH_PATH", "gh")

	cases := []struct {
		name         string
		opts         PullRequestLineCommentOpts
		expectedArgs []string
	}{
		{
			name: "single added line",
			opts: PullRequestLineCommentOpts{
				Number: 12, CommitOid: "abc123", Path: "dir/file.go", Body: "Why?",
				StartLine: patch.FileLine{Number: 7},
				EndLine:   patch.FileLine{Number: 7},
			},
			expectedArgs: []string{
				"gh", "api", "--hostname", "github.com", "--method", "POST",
				"repos/jesseduffield/lazygit/pulls/12/comments",
				"-f", "body=Why?", "-f", "commit_id=abc123", "-f", "path=dir/file.go",
				"-F", "line=7", "-f", "side=RIGHT",
			},
		},
		{
			name: "range starting at a deleted line",
			opts: PullRequestLineCommentOpts{
				Number: 12, CommitOid: "abc123", Path: "file.go", Body: "Nice",
				StartLine: patch.FileLine{Number: 3, IsOld: true},
				EndLine:   patch.FileLine{Number: 5},
			},
			expectedArgs: []string{
				"gh", "api", "--hostname", "github.com", "--method", "POST",
				"repos/jesseduffield/lazygit/pulls/12/comments",
				"-f", "body=Nice", "-f", "commit_id=abc123", "-f", "path=file.go",
				"-F", "line=5", "-f", "side=RIGHT",
				"-F", "start_line=3", "-f", "start_side=LEFT",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			runner := oscommands.NewFakeRunner(t).ExpectArgs(c.expectedArgs, "", nil)
			instance := NewGitHubCommands(buildGitCommon(commonDeps{runner: runner}))

			assert.NoError(t, instance.CommentOnPullRequestLines(testGithubRepo, c.opts))
			runner.CheckForMissingCalls()
		})
	}
}

func TestEditPullRequest(t *testing.T) {
	t.Setenv("GH_PATH", "gh")

	runner := oscommands.NewFakeRunner(t).
		ExpectArgs([]string{"gh", "pr", "edit", "12", "--repo", "github.com/jesseduffield/lazygit", "--add-reviewer", "jesseduffield/maintainers"}, "", nil)
	instance := NewGitHubCommands(buildGitCommon(commonDeps{runner: runner}))

	assert.NoError(t, instance.EditPullRequest(testGithubRepo, 12, PullRequestEditAddReviewer, "jesseduffield/maintainers"))
	runner.CheckForMissingCalls()
}

func TestListAssignableUsersAndLabels(t *testing.T) {
	t.Setenv("GH_PATH", "gh")

	runner := oscommands.NewFakeRunner(t).
		ExpectArgs([]string{"gh", "api", "--hostname", "github.com", "--paginate", "repos/jesseduffield/lazygit/assignees", "--jq", ".[].login"}, "alice\nbob\n", nil).
		ExpectArgs([]string{"gh", "label", "list", "--repo", "github.com/jesseduffield/lazygit", "--limit", "1000", "--json", "name", "--jq", ".[].name"}, "", nil)
	instance := NewGitHubCommands(buildGitCommon(commonDeps{runner: runner}))

	users, err := instance.ListAssignableUsers(testGithubRepo)
	assert.NoError(t, err)
	assert.Equal(t, []string{"alice", "bob"}, users)

	labels, err := instance.ListLabels(testGithubRepo)
	assert.NoError(t, err)
	assert.Equal(t, []string{}, labels)
	runner.CheckForMissingCalls()
}

package git_commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/hosting_service"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/samber/lo"
)

var ErrGhNotInstalled = errors.New("gh is not installed")

type PullRequestFilter int

const (
	PullRequestFilterOpen PullRequestFilter = iota
	PullRequestFilterMine
	PullRequestFilterReviewRequested
	PullRequestFilterMerged
	PullRequestFilterClosed
	PullRequestFilterAll
)

// The number of pull requests listed in the pull requests panel. GitHub's
// search gets noticeably slower the more we ask for, and the list is sorted by
// most recently updated, so the ones further down are rarely the ones you're
// after; the filters are the way to get at those.
const pullRequestListLimit = 50

var listPullRequestsQuery = fmt.Sprintf(`query($q: String!) {
  search(query: $q, type: ISSUE, first: %d) {
    nodes {
      ... on PullRequest {
        number
        title
        url
        state
        isDraft
        headRefName
        headRefOid
        baseRefName
        baseRefOid
        reviewDecision
        additions
        deletions
        changedFiles
        updatedAt
        author { login }
        headRepositoryOwner { login }
        comments { totalCount }
        commits(last: 1) { nodes { commit { statusCheckRollup { state } } } }
      }
    }
  }
}`, pullRequestListLimit)

func pullRequestSearchQuery(repo hosting_service.ServiceInfo, filter PullRequestFilter) string {
	var qualifiers string
	switch filter {
	case PullRequestFilterOpen:
		qualifiers = "is:open"
	case PullRequestFilterMine:
		qualifiers = "is:open author:@me"
	case PullRequestFilterReviewRequested:
		qualifiers = "is:open review-requested:@me"
	case PullRequestFilterMerged:
		qualifiers = "is:merged"
	case PullRequestFilterClosed:
		qualifiers = "is:closed is:unmerged"
	case PullRequestFilterAll:
		qualifiers = ""
	}

	return strings.Join(strings.Fields(fmt.Sprintf("repo:%s is:pr %s sort:updated-desc", repo.RepoName, qualifiers)), " ")
}

// ListPullRequests returns the pull requests of the given GitHub repo that
// match the filter, most recently updated first.
func (self *GitHubCommands) ListPullRequests(repo hosting_service.ServiceInfo, filter PullRequestFilter) ([]*models.GithubPullRequest, error) {
	cmdObj, err := self.ghCmdObj(
		"api", "graphql",
		"--hostname", repo.WebDomain,
		"-f", "query="+listPullRequestsQuery,
		"-f", "q="+pullRequestSearchQuery(repo, filter),
	)
	if err != nil {
		return nil, err
	}

	output, err := cmdObj.DontLog().RunWithOutput()
	if err != nil {
		return nil, err
	}

	return parsePullRequestListResponse([]byte(output))
}

func (self *GitHubCommands) ghCmdObj(args ...string) (*oscommands.CmdObj, error) {
	ghExe := ghExecutable()
	if ghExe == "" {
		return nil, ErrGhNotInstalled
	}

	return self.cmd.New(append([]string{ghExe}, args...)), nil
}

type pullRequestListResponse struct {
	Data struct {
		Search struct {
			Nodes []pullRequestListNode `json:"nodes"`
		} `json:"search"`
	} `json:"data"`
}

type pullRequestListNode struct {
	Number              int                   `json:"number"`
	Title               string                `json:"title"`
	Url                 string                `json:"url"`
	State               string                `json:"state"`
	IsDraft             bool                  `json:"isDraft"`
	HeadRefName         string                `json:"headRefName"`
	HeadRefOid          string                `json:"headRefOid"`
	BaseRefName         string                `json:"baseRefName"`
	BaseRefOid          string                `json:"baseRefOid"`
	ReviewDecision      string                `json:"reviewDecision"`
	Additions           int                   `json:"additions"`
	Deletions           int                   `json:"deletions"`
	ChangedFiles        int                   `json:"changedFiles"`
	UpdatedAt           time.Time             `json:"updatedAt"`
	Author              GithubRepositoryOwner `json:"author"`
	HeadRepositoryOwner GithubRepositoryOwner `json:"headRepositoryOwner"`
	Comments            struct {
		TotalCount int `json:"totalCount"`
	} `json:"comments"`
	Commits struct {
		Nodes []struct {
			Commit GithubGitObject `json:"commit"`
		} `json:"nodes"`
	} `json:"commits"`
}

func parsePullRequestListResponse(respBytes []byte) ([]*models.GithubPullRequest, error) {
	var result pullRequestListResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, err
	}

	return lo.FilterMap(result.Data.Search.Nodes, func(node pullRequestListNode, _ int) (*models.GithubPullRequest, bool) {
		// Search results that aren't pull requests come back as empty objects
		if node.Number == 0 {
			return nil, false
		}

		checksState := ""
		if len(node.Commits.Nodes) > 0 {
			checksState = node.Commits.Nodes[0].Commit.StatusCheckRollup.State
		}

		return &models.GithubPullRequest{
			Number:              node.Number,
			Title:               node.Title,
			Url:                 node.Url,
			State:               lo.Ternary(node.IsDraft && node.State != "CLOSED", "DRAFT", node.State),
			ChecksState:         checksState,
			HeadRefName:         node.HeadRefName,
			HeadRepositoryOwner: models.GithubRepositoryOwner{Login: node.HeadRepositoryOwner.Login},
			Author:              node.Author.Login,
			BaseRefName:         node.BaseRefName,
			HeadRefOid:          node.HeadRefOid,
			BaseRefOid:          node.BaseRefOid,
			ReviewDecision:      node.ReviewDecision,
			Additions:           node.Additions,
			Deletions:           node.Deletions,
			ChangedFiles:        node.ChangedFiles,
			CommentCount:        node.Comments.TotalCount,
			UpdatedAt:           node.UpdatedAt,
		}, true
	}), nil
}

// ViewPullRequestCmdObj returns a command that prints a pull request's
// description, status, and comments the way gh shows them in a terminal of the
// given width.
func (self *GitHubCommands) ViewPullRequestCmdObj(repo hosting_service.ServiceInfo, number int, width int) (*oscommands.CmdObj, error) {
	cmdObj, err := self.ghCmdObj("pr", "view", strconv.Itoa(number), "--repo", ghRepoArg(repo), "--comments")
	if err != nil {
		return nil, err
	}

	return withGhTerminalOutput(cmdObj, width), nil
}

// withGhTerminalOutput makes gh format its output (colors, wrapping, tables)
// as if it were writing to a terminal of the given width, even though we
// capture it to show it in a view, and without piping it through a pager.
func withGhTerminalOutput(cmdObj *oscommands.CmdObj, width int) *oscommands.CmdObj {
	return cmdObj.AddEnvVars("GH_FORCE_TTY="+strconv.Itoa(width), "GH_PAGER=cat")
}

// ghRepoArg returns the value for gh's --repo flag, which needs the host
// spelled out for GitHub Enterprise repos.
func ghRepoArg(repo hosting_service.ServiceInfo) string {
	return repo.WebDomain + "/" + repo.RepoName
}

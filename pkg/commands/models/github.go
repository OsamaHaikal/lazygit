package models

import (
	"strconv"
	"time"
)

type GithubPullRequest struct {
	HeadRefName         string                `json:"headRefName"`
	Number              int                   `json:"number"`
	Title               string                `json:"title"`
	State               string                `json:"state"` // "MERGED", "OPEN", "CLOSED", "DRAFT"
	ChecksState         string                `json:"checksState"`
	Url                 string                `json:"url"`
	HeadRepositoryOwner GithubRepositoryOwner `json:"headRepositoryOwner"`

	// The fields below are only filled in for the pull requests listed in the
	// pull requests panel, not for the ones matched up with local branches.
	Author         string
	BaseRefName    string
	HeadRefOid     string
	BaseRefOid     string
	ReviewDecision string // "APPROVED", "CHANGES_REQUESTED", "REVIEW_REQUIRED", or empty
	Additions      int
	Deletions      int
	ChangedFiles   int
	CommentCount   int
	UpdatedAt      time.Time
	Assignees      []string
	// The users and teams whose review has been requested and who haven't
	// reviewed yet
	ReviewRequests []string
	Labels         []string
}

func (pr *GithubPullRequest) ID() string {
	return strconv.Itoa(pr.Number)
}

func (pr *GithubPullRequest) Description() string {
	return "#" + strconv.Itoa(pr.Number) + " " + pr.Title
}

func (pr *GithubPullRequest) UserName() string {
	// e.g. 'jesseduffield'
	return pr.HeadRepositoryOwner.Login
}

func (pr *GithubPullRequest) BranchName() string {
	// e.g. 'feature/my-feature'
	return pr.HeadRefName
}

type GithubRepositoryOwner struct {
	Login string `json:"login"`
}

// PullRequestHead is the head commit of a pull request as a ref whose parent is
// the commit where the pull request branched off its base branch, so that
// diffing it against its parent shows the pull request's changes.
type PullRequestHead struct {
	PullRequest  *GithubPullRequest
	MergeBaseOid string
}

func (self *PullRequestHead) FullRefName() string {
	return self.PullRequest.HeadRefOid
}

func (self *PullRequestHead) RefName() string {
	return self.PullRequest.HeadRefOid
}

func (self *PullRequestHead) ShortRefName() string {
	return "#" + strconv.Itoa(self.PullRequest.Number)
}

func (self *PullRequestHead) ParentRefName() string {
	return self.MergeBaseOid
}

func (self *PullRequestHead) Description() string {
	return self.PullRequest.Description()
}

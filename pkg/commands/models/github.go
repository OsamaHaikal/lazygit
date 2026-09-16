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

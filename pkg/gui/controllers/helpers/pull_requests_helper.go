package helpers

import (
	"errors"
	"fmt"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// How long a loaded pull request list counts as fresh enough to show again
// without reloading it when the pull requests panel is focused.
const pullRequestListMaxAge = time.Minute

type PullRequestsHelper struct {
	c            *HelperCommon
	searchHelper *SearchHelper
}

func NewPullRequestsHelper(c *HelperCommon, searchHelper *SearchHelper) *PullRequestsHelper {
	return &PullRequestsHelper{
		c:            c,
		searchHelper: searchHelper,
	}
}

// LoadIfStale loads the pull request list unless it is already loading or was
// loaded recently.
func (self *PullRequestsHelper) LoadIfStale() {
	state := &self.c.Model().PullRequestListState
	if state.Loading || time.Since(state.LoadedAt) < pullRequestListMaxAge {
		return
	}

	self.Load()
}

// Load (re)loads the pull request list in the background. Must be called on
// the UI thread.
func (self *PullRequestsHelper) Load() {
	state := &self.c.Model().PullRequestListState
	state.LoadID++
	loadID := state.LoadID
	state.Loading = true
	filter := state.Filter
	generation := self.c.State().GetRepoGeneration()
	git := self.c.Git()

	self.rerender()

	// The request goes over the network and can take a while, but runs no git
	// commands against the repo, so it mustn't count towards lazygit being busy.
	self.c.OnWorkerBackground(func(gocui.Task) error {
		remote, prs, err := self.loadPullRequestList(git, filter)

		self.c.OnUIThreadBackground(func() error {
			if self.c.State().GetRepoGeneration() != generation || state.LoadID != loadID {
				return nil
			}

			state.Loading = false
			state.LoadedAt = time.Now()
			if err != nil {
				state.Err = err
				// Keep showing the pull requests we have, if any; they are
				// more useful than nothing even if they're a little out of date.
				if len(self.c.Model().PullRequestList) > 0 {
					self.c.ErrorToast(err.Error())
				}
			} else {
				state.Err = nil
				state.Repo = &remote.serviceInfo
				state.RemoteName = remote.remote.Name
				self.c.Model().PullRequestList = prs
			}

			self.rerender()
			return nil
		})
		return nil
	})
}

func (self *PullRequestsHelper) loadPullRequestList(git *commands.GitCommand, filter git_commands.PullRequestFilter) (*githubRemoteInfo, []*models.GithubPullRequest, error) {
	if !git.GitHub.IsGhInstalled() {
		return nil, nil, errors.New(self.c.Tr.GhNeededForPullRequests)
	}

	remotes, err := git.Loaders.RemoteLoader.GetRemotes()
	if err != nil {
		return nil, nil, err
	}

	baseRemote, _ := findGithubBaseRemote(git, remotes)
	if baseRemote == nil {
		return nil, nil, errors.New(self.c.Tr.NoGithubRepoForPullRequests)
	}

	prs, err := git.GitHub.ListPullRequests(baseRemote.serviceInfo, filter)
	if err != nil {
		return nil, nil, errors.New(utils.ResolvePlaceholderString(
			self.c.Tr.LoadPullRequestsError,
			map[string]string{"error": err.Error()},
		))
	}

	return baseRemote, prs, nil
}

// FetchPullRequestHead makes sure the pull request's commits are available
// locally, fetching them if they aren't, and finds where it branched off its
// base branch. Must be called on a worker.
func (self *PullRequestsHelper) FetchPullRequestHead(task gocui.Task, pr *models.GithubPullRequest, remoteName string) (*models.PullRequestHead, error) {
	github := self.c.Git().GitHub
	if !github.HasCommits(pr.HeadRefOid, pr.BaseRefOid) {
		if err := github.FetchPullRequest(task, remoteName, pr.Number, pr.BaseRefName); err != nil {
			return nil, err
		}

		// The list is a snapshot; if the pull request has been pushed to or
		// its base branch has moved on since, we won't have fetched the
		// commits it knows about.
		if !github.HasCommits(pr.HeadRefOid, pr.BaseRefOid) {
			return nil, errors.New(self.c.Tr.PullRequestChangedSinceLoaded)
		}
	}

	mergeBase, err := github.MergeBase(pr.HeadRefOid, pr.BaseRefOid)
	if err != nil {
		return nil, err
	}

	return &models.PullRequestHead{PullRequest: pr, MergeBaseOid: mergeBase}, nil
}

// SetFilter switches the pull request list to the given filter and loads it.
func (self *PullRequestsHelper) SetFilter(filter git_commands.PullRequestFilter) {
	self.c.Model().PullRequestListState.Filter = filter
	self.c.Model().PullRequestList = nil
	self.c.Contexts().PullRequests.SetSelection(0)
	self.Load()
}

var PullRequestFilters = []git_commands.PullRequestFilter{
	git_commands.PullRequestFilterOpen,
	git_commands.PullRequestFilterReviewRequested,
	git_commands.PullRequestFilterMine,
	git_commands.PullRequestFilterAssigned,
	git_commands.PullRequestFilterMerged,
	git_commands.PullRequestFilterClosed,
	git_commands.PullRequestFilterAll,
}

func (self *PullRequestsHelper) FilterLabel(filter git_commands.PullRequestFilter) string {
	switch filter {
	case git_commands.PullRequestFilterOpen:
		return self.c.Tr.PullRequestFilterOpen
	case git_commands.PullRequestFilterReviewRequested:
		return self.c.Tr.PullRequestFilterReviewRequested
	case git_commands.PullRequestFilterMine:
		return self.c.Tr.PullRequestFilterMine
	case git_commands.PullRequestFilterAssigned:
		return self.c.Tr.PullRequestFilterAssigned
	case git_commands.PullRequestFilterMerged:
		return self.c.Tr.PullRequestFilterMerged
	case git_commands.PullRequestFilterClosed:
		return self.c.Tr.PullRequestFilterClosed
	case git_commands.PullRequestFilterAll:
		return self.c.Tr.PullRequestFilterAll
	}

	panic(fmt.Sprintf("Unexpected pull request filter: %d", filter))
}

// subtitle shows which filter is active, unless it's the default one; a
// subtitle is squeezed in next to the panel's tabs, which already take up most
// of the space there.
func (self *PullRequestsHelper) subtitle() string {
	filter := self.c.Model().PullRequestListState.Filter
	if filter == git_commands.PullRequestFilterOpen {
		return ""
	}

	return self.FilterLabel(filter)
}

func (self *PullRequestsHelper) rerender() {
	context := self.c.Contexts().PullRequests
	context.GetView().Subtitle = self.subtitle()
	self.searchHelper.ReApplyFilter(context)
	self.c.PostRefreshUpdate(context)
}

// PullRequestOfCommitFiles returns the pull request whose changes the commit
// files view is showing, along with the commit whose diff it shows, which is
// either the pull request's head (for all of its changes) or one of its
// commits. It returns nil if the view isn't showing a pull request's changes,
// or is showing a diff that GitHub has no way to refer to, such as the
// combined diff of a range of commits.
func (self *PullRequestsHelper) PullRequestOfCommitFiles() (*models.GithubPullRequest, string) {
	commitFilesContext := self.c.Contexts().CommitFiles
	if commitFilesContext.GetRefRange() != nil || self.c.Modes().Diffing.Active() {
		return nil, ""
	}

	switch ref := commitFilesContext.GetRef().(type) {
	case *models.PullRequestHead:
		return ref.PullRequest, ref.RefName()
	case *models.Commit:
		parentContext := commitFilesContext.GetParentContext()
		if parentContext == nil || parentContext.GetKey() != context.SUB_COMMITS_CONTEXT_KEY {
			return nil, ""
		}
		if head, ok := self.c.Contexts().SubCommits.GetRef().(*models.PullRequestHead); ok {
			return head.PullRequest, ref.Hash()
		}
	}

	return nil, ""
}

// WithReviewComments calls f with the pull request's review comments, loading
// them in the background first unless they're loaded already. f is called on
// the UI thread, and not at all if loading them fails.
func (self *PullRequestsHelper) WithReviewComments(pr *models.GithubPullRequest, f func([]*models.PullRequestReviewComment)) {
	state := &self.c.Model().PullRequestListState
	if cached, ok := state.ReviewComments[pr.Number]; ok && cached.PullRequestUpdatedAt.Equal(pr.UpdatedAt) {
		f(cached.Comments)
		return
	}

	repo := *state.Repo
	generation := self.c.State().GetRepoGeneration()
	self.c.OnWorkerBackground(func(gocui.Task) error {
		comments, err := self.c.Git().GitHub.ListPullRequestReviewComments(repo, pr.Number)
		if err != nil {
			self.c.Log.Error(err)
			return nil
		}

		self.c.OnUIThreadBackground(func() error {
			if self.c.State().GetRepoGeneration() != generation {
				return nil
			}

			if state.ReviewComments == nil {
				state.ReviewComments = map[int]*types.PullRequestReviewComments{}
			}
			state.ReviewComments[pr.Number] = &types.PullRequestReviewComments{
				PullRequestUpdatedAt: pr.UpdatedAt,
				Comments:             comments,
			}
			f(comments)
			return nil
		})
		return nil
	})
}

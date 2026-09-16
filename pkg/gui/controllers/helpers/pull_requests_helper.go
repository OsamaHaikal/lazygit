package helpers

import (
	"errors"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/hosting_service"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
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
		repo, prs, err := self.loadPullRequestList(git, filter)

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
				state.Repo = repo
				self.c.Model().PullRequestList = prs
			}

			self.rerender()
			return nil
		})
		return nil
	})
}

func (self *PullRequestsHelper) loadPullRequestList(git *commands.GitCommand, filter git_commands.PullRequestFilter) (*hosting_service.ServiceInfo, []*models.GithubPullRequest, error) {
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

	return &baseRemote.serviceInfo, prs, nil
}

func (self *PullRequestsHelper) rerender() {
	context := self.c.Contexts().PullRequests
	self.searchHelper.ReApplyFilter(context)
	self.c.PostRefreshUpdate(context)
}

package helpers

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/hosting_service"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// How many of the things the AI says it's doing are shown while it writes a
// guide
const maxGuideProgressLines = 5

// ShowGuide shows the AI-written guide to the pull request at the given head,
// writing one first if there isn't one yet. Must be called on the UI thread.
func (self *PullRequestsHelper) ShowGuide(pr *models.GithubPullRequest, head *models.PullRequestHead) {
	listState := &self.c.Model().PullRequestListState
	existing := listState.Guides[pr.Number]
	if existing == nil || existing.Err != nil ||
		existing.Head.RefName() != head.RefName() || existing.Head.MergeBaseOid != head.MergeBaseOid {
		self.writeGuide(pr, head, false)
	}

	guideContext := self.c.Contexts().PullRequestGuide
	pullRequestsContext := self.c.Contexts().PullRequests
	guideContext.SetNumber(pr.Number)
	guideContext.SetSelection(0)
	guideContext.SetParentContext(pullRequestsContext)
	guideContext.SetWindowName(pullRequestsContext.GetWindowName())
	guideContext.GetView().TitlePrefix = pullRequestsContext.GetView().TitlePrefix
	guideContext.GetView().Title = self.c.Tr.PullRequestGuideTitle + " (" + utils.TruncateWithEllipsis(pr.Description(), 50) + ")"

	self.c.PostRefreshUpdate(guideContext)
	self.c.Context().Push(guideContext, types.OnFocusOpts{})
}

// RewriteGuide asks for the guide to the pull request to be written again, even
// if there's one already. Must be called on the UI thread.
func (self *PullRequestsHelper) RewriteGuide(pr *models.GithubPullRequest, head *models.PullRequestHead) {
	self.writeGuide(pr, head, true)
	self.c.Contexts().PullRequestGuide.SetSelection(0)
	self.c.PostRefreshUpdate(self.c.Contexts().PullRequestGuide)
}

// writeGuide gets the guide to the pull request in the background: from the
// cache, unless force is true, or else by asking the AI to write it.
func (self *PullRequestsHelper) writeGuide(pr *models.GithubPullRequest, head *models.PullRequestHead, force bool) {
	listState := &self.c.Model().PullRequestListState
	if listState.Guides == nil {
		listState.Guides = map[int]*types.PullRequestGuideState{}
	}
	state := &types.PullRequestGuideState{Head: head, StartedAt: time.Now()}
	listState.Guides[pr.Number] = state

	repo := *listState.Repo
	guideConfig := self.c.UserConfig().Git.PullRequestGuide
	generation := self.c.State().GetRepoGeneration()
	git := self.c.Git()

	// Only ever touch the state on the UI thread, and not at all once the repo
	// has been switched or the guide has been asked for again
	onUIThread := func(f func()) {
		self.c.OnUIThreadBackground(func() error {
			if self.c.State().GetRepoGeneration() == generation && listState.Guides[pr.Number] == state {
				f()
			}
			return nil
		})
	}

	// Writing a guide takes minutes and runs no commands that change the repo,
	// so it mustn't count towards lazygit being busy
	done := make(chan struct{})
	self.c.OnWorkerBackground(func(gocui.Task) error {
		defer close(done)
		guide, err := self.getGuide(git, repo, pr, head, guideConfig, force,
			func(provider string) {
				onUIThread(func() { state.Provider = provider })
			},
			func(progress string) {
				onUIThread(func() {
					state.Progress = append(state.Progress, progress)
					state.Progress = state.Progress[max(len(state.Progress)-maxGuideProgressLines, 0):]
					self.rerenderGuideWhileWriting(pr.Number)
				})
			},
		)

		onUIThread(func() {
			state.Guide, state.Err = guide, err
			guideContext := self.c.Contexts().PullRequestGuide
			if self.c.Context().IsCurrentOrParent(guideContext) && guideContext.GetNumber() == pr.Number {
				self.c.PostRefreshUpdate(guideContext)
			} else if err == nil {
				self.c.Toast(utils.ResolvePlaceholderString(self.c.Tr.PullRequestGuideReady,
					map[string]string{"number": pr.ID()}))
			}
		})
		return nil
	})

	self.c.OnWorkerBackground(func(gocui.Task) error {
		self.rerenderWhileWritingGuide(pr.Number, done, onUIThread)
		return nil
	})
}

func (self *PullRequestsHelper) getGuide(
	git *commands.GitCommand,
	repo hosting_service.ServiceInfo,
	pr *models.GithubPullRequest,
	head *models.PullRequestHead,
	guideConfig config.PullRequestGuideConfig,
	force bool,
	onProviderResolved func(string),
	onProgress func(string),
) (*git_commands.PullRequestGuide, error) {
	provider, err := git.GitHub.ResolveGuideProvider(guideConfig.Provider)
	if errors.Is(err, git_commands.ErrNoGuideProvider) {
		return nil, errors.New(self.c.Tr.NoGuideProvider)
	}
	if err != nil {
		return nil, err
	}
	onProviderResolved(provider)

	description, err := git.GitHub.GetPullRequestDescription(repo, pr.Number)
	if err != nil {
		return nil, err
	}

	opts := git_commands.GeneratePullRequestGuideOpts{
		Provider:    provider,
		Model:       guideConfig.Model,
		Number:      pr.Number,
		Title:       pr.Title,
		Description: description,
		MergeBase:   head.MergeBaseOid,
		Head:        head.RefName(),
		OnProgress:  onProgress,
	}
	cachePath, cachePathErr := config.PullRequestGuideCachePath(git_commands.GuideCacheKey(opts))

	if !force && cachePathErr == nil {
		if guide, err := loadCachedGuide(cachePath); err == nil {
			return guide, nil
		}
	}

	guide, err := git.GitHub.GeneratePullRequestGuide(opts)
	if err != nil {
		return nil, err
	}

	if cachePathErr == nil {
		if err := saveCachedGuide(cachePath, guide); err != nil {
			self.c.Log.Error(err)
		}
	}

	return guide, nil
}

// rerenderWhileWritingGuide keeps the elapsed time shown while a guide is being
// written up to date, until done is closed.
func (self *PullRequestsHelper) rerenderWhileWritingGuide(number int, done <-chan struct{}, onUIThread func(func())) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			onUIThread(func() { self.rerenderGuideWhileWriting(number) })
		}
	}
}

// rerenderGuideWhileWriting shows the latest elapsed time and progress of the
// guide being written, if it's being looked at. Must be called on the UI
// thread.
func (self *PullRequestsHelper) rerenderGuideWhileWriting(number int) {
	guideContext := self.c.Contexts().PullRequestGuide
	if self.c.Context().IsCurrent(guideContext) && guideContext.GetNumber() == number {
		guideContext.HandleRenderToMain()
	}
}

func loadCachedGuide(path string) (*git_commands.PullRequestGuide, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var guide git_commands.PullRequestGuide
	if err := json.Unmarshal(content, &guide); err != nil {
		return nil, err
	}
	return &guide, nil
}

func saveCachedGuide(path string, guide *git_commands.PullRequestGuide) error {
	content, err := json.Marshal(guide)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

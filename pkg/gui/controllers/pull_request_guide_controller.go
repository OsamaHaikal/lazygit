package controllers

import (
	"strconv"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/patch"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

type PullRequestGuideController struct {
	baseController
	*ListControllerTrait[*git_commands.GuideChapter]
	c *ControllerCommon
}

var _ types.IController = &PullRequestGuideController{}

func NewPullRequestGuideController(c *ControllerCommon) *PullRequestGuideController {
	return &PullRequestGuideController{
		baseController: baseController{},
		ListControllerTrait: NewListControllerTrait(
			c,
			c.Contexts().PullRequestGuide,
			c.Contexts().PullRequestGuide.GetSelected,
			c.Contexts().PullRequestGuide.GetSelectedItems,
		),
		c: c,
	}
}

func (self *PullRequestGuideController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	return []*types.Binding{
		{
			Keys:              opts.GetKeys(opts.Config.Universal.GoInto),
			Handler:           self.withItem(self.viewChapterFiles),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.ViewGuideChapterFiles,
			Tooltip:           self.c.Tr.ViewGuideChapterFilesTooltip,
			DisplayOnScreen:   true,
		},
		{
			Keys:              opts.GetKeys(opts.Config.PullRequests.ViewFiles),
			Handler:           self.viewAllFiles,
			GetDisabledReason: self.notWhileWriting,
			Description:       self.c.Tr.ViewPullRequestFiles,
			Tooltip:           self.c.Tr.ViewPullRequestFilesTooltip,
		},
		{
			Keys:        opts.GetKeys(opts.Config.PullRequests.GuideSettings),
			Handler:     self.c.Helpers().PullRequests.OpenGuideSettingsMenu,
			Description: self.c.Tr.PullRequestGuideSettings,
			Tooltip:     self.c.Tr.PullRequestGuideSettingsTooltip,
			OpensMenu:   true,
		},
		{
			Keys:              opts.GetKeys(opts.Config.Universal.Remove),
			Handler:           self.cancel,
			GetDisabledReason: self.whileWriting,
			Description:       self.c.Tr.CancelPullRequestGuide,
			DisplayOnScreen:   true,
		},
		{
			Keys:              opts.GetKeys(opts.Config.Universal.Refresh),
			Handler:           self.rewrite,
			GetDisabledReason: self.notWhileWriting,
			Description:       self.c.Tr.RegeneratePullRequestGuide,
		},
	}
}

func (self *PullRequestGuideController) GetOnRenderToMain() func() {
	return func() {
		self.c.RenderToMainViews(types.RefreshMainOpts{
			Pair: self.c.MainViewPairs().Normal,
			Main: &types.ViewUpdateOpts{
				Title: self.c.Tr.PullRequestGuideTitle,
				Task:  types.NewRenderStringTask(self.mainViewContent()),
			},
		})
	}
}

func (self *PullRequestGuideController) mainViewContent() string {
	state := self.context().GetGuideState()
	if state == nil {
		return ""
	}
	number := strconv.Itoa(self.context().GetNumber())

	switch {
	case state.Err != nil:
		return utils.ResolvePlaceholderString(self.c.Tr.PullRequestGuideFailed, map[string]string{
			"number": number,
			"error":  state.Err.Error(),
			"key":    self.c.UserConfig().Keybinding.Universal.Refresh[0],
		})
	case state.Guide == nil:
		return utils.ResolvePlaceholderString(self.c.Tr.WritingPullRequestGuide, map[string]string{
			"provider": lo.CoalesceOrEmpty(state.Provider, self.c.Helpers().PullRequests.GuideSettings().Provider),
			"number":   number,
			"elapsed":  presentation.FormatElapsed(time.Since(state.StartedAt)),
		}) + presentation.FormatGuideProgress(state.Progress)
	}

	chapter := self.context().GetSelected()
	if chapter == nil {
		return ""
	}

	writtenBy := utils.ResolvePlaceholderString(self.c.Tr.PullRequestGuideWrittenBy,
		map[string]string{"provider": state.Guide.Provider})
	if state.Guide.Model != "" {
		writtenBy = utils.ResolvePlaceholderString(self.c.Tr.PullRequestGuideWrittenByModel,
			map[string]string{"provider": state.Guide.Provider, "model": state.Guide.Model})
	}

	notice := ""
	if pr := self.pullRequest(); pr.HeadRefOid != state.Head.RefName() {
		notice = utils.ResolvePlaceholderString(self.c.Tr.PullRequestChangedSinceGuide,
			map[string]string{"key": self.c.UserConfig().Keybinding.Universal.Refresh[0]})
	}

	return presentation.FormatPullRequestGuideChapter(state.Guide, chapter, notice, writtenBy, self.c.Tr)
}

func (self *PullRequestGuideController) GetOnDoubleClick() func() error {
	return self.withItemGraceful(self.viewChapterFiles)
}

// viewChapterFiles shows the files that the chapter's hunks are in, so that
// their diffs can be gone through like a commit's.
func (self *PullRequestGuideController) viewChapterFiles(chapter *git_commands.GuideChapter) error {
	state := self.context().GetGuideState()
	paths := lo.Uniq(lo.FlatMap(state.Guide.Units(*chapter), func(unit git_commands.GuideReviewUnit, _ int) []string {
		// A renamed file only shows up as a rename if both of its paths are
		// included
		return lo.Compact([]string{unit.OldPath, unit.Path})
	}))

	self.c.Helpers().CommitFiles.ViewCommitFiles(helpers.ViewCommitFilesOpts{
		Ref:           state.Head,
		Context:       self.context(),
		Paths:         paths,
		TitleRef:      chapter.Title,
		LinesToSelect: firstChangedLines(state.Guide.Units(*chapter)),
	})
	return nil
}

// firstChangedLines returns the first line that the given units change in
// each of their files, so that going into a file's diff shows what the
// chapter is about rather than whatever the file's first change is.
func firstChangedLines(units []git_commands.GuideReviewUnit) map[string]patch.FileLine {
	lines := map[string]patch.FileLine{}
	for _, unit := range units {
		if _, ok := lines[unit.Path]; ok {
			continue
		}

		unitPatch := patch.Parse(unit.Diff)
		if line, ok := unitPatch.FileLineOfLine(unitPatch.GetNextChangeIdx(0)); ok {
			lines[unit.Path] = line
		}
	}
	return lines
}

func (self *PullRequestGuideController) viewAllFiles() error {
	state := self.context().GetGuideState()
	self.c.Helpers().CommitFiles.ViewCommitFiles(helpers.ViewCommitFilesOpts{
		Ref:     state.Head,
		Context: self.context(),
	})
	return nil
}

// pullRequest returns the pull request whose guide is shown, as it is in the
// pull request list, which is newer than the one the guide was written for if
// the list has been reloaded since
func (self *PullRequestGuideController) pullRequest() *models.GithubPullRequest {
	pr, found := lo.Find(self.c.Model().PullRequestList, func(pr *models.GithubPullRequest) bool {
		return pr.Number == self.context().GetNumber()
	})
	if !found {
		return self.context().GetGuideState().Head.PullRequest
	}
	return pr
}

func (self *PullRequestGuideController) rewrite() error {
	state := self.context().GetGuideState()
	pr := self.pullRequest()

	write := func() error {
		if pr.HeadRefOid == state.Head.RefName() {
			self.c.Helpers().PullRequests.RewriteGuide(pr, state.Head)
			return nil
		}

		// The pull request has changed since the guide was written, so the
		// new guide needs to be for its new commits
		remoteName := self.c.Model().PullRequestListState.RemoteName
		return self.c.WithWaitingStatus(self.c.Tr.FetchingPullRequest, func(task gocui.Task) error {
			head, err := self.c.Helpers().PullRequests.FetchPullRequestHead(task, pr, remoteName)
			if err != nil {
				return err
			}
			self.c.OnUIThread(func() error {
				self.c.Helpers().PullRequests.RewriteGuide(pr, head)
				return nil
			})
			return nil
		})
	}

	// Retrying after a failure costs nothing, so only ask when there's a guide
	// that would be replaced
	if state.Err != nil {
		return write()
	}

	self.c.Confirm(types.ConfirmOpts{
		Title: self.c.Tr.RegeneratePullRequestGuideTitle,
		Prompt: utils.ResolvePlaceholderString(self.c.Tr.RegeneratePullRequestGuidePrompt,
			map[string]string{"provider": state.Guide.Provider}),
		HandleConfirm: write,
	})
	return nil
}

func (self *PullRequestGuideController) cancel() error {
	self.context().GetGuideState().Cancel()
	return nil
}

func (self *PullRequestGuideController) whileWriting() *types.DisabledReason {
	if state := self.context().GetGuideState(); state == nil || !state.IsGenerating() {
		return &types.DisabledReason{Text: self.c.Tr.PullRequestGuideNotBeingWritten}
	}
	return nil
}

func (self *PullRequestGuideController) notWhileWriting() *types.DisabledReason {
	if state := self.context().GetGuideState(); state == nil || state.IsGenerating() {
		return &types.DisabledReason{Text: self.c.Tr.PullRequestGuideStillBeingWritten}
	}
	return nil
}

func (self *PullRequestGuideController) context() *context.PullRequestGuideContext {
	return self.c.Contexts().PullRequestGuide
}

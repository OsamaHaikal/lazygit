package controllers

import (
	"fmt"
	"strconv"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// Lets you comment on the lines of a pull request's diff while looking at it in
// the patch building view, which is where you end up when entering a file of a
// pull request or of one of its commits.
type PullRequestLineCommentController struct {
	baseController
	c *ControllerCommon
}

var _ types.IController = &PullRequestLineCommentController{}

func NewPullRequestLineCommentController(c *ControllerCommon) *PullRequestLineCommentController {
	return &PullRequestLineCommentController{
		baseController: baseController{},
		c:              c,
	}
}

func (self *PullRequestLineCommentController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	return []*types.Binding{
		{
			Keys:              opts.GetKeys(opts.Config.PullRequests.Comment),
			Handler:           self.comment,
			GetDisabledReason: self.getDisabledReason,
			Description:       self.c.Tr.CommentOnPullRequestLines,
			Tooltip:           self.c.Tr.CommentOnPullRequestLinesTooltip,
		},
	}
}

func (self *PullRequestLineCommentController) Context() types.Context {
	return self.c.Contexts().CustomPatchBuilder
}

func (self *PullRequestLineCommentController) getDisabledReason() *types.DisabledReason {
	if pr, _ := self.c.Helpers().PullRequests.PullRequestOfCommitFiles(); pr == nil {
		return &types.DisabledReason{Text: self.c.Tr.NotViewingPullRequestDiff}
	}

	state := self.c.Contexts().CustomPatchBuilder.GetState()
	if state == nil {
		return &types.DisabledReason{Text: self.c.Tr.NoFileLinesSelected}
	}
	if _, _, ok := state.SelectedFileLines(); !ok {
		return &types.DisabledReason{Text: self.c.Tr.NoFileLinesSelected}
	}

	return nil
}

func (self *PullRequestLineCommentController) comment() error {
	pr, commitOid := self.c.Helpers().PullRequests.PullRequestOfCommitFiles()
	startLine, endLine, _ := self.c.Contexts().CustomPatchBuilder.GetState().SelectedFileLines()
	path := self.c.Contexts().CommitFiles.GetSelectedPath()
	repo := *self.c.Model().PullRequestListState.Repo

	lines := strconv.Itoa(endLine.Number)
	if startLine != endLine {
		lines = fmt.Sprintf("%d-%d", startLine.Number, endLine.Number)
	}

	self.c.Prompt(types.PromptOpts{
		Title: utils.ResolvePlaceholderString(self.c.Tr.CommentOnPullRequestLinesTitle,
			map[string]string{"path": path, "lines": lines}),
		FindSuggestionsFunc:      self.c.Helpers().PullRequests.MentionSuggestionsFunc(),
		SuggestionsCompleteInput: true,
		HandleConfirm: func(body string) error {
			return self.c.WithWaitingStatus(self.c.Tr.PostingComment, func(gocui.Task) error {
				self.c.LogAction(self.c.Tr.Actions.CommentOnPullRequestLines)
				err := self.c.Git().GitHub.CommentOnPullRequestLines(repo, git_commands.PullRequestLineCommentOpts{
					Number:    pr.Number,
					CommitOid: commitOid,
					Path:      path,
					StartLine: startLine,
					EndLine:   endLine,
					Body:      body,
				})
				if err != nil {
					return err
				}

				self.c.Toast(self.c.Tr.PullRequestCommentAdded)
				return nil
			})
		},
	})
	return nil
}

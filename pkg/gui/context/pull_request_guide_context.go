package context

import (
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

// The chapters of an AI-written guide to a pull request
type PullRequestGuideContext struct {
	*ListViewModel[*git_commands.GuideChapter]
	*ListContextTrait

	// The number of the pull request whose guide is shown
	number int
}

var _ types.IListContext = (*PullRequestGuideContext)(nil)

func NewPullRequestGuideContext(c *ContextCommon) *PullRequestGuideContext {
	self := &PullRequestGuideContext{}

	viewModel := NewListViewModel(func() []*git_commands.GuideChapter {
		state := self.GetGuideState()
		if state == nil || state.Guide == nil {
			return nil
		}
		return lo.Map(state.Guide.Chapters, func(_ git_commands.GuideChapter, i int) *git_commands.GuideChapter {
			return &state.Guide.Chapters[i]
		})
	})

	getDisplayStrings := func(_ int, _ int) [][]string {
		return lo.Map(viewModel.getModel(), func(chapter *git_commands.GuideChapter, i int) []string {
			return []string{style.FgBlue.Sprint(fmt.Sprintf("%d.", i+1)), chapter.Title}
		})
	}

	self.ListViewModel = viewModel
	self.ListContextTrait = &ListContextTrait{
		Context: NewSimpleContext(NewBaseContext(NewBaseContextOpts{
			View:       c.Views().PullRequestGuide,
			WindowName: "branches",
			Key:        PULL_REQUEST_GUIDE_CONTEXT_KEY,
			Kind:       types.SIDE_CONTEXT,
			Focusable:  true,
			Transient:  true,
		})),
		ListRenderer: ListRenderer{
			list:              viewModel,
			getDisplayStrings: getDisplayStrings,
		},
		c: c,
	}

	return self
}

func (self *PullRequestGuideContext) SetNumber(number int) {
	self.number = number
}

func (self *PullRequestGuideContext) GetNumber() int {
	return self.number
}

// GetGuideState returns the state of the guide being shown, or nil if none has
// been asked for.
func (self *PullRequestGuideContext) GetGuideState() *types.PullRequestGuideState {
	return self.c.Model().PullRequestListState.Guides[self.number]
}

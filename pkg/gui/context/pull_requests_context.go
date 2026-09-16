package context

import (
	"strconv"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type PullRequestsContext struct {
	*FilteredListViewModel[*models.GithubPullRequest]
	*ListContextTrait
}

var _ types.IListContext = (*PullRequestsContext)(nil)

func NewPullRequestsContext(c *ContextCommon) *PullRequestsContext {
	viewModel := NewFilteredListViewModel(
		func() []*models.GithubPullRequest { return c.Model().PullRequestList },
		func(pr *models.GithubPullRequest) []string {
			return []string{strconv.Itoa(pr.Number), pr.Title, pr.Author, pr.HeadRefName}
		},
	)

	getDisplayStrings := func(_ int, _ int) [][]string {
		return presentation.GetPullRequestListDisplayStrings(
			viewModel.GetFilteredList(),
			c.Tr,
		)
	}

	return &PullRequestsContext{
		FilteredListViewModel: viewModel,
		ListContextTrait: &ListContextTrait{
			Context: NewSimpleContext(NewBaseContext(NewBaseContextOpts{
				View:       c.Views().PullRequests,
				WindowName: "branches",
				Key:        PULL_REQUESTS_CONTEXT_KEY,
				Kind:       types.SIDE_CONTEXT,
				Focusable:  true,
			})),
			ListRenderer: ListRenderer{
				list:              viewModel,
				getDisplayStrings: getDisplayStrings,
			},
			c: c,
		},
	}
}

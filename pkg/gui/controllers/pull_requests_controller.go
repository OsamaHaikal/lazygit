package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type PullRequestsController struct {
	baseController
	*ListControllerTrait[*models.GithubPullRequest]
	c *ControllerCommon
}

var _ types.IController = &PullRequestsController{}

func NewPullRequestsController(
	c *ControllerCommon,
) *PullRequestsController {
	return &PullRequestsController{
		baseController: baseController{},
		ListControllerTrait: NewListControllerTrait(
			c,
			c.Contexts().PullRequests,
			c.Contexts().PullRequests.GetSelected,
			c.Contexts().PullRequests.GetSelectedItems,
		),
		c: c,
	}
}

func (self *PullRequestsController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	bindings := []*types.Binding{
		{
			Keys:        opts.GetKeys(opts.Config.Universal.Refresh),
			Handler:     self.refresh,
			Description: self.c.Tr.RefreshPullRequests,
		},
	}

	return bindings
}

func (self *PullRequestsController) GetOnFocus() func(types.OnFocusOpts) {
	return func(types.OnFocusOpts) {
		self.c.Helpers().PullRequests.LoadIfStale()
	}
}

func (self *PullRequestsController) GetOnRenderToMain() func() {
	return func() {
		self.c.RenderToMainViews(types.RefreshMainOpts{
			Pair: self.c.MainViewPairs().Normal,
			Main: &types.ViewUpdateOpts{
				Title: self.c.Tr.PullRequestsTitle,
				Task:  self.mainViewTask(),
			},
		})
	}
}

func (self *PullRequestsController) mainViewTask() types.UpdateTask {
	state := self.c.Model().PullRequestListState
	pr := self.context().GetSelected()
	if pr == nil {
		switch {
		case state.Err != nil:
			return types.NewRenderStringTask(state.Err.Error())
		case state.Loading:
			return types.NewRenderStringTask(self.c.Tr.FetchingPullRequests + "...")
		default:
			return types.NewRenderStringTask(self.c.Tr.NoPullRequests)
		}
	}

	cmdObj, err := self.c.Git().GitHub.ViewPullRequestCmdObj(*state.Repo, pr.Number, self.c.Views().Main.InnerWidth())
	if err != nil {
		return types.NewRenderStringTask(err.Error())
	}

	return types.NewRunCommandTask(cmdObj.GetCmd())
}

func (self *PullRequestsController) refresh() error {
	self.c.Helpers().PullRequests.Load()
	return nil
}

func (self *PullRequestsController) context() *context.PullRequestsContext {
	return self.c.Contexts().PullRequests
}

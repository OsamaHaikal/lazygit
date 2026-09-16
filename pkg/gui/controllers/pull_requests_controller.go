package controllers

import (
	"strconv"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/hosting_service"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
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
			Keys:              opts.GetKeys(opts.Config.Universal.GoInto),
			Handler:           self.withItem(self.viewCommits),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.ViewPullRequestCommits,
		},
		{
			Keys:              opts.GetKeys(opts.Config.PullRequests.Guide),
			Handler:           self.withItem(self.guide),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.OpenPullRequestGuide,
			Tooltip:           self.c.Tr.OpenPullRequestGuideTooltip,
			DisplayOnScreen:   true,
		},
		{
			Keys:              opts.GetKeys(opts.Config.PullRequests.ViewFiles),
			Handler:           self.withItem(self.viewFiles),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.ViewPullRequestFiles,
			Tooltip:           self.c.Tr.ViewPullRequestFilesTooltip,
			DisplayOnScreen:   true,
		},
		{
			Keys:              opts.GetKeys(opts.Config.Universal.Select),
			Handler:           self.withItem(self.checkout),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.Checkout,
			Tooltip:           self.c.Tr.CheckoutPullRequestTooltip,
			DisplayOnScreen:   true,
		},
		{
			Keys:              opts.GetKeys(opts.Config.PullRequests.Merge),
			Handler:           self.withItem(self.merge),
			GetDisabledReason: self.require(self.singleItemSelected(self.isOpen)),
			Description:       self.c.Tr.MergePullRequestOptions,
			Tooltip:           self.c.Tr.MergePullRequestTooltip,
			OpensMenu:         true,
			DisplayOnScreen:   true,
		},
		{
			Keys:              opts.GetKeys(opts.Config.PullRequests.Review),
			Handler:           self.withItem(self.review),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.ReviewPullRequestOptions,
			OpensMenu:         true,
			DisplayOnScreen:   true,
		},
		{
			Keys:              opts.GetKeys(opts.Config.PullRequests.Comment),
			Handler:           self.withItem(self.comment),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.AddPullRequestComment,
			DisplayOnScreen:   true,
		},
		{
			Keys:              opts.GetKeys(opts.Config.PullRequests.Edit),
			Handler:           self.withItem(self.edit),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.EditPullRequestOptions,
			Tooltip:           self.c.Tr.EditPullRequestTooltip,
			OpensMenu:         true,
			DisplayOnScreen:   true,
		},
		{
			Keys:              opts.GetKeys(opts.Config.PullRequests.ToggleDraft),
			Handler:           self.withItem(self.toggleDraft),
			GetDisabledReason: self.require(self.singleItemSelected(self.isOpen)),
			Description:       self.c.Tr.TogglePullRequestDraft,
			Tooltip:           self.c.Tr.TogglePullRequestDraftTooltip,
		},
		{
			Keys:              opts.GetKeys(opts.Config.PullRequests.CloseOrReopen),
			Handler:           self.withItem(self.closeOrReopen),
			GetDisabledReason: self.require(self.singleItemSelected(self.isNotMerged)),
			Description:       self.c.Tr.CloseOrReopenPullRequest,
		},
		{
			Keys:              opts.GetKeys(opts.Config.Branches.OpenPullRequestInBrowser),
			Handler:           self.withItem(self.openInBrowser),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.OpenPullRequestInBrowser,
		},
		{
			Keys:              opts.GetKeys(opts.Config.Branches.CopyPullRequestURL),
			Handler:           self.withItem(self.copyURL),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.CopyPullRequestURL,
		},
		{
			Keys:            opts.GetKeys(opts.Config.PullRequests.Filter),
			Handler:         self.filter,
			Description:     self.c.Tr.FilterPullRequests,
			Tooltip:         self.c.Tr.FilterPullRequestsTooltip,
			OpensMenu:       true,
			DisplayOnScreen: true,
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.Refresh),
			Handler:     self.refresh,
			Description: self.c.Tr.RefreshPullRequests,
		},
	}

	return bindings
}

func (self *PullRequestsController) GetOnDoubleClick() func() error {
	return self.withItemGraceful(self.viewCommits)
}

func (self *PullRequestsController) GetOnFocus() func(types.OnFocusOpts) {
	return func(types.OnFocusOpts) {
		self.c.Helpers().PullRequests.LoadIfStale()
	}
}

func (self *PullRequestsController) GetOnRenderToMain() func() {
	return func() {
		pr := self.context().GetSelected()
		if pr == nil || pr.ReviewThreadCount == 0 {
			self.c.RenderToMainViews(types.RefreshMainOpts{
				Pair: self.c.MainViewPairs().Normal,
				Main: &types.ViewUpdateOpts{
					Title: self.c.Tr.PullRequestsTitle,
					Task:  self.mainViewTask(),
				},
			})
			return
		}

		// The review comments on the code aren't part of what gh shows for a
		// pull request, so show them below it once they're loaded.
		reviewCommentsView := func(content string) *types.ViewUpdateOpts {
			return &types.ViewUpdateOpts{
				Title: self.c.Tr.ReviewCommentsTitle,
				Task:  types.NewRenderStringTask(content),
			}
		}
		self.c.RenderToMainViews(types.RefreshMainOpts{
			Pair: self.c.MainViewPairs().Normal,
			Main: &types.ViewUpdateOpts{
				Title: self.c.Tr.PullRequestsTitle,
				Task:  self.mainViewTask(),
			},
			Secondary: reviewCommentsView(self.c.Tr.LoadingReviewComments),
		})

		self.c.Helpers().PullRequests.WithReviewComments(pr, func(comments []*models.PullRequestReviewComment) {
			if self.context().GetSelected() != pr || self.c.Context().CurrentSide() != self.context() {
				return
			}

			self.c.RenderToMainViews(types.RefreshMainOpts{
				Pair:      self.c.MainViewPairs().Normal,
				Secondary: reviewCommentsView(presentation.FormatPullRequestReviewComments(comments, self.c.Tr)),
			})
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

func (self *PullRequestsController) viewCommits(pr *models.GithubPullRequest) error {
	return self.withPullRequestHead(pr, func(head *models.PullRequestHead) error {
		return self.c.Helpers().SubCommits.ViewSubCommits(helpers.ViewSubCommitsOpts{
			Ref:          head,
			RefToExclude: head.MergeBaseOid,
			TitleRef:     head.Description(),
			Context:      self.context(),
		})
	})
}

func (self *PullRequestsController) viewFiles(pr *models.GithubPullRequest) error {
	return self.withPullRequestHead(pr, func(head *models.PullRequestHead) error {
		self.c.Helpers().CommitFiles.ViewCommitFiles(helpers.ViewCommitFilesOpts{
			Ref:     head,
			Context: self.context(),
		})
		return nil
	})
}

func (self *PullRequestsController) guide(pr *models.GithubPullRequest) error {
	return self.withPullRequestHead(pr, func(head *models.PullRequestHead) error {
		self.c.Helpers().PullRequests.ShowGuide(pr, head)
		return nil
	})
}

// withPullRequestHead fetches the pull request if needed, then calls f with its
// head on the UI thread.
func (self *PullRequestsController) withPullRequestHead(pr *models.GithubPullRequest, f func(*models.PullRequestHead) error) error {
	remoteName := self.c.Model().PullRequestListState.RemoteName

	return self.c.WithWaitingStatus(self.c.Tr.FetchingPullRequest, func(task gocui.Task) error {
		head, err := self.c.Helpers().PullRequests.FetchPullRequestHead(task, pr, remoteName)
		if err != nil {
			return err
		}

		self.c.OnUIThread(func() error {
			return f(head)
		})
		return nil
	})
}

func (self *PullRequestsController) checkout(pr *models.GithubPullRequest) error {
	return self.runAction(self.c.Tr.Actions.CheckoutPullRequest, self.c.Tr.CheckingOutPullRequest,
		func(repo hosting_service.ServiceInfo) error {
			if err := self.c.Git().GitHub.CheckoutPullRequest(repo, pr.Number); err != nil {
				return err
			}
			// Checking out changes the branches, commits, and files, not just
			// the pull request
			self.c.RefreshFromWorker(types.RefreshOptions{BranchSelection: types.SelectCheckedOutBranch})
			return nil
		})
}

func (self *PullRequestsController) merge(pr *models.GithubPullRequest) error {
	menuItems := func(auto bool) []*types.MenuItem {
		mergeItem := func(label string, method git_commands.PullRequestMergeMethod) *types.MenuItem {
			return &types.MenuItem{
				Label: label,
				OnPress: func() error {
					return self.runAction(self.c.Tr.Actions.MergePullRequest, self.c.Tr.MergingPullRequest,
						func(repo hosting_service.ServiceInfo) error {
							return self.c.Git().GitHub.MergePullRequest(repo, pr.Number, method, auto)
						})
				},
			}
		}

		return []*types.MenuItem{
			mergeItem(self.c.Tr.MergeWithMergeCommit, git_commands.PullRequestMergeMethodMerge),
			mergeItem(self.c.Tr.MergeWithSquash, git_commands.PullRequestMergeMethodSquash),
			mergeItem(self.c.Tr.MergeWithRebase, git_commands.PullRequestMergeMethodRebase),
		}
	}

	items := append(menuItems(false), &types.MenuItem{
		Label:   self.c.Tr.EnableAutoMerge,
		Tooltip: self.c.Tr.EnableAutoMergeTooltip,
		OnPress: func() error {
			return self.c.Menu(types.CreateMenuOptions{
				Title: self.c.Tr.EnableAutoMerge,
				Items: menuItems(true),
			})
		},
		OpensMenu: true,
	})

	return self.c.Menu(types.CreateMenuOptions{
		Title: self.c.Tr.MergePullRequestOptions,
		Items: items,
	})
}

func (self *PullRequestsController) review(pr *models.GithubPullRequest) error {
	reviewItem := func(label string, event git_commands.PullRequestReviewEvent, bodyTitle string, bodyRequired bool) *types.MenuItem {
		return &types.MenuItem{
			Label: label,
			OnPress: func() error {
				self.c.Prompt(types.PromptOpts{
					Title:                    self.resolvePlaceholders(bodyTitle, pr),
					AllowEmptyInput:          !bodyRequired,
					FindSuggestionsFunc:      self.c.Helpers().PullRequests.MentionSuggestionsFunc(),
					SuggestionsCompleteInput: true,
					HandleConfirm: func(body string) error {
						return self.runAction(self.c.Tr.Actions.ReviewPullRequest, self.c.Tr.SubmittingReview,
							func(repo hosting_service.ServiceInfo) error {
								return self.c.Git().GitHub.ReviewPullRequest(repo, pr.Number, event, body)
							})
					},
				})
				return nil
			},
		}
	}

	return self.c.Menu(types.CreateMenuOptions{
		Title: self.c.Tr.ReviewPullRequestOptions,
		Items: []*types.MenuItem{
			reviewItem(self.c.Tr.ApprovePullRequest, git_commands.PullRequestReviewApprove, self.c.Tr.OptionalReviewBodyTitle, false),
			reviewItem(self.c.Tr.RequestPullRequestChanges, git_commands.PullRequestReviewRequestChanges, self.c.Tr.ReviewBodyTitle, true),
			reviewItem(self.c.Tr.CommentReview, git_commands.PullRequestReviewComment, self.c.Tr.ReviewBodyTitle, true),
		},
	})
}

func (self *PullRequestsController) comment(pr *models.GithubPullRequest) error {
	self.c.Prompt(types.PromptOpts{
		Title:                    self.resolvePlaceholders(self.c.Tr.CommentOnPullRequestTitle, pr),
		FindSuggestionsFunc:      self.c.Helpers().PullRequests.MentionSuggestionsFunc(),
		SuggestionsCompleteInput: true,
		HandleConfirm: func(body string) error {
			return self.runAction(self.c.Tr.Actions.CommentOnPullRequest, self.c.Tr.PostingComment,
				func(repo hosting_service.ServiceInfo) error {
					return self.c.Git().GitHub.CommentOnPullRequest(repo, pr.Number, body)
				})
		},
	})
	return nil
}

func (self *PullRequestsController) edit(pr *models.GithubPullRequest) error {
	github := self.c.Git().GitHub

	return self.c.Menu(types.CreateMenuOptions{
		Title: self.c.Tr.EditPullRequestOptions,
		Items: []*types.MenuItem{
			{
				Label: self.c.Tr.RequestReview,
				OnPress: func() error {
					return self.promptForEdit(pr, self.c.Tr.RequestReview, "", github.ListAssignableUsers,
						git_commands.PullRequestEditAddReviewer)
				},
			},
			self.removeMenuItem(pr, self.c.Tr.RemoveReviewRequest, pr.ReviewRequests, self.c.Tr.NoReviewRequests,
				git_commands.PullRequestEditRemoveReviewer),
			{
				Label: self.c.Tr.AssignToMe,
				OnPress: func() error {
					return self.editPullRequest(pr, git_commands.PullRequestEditAddAssignee, "@me")
				},
			},
			{
				Label: self.c.Tr.AddAssignee,
				OnPress: func() error {
					return self.promptForEdit(pr, self.c.Tr.AddAssignee, "", github.ListAssignableUsers,
						git_commands.PullRequestEditAddAssignee)
				},
			},
			self.removeMenuItem(pr, self.c.Tr.RemoveAssignee, pr.Assignees, self.c.Tr.NoAssignees,
				git_commands.PullRequestEditRemoveAssignee),
			{
				Label: self.c.Tr.AddLabel,
				OnPress: func() error {
					return self.promptForEdit(pr, self.c.Tr.AddLabel, "", github.ListLabels,
						git_commands.PullRequestEditAddLabel)
				},
			},
			self.removeMenuItem(pr, self.c.Tr.RemoveLabel, pr.Labels, self.c.Tr.NoLabels,
				git_commands.PullRequestEditRemoveLabel),
			{
				Label: self.c.Tr.EditPullRequestTitle,
				OnPress: func() error {
					return self.promptForEdit(pr, self.c.Tr.EditPullRequestTitle, pr.Title, nil,
						git_commands.PullRequestEditTitle)
				},
			},
		},
	})
}

// promptForEdit asks for the value of an edit, suggesting the values that
// loadSuggestions returns, if given.
func (self *PullRequestsController) promptForEdit(
	pr *models.GithubPullRequest,
	title string,
	initialContent string,
	loadSuggestions func(hosting_service.ServiceInfo) ([]string, error),
	edit git_commands.PullRequestEdit,
) error {
	prompt := func(suggestions []string) {
		self.c.Prompt(types.PromptOpts{
			Title:               title,
			InitialContent:      initialContent,
			FindSuggestionsFunc: helpers.FilterFunc(suggestions, self.c.UserConfig().Gui.UseFuzzySearch()),
			HandleConfirm: func(value string) error {
				return self.editPullRequest(pr, edit, value)
			},
		})
	}

	if loadSuggestions == nil {
		prompt(nil)
		return nil
	}

	repo := *self.c.Model().PullRequestListState.Repo
	return self.c.WithWaitingStatus(self.c.Tr.LoadingSuggestions, func(gocui.Task) error {
		suggestions, err := loadSuggestions(repo)
		if err != nil {
			return err
		}

		self.c.OnUIThread(func() error {
			prompt(suggestions)
			return nil
		})
		return nil
	})
}

// removeMenuItem returns a menu item that opens a menu for removing one of the
// given values from the pull request.
func (self *PullRequestsController) removeMenuItem(
	pr *models.GithubPullRequest,
	label string,
	values []string,
	noValuesReason string,
	edit git_commands.PullRequestEdit,
) *types.MenuItem {
	var disabledReason *types.DisabledReason
	if len(values) == 0 {
		disabledReason = &types.DisabledReason{Text: noValuesReason}
	}

	return &types.MenuItem{
		Label:          label,
		DisabledReason: disabledReason,
		OpensMenu:      true,
		OnPress: func() error {
			return self.c.Menu(types.CreateMenuOptions{
				Title: label,
				Items: lo.Map(values, func(value string, _ int) *types.MenuItem {
					return &types.MenuItem{
						Label: value,
						OnPress: func() error {
							return self.editPullRequest(pr, edit, value)
						},
					}
				}),
			})
		},
	}
}

func (self *PullRequestsController) editPullRequest(pr *models.GithubPullRequest, edit git_commands.PullRequestEdit, value string) error {
	return self.runAction(self.c.Tr.Actions.EditPullRequest, self.c.Tr.UpdatingPullRequest,
		func(repo hosting_service.ServiceInfo) error {
			return self.c.Git().GitHub.EditPullRequest(repo, pr.Number, edit, value)
		})
}

func (self *PullRequestsController) toggleDraft(pr *models.GithubPullRequest) error {
	if pr.State == "DRAFT" {
		return self.runAction(self.c.Tr.Actions.MarkPullRequestReady, self.c.Tr.UpdatingPullRequest,
			func(repo hosting_service.ServiceInfo) error {
				return self.c.Git().GitHub.MarkPullRequestReady(repo, pr.Number)
			})
	}

	return self.runAction(self.c.Tr.Actions.ConvertPullRequestToDraft, self.c.Tr.UpdatingPullRequest,
		func(repo hosting_service.ServiceInfo) error {
			return self.c.Git().GitHub.ConvertPullRequestToDraft(repo, pr.Number)
		})
}

func (self *PullRequestsController) closeOrReopen(pr *models.GithubPullRequest) error {
	if pr.State == "CLOSED" {
		self.c.Confirm(types.ConfirmOpts{
			Title:  self.c.Tr.ReopenPullRequestTitle,
			Prompt: self.resolvePlaceholders(self.c.Tr.ReopenPullRequestPrompt, pr),
			HandleConfirm: func() error {
				return self.runAction(self.c.Tr.Actions.ReopenPullRequest, self.c.Tr.ReopeningPullRequest,
					func(repo hosting_service.ServiceInfo) error {
						return self.c.Git().GitHub.ReopenPullRequest(repo, pr.Number)
					})
			},
		})
		return nil
	}

	self.c.Confirm(types.ConfirmOpts{
		Title:  self.c.Tr.ClosePullRequestTitle,
		Prompt: self.resolvePlaceholders(self.c.Tr.ClosePullRequestPrompt, pr),
		HandleConfirm: func() error {
			return self.runAction(self.c.Tr.Actions.ClosePullRequest, self.c.Tr.ClosingPullRequest,
				func(repo hosting_service.ServiceInfo) error {
					return self.c.Git().GitHub.ClosePullRequest(repo, pr.Number)
				})
		},
	})
	return nil
}

func (self *PullRequestsController) openInBrowser(pr *models.GithubPullRequest) error {
	self.c.LogAction(self.c.Tr.Actions.OpenPullRequest)
	return self.c.OS().OpenLink(pr.Url)
}

func (self *PullRequestsController) copyURL(pr *models.GithubPullRequest) error {
	self.c.LogAction(self.c.Tr.Actions.CopyPullRequestURL)
	if err := self.c.OS().CopyToClipboard(pr.Url); err != nil {
		return err
	}

	self.c.Toast(self.c.Tr.PullRequestURLCopiedToClipboard)
	return nil
}

func (self *PullRequestsController) filter() error {
	helper := self.c.Helpers().PullRequests
	currentFilter := self.c.Model().PullRequestListState.Filter

	return self.c.Menu(types.CreateMenuOptions{
		Title: self.c.Tr.FilterPullRequests,
		Items: lo.Map(helpers.PullRequestFilters, func(filter git_commands.PullRequestFilter, _ int) *types.MenuItem {
			return &types.MenuItem{
				Label:  helper.FilterLabel(filter),
				Widget: types.MakeMenuRadioButton(filter == currentFilter),
				OnPress: func() error {
					helper.SetFilter(filter)
					return nil
				},
			}
		}),
	})
}

func (self *PullRequestsController) refresh() error {
	self.c.Helpers().PullRequests.Load()
	return nil
}

// runAction runs a gh command that changes a pull request, then reloads the
// pull request list so that it reflects the change.
func (self *PullRequestsController) runAction(action string, waitingStatus string, f func(repo hosting_service.ServiceInfo) error) error {
	repo := *self.c.Model().PullRequestListState.Repo

	return self.c.WithWaitingStatus(waitingStatus, func(gocui.Task) error {
		self.c.LogAction(action)
		err := f(repo)
		self.c.OnUIThread(func() error {
			self.c.Helpers().PullRequests.Load()
			return nil
		})
		return err
	})
}

func (self *PullRequestsController) isOpen(pr *models.GithubPullRequest) *types.DisabledReason {
	if pr.State != "OPEN" && pr.State != "DRAFT" {
		return &types.DisabledReason{Text: self.c.Tr.PullRequestNotOpen}
	}
	return nil
}

func (self *PullRequestsController) isNotMerged(pr *models.GithubPullRequest) *types.DisabledReason {
	if pr.State == "MERGED" {
		return &types.DisabledReason{Text: self.c.Tr.MergedPullRequestCantBeClosed}
	}
	return nil
}

func (self *PullRequestsController) resolvePlaceholders(template string, pr *models.GithubPullRequest) string {
	return utils.ResolvePlaceholderString(template, map[string]string{"number": strconv.Itoa(pr.Number)})
}

func (self *PullRequestsController) context() *context.PullRequestsContext {
	return self.c.Contexts().PullRequests
}

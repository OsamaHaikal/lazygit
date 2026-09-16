package helpers

import (
	"path/filepath"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/patch"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type CommitFilesHelper struct {
	c *HelperCommon
}

func NewCommitFilesHelper(c *HelperCommon) *CommitFilesHelper {
	return &CommitFilesHelper{
		c: c,
	}
}

type ViewCommitFilesOpts struct {
	Ref      models.Ref
	RefRange *types.RefRange
	// Whether the files can be used to build a custom patch that is then
	// applied by rebasing
	CanRebase bool
	// The context to return to when leaving the files view
	Context types.IListContext
	// If not empty, only the files at these paths are shown
	Paths []string
	// What to show the files of in the view's title, instead of the ref
	TitleRef string
	// The line to select first when going into the diff of the file at a path,
	// instead of its first change
	LinesToSelect map[string]patch.FileLine
}

// ViewCommitFiles shows the files changed by a ref (or by a range of refs) in
// the commit files view, in place of the given context.
func (self *CommitFilesHelper) ViewCommitFiles(opts ViewCommitFilesOpts) {
	commitFilesContext := self.c.Contexts().CommitFiles

	commitFilesContext.ClearFilter()
	commitFilesContext.ReInitForPaths(opts.Ref, opts.RefRange, opts.Paths)
	commitFilesContext.SetLinesToSelect(opts.LinesToSelect)
	if opts.TitleRef != "" {
		commitFilesContext.SetTitleRef(opts.TitleRef)
		commitFilesContext.GetView().Title = commitFilesContext.Title()
	}
	commitFilesContext.SetSelection(0)
	commitFilesContext.SetCanRebase(opts.CanRebase)
	commitFilesContext.SetParentContext(opts.Context)
	commitFilesContext.SetWindowName(opts.Context.GetWindowName())
	commitFilesContext.GetView().TitlePrefix = opts.Context.GetView().TitlePrefix

	self.c.Refresh(types.RefreshOptions{
		Scope: []types.RefreshableView{types.COMMIT_FILES},
		Then: func() error {
			if filterPath := self.c.Modes().Filtering.GetPath(); filterPath != "" {
				path, err := filepath.Rel(self.c.Git().RepoPaths.RepoPath(), filterPath)
				if err != nil {
					path = filterPath
				}
				commitFilesContext.CommitFileTreeViewModel.SelectPath(
					filepath.ToSlash(path), self.c.UserConfig().Gui.ShowRootItemInFileTree)
			}
			self.c.Context().Push(commitFilesContext, types.OnFocusOpts{})
			return nil
		},
	})
}

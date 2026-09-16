package pull_request

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var CommentOnPullRequestLines = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Add a review comment on a line of a pull request's diff",
	ExtraCmdArgs: []string{},
	ExtraEnvVars: ghExtraEnvVars,
	Skip:         false,
	SetupConfig:  setupGhConfig,
	SetupRepo: func(shell *Shell) {
		setupGhRepo(shell)
		setupPullRequestCommits(shell)
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().PullRequests().
			Focus().
			Lines(
				Contains("#12").IsSelected(),
				Contains("#11"),
			).
			Press(keys.PullRequests.ViewFiles)

		t.Views().CommitFiles().
			IsFocused().
			Lines(
				Equals("A feature.txt").IsSelected(),
			).
			PressEnter()

		t.Views().PatchBuilding().
			IsFocused().
			SelectedLine(Contains("+feature, improved")).
			Press(keys.PullRequests.Comment)

		t.ExpectPopup().Prompt().
			Title(Equals("Comment on feature.txt:1")).
			Type("Why improved?").
			Confirm()

		t.ExpectToast(Equals("Comment added"))

		t.FileSystem().FileContent("gh-calls.txt",
			Contains("api --hostname github.invalid --method POST repos/owner/repo/pulls/12/comments -f body=Why improved? -f commit_id=").
				Contains(" -f path=feature.txt -F line=1 -f side=RIGHT\n"))
	},
})

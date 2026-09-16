package pull_request

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ViewPullRequestCommitsAndFiles = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Drill into a pull request's commits and changed files, fetching it first",
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
			PressEnter()

		t.Views().SubCommits().
			IsFocused().
			Title(Contains("#12 Add a feature")).
			Lines(
				Contains("improve feature").IsSelected(),
				Contains("add feature"),
			).
			PressEnter()

		t.Views().CommitFiles().
			IsFocused().
			Lines(
				Equals("M feature.txt"),
			)

		t.Views().Main().Content(Contains("-feature").Contains("+feature, improved"))

		t.Views().CommitFiles().PressEscape()
		t.Views().SubCommits().PressEscape()

		t.Views().PullRequests().
			IsFocused().
			Press(keys.PullRequests.ViewFiles)

		t.Views().CommitFiles().
			IsFocused().
			Title(Contains("#12 Add a feature")).
			Lines(
				Equals("A feature.txt"),
			)

		t.Views().Main().Content(Contains("+feature, improved"))

		t.Views().CommitFiles().PressEscape()
		t.Views().PullRequests().IsFocused()
	},
})

package pull_request

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FilterPullRequests = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Switch the pull requests panel between filters",
	ExtraCmdArgs: []string{},
	ExtraEnvVars: ghExtraEnvVars,
	Skip:         false,
	SetupConfig:  setupGhConfig,
	SetupRepo:    setupGhRepo,
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().PullRequests().
			Focus().
			Lines(
				Contains("#12").IsSelected(),
				Contains("#11"),
			).
			Press(keys.PullRequests.Filter).
			Tap(func() {
				t.ExpectPopup().Menu().
					Title(Equals("Filter pull requests")).
					Lines(
						Contains("Open").IsSelected(),
						Contains("Review requested from me"),
						Contains("Created by me"),
						Contains("Assigned to me"),
						Contains("Merged"),
						Contains("Closed"),
						Contains("All"),
						Contains("Cancel"),
					).
					Select(Contains("Review requested from me")).
					Confirm()
			}).
			Lines(
				Contains("#12"),
				Contains("#11"),
			)

		t.FileSystem().FileContent("gh-searches.txt", Equals(
			"q=repo:owner/repo is:pr is:open sort:updated-desc\n"+
				"q=repo:owner/repo is:pr is:open review-requested:@me sort:updated-desc\n",
		))
	},
})

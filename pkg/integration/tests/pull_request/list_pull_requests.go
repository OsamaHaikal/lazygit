package pull_request

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ListPullRequests = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "List a repo's pull requests and show the selected one in the main view",
	ExtraCmdArgs: []string{},
	ExtraEnvVars: ghExtraEnvVars,
	Skip:         false,
	SetupConfig:  setupGhConfig,
	SetupRepo:    setupGhRepo,
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().PullRequests().
			Focus().
			Lines(
				Contains("#12").Contains("alice").Contains("Add a feature").Contains("Approved").Contains("+10 -2").IsSelected(),
				Contains("#11").Contains("bob").Contains("Fix a bug").Contains("+1 -1"),
			)

		t.Views().Main().Content(Contains("Viewing pull request 12"))

		t.Views().PullRequests().
			SelectNextItem()

		t.Views().Main().Content(Contains("Viewing pull request 11"))
	},
})

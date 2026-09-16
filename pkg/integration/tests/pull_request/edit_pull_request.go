package pull_request

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var EditPullRequest = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Request reviews, assign, and label a pull request",
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
			Press(keys.PullRequests.Edit).
			Tap(func() {
				t.ExpectPopup().Menu().
					Title(Equals("Edit pull request")).
					Select(Equals("Request review")).
					Confirm()

				t.ExpectPopup().Prompt().
					Title(Equals("Request review")).
					SuggestionLines(
						Contains("alice"),
						Contains("carol"),
					).
					Type("car").
					SuggestionLines(
						Contains("carol"),
					).
					ConfirmFirstSuggestion()
			}).
			Press(keys.PullRequests.Edit).
			Tap(func() {
				t.ExpectPopup().Menu().
					Title(Equals("Edit pull request")).
					Select(Equals("Assign to me")).
					Confirm()
			}).
			Press(keys.PullRequests.Edit).
			Tap(func() {
				t.ExpectPopup().Menu().
					Title(Equals("Edit pull request")).
					Select(Equals("Remove label...")).
					Confirm()

				t.ExpectPopup().Menu().
					Title(Equals("Remove label")).
					Select(Equals("enhancement")).
					Confirm()
			})

		t.FileSystem().FileContent("gh-calls.txt", Equals(
			"pr edit 12 --repo github.invalid/owner/repo --add-reviewer carol\n"+
				"pr edit 12 --repo github.invalid/owner/repo --add-assignee @me\n"+
				"pr edit 12 --repo github.invalid/owner/repo --remove-label enhancement\n",
		))
	},
})

package pull_request

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PullRequestActions = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Comment on, review, merge, and close pull requests through gh",
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
			Press(keys.PullRequests.Comment).
			Tap(func() {
				t.ExpectPopup().Prompt().
					Title(Equals("Comment on #12")).
					Type("Nice work").
					Confirm()
			}).
			Press(keys.PullRequests.Review).
			Tap(func() {
				t.ExpectPopup().Menu().
					Title(Equals("Review pull request")).
					Select(Equals("Request changes")).
					Confirm()

				t.ExpectPopup().Prompt().
					Title(Equals("Review of #12")).
					Type("Please add tests").
					Confirm()
			}).
			Press(keys.PullRequests.Merge).
			Tap(func() {
				t.ExpectPopup().Menu().
					Title(Equals("Merge pull request")).
					Select(Equals("Squash and merge")).
					Confirm()
			}).
			SelectNextItem().
			Press(keys.PullRequests.ToggleDraft).
			Press(keys.PullRequests.CloseOrReopen).
			Tap(func() {
				t.ExpectPopup().Confirmation().
					Title(Equals("Close pull request")).
					Content(Equals("Are you sure you want to close pull request #11?")).
					Confirm()
			})

		t.FileSystem().FileContent("gh-calls.txt", Equals(
			"pr comment 12 --repo github.invalid/owner/repo --body Nice work\n"+
				"pr review 12 --repo github.invalid/owner/repo --request-changes --body Please add tests\n"+
				"pr merge 12 --repo github.invalid/owner/repo --squash\n"+
				"pr ready 11 --repo github.invalid/owner/repo\n"+
				"pr close 11 --repo github.invalid/owner/repo\n",
		))
	},
})

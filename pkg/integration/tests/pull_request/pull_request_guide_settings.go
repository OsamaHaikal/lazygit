package pull_request

import (
	"os"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PullRequestGuideSettings = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Choose the AI and model that write pull request guides, overriding the config",
	ExtraCmdArgs: []string{},
	ExtraEnvVars: map[string]string{
		"GH_PATH": ghExtraEnvVars["GH_PATH"],
		"PATH":    "{{actualPath}}/bin:" + os.Getenv("PATH"),
	},
	Skip: false,
	SetupConfig: func(cfg *config.AppConfig) {
		setupGhConfig(cfg)
		cfg.GetUserConfig().Git.PullRequestGuide.Provider = "codex"
		cfg.GetUserConfig().Git.PullRequestGuide.Model = "some-codex-model"
	},
	SetupRepo: func(shell *Shell) {
		setupGhRepo(shell)
		setupPullRequestCommits(shell)

		shell.CreateFile("../bin/claude", `#!/bin/sh
cat > /dev/null
echo "$@" | grep -o -- "--model [a-z]*" >> "$(dirname "$0")/claude-calls.txt"
echo '{"type": "result", "is_error": false, "result": "", "structured_output": {"chapters": [{"title": "Add the feature", "explanation": "Adds it.", "hunks": ["f0-h0"]}]}}'
`)
		shell.MakeExecutable("../bin/claude")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().PullRequests().
			Focus().
			Press(keys.PullRequests.GuideSettings)

		t.ExpectPopup().Menu().
			Title(Equals("AI guide settings")).
			Lines(
				Equals("( ) Auto (Codex if installed, else Claude Code)"),
				Equals("(•) Codex"),
				Equals("( ) Claude Code"),
				Contains("Model: some-codex-model"),
				Contains("Cancel"),
			).
			Select(Equals("( ) Claude Code")).
			Confirm()

		t.ExpectToast(Equals("New guides will be written by claude (some-codex-model)"))

		t.Views().PullRequests().
			Press(keys.PullRequests.GuideSettings)

		t.ExpectPopup().Menu().
			Title(Equals("AI guide settings")).
			Select(Contains("Model:")).
			Confirm()

		t.ExpectPopup().Prompt().
			Title(Equals("Model (leave empty for the AI's default)")).
			InitialText(Equals("some-codex-model")).
			Clear().
			Type("hai").
			SuggestionLines(Equals("haiku")).
			ConfirmFirstSuggestion()

		t.ExpectToast(Equals("New guides will be written by claude (haiku)"))

		t.Views().PullRequests().
			Press(keys.PullRequests.Guide)

		t.Views().Main().ContentEventually(Contains("Written by claude (haiku)"))

		t.FileSystem().FileContent("../bin/claude-calls.txt", Equals("--model haiku\n"))
	},
})

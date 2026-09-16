package pull_request

import (
	"os"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var CancelPullRequestGuide = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Stop the AI while it's writing a pull request guide, seeing what it's doing until then",
	ExtraCmdArgs: []string{},
	ExtraEnvVars: map[string]string{
		"GH_PATH": ghExtraEnvVars["GH_PATH"],
		"PATH":    "{{actualPath}}/bin:" + os.Getenv("PATH"),
	},
	Skip: false,
	SetupConfig: func(cfg *config.AppConfig) {
		setupGhConfig(cfg)
		cfg.GetUserConfig().Git.PullRequestGuide.Provider = "claude"
	},
	SetupRepo: func(shell *Shell) {
		setupGhRepo(shell)
		setupPullRequestCommits(shell)

		// A fake Claude Code that says what it's doing, then never finishes
		shell.CreateFile("../bin/claude", `#!/bin/sh
cat > /dev/null
echo '{"type": "assistant", "message": {"content": [{"type": "text", "text": "Progress: Reading the feature"}]}}'
exec sleep 60
`)
		shell.MakeExecutable("../bin/claude")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().PullRequests().
			Focus().
			Lines(
				Contains("#12").IsSelected(),
				Contains("#11"),
			).
			Press(keys.PullRequests.Guide)

		t.Views().Main().ContentEventually(
			Contains("claude is writing a guide to #12").
				Contains("› Reading the feature"),
		)

		t.Views().PullRequestGuide().
			IsFocused().
			Press(keys.Universal.Remove)

		t.Views().Main().ContentEventually(
			Contains("Couldn't write a guide to #12: you stopped it").
				Contains("Press R to try again."),
		)
	},
})

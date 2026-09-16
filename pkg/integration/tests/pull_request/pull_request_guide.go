package pull_request

import (
	"os"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PullRequestGuide = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Show an AI-written guide to a pull request, and reuse it when asked for it again",
	ExtraCmdArgs: []string{},
	ExtraEnvVars: map[string]string{
		"GH_PATH": ghExtraEnvVars["GH_PATH"],
		"PATH":    "{{actualPath}}/bin:" + os.Getenv("PATH"),
	},
	Skip: false,
	SetupConfig: func(cfg *config.AppConfig) {
		setupGhConfig(cfg)
		cfg.GetUserConfig().Git.PullRequestGuide.Provider = "claude"
		cfg.GetUserConfig().Git.PullRequestGuide.Model = "sonnet"
	},
	SetupRepo: func(shell *Shell) {
		setupGhRepo(shell)
		setupPullRequestCommits(shell)

		// A fake Claude Code that records how often it was asked, and answers
		// with a guide covering the pull request's one hunk
		shell.CreateFile("../bin/claude", `#!/bin/sh
cat > /dev/null
echo "$@" | grep -o -- "--model [a-z]*" >> claude-calls.txt
cat <<'GUIDE'
{"type": "result", "is_error": false, "result": "", "structured_output": {"chapters": [{"title": "Add the feature", "explanation": "Creates `+"`feature.txt`"+` with the improved feature.", "hunks": ["f0-h0"]}]}}
GUIDE
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

		t.Views().PullRequestGuide().
			IsFocused().
			Title(Contains("#12 Add a feature")).
			Lines(
				Equals("1. Add the feature").IsSelected(),
			)

		t.Views().Main().Content(
			Contains("Add the feature").
				Contains("Written by claude (sonnet)").
				Contains("Creates feature.txt with the improved feature.").
				Contains("feature.txt").
				Contains("+feature, improved"),
		)

		t.Views().PullRequestGuide().PressEscape()

		t.Views().PullRequests().
			IsFocused().
			Press(keys.PullRequests.Guide)

		t.Views().PullRequestGuide().
			IsFocused().
			Lines(
				Equals("1. Add the feature").IsSelected(),
			)

		t.FileSystem().FileContent("claude-calls.txt", Equals("--model sonnet\n"))
	},
})

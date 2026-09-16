package pull_request

import (
	"os"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PullRequestGuide = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Show an AI-written guide to a pull request, go into a chapter's files, and reuse the guide when asked for it again",
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

		// Give the pull request a second file, so that its guide has two
		// chapters
		shell.CreateFile("../contributor/docs.md", "How to use the feature\n")
		shell.RunCommand([]string{"git", "-C", "../contributor", "add", "docs.md"})
		shell.RunCommand([]string{"git", "-C", "../contributor", "commit", "-m", "document feature"})
		shell.RunCommand([]string{"git", "-C", "../contributor", "push", "--force", "../origin", "HEAD:refs/pull/12/head"})

		// A fake Claude Code that records how often it was asked, and answers
		// with a chapter for each of the pull request's hunks
		shell.CreateFile("../bin/claude", `#!/bin/sh
cat > /dev/null
echo "$@" | grep -o -- "--model [a-z]*" >> "$(dirname "$0")/claude-calls.txt"
cat <<'GUIDE'
{"type": "assistant", "message": {"content": [{"type": "text", "text": "Progress: Grouping the hunks"}]}}
{"type": "result", "is_error": false, "result": "", "structured_output": {"chapters": [{"title": "Add the feature", "explanation": "Creates `+"`feature.txt`"+` with the improved feature.", "hunks": ["f1-h0"]}, {"title": "Document the feature", "explanation": "Explains how to use it.", "hunks": ["f0-h0"]}]}}
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
				Equals("2. Document the feature"),
			)

		t.Views().Main().Content(
			Contains("Add the feature").
				Contains("Written by claude (sonnet)").
				Contains("Creates feature.txt with the improved feature.").
				Contains("feature.txt").
				Contains("+feature, improved").
				DoesNotContain("How to use the feature"),
		)

		t.Views().PullRequestGuide().PressEnter()

		t.Views().CommitFiles().
			IsFocused().
			Title(Contains("Add the feature")).
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
			Cancel()

		t.Views().PatchBuilding().PressEscape()
		t.Views().CommitFiles().IsFocused().PressEscape()

		t.Views().PullRequestGuide().
			IsFocused().
			NavigateToLine(Equals("2. Document the feature")).
			PressEnter()

		t.Views().CommitFiles().
			IsFocused().
			Title(Contains("Document the feature")).
			Lines(
				Equals("A docs.md").IsSelected(),
			).
			PressEscape()

		t.Views().PullRequestGuide().
			IsFocused().
			PressEscape()

		t.Views().PullRequests().
			IsFocused().
			Press(keys.PullRequests.Guide)

		t.Views().PullRequestGuide().
			IsFocused().
			Lines(
				Equals("1. Add the feature").IsSelected(),
				Equals("2. Document the feature"),
			)

		t.FileSystem().FileContent("../bin/claude-calls.txt", Equals("--model sonnet\n"))
	},
})

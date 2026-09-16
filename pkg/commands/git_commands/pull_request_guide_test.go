package git_commands

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stretchr/testify/assert"
)

func TestSplitDiffIntoReviewUnits(t *testing.T) {
	diff := `diff --git a/main.go b/main.go
index 1111111..2222222 100644
--- a/main.go
+++ b/main.go
@@ -1,3 +1,3 @@
 package main
-var a = 1
+var a = 2
@@ -10,2 +10,3 @@ func main() {
 	run()
+	stop()
diff --git a/old name.txt b/new name.txt
similarity index 100%
rename from old name.txt
rename to new name.txt
diff --git a/image.png b/image.png
index 3333333..4444444 100644
Binary files a/image.png and b/image.png differ
`

	assert.Equal(t, []GuideFile{
		{
			Path: "main.go",
			Units: []GuideReviewUnit{
				{ID: "f0-h0", Path: "main.go", Diff: "@@ -1,3 +1,3 @@\n package main\n-var a = 1\n+var a = 2"},
				{ID: "f0-h1", Path: "main.go", Diff: "@@ -10,2 +10,3 @@ func main() {\n \trun()\n+\tstop()"},
			},
		},
		{
			Path:    "new name.txt",
			OldPath: "old name.txt",
			Units: []GuideReviewUnit{
				{
					ID: "f1-meta", Path: "new name.txt", OldPath: "old name.txt",
					Diff: "diff --git a/old name.txt b/new name.txt\nsimilarity index 100%\nrename from old name.txt\nrename to new name.txt",
				},
			},
		},
		{
			Path: "image.png",
			Units: []GuideReviewUnit{
				{
					ID: "f2-meta", Path: "image.png",
					Diff: "diff --git a/image.png b/image.png\nindex 3333333..4444444 100644\nBinary files a/image.png and b/image.png differ",
				},
			},
		},
	}, splitDiffIntoReviewUnits(diff))
}

func TestPathsFromDiffGitLine(t *testing.T) {
	oldPath, newPath := pathsFromDiffGitLine("diff --git a/dir b/file b/dir b/file")
	assert.Equal(t, "dir b/file", oldPath)
	assert.Equal(t, "dir b/file", newPath)
}

func TestValidateGuide(t *testing.T) {
	files := []GuideFile{
		{Path: "a.go", Units: []GuideReviewUnit{{ID: "f0-h0"}, {ID: "f0-h1"}}},
		{Path: "b.png", Units: []GuideReviewUnit{{ID: "f1-meta"}}},
	}

	cases := []struct {
		name             string
		chapters         []GuideChapter
		expectedError    string
		expectedChapters []GuideChapter
	}{
		{
			name:          "no chapters",
			expectedError: "it has no chapters",
		},
		{
			name: "invented unit",
			chapters: []GuideChapter{
				{Title: "One", UnitIDs: []string{"f0-h0", "f0-h1", "f1-meta", "f9-h9"}},
			},
			expectedError: "chapter 'One' refers to 'f9-h9', which isn't part of the diff",
		},
		{
			name: "missing units",
			chapters: []GuideChapter{
				{Title: "One", UnitIDs: []string{"f0-h1"}},
			},
			expectedError: "it leaves out f0-h0, f1-meta",
		},
		{
			name: "units repeated within a chapter and used by several chapters",
			chapters: []GuideChapter{
				{Title: "One", UnitIDs: []string{"f0-h1", "f0-h0", "f0-h1"}},
				{Title: "Two", UnitIDs: []string{"f0-h0", "f1-meta"}},
			},
			expectedChapters: []GuideChapter{
				{Title: "One", UnitIDs: []string{"f0-h1", "f0-h0"}},
				{Title: "Two", UnitIDs: []string{"f0-h0", "f1-meta"}},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			guide := &PullRequestGuide{Chapters: c.chapters, Files: files}
			err := validateGuide(guide)
			if c.expectedError != "" {
				assert.EqualError(t, err, c.expectedError)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, c.expectedChapters, guide.Chapters)
		})
	}
}

func TestGeneratePullRequestGuideWithClaude(t *testing.T) {
	diff := "diff --git a/a.go b/a.go\n--- a/a.go\n+++ b/a.go\n@@ -1 +1 @@\n-x\n+y\n"
	claudeOutput := `{"type": "system", "subtype": "init"}
{"type": "assistant", "message": {"content": [{"type": "text", "text": "Progress: Reading the renamed variable's uses"}]}}
{"type": "result", "is_error": false, "result": "", "structured_output": {"chapters": [{"title": "Rename x", "explanation": "x becomes y", "hunks": ["f0-h0"]}]}}
`

	var progress []string
	runner := oscommands.NewFakeRunner(t).
		ExpectGitArgs([]string{"-c", "core.quotePath=false", "-c", "diff.noprefix=false", "diff", "--no-ext-diff", "--no-color", "--find-renames", "base123", "head456"}, diff, nil).
		ExpectFunc("claude with the guide prompt", func(cmdObj *oscommands.CmdObj) bool {
			args := cmdObj.Args()
			return args[0] == "claude" && args[1] == "-p" &&
				slices.Contains(args, "stream-json") &&
				slices.Contains(args, "--json-schema") &&
				slices.Equal(args[len(args)-2:], []string{"--model", "sonnet"}) &&
				cmdObj.GetCmd().Stdin != nil &&
				// It runs outside of the repo
				cmdObj.GetCmd().Dir != ""
		}, claudeOutput, nil)
	instance := NewGitHubCommands(buildGitCommon(commonDeps{runner: runner}))

	guide, err := instance.GeneratePullRequestGuide(GeneratePullRequestGuideOpts{
		Provider: GuideProviderClaude, Model: "sonnet",
		Number: 12, Title: "Rename", Description: "Renames x",
		MergeBase: "base123", Head: "head456",
		OnProgress: func(message string) { progress = append(progress, message) },
	})
	assert.NoError(t, err)
	runner.CheckForMissingCalls()

	assert.Equal(t, []string{"Reading the renamed variable's uses"}, progress)
	assert.Equal(t, &PullRequestGuide{
		Chapters: []GuideChapter{{Title: "Rename x", Explanation: "x becomes y", UnitIDs: []string{"f0-h0"}}},
		Files: []GuideFile{{Path: "a.go", Units: []GuideReviewUnit{
			{ID: "f0-h0", Path: "a.go", Diff: "@@ -1 +1 @@\n-x\n+y"},
		}}},
		Provider: GuideProviderClaude,
		Model:    "sonnet",
	}, guide)
}

func TestParseCodexGuideEvent(t *testing.T) {
	cases := []struct {
		line             string
		expectedProgress string
		expectedGuide    string
	}{
		{`{"type":"turn.started"}`, "", ""},
		{`{"type":"item.completed","item":{"type":"agent_message","text":"Progress: Grouping the hunks into chapters."}}`, "Grouping the hunks into chapters.", ""},
		{`{"type":"item.started","item":{"type":"command_execution","command":"/bin/zsh -lc 'git -C /repo show abc:main.go'"}}`, "Running git show abc:main.go", ""},
		{`{"type":"item.started","item":{"type":"command_execution","command":"/bin/zsh -lc \"git -C /repo show abc:a.yml | nl -ba\n  | sed -n '1,9p'\""}}`, "Running git show abc:a.yml | nl -ba | sed -n '1,9p'", ""},
		{`{"type":"item.completed","item":{"type":"reasoning","text":"**Reviewing workflow changes**\n\nI need to look at..."}}`, "Reviewing workflow changes", ""},
		{`{"type":"item.completed","item":{"type":"reasoning","text":"I need to look at the workflows"}}`, "", ""},
		{`{"type":"item.completed","item":{"type":"command_execution","command":"/bin/zsh -lc 'git log'"}}`, "", ""},
		{`{"type":"item.completed","item":{"type":"agent_message","text":"{\"chapters\":[]}"}}`, "", `{"chapters":[]}`},
		{`not json`, "", ""},
	}

	for _, c := range cases {
		progress, guide := parseCodexGuideEvent(c.line, "/repo")
		assert.Equal(t, c.expectedProgress, progress, c.line)
		assert.Equal(t, c.expectedGuide, guide, c.line)
	}
}

func TestParseClaudeGuideEvent(t *testing.T) {
	cases := []struct {
		line             string
		expectedProgress string
		expectedGuide    string
		expectedError    string
	}{
		{`{"type":"system","subtype":"init"}`, "", "", ""},
		{`{"type":"assistant","message":{"content":[{"type":"text","text":"Progress: Checking the callers"}]}}`, "Checking the callers", "", ""},
		{`{"type":"assistant","message":{"content":[{"type":"text","text":"Let me look"}]}}`, "", "", ""},
		{`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"git -C /repo show abc:main.go"}}]}}`, "Running git show abc:main.go", "", ""},
		{`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"/repo/pkg/main.go"}}]}}`, "Reading pkg/main.go", "", ""},
		{`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"git -C /repo log --oneline ` + strings.Repeat("x", 200) + `"}}]}}`, "Running git log --oneline " + strings.Repeat("x", 81) + "…", "", ""},
		{`{"type":"result","is_error":false,"structured_output":{"chapters":[]}}`, "", `{"chapters":[]}`, ""},
		{`{"type":"result","is_error":true,"result":"Credit balance is too low"}`, "", "", "claude couldn't write the guide: Credit balance is too low"},
	}

	for _, c := range cases {
		progress, guide, err := parseClaudeGuideEvent(c.line, "/repo")
		assert.Equal(t, c.expectedProgress, progress, c.line)
		assert.Equal(t, c.expectedGuide, guide, c.line)
		if c.expectedError == "" {
			assert.NoError(t, err, c.line)
		} else {
			assert.EqualError(t, err, c.expectedError, c.line)
		}
	}
}

func TestResolveGuideProvider(t *testing.T) {
	binDir := t.TempDir()
	assert.NoError(t, os.WriteFile(filepath.Join(binDir, "claude"), []byte("#!/bin/sh\n"), 0o755))
	t.Setenv("PATH", binDir)

	instance := NewGitHubCommands(buildGitCommon(commonDeps{}))

	provider, err := instance.ResolveGuideProvider(GuideProviderAuto)
	assert.NoError(t, err)
	assert.Equal(t, GuideProviderClaude, provider)

	provider, err = instance.ResolveGuideProvider(GuideProviderCodex)
	assert.NoError(t, err)
	assert.Equal(t, GuideProviderCodex, provider)

	t.Setenv("PATH", t.TempDir())
	_, err = instance.ResolveGuideProvider(GuideProviderAuto)
	assert.ErrorIs(t, err, ErrNoGuideProvider)
}

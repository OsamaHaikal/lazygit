package git_commands

import (
	"os"
	"path/filepath"
	"slices"
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
	claudeOutput := `{"type": "result", "is_error": false, "result": "", "structured_output": {"chapters": [{"title": "Rename x", "explanation": "x becomes y", "hunks": ["f0-h0"]}]}}`

	runner := oscommands.NewFakeRunner(t).
		ExpectGitArgs([]string{"-c", "core.quotePath=false", "-c", "diff.noprefix=false", "diff", "--no-ext-diff", "--no-color", "--find-renames", "base123", "head456"}, diff, nil).
		ExpectFunc("claude with the guide prompt", func(cmdObj *oscommands.CmdObj) bool {
			args := cmdObj.Args()
			stdin := cmdObj.GetCmd().Stdin
			return args[0] == "claude" && args[1] == "-p" &&
				slices.Contains(args, "--json-schema") &&
				slices.Equal(args[len(args)-2:], []string{"--model", "sonnet"}) &&
				stdin != nil
		}, claudeOutput, nil)
	instance := NewGitHubCommands(buildGitCommon(commonDeps{runner: runner}))

	guide, err := instance.GeneratePullRequestGuide(GeneratePullRequestGuideOpts{
		Provider: GuideProviderClaude, Model: "sonnet",
		Number: 12, Title: "Rename", Description: "Renames x",
		MergeBase: "base123", Head: "head456",
	})
	assert.NoError(t, err)
	runner.CheckForMissingCalls()

	assert.Equal(t, &PullRequestGuide{
		Chapters: []GuideChapter{{Title: "Rename x", Explanation: "x becomes y", UnitIDs: []string{"f0-h0"}}},
		Files: []GuideFile{{Path: "a.go", Units: []GuideReviewUnit{
			{ID: "f0-h0", Path: "a.go", Diff: "@@ -1 +1 @@\n-x\n+y"},
		}}},
		Provider: GuideProviderClaude,
		Model:    "sonnet",
	}, guide)
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

package git_commands

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/stretchr/testify/assert"
)

func TestGetCommitFilesFromFilenames(t *testing.T) {
	tests := []struct {
		testName string
		input    string
		output   []*models.CommitFile
	}{
		{
			testName: "no files",
			input:    "",
			output:   []*models.CommitFile{},
		},
		{
			testName: "one file",
			input:    "MM\x00Myfile\x00",
			output: []*models.CommitFile{
				{
					Path:         "Myfile",
					ChangeStatus: "MM",
				},
			},
		},
		{
			testName: "two files",
			input:    "MM\x00Myfile\x00M \x00MyOtherFile\x00",
			output: []*models.CommitFile{
				{
					Path:         "Myfile",
					ChangeStatus: "MM",
				},
				{
					Path:         "MyOtherFile",
					ChangeStatus: "M ",
				},
			},
		},
		{
			testName: "three files",
			input:    "MM\x00Myfile\x00M \x00MyOtherFile\x00 M\x00YetAnother\x00",
			output: []*models.CommitFile{
				{
					Path:         "Myfile",
					ChangeStatus: "MM",
				},
				{
					Path:         "MyOtherFile",
					ChangeStatus: "M ",
				},
				{
					Path:         "YetAnother",
					ChangeStatus: " M",
				},
			},
		},
		{
			testName: "a rename among regular files",
			input:    "M\x00Myfile\x00R100\x00before\x00after\x00A\x00Added\x00",
			output: []*models.CommitFile{
				{
					Path:         "Myfile",
					ChangeStatus: "M",
				},
				{
					Path:         "after",
					PreviousPath: "before",
					ChangeStatus: "R",
				},
				{
					Path:         "Added",
					ChangeStatus: "A",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			result := getCommitFilesFromFilenames(test.input)
			assert.Equal(t, test.output, result)
		})
	}
}

func TestGetFilesInDiff(t *testing.T) {
	cases := []struct {
		name         string
		paths        []string
		expectedArgs []string
	}{
		{
			name:         "all files",
			paths:        nil,
			expectedArgs: []string{"-c", "diff.noprefix=false", "diff", "--submodule", "--no-ext-diff", "--name-status", "-z", "--find-renames=50%", "abc", "def"},
		},
		{
			name:         "some paths",
			paths:        []string{"a.go", "dir/b.go"},
			expectedArgs: []string{"-c", "diff.noprefix=false", "diff", "--submodule", "--no-ext-diff", "--name-status", "-z", "--find-renames=50%", "abc", "def", "--", "a.go", "dir/b.go"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			runner := oscommands.NewFakeRunner(t).ExpectGitArgs(c.expectedArgs, "M\x00a.go\x00", nil)
			loader := NewCommitFileLoader(common.NewDummyCommon(), oscommands.NewDummyCmdObjBuilder(runner))

			files, err := loader.GetFilesInDiff("abc", "def", false, c.paths)
			assert.NoError(t, err)
			assert.Len(t, files, 1)
			runner.CheckForMissingCalls()
		})
	}
}

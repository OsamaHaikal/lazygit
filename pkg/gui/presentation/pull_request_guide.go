package presentation

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/samber/lo"
)

var inlineCodeRegex = regexp.MustCompile("`([^`\n]+)`")

// FormatPullRequestGuideChapter renders a chapter of a pull request guide: its
// explanation, followed by the hunks it's about, grouped by file.
func FormatPullRequestGuideChapter(guide *git_commands.PullRequestGuide, chapter *git_commands.GuideChapter, writtenBy string) string {
	var builder strings.Builder

	builder.WriteString(style.FgYellow.SetBold().Sprint(chapter.Title) + "\n")
	builder.WriteString(style.FgDefault.Sprint(writtenBy) + "\n\n")
	builder.WriteString(inlineCodeRegex.ReplaceAllStringFunc(strings.TrimSpace(chapter.Explanation), func(code string) string {
		return style.FgCyan.Sprint(strings.Trim(code, "`"))
	}))
	builder.WriteString("\n")

	previousPath := ""
	for _, unit := range guide.Units(*chapter) {
		if unit.Path != previousPath {
			path := unit.Path
			if unit.OldPath != "" {
				path = unit.OldPath + " → " + unit.Path
			}
			builder.WriteString("\n" + style.FgCyan.SetBold().Sprint(path) + "\n")
			previousPath = unit.Path
		}

		builder.WriteString(strings.Join(lo.Map(strings.Split(unit.Diff, "\n"), func(line string, _ int) string {
			return guideDiffLineStyle(line).Sprint(line)
		}), "\n") + "\n")
	}

	return builder.String()
}

func guideDiffLineStyle(line string) style.TextStyle {
	switch {
	case strings.HasPrefix(line, "@@"):
		return style.FgMagenta
	case strings.HasPrefix(line, "diff --git"), strings.HasPrefix(line, "index "):
		return style.FgDefault.SetBold()
	default:
		return diffLineStyle(line)
	}
}

// FormatElapsed shows a duration like "1:05"
func FormatElapsed(elapsed time.Duration) string {
	seconds := int(elapsed.Seconds())
	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}

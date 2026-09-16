package presentation

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

var inlineCodeRegex = regexp.MustCompile("`([^`\n]+)`")

// FormatPullRequestGuideChapter renders a chapter of a pull request guide: its
// explanation, followed by the hunks it's about, grouped by file, each saying
// which other chapters explain it too.
func FormatPullRequestGuideChapter(
	guide *git_commands.PullRequestGuide,
	chapter *git_commands.GuideChapter,
	notice string,
	writtenBy string,
	tr *i18n.TranslationSet,
) string {
	var builder strings.Builder

	if notice != "" {
		builder.WriteString(style.FgYellow.Sprint("⚠ "+notice) + "\n\n")
	}

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

		otherChapters := lo.FilterMap(guide.Chapters, func(other git_commands.GuideChapter, i int) (string, bool) {
			return fmt.Sprintf("%d. %s", i+1, other.Title),
				other.Title != chapter.Title && lo.Contains(other.UnitIDs, unit.ID)
		})
		if len(otherChapters) > 0 {
			builder.WriteString(style.FgBlue.Sprint("↳ "+utils.ResolvePlaceholderString(tr.GuideHunkAlsoExplainedIn,
				map[string]string{"chapters": strings.Join(otherChapters, ", ")})) + "\n")
		}
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

// FormatGuideProgress lists what the AI said it's doing while writing a guide,
// with the latest thing highlighted.
func FormatGuideProgress(progress []string) string {
	if len(progress) == 0 {
		return ""
	}

	lines := lo.Map(progress, func(message string, i int) string {
		if i == len(progress)-1 {
			return style.FgCyan.Sprint("› " + message)
		}
		return style.FgDefault.Sprint("  " + message)
	})
	return "\n\n" + strings.Join(lines, "\n")
}

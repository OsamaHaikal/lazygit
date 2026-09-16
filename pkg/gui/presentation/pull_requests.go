package presentation

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation/authors"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/jesseduffield/lazygit/pkg/theme"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

func GetPullRequestListDisplayStrings(prs []*models.GithubPullRequest, tr *i18n.TranslationSet) [][]string {
	return lo.Map(prs, func(pr *models.GithubPullRequest, _ int) []string {
		return getPullRequestListDisplayString(pr, tr)
	})
}

func getPullRequestListDisplayString(pr *models.GithubPullRequest, tr *i18n.TranslationSet) []string {
	checksIcon, _, checksStyle := checksStatePresentation(pr.ChecksState, tr)

	return []string{
		// The number's color tells whether the pull request is open, a draft,
		// merged, or closed, the same way the branches panel colors its pull
		// request icons.
		WithPrColor(pr.State, fmt.Sprintf("#%d", pr.Number), false),
		checksStyle.Sprint(checksIcon),
		style.FgBlue.Sprint(utils.UnixToTimeAgo(pr.UpdatedAt.Unix())),
		authors.AuthorStyle(pr.Author).Sprint(pr.Author),
		theme.DefaultTextColor.Sprint(pr.Title) + reviewDecisionText(pr.ReviewDecision, tr),
		fmt.Sprintf("%s %s", style.FgGreen.Sprintf("+%d", pr.Additions), style.FgRed.Sprintf("-%d", pr.Deletions)),
	}
}

func reviewDecisionText(reviewDecision string, tr *i18n.TranslationSet) string {
	switch reviewDecision {
	case "APPROVED":
		return " " + style.FgGreen.Sprint(tr.PullRequestApproved)
	case "CHANGES_REQUESTED":
		return " " + style.FgRed.Sprint(tr.PullRequestChangesRequested)
	default:
		return ""
	}
}

// The number of lines of a comment's diff hunk shown above it, counting up
// from the last commented line, like GitHub does
const reviewCommentContextLines = 4

// FormatPullRequestReviewComments renders comments on a pull request's code as
// threads, each one headed by the commented file and lines.
func FormatPullRequestReviewComments(comments []*models.PullRequestReviewComment, tr *i18n.TranslationSet) string {
	repliesByID := lo.GroupBy(
		lo.Filter(comments, func(comment *models.PullRequestReviewComment, _ int) bool { return comment.InReplyToID != 0 }),
		func(comment *models.PullRequestReviewComment) int { return comment.InReplyToID },
	)
	threadStarts := lo.Filter(comments, func(comment *models.PullRequestReviewComment, _ int) bool {
		return comment.InReplyToID == 0
	})

	threads := lo.Map(threadStarts, func(start *models.PullRequestReviewComment, _ int) string {
		var builder strings.Builder

		location := start.Path
		if start.Line != 0 {
			location += ":" + strconv.Itoa(start.Line)
		} else {
			location += " " + style.FgDefault.Sprintf("(%s)", tr.Outdated)
		}
		builder.WriteString(style.FgCyan.SetBold().Sprint(location) + "\n")

		for _, line := range lastLines(start.DiffHunk, reviewCommentContextLines) {
			builder.WriteString("  " + diffLineStyle(line).Sprint(line) + "\n")
		}

		for _, comment := range append([]*models.PullRequestReviewComment{start}, repliesByID[start.ID]...) {
			builder.WriteString("\n  " + authors.AuthorStyle(comment.Author).Sprint(comment.Author) +
				" " + style.FgBlue.Sprint(utils.UnixToTimeAgo(comment.CreatedAt.Unix())) + "\n")
			for _, line := range strings.Split(strings.TrimSpace(comment.Body), "\n") {
				builder.WriteString("  " + strings.TrimRight(line, "\r") + "\n")
			}
		}

		return builder.String()
	})

	return strings.Join(threads, "\n\n")
}

func lastLines(diffHunk string, count int) []string {
	lines := lo.Filter(strings.Split(diffHunk, "\n"), func(line string, _ int) bool {
		return !strings.HasPrefix(line, "@@")
	})
	return lines[max(len(lines)-count, 0):]
}

func diffLineStyle(line string) style.TextStyle {
	switch {
	case strings.HasPrefix(line, "+"):
		return style.FgGreen
	case strings.HasPrefix(line, "-"):
		return style.FgRed
	default:
		return theme.DefaultTextColor
	}
}

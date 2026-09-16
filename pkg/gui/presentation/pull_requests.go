package presentation

import (
	"fmt"

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

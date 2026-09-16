package helpers

import (
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/jesseduffield/lazygit/pkg/commands/hosting_service"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

// A mention being typed at the end of the input: an @ at the start of the input
// or after whitespace, followed by the part of the login typed so far
var mentionAtEndRegex = regexp.MustCompile(`(?:^|\s)@([A-Za-z0-9-]*)$`)

// The GitHub users that can be mentioned in pull request comments, by repo.
// Loading them takes a request per page of users, so they're loaded once per
// repo and then kept.
type mentionableUsers struct {
	mutex   sync.Mutex
	byRepo  map[string][]string
	loading map[string]bool
}

// MentionSuggestionsFunc returns a function for suggesting GitHub users to
// mention while typing a comment on one of the repo's pull requests. The
// suggestions complete the mention being typed at the end of the input, so the
// prompt using them should set SuggestionsCompleteInput. Must be called on the
// UI thread.
func (self *PullRequestsHelper) MentionSuggestionsFunc() func(string) []*types.Suggestion {
	repo := *self.c.Model().PullRequestListState.Repo
	authors := lo.Map(self.c.Model().PullRequestList, func(pr *models.GithubPullRequest, _ int) string {
		return pr.Author
	})
	self.loadMentionableUsers(repo)

	return func(input string) []*types.Suggestion {
		match := mentionAtEndRegex.FindStringSubmatchIndex(input)
		if match == nil {
			return nil
		}

		textBeforeLogin, typedLogin := input[:match[2]], input[match[2]:]
		logins := lo.Uniq(append(self.mentionableUsersOf(repo), authors...))
		slices.Sort(logins)
		if typedLogin != "" {
			logins = utils.FilterStrings(typedLogin, logins, self.c.UserConfig().Gui.UseFuzzySearch())
		}

		return lo.Map(logins, func(login string, _ int) *types.Suggestion {
			return &types.Suggestion{
				Value: textBeforeLogin + login + " ",
				Label: style.FgCyan.Sprint("@" + login),
			}
		})
	}
}

func (self *PullRequestsHelper) mentionableUsersOf(repo hosting_service.ServiceInfo) []string {
	self.mentionableUsers.mutex.Lock()
	defer self.mentionableUsers.mutex.Unlock()

	return self.mentionableUsers.byRepo[repo.RepoName]
}

func (self *PullRequestsHelper) loadMentionableUsers(repo hosting_service.ServiceInfo) {
	users := &self.mentionableUsers
	users.mutex.Lock()
	defer users.mutex.Unlock()

	if _, loaded := users.byRepo[repo.RepoName]; loaded || users.loading[repo.RepoName] {
		return
	}
	if users.loading == nil {
		users.loading = map[string]bool{}
		users.byRepo = map[string][]string{}
	}
	users.loading[repo.RepoName] = true

	self.c.OnWorkerBackground(func(gocui.Task) error {
		logins, err := self.c.Git().GitHub.ListAssignableUsers(repo)

		users.mutex.Lock()
		delete(users.loading, repo.RepoName)
		if err == nil {
			users.byRepo[repo.RepoName] = logins
		}
		users.mutex.Unlock()

		if err != nil {
			// Suggestions are a convenience; mentions can still be typed out
			self.c.Log.Error(err)
			return nil
		}

		// Show the suggestions for what has been typed while they were loading
		self.c.OnUIThreadBackground(func() error {
			if strings.Contains(self.c.GetPromptInput(), "@") {
				self.c.Contexts().Suggestions.RefreshSuggestions()
			}
			return nil
		})
		return nil
	})
}

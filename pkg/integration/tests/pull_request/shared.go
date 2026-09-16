package pull_request

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// The pull requests panel talks to GitHub through gh, so the tests put a fake
// gh in its place. It answers the GraphQL query with pullRequestsJson, prints a
// placeholder for `gh pr view`, and appends the arguments of every other
// invocation to gh-calls.txt in the repo so tests can check what was run. The
// search query of each GraphQL request goes to gh-searches.txt.
//
// The remote lives on a host under .invalid (configured as a GitHub instance)
// so that the pull request lookups the branches panel does over HTTP fail
// straight away instead of reaching out to github.com.
var ghExtraEnvVars = map[string]string{
	"GH_PATH": "{{actualPath}}/bin/gh",
}

func setupGhConfig(cfg *config.AppConfig) {
	cfg.GetUserConfig().Services = map[string]string{
		"github.invalid": "github:github.invalid",
	}
}

const pullRequestsJson = `{"data":{"search":{"nodes":[
	{
		"number": 12,
		"title": "Add a feature",
		"url": "https://github.invalid/owner/repo/pull/12",
		"state": "OPEN",
		"isDraft": false,
		"headRefName": "feature",
		"baseRefName": "master",
		"reviewDecision": "APPROVED",
		"additions": 10,
		"deletions": 2,
		"updatedAt": "2026-09-16T14:00:27Z",
		"author": {"login": "alice"},
		"headRepositoryOwner": {"login": "owner"},
		"comments": {"totalCount": 0},
		"commits": {"nodes": [{"commit": {"statusCheckRollup": {"state": "SUCCESS"}}}]}
	},
	{
		"number": 11,
		"title": "Fix a bug",
		"url": "https://github.invalid/owner/repo/pull/11",
		"state": "OPEN",
		"isDraft": true,
		"headRefName": "bugfix",
		"baseRefName": "master",
		"additions": 1,
		"deletions": 1,
		"updatedAt": "2026-09-15T14:00:27Z",
		"author": {"login": "bob"},
		"headRepositoryOwner": {"login": "owner"},
		"comments": {"totalCount": 0},
		"commits": {"nodes": [{"commit": {"statusCheckRollup": {"state": "FAILURE"}}}]}
	}
]}}}`

func setupGhRepo(shell *Shell) {
	shell.EmptyCommit("initial")
	shell.RunCommand([]string{"git", "remote", "add", "origin", "https://github.invalid/owner/repo.git"})

	shell.CreateFile("../bin/pull_requests.json", pullRequestsJson)
	shell.CreateFile("../bin/gh", `#!/bin/sh
case "$1 $2" in
"auth token")
    echo fake-token
    ;;
"api graphql")
    for last; do :; done
    echo "$last" >> gh-searches.txt
    cat "$(dirname "$0")/pull_requests.json"
    ;;
"pr view")
    echo "Viewing pull request $3"
    ;;
*)
    echo "$@" >> gh-calls.txt
    ;;
esac
`)
	shell.MakeExecutable("../bin/gh")
}

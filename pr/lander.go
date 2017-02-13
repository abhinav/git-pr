package pr

import (
	"bytes"
	"errors"
	"strings"
	"text/template"

	"github.com/abhinav/git-fu/editor"

	"github.com/google/go-github/github"
)

var _interactiveTmpl = template.Must(template.New("interactive").Parse(
	`{{.Title}} (#{{.Number}})

{{if .Body}}{{.Body}}

{{end}}# Landing Pull Request: {{.HTMLURL}}
#
# Enter the commit message above. Lines starting with '#' will be
# ignored. There must be an empty line between the title and the body.
# Leaving this file empty will abort the operation.
`))

func parseInteractiveMessage(s string) (title string, body string, err error) {
	lines := strings.Split(s, "\n")
	{
		newLines := lines[:0]
		for _, l := range lines {
			if len(l) > 0 && l[0] == '#' {
				continue
			}

			newLines = append(newLines, l)
		}
		lines = newLines
	}

	if len(lines) == 0 {
		err = errors.New("file is empty")
		return
	}

	if len(lines) > 1 && len(lines[1]) > 0 {
		err = errors.New("there must be an empty line between the title and the body")
		return
	}

	title = lines[0]
	if len(lines) > 2 {
		body = strings.Join(lines[2:], "\n")
	}
	return
}

// PullRequestService provides read/write access to pull requests.
type PullRequestService interface {
	Merge(
		owner string, repo string, number int,
		commitMessage string,
		options *github.PullRequestOptions,
	) (*github.PullRequestMergeResult, *github.Response, error)
}

var _ PullRequestService = (*github.PullRequestsService)(nil)

// InteractiveLander lands pull requests while allowing users to edit the
// title and message interactively.
type InteractiveLander struct {
	Editor editor.Editor
	Pulls  PullRequestService
}

// Land the given pull request.
func (l *InteractiveLander) Land(pr *github.PullRequest) error {
	var buff bytes.Buffer
	if err := _interactiveTmpl.Execute(&buff, pr); err != nil {
		return err
	}

	message, err := l.Editor.EditString(buff.String())
	if err != nil {
		return err
	}

	title, body, err := parseInteractiveMessage(message)
	if err != nil {
		return err
	}

	result, _, err := l.Pulls.Merge(
		*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Number, body,
		&github.PullRequestOptions{CommitTitle: title, MergeMethod: "squash"})
	if err != nil {
		return err
	}

	if result.Merged == nil || !*result.Merged {
		return errors.New(*result.Message)
	}

	return nil
}

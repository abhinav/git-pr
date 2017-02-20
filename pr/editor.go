package pr

import (
	"bytes"
	"errors"
	"strings"
	"text/template"

	"github.com/abhinav/git-fu/editor"

	"github.com/google/go-github/github"
)

// UpdateMessage uses the given editor to edit the commit message of the given
// PR.
func UpdateMessage(ed editor.Editor, pr *github.PullRequest) error {
	var buff bytes.Buffer
	if err := _interactiveTmpl.Execute(&buff, pr); err != nil {
		return err
	}

	message, err := ed.EditString(buff.String())
	if err != nil {
		return err
	}

	title, body, err := _parseMessage(message)
	if err != nil {
		return err
	}

	pr.Title = &title
	pr.Body = &body
	return nil
}

var _interactiveTmpl = template.Must(template.New("interactive").Parse(
	`{{.Title}} (#{{.Number}})

{{if .Body}}{{.Body}}

{{end}}# Landing Pull Request: {{.HTMLURL}}
#
# Enter the commit message above. Lines starting with '#' will be
# ignored. There must be an empty line between the title and the body.
# Leaving this file empty will abort the operation.
`))

func _parseMessage(s string) (title string, body string, err error) {
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

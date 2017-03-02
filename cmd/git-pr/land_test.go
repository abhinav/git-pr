package main

import (
	"testing"

	"github.com/abhinav/git-fu/cli/clitest"
	"github.com/abhinav/git-fu/editor"
	"github.com/abhinav/git-fu/editor/editortest"
	"github.com/abhinav/git-fu/gateway/gatewaytest"
	"github.com/abhinav/git-fu/ptr"
	"github.com/abhinav/git-fu/repo"
	"github.com/abhinav/git-fu/service"
	"github.com/abhinav/git-fu/service/servicetest"

	"github.com/golang/mock/gomock"
	"github.com/google/go-github/github"
	"github.com/stretchr/testify/assert"
)

func TestLandCmd(t *testing.T) {
	type prMap map[string][]*github.PullRequest

	tests := []struct {
		Desc string

		Head          string
		CurrentBranch string

		// Map of branch name to pull requests with that head.
		PullRequestsByHead prMap

		ExpectLandRequest  *service.LandRequest
		ReturnLandResponse *service.LandResponse

		// If non-empty, an error with a message matching this will be
		// expected
		WantError string
	}{
		{
			Desc:               "no PRs",
			CurrentBranch:      "feature1",
			PullRequestsByHead: prMap{"feature1": nil},
			WantError:          `Could not find PRs with head "feature1"`,
		},
		{
			Desc:          "too many PRs",
			CurrentBranch: "feature2",
			PullRequestsByHead: prMap{
				"feature2": {
					{HTMLURL: ptr.String("foo")},
					{HTMLURL: ptr.String("bar")},
					{HTMLURL: ptr.String("baz")},
				},
			},
			WantError: `Too many PRs found with head "feature2"`,
		},
		{
			Desc:          "no arguments",
			CurrentBranch: "feature3",
			PullRequestsByHead: prMap{
				"feature3": {{HTMLURL: ptr.String("feature3")}},
			},
			ExpectLandRequest: &service.LandRequest{
				LocalBranch: "feature3",
				PullRequest: &github.PullRequest{
					HTMLURL: ptr.String("feature3"),
				},
			},
			ReturnLandResponse: &service.LandResponse{},
		},
		{
			Desc:          "explicit branch",
			Head:          "feature4",
			CurrentBranch: "master",
			PullRequestsByHead: prMap{
				"feature4": {{HTMLURL: ptr.String("feature4")}},
			},
			ExpectLandRequest: &service.LandRequest{
				PullRequest: &github.PullRequest{
					HTMLURL: ptr.String("feature4"),
				},
			},
			ReturnLandResponse: &service.LandResponse{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Desc, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			git := gatewaytest.NewMockGit(mockCtrl)
			github := gatewaytest.NewMockGitHub(mockCtrl)
			svc := servicetest.NewMockPR(mockCtrl)
			ed := editortest.NewMockEditor(mockCtrl)

			cb := &fakeConfigBuilder{
				ConfigBuilder: clitest.ConfigBuilder{
					Git:    git,
					GitHub: github,
					Repo:   &repo.Repo{Owner: "foo", Name: "bar"},
				},
				Service: svc,
			}
			cmd := landCmd{
				getConfig: cb.Build,
				getEditor: func(string) (editor.Editor, error) { return ed, nil },
			}
			cmd.Args.Branch = tt.Head
			if cmd.Editor == "" {
				cmd.Editor = "vi"
			}

			// Always return the current branch if requested.
			git.EXPECT().CurrentBranch().Return(tt.CurrentBranch, nil).AnyTimes()

			for head, prs := range tt.PullRequestsByHead {
				github.EXPECT().ListPullRequestsByHead("", head).Return(prs, nil)
			}

			if tt.ExpectLandRequest != nil {
				if tt.ExpectLandRequest.Editor == nil {
					tt.ExpectLandRequest.Editor = ed
				}
				svc.EXPECT().Land(tt.ExpectLandRequest).Return(tt.ReturnLandResponse, nil)
			}

			err := cmd.Execute(nil)
			if tt.WantError != "" {
				assert.Error(t, err, "expected failure")
				assert.Contains(t, err.Error(), tt.WantError)
			} else {
				assert.NoError(t, err, "command rebase failed")
			}
		})
	}
}

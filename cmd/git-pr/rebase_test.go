package main

import (
	"testing"

	"github.com/abhinav/git-pr/cli/clitest"
	"github.com/abhinav/git-pr/gateway/gatewaytest"
	"github.com/abhinav/git-pr/ptr"
	"github.com/abhinav/git-pr/repo"
	"github.com/abhinav/git-pr/service"
	"github.com/abhinav/git-pr/service/servicetest"

	"github.com/golang/mock/gomock"
	"github.com/google/go-github/github"
	"github.com/stretchr/testify/assert"
)

func TestRebaseCmd(t *testing.T) {
	type prMap map[string][]*github.PullRequest

	tests := []struct {
		// Test description
		Desc string

		Base     string
		Head     string
		OnlyMine bool

		// Name of the current branch (if requested)
		CurrentBranch string

		// Name of the current GitHub user
		GitHubUser string

		// Map of branch name to pull requests with that head.
		PullRequestsByHead prMap
		PullRequestsByBase prMap

		ExpectRebaseRequest  *service.RebaseRequest
		ReturnRebaseResponse *service.RebaseResponse

		// If non-empty, an error with a message matching this will be
		// expected
		WantError string
	}{
		{
			Desc:               "no matching PRs",
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
			Desc:          "no arguments, rebase dependents",
			CurrentBranch: "feature3",
			PullRequestsByHead: prMap{
				"feature3": {
					{Head: &github.PullRequestBranch{Ref: ptr.String("feature3")}},
				},
			},
			PullRequestsByBase: prMap{
				"feature3": {
					{HTMLURL: ptr.String("foo")},
					{HTMLURL: ptr.String("bar")},
					{HTMLURL: ptr.String("baz")},
				},
			},
			ExpectRebaseRequest: &service.RebaseRequest{
				PullRequests: []*github.PullRequest{
					{HTMLURL: ptr.String("foo")},
					{HTMLURL: ptr.String("bar")},
					{HTMLURL: ptr.String("baz")},
				},
				Base: "feature3",
			},
			ReturnRebaseResponse: &service.RebaseResponse{},
		},
		{
			Desc:          "explicit head branch",
			CurrentBranch: "master",
			Head:          "feature4",
			PullRequestsByHead: prMap{
				"feature4": {
					{
						Head: &github.PullRequestBranch{Ref: ptr.String("feature4")},
					},
				},
			},
			PullRequestsByBase: prMap{
				"feature4": {
					{HTMLURL: ptr.String("foo")},
					{HTMLURL: ptr.String("bar")},
					{HTMLURL: ptr.String("baz")},
				},
			},
			ExpectRebaseRequest: &service.RebaseRequest{
				PullRequests: []*github.PullRequest{
					{HTMLURL: ptr.String("foo")},
					{HTMLURL: ptr.String("bar")},
					{HTMLURL: ptr.String("baz")},
				},
				Base: "feature4",
			},
			ReturnRebaseResponse: &service.RebaseResponse{},
		},
		{
			Desc:          "explicit base branch",
			CurrentBranch: "feature5",
			Base:          "dev",
			PullRequestsByHead: prMap{
				"feature5": {
					{
						Head:    &github.PullRequestBranch{Ref: ptr.String("feature5")},
						HTMLURL: ptr.String("feature5"),
					},
				},
			},
			ExpectRebaseRequest: &service.RebaseRequest{
				PullRequests: []*github.PullRequest{
					{
						Head:    &github.PullRequestBranch{Ref: ptr.String("feature5")},
						HTMLURL: ptr.String("feature5"),
					},
				},
				Base: "dev",
			},
			ReturnRebaseResponse: &service.RebaseResponse{},
		},
		{
			Desc:          "only mine",
			CurrentBranch: "feature6",
			OnlyMine:      true,
			GitHubUser:    "foo",
			PullRequestsByHead: prMap{
				"feature6": {
					{Head: &github.PullRequestBranch{Ref: ptr.String("feature6")}},
				},
			},
			PullRequestsByBase: prMap{
				"feature6": {
					{
						HTMLURL: ptr.String("x"),
						User:    &github.User{Login: ptr.String("foo")},
					},
					{
						HTMLURL: ptr.String("y"),
						User:    &github.User{Login: ptr.String("bar")},
					},
					{
						HTMLURL: ptr.String("z"),
						User:    &github.User{Login: ptr.String("foo")},
					},
				},
			},
			ExpectRebaseRequest: &service.RebaseRequest{
				Author: "foo",
				PullRequests: []*github.PullRequest{
					{
						HTMLURL: ptr.String("x"),
						User:    &github.User{Login: ptr.String("foo")},
					},
					{
						HTMLURL: ptr.String("y"),
						User:    &github.User{Login: ptr.String("bar")},
					},
					{
						HTMLURL: ptr.String("z"),
						User:    &github.User{Login: ptr.String("foo")},
					},
				},
				Base: "feature6",
			},
			ReturnRebaseResponse: &service.RebaseResponse{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Desc, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			git := gatewaytest.NewMockGit(mockCtrl)
			github := gatewaytest.NewMockGitHub(mockCtrl)
			svc := servicetest.NewMockPR(mockCtrl)

			cb := &fakeConfigBuilder{
				ConfigBuilder: clitest.ConfigBuilder{
					Git:        git,
					GitHub:     github,
					Repo:       &repo.Repo{Owner: "foo", Name: "bar"},
					GitHubUser: tt.GitHubUser,
				},
				Service: svc,
			}
			cmd := rebaseCmd{
				getConfig: cb.Build,
				Base:      tt.Base,
				OnlyMine:  tt.OnlyMine,
			}
			cmd.Args.Branch = tt.Head

			// Always return the current branch if requested.
			git.EXPECT().CurrentBranch().Return(tt.CurrentBranch, nil).AnyTimes()

			for head, prs := range tt.PullRequestsByHead {
				github.EXPECT().ListPullRequestsByHead(gomock.Any(), "", head).Return(prs, nil)
			}

			for base, prs := range tt.PullRequestsByBase {
				github.EXPECT().ListPullRequestsByBase(gomock.Any(), base).Return(prs, nil)
			}

			if tt.ExpectRebaseRequest != nil {
				svc.EXPECT().Rebase(gomock.Any(), tt.ExpectRebaseRequest).Return(tt.ReturnRebaseResponse, nil)
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

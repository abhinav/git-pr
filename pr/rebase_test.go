package pr

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/abhinav/git-pr/gateway"
	"github.com/abhinav/git-pr/gateway/gatewaytest"
	"github.com/abhinav/git-pr/git"
	"github.com/abhinav/git-pr/git/gittest"
	"github.com/abhinav/git-pr/service"

	"github.com/golang/mock/gomock"
	"github.com/google/go-github/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceRebase(t *testing.T) {
	type testCase struct {
		Desc string

		Request service.RebaseRequest

		// Values to return from rebasePullRequests
		RebasePRsResult []rebasedPullRequest
		RebasePRsError  error

		// Skip common mock setup if true.
		SkipCommon bool

		// SHA1 hashes for branches that may be queried. May be empty or
		// partial if SetupGit is handling this.
		SHA1Hashes map[string]string // branch name -> sha1 hash

		// Branches which don't have a local version. SHA lookup for these
		// will fail.
		SHA1Failures []string

		// If present, these may be used for more complicated setup on the
		// mocks.
		SetupGit    func(*gatewaytest.MockGit)
		SetupGitHub func(*gatewaytest.MockGitHub)

		// Expected Git.ResetBranch calls. May be empty or partial if SetupGit
		// is handling this.
		WantBranchResets []string // branch name -> ref

		// Expected items in Push(). May be empty if SetupGitHub is handling
		// this.
		WantPushes map[string]string // local ref -> branch name

		// List of pull requests for which we expect the PR base to change to
		// Request.Base. May be empty or partial if SetupGitHub is handling
		// this.
		WantBaseChanges []int

		WantResponse service.RebaseResponse
		WantErrors   []string
	}

	tests := []testCase{
		{
			Desc:         "empty",
			Request:      service.RebaseRequest{Base: "foo"},
			WantResponse: service.RebaseResponse{},
			SkipCommon:   true,
		},
		func() (tt testCase) {
			tt.Desc = "single"

			pr := &github.PullRequest{
				Number:  github.Int(1),
				HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/1"),
				Base: &github.PullRequestBranch{
					Ref: github.String("master"),
				},
				Head: &github.PullRequestBranch{
					SHA: github.String("headsha"),
					Ref: github.String("myfeature"),
				},
			}

			tt.Request = service.RebaseRequest{
				Base:         "master",
				PullRequests: []*github.PullRequest{pr},
			}
			tt.RebasePRsResult = []rebasedPullRequest{
				{PR: pr, LocalRef: "git-pr/rebase/headsha"},
			}
			tt.SHA1Hashes = map[string]string{"myfeature": "headsha"}

			tt.WantPushes = map[string]string{"git-pr/rebase/headsha": "myfeature"}
			tt.WantBranchResets = []string{"myfeature"}

			return
		}(),
		{
			Desc: "no rebases",
			Request: service.RebaseRequest{
				Base: "master",
				PullRequests: []*github.PullRequest{
					&github.PullRequest{
						Number:  github.Int(1),
						HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/1"),
						Base: &github.PullRequestBranch{
							Ref: github.String("master"),
						},
						Head: &github.PullRequestBranch{
							SHA: github.String("headsha"),
							Ref: github.String("myfeature"),
						},
					},
				},
			},
			RebasePRsResult: []rebasedPullRequest{},
		},
		func() (tt testCase) {
			tt.Desc = "single base change"

			pr := &github.PullRequest{
				Number:  github.Int(1),
				HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/1"),
				Base: &github.PullRequestBranch{
					Ref: github.String("dev"),
				},
				Head: &github.PullRequestBranch{
					SHA: github.String("somesha"),
					Ref: github.String("myfeature"),
				},
			}

			tt.Request = service.RebaseRequest{
				Base:         "master",
				PullRequests: []*github.PullRequest{pr},
			}
			tt.RebasePRsResult = []rebasedPullRequest{
				{PR: pr, LocalRef: "git-pr/rebase/somesha"},
			}
			tt.SHA1Hashes = map[string]string{"myfeature": "differentsha"}

			tt.WantPushes = map[string]string{"git-pr/rebase/somesha": "myfeature"}
			tt.WantBaseChanges = []int{1}
			tt.WantResponse = service.RebaseResponse{
				BranchesNotUpdated: []string{"myfeature"},
			}

			return
		}(),
		func() (tt testCase) {
			tt.Desc = "multiple"

			pr1 := &github.PullRequest{
				Number:  github.Int(1),
				HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/1"),
				Base: &github.PullRequestBranch{
					Ref: github.String("dev"),
				},
				Head: &github.PullRequestBranch{
					SHA: github.String("sha1"),
					Ref: github.String("feature-1"),
				},
			}

			pr2 := &github.PullRequest{
				Number:  github.Int(2),
				HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/2"),
				Base: &github.PullRequestBranch{
					Ref: github.String("master"),
				},
				Head: &github.PullRequestBranch{
					SHA: github.String("sha2"),
					Ref: github.String("feature-2"),
				},
			}

			pr3 := &github.PullRequest{
				Number:  github.Int(3),
				HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/3"),
				Base: &github.PullRequestBranch{
					Ref: github.String("master"),
				},
				Head: &github.PullRequestBranch{
					SHA: github.String("sha3"),
					Ref: github.String("feature-3"),
				},
			}

			tt.Request = service.RebaseRequest{
				Base:         "dev",
				PullRequests: []*github.PullRequest{pr1, pr2, pr3},
			}
			tt.RebasePRsResult = []rebasedPullRequest{
				{PR: pr1, LocalRef: "git-pr/rebase/sha1"},
				{PR: pr2, LocalRef: "git-pr/rebase/sha2"},
				{PR: pr3, LocalRef: "git-pr/rebase/sha3"},
			}
			tt.SHA1Hashes = map[string]string{
				"feature-1": "sha1",
				"feature-2": "sha2",
				"feature-3": "not-sha3",
			}
			tt.WantPushes = map[string]string{
				"git-pr/rebase/sha1": "feature-1",
				"git-pr/rebase/sha2": "feature-2",
				"git-pr/rebase/sha3": "feature-3",
			}
			tt.WantBaseChanges = []int{2, 3}
			tt.WantBranchResets = []string{"feature-1", "feature-2"}
			tt.WantResponse = service.RebaseResponse{
				BranchesNotUpdated: []string{"feature-3"},
			}

			return
		}(),
		func() (tt testCase) {
			tt.Desc = "simple stack"

			// dev -> feature-1 -> feature-2 -> feature-3

			pr := &github.PullRequest{
				Number:  github.Int(1),
				HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/1"),
				Base: &github.PullRequestBranch{
					Ref: github.String("dev"),
				},
				Head: &github.PullRequestBranch{
					SHA: github.String("sha1"),
					Ref: github.String("feature-1"),
				},
			}

			tt.Request = service.RebaseRequest{
				Base:         "dev",
				PullRequests: []*github.PullRequest{pr},
			}
			tt.RebasePRsResult = []rebasedPullRequest{
				{PR: pr, LocalRef: "git-pr/rebase/sha1"},
				{
					LocalRef: "git-pr/rebase/sha2",
					PR: &github.PullRequest{
						Number:  github.Int(2),
						HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/2"),
						Base: &github.PullRequestBranch{
							Ref: github.String("feature-1"),
						},
						Head: &github.PullRequestBranch{
							SHA: github.String("sha2"),
							Ref: github.String("feature-2"),
						},
					},
				},
				{
					LocalRef: "git-pr/rebase/sha3",
					PR: &github.PullRequest{
						Number:  github.Int(3),
						HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/3"),
						Base: &github.PullRequestBranch{
							Ref: github.String("feature-2"),
						},
						Head: &github.PullRequestBranch{
							SHA: github.String("sha3"),
							Ref: github.String("feature-3"),
						},
					},
				},
			}

			tt.SHA1Hashes = map[string]string{
				"feature-1": "not-sha1",
				"feature-3": "sha3",
			}
			tt.SHA1Failures = []string{"feature-2"}

			tt.WantBranchResets = []string{"feature-3"}
			tt.WantPushes = map[string]string{
				"git-pr/rebase/sha1": "feature-1",
				"git-pr/rebase/sha2": "feature-2",
				"git-pr/rebase/sha3": "feature-3",
			}
			tt.WantResponse = service.RebaseResponse{
				BranchesNotUpdated: []string{"feature-1"},
			}

			return
		}(),
		func() (tt testCase) {
			tt.Desc = "graph"

			// dev-----------.
			//  |             \
			//  +-> feature-1  +-> feature-2 -> feature-3
			//  |                    |
			//  +-> feature-4        +-> feature-5
			//                                |
			//                                +-> feature-6

			pr1 := &github.PullRequest{
				Number:  github.Int(1),
				HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/1"),
				Base: &github.PullRequestBranch{
					Ref: github.String("dev"),
				},
				Head: &github.PullRequestBranch{
					SHA: github.String("sha1"),
					Ref: github.String("feature-1"),
				},
			}

			pr2 := &github.PullRequest{
				Number:  github.Int(2),
				HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/2"),
				Base: &github.PullRequestBranch{
					Ref: github.String("master"),
				},
				Head: &github.PullRequestBranch{
					SHA: github.String("sha2"),
					Ref: github.String("feature-2"),
				},
			}

			tt.Request = service.RebaseRequest{
				Base:         "dev",
				PullRequests: []*github.PullRequest{pr1, pr2},
			}
			tt.RebasePRsResult = []rebasedPullRequest{
				{PR: pr1, LocalRef: "git-pr/rebase/sha1"},
				{PR: pr2, LocalRef: "git-pr/rebase/sha2"},
				{
					LocalRef: "git-pr/rebase/sha3",
					PR: &github.PullRequest{
						Number:  github.Int(3),
						HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/3"),
						Base: &github.PullRequestBranch{
							Ref: github.String("feature-2"),
						},
						Head: &github.PullRequestBranch{
							SHA: github.String("sha3"),
							Ref: github.String("feature-3"),
						},
					},
				},
				{
					LocalRef: "git-pr/rebase/sha4",
					PR: &github.PullRequest{
						Number:  github.Int(4),
						HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/4"),
						Base: &github.PullRequestBranch{
							Ref: github.String("feature-1"),
						},
						Head: &github.PullRequestBranch{
							SHA: github.String("sha4"),
							Ref: github.String("feature-4"),
						},
					},
				},
				{
					LocalRef: "git-pr/rebase/sha5",
					PR: &github.PullRequest{
						Number:  github.Int(5),
						HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/5"),
						Base: &github.PullRequestBranch{
							Ref: github.String("feature-2"),
						},
						Head: &github.PullRequestBranch{
							SHA: github.String("sha5"),
							Ref: github.String("feature-5"),
						},
					},
				},
				{
					LocalRef: "git-pr/rebase/sha6",
					PR: &github.PullRequest{
						Number:  github.Int(6),
						HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/6"),
						Base: &github.PullRequestBranch{
							Ref: github.String("feature-5"),
						},
						Head: &github.PullRequestBranch{
							SHA: github.String("sha6"),
							Ref: github.String("feature-6"),
						},
					},
				},
			}

			tt.SHA1Hashes = map[string]string{
				"feature-1": "sha1",
				"feature-2": "not-sha2",
				"feature-4": "not-sha4",
				"feature-5": "sha5",
			}
			tt.SHA1Failures = []string{"feature-3", "feature-6"}

			tt.WantBranchResets = []string{"feature-1", "feature-5"}
			tt.WantPushes = map[string]string{
				"git-pr/rebase/sha1": "feature-1",
				"git-pr/rebase/sha2": "feature-2",
				"git-pr/rebase/sha3": "feature-3",
				"git-pr/rebase/sha4": "feature-4",
				"git-pr/rebase/sha5": "feature-5",
				"git-pr/rebase/sha6": "feature-6",
			}
			tt.WantBaseChanges = []int{2}
			tt.WantResponse = service.RebaseResponse{
				BranchesNotUpdated: []string{"feature-2", "feature-4"},
			}

			return
		}(),
		{
			Desc: "current branch error",
			Request: service.RebaseRequest{
				Base: "derp",
				PullRequests: []*github.PullRequest{
					{}, // doesn't matter
				},
			},
			SkipCommon: true,
			SetupGit: func(git *gatewaytest.MockGit) {
				git.EXPECT().CurrentBranch().
					Return("", errors.New("not a git repository"))
			},
			WantErrors: []string{"not a git repository"},
		},
		{
			Desc: "fetch error",
			Request: service.RebaseRequest{
				Base: "derp",
				PullRequests: []*github.PullRequest{
					{}, // doesn't matter
				},
			},
			SkipCommon: true,
			SetupGit: func(git *gatewaytest.MockGit) {
				git.EXPECT().CurrentBranch().Return("master", nil)
				git.EXPECT().Fetch(&gateway.FetchRequest{
					Remote: "origin",
				}).Return(errors.New("remote origin doesn't exist"))
				git.EXPECT().Checkout("master").Return(nil)
			},
			WantErrors: []string{"remote origin doesn't exist"},
		},
		{
			Desc: "fetch error",
			Request: service.RebaseRequest{
				Base: "derp",
				PullRequests: []*github.PullRequest{
					{}, // doesn't matter
				},
			},
			SkipCommon: true,
			SetupGit: func(git *gatewaytest.MockGit) {
				git.EXPECT().CurrentBranch().Return("master", nil)
				git.EXPECT().Fetch(&gateway.FetchRequest{Remote: "origin"}).Return(nil)

				git.EXPECT().SHA1("origin/derp").
					Return("", errors.New("could not find ref origin/derp"))

				git.EXPECT().Checkout("master").Return(nil)
			},
			WantErrors: []string{"could not find ref origin/derp"},
		},
		{
			Desc: "rebase error",
			Request: service.RebaseRequest{
				Base: "derp",
				PullRequests: []*github.PullRequest{
					{}, // doesn't matter
				},
			},
			RebasePRsError: errors.New("could not rebase stuff"),
			WantErrors:     []string{"could not rebase stuff"},
		},
		func() (tt testCase) {
			tt.Desc = "push error"

			pr := &github.PullRequest{
				Number:  github.Int(1),
				HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/1"),
				Base: &github.PullRequestBranch{
					Ref: github.String("master"),
				},
				Head: &github.PullRequestBranch{
					SHA: github.String("headsha"),
					Ref: github.String("myfeature"),
				},
			}

			tt.Request = service.RebaseRequest{
				Base:         "master",
				PullRequests: []*github.PullRequest{pr},
			}
			tt.RebasePRsResult = []rebasedPullRequest{
				{PR: pr, LocalRef: "git-pr/rebase/headsha"},
			}
			tt.SHA1Hashes = map[string]string{"myfeature": "headsha"}

			tt.SetupGit = func(git *gatewaytest.MockGit) {
				git.EXPECT().Push(gomock.Any()).
					Return(errors.New("remote timed out"))
			}
			tt.WantErrors = []string{"remote timed out"}

			return
		}(),
		func() (tt testCase) {
			tt.Desc = "update base error"

			pr := &github.PullRequest{
				Number:  github.Int(1),
				HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/1"),
				Base: &github.PullRequestBranch{
					Ref: github.String("master"),
				},
				Head: &github.PullRequestBranch{
					SHA: github.String("headsha"),
					Ref: github.String("myfeature"),
				},
			}

			tt.Request = service.RebaseRequest{
				Base:         "dev",
				PullRequests: []*github.PullRequest{pr},
			}
			tt.RebasePRsResult = []rebasedPullRequest{
				{PR: pr, LocalRef: "git-pr/rebase/headsha"},
			}
			tt.SHA1Hashes = map[string]string{"myfeature": "headsha"}

			tt.WantPushes = map[string]string{"git-pr/rebase/headsha": "myfeature"}
			tt.WantBranchResets = []string{"myfeature"}

			tt.SetupGitHub = func(gh *gatewaytest.MockGitHub) {
				gh.EXPECT().SetPullRequestBase(gomock.Any(), 1, "dev").
					Return(errors.New("unauthorized operation"))
			}
			tt.WantErrors = []string{"unauthorized operation"}

			return
		}(),
	}

	for _, tt := range tests {
		t.Run(tt.Desc, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			git := gatewaytest.NewMockGit(mockCtrl)
			gh := gatewaytest.NewMockGitHub(mockCtrl)

			if !tt.SkipCommon {
				git.EXPECT().CurrentBranch().Return("oldbranch", nil)
				git.EXPECT().Checkout("oldbranch").Return(nil)
				git.EXPECT().Fetch(&gateway.FetchRequest{Remote: "origin"}).Return(nil)
				git.EXPECT().SHA1("origin/"+tt.Request.Base).Return("originbasesha", nil)
			}

			for _, branch := range tt.WantBranchResets {
				git.EXPECT().ResetBranch(branch, "origin/"+branch).Return(nil)
			}

			if len(tt.WantPushes) > 0 {
				git.EXPECT().Push(&gateway.PushRequest{
					Remote: "origin",
					Force:  true,
					Refs:   tt.WantPushes,
				}).Return(nil)
			}

			for branch, sha := range tt.SHA1Hashes {
				git.EXPECT().SHA1(branch).Return(sha, nil)
			}
			for _, branch := range tt.SHA1Failures {
				git.EXPECT().SHA1(branch).
					Return("", fmt.Errorf("unknown branch %q", branch))
			}

			for _, prNum := range tt.WantBaseChanges {
				gh.EXPECT().
					SetPullRequestBase(gomock.Any(), prNum, tt.Request.Base).
					Return(nil)
			}

			if tt.SetupGit != nil {
				tt.SetupGit(git)
			}
			if tt.SetupGitHub != nil {
				tt.SetupGitHub(gh)
			}

			service := NewService(ServiceConfig{
				Git:    git,
				GitHub: gh,
			})
			service.rebasePullRequests = fakeRebasePullRequests(
				tt.RebasePRsResult, tt.RebasePRsError)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			res, err := service.Rebase(ctx, &tt.Request)
			if len(tt.WantErrors) > 0 {
				require.Error(t, err, "expected failure")
				for _, msg := range tt.WantErrors {
					assert.Contains(t, err.Error(), msg)
				}
				return
			}

			require.NoError(t, err, "expected success")

			wantBranchesNotUpdated := make(map[string]struct{}, len(tt.WantResponse.BranchesNotUpdated))
			for _, br := range tt.WantResponse.BranchesNotUpdated {
				wantBranchesNotUpdated[br] = struct{}{}
			}

			gotBranchesNotUpdated := make(map[string]struct{}, len(res.BranchesNotUpdated))
			for _, br := range res.BranchesNotUpdated {
				gotBranchesNotUpdated[br] = struct{}{}
			}

			assert.Equal(t, wantBranchesNotUpdated, gotBranchesNotUpdated,
				"BranchesNotUpdated must match")
		})
	}
}

func fakeRebasePullRequests(
	results []rebasedPullRequest, err error,
) func(rebasePRConfig) (map[int]rebasedPullRequest, error) {
	// For convenience, build the map from the list rather than manually in
	// the test case.
	resultMap := make(map[int]rebasedPullRequest)
	for _, r := range results {
		resultMap[r.PR.GetNumber()] = r
	}

	return func(rebasePRConfig) (map[int]rebasedPullRequest, error) {
		return resultMap, err
	}
}

type fakeRebase struct {
	// Range of commits to rebase
	FromRef string
	ToRef   string

	// ToRef after the rebase. This will be the Base of the returned by
	// the RebaseHandle.
	GiveRef string

	// Rebases expected on the returned handle
	WantRebases []fakeRebase
}

func setupFakeRebases(ctrl *gomock.Controller, h *gittest.MockRebaseHandle, rebases []fakeRebase) {
	for _, r := range rebases {
		newH := gittest.NewMockRebaseHandle(ctrl)
		newH.EXPECT().Base().Return(r.GiveRef)
		setupFakeRebases(ctrl, newH, r.WantRebases)

		h.EXPECT().Rebase(r.FromRef, r.ToRef).Return(newH)
	}
}

func TestRebasePullRequests(t *testing.T) {
	type testCase struct {
		Desc string

		Author       string
		Base         string
		PullRequests []*github.PullRequest

		// Dependents of different pull requests. May be partial or empty if
		// part of the work is done by SetupGitHub.
		Dependents map[string][]*github.PullRequest // base branch -> PRs

		// Whether the given branches are owned by the current repo or not.
		// May be partial or empty if part of the work is done by SetupGitHub.
		BranchOwnership map[string]bool

		// Customize the GitHub gateway mock.
		SetupGitHub func(*gatewaytest.MockGitHub)

		// Whether the bulkRebaser fails
		RebaserError error

		// Rebases expected on the base branch.
		WantRebases []fakeRebase

		WantResults []rebasedPullRequest
		WantErrors  []string
	}

	tests := []testCase{
		{Desc: "empty", Base: "shrug"},
		func() (tt testCase) {
			tt.Desc = "single"

			pr := &github.PullRequest{
				Number:  github.Int(1),
				HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/1"),
				Base: &github.PullRequestBranch{
					SHA: github.String("basesha"),
				},
				Head: &github.PullRequestBranch{
					SHA: github.String("headsha"),
					Ref: github.String("feature-1"),
				},
			}

			tt.Base = "origin/master"
			tt.PullRequests = []*github.PullRequest{pr}
			tt.BranchOwnership = map[string]bool{
				"feature-1": true,
			}
			tt.Dependents = map[string][]*github.PullRequest{"feature-1": {}}
			tt.WantRebases = []fakeRebase{
				{
					FromRef: "basesha",
					ToRef:   "headsha",
					GiveRef: "newsha",
				},
			}
			tt.WantResults = []rebasedPullRequest{
				{LocalRef: "newsha", PR: pr},
			}

			return
		}(),
		func() (tt testCase) {
			tt.Desc = "single wrong author"

			pr := &github.PullRequest{
				Number:  github.Int(1),
				HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/1"),
				Base: &github.PullRequestBranch{
					SHA: github.String("basesha"),
				},
				User: &github.User{Login: github.String("probablynotarealusername")},
				Head: &github.PullRequestBranch{
					SHA: github.String("headsha"),
					Ref: github.String("feature-1"),
				},
			}

			tt.Author = "abhinav"
			tt.Base = "origin/master"
			tt.PullRequests = []*github.PullRequest{pr}
			tt.BranchOwnership = map[string]bool{"feature-1": true}

			return
		}(),
		func() (tt testCase) {
			tt.Desc = "github dependents failure"

			pr := &github.PullRequest{
				Number:  github.Int(1),
				HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/1"),
				Base: &github.PullRequestBranch{
					SHA: github.String("basesha"),
				},
				Head: &github.PullRequestBranch{
					SHA: github.String("headsha"),
					Ref: github.String("feature-1"),
				},
			}

			tt.Base = "origin/master"
			tt.PullRequests = []*github.PullRequest{pr}
			tt.BranchOwnership = map[string]bool{"feature-1": true}

			tt.SetupGitHub = func(gh *gatewaytest.MockGitHub) {
				gh.EXPECT().ListPullRequestsByBase(gomock.Any(), "feature-1").
					Return(nil, errors.New("great sadness"))
			}

			tt.RebaserError = errors.New("great sadness")
			tt.WantRebases = []fakeRebase{
				{
					FromRef: "basesha",
					ToRef:   "headsha",
					GiveRef: "newsha",
				},
			}
			tt.WantErrors = []string{"great sadness"}

			return
		}(),
		func() (tt testCase) {
			tt.Desc = "rebase failure"

			pr := &github.PullRequest{
				Number:  github.Int(1),
				HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/1"),
				Base: &github.PullRequestBranch{
					SHA: github.String("basesha"),
				},
				Head: &github.PullRequestBranch{
					SHA: github.String("headsha"),
					Ref: github.String("feature-1"),
				},
			}

			tt.Base = "origin/master"
			tt.PullRequests = []*github.PullRequest{pr}
			tt.BranchOwnership = map[string]bool{
				"feature-1": true,
			}
			tt.Dependents = map[string][]*github.PullRequest{"feature-1": {}}
			tt.RebaserError = errors.New("great sadness")
			tt.WantRebases = []fakeRebase{
				{
					FromRef: "basesha",
					ToRef:   "headsha",
					GiveRef: "newsha",
				},
			}
			tt.WantErrors = []string{"great sadness"}

			return
		}(),
		func() (tt testCase) {
			tt.Desc = "single not owned"

			pr := &github.PullRequest{
				Number:  github.Int(1),
				HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/1"),
				Base: &github.PullRequestBranch{
					SHA: github.String("basesha"),
				},
				Head: &github.PullRequestBranch{
					SHA: github.String("headsha"),
					Ref: github.String("feature-1"),
				},
			}

			tt.Base = "origin/master"
			tt.PullRequests = []*github.PullRequest{pr}
			tt.BranchOwnership = map[string]bool{
				"feature-1": false,
			}
			tt.WantRebases = []fakeRebase{}
			tt.WantResults = []rebasedPullRequest{}

			return
		}(),
		func() (tt testCase) {
			tt.Desc = "stack"

			pr1 := &github.PullRequest{
				Number:  github.Int(1),
				HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/1"),
				Base: &github.PullRequestBranch{
					SHA: github.String("mastersha"),
				},
				Head: &github.PullRequestBranch{
					SHA: github.String("sha1"),
					Ref: github.String("feature-1"),
				},
			}

			pr2 := &github.PullRequest{
				Number:  github.Int(2),
				HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/2"),
				Base: &github.PullRequestBranch{
					SHA: github.String("sha1"),
				},
				Head: &github.PullRequestBranch{
					SHA: github.String("sha2"),
					Ref: github.String("feature-2"),
				},
			}

			pr3 := &github.PullRequest{
				Number:  github.Int(3),
				HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/3"),
				Base: &github.PullRequestBranch{
					SHA: github.String("sha2"),
				},
				Head: &github.PullRequestBranch{
					SHA: github.String("sha3"),
					Ref: github.String("feature-3"),
				},
			}

			tt.Base = "origin/master"
			tt.PullRequests = []*github.PullRequest{pr1}
			tt.Dependents = map[string][]*github.PullRequest{
				"feature-1": []*github.PullRequest{pr2},
				"feature-2": []*github.PullRequest{pr3},
				"feature-3": []*github.PullRequest{},
			}
			tt.BranchOwnership = map[string]bool{
				"feature-1": true,
				"feature-2": true,
				"feature-3": true,
			}

			tt.WantRebases = []fakeRebase{
				{
					FromRef: "mastersha",
					ToRef:   "sha1",
					GiveRef: "newsha1",
					WantRebases: []fakeRebase{
						{
							FromRef: "sha1",
							ToRef:   "sha2",
							GiveRef: "newsha2",
							WantRebases: []fakeRebase{
								{FromRef: "sha2", ToRef: "sha3", GiveRef: "newsha3"},
							},
						},
					},
				},
			}
			tt.WantResults = []rebasedPullRequest{
				{LocalRef: "newsha1", PR: pr1},
				{LocalRef: "newsha2", PR: pr2},
				{LocalRef: "newsha3", PR: pr3},
			}

			return
		}(),
		func() (tt testCase) {
			tt.Desc = "stack partly not owned"

			pr1 := &github.PullRequest{
				Number:  github.Int(1),
				HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/1"),
				Base: &github.PullRequestBranch{
					SHA: github.String("mastersha"),
				},
				Head: &github.PullRequestBranch{
					SHA: github.String("sha1"),
					Ref: github.String("feature-1"),
				},
			}

			pr2 := &github.PullRequest{
				Number:  github.Int(2),
				HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/2"),
				Base: &github.PullRequestBranch{
					SHA: github.String("sha1"),
				},
				Head: &github.PullRequestBranch{
					SHA: github.String("sha2"),
					Ref: github.String("feature-2"),
				},
			}

			tt.Base = "origin/master"
			tt.PullRequests = []*github.PullRequest{pr1}
			tt.Dependents = map[string][]*github.PullRequest{
				"feature-1": []*github.PullRequest{pr2},
			}
			tt.BranchOwnership = map[string]bool{
				"feature-1": true,
				"feature-2": false,
			}

			tt.WantRebases = []fakeRebase{
				{
					FromRef: "mastersha",
					ToRef:   "sha1",
					GiveRef: "newsha1",
				},
			}
			tt.WantResults = []rebasedPullRequest{
				{LocalRef: "newsha1", PR: pr1},
			}

			return
		}(),
		func() (tt testCase) {
			tt.Desc = "stack partly wrong user"

			pr1 := &github.PullRequest{
				Number:  github.Int(1),
				HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/1"),
				User:    &github.User{Login: github.String("abhinav")},
				Base: &github.PullRequestBranch{
					SHA: github.String("mastersha"),
				},
				Head: &github.PullRequestBranch{
					SHA: github.String("sha1"),
					Ref: github.String("feature-1"),
				},
			}

			pr2 := &github.PullRequest{
				Number:  github.Int(2),
				HTMLURL: github.String("http://github.com/abhinav/git-pr/pulls/2"),
				User:    &github.User{Login: github.String("probablynotarealusername")},
				Base: &github.PullRequestBranch{
					SHA: github.String("sha1"),
				},
				Head: &github.PullRequestBranch{
					SHA: github.String("sha2"),
					Ref: github.String("feature-2"),
				},
			}

			tt.Author = "abhinav"
			tt.Base = "origin/master"
			tt.PullRequests = []*github.PullRequest{pr1}
			tt.Dependents = map[string][]*github.PullRequest{
				"feature-1": []*github.PullRequest{pr2},
			}
			tt.BranchOwnership = map[string]bool{
				"feature-1": true,
				"feature-2": true,
			}

			tt.WantRebases = []fakeRebase{
				{
					FromRef: "mastersha",
					ToRef:   "sha1",
					GiveRef: "newsha1",
				},
			}
			tt.WantResults = []rebasedPullRequest{
				{LocalRef: "newsha1", PR: pr1},
			}

			return
		}(),
	}

	for _, tt := range tests {
		t.Run(tt.Desc, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			rebaser := newMockBulkRebaser(mockCtrl)
			gh := gatewaytest.NewMockGitHub(mockCtrl)

			for branch, deps := range tt.Dependents {
				gh.EXPECT().
					ListPullRequestsByBase(gomock.Any(), branch).
					Return(deps, nil)
			}

			for br, owned := range tt.BranchOwnership {
				gh.EXPECT().
					IsOwned(gomock.Any(), prBranchMatcher(br)).
					Return(owned)
			}

			if tt.SetupGitHub != nil {
				tt.SetupGitHub(gh)
			}

			mockHandle := gittest.NewMockRebaseHandle(mockCtrl)
			setupFakeRebases(mockCtrl, mockHandle, tt.WantRebases)
			rebaser.EXPECT().Onto(tt.Base).Return(mockHandle)
			rebaser.EXPECT().Err().Return(tt.RebaserError).AnyTimes()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			results, err := rebasePullRequests(rebasePRConfig{
				Context:      ctx,
				GitRebaser:   rebaser,
				GitHub:       gh,
				Author:       tt.Author,
				Base:         tt.Base,
				PullRequests: tt.PullRequests,
			})

			if len(tt.WantErrors) > 0 {
				require.Error(t, err, "expected failure")
				for _, msg := range tt.WantErrors {
					assert.Contains(t, err.Error(), msg)
				}
				return
			}

			require.NoError(t, err, "expected success")

			wantResults := make(map[int]rebasedPullRequest)
			for _, r := range tt.WantResults {
				wantResults[r.PR.GetNumber()] = r
			}
			assert.Equal(t, wantResults, results)
		})
	}
}

type mockBulkRebaser struct {
	ctrl *gomock.Controller
}

var _ bulkRebaser = (*mockBulkRebaser)(nil)

func newMockBulkRebaser(ctrl *gomock.Controller) *mockBulkRebaser {
	return &mockBulkRebaser{ctrl: ctrl}
}

func (m *mockBulkRebaser) Err() error {
	results := m.ctrl.Call(m, "Err")
	err, _ := results[0].(error)
	return err
}

func (m *mockBulkRebaser) Onto(name string) git.RebaseHandle {
	results := m.ctrl.Call(m, "Onto", name)
	h, _ := results[0].(git.RebaseHandle)
	return h
}

func (m *mockBulkRebaser) EXPECT() _mockBulkRebaserRecorder {
	return _mockBulkRebaserRecorder{m: m, ctrl: m.ctrl}
}

type _mockBulkRebaserRecorder struct {
	m    *mockBulkRebaser
	ctrl *gomock.Controller
}

func (r _mockBulkRebaserRecorder) Err() *gomock.Call {
	return r.ctrl.RecordCall(r.m, "Err")
}

func (r _mockBulkRebaserRecorder) Onto(name interface{}) *gomock.Call {
	return r.ctrl.RecordCall(r.m, "Onto", name)
}

// Matches *github.PullRequestBranch objects with the given branch.
type prBranchMatcher string

var _ gomock.Matcher = prBranchMatcher("")

func (m prBranchMatcher) String() string {
	return fmt.Sprintf("pull request branch %q", string(m))
}

func (m prBranchMatcher) Matches(x interface{}) bool {
	b, ok := x.(*github.PullRequestBranch)
	if !ok {
		return false
	}

	return b.GetRef() == string(m)
}

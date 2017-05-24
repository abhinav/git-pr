package git

import (
	"errors"
	"testing"

	"github.com/abhinav/git-fu/gateway"
	"github.com/abhinav/git-fu/gateway/gatewaytest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBulkRebaser(t *testing.T) {
	type deletion struct {
		Checkout string
		Delete   string
	}

	type rebaseCall struct {
		From string
		To   string

		// Base() and Err() expected on the returned RebaseHandle.
		WantBase string
		WantErr  string
	}

	type ontoCall struct {
		Onto    string
		Rebases []rebaseCall
	}

	tests := []struct {
		Desc         string
		Do           []ontoCall
		SetupGateway func(*gatewaytest.MockGit)

		// ExpectRebases is a convenience option for setting up Rebase
		// requests on the gateway that never fail. This may be omitted or
		// partial if the test has more complex rebase setup in SetupGateway.
		ExpectRebases []*gateway.RebaseRequest

		// ExpectDeletions is a convenience option for setting up
		// Checkout(parent), Delete(branch) in-order without any errors. This
		// may be omitted or partial if the test has a more complex deletion
		// setup in SetupGateway.
		ExpectDeletions []deletion

		WantErrors []string
	}{
		{
			Desc: "single rebase",
			Do: []ontoCall{
				{
					Onto: "master",
					Rebases: []rebaseCall{
						{
							From:     "feature-1",
							To:       "feature-2",
							WantBase: "git-fu/rebase/feature-2",
						},
					},
				},
			},
			ExpectRebases: []*gateway.RebaseRequest{
				{
					Onto:   "master",
					From:   "feature-1",
					Branch: "git-fu/rebase/feature-2",
				},
			},
			ExpectDeletions: []deletion{
				{Checkout: "master", Delete: "git-fu/rebase/feature-2"},
			},
		},
		{
			Desc: "rebase stack",
			Do: []ontoCall{
				{
					Onto: "origin/dev",
					Rebases: []rebaseCall{
						{
							From:     "dev",
							To:       "feature-1",
							WantBase: "git-fu/rebase/feature-1",
						},
						{
							From:     "feature-1",
							To:       "feature-2",
							WantBase: "git-fu/rebase/feature-2",
						},
						{
							From:     "feature-2",
							To:       "feature-3",
							WantBase: "git-fu/rebase/feature-3",
						},
						{
							From:     "feature-3",
							To:       "feature-4",
							WantBase: "git-fu/rebase/feature-4",
						},
					},
				},
			},
			ExpectRebases: []*gateway.RebaseRequest{
				{
					Onto:   "origin/dev",
					From:   "dev",
					Branch: "git-fu/rebase/feature-1",
				},
				{
					Onto:   "git-fu/rebase/feature-1",
					From:   "feature-1",
					Branch: "git-fu/rebase/feature-2",
				},
				{
					Onto:   "git-fu/rebase/feature-2",
					From:   "feature-2",
					Branch: "git-fu/rebase/feature-3",
				},
				{
					Onto:   "git-fu/rebase/feature-3",
					From:   "feature-3",
					Branch: "git-fu/rebase/feature-4",
				},
			},
			ExpectDeletions: []deletion{
				{
					Checkout: "git-fu/rebase/feature-3",
					Delete:   "git-fu/rebase/feature-4",
				},
				{
					Checkout: "git-fu/rebase/feature-2",
					Delete:   "git-fu/rebase/feature-3",
				},
				{
					Checkout: "git-fu/rebase/feature-1",
					Delete:   "git-fu/rebase/feature-2",
				},
				{
					Checkout: "origin/dev",
					Delete:   "git-fu/rebase/feature-1",
				},
			},
		},
		{
			Desc: "rebase failure",
			Do: []ontoCall{
				{
					Onto: "origin/master",
					Rebases: []rebaseCall{
						{
							From:    "master",
							To:      "feature-1",
							WantErr: "great sadness",
						},
						{
							From:    "feature-1",
							To:      "feature-2",
							WantErr: "great sadness",
						},
						{
							From:    "feature-2",
							To:      "feature-3",
							WantErr: "great sadness",
						},
					},
				},
				{
					Onto: "origin/master",
					Rebases: []rebaseCall{
						{
							From:     "feature-3",
							To:       "feature-4",
							WantBase: "git-fu/rebase/feature-4",
						},
					},
				},
			},
			ExpectRebases: []*gateway.RebaseRequest{
				{
					Onto:   "origin/master",
					From:   "feature-3",
					Branch: "git-fu/rebase/feature-4",
				},
			},
			SetupGateway: func(git *gatewaytest.MockGit) {
				git.EXPECT().
					Rebase(&gateway.RebaseRequest{
						Onto:   "origin/master",
						From:   "master",
						Branch: "git-fu/rebase/feature-1",
					}).
					Return(errors.New("great sadness"))
			},
			ExpectDeletions: []deletion{
				{Checkout: "origin/master", Delete: "git-fu/rebase/feature-4"},
				{Checkout: "origin/master", Delete: "git-fu/rebase/feature-1"},
			},
			WantErrors: []string{"great sadness"},
		},
		{
			Desc: "multiple rebase failures",
			Do: []ontoCall{
				{
					Onto: "origin/master",
					Rebases: []rebaseCall{
						{
							From:    "master",
							To:      "feature-1",
							WantErr: "feature 1 failed",
						},
					},
				},
				{
					Onto: "origin/master",
					Rebases: []rebaseCall{
						{
							From:    "feature-1",
							To:      "feature-2",
							WantErr: "feature 2 failed",
						},
					},
				},
				{
					Onto: "origin/master",
					Rebases: []rebaseCall{
						{
							From:    "feature-2",
							To:      "feature-3",
							WantErr: "feature 3 failed",
						},
					},
				},
			},
			SetupGateway: func(git *gatewaytest.MockGit) {
				git.EXPECT().
					Rebase(&gateway.RebaseRequest{
						Onto:   "origin/master",
						From:   "master",
						Branch: "git-fu/rebase/feature-1",
					}).
					Return(errors.New("feature 1 failed"))

				git.EXPECT().
					Rebase(&gateway.RebaseRequest{
						Onto:   "origin/master",
						From:   "feature-1",
						Branch: "git-fu/rebase/feature-2",
					}).
					Return(errors.New("feature 2 failed"))

				git.EXPECT().
					Rebase(&gateway.RebaseRequest{
						Onto:   "origin/master",
						From:   "feature-2",
						Branch: "git-fu/rebase/feature-3",
					}).
					Return(errors.New("feature 3 failed"))
			},
			ExpectDeletions: []deletion{
				{Checkout: "origin/master", Delete: "git-fu/rebase/feature-3"},
				{Checkout: "origin/master", Delete: "git-fu/rebase/feature-2"},
				{Checkout: "origin/master", Delete: "git-fu/rebase/feature-1"},
			},
			WantErrors: []string{
				"feature 1 failed",
				"feature 2 failed",
				"feature 3 failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Desc, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			gw := gatewaytest.NewMockGit(mockCtrl)
			if tt.SetupGateway != nil {
				tt.SetupGateway(gw)
			}

			for _, req := range tt.ExpectRebases {
				gw.EXPECT().Rebase(req).Return(nil)
			}

			var deletions []*gomock.Call
			for _, x := range tt.ExpectDeletions {
				deletions = append(deletions,
					gw.EXPECT().Checkout(x.Checkout).Return(nil),
					gw.EXPECT().DeleteBranch(x.Delete).Return(nil),
				)
			}
			gomock.InOrder(deletions...)

			rebaser := NewBulkRebaser(gw)
			rebaser.checkoutUniqueBranch = checkoutUniqueBranchAlwaysSuccessful

			defer func() {
				assert.NoError(t, rebaser.Cleanup(),
					"cleanup failed")
			}()

			for _, ontoCall := range tt.Do {
				h := rebaser.Onto(ontoCall.Onto)
				for _, rebaseCall := range ontoCall.Rebases {
					h = h.Rebase(rebaseCall.From, rebaseCall.To)
					assert.Equal(t, rebaseCall.WantBase, h.Base())
					if rebaseCall.WantErr != "" {
						err := h.Err()
						if assert.Error(t, err) {
							assert.Contains(t, err.Error(), rebaseCall.WantErr)
						}
					}
				}
			}

			if len(tt.WantErrors) > 0 {
				err := rebaser.Err()
				require.Error(t, err, "expected failure")
				for _, msg := range tt.WantErrors {
					assert.Contains(t, err.Error(), msg)
				}
				return
			}

			require.NoError(t, rebaser.Err(), "expected success")
		})
	}
}

func checkoutUniqueBranchAlwaysSuccessful(
	_ gateway.Git, prefix string, _ string,
) (string, error) {
	return prefix, nil
}

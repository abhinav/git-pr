package git

import (
	"fmt"
	"testing"

	"github.com/abhinav/git-pr/gateway/gatewaytest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckoutUniqueBranch(t *testing.T) {
	tests := []struct {
		Desc             string
		Prefix           string
		ExistingBranches []string

		Want    string
		WantErr []string
	}{
		{
			Desc:   "no conflicts",
			Prefix: "foo/bar",
			Want:   "foo/bar",
		},
		{
			Desc:             "one conflict",
			Prefix:           "foo-bar",
			ExistingBranches: []string{"foo-bar"},
			Want:             "foo-bar/2",
		},
		{
			Desc:             "two conflicts",
			Prefix:           "foo",
			ExistingBranches: []string{"foo", "foo/2"},
			Want:             "foo/3",
		},
		{
			Desc:   "too many attempts",
			Prefix: "bar",
			ExistingBranches: []string{
				"bar", "bar/2", "bar/3", "bar/4", "bar/5", "bar/6", "bar/7",
				"bar/8", "bar/9", "bar/10",
			},
			WantErr: []string{
				`could not find a unique branch name with prefix "bar"`,
				`"someref" may not be a valid git ref`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Desc, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			git := gatewaytest.NewMockGit(mockCtrl)
			for _, br := range tt.ExistingBranches {
				git.EXPECT().CreateBranchAndCheckout(br, "someref").
					Return(fmt.Errorf("branch %q already exists", br))
			}

			if tt.Want != "" {
				git.EXPECT().CreateBranchAndCheckout(tt.Want, "someref").
					Return(nil)
			}

			got, err := CheckoutUniqueBranch(git, tt.Prefix, "someref")
			if len(tt.WantErr) > 0 {
				require.Error(t, err, "expected failure")
				for _, msg := range tt.WantErr {
					assert.Contains(t, err.Error(), msg)
				}
				return
			}

			require.NoError(t, err, "expected success")
			assert.Equal(t, tt.Want, got)
		})
	}
}

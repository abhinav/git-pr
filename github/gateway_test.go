package github

import (
	"context"
	"fmt"
	"testing"

	"github.com/abhinav/git-fu/gateway"
	"github.com/abhinav/git-fu/ptr"

	"github.com/golang/mock/gomock"
	"github.com/google/go-github/github"
	"github.com/stretchr/testify/require"
)

func TestListPullRequestReviews(t *testing.T) {
	tests := []struct {
		give []*github.PullRequestReview
		want []*gateway.PullRequestReview
	}{
		{give: nil, want: []*gateway.PullRequestReview{}},
		{
			give: []*github.PullRequestReview{
				{
					State: ptr.String("APPROVED"),
					User:  &github.User{Login: ptr.String("foo")},
				},
				{
					State: ptr.String("CHANGES_REQUESTED"),
					User:  &github.User{Login: ptr.String("bar")},
				},
			},
			want: []*gateway.PullRequestReview{
				{User: "foo", Status: gateway.PullRequestApproved},
				{User: "bar", Status: gateway.PullRequestChangesRequested},
			},
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			prService := NewMockPullRequestsService(mockCtrl)
			prService.EXPECT().
				ListReviews(gomock.Any(), "foo", "bar", 42).
				Return(tt.give, &github.Response{}, nil)

			gw := Gateway{
				owner: "foo",
				repo:  "bar",
				pulls: prService,
			}

			gotReviews, err := gw.ListPullRequestReviews(context.Background(), 42)
			require.NoError(t, err)
			require.Equal(t, tt.want, gotReviews)
		})
	}
}

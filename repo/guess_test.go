package repo

import (
	"testing"

	"github.com/abhinav/git-fu/gateway/gatewaytest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestGuess(t *testing.T) {
	tests := []struct {
		url string

		want    Repo
		wantErr string
	}{
		{url: "git@github.com:foo/bar", want: Repo{Owner: "foo", Name: "bar"}},
		{url: "https://github.com/baz/qux", want: Repo{Owner: "baz", Name: "qux"}},
		{url: "ssh://git@github.com/abc/def", want: Repo{Owner: "abc", Name: "def"}},
		{url: "/home/foo/bar", wantErr: `remote "origin" (/home/foo/bar) is not a GitHub remote`},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			git := gatewaytest.NewMockGit(mockCtrl)
			git.EXPECT().RemoteURL("origin").Return(tt.url, nil).AnyTimes()

			got, err := Guess(git)
			if tt.wantErr != "" {
				if assert.Error(t, err) {
					assert.Contains(t, err.Error(), tt.wantErr)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, &tt.want, got)
			}
		})
	}
}

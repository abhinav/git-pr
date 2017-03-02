package repo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	tests := []struct {
		give    string
		want    Repo
		wantErr string
	}{
		{
			give: "foo/bar",
			want: Repo{Owner: "foo", Name: "bar"},
		},
		{
			give:    "foobar",
			wantErr: "repository must be in the form owner/repo",
		},
		{
			give:    "/foo",
			wantErr: `owner in repository "/foo" cannot be empty`,
		},
		{
			give:    "foo/",
			wantErr: `name in repository "foo/" cannot be empty`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.give, func(t *testing.T) {
			got, err := Parse(tt.give)
			if tt.wantErr != "" {
				if assert.Error(t, err) {
					assert.Contains(t, err.Error(), tt.wantErr)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, &tt.want, got)
				assert.Equal(t, tt.give, got.String())
			}
		})
	}
}

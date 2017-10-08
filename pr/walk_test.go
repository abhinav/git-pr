package pr_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/abhinav/git-pr/pr"
	"github.com/abhinav/git-pr/pr/prtest"

	"github.com/golang/mock/gomock"
	"github.com/google/go-github/github"
	"github.com/stretchr/testify/assert"
)

type mockChildren struct {
	ctrl *gomock.Controller
}

func newMockChildren(ctrl *gomock.Controller) *mockChildren {
	return &mockChildren{ctrl: ctrl}
}

func (m *mockChildren) Expect(pr interface{}) *gomock.Call {
	return m.ctrl.RecordCall(m, "Call", pr)
}

func (m *mockChildren) Call(pr *github.PullRequest) (out []*github.PullRequest, err error) {
	results := m.ctrl.Call(m, "Call", pr)
	out, _ = results[0].([]*github.PullRequest)
	err, _ = results[1].(error)
	return
}

func TestWalk(t *testing.T) {
	type children map[int][]int

	type visit struct {
		// Whether the children of this node should be visited.
		VisitChildren bool

		// If non-nil, this visit will fail with an error.
		Error error

		// If non-nil, this visit will panic with the given value.
		Panic interface{}
	}

	type visits map[int]visit

	concurrency := []int{0, 1, 2, 4, 8}

	tests := []struct {
		Desc  string
		Pulls []int

		Children children
		Visits   map[int]visit

		WantErr []string
	}{
		{Desc: "empty", Pulls: []int{}},
		{
			Desc:     "single",
			Pulls:    []int{1},
			Children: children{1: {}},
			Visits:   visits{1: {VisitChildren: true}},
		},
		{
			Desc:    "single error",
			Pulls:   []int{1},
			Visits:  visits{1: {Error: errors.New("great sadness")}},
			WantErr: []string{"great sadness"},
		},
		{
			Desc:  "single panic error",
			Pulls: []int{1},
			Visits: visits{
				1: {Panic: errors.New("great sadness")},
			},
			WantErr: []string{"great sadness"},
		},
		{
			Desc:  "single panic",
			Pulls: []int{1},
			Visits: visits{
				1: {Panic: "great sadness"},
			},
			WantErr: []string{"panic: great sadness"},
		},
		{
			Desc:   "single no children",
			Pulls:  []int{1},
			Visits: visits{1: {}},
		},
		{
			Desc:  "single hop",
			Pulls: []int{1, 2, 3},
			Children: children{
				1: {4, 5},
				2: {},
				4: {},
				5: {},
			},
			Visits: visits{
				1: {VisitChildren: true},
				2: {VisitChildren: true},
				3: {},
				4: {VisitChildren: true},
				5: {VisitChildren: true},
			},
		},
		{
			Desc:  "multi hop",
			Pulls: []int{1, 2},
			Children: children{
				1:  {3},
				2:  {4, 5},
				3:  {},
				4:  {6},
				5:  {7},
				6:  {8, 9},
				7:  {},
				8:  {10},
				9:  {},
				10: {11},
				11: {},
			},
			Visits: visits{
				1:  {VisitChildren: true},
				2:  {VisitChildren: true},
				3:  {VisitChildren: true},
				4:  {VisitChildren: true},
				5:  {VisitChildren: true},
				6:  {VisitChildren: true},
				7:  {VisitChildren: true},
				8:  {VisitChildren: true},
				9:  {VisitChildren: true},
				10: {VisitChildren: true},
				11: {VisitChildren: true},
			},
		},
		{
			Desc:  "multi hop errors",
			Pulls: []int{1, 2},
			Children: children{
				1: {3},
				2: {4, 5},
				4: {6},
				5: {7},
				7: {},
			},
			Visits: visits{
				1: {VisitChildren: true},
				2: {VisitChildren: true},
				3: {VisitChildren: true, Error: errors.New("something went wrong")},
				4: {VisitChildren: true},
				5: {VisitChildren: true},
				6: {VisitChildren: true, Error: errors.New("great sadness")},
				7: {VisitChildren: true},
			},
			WantErr: []string{"great sadness", "something went wrong"},
		},
		{
			Desc:  "channel overflow",
			Pulls: []int{1, 21},
			Children: children{
				1: {2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
				2: {}, 3: {}, 4: {}, 5: {}, 6: {}, 7: {}, 8: {}, 9: {}, 10: {},
				11: {}, 12: {}, 13: {}, 14: {}, 15: {}, 16: {}, 17: {}, 18: {}, 19: {}, 20: {},
				21: {22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40},
				22: {}, 23: {}, 24: {}, 25: {}, 26: {}, 27: {}, 28: {}, 29: {}, 30: {},
				31: {}, 32: {}, 33: {}, 34: {}, 35: {}, 36: {}, 37: {}, 38: {}, 39: {}, 40: {},
			},
			Visits: visits{
				1:  {VisitChildren: true},
				2:  {VisitChildren: true},
				3:  {VisitChildren: true},
				4:  {VisitChildren: true},
				5:  {VisitChildren: true},
				6:  {VisitChildren: true},
				7:  {VisitChildren: true},
				8:  {VisitChildren: true},
				9:  {VisitChildren: true},
				10: {VisitChildren: true},
				11: {VisitChildren: true},
				12: {VisitChildren: true},
				13: {VisitChildren: true},
				14: {VisitChildren: true},
				15: {VisitChildren: true},
				16: {VisitChildren: true},
				17: {VisitChildren: true},
				18: {VisitChildren: true},
				19: {VisitChildren: true},
				20: {VisitChildren: true},
				21: {VisitChildren: true},
				22: {VisitChildren: true},
				23: {VisitChildren: true},
				24: {VisitChildren: true},
				25: {VisitChildren: true},
				26: {VisitChildren: true},
				27: {VisitChildren: true},
				28: {VisitChildren: true},
				29: {VisitChildren: true},
				30: {VisitChildren: true},
				31: {VisitChildren: true},
				32: {VisitChildren: true},
				33: {VisitChildren: true},
				34: {VisitChildren: true},
				35: {VisitChildren: true},
				36: {VisitChildren: true},
				37: {VisitChildren: true},
				38: {VisitChildren: true},
				39: {VisitChildren: true},
				40: {VisitChildren: true},
			},
		},
	}

	for _, conc := range concurrency {
		for _, tt := range tests {
			name := fmt.Sprintf("concurrency=%v/%v", conc, tt.Desc)
			t.Run(name, func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				getChildren := newMockChildren(ctrl)
				for parent, children := range tt.Children {
					getChildren.
						Expect(prMatcher{Number: parent}).
						Return(fakePullRequests(children), nil)
				}

				visitor := prtest.NewMockVisitor(ctrl)
				for num, visit := range tt.Visits {
					var v pr.Visitor
					if visit.VisitChildren {
						v = visitor
					}

					call := visitor.EXPECT().Visit(prMatcher{Number: num})
					switch {
					case visit.Error != nil:
						call.Return(v, visit.Error)
					case visit.Panic != nil:
						p := visit.Panic
						call.Do(func(*github.PullRequest) { panic(p) }).Return(v, nil)
					default:
						call.Return(v, nil)
					}
				}

				cfg := pr.WalkConfig{
					Children:    getChildren.Call,
					Concurrency: conc,
				}
				err := pr.Walk(cfg, fakePullRequests(tt.Pulls), visitor)

				if len(tt.WantErr) > 0 {
					if !assert.Error(t, err) {
						return
					}

					for _, msg := range tt.WantErr {
						assert.Contains(t, err.Error(), msg)
					}
					return
				}

				assert.NoError(t, err)
			})
		}
	}
}

type prMatcher struct {
	Number int
}

var _ gomock.Matcher = prMatcher{}

func (m prMatcher) String() string {
	return fmt.Sprintf("pull request #%v", m.Number)
}

func (m prMatcher) Matches(x interface{}) bool {
	pr, ok := x.(*github.PullRequest)
	if !ok {
		return false
	}

	return pr.GetNumber() == m.Number
}

func fakePullRequests(nums []int) []*github.PullRequest {
	prs := make([]*github.PullRequest, len(nums))
	for i, n := range nums {
		prs[i] = &github.PullRequest{Number: github.Int(n)}
	}
	return prs
}

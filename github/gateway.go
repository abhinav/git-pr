package github

import (
	"context"
	"fmt"

	"github.com/abhinav/git-fu/gateway"
	"github.com/abhinav/git-fu/repo"

	"github.com/google/go-github/github"
)

//go:generate mockgen -package github -destination=mocks_test.go github.com/abhinav/git-fu/github GitService,PullRequestsService

// GitService is a subset of the GitHub Git API.
type GitService interface {
	DeleteRef(
		ctx context.Context,
		owner string, repo string, ref string,
	) (*github.Response, error)
}

var _ GitService = (*github.GitService)(nil)

// PullRequestsService is a subset of the GitHub Pull Requests API.
type PullRequestsService interface {
	Edit(
		ctx context.Context,
		owner string, repo string, number int,
		pull *github.PullRequest,
	) (*github.PullRequest, *github.Response, error)

	GetRaw(
		ctx context.Context,
		owner string, repo string, number int, opt github.RawOptions,
	) (string, *github.Response, error)

	List(
		ctx context.Context,
		owner string, repo string, opt *github.PullRequestListOptions,
	) ([]*github.PullRequest, *github.Response, error)

	ListReviews(
		ctx context.Context,
		owner, repo string, number int,
	) ([]*github.PullRequestReview, *github.Response, error)

	Merge(
		ctx context.Context,
		owner string, repo string, number int,
		commitMessage string,
		options *github.PullRequestOptions,
	) (*github.PullRequestMergeResult, *github.Response, error)
}

var _ PullRequestsService = (*github.PullRequestsService)(nil)

// RepositoriesService is a subset of the GitHub Repositories API.
type RepositoriesService interface {
	GetCombinedStatus(ctx context.Context,
		owner, repo, ref string, opt *github.ListOptions,
	) (*github.CombinedStatus, *github.Response, error)
}

var _ RepositoriesService = (*github.RepositoriesService)(nil)

// Gateway is a GitHub gateway that makes actual requests to GitHub.
type Gateway struct {
	owner string
	repo  string

	git   GitService
	pulls PullRequestsService
	repos RepositoriesService
}

var _ gateway.GitHub = (*Gateway)(nil)

// NewGatewayForRepository builds a new GitHub gateway for the given GitHub
// repository.
func NewGatewayForRepository(client *github.Client, repo *repo.Repo) *Gateway {
	return &Gateway{
		owner: repo.Owner,
		repo:  repo.Name,
		pulls: client.PullRequests,
		repos: client.Repositories,
		git:   client.Git,
	}
}

func (g *Gateway) urlFor(number int) string {
	return fmt.Sprintf("https://github.com/%v/%v/pull/%v", g.owner, g.repo, number)
}

// IsOwned checks if this branch is local to this repository.
func (g *Gateway) IsOwned(ctx context.Context, br *github.PullRequestBranch) bool {
	return *br.Repo.Owner.Login == g.owner && *br.Repo.Name == g.repo
}

// ListPullRequestReviews lists reviews for a pull request.
func (g *Gateway) ListPullRequestReviews(ctx context.Context, number int) ([]*gateway.PullRequestReview, error) {
	reviews, _, err := g.pulls.ListReviews(ctx, g.owner, g.repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to list reviews for %v: %v", g.urlFor(number), err)
	}

	result := make([]*gateway.PullRequestReview, len(reviews))
	for i, r := range reviews {
		result[i] = &gateway.PullRequestReview{
			User:   r.User.GetLogin(),
			Status: gateway.PullRequestReviewState(r.GetState()),
		}
	}
	return result, nil
}

// GetBuildStatus gets the build status for the given ref.
func (g *Gateway) GetBuildStatus(ctx context.Context, ref string) (*gateway.BuildStatus, error) {
	s, _, err := g.repos.GetCombinedStatus(ctx, g.owner, g.repo, ref, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get build status for %q: %v", ref, err)
	}

	bs := gateway.BuildStatus{State: gateway.BuildState(s.GetState())}
	for _, status := range s.Statuses {
		bs.Statuses = append(bs.Statuses, &gateway.BuildContextStatus{
			Name:    status.GetContext(),
			Message: status.GetDescription(),
			State:   gateway.BuildState(status.GetState()),
		})
	}

	return &bs, nil
}

// ListPullRequestsByHead lists pull requests with the given head.
func (g *Gateway) ListPullRequestsByHead(ctx context.Context, owner, branch string) ([]*github.PullRequest, error) {
	if owner == "" {
		owner = g.owner
	}
	// TODO: account for pagination
	prs, _, err := g.pulls.List(
		ctx,
		g.owner,
		g.repo,
		&github.PullRequestListOptions{Head: owner + ":" + branch})
	if err != nil {
		err = fmt.Errorf(
			"failed to list pull requests with head %v:%v: %v", owner, branch, err)
	}
	return prs, err
}

// ListPullRequestsByBase lists pull requests made against the given merge base.
func (g *Gateway) ListPullRequestsByBase(ctx context.Context, branch string) ([]*github.PullRequest, error) {
	// TODO: account for pagination
	prs, _, err := g.pulls.List(
		ctx,
		g.owner,
		g.repo,
		&github.PullRequestListOptions{Base: branch})
	if err != nil {
		err = fmt.Errorf(
			"failed to list pull requests with base %v: %v", branch, err)
	}
	return prs, err
}

// GetPullRequestPatch retrieves the raw patch for the given PR. The contents
// of the patch may be applied using the git-am command.
func (g *Gateway) GetPullRequestPatch(ctx context.Context, number int) (string, error) {
	patch, _, err := g.pulls.GetRaw(
		ctx, g.owner, g.repo, number, github.RawOptions{Type: github.Patch})
	if err != nil {
		err = fmt.Errorf("failed to retrieve patch for %v: %v", g.urlFor(number), err)
	}
	return patch, err
}

// SetPullRequestBase changes the merge base for the given PR.
func (g *Gateway) SetPullRequestBase(ctx context.Context, number int, base string) error {
	_, _, err := g.pulls.Edit(ctx, g.owner, g.repo, number,
		&github.PullRequest{Base: &github.PullRequestBranch{Ref: &base}})
	if err != nil {
		return fmt.Errorf(
			"failed to change merge base of %v to %v: %v", g.urlFor(number), base, err)
	}
	return nil
}

// SquashPullRequest merges given pull request. The title and description are
// used as-is for the commit message.
func (g *Gateway) SquashPullRequest(ctx context.Context, pr *github.PullRequest) error {
	result, _, err := g.pulls.Merge(ctx, g.owner, g.repo, *pr.Number, *pr.Body,
		&github.PullRequestOptions{CommitTitle: *pr.Title, MergeMethod: "squash"})
	if err != nil {
		return fmt.Errorf("failed to merge %v: %v", g.urlFor(*pr.Number), err)
	}

	if result.Merged == nil || !*result.Merged {
		return fmt.Errorf("failed to merge %v: %v", g.urlFor(*pr.Number), *result.Message)
	}

	return nil
}

// DeleteBranch deletes the given remote branch.
func (g *Gateway) DeleteBranch(ctx context.Context, name string) error {
	if _, err := g.git.DeleteRef(ctx, g.owner, g.repo, "heads/"+name); err != nil {
		return fmt.Errorf("failed to delete remote branch %v: %v", name, err)
	}
	return nil
}

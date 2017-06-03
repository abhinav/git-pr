package pr

import (
	"container/list"
	"context"
	"fmt"
	"sync"

	"github.com/abhinav/git-fu/gateway"
	"github.com/abhinav/git-fu/git"
	"github.com/abhinav/git-fu/service"

	"github.com/google/go-github/github"
	"go.uber.org/multierr"
)

// Rebase a pull request and its dependencies.
func (s *Service) Rebase(ctx context.Context, req *service.RebaseRequest) (_ *service.RebaseResponse, err error) {
	if len(req.PullRequests) == 0 {
		return &service.RebaseResponse{}, nil
	}

	// Go back to the original branch after everything is done.
	oldBranch, err := s.git.CurrentBranch()
	if err != nil {
		return nil, err
	}
	defer func(oldBranch string) {
		err = multierr.Append(err, s.git.Checkout(oldBranch))
	}(oldBranch)

	if err := s.git.Fetch(&gateway.FetchRequest{Remote: "origin"}); err != nil {
		return nil, err
	}

	// TODO: support remotes besides origin
	baseRef, err := s.git.SHA1("origin/" + req.Base)
	if err != nil {
		return nil, err
	}

	rebaser := git.NewBulkRebaser(s.git)
	defer func() {
		err = multierr.Append(err, rebaser.Cleanup())
	}()

	results, err := s.rebasePullRequests(rebasePRConfig{
		Context:      ctx,
		GitRebaser:   rebaser,
		GitHub:       s.gh,
		Base:         baseRef,
		PullRequests: req.PullRequests,
		Author:       req.Author,
	})
	if err != nil {
		return nil, err
	}

	// Nothing to do
	if len(results) == 0 {
		return &service.RebaseResponse{}, nil
	}

	var (
		// Branches to reset to new positions for their remotes after rebasing.
		branchesToReset []string

		// Branches not updated because their heads were out of date
		branchesNotUpdated []string

		// Pushes to perform. local ref -> remote branch
		pushes = make(map[string]string)
	)

	for _, r := range results {
		prBranch := r.PR.Head.GetRef()
		if sha, err := s.git.SHA1(prBranch); err == nil {
			if sha == r.PR.Head.GetSHA() {
				branchesToReset = append(branchesToReset, prBranch)
			} else {
				branchesNotUpdated = append(branchesNotUpdated, prBranch)
			}
		}
		pushes[r.LocalRef] = prBranch
	}

	if err := s.git.Push(&gateway.PushRequest{
		Remote: "origin",
		Force:  true,
		Refs:   pushes,
	}); err != nil {
		return nil, err
	}

	for _, br := range branchesToReset {
		err = multierr.Append(err, s.git.ResetBranch(br, "origin/"+br))
	}

	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	for _, pr := range req.PullRequests {
		// TODO: --only-mine should apply
		if pr.Base.GetRef() != req.Base {
			wg.Add(1)
			// TODO: fix unbounded goroutine count
			go func(pr *github.PullRequest) {
				defer wg.Done()
				e := s.gh.SetPullRequestBase(ctx, *pr.Number, req.Base)
				if e == nil {
					return
				}

				mu.Lock()
				err = multierr.Append(err, fmt.Errorf(
					"failed to set base for %v to %q: %v", *pr.HTMLURL, req.Base, e))
				mu.Unlock()
			}(pr)
		}
	}
	wg.Wait()

	return &service.RebaseResponse{
		BranchesNotUpdated: branchesNotUpdated,
	}, err
}

type rebasedPullRequest struct {
	PR *github.PullRequest

	// We should do,
	//
	// 	git push origin $LocalRef:$Branch
	//
	// Where $Branch is pr.Head.GetRef()
	LocalRef string
}

// Part of the interface of git.BulkRebaser that we need here.
type bulkRebaser interface {
	Err() error
	Onto(string) git.RebaseHandle
}

type rebasePRConfig struct {
	// If non-empty, only PRs authored by this user will be considered.
	Author string

	Context      context.Context
	GitRebaser   bulkRebaser
	GitHub       gateway.GitHub
	Base         string
	PullRequests []*github.PullRequest
}

func rebasePullRequests(cfg rebasePRConfig) (map[int]rebasedPullRequest, error) {
	v := rebaseVisitor{
		rebasePRConfig: &cfg,
		handle:         cfg.GitRebaser.Onto(cfg.Base),
		mu:             new(sync.Mutex),
		results:        list.New(),
	}

	walkCfg := WalkConfig{Children: getDependentPRs(cfg.Context, cfg.GitHub)}
	if err := Walk(walkCfg, cfg.PullRequests, v); err != nil {
		return nil, err
	}

	if err := cfg.GitRebaser.Err(); err != nil {
		return nil, err
	}

	results := make(map[int]rebasedPullRequest, v.results.Len())
	for e := v.results.Front(); e != nil; e = e.Next() {
		p := e.Value.(rebasedPullRequest)
		results[p.PR.GetNumber()] = p
	}

	return results, nil
}

func getDependentPRs(
	ctx context.Context, gh gateway.GitHub,
) func(*github.PullRequest) ([]*github.PullRequest, error) {
	return func(pr *github.PullRequest) ([]*github.PullRequest, error) {
		return gh.ListPullRequestsByBase(ctx, pr.Head.GetRef())
	}
}

type rebaseVisitor struct {
	*rebasePRConfig

	mu      *sync.Mutex
	results *list.List // list<rebasedPullRequest>

	handle git.RebaseHandle
}

func (v rebaseVisitor) Visit(pr *github.PullRequest) (Visitor, error) {
	// Don't rebase if we don't own the PR.
	if !v.GitHub.IsOwned(v.Context, pr.Head) {
		// TODO: There is more nuance to this. We should check if we have
		// write access instead.
		// TODO: Log if we skip
		return nil, nil
	}

	if v.Author != "" && pr.User.GetLogin() != v.Author {
		// TODO: log skipped PR
		return nil, nil
	}

	h := v.handle.Rebase(pr.Base.GetSHA(), pr.Head.GetSHA())
	v.mu.Lock()
	v.results.PushBack(rebasedPullRequest{PR: pr, LocalRef: h.Base()})
	v.mu.Unlock()

	// We are operating on a shallow copy of v so we can just modify and
	// return it.
	v.handle = h
	return v, nil
}

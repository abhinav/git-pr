package pr

import (
	"fmt"
	"sync"

	"github.com/abhinav/git-fu/gateway"
	"github.com/abhinav/git-fu/internal"
	"github.com/google/go-github/github"
)

// RebaseRequest is a request to rebase the given list of pull requests and
// their dependencies onto the given base branch.
//
// If the base branch for the given PRs on GitHub is not the same as Base,
// this will be updated too.
type RebaseRequest struct {
	PullRequests []*github.PullRequest
	Base         string
}

// Rebase a pull request and its dependencies.
func (s *Service) Rebase(req *RebaseRequest) (err error) {
	if err := s.Git.Fetch(&gateway.FetchRequest{Remote: "origin"}); err != nil {
		return err
	}

	baseRef, err := s.Git.SHA1("origin/" + req.Base)
	if err != nil {
		return err
	}

	result, err := dryRebase(s, baseRef, req.PullRequests)
	if err != nil {
		return err
	}
	defer func() {
		for _, r := range result {
			err = internal.MultiError(err, s.Git.DeleteBranch(r.LocalBranch))
		}
	}()

	pushRefs := make(map[string]string)
	for _, r := range result {
		pushRefs[r.LocalBranch] = *r.PullRequest.Head.Ref
	}

	if err := s.Git.PushMany(&gateway.PushManyRequest{
		Remote: "origin",
		Force:  true,
		Refs:   pushRefs,
	}); err != nil {
		return err
	}

	var (
		mu     sync.Mutex
		wg     sync.WaitGroup
		errors []error
	)
	for _, pr := range req.PullRequests {
		if *pr.Base.Ref != req.Base {
			wg.Add(1)
			go func(pr *github.PullRequest) {
				defer wg.Done()
				err := s.GitHub.SetPullRequestBase(*pr.Number, req.Base)
				if err == nil {
					return
				}

				mu.Lock()
				errors = append(errors, fmt.Errorf(
					"failed to set base for %v to %q", *pr.HTMLURL, req.Base))
				mu.Unlock()
			}(pr)
		}
	}
	wg.Wait()

	return internal.MultiError(errors...)
}

type rebasedPullRequest struct {
	PullRequest *github.PullRequest
	LocalBranch string
}

// Do all rebasing locally without pushing anything. It is the caller's
// responsibility to delete the temporary local branches in result list.
func dryRebase(s *Service, baseRef string, prs []*github.PullRequest) (_ []rebasedPullRequest, err error) {
	baseBranch := uniqueBranchName(s.Git, "base-"+baseRef)
	if err := s.Git.CreateBranch(baseBranch, baseRef); err != nil {
		return nil, fmt.Errorf("failed to create temporary branch: %v", err)
	}
	// Can't rely on branchesCreated because this should always be cleaned up
	defer func() { err = internal.MultiError(err, s.Git.DeleteBranch(baseBranch)) }()

	// Rebase changes the current branch so we should restore it after we are
	// done.
	oldBranch, err := s.Git.CurrentBranch()
	if err != nil {
		return nil, err
	}
	defer func() { err = internal.MultiError(err, s.Git.Checkout(oldBranch)) }()

	var (
		// List of temporary branches created locally. If we fail with an error,
		// we will be sure to delete all of these.
		branchesCreated []string
		errors          []error
		result          []rebasedPullRequest
	)
	defer func() {
		if err == nil {
			return
		}

		// The operation failed for some reason. Clean up whatever we have
		// created so far.
		for _, br := range branchesCreated {
			err = internal.MultiError(err, s.Git.DeleteBranch(br))
		}
	}()

	for _, pr := range prs {
		// We don't own this branch so we can't rebase it.
		if !s.GitHub.IsOwned(pr.Head) {
			// TODO: log or record which PRs are skipped
			continue
		}

		prBranch := uniqueBranchName(s.Git, fmt.Sprintf("rebase-%v", *pr.Number))
		if err := s.Git.CreateBranch(prBranch, *pr.Head.SHA); err != nil {
			errors = append(errors, fmt.Errorf(
				"could not find head %v for PR %v: %v", *pr.Head.SHA, *pr.HTMLURL, err))
			continue
		}
		branchesCreated = append(branchesCreated, prBranch)

		if err := s.Git.Rebase(&gateway.RebaseRequest{
			Onto:   baseBranch,
			From:   *pr.Base.SHA,
			Branch: prBranch,
		}); err != nil {
			errors = append(errors, fmt.Errorf(
				"failed to rebase PR %v: %v", *pr.HTMLURL, err))
			continue
		}
		result = append(result, rebasedPullRequest{PullRequest: pr, LocalBranch: prBranch})

		dependents, err := s.GitHub.ListPullRequestsByBase(*pr.Head.Ref)
		if err != nil {
			errors = append(errors, fmt.Errorf(
				"could not get dependents of %v: %v", *pr.HTMLURL, err))
			continue
		}

		depResult, err := dryRebase(s, prBranch, dependents)
		if err != nil {
			errors = append(errors, fmt.Errorf(
				"could not rebase dependents of %v: %v", *pr.HTMLURL, err))
			continue
		}
		result = append(result, depResult...)
	}

	return result, internal.MultiError(errors...)
}

func uniqueBranchName(git gateway.Git, template string) string {
	name := template
	for i := 1; git.DoesBranchExist(name); i++ {
		name = fmt.Sprintf("%v-%v", template, i)
	}
	return name
}

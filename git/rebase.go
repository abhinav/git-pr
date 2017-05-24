package git

import (
	"container/list"
	"fmt"
	"sync"

	"github.com/abhinav/git-fu/gateway"

	"go.uber.org/multierr"
)

// BulkRebaser rebases multiple interdependent branches in a safe way. No
// changes are made to existing branches. Callers can commit changes by
// retrieving information from RebaseHandles.
//
// 	r := NewBulkRebaser(g)
// 	defer r.Cleanup()
// 	h := r.Onto("origin/master").Rebase("master", "myfeature")
// 	if err := r.Err(); err != nil {
// 		return err
// 	}
// 	g.ResetBranch("myfeature", h.Base())
type BulkRebaser struct {
	git gateway.Git

	errorsMu sync.Mutex
	errors   []error

	tempBranchesMu sync.Mutex
	tempBranches   *list.List // list<temporaryBranch>

	// Hidden option to change how we find unique branches. Changed only
	// during testing.
	checkoutUniqueBranch func(gateway.Git, string, string) (string, error)
}

// NewBulkRebaser builds a new Bulk Rebaser.
func NewBulkRebaser(g gateway.Git) *BulkRebaser {
	return &BulkRebaser{
		git:                  g,
		tempBranches:         list.New(),
		checkoutUniqueBranch: CheckoutUniqueBranch,
	}
}

// Err returns a non-nil value if any of the operations on BulkRebaser failed.
//
// Failures encountered during Cleanup are not recorded here.
func (br *BulkRebaser) Err() error {
	return multierr.Combine(br.errors...)
}

func (br *BulkRebaser) recordError(err error) {
	br.errorsMu.Lock()
	br.errors = append(br.errors, err)
	br.errorsMu.Unlock()
}

type temporaryBranch struct {
	Name   string // Name of the branch
	Parent string // Previous branch
	// Invariant: If branch $Name exists, $Parent MUST exist.
}

func (br *BulkRebaser) checkoutTemporaryBranch(parent, ref string) (string, error) {
	name, err := br.checkoutUniqueBranch(br.git, fmt.Sprintf("git-fu/rebase/%v", ref), ref)
	if err != nil {
		br.recordError(err)
		return name, err
	}

	br.tempBranchesMu.Lock()
	br.tempBranches.PushBack(temporaryBranch{
		Name:   name,
		Parent: parent,
	})
	br.tempBranchesMu.Unlock()
	return name, nil
}

// Cleanup deletes temporary branches created by the rebaser. The BulkRebaser
// ceases to be valid after this function has been called. No other operations
// must be made on the BulkRebaser after this function has been called.
func (br *BulkRebaser) Cleanup() (err error) {
	br.tempBranchesMu.Lock()
	defer br.tempBranchesMu.Unlock()

	for br.tempBranches.Len() > 0 {
		b := br.tempBranches.Remove(br.tempBranches.Back()).(temporaryBranch)
		e := br.git.Checkout(b.Parent)
		if e == nil {
			e = br.git.DeleteBranch(b.Name)
		}
		err = multierr.Append(err, e)
	}
	return
}

// RebaseHandle is an ongoing rebase, allowing chaining on more rebase
// requests.
type RebaseHandle interface {
	// Error, if any, encountered by rebase operations executed in the stack
	// of branches behind this handle.
	Err() error

	// Base on which this handle will rebase items.
	//
	// Empty if a prior operation failed.
	Base() string

	// Rebase requests that the given range of commits be rebased onto the
	// base of this RebaseHandle.
	//
	// A new RebaseHandle is returned whose base is the rebased position of
	// toRef.
	//
	// 	h := rebaser.Onto("dev").Rebase("master", "feature1")
	//
	// h.Base() may now be used to reference the rebased position of feature1,
	// possibly moving it to that position.
	//
	// Any Rebase calls onto the returned RebaseHandle will be against this
	// new base. This allows for rebasing branhes that depend on previously
	// rebased branches.
	//
	// For example, the following rebases the range of commits
	// master..feature1 onto dev and the range feature1..feature2 onto
	// feature1 after it has been rebased.
	//
	// 	rebaser.Onto("dev").
	// 		Rebase("master", "feature1").
	// 		Rebase("feature1", "feature2")
	//
	// This function MUST NOT be called after Cleanup.
	Rebase(fromRef, toRef string) RebaseHandle
}

// Onto starts a new rebase onto the given base. Rebase calls on the returned
// object will be onto the given ref as base.
//
// For example, the following,
//
// 	rebaser.Onto("master").Rebase("oldfeature", "newfeature")
//
// Is roughly equivalent to,
//
// 	git rebase --onto master oldfeature newfeature
//
// This function MUST NOT be called after Cleanup.
func (br *BulkRebaser) Onto(ref string) RebaseHandle {
	return rebaseHandle{br: br, base: ref}
}

type rebaseHandle struct {
	err error // if non-nil, everything else may be empty

	br   *BulkRebaser
	base string
}

func (h rebaseHandle) Err() error   { return h.err }
func (h rebaseHandle) Base() string { return h.base }

func (h rebaseHandle) Rebase(fromRef, toRef string) RebaseHandle {
	if h.err != nil {
		return h
	}
	br := h.br

	branch, err := br.checkoutTemporaryBranch(h.base, toRef)
	if err != nil {
		return rebaseHandle{err: err}
	}

	req := gateway.RebaseRequest{Onto: h.base, From: fromRef, Branch: branch}
	if err := br.git.Rebase(&req); err != nil {
		br.recordError(err)
		return rebaseHandle{err: err}
	}

	return rebaseHandle{br: h.br, base: branch}
}

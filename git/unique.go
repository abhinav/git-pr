package git

import (
	"fmt"

	"github.com/abhinav/git-fu/gateway"
)

const _uniqeBranchAttempts = 10

// CheckoutUniqueBranch atomically finds a unique branch name and checks it
// out at the given ref.
//
// The final branch name is returned.
func CheckoutUniqueBranch(git gateway.Git, prefix, ref string) (name string, err error) {
	name = prefix
	for i := 0; i < _uniqeBranchAttempts; i++ {
		err = git.CreateBranchAndCheckout(name, ref)
		if err == nil {
			return
		}
		name = fmt.Sprintf("%v/%v", prefix, i+2) // start numbering at 2
	}

	// Nothing found in 10 attempts.
	err = fmt.Errorf(
		"could not find a unique branch name with prefix %q after 10 attempts; "+
			"%q may not be a valid git ref: %v", prefix, ref, err)
	return
}

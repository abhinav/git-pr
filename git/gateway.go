package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/abhinav/git-pr/gateway"

	"go.uber.org/multierr"
)

// Gateway is a git gateway.
type Gateway struct {
	mu sync.RWMutex

	dir string
}

var _ gateway.Git = (*Gateway)(nil)

// NewGateway builds a new Git gateway.
func NewGateway(startDir string) (*Gateway, error) {
	if startDir == "" {
		dir, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf(
				"failed to determine current working directory: %v", err)
		}
		startDir = dir
	} else {
		dir, err := filepath.Abs(startDir)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to determine absolute path of %v: %v", startDir, err)
		}
		startDir = dir
	}

	dir := startDir
	for {
		_, err := os.Stat(filepath.Join(dir, ".git"))
		if err == nil {
			break
		}
		newDir := filepath.Dir(dir)
		if dir == newDir {
			return nil, fmt.Errorf(
				"could not find git repository at %v", startDir)
		}
		dir = newDir
	}

	return &Gateway{dir: dir}, nil
}

// CurrentBranch determines the current branch name.
func (g *Gateway) CurrentBranch() (string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	out, err := g.output("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("could not determine current branch: %v", err)
	}
	return strings.TrimSpace(out), nil
}

// DoesBranchExist checks if this branch exists locally.
func (g *Gateway) DoesBranchExist(name string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	err := g.cmd("show-ref", "--verify", "--quiet", "refs/heads/"+name).Run()
	return err == nil
}

// CreateBranchAndCheckout creates a branch with the given name and head and
// switches to it.
func (g *Gateway) CreateBranchAndCheckout(name, head string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.cmd("checkout", "-b", name, head).Run(); err != nil {
		return fmt.Errorf(
			"failed to create and checkout branch %q at ref %q: %v", name, head, err)
	}
	return nil
}

// CreateBranch creates a branch with the given name and head but does not
// check it out.
func (g *Gateway) CreateBranch(name, head string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.cmd("branch", name, head).Run(); err != nil {
		return fmt.Errorf("failed to create branch %q at ref %q: %v", name, head, err)
	}
	return nil
}

// SHA1 gets the SHA1 hash for the given ref.
func (g *Gateway) SHA1(ref string) (string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	out, err := g.output("rev-parse", "--verify", "-q", ref)
	if err != nil {
		return "", fmt.Errorf("could not resolve ref %q: %v", ref, err)
	}
	return strings.TrimSpace(out), nil
}

// DeleteBranch deletes the given branch.
func (g *Gateway) DeleteBranch(name string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.cmd("branch", "-D", name).Run(); err != nil {
		return fmt.Errorf("failed to delete branch %q: %v", name, err)
	}
	return nil
}

// DeleteRemoteTrackingBranch deletes the remote tracking branch with the
// given name.
func (g *Gateway) DeleteRemoteTrackingBranch(remote, name string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.cmd("branch", "-dr", remote+"/"+name).Run(); err != nil {
		return fmt.Errorf("failed to delete remote tracking branch %q: %v", name, err)
	}
	return nil
}

// Checkout checks the given branch out.
func (g *Gateway) Checkout(name string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.cmd("checkout", name).Run(); err != nil {
		err = fmt.Errorf("failed to checkout branch %q: %v", name, err)
	}
	return nil
}

// Fetch a git ref
func (g *Gateway) Fetch(req *gateway.FetchRequest) error {
	ref := req.RemoteRef
	if req.LocalRef != "" {
		ref = ref + ":" + req.LocalRef
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.cmd("fetch", req.Remote, ref).Run(); err != nil {
		return fmt.Errorf("failed to fetch %q from %q: %v", ref, req.Remote, err)
	}
	return nil
}

// Push pushes refs to a remote.
func (g *Gateway) Push(req *gateway.PushRequest) error {
	if len(req.Refs) == 0 {
		return nil
	}

	args := append(make([]string, 0, len(req.Refs)+2), "push")
	if req.Force {
		args = append(args, "-f")
	}
	args = append(args, req.Remote)

	for ref, remote := range req.Refs {
		if remote != "" {
			ref = ref + ":" + remote
		}
		args = append(args, ref)
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.cmd(args...).Run(); err != nil {
		return fmt.Errorf("failed to push refs to %q: %v", req.Remote, err)
	}
	return nil
}

// Pull pulls the given branch.
func (g *Gateway) Pull(remote, name string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.cmd("pull", remote, name).Run(); err != nil {
		return fmt.Errorf("failed to pull %q from %q: %v", name, remote, err)
	}
	return nil
}

// Rebase a branch.
func (g *Gateway) Rebase(req *gateway.RebaseRequest) error {
	var _args [5]string

	args := append(_args[:0], "rebase")
	if req.Onto != "" {
		args = append(args, "--onto", req.Onto)
	}
	if req.From != "" {
		args = append(args, req.From)
	}
	args = append(args, req.Branch)

	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.cmd(args...).Run(); err != nil {
		return multierr.Append(
			fmt.Errorf("failed to rebase %q: %v", req.Branch, err),
			// If this failed, abort the rebase so that we're not left in a
			// bad state.
			g.cmd("rebase", "--abort").Run(),
		)
	}
	return nil
}

// ResetBranch resets the given branch to the given head.
func (g *Gateway) ResetBranch(branch, head string) error {
	curr, err := g.CurrentBranch()
	if err != nil {
		return fmt.Errorf("could not reset %q to %q: %v", branch, head, err)
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	if curr == branch {
		err = g.cmd("reset", "--hard", head).Run()
	} else {
		err = g.cmd("branch", "-f", branch, head).Run()
	}

	if err != nil {
		err = fmt.Errorf("could not reset %q to %q: %v", branch, head, err)
	}
	return err
}

// RemoteURL gets the URL for the given remote.
func (g *Gateway) RemoteURL(name string) (string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	out, err := g.output("remote", "get-url", name)
	if err != nil {
		return "", fmt.Errorf("failed to get URL for remote %q: %v", name, err)
	}
	return strings.TrimSpace(out), nil
}

// run the given git command.
func (g *Gateway) cmd(args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.dir
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd
}

func (g *Gateway) output(args ...string) (string, error) {
	var stdout bytes.Buffer
	cmd := g.cmd(args...)
	cmd.Stdout = &stdout
	err := cmd.Run()
	return stdout.String(), err
}

package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/abhinav/git-fu/gateway"
)

// Gateway is a git gateway.
type Gateway struct {
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
	out, err := g.output("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("could not determine current branch: %v", err)
	}
	return strings.TrimSpace(out), nil
}

// DoesBranchExist checks if this branch exists locally.
func (g *Gateway) DoesBranchExist(name string) bool {
	err := g.cmd("show-ref", "--verify", "--quiet", "refs/heads/"+name).Run()
	return err == nil
}

// CreateBranch creates
func (g *Gateway) CreateBranch(name, head string) error {
	if err := g.cmd("branch", name, head).Run(); err != nil {
		return fmt.Errorf("failed to create branch %q at ref %q: %v", name, head, err)
	}
	return nil
}

// CreateBranchAndSwitch checks out a new branch with the given name and head.
func (g *Gateway) CreateBranchAndSwitch(name, head string) error {
	if err := g.cmd("checkout", "-b", name, head).Run(); err != nil {
		return fmt.Errorf("failed to create branch %q at ref %q: %v", name, head, err)
	}
	return nil
}

// SHA1 gets the SHA1 hash for the given ref.
func (g *Gateway) SHA1(ref string) (string, error) {
	out, err := g.output("rev-parse", ref)
	if err != nil {
		return "", fmt.Errorf("could not resolve ref %q: %v", ref, err)
	}
	return strings.TrimSpace(out), nil
}

// DeleteBranch deletes the given branch.
func (g *Gateway) DeleteBranch(name string) error {
	if err := g.cmd("branch", "-D", name).Run(); err != nil {
		return fmt.Errorf("failed to delete branch %q: %v", name, err)
	}
	return nil
}

// DeleteRemoteTrackingBranch deletes the remote tracking branch with the
// given name.
func (g *Gateway) DeleteRemoteTrackingBranch(remote, name string) error {
	if err := g.cmd("branch", "-dr", remote+"/"+name).Run(); err != nil {
		return fmt.Errorf("failed to delete remote tracking branch %q: %v", name, err)
	}
	return nil
}

// Checkout checks the given branch out.
func (g *Gateway) Checkout(name string) error {
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

	if err := g.cmd("fetch", req.Remote, ref).Run(); err != nil {
		return fmt.Errorf("failed to fetch %q from %q: %v", ref, req.Remote, err)
	}
	return nil
}

// Push a branch
func (g *Gateway) Push(req *gateway.PushRequest) error {
	ref := req.LocalRef
	if req.RemoteRef != "" {
		ref = ref + ":" + req.RemoteRef
	}

	args := []string{"push"}
	if req.Force {
		args = append(args, "-f")
	}
	args = append(args, req.Remote, ref)

	if err := g.cmd(args...).Run(); err != nil {
		return fmt.Errorf("failed to push %q to %q: %v", ref, req.Remote, err)
	}
	return nil
}

// Pull pulls the given branch.
func (g *Gateway) Pull(remote, name string) error {
	if err := g.cmd("pull", remote, name).Run(); err != nil {
		return fmt.Errorf("failed to pull %q from %q: %v", name, remote, err)
	}
	return nil
}

// ApplyPatches applies the given patch.
func (g *Gateway) ApplyPatches(patches string) error {
	cmd := g.cmd("am")
	cmd.Stdin = strings.NewReader(patches)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to apply patches: %v", err)
	}
	return nil
}

// RemoteURL gets the URL for the given remote.
func (g *Gateway) RemoteURL(name string) (string, error) {
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
package pr

import (
	"container/list"
	"fmt"
	"runtime"
	"sync"

	"go.uber.org/multierr"

	"github.com/google/go-github/github"
)

// Visitor defines what to do at each pull request during a walk.
type Visitor interface {
	// Visits the given pull request and returns a new visitor to visit its
	// children.
	//
	// If a visitor was not returned, the children of this PR will not be
	// visited.
	//
	// This function MAY be called concurrently. Implementations MUST be
	// thread-safe.
	Visit(*github.PullRequest) (Visitor, error)
}

//go:generate mockgen -package=prtest -destination=prtest/mocks.go github.com/abhinav/git-fu/pr Visitor

// WalkConfig configures a pull request traversal.
type WalkConfig struct {
	// Maximum number of pull requests to visit at the same time.
	//
	// Defaults to the number of CPUs available to this process.
	Concurrency int

	// Children retrieves the children of the given pull request.
	//
	// The definition of what constitutes a child of a PR is left up to the
	// implementation.
	//
	// This function MAY be called concurrently. Implementations MUST be
	// thread-safe.
	Children func(*github.PullRequest) ([]*github.PullRequest, error)
}

// Walk traverses a pull request tree by visiting the given pull requests and
// their children in an unspecified order. The only ordering guarantee is that
// parents are visited before their children.
//
// Errors encountered while visiting pull requests are collatted and presented
// as one.
func Walk(cfg WalkConfig, pulls []*github.PullRequest, v Visitor) error {
	if cfg.Children == nil {
		panic("WalkConfig.Children must be set")
	}

	if cfg.Concurrency <= 0 {
		cfg.Concurrency = runtime.NumCPU()
	}

	w := walker{
		// TODO: Magic number. Should make this customizable or leave it the
		// same as Concurrency.
		tasks:    make(chan task, 8),
		children: cfg.Children,
	}

	w.ongoing.Add(len(pulls))
	go func() {
		// If pulls contains more than 8 items, we don't want to block on
		// filling tasks just yet.
		for _, pr := range pulls {
			w.tasks <- task{PR: pr, Visitor: v}
		}
	}()

	for i := 0; i < cfg.Concurrency; i++ {
		go w.Worker()
	}
	w.ongoing.Wait()
	close(w.tasks)

	return multierr.Combine(w.errors...)
}

// Request to visit a single pull request with a specific visitor.
type task struct {
	PR      *github.PullRequest
	Visitor Visitor
}

type walker struct {
	// Incoming tasks. Any worker can handle these.
	tasks chan task

	// Number of ongoing tasks.
	ongoing sync.WaitGroup

	children func(*github.PullRequest) ([]*github.PullRequest, error)

	// Errors encountered while processing.
	errorsMu sync.Mutex
	errors   []error
}

func (w *walker) Worker() {
	// Walker-local buffer for incoming tasks that should be pushed into
	// w.tasks when it's empty.
	taskBuffer := list.New()

worker:
	for {
	fill:
		// Exhaust as much of the buffer as we can.
		for taskBuffer.Len() > 0 {
			e := taskBuffer.Front()
			select {
			case w.tasks <- e.Value.(task):
				taskBuffer.Remove(e)
			default:
				// No more room in channel.
				break fill
			}
		}

		t, ok := <-w.tasks
		if !ok {
			// Channel closed. We're done.
			break worker
		}

		newTasks, err := w.visit(t)
		if err != nil {
			w.errorsMu.Lock()
			w.errors = append(w.errors, err)
			w.errorsMu.Unlock()
		} else if len(newTasks) > 0 {
			for _, task := range newTasks {
				taskBuffer.PushBack(task)
			}
		}
		w.ongoing.Add(len(newTasks) - 1)
	}
}

func (w *walker) visit(t task) (_ []task, err error) {
	defer func() {
		if x := recover(); x != nil {
			if e, ok := x.(error); ok {
				err = e
			} else {
				// TODO: log the panic
				err = fmt.Errorf("panic: %v", x)
			}
		}
	}()

	v, err := t.Visitor.Visit(t.PR)
	if err != nil {
		return nil, err
	}

	if v == nil {
		return nil, nil
	}

	children, err := w.children(t.PR)
	if err != nil {
		return nil, err
	}

	tasks := make([]task, len(children))
	for i, pr := range children {
		tasks[i] = task{PR: pr, Visitor: v}
	}
	return tasks, nil
}

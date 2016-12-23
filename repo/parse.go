package repo

import (
	"errors"
	"fmt"
	"strings"

	"github.com/abhinav/git-fu/entity"
)

// Parse parses a repository name in the format 'owner/repo'.
func Parse(value string) (*entity.Repo, error) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		return nil, errors.New("repository must be in the form owner/repo")
	}

	owner := parts[0]
	if owner == "" {
		return nil, fmt.Errorf("owner in repository %q cannot be empty", value)
	}

	name := parts[1]
	if name == "" {
		return nil, fmt.Errorf("name in repository %q cannot be empty", value)
	}

	fmt.Printf("Using repo %v:%v\n", owner, name)
	return &entity.Repo{Owner: owner, Name: name}, nil
}

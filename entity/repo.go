package entity

import "fmt"

// Repo uniquely identifies a GitHub repository.
type Repo struct {
	Owner string
	Name  string
}

func (r *Repo) String() string {
	if r.Owner == "" && r.Name == "" {
		return ""
	}
	return fmt.Sprintf("%v/%v", r.Owner, r.Name)
}

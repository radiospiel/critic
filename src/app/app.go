package app

import (
	"git.15b.it/eno/critic/src/git"
	"git.15b.it/eno/critic/simple-go/must"
	"github.com/samber/lo"
)

// GetDefaultBases returns the default base points based on git state
func GetDefaultBases() []string {
	candidates := []string{
		"main", "master", "origin/" + must.Must2(git.GetCurrentBranch()), "HEAD",
	}

	return lo.Filter(candidates, func(ref string, _ int) bool {
		return git.HasRef(ref)
	})
}

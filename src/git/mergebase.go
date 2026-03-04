package git

import (
	"strings"

	"github.com/samber/lo"
)

// GetCurrentBranch returns the name of the current branch
func GetCurrentBranch() string {
	output := git("rev-parse", "--abbrev-ref", "HEAD")
	return strings.TrimSpace(string(output))
}

// IsGitRepo checks if the current directory is inside a git repository
func IsGitRepo() bool {
	_, err := tryGit("rev-parse", "--git-dir")
	return err == nil
}

// LocalBranchesOnPath returns all local branch names that decorate commits
// in the range ancestorRef..HEAD, ordered by git log output (newest first).
// The ancestorRef itself and HEAD are excluded from the result.
//
// Uses git log --decorate-refs to efficiently discover branches on the
// ancestry path without manual sorting.
func LocalBranchesOnPath(ancestorRef string) []string {
	if ancestorRef == "" {
		return nil
	}

	// git log --decorate-refs=refs/heads/ --format=%D outputs branch decorations
	// for each commit in the range, in reverse chronological order.
	output, err := tryGit("log", "--decorate-refs=refs/heads/", "--format=%D", ancestorRef+"..HEAD")
	if err != nil {
		return nil
	}

	headBranch := GetCurrentBranch()

	// Parse decorations: each line may contain comma-separated branch refs
	// like "HEAD -> main, feature-branch" or just "feature-branch" or be empty.
	seen := make(map[string]bool)
	var result []string

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		refs := lo.Map(
			strings.Split(line, ","),
			func(s string, _ int) string { return strings.TrimSpace(s) },
		)

		for _, ref := range refs {
			// Strip "HEAD -> " prefix from decorated refs
			ref = strings.TrimPrefix(ref, "HEAD -> ")

			if ref == "" || ref == "HEAD" {
				continue
			}

			// Skip the current branch (HEAD) and the ancestor ref
			if ref == headBranch || ref == ancestorRef {
				continue
			}

			if !seen[ref] {
				seen[ref] = true
				result = append(result, ref)
			}
		}
	}

	// Reverse so oldest (furthest from HEAD) comes first, matching git log order
	lo.Reverse(result)

	return result
}

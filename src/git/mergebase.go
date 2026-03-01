package git

import (
	"math"
	"sort"
	"strconv"
	"strings"
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

// graphDistance returns the number of commits between ref and HEAD
// (i.e., how many commits HEAD is ahead of ref). A larger number means
// the ref is further back in the graph (older). Returns MaxInt if the
// ref is not an ancestor of HEAD or cannot be resolved.
func graphDistance(ref string) int {
	output, err := tryGit("rev-list", "--count", ref+"..HEAD")
	if err != nil {
		return math.MaxInt
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return math.MaxInt
	}
	return n
}

// SortRefsByGraphOrder sorts refs by their topological distance from HEAD,
// oldest (furthest from HEAD) first. Refs that cannot be resolved are
// placed at the end.
func SortRefsByGraphOrder(refs []string) {
	// Pre-fetch distances so each ref is resolved once.
	distances := make(map[string]int, len(refs))
	for _, ref := range refs {
		distances[ref] = graphDistance(ref)
	}

	sort.SliceStable(refs, func(i, j int) bool {
		// Larger distance = further from HEAD = should come first (oldest).
		return distances[refs[i]] > distances[refs[j]]
	})
}

// LocalBranchesOnPath returns all local branch names whose tips are ancestors
// of HEAD and descendants of (or equal to) ancestorRef, sorted by graph
// distance from HEAD (oldest first). The ancestorRef itself is excluded from
// the result if it matches a local branch. HEAD is also excluded since it's
// always implicitly present.
func LocalBranchesOnPath(ancestorRef string) []string {
	if ancestorRef == "" {
		return nil
	}

	// Get all local branches merged into HEAD (tips reachable from HEAD).
	output, err := tryGit("branch", "--merged", "HEAD", "--format=%(refname:short)")
	if err != nil {
		return nil
	}

	ancestorSHA := ResolveRef(ancestorRef)
	headSHA := ResolveRef("HEAD")

	var result []string
	for _, branch := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		branch = strings.TrimSpace(branch)
		if branch == "" {
			continue
		}

		branchSHA := ResolveRef(branch)

		// Skip if the branch tip equals the ancestor (that's our starting point).
		if branchSHA == ancestorSHA {
			continue
		}

		// Skip HEAD — it's always implicitly included.
		if branchSHA == headSHA {
			continue
		}

		// Branch tip must be a descendant of ancestorRef.
		_, err := tryGit("merge-base", "--is-ancestor", ancestorRef, branch)
		if err != nil {
			continue
		}

		result = append(result, branch)
	}

	SortRefsByGraphOrder(result)
	return result
}

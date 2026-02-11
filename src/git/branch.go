package git

import (
	"math"
	"strconv"
	"strings"
)

// AncestryCache caches merge-base --is-ancestor results for the duration of a
// batch of calls (e.g. one GetConversations request). Create with NewAncestryCache().
type AncestryCache map[[2]string]bool

func NewAncestryCache() AncestryCache {
	return make(AncestryCache)
}

func (c AncestryCache) isAncestor(sha, ref string) bool {
	key := [2]string{sha, ref}
	if v, ok := c[key]; ok {
		return v
	}
	_, err := tryGit("merge-base", "--is-ancestor", sha, ref)
	result := err == nil
	c[key] = result
	return result
}

// ClosestBranchForSHA returns the branch name from the given list of refs
// that is closest to (at or following) the given commit SHA.
// "Closest" means the ref whose resolved commit has the fewest commits
// between it and the SHA (i.e., the SHA is an ancestor of the ref with
// the shortest path). Returns empty string if no branch contains the SHA.
func ClosestBranchForSHA(sha string, refs []string) string {
	return closestBranchForSHA(sha, refs, nil)
}

// ClosestBranchForSHACached is like ClosestBranchForSHA but uses a shared
// AncestryCache to avoid redundant git merge-base calls.
func ClosestBranchForSHACached(sha string, refs []string, cache AncestryCache) string {
	return closestBranchForSHA(sha, refs, cache)
}

func closestBranchForSHA(sha string, refs []string, cache AncestryCache) string {
	if sha == "" || len(refs) == 0 {
		return ""
	}

	bestRef := ""
	bestDistance := math.MaxInt64

	for _, ref := range refs {
		if !HasRef(ref) {
			continue
		}

		// Check if sha is an ancestor of (or equal to) this ref
		var isAnc bool
		if cache != nil {
			isAnc = cache.isAncestor(sha, ref)
		} else {
			_, err := tryGit("merge-base", "--is-ancestor", sha, ref)
			isAnc = err == nil
		}
		if !isAnc {
			continue
		}

		// Count commits between sha and ref
		output, err := tryGit("rev-list", "--count", sha+".."+ref)
		if err != nil {
			continue
		}

		count, err := strconv.Atoi(strings.TrimSpace(string(output)))
		if err != nil {
			continue
		}

		if count < bestDistance {
			bestDistance = count
			bestRef = ref
		}
	}

	return bestRef
}

// IsCommitInRange checks whether a commit SHA is in the range start..end.
// This means the commit is reachable from end but not from start
// (i.e., it was introduced after start and before/at end).
// If start is empty, all ancestors of end are included.
// If end is empty, HEAD is used.
func IsCommitInRange(sha, start, end string) bool {
	return isCommitInRange(sha, start, end, nil)
}

// IsCommitInRangeCached is like IsCommitInRange but uses a shared
// AncestryCache to avoid redundant git merge-base calls.
func IsCommitInRangeCached(sha, start, end string, cache AncestryCache) bool {
	return isCommitInRange(sha, start, end, cache)
}

func isCommitInRange(sha, start, end string, cache AncestryCache) bool {
	if sha == "" {
		return false
	}

	isWorkingDir := end == ""
	if isWorkingDir {
		end = "HEAD"
	}

	// Check that sha is an ancestor of (or equal to) end
	var isAncEnd bool
	if cache != nil {
		isAncEnd = cache.isAncestor(sha, end)
	} else {
		_, err := tryGit("merge-base", "--is-ancestor", sha, end)
		isAncEnd = err == nil
	}
	if !isAncEnd {
		return false
	}

	// If no start, everything reachable from end is in range
	if start == "" {
		return true
	}

	// Check that sha is NOT an ancestor of start (meaning it came after start)
	var isAncStart bool
	if cache != nil {
		isAncStart = cache.isAncestor(sha, start)
	} else {
		_, err := tryGit("merge-base", "--is-ancestor", sha, start)
		isAncStart = err == nil
	}
	if !isAncStart {
		// sha is NOT an ancestor of start -> it's in the range
		return true
	}

	// sha IS an ancestor of start (or equal) -> check if it equals start
	startSHA := ResolveRef(start)
	resolvedSHA := sha
	if len(sha) < 40 {
		// It might be a short SHA, resolve it
		output, err := tryGit("rev-parse", "--verify", sha)
		if err == nil {
			resolvedSHA = strings.TrimSpace(string(output))
		}
	}

	// When end is working dir, include the start commit itself — comments
	// created at HEAD belong to the working tree even when HEAD == start.
	if isWorkingDir {
		return true
	}

	// If sha equals the start commit, it's not in the range (exclusive of start)
	return resolvedSHA != startSHA
}

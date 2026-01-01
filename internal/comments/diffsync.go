package comments

import (
	"git.15b.it/eno/critic/pkg/types"
)

// SyncComments synchronizes comments when the original file changes
// It applies the diff between oldContent and newContent to the critic file
func SyncComments(criticFile *types.CriticFile, oldContent, newContent []string) (*types.CriticFile, error) {
	// Calculate the diff using a simple line-based diff algorithm
	operations := calculateLineDiff(oldContent, newContent)

	// Create a new critic file with updated content and adjusted comment positions
	newCriticFile := &types.CriticFile{
		FilePath:      criticFile.FilePath,
		OriginalLines: make([]string, len(newContent)),
		Comments:      make(map[int]*types.CriticBlock),
	}
	copy(newCriticFile.OriginalLines, newContent)

	// Apply operations and adjust comment positions
	lineMapping := make(map[int]int) // Maps old line numbers to new line numbers
	newLineNum := 0

	for oldLineNum := 0; oldLineNum < len(oldContent); oldLineNum++ {
		op := operations[oldLineNum]

		switch op.Type {
		case OpKeep:
			// Line is unchanged, map old position to new position
			lineMapping[oldLineNum] = newLineNum
			newLineNum++

		case OpDelete:
			// Line was deleted, comments at this line are lost
			// We could optionally preserve them at the next available line
			// For now, we'll drop them as the line no longer exists

		case OpReplace:
			// Line was modified, keep the comment at the new position
			lineMapping[oldLineNum] = newLineNum
			newLineNum++
		}
	}

	// Remap comments to their new positions
	for oldLine, comment := range criticFile.Comments {
		if newLine, exists := lineMapping[oldLine]; exists {
			newCriticFile.Comments[newLine] = &types.CriticBlock{
				LineNumber: newLine,
				Lines:      comment.Lines,
			}
		}
		// If the line doesn't exist in the mapping, the comment is dropped
		// (the line was deleted)
	}

	return newCriticFile, nil
}

// DiffOpType represents the type of diff operation
type DiffOpType int

const (
	OpKeep DiffOpType = iota
	OpDelete
	OpInsert
	OpReplace
)

// DiffOp represents a single diff operation
type DiffOp struct {
	Type    DiffOpType
	OldLine string
	NewLine string
}

// calculateLineDiff calculates a simple line-based diff between old and new content
func calculateLineDiff(oldLines, newLines []string) []DiffOp {
	ops := make([]DiffOp, len(oldLines))

	// Build a map of new lines for quick lookup
	newLineMap := make(map[string][]int)
	for i, line := range newLines {
		newLineMap[line] = append(newLineMap[line], i)
	}

	// Track which new lines have been matched
	matched := make([]bool, len(newLines))

	// First pass: identify unchanged lines
	for i, oldLine := range oldLines {
		if indices, exists := newLineMap[oldLine]; exists {
			// Find the first unmatched occurrence
			for _, newIdx := range indices {
				if !matched[newIdx] {
					ops[i] = DiffOp{Type: OpKeep, OldLine: oldLine, NewLine: newLines[newIdx]}
					matched[newIdx] = true
					break
				}
			}
		}

		// If no match found, mark as delete (will be refined in second pass)
		if ops[i].Type == 0 {
			ops[i] = DiffOp{Type: OpDelete, OldLine: oldLine}
		}
	}

	// Second pass: identify replacements vs pure deletes
	// This is a simplified approach; a full diff would use LCS or Myers algorithm
	newIdx := 0
	for i := 0; i < len(ops); i++ {
		if ops[i].Type == OpDelete {
			// Check if there's an unmatched new line at a similar position
			for newIdx < len(newLines) && matched[newIdx] {
				newIdx++
			}
			if newIdx < len(newLines) {
				// Treat this as a replacement
				ops[i] = DiffOp{Type: OpReplace, OldLine: ops[i].OldLine, NewLine: newLines[newIdx]}
				matched[newIdx] = true
				newIdx++
			}
		}
	}

	return ops
}

// ValidateSync validates that the synchronized critic file is consistent
func ValidateSync(criticFile *types.CriticFile, expectedContent []string) error {
	return ValidateCriticFile(criticFile, expectedContent)
}

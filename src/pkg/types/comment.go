package types

// CriticBlock represents a single critic comment block
type CriticBlock struct {
	// LineNumber is the line number in the original file where this comment appears
	LineNumber int
	// Lines contains the comment content (without the CRITIC fence markers)
	Lines []string
	// UUID is the unique identifier for this comment (linked to messagedb)
	UUID string
}

// CriticFile represents a .critic.md file with embedded comments
type CriticFile struct {
	// FilePath is the path to the original file being commented on
	FilePath string
	// OriginalLines are the lines from the original file
	OriginalLines []string
	// Comments maps line numbers to critic blocks
	Comments map[int]*CriticBlock
}

// CommentUpdate represents a change to comments for a file
type CommentUpdate struct {
	FilePath string
	Comments map[int]*CriticBlock
}

package git

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	sio "git.15b.it/eno/critic/simple-go/io"
	"git.15b.it/eno/critic/simple-go/must"
	"git.15b.it/eno/critic/simple-go/preconditions"
	"git.15b.it/eno/critic/simple-go/utils"
	"github.com/samber/lo"
)

type ld_mode int

const (
	ld_add ld_mode = iota
	ld_del
	ld_move
	ld_move_add // unpaired: destination of a move (has firstNewLine)
	ld_move_del // unpaired: source of a move (has firstOldLine)
)

type LineDisplacementBlock struct {
	mode           ld_mode
	numOfLInes     int
	firstOldLine   int
	firstNewLine   int
	txtHash        string
	txtWithContext string
}

type LineDisplacement struct {
	blocks   []LineDisplacementBlock
	ref1SHA1 string // SHA1 of the "old" commit
	ref2SHA1 string // SHA1 of the "new" commit
}

func (m ld_mode) String() string {
	switch m {
	case ld_add:
		return "add"
	case ld_del:
		return "del"
	case ld_move:
		return "move"
	case ld_move_add:
		return "move_add"
	case ld_move_del:
		return "move_del"
	default:
		return fmt.Sprintf("unknown(%d)", m)
	}
}

func (b LineDisplacementBlock) String() string {
	return fmt.Sprintf("{%s lines=%d old=%d new=%d}", b.mode, b.numOfLInes, b.firstOldLine, b.firstNewLine)
}

func (ld LineDisplacement) String() string {
	var buf strings.Builder
	buf.WriteString("LineDisplacement{\n")
	for _, block := range ld.blocks {
		buf.WriteString("  ")
		buf.WriteString(block.String())
		buf.WriteString("\n")
	}
	buf.WriteString("}")
	return buf.String()
}

/*

Note on diff parsing:

A diff looks like below. We ignore the file header (everything until the
line starting with "@@ "). (Parts of) this section would repeat for each file, but we only
diff between individual files.

We parse hunk headers (@@ ... @@), and then each hunk may contain multiple blocks that are
either added or removed. We use additional color information to discriminate further between
added to/removed from the file or moved wihin the file.

	diff --git a/data.txt b/data.txt
	index c4352f8..e8c5b4a 100644
	--- a/data.txt
	+++ b/data.txt
	@@ -1,10 +1,7 @@
	 line 1
	 line 2
	 line 3
	-line 4
	-line 5
	-line 6
	-line 7
	+linie fünf
	 line 8
	 line 9
	 line 10
*/

type gitColor struct {
	name string
	pre  []byte // prefix, including the + or - sign
	post []byte
}

func xp(s string) []byte {
	return lo.Map(strings.Split(s, " "), func(digit string, _ int) byte {
		return byte(must.ParseInt(digit, 16))
	})
}

// On git color names
//
// - git help --config |grep color.diff shows the colors that are used by git diff
// - man git-config has git color names. Also see https://stackoverflow.com/questions/15458237/git-pretty-format-colors
var colors = map[string]gitColor{
	"new":                 {name: "green", pre: xp("1B 5B 33 32 6D 2B 1B 5B 6D 1B 5B 33 32 6D"), post: xp("1B 5B 6D")},
	"newMoved":            {name: "yellow", pre: xp("1B 5B 33 33 6D 2B 1B 5B 6D 1B 5B 33 33 6D"), post: xp("1B 5B 6D")},
	"newMovedAlternative": {name: "blue", pre: xp("1B 5B 33 34 6D 2B 1B 5B 6D 1B 5B 33 34 6D"), post: xp("1B 5B 6D")},
	"old":                 {name: "red", pre: xp("1B 5B 33 31 6D 2D"), post: xp("1B 5B 6D")},
	"oldMoved":            {name: "yellow reverse", pre: xp("1B 5B 37 3B 33 33 6D 2D"), post: xp("1B 5B 6D")},
	"oldMovedAlternative": {name: "blue reverse", pre: xp("1B 5B 37 3B 33 34 6D 2D"), post: xp("1B 5B 6D")},
}

func gitColorConfig(colorRole string) string {
	var color = colors[colorRole]
	return "color.diff." + colorRole + "=" + color.name
}

func HasPrefix(buf []byte, prefix []byte) bool {
	if len(buf)-len(prefix) < 0 {
		return false
	}

	return bytes.Compare(buf[0:len(prefix)], prefix) == 0
}

func HasSuffix(buf []byte, suffix []byte) bool {
	if len(buf)-len(suffix) < 0 {
		return false
	}

	return bytes.Compare(buf[len(buf)-len(suffix):], suffix) == 0
}

// matchColor checks if a line matches any known git diff color and returns the color key and text content
func matchColor(line []byte) (colorKey string, textContent []byte) {
	for key, color := range colors {
		if HasPrefix(line, color.pre) && HasSuffix(line, color.post) {
			return key, line[len(color.pre) : len(line)-len(color.post)]
		}
	}
	return "", nil
}

// computeHash calculates SHA1 hash of combined text
func computeHash(text []byte) string {
	h := sha1.New()
	h.Write(text)
	return hex.EncodeToString(h.Sum(nil))
}

// colorToBlock converts a git diff color name to a LineDisplacementBlock.
// Returns nil if the color doesn't map to a block type.
func colorToBlock(colorName string, lineCount, oldLine, newLine int, textHash string) *LineDisplacementBlock {
	switch colorName {
	case "new":
		return &LineDisplacementBlock{
			mode:         ld_add,
			numOfLInes:   lineCount,
			firstOldLine: 0,
			firstNewLine: newLine,
			txtHash:      "",
		}
	case "old":
		return &LineDisplacementBlock{
			mode:         ld_del,
			numOfLInes:   lineCount,
			firstOldLine: oldLine,
			firstNewLine: 0,
			txtHash:      "",
		}
	case "newMoved", "newMovedAlternative":
		return &LineDisplacementBlock{
			mode:         ld_move_add,
			numOfLInes:   lineCount,
			firstOldLine: 0,
			firstNewLine: newLine,
			txtHash:      textHash,
		}
	case "oldMoved", "oldMovedAlternative":
		return &LineDisplacementBlock{
			mode:         ld_move_del,
			numOfLInes:   lineCount,
			firstOldLine: oldLine,
			firstNewLine: 0,
			txtHash:      textHash,
		}
	default:
		return nil
	}
}

// getColoredDiff calls git diff with color configuration for move detection
func getColoredDiff(path string, ref1 string, ref2 string) []byte {
	return must.Exec("git",
		"-c", gitColorConfig("new"),
		"-c", gitColorConfig("newMoved"),
		"-c", gitColorConfig("newMovedAlternative"),
		"-c", gitColorConfig("old"),
		"-c", gitColorConfig("oldMoved"),
		"-c", gitColorConfig("oldMovedAlternative"),
		"-c", "color.diff.frag=normal",
		"diff",
		"--color-moved=zebra", "--color=always",
		ref1, ref2, "--", path)
}

// parseDiffToBlocks parses colored diff output into raw blocks (unpaired moves)
func parseDiffToBlocks(output []byte) []LineDisplacementBlock {
	lines := bytes.Split(output, []byte{'\n'})

	var currentHunk *hunkHeader
	var blocks []LineDisplacementBlock

	// Track current line positions within the diff
	currentOldLine := 0
	currentNewLine := 0

	// Track current block state
	previousMatchingColor := ""
	blockStartOldLine := 0
	blockStartNewLine := 0
	blockLineCount := 0
	var blockText bytes.Buffer

	flushCurrentBlock := func() {
		defer func() {
			previousMatchingColor = ""
			blockLineCount = 0
			blockText.Reset()
		}()

		if previousMatchingColor == "" || blockLineCount == 0 {
			return
		}

		block := colorToBlock(previousMatchingColor, blockLineCount, blockStartOldLine, blockStartNewLine, computeHash(blockText.Bytes()))
		if block != nil {
			blocks = append(blocks, *block)
		}
	}

	for _, line := range lines {
		parsedHunk := parseHunkHeader(line)
		if parsedHunk != nil {
			flushCurrentBlock()
			currentHunk = parsedHunk
			currentOldLine = currentHunk.OldStart
			currentNewLine = currentHunk.NewStart
			continue
		}

		if currentHunk == nil {
			continue
		}

		matchingColor, textContent := matchColor(line)

		// Handle context lines (no color match)
		if matchingColor == "" {
			flushCurrentBlock()
			// Context lines exist in both old and new
			if len(line) > 0 && line[0] == ' ' {
				currentOldLine++
				currentNewLine++
			}
			continue
		}

		// Color changed - flush previous block and start new one
		if matchingColor != previousMatchingColor {
			flushCurrentBlock()
			previousMatchingColor = matchingColor

			// Record the start positions for the new block
			if strings.HasPrefix(matchingColor, "new") {
				blockStartNewLine = currentNewLine
			} else if strings.HasPrefix(matchingColor, "old") {
				blockStartOldLine = currentOldLine
			}
		}

		// Process the line
		blockLineCount++

		// For moved blocks, accumulate text for hash calculation
		if strings.Contains(matchingColor, "Moved") {
			blockText.Write(textContent)
			blockText.WriteByte('\n')
		}

		// Update line counters based on whether this is an add or delete line
		if strings.HasPrefix(matchingColor, "new") {
			currentNewLine++
		} else if strings.HasPrefix(matchingColor, "old") {
			currentOldLine++
		}
	}

	// Flush any remaining block
	flushCurrentBlock()

	return blocks
}

// processBlocks pairs move blocks, sorts, and returns processed blocks
func processBlocks(blocks []LineDisplacementBlock) []LineDisplacementBlock {
	// Separate move blocks from other blocks
	moveBlocks, allBlocks := utils.Partition(blocks, func(b LineDisplacementBlock) bool {
		return b.mode == ld_move_add || b.mode == ld_move_del
	})

	// Group move blocks by hash
	groupedByHash := lo.GroupBy(moveBlocks, func(b LineDisplacementBlock) string {
		return b.txtHash
	})

	// Pair up move blocks
	for hash, blocks := range groupedByHash {
		addBlocks, delBlocks := utils.Partition(blocks, func(b LineDisplacementBlock) bool {
			return b.mode == ld_move_add
		})
		preconditions.Check(len(addBlocks) == len(delBlocks),
			"mismatched move blocks for hash %s: %d add, %d del", hash, len(addBlocks), len(delBlocks))

		for _, pair := range lo.Zip2(addBlocks, delBlocks) {
			allBlocks = append(allBlocks, LineDisplacementBlock{
				mode:         ld_move,
				numOfLInes:   pair.A.numOfLInes,
				firstOldLine: pair.B.firstOldLine,
				firstNewLine: pair.A.firstNewLine,
				txtHash:      hash,
			})
		}
	}

	// Sort blocks by line position then mode
	allBlocks = utils.SortBy(allBlocks, func(b LineDisplacementBlock) int {
		line := utils.IfElse(b.firstOldLine != 0, b.firstOldLine, b.firstNewLine)
		return line*10 + int(b.mode)
	})

	// Final sanity check: only ld_add, ld_del, and ld_move should remain
	for _, block := range allBlocks {
		preconditions.Check(block.mode == ld_add || block.mode == ld_del || block.mode == ld_move,
			"unexpected block mode %d after pairing", block.mode)
	}

	return allBlocks
}

// BuildLineDisplacement builds a LineDisplacement for translating line numbers between refs
func BuildLineDisplacement(path string, ref1 string, ref2 string) (LineDisplacement, error) {
	// Resolve refs to SHA1s
	ref1SHA1 := revParse(ref1)
	ref2SHA1 := revParse(ref2)

	output := getColoredDiff(path, ref1, ref2)
	blocks := parseDiffToBlocks(output)
	blocks = processBlocks(blocks)
	blocks = populateBlockContent(blocks, path, ref1SHA1)

	return LineDisplacement{
		blocks:   blocks,
		ref1SHA1: ref1SHA1,
		ref2SHA1: ref2SHA1,
	}, nil
}

// Translate takes a line number from ref1 and returns the corresponding line number in ref2.
// Returns an error if the line was deleted or cannot be mapped.
func (ld *LineDisplacement) Translate(lineNo int) (int, error) {
	displacement := 0

	for _, block := range ld.blocks {
		switch block.mode {
		case ld_add:
			// Addition: if the addition happened before our line, we need to account for it
			// Additions shift all subsequent lines in the new file
			if block.firstNewLine <= lineNo+displacement {
				displacement += block.numOfLInes
			}

		case ld_del:
			// Deletion: check if our line was deleted
			if lineNo >= block.firstOldLine && lineNo < block.firstOldLine+block.numOfLInes {
				return 0, errors.New("line was deleted")
			}
			// If deletion happened before our line, it shifts our line up
			if block.firstOldLine < lineNo {
				displacement -= block.numOfLInes
			}

		case ld_move:
			// Check if our line is in the moved range
			if lineNo >= block.firstOldLine && lineNo < block.firstOldLine+block.numOfLInes {
				offsetInBlock := lineNo - block.firstOldLine
				return block.firstNewLine + offsetInBlock, nil
			}
			// Move acts like delete+add for non-moved lines
			// If source was before our line, it shifts our line up
			if block.firstOldLine < lineNo {
				displacement -= block.numOfLInes
			}
			// If destination is before our new position, it shifts our line down
			if block.firstNewLine <= lineNo+displacement {
				displacement += block.numOfLInes
			}
		}
	}

	return lineNo + displacement, nil
}

// hunkHeader represents a single hunk from a unified diff
type hunkHeader struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
}

var hunkHeaderPattern = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)
var sha1Pattern = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

func parseHunkHeader(lineBytes []byte) *hunkHeader {
	line := string(lineBytes)
	matches := hunkHeaderPattern.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}
	header := hunkHeader{}

	// Parse old start and count
	header.OldStart, _ = strconv.Atoi(matches[1])
	if matches[2] != "" {
		header.OldCount, _ = strconv.Atoi(matches[2])
	} else {
		header.OldCount = 1
	}

	// Parse new start and count
	header.NewStart, _ = strconv.Atoi(matches[3])
	if matches[4] != "" {
		header.NewCount, _ = strconv.Atoi(matches[4])
	} else {
		header.NewCount = 1
	}
	return &header
}

// contextLines is the number of context lines to include before and after a block
const contextLines = 3

// readLinesFromGitShow reads a range of lines from a file at a specific git ref.
// It reads from startLine to endLine (inclusive) using git show.
func readLinesFromGitShow(path string, ref string, startLine int, endLine int) string {
	if startLine < 1 {
		startLine = 1
	}

	// Use SectionPipe to extract only the relevant lines from git show output
	// skip = startLine - 1, take = endLine - startLine + 1
	skip := startLine - 1
	take := endLine - startLine + 1
	pipe := sio.NewSectionPipe(skip, take)
	output := must.PipeInto(pipe, "git", "show", ref+":"+path)
	return string(output)
}

// populateBlockContent reads the content for each block from the git repository
// and stores it in the block's txtWithContext field. For blocks with firstOldLine set,
// it reads from (firstOldLine - contextLines) to (firstOldLine + numOfLines + contextLines).
func populateBlockContent(blocks []LineDisplacementBlock, path string, ref string) []LineDisplacementBlock {
	for i := range blocks {
		block := &blocks[i]
		if block.firstOldLine == 0 {
			continue
		}

		startLine := block.firstOldLine - contextLines
		endLine := block.firstOldLine + block.numOfLInes + contextLines

		block.txtWithContext = readLinesFromGitShow(path, ref, startLine, endLine)
	}
	return blocks
}

// GetLineContext returns the context around a specific line in a file.
// It reads contextLines lines before and after the specified line from the given git ref.
// If ref is empty, it reads from the working directory using the filesystem.
func GetLineContext(path string, lineNum int, ref string) string {
	startLine := lineNum - contextLines
	endLine := lineNum + contextLines

	if ref == "" {
		// Read from working directory
		return readLinesFromWorkingDir(path, startLine, endLine)
	}
	return readLinesFromGitShow(path, ref, startLine, endLine)
}

// readLinesFromWorkingDir reads a range of lines from a file in the working directory.
// It reads from startLine to endLine (inclusive).
func readLinesFromWorkingDir(path string, startLine int, endLine int) string {
	// Use ReadFileLines to extract only the relevant lines from the file
	output, err := sio.ReadFileLines(path, startLine, endLine)
	if err != nil {
		panic(fmt.Sprintf("readLinesFromWorkingDir(%s, %d, %d): %v", path, startLine, endLine, err))
	}
	return output
}

// revParse converts a git ref (branch name, tag, commit SHA1) to a full SHA1 commit hash.
// If the input is already a SHA1, it returns it as-is.
func revParse(ref string) string {
	// Check if it's already a 40-character hex string (SHA1)
	if len(ref) == 40 && sha1Pattern.MatchString(ref) {
		return ref
	}

	// Use git rev-parse to get the SHA1
	output := must.Exec("git", "rev-parse", ref)
	return strings.TrimSpace(string(output))
}

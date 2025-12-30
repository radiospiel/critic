Bug: Deleted lines showing wrong content in "Last Commit" mode

When viewing diffs in "Last Commit" mode, deleted lines were displaying content from the
new version of the file instead of the old version. For example, when a comment was deleted,
the diff would show the function declaration below it as the deleted content.

Example: Deleting lines 11-12 (comment + blank) should show:
  11 - // compareDiff compares actual and expected diffs using JSON serialization
  12 -
  11   func TestParseDiff_Empty(t *testing.T) {
  12       actual, err := ParseDiff("")

But instead showed:
  11 - func TestParseDiff_Empty(t *testing.T) {
  12 -     actual, err := ParseDiff("")
  11   func TestParseDiff_Empty(t *testing.T) {
  12       actual, err := ParseDiff("")

----------------------------------------------------------------------------------------------------

Root cause: internal/ui/diffview.go:445 was using HEAD to get old file content for syntax
highlighting. HEAD is only the old version for unstaged diffs. In "Last Commit" mode, HEAD
is the NEW version (HEAD~1 is old), causing deleted lines to show new file content.

Fix: Always use hunk-based reconstruction for old version syntax highlighting, which correctly
rebuilds the old file from diff hunks regardless of diff mode (unstaged, last commit, or
merge-base).

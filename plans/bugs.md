I see this diff on internal/git/parser_test.go:

M internal/git/diff.go              │  11 - func TestParseDiff_Empty(t *testing.T) {
M internal/git/mergebase.go         │  12 -     actual, err := ParseDiff("")
M internal/git/parser.go            │  11   func TestParseDiff_Empty(t *testing.T) {
M internal/git/parser_test.go       │  12       actual, err := ParseDiff("")
+ internal/git/path_test.go         │  13       if err != nil {
M internal/git/watcher.go           │

but this is wrong.

Correct is this:

diff --git a/internal/git/parser_test.go b/internal/git/parser_test.go
index db5c608..8c84ef5 100644
--- a/internal/git/parser_test.go
+++ b/internal/git/parser_test.go
@@ -8,8 +8,6 @@ import (
        "git.15b.it/eno/critic/internal/assert"
 )

-// compareDiff compares actual and expected diffs using JSON serialization
-
 func TestParseDiff_Empty(t *testing.T) {
        actual, err := ParseDiff("")

package git

import (
	"testing"

	ctypes "github.com/radiospiel/critic/src/pkg/types"

	"github.com/radiospiel/critic/simple-go/assert"
)

func TestParseDiff_Empty(t *testing.T) {
	actual, err := ParseDiff("")
	assert.NoError(t, err)

	expected := &ctypes.Diff{
		Files: []*ctypes.FileDiff{},
	}

	assert.CompareJSON(t, actual, expected)
}

func TestParseDiff_SingleFile(t *testing.T) {
	input := `diff --git a/file.go b/file.go
--- a/file.go
+++ b/file.go
@@ -1,3 +1,3 @@ func main() {
 package main
-import "fmt"
+import "log"
 func main() {`

	actual, err := ParseDiff(input)
	assert.NoError(t, err)

	expected := &ctypes.Diff{
		Files: []*ctypes.FileDiff{
			{
				OldPath: "file.go",
				NewPath: "file.go",
				Hunks: []*ctypes.Hunk{
					{
						OldStart: 1,
						OldLines: 3,
						NewStart: 1,
						NewLines: 3,
						Header:   "func main() {",
						Lines: []*ctypes.Line{
							{Type: ctypes.LineContext, Content: "package main", OldNum: 1, NewNum: 1},
							{Type: ctypes.LineDeleted, Content: `import "fmt"`, OldNum: 2, NewNum: 0},
							{Type: ctypes.LineAdded, Content: `import "log"`, OldNum: 0, NewNum: 2},
							{Type: ctypes.LineContext, Content: "func main() {", OldNum: 3, NewNum: 3},
						},
						Stats: ctypes.HunkStats{Added: 1, Deleted: 1},
					},
				},
			},
		},
	}

	assert.CompareJSON(t, actual, expected)
}

func TestParseDiff_MultipleFiles(t *testing.T) {
	input := `diff --git a/file1.go b/file1.go
--- a/file1.go
+++ b/file1.go
@@ -1,1 +1,1 @@
-old line
+new line
diff --git a/file2.go b/file2.go
--- a/file2.go
+++ b/file2.go
@@ -1,1 +1,1 @@
-another old
+another new`

	actual, err := ParseDiff(input)
	assert.NoError(t, err)

	expected := &ctypes.Diff{
		Files: []*ctypes.FileDiff{
			{
				OldPath: "file1.go",
				NewPath: "file1.go",
				Hunks: []*ctypes.Hunk{
					{
						OldStart: 1,
						OldLines: 1,
						NewStart: 1,
						NewLines: 1,
						Lines: []*ctypes.Line{
							{Type: ctypes.LineDeleted, Content: "old line", OldNum: 1, NewNum: 0},
							{Type: ctypes.LineAdded, Content: "new line", OldNum: 0, NewNum: 1},
						},
						Stats: ctypes.HunkStats{Added: 1, Deleted: 1},
					},
				},
			},
			{
				OldPath: "file2.go",
				NewPath: "file2.go",
				Hunks: []*ctypes.Hunk{
					{
						OldStart: 1,
						OldLines: 1,
						NewStart: 1,
						NewLines: 1,
						Lines: []*ctypes.Line{
							{Type: ctypes.LineDeleted, Content: "another old", OldNum: 1, NewNum: 0},
							{Type: ctypes.LineAdded, Content: "another new", OldNum: 0, NewNum: 1},
						},
						Stats: ctypes.HunkStats{Added: 1, Deleted: 1},
					},
				},
			},
		},
	}

	assert.CompareJSON(t, actual, expected)
}

func TestParseDiff_MultipleHunks(t *testing.T) {
	input := `diff --git a/file.go b/file.go
--- a/file.go
+++ b/file.go
@@ -1,2 +1,2 @@ first hunk
-line 1
+line one
 line 2
@@ -10,2 +10,2 @@ second hunk
 line 10
-line 11
+line eleven`

	actual, err := ParseDiff(input)
	assert.NoError(t, err)

	expected := &ctypes.Diff{
		Files: []*ctypes.FileDiff{
			{
				OldPath: "file.go",
				NewPath: "file.go",
				Hunks: []*ctypes.Hunk{
					{
						OldStart: 1,
						OldLines: 2,
						NewStart: 1,
						NewLines: 2,
						Header:   "first hunk",
						Lines: []*ctypes.Line{
							{Type: ctypes.LineDeleted, Content: "line 1", OldNum: 1, NewNum: 0},
							{Type: ctypes.LineAdded, Content: "line one", OldNum: 0, NewNum: 1},
							{Type: ctypes.LineContext, Content: "line 2", OldNum: 2, NewNum: 2},
						},
						Stats: ctypes.HunkStats{Added: 1, Deleted: 1},
					},
					{
						OldStart: 10,
						OldLines: 2,
						NewStart: 10,
						NewLines: 2,
						Header:   "second hunk",
						Lines: []*ctypes.Line{
							{Type: ctypes.LineContext, Content: "line 10", OldNum: 10, NewNum: 10},
							{Type: ctypes.LineDeleted, Content: "line 11", OldNum: 11, NewNum: 0},
							{Type: ctypes.LineAdded, Content: "line eleven", OldNum: 0, NewNum: 11},
						},
						Stats: ctypes.HunkStats{Added: 1, Deleted: 1},
					},
				},
			},
		},
	}

	assert.CompareJSON(t, actual, expected)
}

func TestParseDiff_NewFile(t *testing.T) {
	input := `diff --git a/newfile.go b/newfile.go
new file mode 100644
--- /dev/null
+++ b/newfile.go
@@ -0,0 +1,3 @@
+package main
+
+func main() {}`

	actual, err := ParseDiff(input)
	assert.NoError(t, err)

	expected := &ctypes.Diff{
		Files: []*ctypes.FileDiff{
			{
				OldPath: "newfile.go",
				NewPath: "newfile.go",
				NewMode: "100644",
				IsNew:   true,
				Hunks: []*ctypes.Hunk{
					{
						OldStart: 0,
						OldLines: 0,
						NewStart: 1,
						NewLines: 3,
						Lines: []*ctypes.Line{
							{Type: ctypes.LineAdded, Content: "package main", OldNum: 0, NewNum: 1},
							{Type: ctypes.LineAdded, Content: "", OldNum: 0, NewNum: 2},
							{Type: ctypes.LineAdded, Content: "func main() {}", OldNum: 0, NewNum: 3},
						},
						Stats: ctypes.HunkStats{Added: 3, Deleted: 0},
					},
				},
			},
		},
	}

	assert.CompareJSON(t, actual, expected)
}

func TestParseDiff_DeletedFile(t *testing.T) {
	input := `diff --git a/oldfile.go b/oldfile.go
deleted file mode 100644
--- a/oldfile.go
+++ /dev/null
@@ -1,3 +0,0 @@
-package main
-
-func main() {}`

	actual, err := ParseDiff(input)
	assert.NoError(t, err)

	expected := &ctypes.Diff{
		Files: []*ctypes.FileDiff{
			{
				OldPath:   "oldfile.go",
				NewPath:   "oldfile.go",
				OldMode:   "100644",
				IsDeleted: true,
				Hunks: []*ctypes.Hunk{
					{
						OldStart: 1,
						OldLines: 3,
						NewStart: 0,
						NewLines: 0,
						Lines: []*ctypes.Line{
							{Type: ctypes.LineDeleted, Content: "package main", OldNum: 1, NewNum: 0},
							{Type: ctypes.LineDeleted, Content: "", OldNum: 2, NewNum: 0},
							{Type: ctypes.LineDeleted, Content: "func main() {}", OldNum: 3, NewNum: 0},
						},
						Stats: ctypes.HunkStats{Added: 0, Deleted: 3},
					},
				},
			},
		},
	}

	assert.CompareJSON(t, actual, expected)
}

func TestParseDiff_RenamedFile(t *testing.T) {
	input := `diff --git a/old.go b/new.go
rename from old.go
rename to new.go
--- a/old.go
+++ b/new.go
@@ -1,1 +1,1 @@
 package main`

	actual, err := ParseDiff(input)
	assert.NoError(t, err)

	expected := &ctypes.Diff{
		Files: []*ctypes.FileDiff{
			{
				OldPath:   "old.go",
				NewPath:   "new.go",
				IsRenamed: true,
				Hunks: []*ctypes.Hunk{
					{
						OldStart: 1,
						OldLines: 1,
						NewStart: 1,
						NewLines: 1,
						Lines: []*ctypes.Line{
							{Type: ctypes.LineContext, Content: "package main", OldNum: 1, NewNum: 1},
						},
						Stats: ctypes.HunkStats{Added: 0, Deleted: 0},
					},
				},
			},
		},
	}

	assert.CompareJSON(t, actual, expected)
}

func TestParseDiff_BinaryFile(t *testing.T) {
	input := `diff --git a/image.png b/image.png
Binary files a/image.png and b/image.png differ`

	actual, err := ParseDiff(input)
	assert.NoError(t, err)

	expected := &ctypes.Diff{
		Files: []*ctypes.FileDiff{
			{
				OldPath:  "image.png",
				NewPath:  "image.png",
				IsBinary: true,
				Hunks:    []*ctypes.Hunk{},
			},
		},
	}

	assert.CompareJSON(t, actual, expected)
}

func TestParseDiff_LineNumbers(t *testing.T) {
	input := `diff --git a/file.go b/file.go
--- a/file.go
+++ b/file.go
@@ -5,4 +5,5 @@
 line 5
-line 6
+line 6 modified
+line 6.5 added
 line 7
 line 8`

	actual, err := ParseDiff(input)
	assert.NoError(t, err)

	expected := &ctypes.Diff{
		Files: []*ctypes.FileDiff{
			{
				OldPath: "file.go",
				NewPath: "file.go",
				Hunks: []*ctypes.Hunk{
					{
						OldStart: 5,
						OldLines: 4,
						NewStart: 5,
						NewLines: 5,
						Lines: []*ctypes.Line{
							{Type: ctypes.LineContext, Content: "line 5", OldNum: 5, NewNum: 5},
							{Type: ctypes.LineDeleted, Content: "line 6", OldNum: 6, NewNum: 0},
							{Type: ctypes.LineAdded, Content: "line 6 modified", OldNum: 0, NewNum: 6},
							{Type: ctypes.LineAdded, Content: "line 6.5 added", OldNum: 0, NewNum: 7},
							{Type: ctypes.LineContext, Content: "line 7", OldNum: 7, NewNum: 8},
							{Type: ctypes.LineContext, Content: "line 8", OldNum: 8, NewNum: 9},
						},
						Stats: ctypes.HunkStats{Added: 2, Deleted: 1},
					},
				},
			},
		},
	}

	assert.CompareJSON(t, actual, expected)
}

func TestParseDiff_ModeChange(t *testing.T) {
	input := `diff --git a/script.sh b/script.sh
old mode 100644
new mode 100755
--- a/script.sh
+++ b/script.sh
@@ -1,1 +1,1 @@
 #!/bin/bash`

	actual, err := ParseDiff(input)
	assert.NoError(t, err)

	expected := &ctypes.Diff{
		Files: []*ctypes.FileDiff{
			{
				OldPath: "script.sh",
				NewPath: "script.sh",
				OldMode: "100644",
				NewMode: "100755",
				Hunks: []*ctypes.Hunk{
					{
						OldStart: 1,
						OldLines: 1,
						NewStart: 1,
						NewLines: 1,
						Lines: []*ctypes.Line{
							{Type: ctypes.LineContext, Content: "#!/bin/bash", OldNum: 1, NewNum: 1},
						},
						Stats: ctypes.HunkStats{Added: 0, Deleted: 0},
					},
				},
			},
		},
	}

	assert.CompareJSON(t, actual, expected)
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "Unix line endings",
			input: "line1\nline2\nline3",
			want:  []string{"line1", "line2", "line3"},
		},
		{
			name:  "Windows line endings",
			input: "line1\r\nline2\r\nline3",
			want:  []string{"line1", "line2", "line3"},
		},
		{
			name:  "Mixed line endings",
			input: "line1\nline2\r\nline3",
			want:  []string{"line1", "line2", "line3"},
		},
		{
			name:  "Empty string",
			input: "",
			want:  []string{},
		},
		{
			name:  "Single line no newline",
			input: "single",
			want:  []string{"single"},
		},
		{
			name:  "Trailing newline",
			input: "line1\nline2\n",
			want:  []string{"line1", "line2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitLines(tt.input)
			assert.Equals(t, len(got), len(tt.want), "splitLines() length mismatch")
			for i := range got {
				assert.Equals(t, got[i], tt.want[i], "splitLines()[%d] mismatch", i)
			}
		})
	}
}

func TestParseDiffNameStatus_Empty(t *testing.T) {
	actual, err := ParseDiffNameStatus("")
	assert.NoError(t, err)
	assert.Equals(t, len(actual.Files), 0, "empty output should have no files")
}

func TestParseDiffNameStatus_ModifiedFile(t *testing.T) {
	output := "M\tpath/to/file.go"
	actual, err := ParseDiffNameStatus(output)
	assert.NoError(t, err)
	assert.Equals(t, len(actual.Files), 1, "should have 1 file")
	assert.Equals(t, actual.Files[0].OldPath, "path/to/file.go", "old path should match")
	assert.Equals(t, actual.Files[0].NewPath, "path/to/file.go", "new path should match")
	assert.Equals(t, actual.Files[0].IsNew, false, "should not be new")
	assert.Equals(t, actual.Files[0].IsDeleted, false, "should not be deleted")
	assert.Equals(t, actual.Files[0].IsRenamed, false, "should not be renamed")
	assert.Equals(t, len(actual.Files[0].Hunks), 0, "hunks should be empty")
}

func TestParseDiffNameStatus_AddedFile(t *testing.T) {
	output := "A\tnew_file.go"
	actual, err := ParseDiffNameStatus(output)
	assert.NoError(t, err)
	assert.Equals(t, actual.Files[0].IsNew, true, "should be new")
	assert.Equals(t, actual.Files[0].NewPath, "new_file.go", "new path should match")
}

func TestParseDiffNameStatus_DeletedFile(t *testing.T) {
	output := "D\told_file.go"
	actual, err := ParseDiffNameStatus(output)
	assert.NoError(t, err)
	assert.Equals(t, actual.Files[0].IsDeleted, true, "should be deleted")
	assert.Equals(t, actual.Files[0].OldPath, "old_file.go", "old path should match")
}

func TestParseDiffNameStatus_RenamedFile(t *testing.T) {
	output := "R100\told_name.go\tnew_name.go"
	actual, err := ParseDiffNameStatus(output)
	assert.NoError(t, err)
	assert.Equals(t, actual.Files[0].IsRenamed, true, "should be renamed")
	assert.Equals(t, actual.Files[0].OldPath, "old_name.go", "old path should match")
	assert.Equals(t, actual.Files[0].NewPath, "new_name.go", "new path should match")
}

func TestParseDiffNameStatus_MultipleFiles(t *testing.T) {
	output := "M\tfile1.go\nA\tfile2.go\nD\tfile3.go"
	actual, err := ParseDiffNameStatus(output)
	assert.NoError(t, err)
	assert.Equals(t, len(actual.Files), 3, "should have 3 files")

	// First file is modified
	assert.Equals(t, actual.Files[0].NewPath, "file1.go", "first file path")
	assert.Equals(t, actual.Files[0].IsNew, false, "first file not new")

	// Second file is added
	assert.Equals(t, actual.Files[1].NewPath, "file2.go", "second file path")
	assert.Equals(t, actual.Files[1].IsNew, true, "second file is new")

	// Third file is deleted
	assert.Equals(t, actual.Files[2].OldPath, "file3.go", "third file path")
	assert.Equals(t, actual.Files[2].IsDeleted, true, "third file is deleted")
}

func TestParseDiffNameStatus_TypeChange(t *testing.T) {
	output := "T\tchanged_type.go"
	actual, err := ParseDiffNameStatus(output)
	assert.NoError(t, err)
	assert.Equals(t, len(actual.Files), 1, "should have 1 file")
	// T status is treated like M (modified)
	assert.Equals(t, actual.Files[0].IsNew, false, "should not be new")
	assert.Equals(t, actual.Files[0].IsDeleted, false, "should not be deleted")
}

func TestParseDiffNameStatus_CopiedFile(t *testing.T) {
	output := "C100\tsource.go\tdest.go"
	actual, err := ParseDiffNameStatus(output)
	assert.NoError(t, err)
	assert.Equals(t, actual.Files[0].OldPath, "source.go", "source path should match")
	assert.Equals(t, actual.Files[0].NewPath, "dest.go", "dest path should match")
	// Note: Copy status doesn't set IsRenamed
	assert.Equals(t, actual.Files[0].IsRenamed, false, "copy should not be marked as renamed")
}

package git

import (
	"testing"

	ctypes "git.15b.it/eno/critic/pkg/types"

	tu "git.15b.it/eno/critic/internal/testutils"
)

// compareDiff compares actual and expected diffs using JSON serialization
func compareDiff(t *testing.T, actual, expected *ctypes.Diff) {
	t.Helper()
	tu.CompareJSON(t, actual, expected)
}

func TestParseDiff_Empty(t *testing.T) {
	actual, err := ParseDiff("")
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}

	expected := &ctypes.Diff{
		Files: []*ctypes.FileDiff{},
	}

	compareDiff(t, actual, expected)
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
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}

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
					},
				},
			},
		},
	}

	compareDiff(t, actual, expected)
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
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}

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
					},
				},
			},
		},
	}

	compareDiff(t, actual, expected)
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
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}

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
					},
				},
			},
		},
	}

	compareDiff(t, actual, expected)
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
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}

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
					},
				},
			},
		},
	}

	compareDiff(t, actual, expected)
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
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}

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
					},
				},
			},
		},
	}

	compareDiff(t, actual, expected)
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
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}

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
					},
				},
			},
		},
	}

	compareDiff(t, actual, expected)
}

func TestParseDiff_BinaryFile(t *testing.T) {
	input := `diff --git a/image.png b/image.png
Binary files a/image.png and b/image.png differ`

	actual, err := ParseDiff(input)
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}

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

	compareDiff(t, actual, expected)
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
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}

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
					},
				},
			},
		},
	}

	compareDiff(t, actual, expected)
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
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}

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
					},
				},
			},
		},
	}

	compareDiff(t, actual, expected)
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
			if len(got) != len(tt.want) {
				t.Errorf("splitLines() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitLines()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

package git

import (
	"testing"

	ctypes "git.15b.it/eno/critic/pkg/types"
)

func TestParseDiff_Empty(t *testing.T) {
	diff, err := ParseDiff("")
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}

	if diff == nil {
		t.Fatal("ParseDiff() returned nil diff")
	}

	if len(diff.Files) != 0 {
		t.Errorf("ParseDiff() files = %d, want 0", len(diff.Files))
	}
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

	diff, err := ParseDiff(input)
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}

	if len(diff.Files) != 1 {
		t.Fatalf("ParseDiff() files = %d, want 1", len(diff.Files))
	}

	file := diff.Files[0]
	if file.OldPath != "file.go" || file.NewPath != "file.go" {
		t.Errorf("File paths = %q -> %q, want file.go -> file.go", file.OldPath, file.NewPath)
	}

	if len(file.Hunks) != 1 {
		t.Fatalf("File hunks = %d, want 1", len(file.Hunks))
	}

	hunk := file.Hunks[0]
	if hunk.OldStart != 1 || hunk.OldLines != 3 || hunk.NewStart != 1 || hunk.NewLines != 3 {
		t.Errorf("Hunk range = @@ -%d,%d +%d,%d @@, want @@ -1,3 +1,3 @@",
			hunk.OldStart, hunk.OldLines, hunk.NewStart, hunk.NewLines)
	}

	if hunk.Header != "func main() {" {
		t.Errorf("Hunk header = %q, want %q", hunk.Header, "func main() {")
	}

	if len(hunk.Lines) != 4 {
		t.Fatalf("Hunk lines = %d, want 4", len(hunk.Lines))
	}

	// Check line types
	if hunk.Lines[0].Type != ctypes.LineContext {
		t.Errorf("Line 0 type = %v, want LineContext", hunk.Lines[0].Type)
	}
	if hunk.Lines[1].Type != ctypes.LineDeleted {
		t.Errorf("Line 1 type = %v, want LineDeleted", hunk.Lines[1].Type)
	}
	if hunk.Lines[2].Type != ctypes.LineAdded {
		t.Errorf("Line 2 type = %v, want LineAdded", hunk.Lines[2].Type)
	}
	if hunk.Lines[3].Type != ctypes.LineContext {
		t.Errorf("Line 3 type = %v, want LineContext", hunk.Lines[3].Type)
	}

	// Check line content
	if hunk.Lines[0].Content != "package main" {
		t.Errorf("Line 0 content = %q, want %q", hunk.Lines[0].Content, "package main")
	}
	if hunk.Lines[1].Content != `import "fmt"` {
		t.Errorf("Line 1 content = %q, want %q", hunk.Lines[1].Content, `import "fmt"`)
	}
	if hunk.Lines[2].Content != `import "log"` {
		t.Errorf("Line 2 content = %q, want %q", hunk.Lines[2].Content, `import "log"`)
	}
	if hunk.Lines[3].Content != "func main() {" {
		t.Errorf("Line 3 content = %q, want %q", hunk.Lines[3].Content, "func main() {")
	}
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

	diff, err := ParseDiff(input)
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}

	if len(diff.Files) != 2 {
		t.Fatalf("ParseDiff() files = %d, want 2", len(diff.Files))
	}

	if diff.Files[0].NewPath != "file1.go" {
		t.Errorf("First file = %q, want file1.go", diff.Files[0].NewPath)
	}
	if diff.Files[1].NewPath != "file2.go" {
		t.Errorf("Second file = %q, want file2.go", diff.Files[1].NewPath)
	}
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

	diff, err := ParseDiff(input)
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}

	if len(diff.Files) != 1 {
		t.Fatalf("ParseDiff() files = %d, want 1", len(diff.Files))
	}

	file := diff.Files[0]
	if len(file.Hunks) != 2 {
		t.Fatalf("File hunks = %d, want 2", len(file.Hunks))
	}

	if file.Hunks[0].Header != "first hunk" {
		t.Errorf("First hunk header = %q, want %q", file.Hunks[0].Header, "first hunk")
	}
	if file.Hunks[1].Header != "second hunk" {
		t.Errorf("Second hunk header = %q, want %q", file.Hunks[1].Header, "second hunk")
	}
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

	diff, err := ParseDiff(input)
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}

	if len(diff.Files) != 1 {
		t.Fatalf("ParseDiff() files = %d, want 1", len(diff.Files))
	}

	file := diff.Files[0]
	if !file.IsNew {
		t.Error("File.IsNew = false, want true")
	}
	if file.NewMode != "100644" {
		t.Errorf("File.NewMode = %q, want %q", file.NewMode, "100644")
	}
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

	diff, err := ParseDiff(input)
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}

	if len(diff.Files) != 1 {
		t.Fatalf("ParseDiff() files = %d, want 1", len(diff.Files))
	}

	file := diff.Files[0]
	if !file.IsDeleted {
		t.Error("File.IsDeleted = false, want true")
	}
	if file.OldMode != "100644" {
		t.Errorf("File.OldMode = %q, want %q", file.OldMode, "100644")
	}
}

func TestParseDiff_RenamedFile(t *testing.T) {
	input := `diff --git a/old.go b/new.go
rename from old.go
rename to new.go
--- a/old.go
+++ b/new.go
@@ -1,1 +1,1 @@
 package main`

	diff, err := ParseDiff(input)
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}

	if len(diff.Files) != 1 {
		t.Fatalf("ParseDiff() files = %d, want 1", len(diff.Files))
	}

	file := diff.Files[0]
	if !file.IsRenamed {
		t.Error("File.IsRenamed = false, want true")
	}
	if file.OldPath != "old.go" {
		t.Errorf("File.OldPath = %q, want old.go", file.OldPath)
	}
	if file.NewPath != "new.go" {
		t.Errorf("File.NewPath = %q, want new.go", file.NewPath)
	}
}

func TestParseDiff_BinaryFile(t *testing.T) {
	input := `diff --git a/image.png b/image.png
Binary files a/image.png and b/image.png differ`

	diff, err := ParseDiff(input)
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}

	if len(diff.Files) != 1 {
		t.Fatalf("ParseDiff() files = %d, want 1", len(diff.Files))
	}

	file := diff.Files[0]
	if !file.IsBinary {
		t.Error("File.IsBinary = false, want true")
	}
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

	diff, err := ParseDiff(input)
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}

	hunk := diff.Files[0].Hunks[0]

	// Context line should have both old and new line numbers
	if hunk.Lines[0].OldNum != 5 || hunk.Lines[0].NewNum != 5 {
		t.Errorf("Context line numbers = %d/%d, want 5/5", hunk.Lines[0].OldNum, hunk.Lines[0].NewNum)
	}

	// Deleted line should have old num but not new num
	if hunk.Lines[1].OldNum != 6 || hunk.Lines[1].NewNum != 0 {
		t.Errorf("Deleted line numbers = %d/%d, want 6/0", hunk.Lines[1].OldNum, hunk.Lines[1].NewNum)
	}

	// Added line should have new num but not old num
	if hunk.Lines[2].OldNum != 0 || hunk.Lines[2].NewNum != 6 {
		t.Errorf("Added line 1 numbers = %d/%d, want 0/6", hunk.Lines[2].OldNum, hunk.Lines[2].NewNum)
	}

	if hunk.Lines[3].OldNum != 0 || hunk.Lines[3].NewNum != 7 {
		t.Errorf("Added line 2 numbers = %d/%d, want 0/7", hunk.Lines[3].OldNum, hunk.Lines[3].NewNum)
	}

	// Context lines after should continue numbering
	if hunk.Lines[4].OldNum != 7 || hunk.Lines[4].NewNum != 8 {
		t.Errorf("Context line 2 numbers = %d/%d, want 7/8", hunk.Lines[4].OldNum, hunk.Lines[4].NewNum)
	}
}

func TestParseDiff_ModeChange(t *testing.T) {
	input := `diff --git a/script.sh b/script.sh
old mode 100644
new mode 100755
--- a/script.sh
+++ b/script.sh
@@ -1,1 +1,1 @@
 #!/bin/bash`

	diff, err := ParseDiff(input)
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}

	file := diff.Files[0]
	if file.OldMode != "100644" {
		t.Errorf("File.OldMode = %q, want 100644", file.OldMode)
	}
	if file.NewMode != "100755" {
		t.Errorf("File.NewMode = %q, want 100755", file.NewMode)
	}
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

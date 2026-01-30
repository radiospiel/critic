package pprof

import (
	"os"
	"os/exec"
	"runtime/pprof"
	"testing"

	"github.com/radiospiel/critic/src/git"
	"github.com/radiospiel/critic/src/tui"
	"github.com/radiospiel/critic/src/pkg/critic"
	ctypes "github.com/radiospiel/critic/src/pkg/types"
	"github.com/radiospiel/critic/teapot"
)

// createSampleFileDiff creates a sample FileDiff with multiple hunks and lines
// for realistic performance testing.
func createSampleFileDiff(numHunks, linesPerHunk int) *ctypes.FileDiff {
	hunks := make([]*ctypes.Hunk, numHunks)
	lineNum := 1

	for h := range numHunks {
		lines := make([]*ctypes.Line, linesPerHunk)
		for l := range linesPerHunk {
			lineType := ctypes.LineContext
			switch l % 3 {
			case 0:
				lineType = ctypes.LineAdded
			case 1:
				lineType = ctypes.LineDeleted
			case 2:
				lineType = ctypes.LineContext
			}

			lines[l] = &ctypes.Line{
				Type:    lineType,
				Content: "    sample line content for performance testing with sufficient length to simulate real code",
				OldNum:  lineNum,
				NewNum:  lineNum,
			}
			lineNum++
		}

		hunks[h] = &ctypes.Hunk{
			OldStart: h*linesPerHunk + 1,
			OldLines: linesPerHunk,
			NewStart: h*linesPerHunk + 1,
			NewLines: linesPerHunk,
			Header:   "func exampleFunction()",
			Lines:    lines,
			Stats: ctypes.HunkStats{
				Added:   linesPerHunk / 3,
				Deleted: linesPerHunk / 3,
			},
		}
	}

	return &ctypes.FileDiff{
		OldPath: "pkg/example/sample_file.go",
		NewPath: "pkg/example/sample_file.go",
		Hunks:   hunks,
	}
}

// createSampleFiles creates multiple sample FileDiff objects for FileListWidget testing.
func createSampleFiles(count int) []*ctypes.FileDiff {
	files := make([]*ctypes.FileDiff, count)
	for i := range count {
		files[i] = &ctypes.FileDiff{
			OldPath:   "",
			NewPath:   "pkg/example/file_" + string(rune('a'+i%26)) + ".go",
			IsNew:     i%4 == 0,
			IsDeleted: i%5 == 0,
			IsRenamed: i%7 == 0,
			Hunks: []*ctypes.Hunk{
				{
					OldStart: 1,
					OldLines: 10,
					NewStart: 1,
					NewLines: 12,
					Lines:    []*ctypes.Line{},
				},
			},
		}
	}
	return files
}

// getGitDiffBetweenTags returns the raw diff output between two git tags.
func getGitDiffBetweenTags(fromTag, toTag string) (string, error) {
	cmd := exec.Command("git", "diff", fromTag+".."+toTag, "--patch", "--no-color")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// cpuProfile runs a function while capturing a CPU profile to the specified file.
// It logs the message before profiling and writes analysis instructions after.
func cpuProfile(t *testing.T, filename string, msg string, fn func()) {
	t.Helper()

	profileFile, err := os.Create(filename)
	if err != nil {
		t.Fatalf("failed to create profile file %s: %v", filename, err)
	}
	defer profileFile.Close()

	t.Logf("Profiling: %s", msg)

	if err := pprof.StartCPUProfile(profileFile); err != nil {
		t.Fatalf("failed to start CPU profile: %v", err)
	}

	fn()

	pprof.StopCPUProfile()

	t.Logf("CPU profile written to %s", filename)
	t.Logf("Analyze with: go tool pprof %s", filename)
}

// TestRenderProfile profiles rendering of FileListWidget and DiffViewWidget in a compositor.
// It loads a real git diff between ex1 and ex2 commits and profiles the initial render.
func TestRenderProfile(t *testing.T) {
	// Get real diff between ex1 and ex2 commits
	diffOutput, err := getGitDiffBetweenTags("2ce5966eb06280e48496372b1c0ab11f81d1b138", "e85a45510b488e01f4119448281c5318fe1b7644")
	if err != nil {
		t.Fatalf("failed to get git diff: %v", err)
	}

	diff, err := git.ParseDiff(diffOutput)
	if err != nil {
		t.Fatalf("failed to parse diff: %v", err)
	}

	if len(diff.Files) == 0 {
		t.Fatal("no files in diff")
	}

	// Find internal/app/app.go in the diff
	var appFile *ctypes.FileDiff
	for _, f := range diff.Files {
		if f.NewPath == "internal/app/app.go" {
			appFile = f
			break
		}
	}
	if appFile == nil {
		t.Fatal("internal/app/app.go not found in diff")
	}

	// Create widgets
	fileListWidget := tui.NewFilesListView(nil, critic.NewDummyMessaging())
	fileListWidget.SetFiles(diff.Files)

	diffViewWidget := tui.NewDiffView(nil, nil)
	// Load internal/app/app.go into the diff view
	diffViewWidget.SetFile(appFile)

	// Create an HSplit layout with file list on left, diff view on right
	split := teapot.NewHSplit(fileListWidget, diffViewWidget, 0.25)

	// Create compositor with the split as root
	compositor := teapot.NewCompositor(split)
	compositor.Resize(160, 50)

	t.Logf("Diff contains %d files", len(diff.Files))
	t.Logf("Loaded file: %s with %d hunks", appFile.NewPath, len(appFile.Hunks))

	// Profile the initial render
	cpuProfile(t, "initial_render_profile.prof", "initial compositor render", func() {
		_ = compositor.View()
	})
}

// BenchmarkFileListWidgetRender benchmarks FileListWidget rendering.
func BenchmarkFileListWidgetRender(b *testing.B) {
	files := createSampleFiles(50)
	widget := tui.NewFilesListView(nil, critic.NewDummyMessaging())
	widget.SetFiles(files)
	widget.SetBounds(teapot.NewRect(0, 0, 40, 50))

	buf := teapot.NewBuffer(40, 50)

	// Warmup
	widget.Render(teapot.NewSubBuffer(buf, teapot.NewRect(0, 0, 40, 50)))

	b.ResetTimer()
	for b.Loop() {
		buf.Clear()
		widget.Render(teapot.NewSubBuffer(buf, teapot.NewRect(0, 0, 40, 50)))
	}
}

// BenchmarkDiffViewWidgetRender benchmarks DiffViewWidget rendering.
func BenchmarkDiffViewWidgetRender(b *testing.B) {
	fileDiff := createSampleFileDiff(10, 30)
	widget := tui.NewDiffView(nil, nil)
	widget.SetFile(fileDiff)
	widget.SetBounds(teapot.NewRect(0, 0, 120, 50))

	buf := teapot.NewBuffer(120, 50)

	// Warmup
	widget.Render(teapot.NewSubBuffer(buf, teapot.NewRect(0, 0, 120, 50)))

	b.ResetTimer()
	for b.Loop() {
		buf.Clear()
		widget.Render(teapot.NewSubBuffer(buf, teapot.NewRect(0, 0, 120, 50)))
	}
}

// BenchmarkCombinedRender benchmarks rendering both FileListWidget and DiffViewWidget together.
func BenchmarkCombinedRender(b *testing.B) {
	files := createSampleFiles(50)
	fileDiff := createSampleFileDiff(10, 30)

	fileListWidget := tui.NewFilesListView(nil, critic.NewDummyMessaging())
	fileListWidget.SetFiles(files)
	fileListWidget.SetBounds(teapot.NewRect(0, 0, 40, 50))

	diffViewWidget := tui.NewDiffView(nil, nil)
	diffViewWidget.SetFile(fileDiff)
	diffViewWidget.SetBounds(teapot.NewRect(0, 0, 120, 50))

	fileListBuf := teapot.NewBuffer(40, 50)
	diffViewBuf := teapot.NewBuffer(120, 50)

	// Warmup
	fileListWidget.Render(teapot.NewSubBuffer(fileListBuf, teapot.NewRect(0, 0, 40, 50)))
	diffViewWidget.Render(teapot.NewSubBuffer(diffViewBuf, teapot.NewRect(0, 0, 120, 50)))

	b.ResetTimer()
	for b.Loop() {
		fileListBuf.Clear()
		diffViewBuf.Clear()
		fileListWidget.Render(teapot.NewSubBuffer(fileListBuf, teapot.NewRect(0, 0, 40, 50)))
		diffViewWidget.Render(teapot.NewSubBuffer(diffViewBuf, teapot.NewRect(0, 0, 120, 50)))
	}
}

// BenchmarkLargeDiffViewRender benchmarks rendering a large diff view.
func BenchmarkLargeDiffViewRender(b *testing.B) {
	// Large file: 50 hunks, 100 lines each = 5000 lines
	fileDiff := createSampleFileDiff(50, 100)
	widget := tui.NewDiffView(nil, nil)
	widget.SetFile(fileDiff)
	widget.SetBounds(teapot.NewRect(0, 0, 200, 100))

	buf := teapot.NewBuffer(200, 100)

	// Warmup
	widget.Render(teapot.NewSubBuffer(buf, teapot.NewRect(0, 0, 200, 100)))

	b.ResetTimer()
	for b.Loop() {
		buf.Clear()
		widget.Render(teapot.NewSubBuffer(buf, teapot.NewRect(0, 0, 200, 100)))
	}
}

// BenchmarkCompositorView benchmarks the compositor rendering both widgets together.
// This measures the full rendering pipeline including caching and buffer composition.
func BenchmarkCompositorView(b *testing.B) {
	files := createSampleFiles(50)
	fileDiff := createSampleFileDiff(10, 30)

	// Create widgets
	fileListWidget := tui.NewFilesListView(nil, critic.NewDummyMessaging())
	fileListWidget.SetFiles(files)

	diffViewWidget := tui.NewDiffView(nil, nil)
	diffViewWidget.SetFile(fileDiff)

	// Create an HSplit layout with file list on left, diff view on right
	split := teapot.NewHSplit(fileListWidget, diffViewWidget, 0.25)

	// Create compositor with the split as root
	compositor := teapot.NewCompositor(split)
	compositor.Resize(160, 50)

	// First render (warmup / cache population)
	_ = compositor.View()

	// Mark dirty to force re-render for second call
	compositor.MarkDirty()

	// Second render (still establishing baseline)
	_ = compositor.View()

	b.ResetTimer()
	for b.Loop() {
		// Mark dirty to ensure actual rendering happens
		compositor.MarkDirty()
		_ = compositor.View()
	}
}

// BenchmarkCompositorViewCached benchmarks compositor with caching (no dirty marking).
// This measures the performance when widgets don't need re-rendering.
func BenchmarkCompositorViewCached(b *testing.B) {
	files := createSampleFiles(50)
	fileDiff := createSampleFileDiff(10, 30)

	fileListWidget := tui.NewFilesListView(nil, critic.NewDummyMessaging())
	fileListWidget.SetFiles(files)

	diffViewWidget := tui.NewDiffView(nil, nil)
	diffViewWidget.SetFile(fileDiff)

	split := teapot.NewHSplit(fileListWidget, diffViewWidget, 0.25)
	compositor := teapot.NewCompositor(split)
	compositor.Resize(160, 50)

	// First render (warmup)
	_ = compositor.View()

	// Second render (cache populated)
	_ = compositor.View()

	b.ResetTimer()
	for b.Loop() {
		// Don't mark dirty - test cached path
		_ = compositor.View()
	}
}

// TestCompositorRenderProfile profiles the compositor View() with both widgets.
func TestCompositorRenderProfile(t *testing.T) {
	files := createSampleFiles(50)
	fileDiff := createSampleFileDiff(10, 30)

	fileListWidget := tui.NewFilesListView(nil, critic.NewDummyMessaging())
	fileListWidget.SetFiles(files)

	diffViewWidget := tui.NewDiffView(nil, nil)
	diffViewWidget.SetFile(fileDiff)

	split := teapot.NewHSplit(fileListWidget, diffViewWidget, 0.25)
	compositor := teapot.NewCompositor(split)
	compositor.Resize(160, 50)

	// First render (warmup)
	_ = compositor.View()

	// Second render (rerender before profiling)
	compositor.MarkDirty()
	_ = compositor.View()

	// Profile the subsequent rerenders
	cpuProfile(t, "compositor_profile.prof", "compositor rerenders (10x)", func() {
		for range 10 {
			compositor.MarkDirty()
			_ = compositor.View()
		}
	})
}

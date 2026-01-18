package pprof

import (
	"os"
	"runtime/pprof"
	"testing"

	"git.15b.it/eno/critic/internal/tui"
	ctypes "git.15b.it/eno/critic/pkg/types"
	"git.15b.it/eno/critic/teapot"
)

// createSampleFileDiff creates a sample FileDiff with multiple hunks and lines
// for realistic performance testing.
func createSampleFileDiff(numHunks, linesPerHunk int) *ctypes.FileDiff {
	hunks := make([]*ctypes.Hunk, numHunks)
	lineNum := 1

	for h := 0; h < numHunks; h++ {
		lines := make([]*ctypes.Line, linesPerHunk)
		for l := 0; l < linesPerHunk; l++ {
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
	for i := 0; i < count; i++ {
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

// TestRenderProfile profiles rendering of FileListWidget and DiffViewWidget.
// It builds the widgets, renders once, then profiles the second rerender 10 times.
func TestRenderProfile(t *testing.T) {
	// Create sample data
	files := createSampleFiles(50)
	fileDiff := createSampleFileDiff(10, 30) // 10 hunks, 30 lines each

	// Create widgets
	fileListWidget := tui.NewFileListWidget()
	fileListWidget.SetFiles(files)
	fileListWidget.SetBounds(teapot.NewRect(0, 0, 40, 50))

	diffViewWidget := tui.NewDiffViewWidget()
	diffViewWidget.SetFile(fileDiff, nil, nil, nil, nil)
	diffViewWidget.SetBounds(teapot.NewRect(0, 0, 120, 50))

	// Create buffers
	fileListBuf := teapot.NewBuffer(40, 50)
	diffViewBuf := teapot.NewBuffer(120, 50)

	// First render (warmup)
	fileListWidget.Render(fileListBuf.Sub(teapot.Rect{X: 0, Y: 0, Width: 40, Height: 50}))
	diffViewWidget.Render(diffViewBuf.Sub(teapot.Rect{X: 0, Y: 0, Width: 120, Height: 50}))

	// Second render (rerender before profiling)
	fileListBuf.Clear()
	diffViewBuf.Clear()
	fileListWidget.Render(fileListBuf.Sub(teapot.Rect{X: 0, Y: 0, Width: 40, Height: 50}))
	diffViewWidget.Render(diffViewBuf.Sub(teapot.Rect{X: 0, Y: 0, Width: 120, Height: 50}))

	// Now profile the second rerender 10 times
	profileFile, err := os.Create("render_profile.prof")
	if err != nil {
		t.Fatalf("failed to create profile file: %v", err)
	}
	defer profileFile.Close()

	if err := pprof.StartCPUProfile(profileFile); err != nil {
		t.Fatalf("failed to start CPU profile: %v", err)
	}

	for i := 0; i < 10; i++ {
		fileListBuf.Clear()
		diffViewBuf.Clear()
		fileListWidget.Render(fileListBuf.Sub(teapot.Rect{X: 0, Y: 0, Width: 40, Height: 50}))
		diffViewWidget.Render(diffViewBuf.Sub(teapot.Rect{X: 0, Y: 0, Width: 120, Height: 50}))
	}

	pprof.StopCPUProfile()

	t.Logf("CPU profile written to render_profile.prof")
	t.Logf("Analyze with: go tool pprof render_profile.prof")
}

// BenchmarkFileListWidgetRender benchmarks FileListWidget rendering.
func BenchmarkFileListWidgetRender(b *testing.B) {
	files := createSampleFiles(50)
	widget := tui.NewFileListWidget()
	widget.SetFiles(files)
	widget.SetBounds(teapot.NewRect(0, 0, 40, 50))

	buf := teapot.NewBuffer(40, 50)

	// Warmup
	widget.Render(buf.Sub(teapot.Rect{X: 0, Y: 0, Width: 40, Height: 50}))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Clear()
		widget.Render(buf.Sub(teapot.Rect{X: 0, Y: 0, Width: 40, Height: 50}))
	}
}

// BenchmarkDiffViewWidgetRender benchmarks DiffViewWidget rendering.
func BenchmarkDiffViewWidgetRender(b *testing.B) {
	fileDiff := createSampleFileDiff(10, 30)
	widget := tui.NewDiffViewWidget()
	widget.SetFile(fileDiff, nil, nil, nil, nil)
	widget.SetBounds(teapot.NewRect(0, 0, 120, 50))

	buf := teapot.NewBuffer(120, 50)

	// Warmup
	widget.Render(buf.Sub(teapot.Rect{X: 0, Y: 0, Width: 120, Height: 50}))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Clear()
		widget.Render(buf.Sub(teapot.Rect{X: 0, Y: 0, Width: 120, Height: 50}))
	}
}

// BenchmarkCombinedRender benchmarks rendering both FileListWidget and DiffViewWidget together.
func BenchmarkCombinedRender(b *testing.B) {
	files := createSampleFiles(50)
	fileDiff := createSampleFileDiff(10, 30)

	fileListWidget := tui.NewFileListWidget()
	fileListWidget.SetFiles(files)
	fileListWidget.SetBounds(teapot.NewRect(0, 0, 40, 50))

	diffViewWidget := tui.NewDiffViewWidget()
	diffViewWidget.SetFile(fileDiff, nil, nil, nil, nil)
	diffViewWidget.SetBounds(teapot.NewRect(0, 0, 120, 50))

	fileListBuf := teapot.NewBuffer(40, 50)
	diffViewBuf := teapot.NewBuffer(120, 50)

	// Warmup
	fileListWidget.Render(fileListBuf.Sub(teapot.Rect{X: 0, Y: 0, Width: 40, Height: 50}))
	diffViewWidget.Render(diffViewBuf.Sub(teapot.Rect{X: 0, Y: 0, Width: 120, Height: 50}))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fileListBuf.Clear()
		diffViewBuf.Clear()
		fileListWidget.Render(fileListBuf.Sub(teapot.Rect{X: 0, Y: 0, Width: 40, Height: 50}))
		diffViewWidget.Render(diffViewBuf.Sub(teapot.Rect{X: 0, Y: 0, Width: 120, Height: 50}))
	}
}

// BenchmarkLargeDiffViewRender benchmarks rendering a large diff view.
func BenchmarkLargeDiffViewRender(b *testing.B) {
	// Large file: 50 hunks, 100 lines each = 5000 lines
	fileDiff := createSampleFileDiff(50, 100)
	widget := tui.NewDiffViewWidget()
	widget.SetFile(fileDiff, nil, nil, nil, nil)
	widget.SetBounds(teapot.NewRect(0, 0, 200, 100))

	buf := teapot.NewBuffer(200, 100)

	// Warmup
	widget.Render(buf.Sub(teapot.Rect{X: 0, Y: 0, Width: 200, Height: 100}))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Clear()
		widget.Render(buf.Sub(teapot.Rect{X: 0, Y: 0, Width: 200, Height: 100}))
	}
}

// BenchmarkCompositorView benchmarks the compositor rendering both widgets together.
// This measures the full rendering pipeline including caching and buffer composition.
func BenchmarkCompositorView(b *testing.B) {
	files := createSampleFiles(50)
	fileDiff := createSampleFileDiff(10, 30)

	// Create widgets
	fileListWidget := tui.NewFileListWidget()
	fileListWidget.SetFiles(files)

	diffViewWidget := tui.NewDiffViewWidget()
	diffViewWidget.SetFile(fileDiff, nil, nil, nil, nil)

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
	for i := 0; i < b.N; i++ {
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

	fileListWidget := tui.NewFileListWidget()
	fileListWidget.SetFiles(files)

	diffViewWidget := tui.NewDiffViewWidget()
	diffViewWidget.SetFile(fileDiff, nil, nil, nil, nil)

	split := teapot.NewHSplit(fileListWidget, diffViewWidget, 0.25)
	compositor := teapot.NewCompositor(split)
	compositor.Resize(160, 50)

	// First render (warmup)
	_ = compositor.View()

	// Second render (cache populated)
	_ = compositor.View()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Don't mark dirty - test cached path
		_ = compositor.View()
	}
}

// TestCompositorRenderProfile profiles the compositor View() with both widgets.
func TestCompositorRenderProfile(t *testing.T) {
	files := createSampleFiles(50)
	fileDiff := createSampleFileDiff(10, 30)

	fileListWidget := tui.NewFileListWidget()
	fileListWidget.SetFiles(files)

	diffViewWidget := tui.NewDiffViewWidget()
	diffViewWidget.SetFile(fileDiff, nil, nil, nil, nil)

	split := teapot.NewHSplit(fileListWidget, diffViewWidget, 0.25)
	compositor := teapot.NewCompositor(split)
	compositor.Resize(160, 50)

	// First render (warmup)
	_ = compositor.View()

	// Second render (rerender before profiling)
	compositor.MarkDirty()
	_ = compositor.View()

	// Profile the subsequent rerenders
	profileFile, err := os.Create("compositor_profile.prof")
	if err != nil {
		t.Fatalf("failed to create profile file: %v", err)
	}
	defer profileFile.Close()

	if err := pprof.StartCPUProfile(profileFile); err != nil {
		t.Fatalf("failed to start CPU profile: %v", err)
	}

	for i := 0; i < 10; i++ {
		compositor.MarkDirty()
		_ = compositor.View()
	}

	pprof.StopCPUProfile()

	t.Logf("CPU profile written to compositor_profile.prof")
	t.Logf("Analyze with: go tool pprof compositor_profile.prof")
}

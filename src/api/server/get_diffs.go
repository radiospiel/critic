package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/types"
)

// GetDiffs returns the current diffs and state from the session.
func (s *Server) GetDiffs(
	ctx context.Context,
	req *connect.Request[api.GetDiffsRequest],
) (*connect.Response[api.GetDiffsResponse], error) {
	session := s.GetSession()
	state := session.GetState()
	diff := session.GetDiff()

	res := connect.NewResponse(&api.GetDiffsResponse{
		State: string(state),
		Diff:  convertDiff(diff),
	})
	return res, nil
}

// convertDiff converts a types.Diff to an api.Diff
func convertDiff(d *types.Diff) *api.Diff {
	if d == nil {
		return nil
	}

	files := make([]*api.FileDiff, len(d.Files))
	for i, f := range d.Files {
		files[i] = convertFileDiff(f)
	}

	return &api.Diff{
		Files: files,
	}
}

// convertFileDiff converts a types.FileDiff to an api.FileDiff
func convertFileDiff(f *types.FileDiff) *api.FileDiff {
	if f == nil {
		return nil
	}

	hunks := make([]*api.Hunk, len(f.Hunks))
	for i, h := range f.Hunks {
		hunks[i] = convertHunk(h)
	}

	return &api.FileDiff{
		OldPath:   f.OldPath,
		NewPath:   f.NewPath,
		OldMode:   f.OldMode,
		NewMode:   f.NewMode,
		IsNew:     f.IsNew,
		IsDeleted: f.IsDeleted,
		IsRenamed: f.IsRenamed,
		IsBinary:  f.IsBinary,
		Hunks:     hunks,
	}
}

// convertHunk converts a types.Hunk to an api.Hunk
func convertHunk(h *types.Hunk) *api.Hunk {
	if h == nil {
		return nil
	}

	lines := make([]*api.Line, len(h.Lines))
	for i, l := range h.Lines {
		lines[i] = convertLine(l)
	}

	return &api.Hunk{
		OldStart: int32(h.OldStart),
		OldLines: int32(h.OldLines),
		NewStart: int32(h.NewStart),
		NewLines: int32(h.NewLines),
		Header:   h.Header,
		Lines:    lines,
		Stats: &api.HunkStats{
			Added:   int32(h.Stats.Added),
			Deleted: int32(h.Stats.Deleted),
		},
	}
}

// convertLine converts a types.Line to an api.Line
func convertLine(l *types.Line) *api.Line {
	if l == nil {
		return nil
	}

	var lineType string
	switch l.Type {
	case types.LineContext:
		lineType = "context"
	case types.LineAdded:
		lineType = "added"
	case types.LineDeleted:
		lineType = "deleted"
	default:
		lineType = "unknown"
	}

	return &api.Line{
		Type:    lineType,
		Content: l.Content,
		OldNum:  int32(l.OldNum),
		NewNum:  int32(l.NewNum),
	}
}

package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/types"
)

// GetDiff returns the full diff for a specific file path.
func (s *Server) GetDiff(
	ctx context.Context,
	req *connect.Request[api.GetDiffRequest],
) (*connect.Response[api.GetDiffResponse], error) {
	session := s.GetSession()
	path := req.Msg.GetPath()

	fileDiff := session.GetFileDiff(path)

	res := connect.NewResponse(&api.GetDiffResponse{
		File: convertFileDiff(fileDiff),
	})
	return res, nil
}

// convertFileDiff converts a types.FileDiff to an api.FileDiff (with hunks)
func convertFileDiff(f *types.FileDiff) *api.FileDiff {
	if f == nil {
		return nil
	}

	hunks := make([]*api.Hunk, len(f.Hunks))
	for i, h := range f.Hunks {
		hunks[i] = convertHunk(h)
	}

	status := api.FileStatus_FILE_STATUS_MODIFIED
	switch {
	case f.IsNew:
		status = api.FileStatus_FILE_STATUS_NEW
	case f.IsDeleted:
		status = api.FileStatus_FILE_STATUS_DELETED
	case f.IsRenamed:
		status = api.FileStatus_FILE_STATUS_RENAMED
	}

	return &api.FileDiff{
		OldPath:     f.OldPath,
		NewPath:     f.NewPath,
		FileModeOld: f.OldMode,
		FileModeNew: f.NewMode,
		Status:      status,
		IsBinary:    f.IsBinary,
		Hunks:       hunks,
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

	var lineType api.LineType
	switch l.Type {
	case types.LineContext:
		lineType = api.LineType_LINE_TYPE_CONTEXT
	case types.LineAdded:
		lineType = api.LineType_LINE_TYPE_ADDED
	case types.LineDeleted:
		lineType = api.LineType_LINE_TYPE_DELETED
	default:
		lineType = api.LineType_LINE_TYPE_UNSPECIFIED
	}

	return &api.Line{
		Type:      lineType,
		Content:   l.Content,
		LineNoOld: int32(l.OldNum),
		LineNoNew: int32(l.NewNum),
	}
}

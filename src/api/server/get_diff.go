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
	response := depanic(func() (*api.GetDiffResponse, error) {
		return getDiffImpl(s, req.Msg)
	})
	return connect.NewResponse(response), nil
}

func getDiffImpl(server *Server, req *api.GetDiffRequest) (*api.GetDiffResponse, error) {
	session := server.GetSession()
	path := req.GetPath()
	contextLines := int(req.GetContextLines())

	fileDiff := session.GetFileDiff(path, contextLines)

	return &api.GetDiffResponse{
		File: convertFileDiff(fileDiff),
	}, nil
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

	return &api.FileDiff{
		OldPath:     f.OldPath,
		NewPath:     f.NewPath,
		FileModeOld: f.OldMode,
		FileModeNew: f.NewMode,
		Status:      convertFileStatus(f.FileStatus),
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

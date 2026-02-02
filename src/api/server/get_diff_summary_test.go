package server

import (
	"testing"

	"github.com/radiospiel/critic/simple-go/assert"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/types"
)

func TestConvertDiffSummary(t *testing.T) {
	// Test nil diff
	result := convertDiffSummary(nil)
	assert.Nil(t, result, "nil diff should return nil")

	// Test empty diff
	files := []*types.FileDiff{}
	result = convertDiffSummary(files)
	assert.NotNil(t, result, "empty diff should not be nil")
	assert.Equals(t, len(result.Files), 0, "empty diff should have no files")

	// Test diff with files
	files = []*types.FileDiff{
		{
			OldPath:   "old.go",
			NewPath:   "new.go",
			IsRenamed: true,
			Hunks: []*types.Hunk{
				{
					OldStart: 1,
					OldLines: 5,
					NewStart: 1,
					NewLines: 7,
					Header:   "@@ -1,5 +1,7 @@",
					Stats:    types.HunkStats{Added: 3, Deleted: 1},
					Lines: []*types.Line{
						{Type: types.LineContext, Content: "context line", OldNum: 1, NewNum: 1},
					},
				},
			},
		},
	}

	result = convertDiffSummary(files)
	assert.NotNil(t, result, "diff should not be nil")
	assert.Equals(t, len(result.Files), 1, "should have 1 file")

	file := result.Files[0]
	assert.Equals(t, file.OldPath, "old.go", "old path should match")
	assert.Equals(t, file.NewPath, "new.go", "new path should match")
	assert.Equals(t, file.Status, api.FileStatus_FILE_STATUS_RENAMED, "status should be RENAMED")
}

func TestConvertFileSummary(t *testing.T) {
	// Test nil file diff
	result := convertFileSummary(nil)
	assert.Nil(t, result, "nil file diff should return nil")

	// Test file diff with all fields - renamed file
	fd := &types.FileDiff{
		OldPath:   "path/to/old.go",
		NewPath:   "path/to/new.go",
		OldMode:   "100644",
		NewMode:   "100755",
		IsNew:     false,
		IsDeleted: false,
		IsRenamed: true,
		IsBinary:  false,
		Hunks: []*types.Hunk{
			{OldStart: 1, OldLines: 5},
		},
	}

	result = convertFileSummary(fd)
	assert.NotNil(t, result, "result should not be nil")
	assert.Equals(t, result.OldPath, "path/to/old.go", "old path should match")
	assert.Equals(t, result.NewPath, "path/to/new.go", "new path should match")
	assert.Equals(t, result.FileModeOld, "100644", "file_mode_old should match")
	assert.Equals(t, result.FileModeNew, "100755", "file_mode_new should match")
	assert.Equals(t, result.Status, api.FileStatus_FILE_STATUS_RENAMED, "status should be RENAMED")
	assert.Equals(t, result.IsBinary, false, "is_binary should be false")

	// Test new file
	fd = &types.FileDiff{IsNew: true}
	result = convertFileSummary(fd)
	assert.Equals(t, result.Status, api.FileStatus_FILE_STATUS_NEW, "status should be NEW")

	// Test deleted file
	fd = &types.FileDiff{IsDeleted: true}
	result = convertFileSummary(fd)
	assert.Equals(t, result.Status, api.FileStatus_FILE_STATUS_DELETED, "status should be DELETED")

	// Test modified file (default)
	fd = &types.FileDiff{}
	result = convertFileSummary(fd)
	assert.Equals(t, result.Status, api.FileStatus_FILE_STATUS_MODIFIED, "status should be MODIFIED")
}

func TestGetDiffSummaryResponseTypes(t *testing.T) {
	// Test that the API types implement ProtoMessage
	var _ interface{ ProtoMessage() } = (*api.GetDiffSummaryRequest)(nil)
	var _ interface{ ProtoMessage() } = (*api.GetDiffSummaryResponse)(nil)
	var _ interface{ ProtoMessage() } = (*api.DiffSummary)(nil)
	var _ interface{ ProtoMessage() } = (*api.FileSummary)(nil)

	// Test getters on GetDiffSummaryResponse
	resp := &api.GetDiffSummaryResponse{
		State: "READY",
		Diff:  &api.DiffSummary{Files: []*api.FileSummary{}},
	}
	assert.Equals(t, resp.GetState(), "READY", "state should be READY")
	assert.NotNil(t, resp.GetDiff(), "diff should not be nil")

	// Test nil receiver
	var nilResp *api.GetDiffSummaryResponse
	assert.Equals(t, nilResp.GetState(), "", "nil receiver should return empty string")
	assert.Nil(t, nilResp.GetDiff(), "nil receiver should return nil diff")
}

func TestGetDiffResponseTypes(t *testing.T) {
	// Test that the API types implement ProtoMessage
	var _ interface{ ProtoMessage() } = (*api.GetDiffRequest)(nil)
	var _ interface{ ProtoMessage() } = (*api.GetDiffResponse)(nil)
	var _ interface{ ProtoMessage() } = (*api.Diff)(nil)
	var _ interface{ ProtoMessage() } = (*api.FileDiff)(nil)
	var _ interface{ ProtoMessage() } = (*api.Hunk)(nil)
	var _ interface{ ProtoMessage() } = (*api.HunkStats)(nil)
	var _ interface{ ProtoMessage() } = (*api.Line)(nil)

	// Test getters on GetDiffRequest
	req := &api.GetDiffRequest{Path: "file.go"}
	assert.Equals(t, req.GetPath(), "file.go", "path should match")

	// Test getters on GetDiffResponse
	resp := &api.GetDiffResponse{
		File: &api.FileDiff{NewPath: "file.go"},
	}
	assert.NotNil(t, resp.GetFile(), "file should not be nil")
	assert.Equals(t, resp.GetFile().GetNewPath(), "file.go", "path should match")

	// Test nil receiver
	var nilResp *api.GetDiffResponse
	assert.Nil(t, nilResp.GetFile(), "nil receiver should return nil file")
}

func TestFileStatusEnum(t *testing.T) {
	// Test FileStatus enum values
	assert.Equals(t, api.FileStatus_FILE_STATUS_UNSPECIFIED, api.FileStatus(0), "UNSPECIFIED should be 0")
	assert.Equals(t, api.FileStatus_FILE_STATUS_MODIFIED, api.FileStatus(1), "MODIFIED should be 1")
	assert.Equals(t, api.FileStatus_FILE_STATUS_NEW, api.FileStatus(2), "NEW should be 2")
	assert.Equals(t, api.FileStatus_FILE_STATUS_DELETED, api.FileStatus(3), "DELETED should be 3")
	assert.Equals(t, api.FileStatus_FILE_STATUS_RENAMED, api.FileStatus(4), "RENAMED should be 4")

	// Test String() method
	assert.Equals(t, api.FileStatus_FILE_STATUS_MODIFIED.String(), "FILE_STATUS_MODIFIED", "String() should return correct name")
}

func TestLineTypeEnum(t *testing.T) {
	// Test LineType enum values
	assert.Equals(t, api.LineType_LINE_TYPE_UNSPECIFIED, api.LineType(0), "UNSPECIFIED should be 0")
	assert.Equals(t, api.LineType_LINE_TYPE_CONTEXT, api.LineType(1), "CONTEXT should be 1")
	assert.Equals(t, api.LineType_LINE_TYPE_ADDED, api.LineType(2), "ADDED should be 2")
	assert.Equals(t, api.LineType_LINE_TYPE_DELETED, api.LineType(3), "DELETED should be 3")

	// Test String() method
	assert.Equals(t, api.LineType_LINE_TYPE_CONTEXT.String(), "LINE_TYPE_CONTEXT", "String() should return correct name")
}

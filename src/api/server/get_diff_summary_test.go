package server

import (
	"testing"

	"github.com/radiospiel/critic/simple-go/assert"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/types"
)

func TestConvertDiffWithoutHunks(t *testing.T) {
	// Test nil diff
	result := convertDiffWithoutHunks(nil)
	assert.Nil(t, result, "nil diff should return nil")

	// Test empty diff
	diff := &types.Diff{Files: []*types.FileDiff{}}
	result = convertDiffWithoutHunks(diff)
	assert.NotNil(t, result, "empty diff should not be nil")
	assert.Equals(t, len(result.Files), 0, "empty diff should have no files")

	// Test diff with files (hunks should be dropped)
	diff = &types.Diff{
		Files: []*types.FileDiff{
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
		},
	}

	result = convertDiffWithoutHunks(diff)
	assert.NotNil(t, result, "diff should not be nil")
	assert.Equals(t, len(result.Files), 1, "should have 1 file")

	file := result.Files[0]
	assert.Equals(t, file.OldPath, "old.go", "old path should match")
	assert.Equals(t, file.NewPath, "new.go", "new path should match")
	assert.Equals(t, file.Status, api.FileStatus_FILE_STATUS_RENAMED, "status should be RENAMED")

	// Key test: hunks should not be included in summary
	assert.Nil(t, file.Hunks, "hunks should be nil in summary response")
}

func TestConvertFileDiffSummary(t *testing.T) {
	// Test nil file diff
	result := convertFileDiffSummary(nil)
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

	result = convertFileDiffSummary(fd)
	assert.NotNil(t, result, "result should not be nil")
	assert.Equals(t, result.OldPath, "path/to/old.go", "old path should match")
	assert.Equals(t, result.NewPath, "path/to/new.go", "new path should match")
	assert.Equals(t, result.FileModeOld, "100644", "file_mode_old should match")
	assert.Equals(t, result.FileModeNew, "100755", "file_mode_new should match")
	assert.Equals(t, result.Status, api.FileStatus_FILE_STATUS_RENAMED, "status should be RENAMED")
	assert.Equals(t, result.IsBinary, false, "is_binary should be false")

	// Key test: hunks should not be included
	assert.Nil(t, result.Hunks, "hunks should be nil in summary")

	// Test new file
	fd = &types.FileDiff{IsNew: true}
	result = convertFileDiffSummary(fd)
	assert.Equals(t, result.Status, api.FileStatus_FILE_STATUS_NEW, "status should be NEW")

	// Test deleted file
	fd = &types.FileDiff{IsDeleted: true}
	result = convertFileDiffSummary(fd)
	assert.Equals(t, result.Status, api.FileStatus_FILE_STATUS_DELETED, "status should be DELETED")

	// Test modified file (default)
	fd = &types.FileDiff{}
	result = convertFileDiffSummary(fd)
	assert.Equals(t, result.Status, api.FileStatus_FILE_STATUS_MODIFIED, "status should be MODIFIED")
}

func TestGetDiffsResponseTypes(t *testing.T) {
	// Test that the API types implement ProtoMessage
	var _ interface{ ProtoMessage() } = (*api.GetDiffsRequest)(nil)
	var _ interface{ ProtoMessage() } = (*api.GetDiffsResponse)(nil)
	var _ interface{ ProtoMessage() } = (*api.Diff)(nil)
	var _ interface{ ProtoMessage() } = (*api.FileDiff)(nil)
	var _ interface{ ProtoMessage() } = (*api.Hunk)(nil)
	var _ interface{ ProtoMessage() } = (*api.HunkStats)(nil)
	var _ interface{ ProtoMessage() } = (*api.Line)(nil)

	// Test getters on GetDiffsResponse
	resp := &api.GetDiffsResponse{
		State: "READY",
		Diff:  &api.Diff{Files: []*api.FileDiff{}},
	}
	assert.Equals(t, resp.GetState(), "READY", "state should be READY")
	assert.NotNil(t, resp.GetDiff(), "diff should not be nil")

	// Test nil receiver
	var nilResp *api.GetDiffsResponse
	assert.Equals(t, nilResp.GetState(), "", "nil receiver should return empty string")
	assert.Nil(t, nilResp.GetDiff(), "nil receiver should return nil diff")
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

package server

import (
	"testing"

	"github.com/radiospiel/critic/simple-go/assert"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/types"
)

func TestConvertDiff(t *testing.T) {
	// Test nil diff
	result := convertDiff(nil)
	assert.Nil(t, result, "nil diff should return nil")

	// Test empty diff
	diff := &types.Diff{Files: []*types.FileDiff{}}
	result = convertDiff(diff)
	assert.NotNil(t, result, "empty diff should not be nil")
	assert.Equals(t, len(result.Files), 0, "empty diff should have no files")

	// Test diff with files
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
							{Type: types.LineDeleted, Content: "deleted line", OldNum: 2, NewNum: 0},
							{Type: types.LineAdded, Content: "added line 1", OldNum: 0, NewNum: 2},
							{Type: types.LineAdded, Content: "added line 2", OldNum: 0, NewNum: 3},
						},
					},
				},
			},
		},
	}

	result = convertDiff(diff)
	assert.NotNil(t, result, "diff should not be nil")
	assert.Equals(t, len(result.Files), 1, "should have 1 file")

	file := result.Files[0]
	assert.Equals(t, file.OldPath, "old.go", "old path should match")
	assert.Equals(t, file.NewPath, "new.go", "new path should match")
	assert.Equals(t, file.IsRenamed, true, "is_renamed should be true")
	assert.Equals(t, len(file.Hunks), 1, "should have 1 hunk")

	hunk := file.Hunks[0]
	assert.Equals(t, hunk.OldStart, int32(1), "old_start should be 1")
	assert.Equals(t, hunk.OldLines, int32(5), "old_lines should be 5")
	assert.Equals(t, hunk.NewStart, int32(1), "new_start should be 1")
	assert.Equals(t, hunk.NewLines, int32(7), "new_lines should be 7")
	assert.Equals(t, hunk.Header, "@@ -1,5 +1,7 @@", "header should match")
	assert.Equals(t, hunk.Stats.Added, int32(3), "stats.added should be 3")
	assert.Equals(t, hunk.Stats.Deleted, int32(1), "stats.deleted should be 1")
	assert.Equals(t, len(hunk.Lines), 4, "should have 4 lines")

	// Check line types
	assert.Equals(t, hunk.Lines[0].Type, "context", "first line should be context")
	assert.Equals(t, hunk.Lines[1].Type, "deleted", "second line should be deleted")
	assert.Equals(t, hunk.Lines[2].Type, "added", "third line should be added")
	assert.Equals(t, hunk.Lines[3].Type, "added", "fourth line should be added")
}

func TestConvertFileDiff(t *testing.T) {
	// Test nil file diff
	result := convertFileDiff(nil)
	assert.Nil(t, result, "nil file diff should return nil")

	// Test file diff with all fields
	fd := &types.FileDiff{
		OldPath:   "path/to/old.go",
		NewPath:   "path/to/new.go",
		OldMode:   "100644",
		NewMode:   "100755",
		IsNew:     false,
		IsDeleted: false,
		IsRenamed: true,
		IsBinary:  false,
	}

	result = convertFileDiff(fd)
	assert.NotNil(t, result, "result should not be nil")
	assert.Equals(t, result.OldPath, "path/to/old.go", "old path should match")
	assert.Equals(t, result.NewPath, "path/to/new.go", "new path should match")
	assert.Equals(t, result.OldMode, "100644", "old mode should match")
	assert.Equals(t, result.NewMode, "100755", "new mode should match")
	assert.Equals(t, result.IsNew, false, "is_new should be false")
	assert.Equals(t, result.IsDeleted, false, "is_deleted should be false")
	assert.Equals(t, result.IsRenamed, true, "is_renamed should be true")
	assert.Equals(t, result.IsBinary, false, "is_binary should be false")
}

func TestConvertHunk(t *testing.T) {
	// Test nil hunk
	result := convertHunk(nil)
	assert.Nil(t, result, "nil hunk should return nil")

	// Test hunk with stats
	h := &types.Hunk{
		OldStart: 10,
		OldLines: 20,
		NewStart: 15,
		NewLines: 25,
		Header:   "@@ -10,20 +15,25 @@",
		Stats:    types.HunkStats{Added: 10, Deleted: 5},
	}

	result = convertHunk(h)
	assert.NotNil(t, result, "result should not be nil")
	assert.Equals(t, result.OldStart, int32(10), "old_start should be 10")
	assert.Equals(t, result.OldLines, int32(20), "old_lines should be 20")
	assert.Equals(t, result.NewStart, int32(15), "new_start should be 15")
	assert.Equals(t, result.NewLines, int32(25), "new_lines should be 25")
	assert.NotNil(t, result.Stats, "stats should not be nil")
	assert.Equals(t, result.Stats.Added, int32(10), "stats.added should be 10")
	assert.Equals(t, result.Stats.Deleted, int32(5), "stats.deleted should be 5")
}

func TestConvertLine(t *testing.T) {
	// Test nil line
	result := convertLine(nil)
	assert.Nil(t, result, "nil line should return nil")

	// Test context line
	l := &types.Line{Type: types.LineContext, Content: "context", OldNum: 5, NewNum: 5}
	result = convertLine(l)
	assert.Equals(t, result.Type, "context", "type should be context")
	assert.Equals(t, result.Content, "context", "content should match")
	assert.Equals(t, result.OldNum, int32(5), "old_num should be 5")
	assert.Equals(t, result.NewNum, int32(5), "new_num should be 5")

	// Test added line
	l = &types.Line{Type: types.LineAdded, Content: "added", OldNum: 0, NewNum: 6}
	result = convertLine(l)
	assert.Equals(t, result.Type, "added", "type should be added")

	// Test deleted line
	l = &types.Line{Type: types.LineDeleted, Content: "deleted", OldNum: 6, NewNum: 0}
	result = convertLine(l)
	assert.Equals(t, result.Type, "deleted", "type should be deleted")
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

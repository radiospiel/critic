package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/radiospiel/critic/simple-go/assert"
)

// findCategory returns the patterns for a named category, or nil if not found.
func findCategory(categories []FileCategory, name string) []string {
	for _, cat := range categories {
		if cat.Name == name {
			return cat.Patterns
		}
	}
	return nil
}

func TestParseProjectConfig(t *testing.T) {
	yaml := `
project:
  name: "critic"

paths:
  - src/go

categories:
  - name: test
    patterns:
      - "*_test.go"
      - "/test/*"
  - name: hidden
    patterns:
      - ".*"

editor:
  url: "vscode://file/{file}:{line}"
`
	config, err := ParseProjectConfig([]byte(yaml))
	assert.NoError(t, err)
	assert.Equals(t, config.Project.Name, "critic", "project name")
	assert.Equals(t, len(config.Paths), 1, "paths count")
	assert.Equals(t, config.Paths[0], "src/go", "paths[0]")
	assert.Equals(t, len(config.Categories), 2, "categories count")
	testPatterns := findCategory(config.Categories, "test")
	assert.Equals(t, len(testPatterns), 2, "test patterns count")
	assert.Equals(t, testPatterns[0], "*_test.go", "test pattern 0")
	assert.Equals(t, testPatterns[1], "/test/*", "test pattern 1")
	hiddenPatterns := findCategory(config.Categories, "hidden")
	assert.Equals(t, len(hiddenPatterns), 1, "hidden patterns count")
	assert.Equals(t, hiddenPatterns[0], ".*", "hidden pattern 0")
	assert.Equals(t, config.Editor.URL, "vscode://file/{file}:{line}", "editor url")
}

func TestParseProjectConfig_Minimal(t *testing.T) {
	yaml := `
project:
  name: "myproject"
`
	config, err := ParseProjectConfig([]byte(yaml))
	assert.NoError(t, err)
	assert.Equals(t, config.Project.Name, "myproject", "project name")
	assert.Nil(t, config.Paths, "paths should be nil")
	assert.Nil(t, config.Categories, "categories should be nil")
	assert.Equals(t, config.Editor.URL, "", "editor url should be empty")
}

func TestParseProjectConfig_InvalidYAML(t *testing.T) {
	yaml := `{{{invalid yaml`
	_, err := ParseProjectConfig([]byte(yaml))
	assert.True(t, err != nil, "should return error for invalid YAML")
}

func TestDefaultProjectConfig(t *testing.T) {
	config := DefaultProjectConfig()
	assert.NotNil(t, config, "default config should not be nil")
	testPatterns := findCategory(config.Categories, "test")
	assert.Equals(t, len(testPatterns), 1, "default test patterns")
	assert.Equals(t, testPatterns[0], "*_test.go", "default test pattern")
	hiddenPatterns := findCategory(config.Categories, "hidden")
	assert.Equals(t, len(hiddenPatterns), 1, "default hidden patterns")
	assert.Equals(t, hiddenPatterns[0], ".*", "default hidden pattern")
}

func TestLoadProjectConfig_FileNotFound(t *testing.T) {
	dir := t.TempDir()
	_, pc, err := LoadProjectConfig("", dir)
	assert.NoError(t, err, "missing file should return default config")
	assert.NotNil(t, pc, "should return default config")
	testPatterns := findCategory(pc.Categories, "test")
	assert.Equals(t, len(testPatterns), 1, "should have default test patterns")
}

func TestLoadProjectConfig_ExplicitFileNotFound(t *testing.T) {
	_, _, err := LoadProjectConfig("/nonexistent/file.critic", "")
	assert.Error(t, err, "no such file")
}

func TestLoadProjectConfig_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "project.critic")
	yaml := `
project:
  name: "test-project"

paths:
  - src

categories:
  - name: test
    patterns:
      - "*_test.go"
  - name: hidden
    patterns:
      - ".*"
      - "vendor/**"

editor:
  url: "idea://open?file={file}&line={line}"
`
	err := os.WriteFile(path, []byte(yaml), 0644)
	assert.NoError(t, err)

	_, config, err := LoadProjectConfig("", dir)
	assert.NoError(t, err)
	assert.Equals(t, config.Project.Name, "test-project", "project name")
	assert.Equals(t, len(config.Paths), 1, "paths count")
	hiddenPatterns := findCategory(config.Categories, "hidden")
	assert.Equals(t, len(hiddenPatterns), 2, "hidden patterns count")
	assert.Equals(t, config.Editor.URL, "idea://open?file={file}&line={line}", "editor url")
}

func TestCategorizeFile(t *testing.T) {
	config := &ProjectConfig{
		Categories: []FileCategory{
			{Name: "test", Patterns: []string{"*_test.go", "/test/*"}},
			{Name: "hidden", Patterns: []string{".*"}},
		},
	}

	tests := []struct {
		path     string
		expected string
	}{
		{"src/main.go", "source"},
		{"src/main_test.go", "test"},
		{"test/fixture.go", "test"},
		{".gitignore", "hidden"},
		{".env", "hidden"},
		{"src/.hidden", "hidden"},
		{"README.md", "source"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := config.CategorizeFile(tt.path)
			assert.Equals(t, result, tt.expected, "CategorizeFile(%q)", tt.path)
		})
	}
}

func TestGetFileCategories(t *testing.T) {
	config := &ProjectConfig{
		Categories: []FileCategory{
			{Name: "test", Patterns: []string{"*_test.go"}},
			{Name: "hidden", Patterns: []string{".*"}},
		},
	}

	categories := config.GetFileCategories()
	assert.Equals(t, len(categories), 2, "should have 2 categories")

	found := map[string]bool{}
	for _, cat := range categories {
		found[cat.Name] = true
	}
	assert.True(t, found["test"], "should have test category")
	assert.True(t, found["hidden"], "should have hidden category")
}

// TestCategorizeFile_ProjectCritic tests categorization using the actual categories
// from this project's project.critic config. This ensures the backend matches what
// the frontend previously computed.
func TestCategorizeFile_ProjectCritic(t *testing.T) {
	// Mirror of the categories in project.critic
	config := &ProjectConfig{
		Categories: []FileCategory{
			{Name: "Backend", Patterns: []string{"src/**/*.go", "!*_test.go"}},
			{Name: "Backend Tests", Patterns: []string{"*_test.go"}},
			{Name: "Web UI", Patterns: []string{"src/webui/**/*"}},
			{Name: "VSCode UI", Patterns: []string{"editors/vscode/**/*"}},
			{Name: "integration tests", Patterns: []string{"/test/*"}},
			{Name: "auto generated", Patterns: []string{"src/webui/frontend/src/gen/*.ts", "package-lock.json", "src/api/critic.pb.go"}},
			{Name: "Agent Integration", Patterns: []string{"agents/**/*"}},
			{Name: "hidden", Patterns: []string{".*"}},
		},
	}

	tests := []struct {
		path     string
		expected string
	}{
		// Backend Go source files
		{"src/config/project.go", "Backend"},
		{"src/config/pathspec.go", "Backend"},
		{"src/api/server/server.go", "Backend"},
		{"src/api/server/get_diff_summary.go", "Backend"},
		{"src/git/diff.go", "Backend"},
		{"src/pkg/types/diff.go", "Backend"},

		// Backend test files — negation in "Backend" excludes *_test.go,
		// so they fall through to "Backend Tests"
		{"src/config/project_test.go", "Backend Tests"},
		{"src/config/pathspec_test.go", "Backend Tests"},
		{"src/api/server/get_diff_summary_test.go", "Backend Tests"},
		{"src/git/diff_test.go", "Backend Tests"},

		// Test files at any depth
		{"simple-go/fnmatch/fnmatch_test.go", "Backend Tests"},

		// Web UI — matches "src/webui/**/*"
		// Note: Web UI comes after Backend, so Go files under src/webui/ match Backend first
		{"src/webui/frontend/src/App.tsx", "Web UI"},
		{"src/webui/frontend/src/components/FileList.tsx", "Web UI"},
		{"src/webui/frontend/package.json", "Web UI"},

		// Web UI Go files match Backend (higher priority) not Web UI
		{"src/webui/webui.go", "Backend"},

		// VSCode UI
		{"editors/vscode/src/extension.ts", "VSCode UI"},
		{"editors/vscode/package.json", "VSCode UI"},

		// Integration tests — rooted pattern /test/*
		{"test/integration.go", "integration tests"},

		// Auto generated files — note: files under src/webui/ match "Web UI" first,
		// and src/api/*.go matches "Backend" first, because categories are checked
		// in order. Only package-lock.json uniquely matches "auto generated".
		{"src/webui/frontend/src/gen/critic_pb.ts", "Web UI"},
		{"src/webui/frontend/src/gen/critic_connect.ts", "Web UI"},
		{"package-lock.json", "auto generated"},
		{"src/api/critic.pb.go", "Backend"},

		// Agent Integration
		{"agents/logs/some-log.md", "Agent Integration"},
		{"agents/strategy-guide.md", "Agent Integration"},

		// Hidden (dot files)
		{".gitignore", "hidden"},
		{".env", "hidden"},
		{"src/.hidden", "hidden"},

		// Default "source" — no category matches
		{"README.md", "source"},
		{"Makefile", "source"},
		{"go.mod", "source"},
		{"go.sum", "source"},
		{"project.critic", "source"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := config.CategorizeFile(tt.path)
			assert.Equals(t, result, tt.expected, "CategorizeFile(%q)", tt.path)
		})
	}
}

// TestCategorizeFile_NegationWithDoublestar tests that negation patterns work
// correctly with doublestar patterns, which is the key combination used in
// the Backend category: "src/**/*.go" + "!*_test.go"
func TestCategorizeFile_NegationWithDoublestar(t *testing.T) {
	config := &ProjectConfig{
		Categories: []FileCategory{
			{Name: "impl", Patterns: []string{"src/**/*.go", "!*_test.go"}},
			{Name: "test", Patterns: []string{"*_test.go"}},
		},
	}

	tests := []struct {
		path     string
		expected string
	}{
		{"src/main.go", "impl"},
		{"src/pkg/deep/file.go", "impl"},
		{"src/main_test.go", "test"},
		{"src/pkg/deep/file_test.go", "test"},
		// Non-Go files don't match either
		{"src/README.md", "source"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := config.CategorizeFile(tt.path)
			assert.Equals(t, result, tt.expected, "CategorizeFile(%q)", tt.path)
		})
	}
}

// TestCategorizeFile_CategoryOrder tests that category ordering matters:
// the first matching category wins.
func TestCategorizeFile_CategoryOrder(t *testing.T) {
	config := &ProjectConfig{
		Categories: []FileCategory{
			{Name: "first", Patterns: []string{"*.go"}},
			{Name: "second", Patterns: []string{"*.go"}},
		},
	}

	result := config.CategorizeFile("main.go")
	assert.Equals(t, result, "first", "first matching category should win")
}

// TestCategorizeFile_EmptyCategories tests behavior with no categories configured.
func TestCategorizeFile_EmptyCategories(t *testing.T) {
	config := &ProjectConfig{}
	result := config.CategorizeFile("anything.go")
	assert.Equals(t, result, "source", "should return 'source' when no categories")
}

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/radiospiel/critic/simple-go/assert"
)

func TestParseProjectConfig(t *testing.T) {
	yaml := `
project:
  name: "critic"

paths:
  - src/go

categories:
  test:
    - "*_test.go"
    - "/test/*"
  hidden:
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
	assert.Equals(t, len(config.Categories["test"]), 2, "test patterns count")
	assert.Equals(t, config.Categories["test"][0], "*_test.go", "test pattern 0")
	assert.Equals(t, config.Categories["test"][1], "/test/*", "test pattern 1")
	assert.Equals(t, len(config.Categories["hidden"]), 1, "hidden patterns count")
	assert.Equals(t, config.Categories["hidden"][0], ".*", "hidden pattern 0")
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
	assert.Equals(t, len(config.Categories["test"]), 1, "default test patterns")
	assert.Equals(t, config.Categories["test"][0], "*_test.go", "default test pattern")
	assert.Equals(t, len(config.Categories["hidden"]), 1, "default hidden patterns")
	assert.Equals(t, config.Categories["hidden"][0], ".*", "default hidden pattern")
}

func TestLoadProjectConfig_FileNotFound(t *testing.T) {
	config, err := LoadProjectConfig("/nonexistent/directory")
	assert.NoError(t, err, "missing file should return default config, not error")
	assert.NotNil(t, config, "should return default config")
	assert.Equals(t, len(config.Categories["test"]), 1, "should have default test patterns")
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
  test:
    - "*_test.go"
  hidden:
    - ".*"
    - "vendor/**"

editor:
  url: "idea://open?file={file}&line={line}"
`
	err := os.WriteFile(path, []byte(yaml), 0644)
	assert.NoError(t, err)

	config, err := LoadProjectConfig(dir)
	assert.NoError(t, err)
	assert.Equals(t, config.Project.Name, "test-project", "project name")
	assert.Equals(t, len(config.Paths), 1, "paths count")
	assert.Equals(t, len(config.Categories["hidden"]), 2, "hidden patterns count")
	assert.Equals(t, config.Editor.URL, "idea://open?file={file}&line={line}", "editor url")
}

func TestCategorizeFile(t *testing.T) {
	config := &ProjectConfig{
		Categories: map[string][]string{
			"test":   {"*_test.go", "/test/*"},
			"hidden": {".*"},
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
		Categories: map[string][]string{
			"test":   {"*_test.go"},
			"hidden": {".*"},
		},
	}

	categories := config.GetFileCategories()
	assert.Equals(t, len(categories), 2, "should have 2 categories")

	// Check that both categories are present (order may vary due to map)
	found := map[string]bool{}
	for _, cat := range categories {
		found[cat.Name] = true
	}
	assert.True(t, found["test"], "should have test category")
	assert.True(t, found["hidden"], "should have hidden category")
}

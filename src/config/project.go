package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FileCategory represents a named category of files with glob patterns.
// Categories are used to classify files in the diff view (e.g., test, hidden).
type FileCategory struct {
	Name     string   `yaml:"name" json:"name"`
	Patterns []string `yaml:"patterns" json:"patterns"`
}

// EditorConfig holds editor integration settings.
type EditorConfig struct {
	// URL is a template for opening files in an editor.
	// Supports placeholders: {file} for the file path, {line} for the line number.
	// Example: "vscode://file/{file}:{line}"
	URL string `yaml:"url" json:"url"`
}

// ProjectConfig represents the parsed project.critic configuration file.
type ProjectConfig struct {
	Project    projectInfo  `yaml:"project"`
	Paths      []string     `yaml:"paths"`
	Categories []FileCategory `yaml:"categories"`
	Editor     EditorConfig `yaml:"editor"`
}

type projectInfo struct {
	Name string `yaml:"name"`
}

// DefaultProjectConfig returns a ProjectConfig with sensible defaults.
// Used when no project.critic file is found.
func DefaultProjectConfig() *ProjectConfig {
	return &ProjectConfig{
		Project: projectInfo{Name: ""},
		Categories: []FileCategory{
			{Name: "test", Patterns: []string{"*_test.go"}},
			{Name: "hidden", Patterns: []string{".*"}},
		},
	}
}

// LoadProjectConfig loads and parses a project.critic YAML file from the given directory.
// Returns DefaultProjectConfig if the file does not exist.
func LoadProjectConfig(dir string) (*ProjectConfig, error) {
	path := filepath.Join(dir, "project.critic")
	return LoadProjectConfigFromFile(path)
}

// LoadProjectConfigFromFile loads and parses a project.critic YAML file from the given path.
// Returns DefaultProjectConfig if the file does not exist.
func LoadProjectConfigFromFile(path string) (*ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultProjectConfig(), nil
		}
		return nil, fmt.Errorf("failed to read project config: %w", err)
	}

	return ParseProjectConfig(data)
}

// ParseProjectConfig parses YAML data into a ProjectConfig.
func ParseProjectConfig(data []byte) (*ProjectConfig, error) {
	var config ProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse project config YAML: %w", err)
	}

	return &config, nil
}

// GetFileCategories returns the list of FileCategory entries from the config.
func (c *ProjectConfig) GetFileCategories() []FileCategory {
	return c.Categories
}

// CategorizeFile returns the category name for a given file path.
// Categories are checked in order: the first matching category wins.
// Returns "source" if no category matches.
func (c *ProjectConfig) CategorizeFile(path string) string {
	for _, cat := range c.Categories {
		if PathspecMatchAny(cat.Patterns, path) {
			return cat.Name
		}
	}
	return "source"
}

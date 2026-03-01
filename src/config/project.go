package config

import (
	"fmt"
	"os"

	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/samber/lo"
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
	Project    projectInfo    `yaml:"project"`
	Paths      []string       `yaml:"paths"`
	Categories []FileCategory `yaml:"categories"`
	Editor     EditorConfig   `yaml:"editor"`
	DiffBases  []string       `yaml:"diffbases"`
	ConfigPath string         `yaml:"-"`
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
		DiffBases: []string{"green"},
	}
}

// LoadProjectConfig loads the project config from path, falling back to defaults if missing.
//
// currentBranch is the current git branch name (used to build default bases).
// hasRef validates whether a diff base is a valid git ref.
// The returned config's DiffBases will be the merged result of defaults and configured bases.
func LoadProjectConfig(path, currentBranch string, hasRef func(string) bool) (*ProjectConfig, error) {
	pc, err := loadProjectConfigFromFile(path)
	if err != nil {
		return nil, err
	}

	if hasRef != nil {
		allDiffBases := append([]string{"main", "master", "origin/" + currentBranch, "HEAD"}, pc.DiffBases...)
		unique := lo.Uniq(allDiffBases)
		pc.DiffBases = lo.Filter(unique, func(ref string, _ int) bool { return hasRef(ref) })
	}

	logger.Info("Loading critic configuration from %s", pc.ConfigPath)
	return pc, nil
}

// loadProjectConfigFromFile loads a project config from the given path.
// If the file cannot be read, it logs the error and returns DefaultProjectConfig.
// If the file cannot be parsed, it returns the error
func loadProjectConfigFromFile(path string) (*ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Error("Cannot load critic configuration from %s: %v, falling back to hardcoded default", path, err)
		return DefaultProjectConfig(), nil
	}

	pc, err := ParseProjectConfig(data)
	if err != nil {
		logger.Error("Cannot parse critic configuration from %s: %v", path, err)
		return nil, err
	}

	pc.ConfigPath = path
	return pc, nil
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

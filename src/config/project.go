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
	Path     string   `yaml:"path,omitempty" json:"path,omitempty"`
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
	DiffBase   string         `yaml:"diffbase"`
	DiffBases  []string       `yaml:"-"` // computed: diff base + discovered branches
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
	}
}

// GitOps provides git operations needed for diff base discovery.
// When nil is passed to LoadProjectConfig, no git operations are performed.
type GitOps struct {
	HasRef              func(string) bool
	ResolveRef          func(string) string
	LocalBranchesOnPath func(string) []string
}

// LoadProjectConfig loads the project config from path, falling back to defaults if missing.
//
// currentBranch is the current git branch name.
// gitOps provides git operations for ref validation and branch discovery.
// When gitOps is nil, DiffBases are left empty.
//
// The DiffBase is resolved as follows:
//  1. If configured in project.critic, use that value (must exist as a git ref).
//  2. Otherwise, detect "master" or "main" (whichever exists).
//  3. If neither exists, return an error.
//
// DiffBases is then populated with the diff base plus any local branches
// discovered on the path from the diff base to HEAD.
func LoadProjectConfig(path, currentBranch string, gitOps *GitOps) (*ProjectConfig, error) {
	pc, err := loadProjectConfigFromFile(path)
	if err != nil {
		return nil, err
	}

	if gitOps != nil && gitOps.HasRef != nil {
		// Resolve the single diff base
		diffBase := pc.DiffBase
		if diffBase == "" {
			// Auto-detect: try "master" then "main"
			for _, candidate := range []string{"master", "main"} {
				if gitOps.HasRef(candidate) {
					diffBase = candidate
					break
				}
			}
			if diffBase == "" {
				return nil, fmt.Errorf("no diff base configured and neither 'master' nor 'main' branch exists")
			}
		} else if !gitOps.HasRef(diffBase) {
			return nil, fmt.Errorf("configured diff base %q does not exist", diffBase)
		}

		pc.DiffBase = diffBase

		// Start with the diff base itself
		candidates := []string{diffBase}

		// Discover local branches on the path from diff base to HEAD
		if gitOps.LocalBranchesOnPath != nil {
			discovered := gitOps.LocalBranchesOnPath(diffBase)
			candidates = append(candidates, discovered...)
		}

		// Deduplicate by resolved SHA
		if gitOps.ResolveRef != nil {
			seenSHA := make(map[string]bool)
			candidates = lo.Filter(candidates, func(ref string, _ int) bool {
				sha := gitOps.ResolveRef(ref)
				if seenSHA[sha] {
					return false
				}
				seenSHA[sha] = true
				return true
			})
		}

		pc.DiffBases = candidates
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

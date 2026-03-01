package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/peterh/liner"
	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/src/git"
	"github.com/spf13/cobra"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// newREPLCmd creates the repl subcommand
func newREPLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repl",
		Short: "Start an interactive Go REPL",
		Long: `Start an interactive Go REPL (Read-Eval-Print Loop).

The REPL uses the yaegi Go interpreter to evaluate Go code interactively.
You can use standard Go syntax, the standard library, and the critic git package.

Examples:
  critic repl                    # Start the REPL

Standard Go expressions:
  1 + 1
  x := 42
  import "fmt"; fmt.Println(x)

Git package (import "github.com/radiospiel/critic/src/git"):
  git.GetGitRoot()                           - Get the root of the git repo
  git.GetCurrentBranch()                     - Get the current branch name
  git.IsGitRepo()                            - Check if in a git repo
  git.HasRef(ref)                            - Check if a ref exists
  git.ResolveRef(ref)                        - Resolve a ref to a SHA
  git.GetDiff(base, path, ctx)               - Get diff for a file
  git.GetDiffNames(base, paths)              - Get changed file names
  git.DiffNames(base, paths)                 - Get changed file names as []string
  git.GetFileContent(path, rev)              - Get file content at revision
  git.GetLineContext(path, line, ref)        - Get context around a line
  git.ParseDiff(text)                        - Parse a unified diff
  git.AbsPathToGitPath(path)                 - Convert abs path to git-relative

Use Ctrl+D or type "exit" to quit. Type "help" for usage info.
`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runREPL(cmd)
		},
	}

	return cmd
}

const (
	colorCyan      = "\033[36m"
	colorBoldWhite = "\033[1;37m"
	colorRed       = "\033[31m"
	colorReset     = "\033[0m"
)

func runREPL(cmd *cobra.Command) error {
	// Make log output cyan and disable timestamps in REPL mode
	logger.SetLevelColor(logger.INFO, colorCyan)
	logger.SetLogFlags(0)

	// Create yaegi interpreter with proper GoPath
	// yaegi needs a valid GoPath to find standard library packages
	i := interp.New(interp.Options{
		Stdout: cmd.OutOrStdout(),
		Stderr: cmd.ErrOrStderr(),
	})

	// Use standard library
	if err := i.Use(stdlib.Symbols); err != nil {
		return fmt.Errorf("failed to use stdlib: %w", err)
	}

	// Register git package symbols for use in the REPL.
	// The yaegi export key must be "importpath/packagename".
	if err := i.Use(interp.Exports{
		"github.com/radiospiel/critic/src/git/git": map[string]reflect.Value{
			"GetGitRoot":                reflect.ValueOf(git.GetGitRoot),
			"GetCurrentBranch":          reflect.ValueOf(git.GetCurrentBranch),
			"IsGitRepo":                 reflect.ValueOf(git.IsGitRepo),
			"HasRef":                    reflect.ValueOf(git.HasRef),
			"ResolveRef":                reflect.ValueOf(git.ResolveRef),
			"GetDiff":                   reflect.ValueOf(git.GetDiff),
			"GetDiffNames":              reflect.ValueOf(git.GetDiffNames),
			"GetFileContent":            reflect.ValueOf(git.GetFileContent),
			"BuildLineDisplacement":     reflect.ValueOf(git.BuildLineDisplacement),
			"GetLineContext":            reflect.ValueOf(git.GetLineContext),
			"ParseDiff":                 reflect.ValueOf(git.ParseDiff),
			"ParseDiffNameStatus":       reflect.ValueOf(git.ParseDiffNameStatus),
			"ClosestBranchForSHA":       reflect.ValueOf(git.ClosestBranchForSHA),
			"ClosestBranchForSHACached": reflect.ValueOf(git.ClosestBranchForSHACached),
			"IsCommitInRange":           reflect.ValueOf(git.IsCommitInRange),
			"IsCommitInRangeCached":     reflect.ValueOf(git.IsCommitInRangeCached),
			"NewAncestryCache":          reflect.ValueOf(git.NewAncestryCache),
			"AbsPathToGitPath":          reflect.ValueOf(git.AbsPathToGitPath),
			"WithEnd":                   reflect.ValueOf(git.WithEnd),
			"NewGitWatcher":             reflect.ValueOf(git.NewGitWatcher),
			"DiffNames":                reflect.ValueOf(replDiffNames),
		},
	}); err != nil {
		return fmt.Errorf("failed to register packages: %w", err)
	}

	// Auto-import commonly used packages
	i.Eval(`import "fmt"`)
	i.Eval(`import "github.com/radiospiel/critic/src/git"`)

	// Set up liner for readline-like behavior
	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)

	// Load history from file
	historyPath := replHistoryPath()
	if f, err := os.Open(historyPath); err == nil {
		line.ReadHistory(f)
		f.Close()
	}

	fmt.Fprint(cmd.OutOrStdout(), `Critic REPL - Go interpreter
Pre-imported: fmt, git

  git.GetGitRoot()
  git.GetCurrentBranch()
  git.DiffNames("HEAD", []string{"."})
  git.GetFileContent("README.md", "HEAD")

Type 'help' for all functions, 'exit' or Ctrl+D to quit

`)

	for {
		fmt.Fprint(cmd.OutOrStdout(), colorBoldWhite)
		input, err := line.Prompt("> ")
		fmt.Fprint(cmd.OutOrStdout(), colorReset)
		if err == liner.ErrPromptAborted || err == io.EOF {
			fmt.Fprintln(cmd.OutOrStdout())
			break
		}
		if err != nil {
			return fmt.Errorf("prompt error: %w", err)
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		line.AppendHistory(input)

		if input == "exit" || input == "quit" {
			break
		}
		if input == "help" {
			printREPLHelp(cmd)
			continue
		}

		// Evaluate the input
		v, err := i.Eval(input)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "%sError: %v%s\n", colorRed, err, colorReset)
			continue
		}

		if v.IsValid() {
			if s := formatREPLValue(v); s != "" {
				fmt.Fprintln(cmd.OutOrStdout(), s)
			}
		}
	}

	// Save history
	if f, err := os.Create(historyPath); err == nil {
		line.WriteHistory(f)
		f.Close()
	}

	return nil
}

// formatREPLValue formats a reflect.Value for display.
// Primitives and strings print as-is; everything else is JSON-marshaled.
func formatREPLValue(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}

	// Dereference pointers
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}

	iface := v.Interface()

	switch v.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String:
		return fmt.Sprintf("%v", iface)
	case reflect.Func, reflect.Chan:
		return ""
	}

	// For everything else, try JSON
	b, err := json.MarshalIndent(iface, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", iface)
	}
	return string(b)
}

// replDiffNames is a REPL-friendly wrapper around GetDiffNames that returns []string.
func replDiffNames(base string, paths []string, opts ...git.DiffOption) ([]string, error) {
	diffs, err := git.GetDiffNames(base, paths, opts...)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(diffs))
	for i, d := range diffs {
		names[i] = d.NewPath
	}
	return names, nil
}

func replHistoryPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	dir := filepath.Join(home, ".critic")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "repl_history")
}

func printREPLHelp(cmd *cobra.Command) {
	w := cmd.OutOrStdout()
	fmt.Fprintln(w, `Git package usage:
  import "github.com/radiospiel/critic/src/git"

Available functions:
  git.GetGitRoot() string                                      - Git repo root path
  git.GetCurrentBranch() string                                - Current branch name
  git.IsGitRepo() bool                                         - Check if in a git repo
  git.HasRef(ref string) bool                                  - Check if a ref exists
  git.ResolveRef(ref string) string                            - Resolve ref to SHA
  git.GetDiff(base, path string, ctx int, ...DiffOption)       - Get file diff
  git.GetDiffNames(base string, paths []string, ...DiffOption) - Get changed file list
  git.DiffNames(base string, paths []string, ...DiffOption)    - Changed files as []string
  git.GetFileContent(path, revision string) (string, error)    - File content at rev
  git.GetLineContext(path string, line int, ref string) string  - Context around a line
  git.BuildLineDisplacement(path, ref1, ref2 string)           - Line displacement map
  git.ParseDiff(text string)                                   - Parse unified diff
  git.ParseDiffNameStatus(output string)                       - Parse name-status output
  git.AbsPathToGitPath(path string) string                     - Abs path to git-relative
  git.WithEnd(end string) DiffOption                           - Diff option for end ref
  git.NewAncestryCache() AncestryCache                         - Create ancestry cache
  git.ClosestBranchForSHA(sha string, refs []string) string    - Find closest branch
  git.IsCommitInRange(sha, start, end string) bool             - Check commit in range
  git.NewGitWatcher(gitDir string, debounceMs int)             - Watch for git changes

Example:
  import "github.com/radiospiel/critic/src/git"
  git.GetGitRoot()
  git.GetCurrentBranch()
  git.HasRef("main")
  git.GetFileContent("README.md", "HEAD")`)
}


package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"git.15b.it/eno/critic/simple-go/logger"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

// newLogCmd creates the log subcommand
func newLogCmd() *cobra.Command {
	var follow bool
	var topic string
	var ignoreCase bool
	var quietFlag int

	cmd := &cobra.Command{
		Use:   "log [filter...]",
		Short: "Watch and display the critic log file",
		Long: `Watch and display the critic log file.

By default, outputs the current log file path. Use -f to follow the log
file and print new output as it arrives.

Filter expressions can be plain strings or regex patterns (enclosed in //).
Multiple filters are ANDed together (all must match).

Examples:
  critic log                     # Print the log file path
  critic log -f                  # Follow the log file (like tail -f)
  critic log -f -t git           # Follow and filter for [git] topic
  critic log -f ERROR            # Follow and filter for lines containing "ERROR"
  critic log -f -i error         # Case-insensitive filter for "error"
  critic log -f "/error|warn/"   # Filter using regex
  critic log -f ERROR -t git     # Multiple filters (AND)
  critic log -f -q               # Show only WARN, ERROR, FATAL
  critic log -f -qq              # Show only ERROR, FATAL
`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			logPath := logger.LogFilePath()

			if !follow {
				fmt.Println(logPath)
				return nil
			}

			// Build filters from args and topic flag
			var filters []string
			filters = append(filters, args...)
			if topic != "" {
				filters = append(filters, "["+topic+"]")
			}

			return watchLogFile(logPath, filters, ignoreCase, quietFlag)
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow the log file output")
	cmd.Flags().StringVarP(&topic, "topic", "t", "", "Filter log output by topic (e.g., -t git shows only [git] entries)")
	cmd.Flags().BoolVarP(&ignoreCase, "ignore-case", "i", false, "Case-insensitive filtering")
	cmd.Flags().CountVarP(&quietFlag, "quiet", "q", "Filter by log level (-q for WARN+, -qq for ERROR+)")

	return cmd
}

// filter represents a compiled filter (either string or regex)
type filter struct {
	pattern    string
	regex      *regexp.Regexp
	ignoreCase bool
}

// newFilter creates a filter from a pattern string
// Patterns enclosed in // are treated as regex
func newFilter(pattern string, ignoreCase bool) (*filter, error) {
	f := &filter{
		pattern:    pattern,
		ignoreCase: ignoreCase,
	}

	// Check if it's a regex pattern (enclosed in //)
	if strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/") && len(pattern) > 2 {
		regexPattern := pattern[1 : len(pattern)-1]
		if ignoreCase {
			regexPattern = "(?i)" + regexPattern
		}
		re, err := regexp.Compile(regexPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex %s: %w", pattern, err)
		}
		f.regex = re
	}

	return f, nil
}

// matches returns true if the line matches this filter
func (f *filter) matches(line string) bool {
	if f.regex != nil {
		return f.regex.MatchString(line)
	}

	// Plain string matching
	if f.ignoreCase {
		return strings.Contains(strings.ToLower(line), strings.ToLower(f.pattern))
	}
	return strings.Contains(line, f.pattern)
}

// watchLogFile watches the log file and prints new content to stdout
// All filters must match for a line to be printed (AND logic)
// quietLevel filters by log level: 0=all, 1=WARN+, 2=ERROR+, 3+=FATAL only
func watchLogFile(logPath string, filterPatterns []string, ignoreCase bool, quietLevel int) error {
	// Compile filters
	var filters []*filter
	for _, pattern := range filterPatterns {
		f, err := newFilter(pattern, ignoreCase)
		if err != nil {
			return err
		}
		filters = append(filters, f)
	}

	// Build level filter based on quiet flag
	var minLevel logger.Level
	switch quietLevel {
	case 0:
		minLevel = logger.DEBUG
	case 1:
		minLevel = logger.WARN
	case 2:
		minLevel = logger.ERROR
	default:
		minLevel = logger.FATAL
	}

	// Open the log file
	file, err := os.Open(logPath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Seek to end of file to only show new content
	_, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("failed to seek to end of file: %w", err)
	}

	// Create fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	// Watch the log file
	err = watcher.Add(logPath)
	if err != nil {
		return fmt.Errorf("failed to watch log file: %w", err)
	}

	reader := bufio.NewReader(file)

	// Display watching message
	if len(filters) > 0 {
		var filterDescs []string
		for _, f := range filters {
			filterDescs = append(filterDescs, f.pattern)
		}
		fmt.Fprintf(os.Stderr, "Watching %s for: %s...\n", logPath, strings.Join(filterDescs, " AND "))
	} else {
		fmt.Fprintf(os.Stderr, "Watching %s...\n", logPath)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				// Read and print new lines
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						break
					}
					// Check if line matches log level and all filters (AND logic)
					if matchesLogLevel(line, minLevel) && matchesAllFilters(line, filters) {
						fmt.Print(line)
					}
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			return fmt.Errorf("watcher error: %w", err)
		}
	}
}

// matchesLogLevel returns true if the line's log level is >= minLevel
// Log lines contain level prefixes like "INFO:", "WARN:", "ERROR:", "FATAL:", "DEBUG:"
func matchesLogLevel(line string, minLevel logger.Level) bool {
	// If no filtering (DEBUG level), accept all lines
	if minLevel == logger.DEBUG {
		return true
	}

	// Extract log level from line
	lineLevel := logger.DEBUG // Default to lowest level if not found
	if strings.Contains(line, "FATAL:") {
		lineLevel = logger.FATAL
	} else if strings.Contains(line, "ERROR:") {
		lineLevel = logger.ERROR
	} else if strings.Contains(line, "WARN:") {
		lineLevel = logger.WARN
	} else if strings.Contains(line, "INFO:") {
		lineLevel = logger.INFO
	} else if strings.Contains(line, "DEBUG:") {
		lineLevel = logger.DEBUG
	}

	return lineLevel >= minLevel
}

// matchesAllFilters returns true if the line matches all filters
func matchesAllFilters(line string, filters []*filter) bool {
	for _, f := range filters {
		if !f.matches(line) {
			return false
		}
	}
	return true
}

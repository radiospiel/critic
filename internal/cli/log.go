package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"git.15b.it/eno/critic/simple-go/logger"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

// newLogCmd creates the log subcommand
func newLogCmd() *cobra.Command {
	var follow bool
	var topic string

	cmd := &cobra.Command{
		Use:   "log",
		Short: "Watch and display the critic log file",
		Long: `Watch and display the critic log file.

By default, outputs the current log file path. Use -f to follow the log
file and print new output as it arrives. Use --topic to filter by topic.

Examples:
  critic log                  # Print the log file path
  critic log -f               # Follow the log file (like tail -f)
  critic log -f --topic git   # Follow and filter for [git] topic
`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			logPath := logger.LogFilePath()

			if !follow {
				fmt.Println(logPath)
				return nil
			}

			return watchLogFile(logPath, topic)
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow the log file output")
	cmd.Flags().StringVar(&topic, "topic", "", "Filter log output by topic (e.g., --topic git shows only [git] entries)")

	return cmd
}

// watchLogFile watches the log file and prints new content to stdout
// If topic is non-empty, only lines containing [topic] are printed
func watchLogFile(logPath string, topic string) error {
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

	// Build topic filter if specified
	var topicFilter string
	if topic != "" {
		topicFilter = "[" + topic + "]"
		fmt.Fprintf(os.Stderr, "Watching %s for topic %s...\n", logPath, topicFilter)
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
					// Filter by topic if specified
					if topicFilter == "" || strings.Contains(line, topicFilter) {
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

package cli

import (
	"fmt"
	"strings"

	"git.15b.it/eno/critic/internal/webui"
	"github.com/spf13/cobra"
)

// newWebUICmd creates the webui subcommand
func newWebUICmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "webui [flags] [base1,base2,...] [-- path1 path2 ...]",
		Short: "Start the web user interface",
		Long: `Start the web-based git diff viewer.

The web UI provides a browser-based interface for viewing diffs and
managing code review conversations. It uses htmx for interactivity
and WebSockets for real-time updates.

Examples:
  critic webui                           # Start on default port 8080
  critic webui --port=3000               # Start on custom port
  critic webui main                      # Compare against main branch
  critic webui -- src tests              # Only show changes in specific paths
`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args: func(cmd *cobra.Command, args []string) error {
			argsLenAtDash := cmd.ArgsLenAtDash()
			if argsLenAtDash >= 0 {
				if argsLenAtDash > 1 {
					return fmt.Errorf("accepts at most 1 arg before --, received %d", argsLenAtDash)
				}
				return nil
			}
			if len(args) > 1 {
				return fmt.Errorf("accepts at most 1 arg, received %d", len(args))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			defer func() {
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Error: %v\n", err)
				}
			}()
			config := webui.Config{
				Port:  port,
				Paths: []string{"."},
			}

			argsLenAtDash := cmd.ArgsLenAtDash()
			var baseArg string
			if argsLenAtDash >= 0 {
				if argsLenAtDash > 0 {
					baseArg = args[0]
				}
				pathArgs := args[argsLenAtDash:]
				if len(pathArgs) > 0 {
					config.Paths = pathArgs
				}
			} else {
				if len(args) > 0 {
					baseArg = args[0]
				}
			}

			if baseArg != "" {
				config.Bases = strings.Split(baseArg, ",")
			}

			server, err := webui.NewServer(config)
			if err != nil {
				return fmt.Errorf("failed to create server: %w", err)
			}

			return server.Start()
		},
	}

	cmd.Flags().IntVar(&port, "port", 8080, "Port to run the web server on")

	return cmd
}

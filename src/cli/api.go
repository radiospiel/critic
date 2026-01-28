package cli

import (
	"fmt"

	"github.com/radiospiel/critic/src/api/server"
	"github.com/spf13/cobra"
)

// newAPICmd creates the api subcommand
func newAPICmd() *cobra.Command {
	var port int
	var dev bool

	cmd := &cobra.Command{
		Use:   "api [flags]",
		Short: "Start the HTTP API server",
		Long: `Start the HTTP API server using Connect.

The API server provides a programmatic interface for interacting with critic.
It uses Connect-RPC. Read more at https://connectrpc.com/

Examples:
  critic api                    # Start on default port 65432
  critic api --port=8000        # Start on custom port
  critic api --dev              # Development mode with Vite hot reload
`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			defer func() {
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Error: %v\n", err)
				}
			}()

			config := server.Config{
				Port: port,
				Dev:  dev,
			}

			srv := server.NewServer(config)
			return srv.Start()
		},
	}

	cmd.Flags().IntVar(&port, "port", 65432, "Port to run the API server on")
	cmd.Flags().BoolVar(&dev, "dev", false, "Development mode: proxy to Vite dev server for hot reload")

	return cmd
}

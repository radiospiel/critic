package cli

import (
	"fmt"

	"git.15b.it/eno/critic/src/api/server"
	"github.com/spf13/cobra"
)

// newAPICmd creates the api subcommand
func newAPICmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "api [flags]",
		Short: "Start the gRPC/HTTP API server",
		Long: `Start the gRPC/HTTP API server using Connect.

The API server provides a programmatic interface for interacting with critic.
It supports Connect, gRPC, and gRPC-Web protocols over HTTP.

Examples:
  critic api                    # Start on default port 65432
  critic api --port=8000        # Start on custom port
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
			}

			srv := server.NewServer(config)
			return srv.Start()
		},
	}

	cmd.Flags().IntVar(&port, "port", 65432, "Port to run the API server on")

	return cmd
}

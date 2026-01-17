package cli

import (
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"strings"

	"git.15b.it/eno/critic/internal/app"
	"github.com/spf13/cobra"
)

// newTUICmd creates the tui subcommand
func newTUICmd(handler func(*app.Args) error) *cobra.Command {
	var extensionsFlag []string
	var noAnimationFlag bool
	var cpuprofileFlag string

	cmd := &cobra.Command{
		Use:   "tui [flags] [base1,base2,...] [-- path1 path2 ...]",
		Short: "Start the terminal user interface",
		Long: `Start the terminal-based git diff viewer with side-by-side comparison.

Syntax:
  critic tui [base1,base2,base3] [-- path1 path2 path3]

Examples:
  critic tui                           # Compare against default bases (main/master, origin/<branch>, HEAD)
  critic tui main                      # Compare main branch to HEAD
  critic tui main,develop              # Compare against multiple bases
  critic tui -- src tests              # Only show changes in src and tests directories
  critic tui --extensions=go,rs        # Only show .go and .rs files
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

			// CPU profiling
			if cpuprofileFlag != "" {
				f, err := os.Create(cpuprofileFlag)
				if err != nil {
					log.Fatal(err)
				}
				pprof.StartCPUProfile(f)
				defer pprof.StopCPUProfile()
			}
			parsedArgs := &app.Args{
				Extensions:  ensureSlice(extensionsFlag),
				Paths:       []string{"."},
				NoAnimation: noAnimationFlag,
			}

			argsLenAtDash := cmd.ArgsLenAtDash()
			var baseArg string
			if argsLenAtDash >= 0 {
				if argsLenAtDash > 0 {
					baseArg = args[0]
				}
				pathArgs := args[argsLenAtDash:]
				if len(pathArgs) > 0 {
					parsedArgs.Paths = pathArgs
				}
			} else {
				if len(args) > 0 {
					baseArg = args[0]
				}
			}

			if baseArg != "" {
				parsedArgs.Bases = strings.Split(baseArg, ",")
			}

			return handler(parsedArgs)
		},
	}

	cmd.Flags().StringSliceVar(&extensionsFlag, "extensions", nil, "Comma-separated list of file extensions to include")
	cmd.Flags().BoolVar(&noAnimationFlag, "no-animation", false, "Disable animations")
	cmd.Flags().StringVar(&cpuprofileFlag, "cpuprofile", "", "Write CPU profile to file")

	return cmd
}

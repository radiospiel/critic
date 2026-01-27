package cli

import (
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"strings"

	"github.com/spf13/cobra"
	"github.org/radiospiel/critic/simple-go/logger"
	"github.org/radiospiel/critic/simple-go/preconditions"
	"github.org/radiospiel/critic/src/config"
	"github.org/radiospiel/critic/src/git"
	"github.org/radiospiel/critic/src/tui"
)

// newTUICmd creates the tui subcommand
func newTUICmd() *cobra.Command {
	var extensionsFlag []string
	var debugFlag bool
	var cpuprofileFlag string
	var quietFlag int

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

			// Set log level based on quiet flag
			switch quietFlag {
			case 1:
				logger.SetLevel(logger.WARN)
			case 2:
				logger.SetLevel(logger.ERROR)
			default:
				if quietFlag > 2 {
					logger.SetLevel(logger.FATAL)
				}
			}

			// CPU profiling
			if cpuprofileFlag != "" {
				f, err := os.Create(cpuprofileFlag)
				if err != nil {
					log.Fatal(err)
				}
				pprof.StartCPUProfile(f)
				defer pprof.StopCPUProfile()
			}
			parsedArgs := &tui.Args{
				Extensions: ensureSlice(extensionsFlag),
				Paths:      []string{"."},
				Debug:      debugFlag,
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
			} else {
				parsedArgs.Bases = getDefaultBases()
			}

			return runTui(parsedArgs)
		},
	}

	cmd.Flags().StringSliceVar(&extensionsFlag, "extensions", nil, "Comma-separated list of file extensions to include")
	cmd.Flags().BoolVar(&debugFlag, "debug", false, "Enable debug mode (shows UUIDs, etc.)")
	cmd.Flags().StringVar(&cpuprofileFlag, "cpuprofile", "", "Write CPU profile to file")
	cmd.Flags().CountVarP(&quietFlag, "quiet", "q", "Reduce log verbosity (-q for WARN, -qq for ERROR)")

	return cmd
}

// runTui runs the application with the given arguments
func runTui(args *tui.Args) error {
	logger.Info("=== Critic starting ===")

	// Check if we're in a git repository
	if !git.IsGitRepo() {
		return fmt.Errorf("not a git repository")
	}

	// Set default bases if none were specified
	preconditions.Check(len(args.Bases) > 0, "Must have args.Bases")

	// Set default extensions if none were specified
	if len(args.Extensions) == 0 {
		args.Extensions = config.DefaultFileExtensions
	}

	return tui.Run(args)
}

package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

// Debug flags for dumping intermediate representations
var (
	dParse   bool
	dC       bool
	dAsm     bool
	dClight  bool
	dCminor  bool
	dRTL     bool
	dLTL     bool
	dMach    bool
)

// debugFlagInfo holds metadata for a debug flag
type debugFlagInfo struct {
	flag *bool
	desc string
}

// debugFlags maps flag names to descriptions for unimplemented warnings
var debugFlags = map[string]debugFlagInfo{
	"dparse":  {&dParse, "dump after parsing"},
	"dc":      {&dC, "dump CompCert C"},
	"dasm":    {&dAsm, "dump assembly"},
	"dclight": {&dClight, "dump Clight"},
	"dcminor": {&dCminor, "dump Cminor"},
	"drtl":    {&dRTL, "dump RTL"},
	"dltl":    {&dLTL, "dump LTL"},
	"dmach":   {&dMach, "dump Mach"},
}

// ErrNotImplemented indicates a feature is not yet implemented
var ErrNotImplemented = errors.New("not yet implemented")

// checkDebugFlags checks if any debug flags are set and returns an error
func checkDebugFlags(w io.Writer) error {
	for name, info := range debugFlags {
		if *info.flag {
			fmt.Fprintf(w, "ralph-cc: warning: -%s (%s) is not yet implemented\n", name, info.desc)
			return ErrNotImplemented
		}
	}
	return nil
}

func main() {
	os.Exit(run())
}

func run() int {
	rootCmd := newRootCmd(os.Stderr)
	if err := rootCmd.Execute(); err != nil {
		return 1
	}
	return 0
}

func newRootCmd(errOut io.Writer) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "ralph-cc [file]",
		Short: "ralph-cc is a C compiler frontend for testing compilation passes",
		Long: `ralph-cc is a C compiler frontend CLI optimized for testing
compilation passes rather than practical use. It follows the
CompCert design with the goal of equivalent output on each IR.`,
		Version:       version,
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkDebugFlags(errOut); err != nil {
				return err
			}
			if len(args) == 0 {
				cmd.Help()
				return nil
			}
			filename := args[0]
			fmt.Fprintf(errOut, "ralph-cc: compiling %s\n", filename)
			return nil
		},
	}
	rootCmd.SetErr(errOut)

	// Add debug flags
	rootCmd.Flags().BoolVarP(&dParse, "dparse", "", false, "Dump after parsing")
	rootCmd.Flags().BoolVarP(&dC, "dc", "", false, "Dump CompCert C")
	rootCmd.Flags().BoolVarP(&dAsm, "dasm", "", false, "Dump assembly")
	rootCmd.Flags().BoolVarP(&dClight, "dclight", "", false, "Dump Clight")
	rootCmd.Flags().BoolVarP(&dCminor, "dcminor", "", false, "Dump Cminor")
	rootCmd.Flags().BoolVarP(&dRTL, "drtl", "", false, "Dump RTL")
	rootCmd.Flags().BoolVarP(&dLTL, "dltl", "", false, "Dump LTL")
	rootCmd.Flags().BoolVarP(&dMach, "dmach", "", false, "Dump Mach")

	return rootCmd
}

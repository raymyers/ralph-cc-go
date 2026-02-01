package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/raymyers/ralph-cc/pkg/cabs"
	"github.com/raymyers/ralph-cc/pkg/lexer"
	"github.com/raymyers/ralph-cc/pkg/parser"
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
// Note: dparse is handled separately as it's now implemented
var debugFlags = map[string]debugFlagInfo{
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

// checkDebugFlags checks if any unimplemented debug flags are set and returns an error
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
	rootCmd := newRootCmd(os.Stdout, os.Stderr)
	if err := rootCmd.Execute(); err != nil {
		return 1
	}
	return 0
}

func newRootCmd(out, errOut io.Writer) *cobra.Command {
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
			// Check unimplemented debug flags first
			if err := checkDebugFlags(errOut); err != nil {
				return err
			}

			if len(args) == 0 {
				cmd.Help()
				return nil
			}
			filename := args[0]

			// Handle -dparse: parse and dump the AST
			if dParse {
				return doParse(filename, out, errOut)
			}

			fmt.Fprintf(errOut, "ralph-cc: compiling %s\n", filename)
			return nil
		},
	}
	rootCmd.SetOut(out)
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

// doParse parses the file and prints the AST
func doParse(filename string, out, errOut io.Writer) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(errOut, "ralph-cc: error reading %s: %v\n", filename, err)
		return err
	}

	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		for _, e := range p.Errors() {
			fmt.Fprintf(errOut, "%s: %s\n", filename, e)
		}
		return fmt.Errorf("parsing failed with %d errors", len(p.Errors()))
	}

	// Print the AST
	printer := cabs.NewPrinter(out)
	printer.PrintProgram(program)
	return nil
}

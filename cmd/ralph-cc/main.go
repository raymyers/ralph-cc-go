package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/raymyers/ralph-cc/pkg/cabs"
	"github.com/raymyers/ralph-cc/pkg/clight"
	"github.com/raymyers/ralph-cc/pkg/clightgen"
	"github.com/raymyers/ralph-cc/pkg/cminor"
	"github.com/raymyers/ralph-cc/pkg/cminorgen"
	"github.com/raymyers/ralph-cc/pkg/csharpminor"
	"github.com/raymyers/ralph-cc/pkg/cshmgen"
	"github.com/raymyers/ralph-cc/pkg/lexer"
	"github.com/raymyers/ralph-cc/pkg/parser"
	"github.com/raymyers/ralph-cc/pkg/rtl"
	"github.com/raymyers/ralph-cc/pkg/rtlgen"
	"github.com/raymyers/ralph-cc/pkg/selection"
	"github.com/spf13/cobra"
)

var version = "0.1.0"

// Debug flags for dumping intermediate representations
var (
	dParse       bool
	dC           bool
	dAsm         bool
	dClight      bool
	dCsharpminor bool
	dCminor      bool
	dRTL         bool
	dLTL         bool
	dMach        bool
)

// debugFlagInfo holds metadata for a debug flag
type debugFlagInfo struct {
	flag *bool
	desc string
}

// debugFlags maps flag names to descriptions for unimplemented warnings
// Note: dparse, dclight, dcsharpminor, dcminor, and drtl are handled separately as they're implemented
var debugFlags = map[string]debugFlagInfo{
	"dc":    {&dC, "dump CompCert C"},
	"dasm":  {&dAsm, "dump assembly"},
	"dltl":  {&dLTL, "dump LTL"},
	"dmach": {&dMach, "dump Mach"},
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
	// Normalize CompCert-style single-dash flags to double-dash for pflag compatibility
	rootCmd.SetArgs(normalizeFlags(os.Args[1:]))
	if err := rootCmd.Execute(); err != nil {
		return 1
	}
	return 0
}

// debugFlagNames lists all debug flags that should accept single-dash style (CompCert compatibility)
var debugFlagNames = []string{"dparse", "dc", "dasm", "dclight", "dcsharpminor", "dcminor", "drtl", "dltl", "dmach"}

// normalizeFlags converts CompCert-style single-dash flags like -dparse to --dparse
func normalizeFlags(args []string) []string {
	result := make([]string, len(args))
	for i, arg := range args {
		// Check if it's a single-dash debug flag (e.g., -dparse)
		for _, flagName := range debugFlagNames {
			if arg == "-"+flagName {
				result[i] = "--" + flagName
				break
			}
		}
		if result[i] == "" {
			result[i] = arg
		}
	}
	return result
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

			// Handle -dclight: transform to Clight and dump
			if dClight {
				return doClight(filename, out, errOut)
			}

			// Handle -dcsharpminor: transform to Csharpminor and dump
			if dCsharpminor {
				return doCsharpminor(filename, out, errOut)
			}

			// Handle -dcminor: transform to Cminor and dump
			if dCminor {
				return doCminor(filename, out, errOut)
			}

			// Handle -drtl: transform to RTL and dump
			if dRTL {
				return doRTL(filename, out, errOut)
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
	rootCmd.Flags().BoolVarP(&dCsharpminor, "dcsharpminor", "", false, "Dump Csharpminor")
	rootCmd.Flags().BoolVarP(&dCminor, "dcminor", "", false, "Dump Cminor")
	rootCmd.Flags().BoolVarP(&dRTL, "drtl", "", false, "Dump RTL")
	rootCmd.Flags().BoolVarP(&dLTL, "dltl", "", false, "Dump LTL")
	rootCmd.Flags().BoolVarP(&dMach, "dmach", "", false, "Dump Mach")

	return rootCmd
}

// doParse parses the file and writes the AST to a .parsed.c file (matching CompCert behavior)
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

	// Compute output filename: input.c -> input.parsed.c
	outputFilename := parsedOutputFilename(filename)

	// Create output file
	outFile, err := os.Create(outputFilename)
	if err != nil {
		fmt.Fprintf(errOut, "ralph-cc: error creating %s: %v\n", outputFilename, err)
		return err
	}
	defer outFile.Close()

	// Print the AST to the file
	printer := cabs.NewPrinter(outFile)
	printer.PrintProgram(program)

	// Also print to stdout for convenience
	printer = cabs.NewPrinter(out)
	printer.PrintProgram(program)

	return nil
}

// parsedOutputFilename returns the output filename for -dparse
// input.c -> input.parsed.c (matching CompCert convention)
func parsedOutputFilename(filename string) string {
	ext := ".c"
	if strings.HasSuffix(filename, ext) {
		return filename[:len(filename)-len(ext)] + ".parsed.c"
	}
	return filename + ".parsed.c"
}

// doClight transforms the file to Clight and writes output to .light.c file
func doClight(filename string, out, errOut io.Writer) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(errOut, "ralph-cc: error reading %s: %v\n", filename, err)
		return err
	}

	// Parse
	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		for _, e := range p.Errors() {
			fmt.Fprintf(errOut, "%s: %s\n", filename, e)
		}
		return fmt.Errorf("parsing failed with %d errors", len(p.Errors()))
	}

	// Transform to Clight
	clightProg := clightgen.TranslateProgram(program)

	// Compute output filename: input.c -> input.light.c
	outputFilename := clightOutputFilename(filename)

	// Create output file
	outFile, err := os.Create(outputFilename)
	if err != nil {
		fmt.Fprintf(errOut, "ralph-cc: error creating %s: %v\n", outputFilename, err)
		return err
	}
	defer outFile.Close()

	// Print the Clight AST to the file
	printer := clight.NewPrinter(outFile)
	printer.PrintProgram(clightProg)

	// Also print to stdout for convenience
	printer = clight.NewPrinter(out)
	printer.PrintProgram(clightProg)

	return nil
}

// clightOutputFilename returns the output filename for -dclight
func clightOutputFilename(filename string) string {
	ext := ".c"
	if strings.HasSuffix(filename, ext) {
		return filename[:len(filename)-len(ext)] + ".light.c"
	}
	return filename + ".light.c"
}

// doCsharpminor transforms the file to Csharpminor and writes output to .csharpminor file
func doCsharpminor(filename string, out, errOut io.Writer) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(errOut, "ralph-cc: error reading %s: %v\n", filename, err)
		return err
	}

	// Parse
	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		for _, e := range p.Errors() {
			fmt.Fprintf(errOut, "%s: %s\n", filename, e)
		}
		return fmt.Errorf("parsing failed with %d errors", len(p.Errors()))
	}

	// Transform to Clight
	clightProg := clightgen.TranslateProgram(program)

	// Transform to Csharpminor
	csharpminorProg := cshmgen.TranslateProgram(clightProg)

	// Compute output filename: input.c -> input.csharpminor
	outputFilename := csharpminorOutputFilename(filename)

	// Create output file
	outFile, err := os.Create(outputFilename)
	if err != nil {
		fmt.Fprintf(errOut, "ralph-cc: error creating %s: %v\n", outputFilename, err)
		return err
	}
	defer outFile.Close()

	// Print the Csharpminor AST to the file
	printer := csharpminor.NewPrinter(outFile)
	printer.PrintProgram(csharpminorProg)

	// Also print to stdout for convenience
	printer = csharpminor.NewPrinter(out)
	printer.PrintProgram(csharpminorProg)

	return nil
}

// csharpminorOutputFilename returns the output filename for -dcsharpminor
func csharpminorOutputFilename(filename string) string {
	ext := ".c"
	if strings.HasSuffix(filename, ext) {
		return filename[:len(filename)-len(ext)] + ".csharpminor"
	}
	return filename + ".csharpminor"
}

// doCminor transforms the file to Cminor and writes output to .cminor file
func doCminor(filename string, out, errOut io.Writer) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(errOut, "ralph-cc: error reading %s: %v\n", filename, err)
		return err
	}

	// Parse
	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		for _, e := range p.Errors() {
			fmt.Fprintf(errOut, "%s: %s\n", filename, e)
		}
		return fmt.Errorf("parsing failed with %d errors", len(p.Errors()))
	}

	// Transform to Clight
	clightProg := clightgen.TranslateProgram(program)

	// Transform to Csharpminor
	csharpminorProg := cshmgen.TranslateProgram(clightProg)

	// Transform to Cminor
	cminorProg := cminorgen.TransformProgram(csharpminorProg)

	// Compute output filename: input.c -> input.cminor
	outputFilename := cminorOutputFilename(filename)

	// Create output file
	outFile, err := os.Create(outputFilename)
	if err != nil {
		fmt.Fprintf(errOut, "ralph-cc: error creating %s: %v\n", outputFilename, err)
		return err
	}
	defer outFile.Close()

	// Print the Cminor AST to the file
	printer := cminor.NewPrinter(outFile)
	printer.PrintProgram(cminorProg)

	// Also print to stdout for convenience
	printer = cminor.NewPrinter(out)
	printer.PrintProgram(cminorProg)

	return nil
}

// cminorOutputFilename returns the output filename for -dcminor
func cminorOutputFilename(filename string) string {
	ext := ".c"
	if strings.HasSuffix(filename, ext) {
		return filename[:len(filename)-len(ext)] + ".cminor"
	}
	return filename + ".cminor"
}

// doRTL transforms the file to RTL and writes output to .rtl.0 file
func doRTL(filename string, out, errOut io.Writer) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(errOut, "ralph-cc: error reading %s: %v\n", filename, err)
		return err
	}

	// Parse
	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		for _, e := range p.Errors() {
			fmt.Fprintf(errOut, "%s: %s\n", filename, e)
		}
		return fmt.Errorf("parsing failed with %d errors", len(p.Errors()))
	}

	// Transform to Clight
	clightProg := clightgen.TranslateProgram(program)

	// Transform to Csharpminor
	csharpminorProg := cshmgen.TranslateProgram(clightProg)

	// Transform to Cminor
	cminorProg := cminorgen.TransformProgram(csharpminorProg)

	// Transform to CminorSel
	selCtx := selection.NewSelectionContext(nil, nil)
	cminorselProg := selCtx.SelectProgram(*cminorProg)

	// Transform to RTL
	rtlProg := rtlgen.TranslateProgram(cminorselProg)

	// Compute output filename: input.c -> input.rtl.0
	outputFilename := rtlOutputFilename(filename)

	// Create output file
	outFile, err := os.Create(outputFilename)
	if err != nil {
		fmt.Fprintf(errOut, "ralph-cc: error creating %s: %v\n", outputFilename, err)
		return err
	}
	defer outFile.Close()

	// Print the RTL AST to the file
	printer := rtl.NewPrinter(outFile)
	printer.PrintProgram(rtlProg)

	// Also print to stdout for convenience
	printer = rtl.NewPrinter(out)
	printer.PrintProgram(rtlProg)

	return nil
}

// rtlOutputFilename returns the output filename for -drtl
func rtlOutputFilename(filename string) string {
	ext := ".c"
	if strings.HasSuffix(filename, ext) {
		return filename[:len(filename)-len(ext)] + ".rtl.0"
	}
	return filename + ".rtl.0"
}

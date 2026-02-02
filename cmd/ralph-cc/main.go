package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/raymyers/ralph-cc/pkg/asm"
	"github.com/raymyers/ralph-cc/pkg/asmgen"
	"github.com/raymyers/ralph-cc/pkg/cabs"
	"github.com/raymyers/ralph-cc/pkg/clight"
	"github.com/raymyers/ralph-cc/pkg/clightgen"
	"github.com/raymyers/ralph-cc/pkg/cminor"
	"github.com/raymyers/ralph-cc/pkg/cminorgen"
	"github.com/raymyers/ralph-cc/pkg/csharpminor"
	"github.com/raymyers/ralph-cc/pkg/cshmgen"
	"github.com/raymyers/ralph-cc/pkg/lexer"
	"github.com/raymyers/ralph-cc/pkg/linearize"
	"github.com/raymyers/ralph-cc/pkg/ltl"
	"github.com/raymyers/ralph-cc/pkg/mach"
	"github.com/raymyers/ralph-cc/pkg/parser"
	"github.com/raymyers/ralph-cc/pkg/preproc"
	"github.com/raymyers/ralph-cc/pkg/regalloc"
	"github.com/raymyers/ralph-cc/pkg/rtl"
	"github.com/raymyers/ralph-cc/pkg/rtlgen"
	"github.com/raymyers/ralph-cc/pkg/selection"
	"github.com/raymyers/ralph-cc/pkg/stacking"
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
	dPP          bool // Debug preprocessor
)

// Preprocessor options
var (
	includePaths   []string
	systemPaths    []string
	defineFlags    []string
	undefineFlags  []string
	preprocessOnly bool // -E flag
	useExternalPP  bool // Use external preprocessor
)

// debugFlagInfo holds metadata for a debug flag
type debugFlagInfo struct {
	flag *bool
	desc string
}

// debugFlags maps flag names to descriptions for unimplemented warnings
// Note: dparse, dclight, dcsharpminor, dcminor, drtl, dltl, dmach, and dasm are handled separately as they're implemented
var debugFlags = map[string]debugFlagInfo{
	"dc": {&dC, "dump CompCert C"},
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
var debugFlagNames = []string{"dparse", "dc", "dasm", "dclight", "dcsharpminor", "dcminor", "drtl", "dltl", "dmach", "dpp"}

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

			// Handle -E: preprocess only
			if preprocessOnly {
				return doPreprocessOnly(filename, out, errOut)
			}

			// Handle -dpp: debug preprocessor output
			if dPP {
				return doPreprocessDebug(filename, out, errOut)
			}

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

			// Handle -dltl: transform to LTL and dump
			if dLTL {
				return doLTL(filename, out, errOut)
			}

			// Handle -dmach: transform to Mach and dump
			if dMach {
				return doMach(filename, out, errOut)
			}

			// Handle -dasm: transform to Assembly and dump
			if dAsm {
				return doAsm(filename, out, errOut)
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
	rootCmd.Flags().BoolVarP(&dPP, "dpp", "", false, "Debug preprocessor operation")

	// Add preprocessor flags
	rootCmd.Flags().StringArrayVarP(&includePaths, "include", "I", nil, "Add directory to include search path")
	rootCmd.Flags().StringArrayVar(&systemPaths, "isystem", nil, "Add directory to system include search path")
	rootCmd.Flags().StringArrayVarP(&defineFlags, "define", "D", nil, "Define macro (NAME or NAME=VALUE)")
	rootCmd.Flags().StringArrayVarP(&undefineFlags, "undefine", "U", nil, "Undefine macro")
	rootCmd.Flags().BoolVarP(&preprocessOnly, "preprocess", "E", false, "Preprocess only, output to stdout")
	rootCmd.Flags().BoolVar(&useExternalPP, "external-cpp", false, "Use external C preprocessor instead of internal")

	return rootCmd
}

// buildPreprocessorOptions creates preproc.Options from CLI flags
func buildPreprocessorOptions() *preproc.Options {
	opts := &preproc.Options{
		IncludePaths: includePaths,
		SystemPaths:  systemPaths,
		Defines:      make(map[string]string),
		Undefines:    undefineFlags,
		UseExternal:  useExternalPP,
	}

	// Parse -D flags (NAME or NAME=VALUE)
	for _, d := range defineFlags {
		if idx := strings.Index(d, "="); idx >= 0 {
			opts.Defines[d[:idx]] = d[idx+1:]
		} else {
			opts.Defines[d] = ""
		}
	}

	return opts
}

// readAndPreprocess reads a C file and optionally preprocesses it.
// It uses our internal preprocessor for .c files to handle #include directives.
// Files with .i or .p extensions are assumed already preprocessed.
func readAndPreprocess(filename string, errOut io.Writer) (string, error) {
	if preproc.NeedsPreprocessing(filename) {
		opts := buildPreprocessorOptions()
		content, err := preproc.Preprocess(filename, opts)
		if err != nil {
			fmt.Fprintf(errOut, "ralph-cc: preprocessing error: %v\n", err)
			return "", err
		}
		return content, nil
	}

	// File doesn't need preprocessing, read directly
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(errOut, "ralph-cc: error reading %s: %v\n", filename, err)
		return "", err
	}
	return string(content), nil
}

// doPreprocessOnly preprocesses and outputs to stdout (-E flag)
func doPreprocessOnly(filename string, out, errOut io.Writer) error {
	opts := buildPreprocessorOptions()
	opts.LineMarkers = true // Include line markers like traditional cpp

	content, err := preproc.Preprocess(filename, opts)
	if err != nil {
		fmt.Fprintf(errOut, "ralph-cc: preprocessing error: %v\n", err)
		return err
	}

	fmt.Fprint(out, content)
	return nil
}

// doPreprocessDebug preprocesses with debug info and outputs to .i file (-dpp flag)
func doPreprocessDebug(filename string, out, errOut io.Writer) error {
	opts := buildPreprocessorOptions()
	opts.LineMarkers = true

	content, err := preproc.Preprocess(filename, opts)
	if err != nil {
		fmt.Fprintf(errOut, "ralph-cc: preprocessing error: %v\n", err)
		return err
	}

	// Compute output filename: input.c -> input.i
	outputFilename := preprocessedOutputFilename(filename)

	// Create output file
	outFile, err := os.Create(outputFilename)
	if err != nil {
		fmt.Fprintf(errOut, "ralph-cc: error creating %s: %v\n", outputFilename, err)
		return err
	}
	defer outFile.Close()

	// Write to file
	outFile.WriteString(content)

	// Also print to stdout
	fmt.Fprint(out, content)

	return nil
}

// preprocessedOutputFilename returns the output filename for -dpp
func preprocessedOutputFilename(filename string) string {
	ext := ".c"
	if strings.HasSuffix(filename, ext) {
		return filename[:len(filename)-len(ext)] + ".i"
	}
	return filename + ".i"
}

// parseFile preprocesses and parses a C file, returning the AST
func parseFile(filename string, errOut io.Writer) (*cabs.Program, error) {
	content, err := readAndPreprocess(filename, errOut)
	if err != nil {
		return nil, err
	}

	l := lexer.New(content)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		for _, e := range p.Errors() {
			fmt.Fprintf(errOut, "%s: %s\n", filename, e)
		}
		return nil, fmt.Errorf("parsing failed with %d errors", len(p.Errors()))
	}
	return program, nil
}

// doParse parses the file and writes the AST to a .parsed.c file (matching CompCert behavior)
func doParse(filename string, out, errOut io.Writer) error {
	program, err := parseFile(filename, errOut)
	if err != nil {
		return err
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
	program, err := parseFile(filename, errOut)
	if err != nil {
		return err
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
	program, err := parseFile(filename, errOut)
	if err != nil {
		return err
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
	program, err := parseFile(filename, errOut)
	if err != nil {
		return err
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
	program, err := parseFile(filename, errOut)
	if err != nil {
		return err
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

// doLTL transforms the file to LTL and writes output to .ltl file
func doLTL(filename string, out, errOut io.Writer) error {
	program, err := parseFile(filename, errOut)
	if err != nil {
		return err
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

	// Transform to LTL
	ltlProg := regalloc.TransformProgram(rtlProg)

	// Compute output filename: input.c -> input.ltl
	outputFilename := ltlOutputFilename(filename)

	// Create output file
	outFile, err := os.Create(outputFilename)
	if err != nil {
		fmt.Fprintf(errOut, "ralph-cc: error creating %s: %v\n", outputFilename, err)
		return err
	}
	defer outFile.Close()

	// Print the LTL AST to the file
	printer := ltl.NewPrinter(outFile)
	printer.PrintProgram(ltlProg)

	// Also print to stdout for convenience
	printer = ltl.NewPrinter(out)
	printer.PrintProgram(ltlProg)

	return nil
}

// ltlOutputFilename returns the output filename for -dltl
func ltlOutputFilename(filename string) string {
	ext := ".c"
	if strings.HasSuffix(filename, ext) {
		return filename[:len(filename)-len(ext)] + ".ltl"
	}
	return filename + ".ltl"
}

// doMach transforms the file to Mach and writes output to .mach file
func doMach(filename string, out, errOut io.Writer) error {
	program, err := parseFile(filename, errOut)
	if err != nil {
		return err
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

	// Transform to LTL
	ltlProg := regalloc.TransformProgram(rtlProg)

	// Transform to Linear
	linearProg := linearize.TransformProgram(ltlProg)

	// Transform to Mach
	machProg := stacking.TransformProgram(linearProg)

	// Compute output filename: input.c -> input.mach
	outputFilename := machOutputFilename(filename)

	// Create output file
	outFile, err := os.Create(outputFilename)
	if err != nil {
		fmt.Fprintf(errOut, "ralph-cc: error creating %s: %v\n", outputFilename, err)
		return err
	}
	defer outFile.Close()

	// Print the Mach AST to the file
	printer := mach.NewPrinter(outFile)
	printer.PrintProgram(machProg)

	// Also print to stdout for convenience
	printer = mach.NewPrinter(out)
	printer.PrintProgram(machProg)

	return nil
}

// machOutputFilename returns the output filename for -dmach
func machOutputFilename(filename string) string {
	ext := ".c"
	if strings.HasSuffix(filename, ext) {
		return filename[:len(filename)-len(ext)] + ".mach"
	}
	return filename + ".mach"
}

// doAsm transforms the file to Assembly and writes output to .s file
func doAsm(filename string, out, errOut io.Writer) error {
	program, err := parseFile(filename, errOut)
	if err != nil {
		return err
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

	// Transform to LTL
	ltlProg := regalloc.TransformProgram(rtlProg)

	// Transform to Linear
	linearProg := linearize.TransformProgram(ltlProg)

	// Transform to Mach
	machProg := stacking.TransformProgram(linearProg)

	// Transform to Assembly
	asmProg := asmgen.TransformProgram(machProg)

	// Compute output filename: input.c -> input.s
	outputFilename := asmOutputFilename(filename)

	// Create output file
	outFile, err := os.Create(outputFilename)
	if err != nil {
		fmt.Fprintf(errOut, "ralph-cc: error creating %s: %v\n", outputFilename, err)
		return err
	}
	defer outFile.Close()

	// Print the Assembly to the file
	printer := asm.NewPrinter(outFile)
	printer.PrintProgram(asmProg)

	// Also print to stdout for convenience
	printer = asm.NewPrinter(out)
	printer.PrintProgram(asmProg)

	return nil
}

// asmOutputFilename returns the output filename for -dasm
func asmOutputFilename(filename string) string {
	ext := ".c"
	if strings.HasSuffix(filename, ext) {
		return filename[:len(filename)-len(ext)] + ".s"
	}
	return filename + ".s"
}

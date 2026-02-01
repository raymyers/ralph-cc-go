package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/raymyers/ralph-cc/pkg/cabs"
	"github.com/raymyers/ralph-cc/pkg/clight"
	"github.com/raymyers/ralph-cc/pkg/ctypes"
	"github.com/raymyers/ralph-cc/pkg/lexer"
	"github.com/raymyers/ralph-cc/pkg/parser"
	"github.com/raymyers/ralph-cc/pkg/simplexpr"
	"github.com/raymyers/ralph-cc/pkg/simpllocals"
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
// Note: dparse and dclight are handled separately as they're now implemented
var debugFlags = map[string]debugFlagInfo{
	"dc":      {&dC, "dump CompCert C"},
	"dasm":    {&dAsm, "dump assembly"},
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
	// Normalize CompCert-style single-dash flags to double-dash for pflag compatibility
	rootCmd.SetArgs(normalizeFlags(os.Args[1:]))
	if err := rootCmd.Execute(); err != nil {
		return 1
	}
	return 0
}

// debugFlagNames lists all debug flags that should accept single-dash style (CompCert compatibility)
var debugFlagNames = []string{"dparse", "dc", "dasm", "dclight", "dcminor", "drtl", "dltl", "dmach"}

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
	clightProg := transformToClight(program)

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

// transformToClight transforms a Cabs program to a Clight program
func transformToClight(prog *cabs.Program) *clight.Program {
	result := &clight.Program{}

	for _, def := range prog.Definitions {
		switch d := def.(type) {
		case cabs.FunDef:
			fn := transformFunction(&d)
			result.Functions = append(result.Functions, fn)
		case cabs.StructDef:
			s := ctypes.Tstruct{
				Name:   d.Name,
				Fields: make([]ctypes.Field, len(d.Fields)),
			}
			for i, f := range d.Fields {
				s.Fields[i] = ctypes.Field{
					Name: f.Name,
					Type: typeFromString(f.TypeSpec),
				}
			}
			result.Structs = append(result.Structs, s)
		case cabs.UnionDef:
			u := ctypes.Tunion{
				Name:   d.Name,
				Fields: make([]ctypes.Field, len(d.Fields)),
			}
			for i, f := range d.Fields {
				u.Fields[i] = ctypes.Field{
					Name: f.Name,
					Type: typeFromString(f.TypeSpec),
				}
			}
			result.Unions = append(result.Unions, u)
		}
	}

	return result
}

// transformFunction transforms a Cabs function to a Clight function
func transformFunction(fn *cabs.FunDef) clight.Function {
	// Create transformers
	simplExpr := simplexpr.New()
	simplLoc := simpllocals.New()

	// Set up type environment for parameters
	for _, param := range fn.Params {
		typ := typeFromString(param.TypeSpec)
		simplExpr.SetType(param.Name, typ)
	}

	// Analyze the function for address-taken variables
	if fn.Body != nil {
		simplLoc.AnalyzeFunction(fn)
	}

	// Collect local variables from the body
	var locals []clight.VarDecl
	if fn.Body != nil {
		collectLocals(fn.Body, &locals, simplExpr)
	}

	// Analyze which locals can be promoted to temps
	localInfos := simplLoc.AnalyzeLocals(locals)
	remainingLocals := simpllocals.FilterUnpromotedLocals(localInfos)

	// Continue temp IDs from simpllocals
	simplExpr.Reset()
	for _, param := range fn.Params {
		simplExpr.SetType(param.Name, typeFromString(param.TypeSpec))
	}

	// Set starting temp ID after simpllocals temps
	nextTemp := 1
	for _, info := range localInfos {
		if info.Promoted && info.TempID >= nextTemp {
			nextTemp = info.TempID + 1
		}
	}

	// Transform the body
	var body clight.Stmt = clight.Sskip{}
	if fn.Body != nil {
		body = transformBlock(fn.Body, simplExpr)
	}

	// Apply simpllocals transformation to the body
	body = simplLoc.TransformStmt(body)

	// Collect temp types
	var temps []ctypes.Type
	temps = append(temps, simplExpr.TempTypes()...)
	temps = append(temps, simplLoc.TempTypes()...)

	// Build params
	params := make([]clight.VarDecl, len(fn.Params))
	for i, p := range fn.Params {
		params[i] = clight.VarDecl{
			Name: p.Name,
			Type: typeFromString(p.TypeSpec),
		}
	}

	return clight.Function{
		Name:   fn.Name,
		Return: typeFromString(fn.ReturnType),
		Params: params,
		Locals: remainingLocals,
		Temps:  temps,
		Body:   body,
	}
}

// collectLocals extracts local variable declarations from a block
func collectLocals(block *cabs.Block, locals *[]clight.VarDecl, simplExpr *simplexpr.Transformer) {
	for _, item := range block.Items {
		switch s := item.(type) {
		case cabs.DeclStmt:
			for _, decl := range s.Decls {
				typ := typeFromString(decl.TypeSpec)
				simplExpr.SetType(decl.Name, typ)
				*locals = append(*locals, clight.VarDecl{
					Name: decl.Name,
					Type: typ,
				})
			}
		case cabs.Block:
			collectLocals(&s, locals, simplExpr)
		case *cabs.Block:
			collectLocals(s, locals, simplExpr)
		}
	}
}

// transformBlock transforms a Cabs block to a Clight statement
func transformBlock(block *cabs.Block, simplExpr *simplexpr.Transformer) clight.Stmt {
	var stmts []clight.Stmt
	for _, item := range block.Items {
		stmt := transformStmt(item, simplExpr)
		stmts = append(stmts, stmt)
	}
	return clight.Seq(stmts...)
}

// transformStmt transforms a Cabs statement to a Clight statement
func transformStmt(stmt cabs.Stmt, simplExpr *simplexpr.Transformer) clight.Stmt {
	switch s := stmt.(type) {
	case cabs.Return:
		if s.Expr == nil {
			return clight.Sreturn{Value: nil}
		}
		result := simplExpr.TransformExpr(s.Expr)
		return clight.Seq(append(result.Stmts, clight.Sreturn{Value: result.Expr})...)

	case cabs.Computation:
		result := simplExpr.TransformExpr(s.Expr)
		return clight.Seq(result.Stmts...)

	case cabs.If:
		condResult := simplExpr.TransformExpr(s.Cond)
		thenStmt := transformStmt(s.Then, simplExpr)
		var elseStmt clight.Stmt = clight.Sskip{}
		if s.Else != nil {
			elseStmt = transformStmt(s.Else, simplExpr)
		}
		ifStmt := clight.Sifthenelse{
			Cond: condResult.Expr,
			Then: thenStmt,
			Else: elseStmt,
		}
		return clight.Seq(append(condResult.Stmts, ifStmt)...)

	case cabs.While:
		// while (cond) body becomes: loop { if (cond) body else break }
		condResult := simplExpr.TransformExpr(s.Cond)
		bodyStmt := transformStmt(s.Body, simplExpr)
		loopBody := clight.Sifthenelse{
			Cond: condResult.Expr,
			Then: bodyStmt,
			Else: clight.Sbreak{},
		}
		// Prepend condition side-effects to loop body
		fullBody := clight.Seq(append(condResult.Stmts, loopBody)...)
		return clight.Sloop{Body: fullBody, Continue: clight.Sskip{}}

	case cabs.DoWhile:
		// do body while (cond) becomes: loop { body; if (!cond) break }
		bodyStmt := transformStmt(s.Body, simplExpr)
		condResult := simplExpr.TransformExpr(s.Cond)
		checkCond := clight.Sifthenelse{
			Cond: clight.Eunop{Op: clight.Onotbool, Arg: condResult.Expr, Typ: ctypes.Int()},
			Then: clight.Sbreak{},
			Else: clight.Sskip{},
		}
		fullBody := clight.Seq(append([]clight.Stmt{bodyStmt}, append(condResult.Stmts, checkCond)...)...)
		return clight.Sloop{Body: fullBody, Continue: clight.Sskip{}}

	case cabs.For:
		// for (init; cond; step) body becomes:
		// init; loop { if (cond) { body; step } else break }
		var initStmt clight.Stmt = clight.Sskip{}
		if s.Init != nil {
			initResult := simplExpr.TransformExpr(s.Init)
			initStmt = clight.Seq(initResult.Stmts...)
		}

		var condExpr clight.Expr = clight.Econst_int{Value: 1, Typ: ctypes.Int()} // default: true
		var condStmts []clight.Stmt
		if s.Cond != nil {
			condResult := simplExpr.TransformExpr(s.Cond)
			condExpr = condResult.Expr
			condStmts = condResult.Stmts
		}

		bodyStmt := transformStmt(s.Body, simplExpr)

		var stepStmt clight.Stmt = clight.Sskip{}
		if s.Step != nil {
			stepResult := simplExpr.TransformExpr(s.Step)
			stepStmt = clight.Seq(stepResult.Stmts...)
		}

		loopBody := clight.Sifthenelse{
			Cond: condExpr,
			Then: clight.Seq(bodyStmt, stepStmt),
			Else: clight.Sbreak{},
		}
		fullBody := clight.Seq(append(condStmts, loopBody)...)
		return clight.Seq(initStmt, clight.Sloop{Body: fullBody, Continue: clight.Sskip{}})

	case cabs.Break:
		return clight.Sbreak{}

	case cabs.Continue:
		return clight.Scontinue{}

	case cabs.Switch:
		exprResult := simplExpr.TransformExpr(s.Expr)
		var cases []clight.SwitchCase
		var defaultStmt clight.Stmt = clight.Sskip{}
		for _, c := range s.Cases {
			if c.Expr == nil {
				// default case
				var stmts []clight.Stmt
				for _, st := range c.Stmts {
					stmts = append(stmts, transformStmt(st, simplExpr))
				}
				defaultStmt = clight.Seq(stmts...)
			} else {
				// case with value
				var stmts []clight.Stmt
				for _, st := range c.Stmts {
					stmts = append(stmts, transformStmt(st, simplExpr))
				}
				if constExpr, ok := c.Expr.(cabs.Constant); ok {
					cases = append(cases, clight.SwitchCase{
						Value: constExpr.Value,
						Body:  clight.Seq(stmts...),
					})
				}
			}
		}
		return clight.Seq(append(exprResult.Stmts, clight.Sswitch{
			Expr:    exprResult.Expr,
			Cases:   cases,
			Default: defaultStmt,
		})...)

	case cabs.Goto:
		return clight.Sgoto{Label: s.Label}

	case cabs.Label:
		innerStmt := transformStmt(s.Stmt, simplExpr)
		return clight.Slabel{Label: s.Name, Stmt: innerStmt}

	case cabs.Block:
		return transformBlock(&s, simplExpr)

	case *cabs.Block:
		return transformBlock(s, simplExpr)

	case cabs.DeclStmt:
		// Declarations with initializers become assignments
		var stmts []clight.Stmt
		for _, decl := range s.Decls {
			if decl.Initializer != nil {
				typ := typeFromString(decl.TypeSpec)
				result := simplExpr.TransformExpr(decl.Initializer)
				stmts = append(stmts, result.Stmts...)
				stmts = append(stmts, clight.Sassign{
					LHS: clight.Evar{Name: decl.Name, Typ: typ},
					RHS: result.Expr,
				})
			}
		}
		return clight.Seq(stmts...)

	default:
		return clight.Sskip{}
	}
}

// typeFromString converts a C type string to a ctypes.Type
func typeFromString(typeName string) ctypes.Type {
	// Remove any leading/trailing whitespace
	typeName = strings.TrimSpace(typeName)

	switch typeName {
	case "void":
		return ctypes.Void()
	case "char":
		return ctypes.Char()
	case "unsigned char":
		return ctypes.UChar()
	case "short":
		return ctypes.Short()
	case "int":
		return ctypes.Int()
	case "unsigned int", "unsigned":
		return ctypes.UInt()
	case "long":
		return ctypes.Long()
	case "unsigned long":
		return ctypes.Tlong{Sign: ctypes.Unsigned}
	case "float":
		return ctypes.Float()
	case "double":
		return ctypes.Double()
	default:
		// Check for pointer types
		if strings.HasSuffix(typeName, "*") {
			baseType := typeFromString(strings.TrimSpace(typeName[:len(typeName)-1]))
			return ctypes.Pointer(baseType)
		}
		// Check for struct types
		if strings.HasPrefix(typeName, "struct ") {
			structName := strings.TrimPrefix(typeName, "struct ")
			return ctypes.Tstruct{Name: strings.TrimSpace(structName)}
		}
		// Check for union types
		if strings.HasPrefix(typeName, "union ") {
			unionName := strings.TrimPrefix(typeName, "union ")
			return ctypes.Tunion{Name: strings.TrimSpace(unionName)}
		}
		return ctypes.Int() // default fallback
	}
}

// Package clight provides AST printing functionality for Clight IR
package clight

import (
	"fmt"
	"io"
	"strings"

	"github.com/raymyers/ralph-cc/pkg/ctypes"
)

// Printer outputs the Clight AST in a human-readable format
// matching CompCert's .light.c output style
type Printer struct {
	w      io.Writer
	indent int
}

// NewPrinter creates a new Clight AST printer
func NewPrinter(w io.Writer) *Printer {
	return &Printer{w: w, indent: 0}
}

// PrintProgram prints a complete Clight program
func (p *Printer) PrintProgram(prog *Program) {
	// Print struct definitions
	for _, s := range prog.Structs {
		p.printStructDef(s)
		fmt.Fprintln(p.w)
	}

	// Print union definitions
	for _, u := range prog.Unions {
		p.printUnionDef(u)
		fmt.Fprintln(p.w)
	}

	// Print global variables
	for _, g := range prog.Globals {
		fmt.Fprintf(p.w, "%s %s;\n", g.Type.String(), g.Name)
	}
	if len(prog.Globals) > 0 {
		fmt.Fprintln(p.w)
	}

	// Print functions
	for _, fn := range prog.Functions {
		p.printFunction(&fn)
		fmt.Fprintln(p.w)
	}
}

func (p *Printer) printStructDef(s ctypes.Tstruct) {
	fmt.Fprintf(p.w, "struct %s {\n", s.Name)
	for _, f := range s.Fields {
		fmt.Fprintf(p.w, "  %s %s;\n", f.Type.String(), f.Name)
	}
	fmt.Fprintln(p.w, "};")
}

func (p *Printer) printUnionDef(u ctypes.Tunion) {
	fmt.Fprintf(p.w, "union %s {\n", u.Name)
	for _, f := range u.Fields {
		fmt.Fprintf(p.w, "  %s %s;\n", f.Type.String(), f.Name)
	}
	fmt.Fprintln(p.w, "};")
}

// PrintFunction prints a function definition
func (p *Printer) printFunction(fn *Function) {
	// Function signature
	fmt.Fprintf(p.w, "%s %s(", fn.Return.String(), fn.Name)
	for i, param := range fn.Params {
		if i > 0 {
			fmt.Fprint(p.w, ", ")
		}
		fmt.Fprintf(p.w, "%s %s", param.Type.String(), param.Name)
	}
	fmt.Fprintln(p.w, ")")
	fmt.Fprintln(p.w, "{")
	p.indent++

	// Print local variables (those that stay in memory)
	for _, local := range fn.Locals {
		p.writeIndent()
		fmt.Fprintf(p.w, "%s %s;\n", local.Type.String(), local.Name)
	}

	// Print temporary variables
	for i, typ := range fn.Temps {
		p.writeIndent()
		fmt.Fprintf(p.w, "%s $%d;\n", typ.String(), i+1)
	}

	if len(fn.Locals) > 0 || len(fn.Temps) > 0 {
		fmt.Fprintln(p.w)
	}

	// Print body
	p.printStmt(fn.Body)

	p.indent--
	fmt.Fprintln(p.w, "}")
}

func (p *Printer) writeIndent() {
	fmt.Fprint(p.w, strings.Repeat("  ", p.indent))
}

// PrintStmt prints a statement
func (p *Printer) printStmt(stmt Stmt) {
	switch s := stmt.(type) {
	case Sskip:
		// Skip produces no output

	case Sassign:
		p.writeIndent()
		p.printExpr(s.LHS)
		fmt.Fprint(p.w, " = ")
		p.printExpr(s.RHS)
		fmt.Fprintln(p.w, ";")

	case Sset:
		p.writeIndent()
		fmt.Fprintf(p.w, "$%d = ", s.TempID)
		p.printExpr(s.RHS)
		fmt.Fprintln(p.w, ";")

	case Scall:
		p.writeIndent()
		if s.Result != nil {
			fmt.Fprintf(p.w, "$%d = ", *s.Result)
		}
		p.printExpr(s.Func)
		fmt.Fprint(p.w, "(")
		for i, arg := range s.Args {
			if i > 0 {
				fmt.Fprint(p.w, ", ")
			}
			p.printExpr(arg)
		}
		fmt.Fprintln(p.w, ");")

	case Sbuiltin:
		p.writeIndent()
		if s.Result != nil {
			fmt.Fprintf(p.w, "$%d = ", *s.Result)
		}
		fmt.Fprintf(p.w, "__builtin_%s(", s.Builtin)
		for i, arg := range s.Args {
			if i > 0 {
				fmt.Fprint(p.w, ", ")
			}
			p.printExpr(arg)
		}
		fmt.Fprintln(p.w, ");")

	case Ssequence:
		p.printStmt(s.First)
		p.printStmt(s.Second)

	case Sifthenelse:
		p.writeIndent()
		fmt.Fprint(p.w, "if (")
		p.printExpr(s.Cond)
		fmt.Fprintln(p.w, ") {")
		p.indent++
		p.printStmt(s.Then)
		p.indent--
		p.writeIndent()
		fmt.Fprintln(p.w, "} else {")
		p.indent++
		p.printStmt(s.Else)
		p.indent--
		p.writeIndent()
		fmt.Fprintln(p.w, "}")

	case Sloop:
		p.writeIndent()
		fmt.Fprintln(p.w, "loop {")
		p.indent++
		p.printStmt(s.Body)
		p.printStmt(s.Continue)
		p.indent--
		p.writeIndent()
		fmt.Fprintln(p.w, "}")

	case Sbreak:
		p.writeIndent()
		fmt.Fprintln(p.w, "break;")

	case Scontinue:
		p.writeIndent()
		fmt.Fprintln(p.w, "continue;")

	case Sreturn:
		p.writeIndent()
		fmt.Fprint(p.w, "return")
		if s.Value != nil {
			fmt.Fprint(p.w, " ")
			p.printExpr(s.Value)
		}
		fmt.Fprintln(p.w, ";")

	case Sswitch:
		p.writeIndent()
		fmt.Fprint(p.w, "switch (")
		p.printExpr(s.Expr)
		fmt.Fprintln(p.w, ") {")
		for _, c := range s.Cases {
			p.writeIndent()
			fmt.Fprintf(p.w, "case %d:\n", c.Value)
			p.indent++
			p.printStmt(c.Body)
			p.indent--
		}
		p.writeIndent()
		fmt.Fprintln(p.w, "default:")
		p.indent++
		p.printStmt(s.Default)
		p.indent--
		p.writeIndent()
		fmt.Fprintln(p.w, "}")

	case Slabel:
		// Labels are not indented
		fmt.Fprintf(p.w, "%s:\n", s.Label)
		p.printStmt(s.Stmt)

	case Sgoto:
		p.writeIndent()
		fmt.Fprintf(p.w, "goto %s;\n", s.Label)

	default:
		p.writeIndent()
		fmt.Fprintf(p.w, "/* unknown stmt %T */\n", stmt)
	}
}

// PrintExpr prints an expression
func (p *Printer) printExpr(expr Expr) {
	switch e := expr.(type) {
	case Econst_int:
		fmt.Fprintf(p.w, "%d", e.Value)

	case Econst_float:
		fmt.Fprintf(p.w, "%g", e.Value)

	case Econst_long:
		fmt.Fprintf(p.w, "%dL", e.Value)

	case Econst_single:
		fmt.Fprintf(p.w, "%gf", e.Value)

	case Estring:
		fmt.Fprintf(p.w, "%q", e.Value)

	case Evar:
		fmt.Fprint(p.w, e.Name)

	case Etempvar:
		fmt.Fprintf(p.w, "$%d", e.ID)

	case Ederef:
		fmt.Fprint(p.w, "*")
		p.printExprParen(e.Ptr)

	case Eaddrof:
		fmt.Fprint(p.w, "&")
		p.printExprParen(e.Arg)

	case Eunop:
		fmt.Fprint(p.w, e.Op.String())
		p.printExprParen(e.Arg)

	case Ebinop:
		p.printExprParen(e.Left)
		fmt.Fprintf(p.w, " %s ", e.Op.String())
		p.printExprParen(e.Right)

	case Ecast:
		fmt.Fprintf(p.w, "(%s)", e.Typ.String())
		p.printExprParen(e.Arg)

	case Efield:
		p.printExprParen(e.Arg)
		fmt.Fprintf(p.w, ".%s", e.FieldName)

	case Esizeof:
		fmt.Fprintf(p.w, "sizeof(%s)", e.ArgType.String())

	case Ealignof:
		fmt.Fprintf(p.w, "_Alignof(%s)", e.ArgType.String())

	default:
		fmt.Fprintf(p.w, "/* unknown expr %T */", expr)
	}
}

// printExprParen prints an expression, wrapping in parens if needed
func (p *Printer) printExprParen(expr Expr) {
	needsParen := false
	switch expr.(type) {
	case Ebinop:
		needsParen = true
	}

	if needsParen {
		fmt.Fprint(p.w, "(")
		p.printExpr(expr)
		fmt.Fprint(p.w, ")")
	} else {
		p.printExpr(expr)
	}
}

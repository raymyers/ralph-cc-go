// Package csharpminor provides AST printing functionality for Csharpminor IR
package csharpminor

import (
	"fmt"
	"io"
	"strings"
)

// Printer outputs the Csharpminor AST in a human-readable format
type Printer struct {
	w      io.Writer
	indent int
}

// NewPrinter creates a new Csharpminor AST printer
func NewPrinter(w io.Writer) *Printer {
	return &Printer{w: w, indent: 0}
}

// PrintProgram prints a complete Csharpminor program
func (p *Printer) PrintProgram(prog *Program) {
	// Print global variables
	for _, g := range prog.Globals {
		fmt.Fprintf(p.w, "var %s[%d];\n", g.Name, g.Size)
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

// printFunction prints a function definition
func (p *Printer) printFunction(fn *Function) {
	// Function signature: return_type name(params)
	fmt.Fprintf(p.w, "%s %s(", fn.Sig.Return.String(), fn.Name)
	for i, param := range fn.Params {
		if i > 0 {
			fmt.Fprint(p.w, ", ")
		}
		fmt.Fprint(p.w, param)
	}
	fmt.Fprintln(p.w, ")")
	fmt.Fprintln(p.w, "{")
	p.indent++

	// Print local variables (stack-allocated)
	for _, local := range fn.Locals {
		p.writeIndent()
		fmt.Fprintf(p.w, "var %s[%d];\n", local.Name, local.Size)
	}

	// Print temporaries
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

// printStmt prints a statement
func (p *Printer) printStmt(stmt Stmt) {
	switch s := stmt.(type) {
	case Sskip:
		// Skip produces no output

	case Sset:
		p.writeIndent()
		fmt.Fprintf(p.w, "$%d = ", s.TempID)
		p.printExpr(s.RHS)
		fmt.Fprintln(p.w, ";")

	case Sstore:
		p.writeIndent()
		fmt.Fprintf(p.w, "%s[", s.Chunk)
		p.printExpr(s.Addr)
		fmt.Fprint(p.w, "] = ")
		p.printExpr(s.Value)
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

	case Stailcall:
		p.writeIndent()
		fmt.Fprint(p.w, "tailcall ")
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

	case Sseq:
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
		p.indent--
		p.writeIndent()
		fmt.Fprintln(p.w, "}")

	case Sblock:
		p.writeIndent()
		fmt.Fprintln(p.w, "block {")
		p.indent++
		p.printStmt(s.Body)
		p.indent--
		p.writeIndent()
		fmt.Fprintln(p.w, "}")

	case Sexit:
		p.writeIndent()
		fmt.Fprintf(p.w, "exit %d;\n", s.N)

	case Sswitch:
		p.writeIndent()
		if s.IsLong {
			fmt.Fprint(p.w, "switchl (")
		} else {
			fmt.Fprint(p.w, "switch (")
		}
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

	case Sreturn:
		p.writeIndent()
		fmt.Fprint(p.w, "return")
		if s.Value != nil {
			fmt.Fprint(p.w, " ")
			p.printExpr(s.Value)
		}
		fmt.Fprintln(p.w, ";")

	case Slabel:
		// Labels are not indented
		fmt.Fprintf(p.w, "%s:\n", s.Label)
		p.printStmt(s.Body)

	case Sgoto:
		p.writeIndent()
		fmt.Fprintf(p.w, "goto %s;\n", s.Label)

	default:
		p.writeIndent()
		fmt.Fprintf(p.w, "/* unknown stmt %T */\n", stmt)
	}
}

// printExpr prints an expression
func (p *Printer) printExpr(expr Expr) {
	switch e := expr.(type) {
	case Evar:
		fmt.Fprint(p.w, e.Name)

	case Etempvar:
		fmt.Fprintf(p.w, "$%d", e.ID)

	case Eaddrof:
		fmt.Fprintf(p.w, "&%s", e.Name)

	case Econst:
		p.printConst(e.Const)

	case Eunop:
		fmt.Fprintf(p.w, "%s(", e.Op)
		p.printExpr(e.Arg)
		fmt.Fprint(p.w, ")")

	case Ebinop:
		fmt.Fprintf(p.w, "%s(", e.Op)
		p.printExpr(e.Left)
		fmt.Fprint(p.w, ", ")
		p.printExpr(e.Right)
		fmt.Fprint(p.w, ")")

	case Ecmp:
		fmt.Fprintf(p.w, "%s %s (", e.Op, e.Cmp)
		p.printExpr(e.Left)
		fmt.Fprint(p.w, ", ")
		p.printExpr(e.Right)
		fmt.Fprint(p.w, ")")

	case Eload:
		fmt.Fprintf(p.w, "%s[", e.Chunk)
		p.printExpr(e.Addr)
		fmt.Fprint(p.w, "]")

	default:
		fmt.Fprintf(p.w, "/* unknown expr %T */", expr)
	}
}

// printConst prints a constant value
func (p *Printer) printConst(c Constant) {
	switch v := c.(type) {
	case Ointconst:
		fmt.Fprintf(p.w, "%d", v.Value)
	case Ofloatconst:
		fmt.Fprintf(p.w, "%g", v.Value)
	case Olongconst:
		fmt.Fprintf(p.w, "%dL", v.Value)
	case Osingleconst:
		fmt.Fprintf(p.w, "%gf", v.Value)
	case Oaddrsymbol:
		if v.Offset == 0 {
			fmt.Fprintf(p.w, "&%s", v.Name)
		} else {
			fmt.Fprintf(p.w, "&%s+%d", v.Name, v.Offset)
		}
	default:
		fmt.Fprintf(p.w, "/* unknown const %T */", c)
	}
}

// Package cminor provides AST printing functionality for Cminor IR
package cminor

import (
	"fmt"
	"io"
	"strings"
)

// Printer outputs the Cminor AST in a human-readable format matching CompCert
type Printer struct {
	w      io.Writer
	indent int
}

// NewPrinter creates a new Cminor AST printer
func NewPrinter(w io.Writer) *Printer {
	return &Printer{w: w, indent: 0}
}

// PrintProgram prints a complete Cminor program
func (p *Printer) PrintProgram(prog *Program) {
	// Print global variables
	for _, g := range prog.Globals {
		fmt.Fprintf(p.w, "var \"%s\"[%d];\n", g.Name, g.Size)
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

// printFunction prints a function definition in Cminor format
// Format: "name"(params): return_type { stack N; var x; ... body }
func (p *Printer) printFunction(fn *Function) {
	// Function signature: "name"(params): return
	fmt.Fprintf(p.w, "\"%s\"(", fn.Name)
	for i, param := range fn.Params {
		if i > 0 {
			fmt.Fprint(p.w, ", ")
		}
		// Print parameter with type from signature if available
		if i < len(fn.Sig.Args) {
			fmt.Fprintf(p.w, "%s: %s", param, fn.Sig.Args[i])
		} else {
			fmt.Fprint(p.w, param)
		}
	}
	fmt.Fprintf(p.w, "): %s\n", fn.Sig.Return)
	fmt.Fprintln(p.w, "{")
	p.indent++

	// Print stack space if non-zero
	if fn.Stackspace > 0 {
		p.writeIndent()
		fmt.Fprintf(p.w, "stack %d;\n", fn.Stackspace)
	}

	// Print local variables
	for _, v := range fn.Vars {
		p.writeIndent()
		fmt.Fprintf(p.w, "var %s;\n", v)
	}

	if fn.Stackspace > 0 || len(fn.Vars) > 0 {
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

	case Sassign:
		p.writeIndent()
		fmt.Fprintf(p.w, "%s = ", s.Name)
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
			fmt.Fprintf(p.w, "%s = ", *s.Result)
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
			fmt.Fprintf(p.w, "%s = ", *s.Result)
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
		fmt.Fprintf(p.w, "\"%s\"", e.Name)

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

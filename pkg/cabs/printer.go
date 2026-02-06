// Package cabs provides AST printing functionality
package cabs

import (
	"fmt"
	"io"
	"strings"
)

// Printer outputs the AST in a human-readable format
type Printer struct {
	w      io.Writer
	indent int
}

// NewPrinter creates a new AST printer
func NewPrinter(w io.Writer) *Printer {
	return &Printer{w: w, indent: 0}
}

// PrintProgram prints a complete program
func (p *Printer) PrintProgram(prog *Program) {
	for _, def := range prog.Definitions {
		p.printDefinition(def)
		fmt.Fprintln(p.w)
	}
}

func (p *Printer) writeIndent() {
	fmt.Fprint(p.w, strings.Repeat("  ", p.indent))
}

func (p *Printer) printDefinition(def Definition) {
	switch d := def.(type) {
	case FunDef:
		p.printFunDef(d)
	case TypedefDef:
		p.printTypedefDef(d)
	case StructDef:
		p.printStructDef(d)
	case UnionDef:
		p.printUnionDef(d)
	case EnumDef:
		p.printEnumDef(d)
	case VarDef:
		p.printVarDef(d)
	default:
		fmt.Fprintf(p.w, "/* unknown definition %T */\n", def)
	}
}

func (p *Printer) printFunDef(f FunDef) {
	fmt.Fprintf(p.w, "%s %s(", f.ReturnType, f.Name)
	for i, param := range f.Params {
		if i > 0 {
			fmt.Fprint(p.w, ", ")
		}
		fmt.Fprintf(p.w, "%s %s", param.TypeSpec, param.Name)
	}
	if f.Variadic {
		if len(f.Params) > 0 {
			fmt.Fprint(p.w, ", ")
		}
		fmt.Fprint(p.w, "...")
	}
	if f.Body == nil {
		// Function declaration (prototype)
		fmt.Fprintln(p.w, ");")
	} else {
		// Function definition with body
		fmt.Fprintln(p.w, ")")
		p.printBlock(f.Body)
	}
}

func (p *Printer) printTypedefDef(t TypedefDef) {
	if t.InlineType != nil {
		// Handle inline struct/union/enum definitions
		switch inline := t.InlineType.(type) {
		case StructDef:
			fmt.Fprint(p.w, "typedef struct {\n")
			p.indent++
			for _, field := range inline.Fields {
				p.writeIndent()
				fmt.Fprintf(p.w, "%s %s;\n", field.TypeSpec, field.Name)
			}
			p.indent--
			fmt.Fprintf(p.w, "} %s;\n", t.Name)
		case UnionDef:
			fmt.Fprint(p.w, "typedef union {\n")
			p.indent++
			for _, field := range inline.Fields {
				p.writeIndent()
				fmt.Fprintf(p.w, "%s %s;\n", field.TypeSpec, field.Name)
			}
			p.indent--
			fmt.Fprintf(p.w, "} %s;\n", t.Name)
		case EnumDef:
			fmt.Fprint(p.w, "typedef enum {\n")
			p.indent++
			for i, val := range inline.Values {
				p.writeIndent()
				fmt.Fprint(p.w, val.Name)
				if val.Value != nil {
					fmt.Fprint(p.w, " = ")
					p.printExpr(val.Value)
				}
				if i < len(inline.Values)-1 {
					fmt.Fprint(p.w, ",")
				}
				fmt.Fprintln(p.w)
			}
			p.indent--
			fmt.Fprintf(p.w, "} %s;\n", t.Name)
		default:
			// Fall back to simple typedef
			fmt.Fprintf(p.w, "typedef %s %s;\n", t.TypeSpec, t.Name)
		}
	} else {
		fmt.Fprintf(p.w, "typedef %s %s;\n", t.TypeSpec, t.Name)
	}
}

func (p *Printer) printStructDef(s StructDef) {
	if s.Name != "" {
		fmt.Fprintf(p.w, "struct %s {\n", s.Name)
	} else {
		fmt.Fprintln(p.w, "struct {")
	}
	p.indent++
	for _, field := range s.Fields {
		p.writeIndent()
		fmt.Fprintf(p.w, "%s %s;\n", field.TypeSpec, field.Name)
	}
	p.indent--
	fmt.Fprintln(p.w, "};")
}

func (p *Printer) printUnionDef(u UnionDef) {
	if u.Name != "" {
		fmt.Fprintf(p.w, "union %s {\n", u.Name)
	} else {
		fmt.Fprintln(p.w, "union {")
	}
	p.indent++
	for _, field := range u.Fields {
		p.writeIndent()
		fmt.Fprintf(p.w, "%s %s;\n", field.TypeSpec, field.Name)
	}
	p.indent--
	fmt.Fprintln(p.w, "};")
}

func (p *Printer) printEnumDef(e EnumDef) {
	if e.Name != "" {
		fmt.Fprintf(p.w, "enum %s {\n", e.Name)
	} else {
		fmt.Fprintln(p.w, "enum {")
	}
	p.indent++
	for i, val := range e.Values {
		p.writeIndent()
		fmt.Fprint(p.w, val.Name)
		if val.Value != nil {
			fmt.Fprint(p.w, " = ")
			p.printExpr(val.Value)
		}
		if i < len(e.Values)-1 {
			fmt.Fprintln(p.w, ",")
		} else {
			fmt.Fprintln(p.w)
		}
	}
	p.indent--
	fmt.Fprintln(p.w, "};")
}

func (p *Printer) printVarDef(v VarDef) {
	if v.StorageClass != "" {
		fmt.Fprintf(p.w, "%s ", v.StorageClass)
	}
	fmt.Fprintf(p.w, "%s %s", v.TypeSpec, v.Name)
	for _, dim := range v.ArrayDims {
		fmt.Fprint(p.w, "[")
		if dim != nil {
			p.printExpr(dim)
		}
		fmt.Fprint(p.w, "]")
	}
	if v.Initializer != nil {
		fmt.Fprint(p.w, " = ")
		p.printExpr(v.Initializer)
	}
	fmt.Fprintln(p.w, ";")
}

func (p *Printer) printBlock(b *Block) {
	p.writeIndent()
	fmt.Fprintln(p.w, "{")
	p.indent++
	for _, stmt := range b.Items {
		p.printStmt(stmt)
	}
	p.indent--
	p.writeIndent()
	fmt.Fprintln(p.w, "}")
}

func (p *Printer) printStmt(stmt Stmt) {
	p.writeIndent()
	switch s := stmt.(type) {
	case Return:
		fmt.Fprint(p.w, "return")
		if s.Expr != nil {
			fmt.Fprint(p.w, " ")
			p.printExpr(s.Expr)
		}
		fmt.Fprintln(p.w, ";")
	case Computation:
		p.printExpr(s.Expr)
		fmt.Fprintln(p.w, ";")
	case If:
		fmt.Fprint(p.w, "if (")
		p.printExpr(s.Cond)
		fmt.Fprintln(p.w, ")")
		p.indent++
		p.printStmt(s.Then)
		p.indent--
		if s.Else != nil {
			p.writeIndent()
			fmt.Fprintln(p.w, "else")
			p.indent++
			p.printStmt(s.Else)
			p.indent--
		}
	case While:
		fmt.Fprint(p.w, "while (")
		p.printExpr(s.Cond)
		fmt.Fprintln(p.w, ")")
		p.indent++
		p.printStmt(s.Body)
		p.indent--
	case DoWhile:
		fmt.Fprintln(p.w, "do")
		p.indent++
		p.printStmt(s.Body)
		p.indent--
		p.writeIndent()
		fmt.Fprint(p.w, "while (")
		p.printExpr(s.Cond)
		fmt.Fprintln(p.w, ");")
	case For:
		fmt.Fprint(p.w, "for (")
		if len(s.InitDecl) > 0 {
			// C99 for-loop declaration
			p.printDeclList(s.InitDecl)
		} else if s.Init != nil {
			p.printExpr(s.Init)
		}
		fmt.Fprint(p.w, "; ")
		if s.Cond != nil {
			p.printExpr(s.Cond)
		}
		fmt.Fprint(p.w, "; ")
		if s.Step != nil {
			p.printExpr(s.Step)
		}
		fmt.Fprintln(p.w, ")")
		p.indent++
		p.printStmt(s.Body)
		p.indent--
	case Break:
		fmt.Fprintln(p.w, "break;")
	case Continue:
		fmt.Fprintln(p.w, "continue;")
	case Switch:
		fmt.Fprint(p.w, "switch (")
		p.printExpr(s.Expr)
		fmt.Fprintln(p.w, ") {")
		for _, c := range s.Cases {
			if c.Expr == nil {
				p.writeIndent()
				fmt.Fprintln(p.w, "default:")
			} else {
				p.writeIndent()
				fmt.Fprint(p.w, "case ")
				p.printExpr(c.Expr)
				fmt.Fprintln(p.w, ":")
			}
			p.indent++
			for _, cs := range c.Stmts {
				p.printStmt(cs)
			}
			p.indent--
		}
		p.writeIndent()
		fmt.Fprintln(p.w, "}")
	case Goto:
		fmt.Fprintf(p.w, "goto %s;\n", s.Label)
	case Label:
		// Labels are printed without indent
		fmt.Fprintf(p.w, "%s:\n", s.Name)
		p.printStmt(s.Stmt)
	case Block:
		// Nested block (value type)
		p.indent--
		p.printBlock(&s)
		p.indent++
	case *Block:
		// Nested block (pointer type)
		p.indent--
		p.printBlock(s)
		p.indent++
	case DeclStmt:
		for _, decl := range s.Decls {
			fmt.Fprintf(p.w, "%s %s", decl.TypeSpec, decl.Name)
			// Print array dimensions
			for _, dim := range decl.ArrayDims {
				fmt.Fprint(p.w, "[")
				if dim != nil {
					p.printExpr(dim)
				}
				fmt.Fprint(p.w, "]")
			}
			if decl.Initializer != nil {
				fmt.Fprint(p.w, " = ")
				p.printExpr(decl.Initializer)
			}
			fmt.Fprintln(p.w, ";")
			if len(s.Decls) > 1 {
				p.writeIndent() // Next decl on new line
			}
		}
	default:
		fmt.Fprintf(p.w, "/* unknown stmt %T */;\n", stmt)
	}
}

// printDeclList prints a list of declarations for C99 for-loop init (no trailing semicolon)
func (p *Printer) printDeclList(decls []Decl) {
	for i, decl := range decls {
		if i > 0 {
			fmt.Fprint(p.w, ", ")
		}
		fmt.Fprintf(p.w, "%s %s", decl.TypeSpec, decl.Name)
		for _, dim := range decl.ArrayDims {
			fmt.Fprint(p.w, "[")
			if dim != nil {
				p.printExpr(dim)
			}
			fmt.Fprint(p.w, "]")
		}
		if decl.Initializer != nil {
			fmt.Fprint(p.w, " = ")
			p.printExpr(decl.Initializer)
		}
	}
}

func (p *Printer) printExpr(expr Expr) {
	switch e := expr.(type) {
	case Constant:
		fmt.Fprintf(p.w, "%d", e.Value)
	case StringLiteral:
		fmt.Fprintf(p.w, "\"%s\"", e.Value)
	case CharLiteral:
		fmt.Fprintf(p.w, "'%s'", e.Value)
	case Variable:
		fmt.Fprint(p.w, e.Name)
	case Unary:
		p.printUnary(e)
	case Binary:
		p.printBinary(e)
	case Paren:
		fmt.Fprint(p.w, "(")
		p.printExpr(e.Expr)
		fmt.Fprint(p.w, ")")
	case Conditional:
		p.printExpr(e.Cond)
		fmt.Fprint(p.w, " ? ")
		p.printExpr(e.Then)
		fmt.Fprint(p.w, " : ")
		p.printExpr(e.Else)
	case Call:
		p.printExpr(e.Func)
		fmt.Fprint(p.w, "(")
		for i, arg := range e.Args {
			if i > 0 {
				fmt.Fprint(p.w, ", ")
			}
			p.printExpr(arg)
		}
		fmt.Fprint(p.w, ")")
	case Index:
		p.printExpr(e.Array)
		fmt.Fprint(p.w, "[")
		p.printExpr(e.Index)
		fmt.Fprint(p.w, "]")
	case Member:
		p.printExpr(e.Expr)
		if e.IsArrow {
			fmt.Fprint(p.w, "->")
		} else {
			fmt.Fprint(p.w, ".")
		}
		fmt.Fprint(p.w, e.Name)
	case SizeofExpr:
		fmt.Fprint(p.w, "sizeof ")
		p.printExpr(e.Expr)
	case SizeofType:
		fmt.Fprintf(p.w, "sizeof(%s)", e.TypeName)
	case Cast:
		fmt.Fprintf(p.w, "(%s)", e.TypeName)
		p.printExpr(e.Expr)
	default:
		fmt.Fprintf(p.w, "/* unknown expr %T */", expr)
	}
}

func (p *Printer) printUnary(u Unary) {
	switch u.Op {
	case OpNeg:
		fmt.Fprint(p.w, "-")
		p.printExpr(u.Expr)
	case OpNot:
		fmt.Fprint(p.w, "!")
		p.printExpr(u.Expr)
	case OpBitNot:
		fmt.Fprint(p.w, "~")
		p.printExpr(u.Expr)
	case OpPreInc:
		fmt.Fprint(p.w, "++")
		p.printExpr(u.Expr)
	case OpPreDec:
		fmt.Fprint(p.w, "--")
		p.printExpr(u.Expr)
	case OpPostInc:
		p.printExpr(u.Expr)
		fmt.Fprint(p.w, "++")
	case OpPostDec:
		p.printExpr(u.Expr)
		fmt.Fprint(p.w, "--")
	case OpAddrOf:
		fmt.Fprint(p.w, "&")
		p.printExpr(u.Expr)
	case OpDeref:
		fmt.Fprint(p.w, "*")
		p.printExpr(u.Expr)
	case OpPlus:
		fmt.Fprint(p.w, "+")
		p.printExpr(u.Expr)
	default:
		fmt.Fprintf(p.w, "/* unknown unary op %d */", u.Op)
		p.printExpr(u.Expr)
	}
}

func (p *Printer) printBinary(b Binary) {
	p.printExpr(b.Left)
	fmt.Fprintf(p.w, " %s ", b.Op.String())
	p.printExpr(b.Right)
}

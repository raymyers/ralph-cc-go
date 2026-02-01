// Package linear provides AST printing functionality for Linear IR
// Format is similar to CompCert's output but adapted for sequential code
package linear

import (
	"fmt"
	"io"

	"github.com/raymyers/ralph-cc/pkg/ltl"
	"github.com/raymyers/ralph-cc/pkg/rtl"
)

// Printer outputs the Linear AST in a readable format
type Printer struct {
	w io.Writer
}

// NewPrinter creates a new Linear AST printer
func NewPrinter(w io.Writer) *Printer {
	return &Printer{w: w}
}

// PrintProgram prints a complete Linear program
func (p *Printer) PrintProgram(prog *Program) {
	// Print global variables
	for _, g := range prog.Globals {
		fmt.Fprintf(p.w, "var \"%s\"[%d]\n", g.Name, g.Size)
	}
	if len(prog.Globals) > 0 {
		fmt.Fprintln(p.w)
	}

	// Print functions
	for i, fn := range prog.Functions {
		p.PrintFunction(&fn)
		if i < len(prog.Functions)-1 {
			fmt.Fprintln(p.w)
		}
	}
}

// PrintFunction prints a function in Linear format
func (p *Printer) PrintFunction(fn *Function) {
	// Function header
	fmt.Fprintf(p.w, "%s() {\n", fn.Name)

	// Stack size info
	if fn.Stacksize > 0 {
		fmt.Fprintf(p.w, "  ; stacksize = %d\n", fn.Stacksize)
	}

	// Print instructions sequentially
	for _, inst := range fn.Code {
		p.printInstruction(inst)
	}

	fmt.Fprintln(p.w, "}")
}

func (p *Printer) printInstruction(inst Instruction) {
	switch i := inst.(type) {
	case Llabel:
		// Labels are printed without indentation
		fmt.Fprintf(p.w, "L%d:\n", i.Lbl)
	case Lgetstack:
		fmt.Fprintf(p.w, "  %s = getstack(%s, %d, %s)\n",
			i.Dest.String(), i.Slot, i.Ofs, i.Ty)
	case Lsetstack:
		fmt.Fprintf(p.w, "  setstack(%s, %s, %d, %s)\n",
			i.Src.String(), i.Slot, i.Ofs, i.Ty)
	case Lop:
		fmt.Fprint(p.w, "  ")
		p.printLoc(i.Dest)
		fmt.Fprint(p.w, " = ")
		p.printOperation(i.Op)
		fmt.Fprint(p.w, "(")
		for j, arg := range i.Args {
			if j > 0 {
				fmt.Fprint(p.w, ", ")
			}
			p.printLoc(arg)
		}
		fmt.Fprintln(p.w, ")")
	case Lload:
		fmt.Fprint(p.w, "  ")
		p.printLoc(i.Dest)
		fmt.Fprintf(p.w, " = load %s, ", chunkName(i.Chunk))
		p.printAddressingMode(i.Addr)
		fmt.Fprint(p.w, "(")
		for j, arg := range i.Args {
			if j > 0 {
				fmt.Fprint(p.w, ", ")
			}
			p.printLoc(arg)
		}
		fmt.Fprintln(p.w, ")")
	case Lstore:
		fmt.Fprintf(p.w, "  store %s, ", chunkName(i.Chunk))
		p.printAddressingMode(i.Addr)
		fmt.Fprint(p.w, "(")
		for j, arg := range i.Args {
			if j > 0 {
				fmt.Fprint(p.w, ", ")
			}
			p.printLoc(arg)
		}
		fmt.Fprint(p.w, "), ")
		p.printLoc(i.Src)
		fmt.Fprintln(p.w)
	case Lcall:
		fmt.Fprint(p.w, "  call ")
		p.printFunRef(i.Fn)
		fmt.Fprintln(p.w)
	case Ltailcall:
		fmt.Fprint(p.w, "  tailcall ")
		p.printFunRef(i.Fn)
		fmt.Fprintln(p.w)
	case Lbuiltin:
		fmt.Fprintf(p.w, "  builtin %q(", i.Builtin)
		for j, arg := range i.Args {
			if j > 0 {
				fmt.Fprint(p.w, ", ")
			}
			p.printLoc(arg)
		}
		fmt.Fprint(p.w, ")")
		if i.Dest != nil {
			fmt.Fprint(p.w, " -> ")
			p.printLoc(*i.Dest)
		}
		fmt.Fprintln(p.w)
	case Lgoto:
		fmt.Fprintf(p.w, "  goto L%d\n", i.Target)
	case Lcond:
		fmt.Fprint(p.w, "  if ")
		p.printConditionCode(i.Cond, i.Args)
		fmt.Fprintf(p.w, " goto L%d\n", i.IfSo)
	case Ljumptable:
		fmt.Fprint(p.w, "  jumptable ")
		p.printLoc(i.Arg)
		fmt.Fprint(p.w, " [")
		for j, target := range i.Targets {
			if j > 0 {
				fmt.Fprint(p.w, ", ")
			}
			fmt.Fprintf(p.w, "L%d", target)
		}
		fmt.Fprintln(p.w, "]")
	case Lreturn:
		fmt.Fprintln(p.w, "  return")
	default:
		fmt.Fprintf(p.w, "  ??? (%T)\n", inst)
	}
}

func (p *Printer) printLoc(loc Loc) {
	switch l := loc.(type) {
	case R:
		fmt.Fprint(p.w, l.Reg.String())
	case S:
		fmt.Fprintf(p.w, "S(%s, %d, %s)", l.Slot, l.Ofs, l.Ty)
	default:
		fmt.Fprint(p.w, "?loc")
	}
}

func (p *Printer) printOperation(op Operation) {
	switch o := op.(type) {
	case rtl.Omove:
		fmt.Fprint(p.w, "move")
	case rtl.Ointconst:
		fmt.Fprintf(p.w, "int %d", o.Value)
	case rtl.Olongconst:
		fmt.Fprintf(p.w, "long %dL", o.Value)
	case rtl.Ofloatconst:
		fmt.Fprintf(p.w, "float %v", o.Value)
	case rtl.Osingleconst:
		fmt.Fprintf(p.w, "single %vf", o.Value)
	case rtl.Oaddrsymbol:
		fmt.Fprintf(p.w, "addrsymbol \"%s\"+%d", o.Symbol, o.Offset)
	case rtl.Oaddrstack:
		fmt.Fprintf(p.w, "addrstack %d", o.Offset)
	case rtl.Oadd:
		fmt.Fprint(p.w, "add")
	case rtl.Oaddimm:
		fmt.Fprintf(p.w, "addimm %d", o.N)
	case rtl.Oneg:
		fmt.Fprint(p.w, "neg")
	case rtl.Osub:
		fmt.Fprint(p.w, "sub")
	case rtl.Omul:
		fmt.Fprint(p.w, "mul")
	case rtl.Omulimm:
		fmt.Fprintf(p.w, "mulimm %d", o.N)
	case rtl.Odiv:
		fmt.Fprint(p.w, "div")
	case rtl.Odivu:
		fmt.Fprint(p.w, "divu")
	case rtl.Omod:
		fmt.Fprint(p.w, "mod")
	case rtl.Omodu:
		fmt.Fprint(p.w, "modu")
	case rtl.Oand:
		fmt.Fprint(p.w, "and")
	case rtl.Oandimm:
		fmt.Fprintf(p.w, "andimm %d", o.N)
	case rtl.Oor:
		fmt.Fprint(p.w, "or")
	case rtl.Oorimm:
		fmt.Fprintf(p.w, "orimm %d", o.N)
	case rtl.Oxor:
		fmt.Fprint(p.w, "xor")
	case rtl.Oxorimm:
		fmt.Fprintf(p.w, "xorimm %d", o.N)
	case rtl.Onot:
		fmt.Fprint(p.w, "not")
	case rtl.Oshl:
		fmt.Fprint(p.w, "shl")
	case rtl.Oshlimm:
		fmt.Fprintf(p.w, "shlimm %d", o.N)
	case rtl.Oshr:
		fmt.Fprint(p.w, "shr")
	case rtl.Oshrimm:
		fmt.Fprintf(p.w, "shrimm %d", o.N)
	case rtl.Oshru:
		fmt.Fprint(p.w, "shru")
	case rtl.Oshruimm:
		fmt.Fprintf(p.w, "shruimm %d", o.N)
	case rtl.Oaddl:
		fmt.Fprint(p.w, "addl")
	case rtl.Oaddlimm:
		fmt.Fprintf(p.w, "addlimm %dL", o.N)
	case rtl.Onegl:
		fmt.Fprint(p.w, "negl")
	case rtl.Osubl:
		fmt.Fprint(p.w, "subl")
	case rtl.Omull:
		fmt.Fprint(p.w, "mull")
	case rtl.Odivl:
		fmt.Fprint(p.w, "divl")
	case rtl.Odivlu:
		fmt.Fprint(p.w, "divlu")
	case rtl.Omodl:
		fmt.Fprint(p.w, "modl")
	case rtl.Omodlu:
		fmt.Fprint(p.w, "modlu")
	case rtl.Oandl:
		fmt.Fprint(p.w, "andl")
	case rtl.Oorl:
		fmt.Fprint(p.w, "orl")
	case rtl.Oxorl:
		fmt.Fprint(p.w, "xorl")
	case rtl.Onotl:
		fmt.Fprint(p.w, "notl")
	case rtl.Oshll:
		fmt.Fprint(p.w, "shll")
	case rtl.Oshllimm:
		fmt.Fprintf(p.w, "shllimm %d", o.N)
	case rtl.Oshrl:
		fmt.Fprint(p.w, "shrl")
	case rtl.Oshrlimm:
		fmt.Fprintf(p.w, "shrlimm %d", o.N)
	case rtl.Oshrlu:
		fmt.Fprint(p.w, "shrlu")
	case rtl.Oshrluimm:
		fmt.Fprintf(p.w, "shrluimm %d", o.N)
	case rtl.Ocast8signed:
		fmt.Fprint(p.w, "cast8s")
	case rtl.Ocast8unsigned:
		fmt.Fprint(p.w, "cast8u")
	case rtl.Ocast16signed:
		fmt.Fprint(p.w, "cast16s")
	case rtl.Ocast16unsigned:
		fmt.Fprint(p.w, "cast16u")
	case rtl.Olongofint:
		fmt.Fprint(p.w, "longofint")
	case rtl.Olongofintu:
		fmt.Fprint(p.w, "longofintu")
	case rtl.Ointoflong:
		fmt.Fprint(p.w, "intoflong")
	case rtl.Onegf:
		fmt.Fprint(p.w, "negf")
	case rtl.Oabsf:
		fmt.Fprint(p.w, "absf")
	case rtl.Oaddf:
		fmt.Fprint(p.w, "addf")
	case rtl.Osubf:
		fmt.Fprint(p.w, "subf")
	case rtl.Omulf:
		fmt.Fprint(p.w, "mulf")
	case rtl.Odivf:
		fmt.Fprint(p.w, "divf")
	case rtl.Onegs:
		fmt.Fprint(p.w, "negs")
	case rtl.Oabss:
		fmt.Fprint(p.w, "abss")
	case rtl.Oadds:
		fmt.Fprint(p.w, "adds")
	case rtl.Osubs:
		fmt.Fprint(p.w, "subs")
	case rtl.Omuls:
		fmt.Fprint(p.w, "muls")
	case rtl.Odivs:
		fmt.Fprint(p.w, "divs")
	case rtl.Osingleoffloat:
		fmt.Fprint(p.w, "singleoffloat")
	case rtl.Ofloatofsingle:
		fmt.Fprint(p.w, "floatofsingle")
	case rtl.Ointoffloat:
		fmt.Fprint(p.w, "intoffloat")
	case rtl.Ointuoffloat:
		fmt.Fprint(p.w, "intuoffloat")
	case rtl.Ofloatofint:
		fmt.Fprint(p.w, "floatofint")
	case rtl.Ofloatofintu:
		fmt.Fprint(p.w, "floatofintu")
	case rtl.Olongoffloat:
		fmt.Fprint(p.w, "longoffloat")
	case rtl.Olonguoffloat:
		fmt.Fprint(p.w, "longuoffloat")
	case rtl.Ofloatoflong:
		fmt.Fprint(p.w, "floatoflong")
	case rtl.Ofloatoflongu:
		fmt.Fprint(p.w, "floatoflongu")
	case rtl.Ocmp:
		fmt.Fprintf(p.w, "cmp %s", o.Cond)
	case rtl.Ocmpu:
		fmt.Fprintf(p.w, "cmpu %s", o.Cond)
	case rtl.Ocmpimm:
		fmt.Fprintf(p.w, "cmpimm %s %d", o.Cond, o.N)
	case rtl.Ocmpuimm:
		fmt.Fprintf(p.w, "cmpuimm %s %d", o.Cond, o.N)
	case rtl.Ocmpl:
		fmt.Fprintf(p.w, "cmpl %s", o.Cond)
	case rtl.Ocmplu:
		fmt.Fprintf(p.w, "cmplu %s", o.Cond)
	case rtl.Ocmplimm:
		fmt.Fprintf(p.w, "cmplimm %s %dL", o.Cond, o.N)
	case rtl.Ocmpluimm:
		fmt.Fprintf(p.w, "cmpluimm %s %dL", o.Cond, o.N)
	case rtl.Ocmpf:
		fmt.Fprintf(p.w, "cmpf %s", o.Cond)
	case rtl.Ocmps:
		fmt.Fprintf(p.w, "cmps %s", o.Cond)
	default:
		fmt.Fprintf(p.w, "op?(%T)", op)
	}
}

func (p *Printer) printAddressingMode(addr AddressingMode) {
	switch a := addr.(type) {
	case ltl.Aindexed:
		fmt.Fprintf(p.w, "[+%d]", a.Offset)
	case ltl.Aindexed2:
		fmt.Fprint(p.w, "[+reg]")
	case ltl.Aindexed2shift:
		fmt.Fprintf(p.w, "[+reg<<%d]", a.Shift)
	case ltl.Aglobal:
		fmt.Fprintf(p.w, "[\"%s\"+%d]", a.Symbol, a.Offset)
	case ltl.Ainstack:
		fmt.Fprintf(p.w, "[sp+%d]", a.Offset)
	default:
		fmt.Fprint(p.w, "[addr?]")
	}
}

func (p *Printer) printFunRef(fn FunRef) {
	switch f := fn.(type) {
	case FunSymbol:
		fmt.Fprintf(p.w, "\"%s\"", f.Name)
	case FunReg:
		fmt.Fprint(p.w, "*")
		p.printLoc(f.Loc)
	}
}

func (p *Printer) printConditionCode(cc ConditionCode, args []Loc) {
	// Print args first
	if len(args) > 0 {
		p.printLoc(args[0])
	}

	switch c := cc.(type) {
	case rtl.Ccomp:
		fmt.Fprintf(p.w, " %s ", c.Cond)
		if len(args) > 1 {
			p.printLoc(args[1])
		}
	case rtl.Ccompu:
		fmt.Fprintf(p.w, " %su ", c.Cond)
		if len(args) > 1 {
			p.printLoc(args[1])
		}
	case rtl.Ccompimm:
		fmt.Fprintf(p.w, " %s %d", c.Cond, c.N)
	case rtl.Ccompuimm:
		fmt.Fprintf(p.w, " %su %d", c.Cond, c.N)
	case rtl.Ccompl:
		fmt.Fprintf(p.w, " %sl ", c.Cond)
		if len(args) > 1 {
			p.printLoc(args[1])
		}
	case rtl.Ccomplu:
		fmt.Fprintf(p.w, " %slu ", c.Cond)
		if len(args) > 1 {
			p.printLoc(args[1])
		}
	case rtl.Ccomplimm:
		fmt.Fprintf(p.w, " %sl %dL", c.Cond, c.N)
	case rtl.Ccompluimm:
		fmt.Fprintf(p.w, " %slu %dL", c.Cond, c.N)
	case rtl.Ccompf:
		fmt.Fprintf(p.w, " %sf ", c.Cond)
		if len(args) > 1 {
			p.printLoc(args[1])
		}
	case rtl.Cnotcompf:
		fmt.Fprintf(p.w, " !%sf ", c.Cond)
		if len(args) > 1 {
			p.printLoc(args[1])
		}
	case rtl.Ccomps:
		fmt.Fprintf(p.w, " %ss ", c.Cond)
		if len(args) > 1 {
			p.printLoc(args[1])
		}
	case rtl.Cnotcomps:
		fmt.Fprintf(p.w, " !%ss ", c.Cond)
		if len(args) > 1 {
			p.printLoc(args[1])
		}
	default:
		fmt.Fprint(p.w, " ??? ")
	}
}

func chunkName(c Chunk) string {
	switch c {
	case Mint8signed:
		return "i8s"
	case Mint8unsigned:
		return "i8u"
	case Mint16signed:
		return "i16s"
	case Mint16unsigned:
		return "i16u"
	case Mint32:
		return "i32"
	case Mint64:
		return "i64"
	case Mfloat32:
		return "f32"
	case Mfloat64:
		return "f64"
	default:
		return "mem?"
	}
}

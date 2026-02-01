// Package rtl provides AST printing functionality for RTL IR
// Format matches CompCert's .rtl.0 output
package rtl

import (
	"fmt"
	"io"
	"sort"
)

// Printer outputs the RTL AST in CompCert-compatible format
type Printer struct {
	w io.Writer
}

// NewPrinter creates a new RTL AST printer
func NewPrinter(w io.Writer) *Printer {
	return &Printer{w: w}
}

// PrintProgram prints a complete RTL program
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

// PrintFunction prints a function in RTL format
func (p *Printer) PrintFunction(fn *Function) {
	// Function header: name(params)
	fmt.Fprintf(p.w, "%s(", fn.Name)
	for i, r := range fn.Params {
		if i > 0 {
			fmt.Fprint(p.w, ", ")
		}
		fmt.Fprintf(p.w, "x%d", r)
	}
	fmt.Fprintln(p.w, ") {")

	// Sort nodes for deterministic output
	nodes := make([]Node, 0, len(fn.Code))
	for n := range fn.Code {
		nodes = append(nodes, n)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i] < nodes[j]
	})

	// Print each instruction
	for _, n := range nodes {
		instr := fn.Code[n]
		fmt.Fprintf(p.w, "  %d: ", n)
		p.printInstruction(instr)
		fmt.Fprintln(p.w)
	}

	fmt.Fprintln(p.w, "}")
	fmt.Fprintf(p.w, "entry: %d\n", fn.Entrypoint)
}

func (p *Printer) printInstruction(instr Instruction) {
	switch i := instr.(type) {
	case Inop:
		fmt.Fprintf(p.w, "nop goto %d", i.Succ)
	case Iop:
		p.printOp(i)
	case Iload:
		p.printLoad(i)
	case Istore:
		p.printStore(i)
	case Icall:
		p.printCall(i)
	case Itailcall:
		p.printTailcall(i)
	case Ibuiltin:
		p.printBuiltin(i)
	case Icond:
		p.printCond(i)
	case Ijumptable:
		p.printJumptable(i)
	case Ireturn:
		p.printReturn(i)
	default:
		fmt.Fprint(p.w, "???")
	}
}

func (p *Printer) printOp(i Iop) {
	fmt.Fprintf(p.w, "x%d = ", i.Dest)
	p.printOperation(i.Op)
	fmt.Fprint(p.w, "(")
	for j, r := range i.Args {
		if j > 0 {
			fmt.Fprint(p.w, ", ")
		}
		fmt.Fprintf(p.w, "x%d", r)
	}
	fmt.Fprintf(p.w, ") goto %d", i.Succ)
}

func (p *Printer) printOperation(op Operation) {
	switch o := op.(type) {
	case Omove:
		fmt.Fprint(p.w, "move")
	case Ointconst:
		fmt.Fprintf(p.w, "int %d", o.Value)
	case Olongconst:
		fmt.Fprintf(p.w, "long %dL", o.Value)
	case Ofloatconst:
		fmt.Fprintf(p.w, "float %v", o.Value)
	case Osingleconst:
		fmt.Fprintf(p.w, "single %vf", o.Value)
	case Oaddrsymbol:
		fmt.Fprintf(p.w, "addrsymbol \"%s\" %d", o.Symbol, o.Offset)
	case Oaddrstack:
		fmt.Fprintf(p.w, "addrstack %d", o.Offset)
	case Oadd:
		fmt.Fprint(p.w, "add")
	case Oaddimm:
		fmt.Fprintf(p.w, "addimm %d", o.N)
	case Oneg:
		fmt.Fprint(p.w, "neg")
	case Osub:
		fmt.Fprint(p.w, "sub")
	case Omul:
		fmt.Fprint(p.w, "mul")
	case Omulimm:
		fmt.Fprintf(p.w, "mulimm %d", o.N)
	case Odiv:
		fmt.Fprint(p.w, "divs")
	case Odivu:
		fmt.Fprint(p.w, "divu")
	case Omod:
		fmt.Fprint(p.w, "mods")
	case Omodu:
		fmt.Fprint(p.w, "modu")
	case Oand:
		fmt.Fprint(p.w, "and")
	case Oandimm:
		fmt.Fprintf(p.w, "andimm %d", o.N)
	case Oor:
		fmt.Fprint(p.w, "or")
	case Oorimm:
		fmt.Fprintf(p.w, "orimm %d", o.N)
	case Oxor:
		fmt.Fprint(p.w, "xor")
	case Oxorimm:
		fmt.Fprintf(p.w, "xorimm %d", o.N)
	case Onot:
		fmt.Fprint(p.w, "not")
	case Oshl:
		fmt.Fprint(p.w, "shl")
	case Oshlimm:
		fmt.Fprintf(p.w, "shlimm %d", o.N)
	case Oshr:
		fmt.Fprint(p.w, "shr")
	case Oshrimm:
		fmt.Fprintf(p.w, "shrimm %d", o.N)
	case Oshru:
		fmt.Fprint(p.w, "shru")
	case Oshruimm:
		fmt.Fprintf(p.w, "shruimm %d", o.N)
	case Oaddl:
		fmt.Fprint(p.w, "addl")
	case Oaddlimm:
		fmt.Fprintf(p.w, "addlimm %dL", o.N)
	case Onegl:
		fmt.Fprint(p.w, "negl")
	case Osubl:
		fmt.Fprint(p.w, "subl")
	case Omull:
		fmt.Fprint(p.w, "mull")
	case Odivl:
		fmt.Fprint(p.w, "divls")
	case Odivlu:
		fmt.Fprint(p.w, "divlu")
	case Omodl:
		fmt.Fprint(p.w, "modls")
	case Omodlu:
		fmt.Fprint(p.w, "modlu")
	case Oandl:
		fmt.Fprint(p.w, "andl")
	case Oorl:
		fmt.Fprint(p.w, "orl")
	case Oxorl:
		fmt.Fprint(p.w, "xorl")
	case Onotl:
		fmt.Fprint(p.w, "notl")
	case Oshll:
		fmt.Fprint(p.w, "shll")
	case Oshllimm:
		fmt.Fprintf(p.w, "shllimm %d", o.N)
	case Oshrl:
		fmt.Fprint(p.w, "shrl")
	case Oshrlimm:
		fmt.Fprintf(p.w, "shrlimm %d", o.N)
	case Oshrlu:
		fmt.Fprint(p.w, "shrlu")
	case Oshrluimm:
		fmt.Fprintf(p.w, "shrluimm %d", o.N)
	case Ocast8signed:
		fmt.Fprint(p.w, "cast8signed")
	case Ocast8unsigned:
		fmt.Fprint(p.w, "cast8unsigned")
	case Ocast16signed:
		fmt.Fprint(p.w, "cast16signed")
	case Ocast16unsigned:
		fmt.Fprint(p.w, "cast16unsigned")
	case Olongofint:
		fmt.Fprint(p.w, "longofint")
	case Olongofintu:
		fmt.Fprint(p.w, "longofintu")
	case Ointoflong:
		fmt.Fprint(p.w, "intoflong")
	case Onegf:
		fmt.Fprint(p.w, "negf")
	case Oabsf:
		fmt.Fprint(p.w, "absf")
	case Oaddf:
		fmt.Fprint(p.w, "addf")
	case Osubf:
		fmt.Fprint(p.w, "subf")
	case Omulf:
		fmt.Fprint(p.w, "mulf")
	case Odivf:
		fmt.Fprint(p.w, "divf")
	case Onegs:
		fmt.Fprint(p.w, "negs")
	case Oabss:
		fmt.Fprint(p.w, "abss")
	case Oadds:
		fmt.Fprint(p.w, "adds")
	case Osubs:
		fmt.Fprint(p.w, "subs")
	case Omuls:
		fmt.Fprint(p.w, "muls")
	case Odivs:
		fmt.Fprint(p.w, "divs")
	case Osingleoffloat:
		fmt.Fprint(p.w, "singleoffloat")
	case Ofloatofsingle:
		fmt.Fprint(p.w, "floatofsingle")
	case Ointoffloat:
		fmt.Fprint(p.w, "intoffloat")
	case Ointuoffloat:
		fmt.Fprint(p.w, "intuoffloat")
	case Ofloatofint:
		fmt.Fprint(p.w, "floatofint")
	case Ofloatofintu:
		fmt.Fprint(p.w, "floatofintu")
	case Olongoffloat:
		fmt.Fprint(p.w, "longoffloat")
	case Olonguoffloat:
		fmt.Fprint(p.w, "longuoffloat")
	case Ofloatoflong:
		fmt.Fprint(p.w, "floatoflong")
	case Ofloatoflongu:
		fmt.Fprint(p.w, "floatoflongu")
	case Ocmp:
		fmt.Fprintf(p.w, "cmp %s", o.Cond)
	case Ocmpu:
		fmt.Fprintf(p.w, "cmpu %s", o.Cond)
	case Ocmpimm:
		fmt.Fprintf(p.w, "cmpimm %s %d", o.Cond, o.N)
	case Ocmpuimm:
		fmt.Fprintf(p.w, "cmpuimm %s %d", o.Cond, o.N)
	case Ocmpl:
		fmt.Fprintf(p.w, "cmpl %s", o.Cond)
	case Ocmplu:
		fmt.Fprintf(p.w, "cmplu %s", o.Cond)
	case Ocmplimm:
		fmt.Fprintf(p.w, "cmplimm %s %dL", o.Cond, o.N)
	case Ocmpluimm:
		fmt.Fprintf(p.w, "cmpluimm %s %dL", o.Cond, o.N)
	case Ocmpf:
		fmt.Fprintf(p.w, "cmpf %s", o.Cond)
	case Ocmps:
		fmt.Fprintf(p.w, "cmps %s", o.Cond)
	default:
		fmt.Fprintf(p.w, "op?(%T)", op)
	}
}

func (p *Printer) printLoad(i Iload) {
	fmt.Fprintf(p.w, "x%d = %s[", i.Dest, chunkName(i.Chunk))
	p.printAddressingMode(i.Addr, i.Args)
	fmt.Fprintf(p.w, "] goto %d", i.Succ)
}

func (p *Printer) printStore(i Istore) {
	fmt.Fprintf(p.w, "%s[", chunkName(i.Chunk))
	p.printAddressingMode(i.Addr, i.Args)
	fmt.Fprintf(p.w, "] = x%d goto %d", i.Src, i.Succ)
}

func (p *Printer) printAddressingMode(addr AddressingMode, args []Reg) {
	switch a := addr.(type) {
	case Aindexed:
		if len(args) > 0 {
			fmt.Fprintf(p.w, "x%d + %d", args[0], a.Offset)
		} else {
			fmt.Fprintf(p.w, "%d", a.Offset)
		}
	case Aindexed2:
		if len(args) >= 2 {
			fmt.Fprintf(p.w, "x%d + x%d", args[0], args[1])
		}
	case Aindexed2shift:
		if len(args) >= 2 {
			fmt.Fprintf(p.w, "x%d + x%d << %d", args[0], args[1], a.Shift)
		}
	case Aglobal:
		fmt.Fprintf(p.w, "\"%s\" + %d", a.Symbol, a.Offset)
	case Ainstack:
		fmt.Fprintf(p.w, "stack(%d)", a.Offset)
	default:
		fmt.Fprint(p.w, "addr?")
	}
}

func (p *Printer) printCall(i Icall) {
	if i.Dest != 0 {
		fmt.Fprintf(p.w, "x%d = ", i.Dest)
	}
	fmt.Fprint(p.w, "call ")
	p.printFunRef(i.Fn)
	fmt.Fprint(p.w, "(")
	for j, r := range i.Args {
		if j > 0 {
			fmt.Fprint(p.w, ", ")
		}
		fmt.Fprintf(p.w, "x%d", r)
	}
	fmt.Fprintf(p.w, ") goto %d", i.Succ)
}

func (p *Printer) printTailcall(i Itailcall) {
	fmt.Fprint(p.w, "tailcall ")
	p.printFunRef(i.Fn)
	fmt.Fprint(p.w, "(")
	for j, r := range i.Args {
		if j > 0 {
			fmt.Fprint(p.w, ", ")
		}
		fmt.Fprintf(p.w, "x%d", r)
	}
	fmt.Fprint(p.w, ")")
}

func (p *Printer) printFunRef(fn FunRef) {
	switch f := fn.(type) {
	case FunSymbol:
		fmt.Fprintf(p.w, "\"%s\"", f.Name)
	case FunReg:
		fmt.Fprintf(p.w, "x%d", f.Reg)
	}
}

func (p *Printer) printBuiltin(i Ibuiltin) {
	if i.Dest != nil {
		fmt.Fprintf(p.w, "x%d = ", *i.Dest)
	}
	fmt.Fprintf(p.w, "builtin %q(", i.Builtin)
	for j, r := range i.Args {
		if j > 0 {
			fmt.Fprint(p.w, ", ")
		}
		fmt.Fprintf(p.w, "x%d", r)
	}
	fmt.Fprintf(p.w, ") goto %d", i.Succ)
}

func (p *Printer) printCond(i Icond) {
	fmt.Fprint(p.w, "if ")
	p.printConditionCode(i.Cond, i.Args)
	fmt.Fprintf(p.w, " goto %d else goto %d", i.IfSo, i.IfNot)
}

func (p *Printer) printConditionCode(cc ConditionCode, args []Reg) {
	switch c := cc.(type) {
	case Ccomp:
		if len(args) >= 2 {
			fmt.Fprintf(p.w, "x%d %s x%d", args[0], c.Cond, args[1])
		}
	case Ccompu:
		if len(args) >= 2 {
			fmt.Fprintf(p.w, "x%d %su x%d", args[0], c.Cond, args[1])
		}
	case Ccompimm:
		if len(args) >= 1 {
			fmt.Fprintf(p.w, "x%d %s %d", args[0], c.Cond, c.N)
		}
	case Ccompuimm:
		if len(args) >= 1 {
			fmt.Fprintf(p.w, "x%d %su %d", args[0], c.Cond, c.N)
		}
	case Ccompl:
		if len(args) >= 2 {
			fmt.Fprintf(p.w, "x%d %sl x%d", args[0], c.Cond, args[1])
		}
	case Ccomplu:
		if len(args) >= 2 {
			fmt.Fprintf(p.w, "x%d %slu x%d", args[0], c.Cond, args[1])
		}
	case Ccomplimm:
		if len(args) >= 1 {
			fmt.Fprintf(p.w, "x%d %sl %dL", args[0], c.Cond, c.N)
		}
	case Ccompluimm:
		if len(args) >= 1 {
			fmt.Fprintf(p.w, "x%d %slu %dL", args[0], c.Cond, c.N)
		}
	case Ccompf:
		if len(args) >= 2 {
			fmt.Fprintf(p.w, "x%d %sf x%d", args[0], c.Cond, args[1])
		}
	case Cnotcompf:
		if len(args) >= 2 {
			fmt.Fprintf(p.w, "!(x%d %sf x%d)", args[0], c.Cond, args[1])
		}
	case Ccomps:
		if len(args) >= 2 {
			fmt.Fprintf(p.w, "x%d %ss x%d", args[0], c.Cond, args[1])
		}
	case Cnotcomps:
		if len(args) >= 2 {
			fmt.Fprintf(p.w, "!(x%d %ss x%d)", args[0], c.Cond, args[1])
		}
	default:
		fmt.Fprint(p.w, "cond?")
	}
}

func (p *Printer) printJumptable(i Ijumptable) {
	fmt.Fprintf(p.w, "jumptable x%d [", i.Arg)
	for j, t := range i.Targets {
		if j > 0 {
			fmt.Fprint(p.w, ", ")
		}
		fmt.Fprintf(p.w, "%d", t)
	}
	fmt.Fprint(p.w, "]")
}

func (p *Printer) printReturn(i Ireturn) {
	if i.Arg != nil {
		fmt.Fprintf(p.w, "return x%d", *i.Arg)
	} else {
		fmt.Fprint(p.w, "return")
	}
}

func chunkName(c Chunk) string {
	switch c {
	case Mint8signed:
		return "int8s"
	case Mint8unsigned:
		return "int8u"
	case Mint16signed:
		return "int16s"
	case Mint16unsigned:
		return "int16u"
	case Mint32:
		return "int32"
	case Mint64:
		return "int64"
	case Mfloat32:
		return "float32"
	case Mfloat64:
		return "float64"
	default:
		return "mem"
	}
}

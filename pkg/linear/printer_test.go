package linear

import (
	"bytes"
	"strings"
	"testing"

	"github.com/raymyers/ralph-cc/pkg/ltl"
	"github.com/raymyers/ralph-cc/pkg/rtl"
)

func TestPrinterEmpty(t *testing.T) {
	fn := NewFunction("empty", Sig{})

	var buf bytes.Buffer
	p := NewPrinter(&buf)
	p.PrintFunction(fn)

	output := buf.String()
	if !strings.Contains(output, "empty()") {
		t.Errorf("Expected function header, got: %s", output)
	}
}

func TestPrinterLabel(t *testing.T) {
	fn := NewFunction("labels", Sig{})
	fn.Append(Llabel{Lbl: 1})
	fn.Append(Lreturn{})

	var buf bytes.Buffer
	p := NewPrinter(&buf)
	p.PrintFunction(fn)

	output := buf.String()
	if !strings.Contains(output, "L1:") {
		t.Errorf("Expected label L1, got: %s", output)
	}
	if !strings.Contains(output, "return") {
		t.Errorf("Expected return, got: %s", output)
	}
}

func TestPrinterOp(t *testing.T) {
	fn := NewFunction("ops", Sig{})
	fn.Append(Llabel{Lbl: 1})
	fn.Append(Lop{Op: rtl.Ointconst{Value: 42}, Dest: R{Reg: ltl.X0}})
	fn.Append(Lreturn{})

	var buf bytes.Buffer
	p := NewPrinter(&buf)
	p.PrintFunction(fn)

	output := buf.String()
	if !strings.Contains(output, "X0 = int 42") {
		t.Errorf("Expected int const op, got: %s", output)
	}
}

func TestPrinterGoto(t *testing.T) {
	fn := NewFunction("gotos", Sig{})
	fn.Append(Llabel{Lbl: 1})
	fn.Append(Lgoto{Target: 2})
	fn.Append(Llabel{Lbl: 2})
	fn.Append(Lreturn{})

	var buf bytes.Buffer
	p := NewPrinter(&buf)
	p.PrintFunction(fn)

	output := buf.String()
	if !strings.Contains(output, "goto L2") {
		t.Errorf("Expected goto L2, got: %s", output)
	}
}

func TestPrinterCond(t *testing.T) {
	fn := NewFunction("conds", Sig{})
	cond := rtl.Ccompimm{Cond: rtl.Ceq, N: 0}
	fn.Append(Llabel{Lbl: 1})
	fn.Append(Lcond{Cond: cond, Args: []Loc{R{Reg: ltl.X0}}, IfSo: 2})
	fn.Append(Lgoto{Target: 3})
	fn.Append(Llabel{Lbl: 2})
	fn.Append(Lreturn{})

	var buf bytes.Buffer
	p := NewPrinter(&buf)
	p.PrintFunction(fn)

	output := buf.String()
	if !strings.Contains(output, "if X0") && !strings.Contains(output, "goto L2") {
		t.Errorf("Expected conditional branch, got: %s", output)
	}
}

func TestPrinterJumptable(t *testing.T) {
	fn := NewFunction("jt", Sig{})
	fn.Append(Llabel{Lbl: 1})
	fn.Append(Ljumptable{Arg: R{Reg: ltl.X0}, Targets: []Label{2, 3, 4}})
	fn.Append(Llabel{Lbl: 2})
	fn.Append(Lreturn{})

	var buf bytes.Buffer
	p := NewPrinter(&buf)
	p.PrintFunction(fn)

	output := buf.String()
	if !strings.Contains(output, "jumptable") {
		t.Errorf("Expected jumptable, got: %s", output)
	}
	if !strings.Contains(output, "L2") || !strings.Contains(output, "L3") {
		t.Errorf("Expected jump targets, got: %s", output)
	}
}

func TestPrinterCall(t *testing.T) {
	fn := NewFunction("caller", Sig{})
	fn.Append(Llabel{Lbl: 1})
	fn.Append(Lcall{Sig: Sig{}, Fn: FunSymbol{Name: "callee"}})
	fn.Append(Lreturn{})

	var buf bytes.Buffer
	p := NewPrinter(&buf)
	p.PrintFunction(fn)

	output := buf.String()
	if !strings.Contains(output, "call \"callee\"") {
		t.Errorf("Expected call callee, got: %s", output)
	}
}

func TestPrinterTailcall(t *testing.T) {
	fn := NewFunction("tailcaller", Sig{})
	fn.Append(Llabel{Lbl: 1})
	fn.Append(Ltailcall{Sig: Sig{}, Fn: FunSymbol{Name: "target"}})

	var buf bytes.Buffer
	p := NewPrinter(&buf)
	p.PrintFunction(fn)

	output := buf.String()
	if !strings.Contains(output, "tailcall \"target\"") {
		t.Errorf("Expected tailcall, got: %s", output)
	}
}

func TestPrinterLoadStore(t *testing.T) {
	fn := NewFunction("loadstore", Sig{})
	fn.Append(Llabel{Lbl: 1})
	fn.Append(Lload{
		Chunk: Mint64,
		Addr:  ltl.Aindexed{Offset: 0},
		Args:  []Loc{R{Reg: ltl.X1}},
		Dest:  R{Reg: ltl.X0},
	})
	fn.Append(Lstore{
		Chunk: Mint64,
		Addr:  ltl.Aindexed{Offset: 8},
		Args:  []Loc{R{Reg: ltl.X1}},
		Src:   R{Reg: ltl.X0},
	})
	fn.Append(Lreturn{})

	var buf bytes.Buffer
	p := NewPrinter(&buf)
	p.PrintFunction(fn)

	output := buf.String()
	if !strings.Contains(output, "load i64") {
		t.Errorf("Expected load, got: %s", output)
	}
	if !strings.Contains(output, "store i64") {
		t.Errorf("Expected store, got: %s", output)
	}
}

func TestPrinterStackOps(t *testing.T) {
	fn := NewFunction("stack", Sig{})
	fn.Stacksize = 32
	fn.Append(Llabel{Lbl: 1})
	fn.Append(Lgetstack{Slot: SlotLocal, Ofs: 0, Ty: Tlong, Dest: ltl.X0})
	fn.Append(Lsetstack{Src: ltl.X0, Slot: SlotLocal, Ofs: 8, Ty: Tlong})
	fn.Append(Lreturn{})

	var buf bytes.Buffer
	p := NewPrinter(&buf)
	p.PrintFunction(fn)

	output := buf.String()
	if !strings.Contains(output, "stacksize = 32") {
		t.Errorf("Expected stack size comment, got: %s", output)
	}
	if !strings.Contains(output, "getstack") {
		t.Errorf("Expected getstack, got: %s", output)
	}
	if !strings.Contains(output, "setstack") {
		t.Errorf("Expected setstack, got: %s", output)
	}
}

func TestPrinterBuiltin(t *testing.T) {
	fn := NewFunction("builtin", Sig{})
	fn.Append(Llabel{Lbl: 1})
	fn.Append(Lbuiltin{Builtin: "memcpy", Args: []Loc{R{Reg: ltl.X0}, R{Reg: ltl.X1}}})
	fn.Append(Lreturn{})

	var buf bytes.Buffer
	p := NewPrinter(&buf)
	p.PrintFunction(fn)

	output := buf.String()
	if !strings.Contains(output, "builtin \"memcpy\"") {
		t.Errorf("Expected builtin memcpy, got: %s", output)
	}
}

func TestPrinterProgram(t *testing.T) {
	prog := &Program{
		Globals: []GlobVar{
			{Name: "x", Size: 8},
		},
		Functions: []Function{
			*NewFunction("main", Sig{}),
		},
	}
	prog.Functions[0].Append(Llabel{Lbl: 1})
	prog.Functions[0].Append(Lreturn{})

	var buf bytes.Buffer
	p := NewPrinter(&buf)
	p.PrintProgram(prog)

	output := buf.String()
	if !strings.Contains(output, "var \"x\"[8]") {
		t.Errorf("Expected global var, got: %s", output)
	}
	if !strings.Contains(output, "main()") {
		t.Errorf("Expected main function, got: %s", output)
	}
}

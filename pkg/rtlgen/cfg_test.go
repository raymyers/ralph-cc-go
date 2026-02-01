package rtlgen

import (
	"testing"

	"github.com/raymyers/ralph-cc/pkg/cminorsel"
	"github.com/raymyers/ralph-cc/pkg/rtl"
)

func TestCFGBuilderAllocNode(t *testing.T) {
	b := NewCFGBuilder()

	n1 := b.AllocNode()
	n2 := b.AllocNode()
	n3 := b.AllocNode()

	if n1 != 1 {
		t.Errorf("first node = %d, want 1", n1)
	}
	if n2 != 2 {
		t.Errorf("second node = %d, want 2", n2)
	}
	if n3 != 3 {
		t.Errorf("third node = %d, want 3", n3)
	}
}

func TestCFGBuilderAllocReg(t *testing.T) {
	b := NewCFGBuilder()

	r1 := b.AllocReg()
	r2 := b.AllocReg()
	r3 := b.AllocReg()

	if r1 != 1 {
		t.Errorf("first reg = %d, want 1", r1)
	}
	if r2 != 2 {
		t.Errorf("second reg = %d, want 2", r2)
	}
	if r3 != 3 {
		t.Errorf("third reg = %d, want 3", r3)
	}
}

func TestCFGBuilderAllocRegs(t *testing.T) {
	b := NewCFGBuilder()

	regs := b.AllocRegs(3)

	if len(regs) != 3 {
		t.Fatalf("got %d regs, want 3", len(regs))
	}
	if regs[0] != 1 || regs[1] != 2 || regs[2] != 3 {
		t.Errorf("regs = %v, want [1 2 3]", regs)
	}
}

func TestCFGBuilderMapVar(t *testing.T) {
	b := NewCFGBuilder()

	r1 := b.MapVar("x")
	r2 := b.MapVar("y")
	r3 := b.MapVar("x") // should return same as r1

	if r1 != r3 {
		t.Errorf("x mapped to %d and %d, should be same", r1, r3)
	}
	if r1 == r2 {
		t.Errorf("x and y mapped to same register %d", r1)
	}
}

func TestCFGBuilderGetVarReg(t *testing.T) {
	b := NewCFGBuilder()

	_, ok := b.GetVarReg("x")
	if ok {
		t.Error("GetVarReg should return false for unmapped var")
	}

	expected := b.MapVar("x")
	got, ok := b.GetVarReg("x")
	if !ok {
		t.Error("GetVarReg should return true for mapped var")
	}
	if got != expected {
		t.Errorf("GetVarReg = %d, want %d", got, expected)
	}
}

func TestCFGBuilderEmitInstr(t *testing.T) {
	b := NewCFGBuilder()

	n1 := b.EmitInstr(rtl.Inop{Succ: 2})
	n2 := b.EmitInstr(rtl.Ireturn{Arg: nil})

	if n1 != 1 {
		t.Errorf("first node = %d, want 1", n1)
	}
	if n2 != 2 {
		t.Errorf("second node = %d, want 2", n2)
	}

	code := b.GetCode()
	if _, ok := code[n1].(rtl.Inop); !ok {
		t.Errorf("node %d should be Inop", n1)
	}
	if _, ok := code[n2].(rtl.Ireturn); !ok {
		t.Errorf("node %d should be Ireturn", n2)
	}
}

func TestCFGBuilderLabels(t *testing.T) {
	b := NewCFGBuilder()

	// First access creates the label
	n1 := b.GetOrCreateLabel("L1")

	// Second access returns same node
	n2 := b.GetOrCreateLabel("L1")

	if n1 != n2 {
		t.Errorf("same label returned different nodes: %d vs %d", n1, n2)
	}

	// GetLabel also works
	n3, ok := b.GetLabel("L1")
	if !ok {
		t.Error("GetLabel should return true for existing label")
	}
	if n3 != n1 {
		t.Errorf("GetLabel returned %d, want %d", n3, n1)
	}

	// Unknown label
	_, ok = b.GetLabel("unknown")
	if ok {
		t.Error("GetLabel should return false for unknown label")
	}
}

func TestCFGBuilderStackVars(t *testing.T) {
	b := NewCFGBuilder()

	_, ok := b.GetStackVar("x")
	if ok {
		t.Error("GetStackVar should return false for unknown var")
	}

	b.SetStackVar("x", 8)
	b.SetStackVar("y", 16)

	offset, ok := b.GetStackVar("x")
	if !ok {
		t.Error("GetStackVar should return true for set var")
	}
	if offset != 8 {
		t.Errorf("x offset = %d, want 8", offset)
	}

	offset, ok = b.GetStackVar("y")
	if !ok {
		t.Error("GetStackVar should return true for set var")
	}
	if offset != 16 {
		t.Errorf("y offset = %d, want 16", offset)
	}
}

func TestCFGBuilderStackSize(t *testing.T) {
	b := NewCFGBuilder()

	if b.GetStackSize() != 0 {
		t.Errorf("initial stack size = %d, want 0", b.GetStackSize())
	}

	b.SetStackSize(32)
	if b.GetStackSize() != 32 {
		t.Errorf("stack size = %d, want 32", b.GetStackSize())
	}
}

func TestExitContext(t *testing.T) {
	e := NewExitContext()

	if e.Depth() != 0 {
		t.Errorf("initial depth = %d, want 0", e.Depth())
	}

	_, ok := e.Get(0)
	if ok {
		t.Error("Get(0) should return false on empty context")
	}

	// Push targets
	e.Push(rtl.Node(10))
	e.Push(rtl.Node(20))
	e.Push(rtl.Node(30))

	if e.Depth() != 3 {
		t.Errorf("depth = %d, want 3", e.Depth())
	}

	// Sexit(0) = innermost = 30
	target, ok := e.Get(0)
	if !ok || target != 30 {
		t.Errorf("Get(0) = %d, %v, want 30, true", target, ok)
	}

	// Sexit(1) = next outer = 20
	target, ok = e.Get(1)
	if !ok || target != 20 {
		t.Errorf("Get(1) = %d, %v, want 20, true", target, ok)
	}

	// Sexit(2) = outermost = 10
	target, ok = e.Get(2)
	if !ok || target != 10 {
		t.Errorf("Get(2) = %d, %v, want 10, true", target, ok)
	}

	// Out of range
	_, ok = e.Get(3)
	if ok {
		t.Error("Get(3) should return false")
	}

	// Pop and check
	e.Pop()
	if e.Depth() != 2 {
		t.Errorf("after pop depth = %d, want 2", e.Depth())
	}

	target, ok = e.Get(0)
	if !ok || target != 20 {
		t.Errorf("after pop Get(0) = %d, %v, want 20, true", target, ok)
	}
}

func TestTranslateCondition(t *testing.T) {
	tests := []struct {
		name     string
		cond     cminorsel.Condition
		wantCode rtl.ConditionCode
		wantArgs int
	}{
		{
			name:     "CondTrue",
			cond:     cminorsel.CondTrue{},
			wantCode: rtl.Ccompimm{Cond: rtl.Cne, N: 0},
			wantArgs: 1,
		},
		{
			name:     "CondFalse",
			cond:     cminorsel.CondFalse{},
			wantCode: rtl.Ccompimm{Cond: rtl.Cne, N: 0},
			wantArgs: 1,
		},
		{
			name: "CondCmp_eq",
			cond: cminorsel.CondCmp{
				Cmp:   cminorsel.Ceq,
				Left:  cminorsel.Evar{Name: "x"},
				Right: cminorsel.Evar{Name: "y"},
			},
			wantCode: rtl.Ccomp{Cond: rtl.Ceq},
			wantArgs: 2,
		},
		{
			name: "CondCmpu_ne",
			cond: cminorsel.CondCmpu{
				Cmp:   cminorsel.Cne,
				Left:  cminorsel.Evar{Name: "x"},
				Right: cminorsel.Evar{Name: "y"},
			},
			wantCode: rtl.Ccompu{Cond: rtl.Cne},
			wantArgs: 2,
		},
		{
			name: "CondCmpl_lt",
			cond: cminorsel.CondCmpl{
				Cmp:   cminorsel.Clt,
				Left:  cminorsel.Evar{Name: "x"},
				Right: cminorsel.Evar{Name: "y"},
			},
			wantCode: rtl.Ccompl{Cond: rtl.Clt},
			wantArgs: 2,
		},
		{
			name: "CondCmplu_ge",
			cond: cminorsel.CondCmplu{
				Cmp:   cminorsel.Cge,
				Left:  cminorsel.Evar{Name: "x"},
				Right: cminorsel.Evar{Name: "y"},
			},
			wantCode: rtl.Ccomplu{Cond: rtl.Cge},
			wantArgs: 2,
		},
		{
			name: "CondCmpf_le",
			cond: cminorsel.CondCmpf{
				Cmp:   cminorsel.Cle,
				Left:  cminorsel.Evar{Name: "x"},
				Right: cminorsel.Evar{Name: "y"},
			},
			wantCode: rtl.Ccompf{Cond: rtl.Cle},
			wantArgs: 2,
		},
		{
			name: "CondCmps_gt",
			cond: cminorsel.CondCmps{
				Cmp:   cminorsel.Cgt,
				Left:  cminorsel.Evar{Name: "x"},
				Right: cminorsel.Evar{Name: "y"},
			},
			wantCode: rtl.Ccomps{Cond: rtl.Cgt},
			wantArgs: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCode, gotArgs := TranslateCondition(tt.cond)
			if gotCode != tt.wantCode {
				t.Errorf("code = %T{%v}, want %T{%v}", gotCode, gotCode, tt.wantCode, tt.wantCode)
			}
			if len(gotArgs) != tt.wantArgs {
				t.Errorf("len(args) = %d, want %d", len(gotArgs), tt.wantArgs)
			}
		})
	}
}

func TestTranslateConditionNot(t *testing.T) {
	// CondNot negates the inner condition
	cond := cminorsel.CondNot{
		Cond: cminorsel.CondCmp{
			Cmp:   cminorsel.Ceq,
			Left:  cminorsel.Evar{Name: "x"},
			Right: cminorsel.Evar{Name: "y"},
		},
	}

	code, args := TranslateCondition(cond)

	// Negated Ceq should be Cne
	expected := rtl.Ccomp{Cond: rtl.Cne}
	if code != expected {
		t.Errorf("code = %T{%v}, want %T{%v}", code, code, expected, expected)
	}
	if len(args) != 2 {
		t.Errorf("len(args) = %d, want 2", len(args))
	}
}

func TestNegateConditionCode(t *testing.T) {
	tests := []struct {
		name string
		in   rtl.ConditionCode
		want rtl.ConditionCode
	}{
		{"Ccomp_eq", rtl.Ccomp{Cond: rtl.Ceq}, rtl.Ccomp{Cond: rtl.Cne}},
		{"Ccomp_ne", rtl.Ccomp{Cond: rtl.Cne}, rtl.Ccomp{Cond: rtl.Ceq}},
		{"Ccomp_lt", rtl.Ccomp{Cond: rtl.Clt}, rtl.Ccomp{Cond: rtl.Cge}},
		{"Ccomp_le", rtl.Ccomp{Cond: rtl.Cle}, rtl.Ccomp{Cond: rtl.Cgt}},
		{"Ccomp_gt", rtl.Ccomp{Cond: rtl.Cgt}, rtl.Ccomp{Cond: rtl.Cle}},
		{"Ccomp_ge", rtl.Ccomp{Cond: rtl.Cge}, rtl.Ccomp{Cond: rtl.Clt}},
		{"Ccompu", rtl.Ccompu{Cond: rtl.Ceq}, rtl.Ccompu{Cond: rtl.Cne}},
		{"Ccompimm", rtl.Ccompimm{Cond: rtl.Ceq, N: 5}, rtl.Ccompimm{Cond: rtl.Cne, N: 5}},
		{"Ccompf", rtl.Ccompf{Cond: rtl.Ceq}, rtl.Cnotcompf{Cond: rtl.Ceq}},
		{"Cnotcompf", rtl.Cnotcompf{Cond: rtl.Ceq}, rtl.Ccompf{Cond: rtl.Ceq}},
		{"Ccomps", rtl.Ccomps{Cond: rtl.Clt}, rtl.Cnotcomps{Cond: rtl.Clt}},
		{"Cnotcomps", rtl.Cnotcomps{Cond: rtl.Clt}, rtl.Ccomps{Cond: rtl.Clt}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := negateConditionCode(tt.in)
			if got != tt.want {
				t.Errorf("negateConditionCode(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

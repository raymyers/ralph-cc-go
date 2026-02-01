package rtlgen

import (
	"testing"

	"github.com/raymyers/ralph-cc/pkg/cminorsel"
	"github.com/raymyers/ralph-cc/pkg/rtl"
)

func TestTranslateUnaryOp(t *testing.T) {
	tests := []struct {
		name string
		op   cminorsel.UnaryOp
		want rtl.Operation
	}{
		{"cast8signed", cminorsel.Ocast8signed, rtl.Ocast8signed{}},
		{"cast8unsigned", cminorsel.Ocast8unsigned, rtl.Ocast8unsigned{}},
		{"cast16signed", cminorsel.Ocast16signed, rtl.Ocast16signed{}},
		{"cast16unsigned", cminorsel.Ocast16unsigned, rtl.Ocast16unsigned{}},
		{"negint", cminorsel.Onegint, rtl.Oneg{}},
		{"negl", cminorsel.Onegl, rtl.Onegl{}},
		{"negf", cminorsel.Onegf, rtl.Onegf{}},
		{"negs", cminorsel.Onegs, rtl.Onegs{}},
		{"notint", cminorsel.Onotint, rtl.Onot{}},
		{"notl", cminorsel.Onotl, rtl.Onotl{}},
		{"singleoffloat", cminorsel.Osingleoffloat, rtl.Osingleoffloat{}},
		{"floatofsingle", cminorsel.Ofloatofsingle, rtl.Ofloatofsingle{}},
		{"intoffloat", cminorsel.Ointoffloat, rtl.Ointoffloat{}},
		{"intuoffloat", cminorsel.Ointuoffloat, rtl.Ointuoffloat{}},
		{"floatofint", cminorsel.Ofloatofint, rtl.Ofloatofint{}},
		{"floatofintu", cminorsel.Ofloatofintu, rtl.Ofloatofintu{}},
		{"longoffloat", cminorsel.Olongoffloat, rtl.Olongoffloat{}},
		{"longuoffloat", cminorsel.Olonguoffloat, rtl.Olonguoffloat{}},
		{"floatoflong", cminorsel.Ofloatoflong, rtl.Ofloatoflong{}},
		{"floatoflongu", cminorsel.Ofloatoflongu, rtl.Ofloatoflongu{}},
		{"intoflong", cminorsel.Ointoflong, rtl.Ointoflong{}},
		{"longofint", cminorsel.Olongofint, rtl.Olongofint{}},
		{"longofintu", cminorsel.Olongofintu, rtl.Olongofintu{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TranslateUnaryOp(tt.op)
			if got != tt.want {
				t.Errorf("TranslateUnaryOp(%v) = %T, want %T", tt.op, got, tt.want)
			}
		})
	}
}

func TestTranslateBinaryOp(t *testing.T) {
	tests := []struct {
		name string
		op   cminorsel.BinaryOp
		want rtl.Operation
	}{
		{"add", cminorsel.Oadd, rtl.Oadd{}},
		{"sub", cminorsel.Osub, rtl.Osub{}},
		{"mul", cminorsel.Omul, rtl.Omul{}},
		{"div", cminorsel.Odiv, rtl.Odiv{}},
		{"divu", cminorsel.Odivu, rtl.Odivu{}},
		{"mod", cminorsel.Omod, rtl.Omod{}},
		{"modu", cminorsel.Omodu, rtl.Omodu{}},
		{"and", cminorsel.Oand, rtl.Oand{}},
		{"or", cminorsel.Oor, rtl.Oor{}},
		{"xor", cminorsel.Oxor, rtl.Oxor{}},
		{"shl", cminorsel.Oshl, rtl.Oshl{}},
		{"shr", cminorsel.Oshr, rtl.Oshr{}},
		{"shru", cminorsel.Oshru, rtl.Oshru{}},
		{"addf", cminorsel.Oaddf, rtl.Oaddf{}},
		{"subf", cminorsel.Osubf, rtl.Osubf{}},
		{"mulf", cminorsel.Omulf, rtl.Omulf{}},
		{"divf", cminorsel.Odivf, rtl.Odivf{}},
		{"adds", cminorsel.Oadds, rtl.Oadds{}},
		{"subs", cminorsel.Osubs, rtl.Osubs{}},
		{"muls", cminorsel.Omuls, rtl.Omuls{}},
		{"divs", cminorsel.Odivs, rtl.Odivs{}},
		{"addl", cminorsel.Oaddl, rtl.Oaddl{}},
		{"subl", cminorsel.Osubl, rtl.Osubl{}},
		{"mull", cminorsel.Omull, rtl.Omull{}},
		{"divl", cminorsel.Odivl, rtl.Odivl{}},
		{"divlu", cminorsel.Odivlu, rtl.Odivlu{}},
		{"modl", cminorsel.Omodl, rtl.Omodl{}},
		{"modlu", cminorsel.Omodlu, rtl.Omodlu{}},
		{"andl", cminorsel.Oandl, rtl.Oandl{}},
		{"orl", cminorsel.Oorl, rtl.Oorl{}},
		{"xorl", cminorsel.Oxorl, rtl.Oxorl{}},
		{"shll", cminorsel.Oshll, rtl.Oshll{}},
		{"shrl", cminorsel.Oshrl, rtl.Oshrl{}},
		{"shrlu", cminorsel.Oshrlu, rtl.Oshrlu{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TranslateBinaryOp(tt.op)
			if got != tt.want {
				t.Errorf("TranslateBinaryOp(%v) = %T, want %T", tt.op, got, tt.want)
			}
		})
	}
}

func TestTranslateConstant(t *testing.T) {
	tests := []struct {
		name string
		c    cminorsel.Constant
		want rtl.Operation
	}{
		{"int", cminorsel.Ointconst{Value: 42}, rtl.Ointconst{Value: 42}},
		{"long", cminorsel.Olongconst{Value: 1234567890}, rtl.Olongconst{Value: 1234567890}},
		{"float", cminorsel.Ofloatconst{Value: 3.14}, rtl.Ofloatconst{Value: 3.14}},
		{"single", cminorsel.Osingleconst{Value: 2.5}, rtl.Osingleconst{Value: 2.5}},
		{"addrsymbol", cminorsel.Oaddrsymbol{Symbol: "foo", Offset: 8}, rtl.Oaddrsymbol{Symbol: "foo", Offset: 8}},
		{"addrstack", cminorsel.Oaddrstack{Offset: 16}, rtl.Oaddrstack{Offset: 16}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TranslateConstant(tt.c)
			if got != tt.want {
				t.Errorf("TranslateConstant(%v) = %v, want %v", tt.c, got, tt.want)
			}
		})
	}
}

func TestTranslateAddressingMode(t *testing.T) {
	tests := []struct {
		name string
		addr cminorsel.AddressingMode
		want rtl.AddressingMode
	}{
		{"indexed", cminorsel.Aindexed{Offset: 8}, rtl.Aindexed{Offset: 8}},
		{"indexed2", cminorsel.Aindexed2{}, rtl.Aindexed2{}},
		{"indexed2shift", cminorsel.Aindexed2shift{Shift: 2}, rtl.Aindexed2shift{Shift: 2}},
		{"global", cminorsel.Aglobal{Symbol: "g", Offset: 4}, rtl.Aglobal{Symbol: "g", Offset: 4}},
		{"instack", cminorsel.Ainstack{Offset: 32}, rtl.Ainstack{Offset: 32}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TranslateAddressingMode(tt.addr)
			if got != tt.want {
				t.Errorf("TranslateAddressingMode(%v) = %v, want %v", tt.addr, got, tt.want)
			}
		})
	}
}

func TestInstrBuilderEmitOp(t *testing.T) {
	cfg := NewCFGBuilder()
	regs := NewRegAllocator()
	b := NewInstrBuilder(cfg, regs)

	succ := b.AllocNode()
	dest := b.Fresh()
	arg1 := b.Fresh()
	arg2 := b.Fresh()

	n := b.EmitOp(rtl.Oadd{}, []rtl.Reg{arg1, arg2}, dest, succ)

	instr := cfg.GetCode()[n]
	iop, ok := instr.(rtl.Iop)
	if !ok {
		t.Fatalf("instruction is %T, want Iop", instr)
	}
	if _, ok := iop.Op.(rtl.Oadd); !ok {
		t.Errorf("op is %T, want Oadd", iop.Op)
	}
	if len(iop.Args) != 2 {
		t.Errorf("len(args) = %d, want 2", len(iop.Args))
	}
	if iop.Dest != dest {
		t.Errorf("dest = %d, want %d", iop.Dest, dest)
	}
	if iop.Succ != succ {
		t.Errorf("succ = %d, want %d", iop.Succ, succ)
	}
}

func TestInstrBuilderEmitMove(t *testing.T) {
	cfg := NewCFGBuilder()
	regs := NewRegAllocator()
	b := NewInstrBuilder(cfg, regs)

	succ := b.AllocNode()
	src := b.Fresh()
	dest := b.Fresh()

	n := b.EmitMove(src, dest, succ)

	instr := cfg.GetCode()[n]
	iop, ok := instr.(rtl.Iop)
	if !ok {
		t.Fatalf("instruction is %T, want Iop", instr)
	}
	if _, ok := iop.Op.(rtl.Omove); !ok {
		t.Errorf("op is %T, want Omove", iop.Op)
	}
	if len(iop.Args) != 1 || iop.Args[0] != src {
		t.Errorf("args = %v, want [%d]", iop.Args, src)
	}
}

func TestInstrBuilderEmitConst(t *testing.T) {
	cfg := NewCFGBuilder()
	regs := NewRegAllocator()
	b := NewInstrBuilder(cfg, regs)

	succ := b.AllocNode()
	dest := b.Fresh()

	n := b.EmitConst(cminorsel.Ointconst{Value: 42}, dest, succ)

	instr := cfg.GetCode()[n]
	iop, ok := instr.(rtl.Iop)
	if !ok {
		t.Fatalf("instruction is %T, want Iop", instr)
	}
	iconst, ok := iop.Op.(rtl.Ointconst)
	if !ok {
		t.Errorf("op is %T, want Ointconst", iop.Op)
	}
	if iconst.Value != 42 {
		t.Errorf("value = %d, want 42", iconst.Value)
	}
}

func TestInstrBuilderEmitLoad(t *testing.T) {
	cfg := NewCFGBuilder()
	regs := NewRegAllocator()
	b := NewInstrBuilder(cfg, regs)

	succ := b.AllocNode()
	dest := b.Fresh()
	base := b.Fresh()

	n := b.EmitLoad(rtl.Mint32, rtl.Aindexed{Offset: 8}, []rtl.Reg{base}, dest, succ)

	instr := cfg.GetCode()[n]
	iload, ok := instr.(rtl.Iload)
	if !ok {
		t.Fatalf("instruction is %T, want Iload", instr)
	}
	if iload.Chunk != rtl.Mint32 {
		t.Errorf("chunk = %v, want Mint32", iload.Chunk)
	}
	aindexed, ok := iload.Addr.(rtl.Aindexed)
	if !ok {
		t.Errorf("addr is %T, want Aindexed", iload.Addr)
	}
	if aindexed.Offset != 8 {
		t.Errorf("offset = %d, want 8", aindexed.Offset)
	}
}

func TestInstrBuilderEmitStore(t *testing.T) {
	cfg := NewCFGBuilder()
	regs := NewRegAllocator()
	b := NewInstrBuilder(cfg, regs)

	succ := b.AllocNode()
	src := b.Fresh()
	base := b.Fresh()

	n := b.EmitStore(rtl.Mint32, rtl.Aindexed{Offset: 0}, []rtl.Reg{base}, src, succ)

	instr := cfg.GetCode()[n]
	istore, ok := instr.(rtl.Istore)
	if !ok {
		t.Fatalf("instruction is %T, want Istore", instr)
	}
	if istore.Src != src {
		t.Errorf("src = %d, want %d", istore.Src, src)
	}
}

func TestInstrBuilderEmitCall(t *testing.T) {
	cfg := NewCFGBuilder()
	regs := NewRegAllocator()
	b := NewInstrBuilder(cfg, regs)

	succ := b.AllocNode()
	dest := b.Fresh()
	arg := b.Fresh()

	sig := rtl.Sig{Args: []string{"int"}, Return: "int"}
	fn := rtl.FunSymbol{Name: "foo"}

	n := b.EmitCall(sig, fn, []rtl.Reg{arg}, dest, succ)

	instr := cfg.GetCode()[n]
	icall, ok := instr.(rtl.Icall)
	if !ok {
		t.Fatalf("instruction is %T, want Icall", instr)
	}
	fsym, ok := icall.Fn.(rtl.FunSymbol)
	if !ok {
		t.Errorf("fn is %T, want FunSymbol", icall.Fn)
	}
	if fsym.Name != "foo" {
		t.Errorf("name = %q, want %q", fsym.Name, "foo")
	}
}

func TestInstrBuilderEmitTailcall(t *testing.T) {
	cfg := NewCFGBuilder()
	regs := NewRegAllocator()
	b := NewInstrBuilder(cfg, regs)

	arg := b.Fresh()
	sig := rtl.Sig{Args: []string{"int"}, Return: "int"}
	fn := rtl.FunSymbol{Name: "bar"}

	n := b.EmitTailcall(sig, fn, []rtl.Reg{arg})

	instr := cfg.GetCode()[n]
	itail, ok := instr.(rtl.Itailcall)
	if !ok {
		t.Fatalf("instruction is %T, want Itailcall", instr)
	}
	if len(itail.Args) != 1 {
		t.Errorf("len(args) = %d, want 1", len(itail.Args))
	}
}

func TestInstrBuilderEmitCond(t *testing.T) {
	cfg := NewCFGBuilder()
	regs := NewRegAllocator()
	b := NewInstrBuilder(cfg, regs)

	ifso := b.AllocNode()
	ifnot := b.AllocNode()
	arg1 := b.Fresh()
	arg2 := b.Fresh()

	n := b.EmitCond(rtl.Ccomp{Cond: rtl.Clt}, []rtl.Reg{arg1, arg2}, ifso, ifnot)

	instr := cfg.GetCode()[n]
	icond, ok := instr.(rtl.Icond)
	if !ok {
		t.Fatalf("instruction is %T, want Icond", instr)
	}
	if icond.IfSo != ifso {
		t.Errorf("ifso = %d, want %d", icond.IfSo, ifso)
	}
	if icond.IfNot != ifnot {
		t.Errorf("ifnot = %d, want %d", icond.IfNot, ifnot)
	}
}

func TestInstrBuilderEmitJumptable(t *testing.T) {
	cfg := NewCFGBuilder()
	regs := NewRegAllocator()
	b := NewInstrBuilder(cfg, regs)

	t0 := b.AllocNode()
	t1 := b.AllocNode()
	t2 := b.AllocNode()
	arg := b.Fresh()

	n := b.EmitJumptable(arg, []rtl.Node{t0, t1, t2})

	instr := cfg.GetCode()[n]
	ijmp, ok := instr.(rtl.Ijumptable)
	if !ok {
		t.Fatalf("instruction is %T, want Ijumptable", instr)
	}
	if len(ijmp.Targets) != 3 {
		t.Errorf("len(targets) = %d, want 3", len(ijmp.Targets))
	}
}

func TestInstrBuilderEmitReturn(t *testing.T) {
	cfg := NewCFGBuilder()
	regs := NewRegAllocator()
	b := NewInstrBuilder(cfg, regs)

	// Return with value
	ret := b.Fresh()
	n := b.EmitReturn(&ret)

	instr := cfg.GetCode()[n]
	iret, ok := instr.(rtl.Ireturn)
	if !ok {
		t.Fatalf("instruction is %T, want Ireturn", instr)
	}
	if iret.Arg == nil || *iret.Arg != ret {
		t.Errorf("arg = %v, want %d", iret.Arg, ret)
	}

	// Return void
	n2 := b.EmitReturn(nil)
	instr2 := cfg.GetCode()[n2]
	iret2, ok := instr2.(rtl.Ireturn)
	if !ok {
		t.Fatalf("instruction is %T, want Ireturn", instr2)
	}
	if iret2.Arg != nil {
		t.Errorf("arg = %v, want nil", iret2.Arg)
	}
}

func TestInstrBuilderEmitNop(t *testing.T) {
	cfg := NewCFGBuilder()
	regs := NewRegAllocator()
	b := NewInstrBuilder(cfg, regs)

	succ := b.AllocNode()
	n := b.EmitNop(succ)

	instr := cfg.GetCode()[n]
	inop, ok := instr.(rtl.Inop)
	if !ok {
		t.Fatalf("instruction is %T, want Inop", instr)
	}
	if inop.Succ != succ {
		t.Errorf("succ = %d, want %d", inop.Succ, succ)
	}
}

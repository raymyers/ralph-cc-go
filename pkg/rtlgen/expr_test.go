package rtlgen

import (
	"testing"

	"github.com/raymyers/ralph-cc/pkg/cminorsel"
	"github.com/raymyers/ralph-cc/pkg/rtl"
)

func TestTranslateExpr_Const(t *testing.T) {
	cfg := NewCFGBuilder()
	regs := NewRegAllocator()
	tr := NewExprTranslator(cfg, regs)
	
	dest := regs.Fresh()
	succ := cfg.AllocNode()
	
	entry := tr.TranslateExpr(cminorsel.Econst{
		Const: cminorsel.Ointconst{Value: 42},
	}, dest, succ)
	
	// Should emit: dest = 42 goto succ
	instr := cfg.GetCode()[entry]
	iop, ok := instr.(rtl.Iop)
	if !ok {
		t.Fatalf("expected Iop, got %T", instr)
	}
	
	iconst, ok := iop.Op.(rtl.Ointconst)
	if !ok {
		t.Fatalf("expected Ointconst, got %T", iop.Op)
	}
	if iconst.Value != 42 {
		t.Errorf("const value = %d, want 42", iconst.Value)
	}
	if iop.Dest != dest {
		t.Errorf("dest = %d, want %d", iop.Dest, dest)
	}
	if iop.Succ != succ {
		t.Errorf("succ = %d, want %d", iop.Succ, succ)
	}
}

func TestTranslateExpr_Var(t *testing.T) {
	cfg := NewCFGBuilder()
	regs := NewRegAllocator()
	tr := NewExprTranslator(cfg, regs)
	
	// Map variable x to register
	xReg := regs.MapVar("x")
	
	dest := regs.Fresh()
	succ := cfg.AllocNode()
	
	entry := tr.TranslateExpr(cminorsel.Evar{Name: "x"}, dest, succ)
	
	// Should emit: dest = x goto succ (move)
	instr := cfg.GetCode()[entry]
	iop, ok := instr.(rtl.Iop)
	if !ok {
		t.Fatalf("expected Iop, got %T", instr)
	}
	
	if _, ok := iop.Op.(rtl.Omove); !ok {
		t.Fatalf("expected Omove, got %T", iop.Op)
	}
	if len(iop.Args) != 1 || iop.Args[0] != xReg {
		t.Errorf("args = %v, want [%d]", iop.Args, xReg)
	}
}

func TestTranslateExpr_VarSameReg(t *testing.T) {
	cfg := NewCFGBuilder()
	regs := NewRegAllocator()
	tr := NewExprTranslator(cfg, regs)
	
	// Map variable x to dest register
	xReg := regs.MapVar("x")
	succ := cfg.AllocNode()
	
	// Translate x into its own register
	entry := tr.TranslateExpr(cminorsel.Evar{Name: "x"}, xReg, succ)
	
	// Should just return succ (no move needed)
	if entry != succ {
		t.Errorf("entry = %d, want succ=%d (no move needed)", entry, succ)
	}
}

func TestTranslateExpr_Unop(t *testing.T) {
	cfg := NewCFGBuilder()
	regs := NewRegAllocator()
	tr := NewExprTranslator(cfg, regs)
	
	dest := regs.Fresh()
	succ := cfg.AllocNode()
	
	// -x where x = 5
	entry := tr.TranslateExpr(cminorsel.Eunop{
		Op:  cminorsel.Onegint,
		Arg: cminorsel.Econst{Const: cminorsel.Ointconst{Value: 5}},
	}, dest, succ)
	
	// Check we got instructions
	code := cfg.GetCode()
	if len(code) < 2 {
		t.Fatalf("expected at least 2 instructions, got %d", len(code))
	}
	
	// Entry should be the const load
	instr := code[entry]
	iop, ok := instr.(rtl.Iop)
	if !ok {
		t.Fatalf("entry instr expected Iop, got %T", instr)
	}
	if _, ok := iop.Op.(rtl.Ointconst); !ok {
		t.Errorf("entry op expected Ointconst, got %T", iop.Op)
	}
	
	// Following instruction should be the negation
	negInstr := code[iop.Succ]
	negOp, ok := negInstr.(rtl.Iop)
	if !ok {
		t.Fatalf("neg instr expected Iop, got %T", negInstr)
	}
	if _, ok := negOp.Op.(rtl.Oneg); !ok {
		t.Errorf("neg op expected Oneg, got %T", negOp.Op)
	}
}

func TestTranslateExpr_Binop(t *testing.T) {
	cfg := NewCFGBuilder()
	regs := NewRegAllocator()
	tr := NewExprTranslator(cfg, regs)
	
	dest := regs.Fresh()
	succ := cfg.AllocNode()
	
	// 3 + 4
	entry := tr.TranslateExpr(cminorsel.Ebinop{
		Op:    cminorsel.Oadd,
		Left:  cminorsel.Econst{Const: cminorsel.Ointconst{Value: 3}},
		Right: cminorsel.Econst{Const: cminorsel.Ointconst{Value: 4}},
	}, dest, succ)
	
	// Check instruction chain: left_const -> right_const -> add -> succ
	code := cfg.GetCode()
	if len(code) < 3 {
		t.Fatalf("expected at least 3 instructions, got %d", len(code))
	}
	
	// Entry should load left const
	instr := code[entry]
	iop, ok := instr.(rtl.Iop)
	if !ok {
		t.Fatalf("entry instr expected Iop, got %T", instr)
	}
	
	leftConst, ok := iop.Op.(rtl.Ointconst)
	if !ok {
		t.Fatalf("expected Ointconst, got %T", iop.Op)
	}
	if leftConst.Value != 3 {
		t.Errorf("left const = %d, want 3", leftConst.Value)
	}
}

func TestTranslateExprList(t *testing.T) {
	cfg := NewCFGBuilder()
	regs := NewRegAllocator()
	tr := NewExprTranslator(cfg, regs)
	
	succ := cfg.AllocNode()
	
	exprs := []cminorsel.Expr{
		cminorsel.Econst{Const: cminorsel.Ointconst{Value: 1}},
		cminorsel.Econst{Const: cminorsel.Ointconst{Value: 2}},
		cminorsel.Econst{Const: cminorsel.Ointconst{Value: 3}},
	}
	
	resultRegs, entry := tr.TranslateExprList(exprs, succ)
	
	if len(resultRegs) != 3 {
		t.Errorf("got %d result regs, want 3", len(resultRegs))
	}
	
	// Walk the instruction chain to verify
	code := cfg.GetCode()
	if len(code) < 3 {
		t.Fatalf("expected at least 3 instructions, got %d", len(code))
	}
	
	// Entry should be first expression
	_ = entry
}

func TestTranslateExprList_Empty(t *testing.T) {
	cfg := NewCFGBuilder()
	regs := NewRegAllocator()
	tr := NewExprTranslator(cfg, regs)
	
	succ := cfg.AllocNode()
	
	resultRegs, entry := tr.TranslateExprList(nil, succ)
	
	if len(resultRegs) != 0 {
		t.Errorf("got %d result regs, want 0", len(resultRegs))
	}
	if entry != succ {
		t.Errorf("entry = %d, want succ=%d", entry, succ)
	}
}

func TestTranslateExpr_Load(t *testing.T) {
	cfg := NewCFGBuilder()
	regs := NewRegAllocator()
	tr := NewExprTranslator(cfg, regs)
	
	baseReg := regs.MapVar("ptr")
	dest := regs.Fresh()
	succ := cfg.AllocNode()
	
	// Load int from *ptr
	entry := tr.TranslateExpr(cminorsel.Eload{
		Chunk: cminorsel.Mint32,
		Mode:  cminorsel.Aindexed{Offset: 0},
		Args:  []cminorsel.Expr{cminorsel.Evar{Name: "ptr"}},
	}, dest, succ)
	
	// Verify load instruction exists
	code := cfg.GetCode()
	foundLoad := false
	for _, instr := range code {
		if _, ok := instr.(rtl.Iload); ok {
			foundLoad = true
			break
		}
	}
	
	if !foundLoad {
		t.Error("expected Iload instruction in generated code")
	}
	_ = entry
	_ = baseReg
}

func TestTranslateCond_Simple(t *testing.T) {
	cfg := NewCFGBuilder()
	regs := NewRegAllocator()
	tr := NewExprTranslator(cfg, regs)
	
	regs.MapVar("x")
	regs.MapVar("y")
	
	ifso := cfg.AllocNode()
	ifnot := cfg.AllocNode()
	
	// x < y
	cond := cminorsel.CondCmp{
		Cmp:   cminorsel.Clt,
		Left:  cminorsel.Evar{Name: "x"},
		Right: cminorsel.Evar{Name: "y"},
	}
	
	entry := tr.TranslateCond(cond, ifso, ifnot)
	
	// Verify we got a conditional branch
	code := cfg.GetCode()
	foundCond := false
	for _, instr := range code {
		if icond, ok := instr.(rtl.Icond); ok {
			foundCond = true
			if icond.IfSo != ifso {
				t.Errorf("ifso = %d, want %d", icond.IfSo, ifso)
			}
			if icond.IfNot != ifnot {
				t.Errorf("ifnot = %d, want %d", icond.IfNot, ifnot)
			}
		}
	}
	
	if !foundCond {
		t.Error("expected Icond instruction in generated code")
	}
	_ = entry
}

func TestTranslateExpr_Condition(t *testing.T) {
	cfg := NewCFGBuilder()
	regs := NewRegAllocator()
	tr := NewExprTranslator(cfg, regs)
	
	regs.MapVar("x")
	dest := regs.Fresh()
	succ := cfg.AllocNode()
	
	// x != 0 ? 1 : 0
	entry := tr.TranslateExpr(cminorsel.Econdition{
		Cond: cminorsel.CondCmp{
			Cmp:   cminorsel.Cne,
			Left:  cminorsel.Evar{Name: "x"},
			Right: cminorsel.Econst{Const: cminorsel.Ointconst{Value: 0}},
		},
		Then: cminorsel.Econst{Const: cminorsel.Ointconst{Value: 1}},
		Else: cminorsel.Econst{Const: cminorsel.Ointconst{Value: 0}},
	}, dest, succ)
	
	// Verify we generated code
	code := cfg.GetCode()
	if len(code) < 3 {
		t.Errorf("expected at least 3 instructions for conditional, got %d", len(code))
	}
	_ = entry
}

func TestTranslateExpr_Let(t *testing.T) {
	ResetLetBindings() // Clear any previous state
	
	cfg := NewCFGBuilder()
	regs := NewRegAllocator()
	tr := NewExprTranslator(cfg, regs)
	
	dest := regs.Fresh()
	succ := cfg.AllocNode()
	
	// let x = 5 in x + 1
	entry := tr.TranslateExpr(cminorsel.Elet{
		Bind: cminorsel.Econst{Const: cminorsel.Ointconst{Value: 5}},
		Body: cminorsel.Ebinop{
			Op:    cminorsel.Oadd,
			Left:  cminorsel.Eletvar{Index: 0}, // Reference to bound value
			Right: cminorsel.Econst{Const: cminorsel.Ointconst{Value: 1}},
		},
	}, dest, succ)
	
	// Should generate code
	code := cfg.GetCode()
	if len(code) < 3 {
		t.Errorf("expected at least 3 instructions, got %d", len(code))
	}
	_ = entry
}

func TestLetBindingStack(t *testing.T) {
	ResetLetBindings()
	
	cfg := NewCFGBuilder()
	regs := NewRegAllocator()
	tr := NewExprTranslator(cfg, regs)
	
	// Push bindings
	r1 := rtl.Reg(10)
	r2 := rtl.Reg(20)
	
	tr.pushLetBinding(r1)
	tr.pushLetBinding(r2)
	
	// Index 0 = innermost = r2
	if got := tr.getLetBinding(0); got != r2 {
		t.Errorf("getLetBinding(0) = %d, want %d", got, r2)
	}
	
	// Index 1 = outer = r1
	if got := tr.getLetBinding(1); got != r1 {
		t.Errorf("getLetBinding(1) = %d, want %d", got, r1)
	}
	
	// Pop and verify
	tr.popLetBinding()
	if got := tr.getLetBinding(0); got != r1 {
		t.Errorf("after pop, getLetBinding(0) = %d, want %d", got, r1)
	}
}

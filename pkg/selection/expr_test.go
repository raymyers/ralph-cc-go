package selection

import (
	"testing"

	"github.com/raymyers/ralph-cc/pkg/cminor"
	"github.com/raymyers/ralph-cc/pkg/cminorsel"
)

func TestSelectExpr_Var(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	expr := cminor.Evar{Name: "x"}
	result := ctx.SelectExpr(expr)

	v, ok := result.(cminorsel.Evar)
	if !ok {
		t.Fatalf("expected Evar, got %T", result)
	}
	if v.Name != "x" {
		t.Errorf("expected name 'x', got %q", v.Name)
	}
}

func TestSelectExpr_GlobalVar(t *testing.T) {
	globals := map[string]bool{"global_var": true}
	ctx := NewSelectionContext(globals, nil)
	expr := cminor.Evar{Name: "global_var"}
	result := ctx.SelectExpr(expr)

	c, ok := result.(cminorsel.Econst)
	if !ok {
		t.Fatalf("expected Econst, got %T", result)
	}
	sym, ok := c.Const.(cminorsel.Oaddrsymbol)
	if !ok {
		t.Fatalf("expected Oaddrsymbol, got %T", c.Const)
	}
	if sym.Symbol != "global_var" {
		t.Errorf("expected symbol 'global_var', got %q", sym.Symbol)
	}
}

func TestSelectExpr_StackVar(t *testing.T) {
	stackVars := map[string]int64{"stack_var": 16}
	ctx := NewSelectionContext(nil, stackVars)
	expr := cminor.Evar{Name: "stack_var"}
	result := ctx.SelectExpr(expr)

	c, ok := result.(cminorsel.Econst)
	if !ok {
		t.Fatalf("expected Econst, got %T", result)
	}
	stk, ok := c.Const.(cminorsel.Oaddrstack)
	if !ok {
		t.Fatalf("expected Oaddrstack, got %T", c.Const)
	}
	if stk.Offset != 16 {
		t.Errorf("expected offset 16, got %d", stk.Offset)
	}
}

func TestSelectExpr_Constants(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)

	tests := []struct {
		name     string
		input    cminor.Constant
		checkVal func(cminorsel.Constant) bool
	}{
		{
			name:  "int",
			input: cminor.Ointconst{Value: 42},
			checkVal: func(c cminorsel.Constant) bool {
				v, ok := c.(cminorsel.Ointconst)
				return ok && v.Value == 42
			},
		},
		{
			name:  "long",
			input: cminor.Olongconst{Value: 1234567890123},
			checkVal: func(c cminorsel.Constant) bool {
				v, ok := c.(cminorsel.Olongconst)
				return ok && v.Value == 1234567890123
			},
		},
		{
			name:  "float",
			input: cminor.Ofloatconst{Value: 3.14},
			checkVal: func(c cminorsel.Constant) bool {
				v, ok := c.(cminorsel.Ofloatconst)
				return ok && v.Value == 3.14
			},
		},
		{
			name:  "single",
			input: cminor.Osingleconst{Value: 2.5},
			checkVal: func(c cminorsel.Constant) bool {
				v, ok := c.(cminorsel.Osingleconst)
				return ok && v.Value == 2.5
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expr := cminor.Econst{Const: tc.input}
			result := ctx.SelectExpr(expr)

			c, ok := result.(cminorsel.Econst)
			if !ok {
				t.Fatalf("expected Econst, got %T", result)
			}
			if !tc.checkVal(c.Const) {
				t.Errorf("constant value check failed for %T", c.Const)
			}
		})
	}
}

func TestSelectExpr_Unop(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	expr := cminor.Eunop{
		Op:  cminor.Onegint,
		Arg: cminor.Evar{Name: "x"},
	}
	result := ctx.SelectExpr(expr)

	u, ok := result.(cminorsel.Eunop)
	if !ok {
		t.Fatalf("expected Eunop, got %T", result)
	}
	if u.Op != cminorsel.Onegint {
		t.Errorf("expected Onegint, got %v", u.Op)
	}
	if v, ok := u.Arg.(cminorsel.Evar); !ok || v.Name != "x" {
		t.Errorf("expected Evar{Name:'x'}, got %T", u.Arg)
	}
}

func TestSelectExpr_Binop(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	expr := cminor.Ebinop{
		Op:    cminor.Oadd,
		Left:  cminor.Evar{Name: "x"},
		Right: cminor.Econst{Const: cminor.Ointconst{Value: 1}},
	}
	result := ctx.SelectExpr(expr)

	b, ok := result.(cminorsel.Ebinop)
	if !ok {
		t.Fatalf("expected Ebinop, got %T", result)
	}
	if b.Op != cminorsel.Oadd {
		t.Errorf("expected Oadd, got %v", b.Op)
	}
}

func TestSelectExpr_CombinedAddShift(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	// x + (y << 2)
	expr := cminor.Ebinop{
		Op:   cminor.Oadd,
		Left: cminor.Evar{Name: "x"},
		Right: cminor.Ebinop{
			Op:    cminor.Oshl,
			Left:  cminor.Evar{Name: "y"},
			Right: cminor.Econst{Const: cminor.Ointconst{Value: 2}},
		},
	}
	result := ctx.SelectExpr(expr)

	add, ok := result.(cminorsel.Eaddshift)
	if !ok {
		t.Fatalf("expected Eaddshift, got %T", result)
	}
	if add.Shift != 2 {
		t.Errorf("expected shift 2, got %d", add.Shift)
	}
	if add.Op != cminorsel.Slsl {
		t.Errorf("expected Slsl, got %v", add.Op)
	}
}

func TestSelectExpr_CombinedSubShift(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	// x - (y << 3)
	expr := cminor.Ebinop{
		Op:   cminor.Osub,
		Left: cminor.Evar{Name: "x"},
		Right: cminor.Ebinop{
			Op:    cminor.Oshl,
			Left:  cminor.Evar{Name: "y"},
			Right: cminor.Econst{Const: cminor.Ointconst{Value: 3}},
		},
	}
	result := ctx.SelectExpr(expr)

	sub, ok := result.(cminorsel.Esubshift)
	if !ok {
		t.Fatalf("expected Esubshift, got %T", result)
	}
	if sub.Shift != 3 {
		t.Errorf("expected shift 3, got %d", sub.Shift)
	}
}

func TestSelectExpr_Cmp(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	expr := cminor.Ecmp{
		Op:    cminor.Ocmp,
		Cmp:   cminor.Ceq,
		Left:  cminor.Evar{Name: "x"},
		Right: cminor.Econst{Const: cminor.Ointconst{Value: 0}},
	}
	result := ctx.SelectExpr(expr)

	cmp, ok := result.(cminorsel.Ecmp)
	if !ok {
		t.Fatalf("expected Ecmp, got %T", result)
	}
	if cmp.Op != cminorsel.Ocmp {
		t.Errorf("expected Ocmp, got %v", cmp.Op)
	}
	if cmp.Cmp != cminorsel.Ceq {
		t.Errorf("expected Ceq, got %v", cmp.Cmp)
	}
}

func TestSelectExpr_Load(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	// Load from x + 8
	expr := cminor.Eload{
		Chunk: cminor.Mint32,
		Addr: cminor.Ebinop{
			Op:    cminor.Oadd,
			Left:  cminor.Evar{Name: "ptr"},
			Right: cminor.Econst{Const: cminor.Ointconst{Value: 8}},
		},
	}
	result := ctx.SelectExpr(expr)

	ld, ok := result.(cminorsel.Eload)
	if !ok {
		t.Fatalf("expected Eload, got %T", result)
	}
	if ld.Chunk != cminorsel.Mint32 {
		t.Errorf("expected Mint32, got %v", ld.Chunk)
	}

	// Should use Aindexed addressing mode
	idx, ok := ld.Mode.(cminorsel.Aindexed)
	if !ok {
		t.Fatalf("expected Aindexed, got %T", ld.Mode)
	}
	if idx.Offset != 8 {
		t.Errorf("expected offset 8, got %d", idx.Offset)
	}
}

func TestSelectExpr_LoadGlobal(t *testing.T) {
	globals := map[string]bool{"array": true}
	ctx := NewSelectionContext(globals, nil)
	// Load from global array
	expr := cminor.Eload{
		Chunk: cminor.Mint64,
		Addr:  cminor.Evar{Name: "array"},
	}
	result := ctx.SelectExpr(expr)

	ld, ok := result.(cminorsel.Eload)
	if !ok {
		t.Fatalf("expected Eload, got %T", result)
	}

	// Should use Aglobal addressing mode
	g, ok := ld.Mode.(cminorsel.Aglobal)
	if !ok {
		t.Fatalf("expected Aglobal, got %T", ld.Mode)
	}
	if g.Symbol != "array" {
		t.Errorf("expected symbol 'array', got %q", g.Symbol)
	}
}

func TestSelectCondition_Cmp(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	expr := cminor.Ecmp{
		Op:    cminor.Ocmp,
		Cmp:   cminor.Clt,
		Left:  cminor.Evar{Name: "x"},
		Right: cminor.Evar{Name: "y"},
	}
	result := ctx.SelectCondition(expr)

	cond, ok := result.(cminorsel.CondCmp)
	if !ok {
		t.Fatalf("expected CondCmp, got %T", result)
	}
	if cond.Cmp != cminorsel.Clt {
		t.Errorf("expected Clt, got %v", cond.Cmp)
	}
}

func TestSelectCondition_ConstTrue(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	expr := cminor.Econst{Const: cminor.Ointconst{Value: 1}}
	result := ctx.SelectCondition(expr)

	if _, ok := result.(cminorsel.CondTrue); !ok {
		t.Fatalf("expected CondTrue, got %T", result)
	}
}

func TestSelectCondition_ConstFalse(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	expr := cminor.Econst{Const: cminor.Ointconst{Value: 0}}
	result := ctx.SelectCondition(expr)

	if _, ok := result.(cminorsel.CondFalse); !ok {
		t.Fatalf("expected CondFalse, got %T", result)
	}
}

func TestSelectCondition_Not(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	expr := cminor.Eunop{
		Op:  cminor.Onotbool,
		Arg: cminor.Evar{Name: "flag"},
	}
	result := ctx.SelectCondition(expr)

	not, ok := result.(cminorsel.CondNot)
	if !ok {
		t.Fatalf("expected CondNot, got %T", result)
	}
	// Inner should be a comparison with zero
	inner, ok := not.Cond.(cminorsel.CondCmp)
	if !ok {
		t.Fatalf("expected inner CondCmp, got %T", not.Cond)
	}
	if inner.Cmp != cminorsel.Cne {
		t.Errorf("expected Cne, got %v", inner.Cmp)
	}
}

func TestSelectCondition_GeneralExpr(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	// Non-comparison expression: check != 0
	expr := cminor.Evar{Name: "flag"}
	result := ctx.SelectCondition(expr)

	cond, ok := result.(cminorsel.CondCmp)
	if !ok {
		t.Fatalf("expected CondCmp, got %T", result)
	}
	if cond.Cmp != cminorsel.Cne {
		t.Errorf("expected Cne, got %v", cond.Cmp)
	}
	// Right should be 0
	if c, ok := cond.Right.(cminorsel.Econst); !ok {
		t.Errorf("expected Econst, got %T", cond.Right)
	} else if v, ok := c.Const.(cminorsel.Ointconst); !ok || v.Value != 0 {
		t.Errorf("expected Ointconst{0}, got %v", c.Const)
	}
}

func TestIsProfitableIfConversion(t *testing.T) {
	tests := []struct {
		name     string
		thenExpr cminor.Expr
		elseExpr cminor.Expr
		expected bool
	}{
		{
			name:     "two vars",
			thenExpr: cminor.Evar{Name: "x"},
			elseExpr: cminor.Evar{Name: "y"},
			expected: true,
		},
		{
			name:     "var and const",
			thenExpr: cminor.Evar{Name: "x"},
			elseExpr: cminor.Econst{Const: cminor.Ointconst{Value: 0}},
			expected: true,
		},
		{
			name:     "simple binop",
			thenExpr: cminor.Ebinop{Op: cminor.Oadd, Left: cminor.Evar{Name: "x"}, Right: cminor.Evar{Name: "y"}},
			elseExpr: cminor.Econst{Const: cminor.Ointconst{Value: 0}},
			expected: true,
		},
		{
			name:     "division not profitable",
			thenExpr: cminor.Ebinop{Op: cminor.Odiv, Left: cminor.Evar{Name: "x"}, Right: cminor.Evar{Name: "y"}},
			elseExpr: cminor.Econst{Const: cminor.Ointconst{Value: 0}},
			expected: false,
		},
		{
			name:     "load not profitable",
			thenExpr: cminor.Eload{Chunk: cminor.Mint32, Addr: cminor.Evar{Name: "ptr"}},
			elseExpr: cminor.Econst{Const: cminor.Ointconst{Value: 0}},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsProfitableIfConversion(tc.thenExpr, tc.elseExpr)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestSelectConditionalExpr(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)

	cond := cminor.Ecmp{
		Op:    cminor.Ocmp,
		Cmp:   cminor.Cgt,
		Left:  cminor.Evar{Name: "x"},
		Right: cminor.Econst{Const: cminor.Ointconst{Value: 0}},
	}
	thenExpr := cminor.Evar{Name: "a"}
	elseExpr := cminor.Evar{Name: "b"}

	result := ctx.SelectConditionalExpr(cond, thenExpr, elseExpr)

	ce, ok := result.(cminorsel.Econdition)
	if !ok {
		t.Fatalf("expected Econdition, got %T", result)
	}

	// Check condition
	if _, ok := ce.Cond.(cminorsel.CondCmp); !ok {
		t.Errorf("expected CondCmp, got %T", ce.Cond)
	}

	// Check then
	if v, ok := ce.Then.(cminorsel.Evar); !ok || v.Name != "a" {
		t.Errorf("expected Evar{a}, got %T", ce.Then)
	}

	// Check else
	if v, ok := ce.Else.(cminorsel.Evar); !ok || v.Name != "b" {
		t.Errorf("expected Evar{b}, got %T", ce.Else)
	}
}

func TestSelectExpr_NestedLoad(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	// *(*ptr + 4)
	expr := cminor.Eload{
		Chunk: cminor.Mint32,
		Addr: cminor.Ebinop{
			Op: cminor.Oadd,
			Left: cminor.Eload{
				Chunk: cminor.Mint64,
				Addr:  cminor.Evar{Name: "ptr"},
			},
			Right: cminor.Econst{Const: cminor.Ointconst{Value: 4}},
		},
	}
	result := ctx.SelectExpr(expr)

	ld, ok := result.(cminorsel.Eload)
	if !ok {
		t.Fatalf("expected Eload, got %T", result)
	}
	if ld.Chunk != cminorsel.Mint32 {
		t.Errorf("expected Mint32, got %v", ld.Chunk)
	}
}

package selection

import (
	"testing"

	"github.com/raymyers/ralph-cc/pkg/cminor"
	"github.com/raymyers/ralph-cc/pkg/cminorsel"
)

func TestSelectStmt_Skip(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	result := ctx.SelectStmt(cminor.Sskip{})

	if _, ok := result.(cminorsel.Sskip); !ok {
		t.Fatalf("expected Sskip, got %T", result)
	}
}

func TestSelectStmt_Assign(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	stmt := cminor.Sassign{
		Name: "x",
		RHS:  cminor.Econst{Const: cminor.Ointconst{Value: 42}},
	}
	result := ctx.SelectStmt(stmt)

	assign, ok := result.(cminorsel.Sassign)
	if !ok {
		t.Fatalf("expected Sassign, got %T", result)
	}
	if assign.Name != "x" {
		t.Errorf("expected name 'x', got %q", assign.Name)
	}
}

func TestSelectStmt_Store(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	// Store value 42 at ptr + 8
	stmt := cminor.Sstore{
		Chunk: cminor.Mint32,
		Addr: cminor.Ebinop{
			Op:    cminor.Oadd,
			Left:  cminor.Evar{Name: "ptr"},
			Right: cminor.Econst{Const: cminor.Ointconst{Value: 8}},
		},
		Value: cminor.Econst{Const: cminor.Ointconst{Value: 42}},
	}
	result := ctx.SelectStmt(stmt)

	store, ok := result.(cminorsel.Sstore)
	if !ok {
		t.Fatalf("expected Sstore, got %T", result)
	}
	if store.Chunk != cminorsel.Mint32 {
		t.Errorf("expected Mint32, got %v", store.Chunk)
	}

	// Should use Aindexed addressing mode
	idx, ok := store.Mode.(cminorsel.Aindexed)
	if !ok {
		t.Fatalf("expected Aindexed, got %T", store.Mode)
	}
	if idx.Offset != 8 {
		t.Errorf("expected offset 8, got %d", idx.Offset)
	}
}

func TestSelectStmt_StoreGlobal(t *testing.T) {
	globals := map[string]bool{"array": true}
	ctx := NewSelectionContext(globals, nil)
	// Store at global array
	stmt := cminor.Sstore{
		Chunk: cminor.Mint64,
		Addr:  cminor.Evar{Name: "array"},
		Value: cminor.Econst{Const: cminor.Olongconst{Value: 100}},
	}
	result := ctx.SelectStmt(stmt)

	store, ok := result.(cminorsel.Sstore)
	if !ok {
		t.Fatalf("expected Sstore, got %T", result)
	}

	// Should use Aglobal addressing mode
	g, ok := store.Mode.(cminorsel.Aglobal)
	if !ok {
		t.Fatalf("expected Aglobal, got %T", store.Mode)
	}
	if g.Symbol != "array" {
		t.Errorf("expected symbol 'array', got %q", g.Symbol)
	}
}

func TestSelectStmt_Call(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	result := "r"
	stmt := cminor.Scall{
		Result: &result,
		Func:   cminor.Evar{Name: "foo"},
		Args: []cminor.Expr{
			cminor.Evar{Name: "x"},
			cminor.Econst{Const: cminor.Ointconst{Value: 1}},
		},
	}
	sel := ctx.SelectStmt(stmt)

	call, ok := sel.(cminorsel.Scall)
	if !ok {
		t.Fatalf("expected Scall, got %T", sel)
	}
	if call.Result == nil || *call.Result != "r" {
		t.Error("result mismatch")
	}
	if len(call.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(call.Args))
	}
}

func TestSelectStmt_CallVoid(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	stmt := cminor.Scall{
		Result: nil,
		Func:   cminor.Evar{Name: "print"},
		Args:   []cminor.Expr{cminor.Evar{Name: "msg"}},
	}
	sel := ctx.SelectStmt(stmt)

	call, ok := sel.(cminorsel.Scall)
	if !ok {
		t.Fatalf("expected Scall, got %T", sel)
	}
	if call.Result != nil {
		t.Error("expected nil result for void call")
	}
}

func TestSelectStmt_Tailcall(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	stmt := cminor.Stailcall{
		Func: cminor.Evar{Name: "foo"},
		Args: []cminor.Expr{cminor.Evar{Name: "x"}},
	}
	sel := ctx.SelectStmt(stmt)

	tc, ok := sel.(cminorsel.Stailcall)
	if !ok {
		t.Fatalf("expected Stailcall, got %T", sel)
	}
	if len(tc.Args) != 1 {
		t.Errorf("expected 1 arg, got %d", len(tc.Args))
	}
}

func TestSelectStmt_Builtin(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	result := "r"
	stmt := cminor.Sbuiltin{
		Result:  &result,
		Builtin: "__builtin_memcpy",
		Args:    []cminor.Expr{cminor.Evar{Name: "dst"}, cminor.Evar{Name: "src"}},
	}
	sel := ctx.SelectStmt(stmt)

	bi, ok := sel.(cminorsel.Sbuiltin)
	if !ok {
		t.Fatalf("expected Sbuiltin, got %T", sel)
	}
	if bi.Builtin != "__builtin_memcpy" {
		t.Errorf("expected __builtin_memcpy, got %q", bi.Builtin)
	}
	if len(bi.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(bi.Args))
	}
}

func TestSelectStmt_Seq(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	stmt := cminor.Sseq{
		First:  cminor.Sassign{Name: "x", RHS: cminor.Econst{Const: cminor.Ointconst{Value: 1}}},
		Second: cminor.Sassign{Name: "y", RHS: cminor.Econst{Const: cminor.Ointconst{Value: 2}}},
	}
	sel := ctx.SelectStmt(stmt)

	seq, ok := sel.(cminorsel.Sseq)
	if !ok {
		t.Fatalf("expected Sseq, got %T", sel)
	}
	if _, ok := seq.First.(cminorsel.Sassign); !ok {
		t.Error("expected First to be Sassign")
	}
	if _, ok := seq.Second.(cminorsel.Sassign); !ok {
		t.Error("expected Second to be Sassign")
	}
}

func TestSelectStmt_Ifthenelse(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	stmt := cminor.Sifthenelse{
		Cond: cminor.Ecmp{
			Op:    cminor.Ocmp,
			Cmp:   cminor.Cgt,
			Left:  cminor.Evar{Name: "x"},
			Right: cminor.Econst{Const: cminor.Ointconst{Value: 0}},
		},
		Then: cminor.Sassign{Name: "y", RHS: cminor.Econst{Const: cminor.Ointconst{Value: 1}}},
		Else: cminor.Sassign{Name: "y", RHS: cminor.Econst{Const: cminor.Ointconst{Value: 0}}},
	}
	sel := ctx.SelectStmt(stmt)

	ite, ok := sel.(cminorsel.Sifthenelse)
	if !ok {
		t.Fatalf("expected Sifthenelse, got %T", sel)
	}

	// Check condition is properly selected
	cond, ok := ite.Cond.(cminorsel.CondCmp)
	if !ok {
		t.Fatalf("expected CondCmp, got %T", ite.Cond)
	}
	if cond.Cmp != cminorsel.Cgt {
		t.Errorf("expected Cgt, got %v", cond.Cmp)
	}
}

func TestSelectStmt_Loop(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	stmt := cminor.Sloop{
		Body: cminor.Sassign{Name: "i", RHS: cminor.Ebinop{
			Op:    cminor.Oadd,
			Left:  cminor.Evar{Name: "i"},
			Right: cminor.Econst{Const: cminor.Ointconst{Value: 1}},
		}},
	}
	sel := ctx.SelectStmt(stmt)

	loop, ok := sel.(cminorsel.Sloop)
	if !ok {
		t.Fatalf("expected Sloop, got %T", sel)
	}
	if _, ok := loop.Body.(cminorsel.Sassign); !ok {
		t.Error("expected body to be Sassign")
	}
}

func TestSelectStmt_Block(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	stmt := cminor.Sblock{
		Body: cminor.Sskip{},
	}
	sel := ctx.SelectStmt(stmt)

	block, ok := sel.(cminorsel.Sblock)
	if !ok {
		t.Fatalf("expected Sblock, got %T", sel)
	}
	if _, ok := block.Body.(cminorsel.Sskip); !ok {
		t.Error("expected body to be Sskip")
	}
}

func TestSelectStmt_Exit(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	stmt := cminor.Sexit{N: 2}
	sel := ctx.SelectStmt(stmt)

	exit, ok := sel.(cminorsel.Sexit)
	if !ok {
		t.Fatalf("expected Sexit, got %T", sel)
	}
	if exit.N != 2 {
		t.Errorf("expected N=2, got %d", exit.N)
	}
}

func TestSelectStmt_Switch(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	stmt := cminor.Sswitch{
		IsLong: false,
		Expr:   cminor.Evar{Name: "x"},
		Cases: []cminor.SwitchCase{
			{Value: 1, Body: cminor.Sassign{Name: "y", RHS: cminor.Econst{Const: cminor.Ointconst{Value: 10}}}},
			{Value: 2, Body: cminor.Sassign{Name: "y", RHS: cminor.Econst{Const: cminor.Ointconst{Value: 20}}}},
		},
		Default: cminor.Sassign{Name: "y", RHS: cminor.Econst{Const: cminor.Ointconst{Value: 0}}},
	}
	sel := ctx.SelectStmt(stmt)

	sw, ok := sel.(cminorsel.Sswitch)
	if !ok {
		t.Fatalf("expected Sswitch, got %T", sel)
	}
	if sw.IsLong {
		t.Error("expected IsLong=false")
	}
	if len(sw.Cases) != 2 {
		t.Errorf("expected 2 cases, got %d", len(sw.Cases))
	}
	if sw.Cases[0].Value != 1 || sw.Cases[1].Value != 2 {
		t.Error("case values mismatch")
	}
}

func TestSelectStmt_Return(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	stmt := cminor.Sreturn{
		Value: cminor.Evar{Name: "result"},
	}
	sel := ctx.SelectStmt(stmt)

	ret, ok := sel.(cminorsel.Sreturn)
	if !ok {
		t.Fatalf("expected Sreturn, got %T", sel)
	}
	if ret.Value == nil {
		t.Error("expected non-nil value")
	}
}

func TestSelectStmt_ReturnVoid(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	stmt := cminor.Sreturn{Value: nil}
	sel := ctx.SelectStmt(stmt)

	ret, ok := sel.(cminorsel.Sreturn)
	if !ok {
		t.Fatalf("expected Sreturn, got %T", sel)
	}
	if ret.Value != nil {
		t.Error("expected nil value for void return")
	}
}

func TestSelectStmt_Label(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	stmt := cminor.Slabel{
		Label: "loop_start",
		Body:  cminor.Sskip{},
	}
	sel := ctx.SelectStmt(stmt)

	label, ok := sel.(cminorsel.Slabel)
	if !ok {
		t.Fatalf("expected Slabel, got %T", sel)
	}
	if label.Label != "loop_start" {
		t.Errorf("expected label 'loop_start', got %q", label.Label)
	}
}

func TestSelectStmt_Goto(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	stmt := cminor.Sgoto{Label: "exit"}
	sel := ctx.SelectStmt(stmt)

	gt, ok := sel.(cminorsel.Sgoto)
	if !ok {
		t.Fatalf("expected Sgoto, got %T", sel)
	}
	if gt.Label != "exit" {
		t.Errorf("expected label 'exit', got %q", gt.Label)
	}
}

func TestSelectFunction(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	f := cminor.Function{
		Name:       "add",
		Sig:        cminor.Sig{Args: []string{"i", "i"}, Return: "i"},
		Params:     []string{"a", "b"},
		Vars:       []string{"result"},
		Stackspace: 8,
		Body: cminor.Sseq{
			First: cminor.Sassign{
				Name: "result",
				RHS: cminor.Ebinop{
					Op:    cminor.Oadd,
					Left:  cminor.Evar{Name: "a"},
					Right: cminor.Evar{Name: "b"},
				},
			},
			Second: cminor.Sreturn{Value: cminor.Evar{Name: "result"}},
		},
	}
	sel := ctx.SelectFunction(f)

	if sel.Name != "add" {
		t.Errorf("expected name 'add', got %q", sel.Name)
	}
	if len(sel.Params) != 2 {
		t.Errorf("expected 2 params, got %d", len(sel.Params))
	}
	if sel.Stackspace != 8 {
		t.Errorf("expected stackspace 8, got %d", sel.Stackspace)
	}
}

func TestSelectProgram(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	prog := cminor.Program{
		Globals: []cminor.GlobVar{
			{Name: "counter", Size: 4, Init: nil},
		},
		Functions: []cminor.Function{
			{
				Name:       "inc",
				Sig:        cminor.Sig{Return: "v"},
				Body:       cminor.Sskip{},
				Stackspace: 0,
			},
		},
	}
	sel := ctx.SelectProgram(prog)

	if len(sel.Globals) != 1 {
		t.Errorf("expected 1 global, got %d", len(sel.Globals))
	}
	if sel.Globals[0].Name != "counter" {
		t.Errorf("expected global 'counter', got %q", sel.Globals[0].Name)
	}
	if len(sel.Functions) != 1 {
		t.Errorf("expected 1 function, got %d", len(sel.Functions))
	}
	if sel.Functions[0].Name != "inc" {
		t.Errorf("expected function 'inc', got %q", sel.Functions[0].Name)
	}
}

func TestSelectProgram_GlobalsPopulated(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	// Program with a function that accesses a global
	prog := cminor.Program{
		Globals: []cminor.GlobVar{
			{Name: "data", Size: 8, Init: nil},
		},
		Functions: []cminor.Function{
			{
				Name:       "read_data",
				Sig:        cminor.Sig{Return: "l"},
				Body:       cminor.Sreturn{Value: cminor.Eload{Chunk: cminor.Mint64, Addr: cminor.Evar{Name: "data"}}},
				Stackspace: 0,
			},
		},
	}
	sel := ctx.SelectProgram(prog)

	// The return should use Aglobal addressing for 'data'
	ret, ok := sel.Functions[0].Body.(cminorsel.Sreturn)
	if !ok {
		t.Fatalf("expected Sreturn, got %T", sel.Functions[0].Body)
	}
	ld, ok := ret.Value.(cminorsel.Eload)
	if !ok {
		t.Fatalf("expected Eload, got %T", ret.Value)
	}
	g, ok := ld.Mode.(cminorsel.Aglobal)
	if !ok {
		t.Fatalf("expected Aglobal addressing mode, got %T", ld.Mode)
	}
	if g.Symbol != "data" {
		t.Errorf("expected symbol 'data', got %q", g.Symbol)
	}
}

func TestSelectProgram_ExternalFunctions(t *testing.T) {
	ctx := NewSelectionContext(nil, nil)
	// Program with a call to an external function (like printf)
	// The external function is not defined in the program, only called
	prog := cminor.Program{
		Globals:   []cminor.GlobVar{},
		Functions: []cminor.Function{
			{
				Name: "main",
				Sig:  cminor.Sig{Return: "i"},
				// Call to "printf" which is external (not in p.Functions)
				Body: cminor.Scall{
					Result: nil,
					Func:   cminor.Evar{Name: "printf"},
					Args:   []cminor.Expr{cminor.Econst{Const: cminor.Ointconst{Value: 0}}},
				},
				Stackspace: 0,
			},
		},
	}
	sel := ctx.SelectProgram(prog)

	// The call's function should be Oaddrsymbol, not Evar
	call, ok := sel.Functions[0].Body.(cminorsel.Scall)
	if !ok {
		t.Fatalf("expected Scall, got %T", sel.Functions[0].Body)
	}
	econst, ok := call.Func.(cminorsel.Econst)
	if !ok {
		t.Fatalf("expected function to be Econst (address of symbol), got %T", call.Func)
	}
	addr, ok := econst.Const.(cminorsel.Oaddrsymbol)
	if !ok {
		t.Fatalf("expected Oaddrsymbol, got %T", econst.Const)
	}
	if addr.Symbol != "printf" {
		t.Errorf("expected symbol 'printf', got %q", addr.Symbol)
	}
}

func TestCollectExternalFunctions(t *testing.T) {
	// Test the collectExternalFunctions helper directly
	defined := map[string]bool{"main": true, "helper": true}

	// Scall with external function
	stmt := cminor.Scall{
		Func: cminor.Evar{Name: "printf"},
		Args: []cminor.Expr{},
	}

	externals := make(map[string]bool)
	collectExternalFunctionsInStmt(stmt, defined, externals)

	if !externals["printf"] {
		t.Error("expected 'printf' to be detected as external")
	}

	// Scall with defined function should NOT be in externals
	stmt2 := cminor.Scall{
		Func: cminor.Evar{Name: "helper"},
		Args: []cminor.Expr{},
	}

	externals2 := make(map[string]bool)
	collectExternalFunctionsInStmt(stmt2, defined, externals2)

	if externals2["helper"] {
		t.Error("'helper' should not be detected as external - it's defined")
	}
}

func TestCollectExternalFunctions_Nested(t *testing.T) {
	// Test that external functions are found in nested statements
	defined := map[string]bool{"main": true}

	// Nested call: if (...) { printf(...); }
	stmt := cminor.Sifthenelse{
		Cond: cminor.Econst{Const: cminor.Ointconst{Value: 1}},
		Then: cminor.Scall{
			Func: cminor.Evar{Name: "printf"},
			Args: []cminor.Expr{},
		},
		Else: cminor.Sskip{},
	}

	externals := make(map[string]bool)
	collectExternalFunctionsInStmt(stmt, defined, externals)

	if !externals["printf"] {
		t.Error("expected 'printf' to be detected as external in nested if-then")
	}
}

package cminorgen

import (
	"testing"

	"github.com/raymyers/ralph-cc/pkg/cminor"
	"github.com/raymyers/ralph-cc/pkg/csharpminor"
	"github.com/raymyers/ralph-cc/pkg/ctypes"
)

func TestTransformExpr_Const(t *testing.T) {
	env := &VarEnv{Vars: make(map[string]*VarInfo)}
	tr := NewTransformer(env, nil)

	tests := []struct {
		name     string
		input    csharpminor.Expr
		expected cminor.Expr
	}{
		{
			name:     "int constant",
			input:    csharpminor.Econst{Const: csharpminor.Ointconst{Value: 42}},
			expected: cminor.Econst{Const: cminor.Ointconst{Value: 42}},
		},
		{
			name:     "long constant",
			input:    csharpminor.Econst{Const: csharpminor.Olongconst{Value: 1234567890123}},
			expected: cminor.Econst{Const: cminor.Olongconst{Value: 1234567890123}},
		},
		{
			name:     "float constant",
			input:    csharpminor.Econst{Const: csharpminor.Ofloatconst{Value: 3.14}},
			expected: cminor.Econst{Const: cminor.Ofloatconst{Value: 3.14}},
		},
		{
			name:     "single constant",
			input:    csharpminor.Econst{Const: csharpminor.Osingleconst{Value: 2.5}},
			expected: cminor.Econst{Const: cminor.Osingleconst{Value: 2.5}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tr.TransformExpr(tt.input)
			if result != tt.expected {
				t.Errorf("got %#v, want %#v", result, tt.expected)
			}
		})
	}
}

func TestTransformExpr_Var(t *testing.T) {
	env := &VarEnv{Vars: make(map[string]*VarInfo)}
	tr := NewTransformer(env, nil)

	// Global variable reference
	result := tr.TransformExpr(csharpminor.Evar{Name: "global_x"})
	expected := cminor.Evar{Name: "global_x"}
	if result != expected {
		t.Errorf("global var: got %#v, want %#v", result, expected)
	}
}

func TestTransformExpr_Tempvar(t *testing.T) {
	env := &VarEnv{Vars: make(map[string]*VarInfo)}
	tr := NewTransformer(env, nil)

	// Temp variable → named variable
	result := tr.TransformExpr(csharpminor.Etempvar{ID: 0})
	expected := cminor.Evar{Name: "_t0"}
	if result != expected {
		t.Errorf("temp var: got %#v, want %#v", result, expected)
	}

	// Same temp ID should return same name
	result2 := tr.TransformExpr(csharpminor.Etempvar{ID: 0})
	if result != result2 {
		t.Error("same temp ID should return same name")
	}
}

func TestTransformExpr_Binop(t *testing.T) {
	env := &VarEnv{Vars: make(map[string]*VarInfo)}
	tr := NewTransformer(env, nil)

	input := csharpminor.Ebinop{
		Op:    csharpminor.Oadd,
		Left:  csharpminor.Econst{Const: csharpminor.Ointconst{Value: 1}},
		Right: csharpminor.Econst{Const: csharpminor.Ointconst{Value: 2}},
	}

	result := tr.TransformExpr(input)
	binop, ok := result.(cminor.Ebinop)
	if !ok {
		t.Fatalf("expected Ebinop, got %T", result)
	}
	if binop.Op != cminor.Oadd {
		t.Errorf("op: got %v, want Oadd", binop.Op)
	}
}

func TestTransformExpr_Cmp(t *testing.T) {
	env := &VarEnv{Vars: make(map[string]*VarInfo)}
	tr := NewTransformer(env, nil)

	input := csharpminor.Ecmp{
		Op:    csharpminor.Ocmp,
		Cmp:   csharpminor.Ceq,
		Left:  csharpminor.Econst{Const: csharpminor.Ointconst{Value: 1}},
		Right: csharpminor.Econst{Const: csharpminor.Ointconst{Value: 2}},
	}

	result := tr.TransformExpr(input)
	cmp, ok := result.(cminor.Ecmp)
	if !ok {
		t.Fatalf("expected Ecmp, got %T", result)
	}
	if cmp.Op != cminor.Ocmp {
		t.Errorf("op: got %v, want Ocmp", cmp.Op)
	}
	if cmp.Cmp != cminor.Ceq {
		t.Errorf("cmp: got %v, want Ceq", cmp.Cmp)
	}
}

func TestTransformExpr_Load(t *testing.T) {
	env := &VarEnv{Vars: make(map[string]*VarInfo)}
	tr := NewTransformer(env, nil)

	input := csharpminor.Eload{
		Chunk: csharpminor.Mint32,
		Addr:  csharpminor.Econst{Const: csharpminor.Olongconst{Value: 100}},
	}

	result := tr.TransformExpr(input)
	load, ok := result.(cminor.Eload)
	if !ok {
		t.Fatalf("expected Eload, got %T", result)
	}
	if load.Chunk != cminor.Mint32 {
		t.Errorf("chunk: got %v, want Mint32", load.Chunk)
	}
}

func TestTransformStmt_Skip(t *testing.T) {
	env := &VarEnv{Vars: make(map[string]*VarInfo)}
	tr := NewTransformer(env, nil)

	result := tr.TransformStmt(csharpminor.Sskip{})
	if _, ok := result.(cminor.Sskip); !ok {
		t.Errorf("expected Sskip, got %T", result)
	}
}

func TestTransformStmt_Set(t *testing.T) {
	env := &VarEnv{Vars: make(map[string]*VarInfo)}
	tr := NewTransformer(env, nil)

	input := csharpminor.Sset{
		TempID: 0,
		RHS:    csharpminor.Econst{Const: csharpminor.Ointconst{Value: 42}},
	}

	result := tr.TransformStmt(input)
	assign, ok := result.(cminor.Sassign)
	if !ok {
		t.Fatalf("expected Sassign, got %T", result)
	}
	if assign.Name != "_t0" {
		t.Errorf("name: got %q, want %q", assign.Name, "_t0")
	}
}

func TestTransformStmt_Store(t *testing.T) {
	env := &VarEnv{Vars: make(map[string]*VarInfo)}
	tr := NewTransformer(env, nil)

	input := csharpminor.Sstore{
		Chunk: csharpminor.Mint32,
		Addr:  csharpminor.Econst{Const: csharpminor.Olongconst{Value: 100}},
		Value: csharpminor.Econst{Const: csharpminor.Ointconst{Value: 42}},
	}

	result := tr.TransformStmt(input)
	store, ok := result.(cminor.Sstore)
	if !ok {
		t.Fatalf("expected Sstore, got %T", result)
	}
	if store.Chunk != cminor.Mint32 {
		t.Errorf("chunk: got %v, want Mint32", store.Chunk)
	}
}

func TestTransformStmt_Seq(t *testing.T) {
	env := &VarEnv{Vars: make(map[string]*VarInfo)}
	tr := NewTransformer(env, nil)

	input := csharpminor.Sseq{
		First:  csharpminor.Sskip{},
		Second: csharpminor.Sskip{},
	}

	result := tr.TransformStmt(input)
	seq, ok := result.(cminor.Sseq)
	if !ok {
		t.Fatalf("expected Sseq, got %T", result)
	}
	if _, ok := seq.First.(cminor.Sskip); !ok {
		t.Error("First should be Sskip")
	}
	if _, ok := seq.Second.(cminor.Sskip); !ok {
		t.Error("Second should be Sskip")
	}
}

func TestTransformStmt_Ifthenelse(t *testing.T) {
	env := &VarEnv{Vars: make(map[string]*VarInfo)}
	tr := NewTransformer(env, nil)

	input := csharpminor.Sifthenelse{
		Cond: csharpminor.Econst{Const: csharpminor.Ointconst{Value: 1}},
		Then: csharpminor.Sskip{},
		Else: csharpminor.Sskip{},
	}

	result := tr.TransformStmt(input)
	ifStmt, ok := result.(cminor.Sifthenelse)
	if !ok {
		t.Fatalf("expected Sifthenelse, got %T", result)
	}
	if _, ok := ifStmt.Then.(cminor.Sskip); !ok {
		t.Error("Then should be Sskip")
	}
}

func TestTransformStmt_Loop(t *testing.T) {
	env := &VarEnv{Vars: make(map[string]*VarInfo)}
	tr := NewTransformer(env, nil)

	input := csharpminor.Sloop{Body: csharpminor.Sskip{}}

	result := tr.TransformStmt(input)
	loop, ok := result.(cminor.Sloop)
	if !ok {
		t.Fatalf("expected Sloop, got %T", result)
	}
	if _, ok := loop.Body.(cminor.Sskip); !ok {
		t.Error("Body should be Sskip")
	}
}

func TestTransformStmt_Block(t *testing.T) {
	env := &VarEnv{Vars: make(map[string]*VarInfo)}
	tr := NewTransformer(env, nil)

	input := csharpminor.Sblock{Body: csharpminor.Sskip{}}

	result := tr.TransformStmt(input)
	block, ok := result.(cminor.Sblock)
	if !ok {
		t.Fatalf("expected Sblock, got %T", result)
	}
	if _, ok := block.Body.(cminor.Sskip); !ok {
		t.Error("Body should be Sskip")
	}
}

func TestTransformStmt_Exit(t *testing.T) {
	env := &VarEnv{Vars: make(map[string]*VarInfo)}
	tr := NewTransformer(env, nil)

	input := csharpminor.Sexit{N: 2}

	result := tr.TransformStmt(input)
	exit, ok := result.(cminor.Sexit)
	if !ok {
		t.Fatalf("expected Sexit, got %T", result)
	}
	if exit.N != 2 {
		t.Errorf("N: got %d, want 2", exit.N)
	}
}

func TestTransformStmt_Return(t *testing.T) {
	env := &VarEnv{Vars: make(map[string]*VarInfo)}
	tr := NewTransformer(env, nil)

	// Return with value
	input := csharpminor.Sreturn{
		Value: csharpminor.Econst{Const: csharpminor.Ointconst{Value: 0}},
	}

	result := tr.TransformStmt(input)
	ret, ok := result.(cminor.Sreturn)
	if !ok {
		t.Fatalf("expected Sreturn, got %T", result)
	}
	if ret.Value == nil {
		t.Error("Value should not be nil")
	}

	// Return without value
	input2 := csharpminor.Sreturn{Value: nil}
	result2 := tr.TransformStmt(input2)
	ret2, ok := result2.(cminor.Sreturn)
	if !ok {
		t.Fatalf("expected Sreturn, got %T", result2)
	}
	if ret2.Value != nil {
		t.Error("Value should be nil for void return")
	}
}

func TestTransformStmt_Label(t *testing.T) {
	env := &VarEnv{Vars: make(map[string]*VarInfo)}
	tr := NewTransformer(env, nil)

	input := csharpminor.Slabel{
		Label: "loop_start",
		Body:  csharpminor.Sskip{},
	}

	result := tr.TransformStmt(input)
	label, ok := result.(cminor.Slabel)
	if !ok {
		t.Fatalf("expected Slabel, got %T", result)
	}
	if label.Label != "loop_start" {
		t.Errorf("Label: got %q, want %q", label.Label, "loop_start")
	}
}

func TestTransformStmt_Goto(t *testing.T) {
	env := &VarEnv{Vars: make(map[string]*VarInfo)}
	tr := NewTransformer(env, nil)

	input := csharpminor.Sgoto{Label: "end"}

	result := tr.TransformStmt(input)
	gt, ok := result.(cminor.Sgoto)
	if !ok {
		t.Fatalf("expected Sgoto, got %T", result)
	}
	if gt.Label != "end" {
		t.Errorf("Label: got %q, want %q", gt.Label, "end")
	}
}

func TestTransformStmt_Switch(t *testing.T) {
	env := &VarEnv{Vars: make(map[string]*VarInfo)}
	tr := NewTransformer(env, nil)

	input := csharpminor.Sswitch{
		IsLong: false,
		Expr:   csharpminor.Etempvar{ID: 0},
		Cases: []csharpminor.SwitchCase{
			{Value: 1, Body: csharpminor.Sexit{N: 1}},
			{Value: 2, Body: csharpminor.Sexit{N: 2}},
		},
		Default: csharpminor.Sexit{N: 0},
	}

	result := tr.TransformStmt(input)
	sw, ok := result.(cminor.Sswitch)
	if !ok {
		t.Fatalf("expected Sswitch, got %T", result)
	}
	if sw.IsLong {
		t.Error("IsLong should be false")
	}
	if len(sw.Cases) != 2 {
		t.Errorf("Cases: got %d, want 2", len(sw.Cases))
	}
}

func TestTransformStmt_Call(t *testing.T) {
	env := &VarEnv{Vars: make(map[string]*VarInfo)}
	tr := NewTransformer(env, nil)

	resultID := 0
	input := csharpminor.Scall{
		Result: &resultID,
		Func:   csharpminor.Evar{Name: "foo"},
		Args: []csharpminor.Expr{
			csharpminor.Econst{Const: csharpminor.Ointconst{Value: 1}},
		},
	}

	result := tr.TransformStmt(input)
	call, ok := result.(cminor.Scall)
	if !ok {
		t.Fatalf("expected Scall, got %T", result)
	}
	if call.Result == nil || *call.Result != "_t0" {
		t.Error("Result should be '_t0'")
	}
	if len(call.Args) != 1 {
		t.Errorf("Args: got %d, want 1", len(call.Args))
	}
}

func TestTransformFunction(t *testing.T) {
	fn := &csharpminor.Function{
		Name: "add",
		Sig: csharpminor.Sig{
			Args:   []ctypes.Type{ctypes.Int(), ctypes.Int()},
			Return: ctypes.Int(),
		},
		Params: []string{"a", "b"},
		Locals: []csharpminor.VarDecl{},
		Temps:  []ctypes.Type{ctypes.Int()},
		Body: csharpminor.Sset{
			TempID: 0,
			RHS: csharpminor.Ebinop{
				Op:    csharpminor.Oadd,
				Left:  csharpminor.Etempvar{ID: 0},
				Right: csharpminor.Etempvar{ID: 0},
			},
		},
	}

	result := TransformFunction(fn, nil)

	if result.Name != "add" {
		t.Errorf("Name: got %q, want %q", result.Name, "add")
	}
	if len(result.Params) != 2 {
		t.Errorf("Params: got %d, want 2", len(result.Params))
	}
	if result.Stackspace != 0 {
		t.Errorf("Stackspace: got %d, want 0", result.Stackspace)
	}
}

func TestTransformFunction_WithStackVar(t *testing.T) {
	// Function with an address-taken local that goes on stack
	fn := &csharpminor.Function{
		Name: "with_stack",
		Sig: csharpminor.Sig{
			Return: ctypes.Int(),
		},
		Params: []string{},
		Locals: []csharpminor.VarDecl{
			{Name: "x", Size: 4},
		},
		Temps: []ctypes.Type{},
		Body: csharpminor.Sstore{
			Chunk: csharpminor.Mint32,
			Addr:  csharpminor.Eaddrof{Name: "x"}, // Address taken
			Value: csharpminor.Econst{Const: csharpminor.Ointconst{Value: 42}},
		},
	}

	result := TransformFunction(fn, nil)

	if result.Stackspace == 0 {
		t.Error("Stackspace should be > 0 for address-taken local")
	}
}

func TestTransformProgram(t *testing.T) {
	prog := &csharpminor.Program{
		Globals: []csharpminor.VarDecl{
			{Name: "g", Size: 4},
		},
		Functions: []csharpminor.Function{
			{
				Name: "main",
				Sig: csharpminor.Sig{
					Return: ctypes.Int(),
				},
				Params: []string{},
				Locals: []csharpminor.VarDecl{},
				Temps:  []ctypes.Type{},
				Body:   csharpminor.Sreturn{Value: csharpminor.Econst{Const: csharpminor.Ointconst{Value: 0}}},
			},
		},
	}

	result := TransformProgram(prog)

	if len(result.Globals) != 1 {
		t.Errorf("Globals: got %d, want 1", len(result.Globals))
	}
	if result.Globals[0].Name != "g" {
		t.Errorf("Global name: got %q, want %q", result.Globals[0].Name, "g")
	}
	if len(result.Functions) != 1 {
		t.Errorf("Functions: got %d, want 1", len(result.Functions))
	}
	if result.Functions[0].Name != "main" {
		t.Errorf("Function name: got %q, want %q", result.Functions[0].Name, "main")
	}
}

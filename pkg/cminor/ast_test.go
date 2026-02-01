package cminor

import (
	"testing"
)

func TestChunkString(t *testing.T) {
	tests := []struct {
		chunk Chunk
		want  string
	}{
		{Mint8signed, "int8s"},
		{Mint8unsigned, "int8u"},
		{Mint16signed, "int16s"},
		{Mint16unsigned, "int16u"},
		{Mint32, "int32"},
		{Mint64, "int64"},
		{Mfloat32, "float32"},
		{Mfloat64, "float64"},
		{Many32, "any32"},
		{Many64, "any64"},
	}
	for _, tt := range tests {
		if got := tt.chunk.String(); got != tt.want {
			t.Errorf("Chunk(%d).String() = %q, want %q", tt.chunk, got, tt.want)
		}
	}
}

func TestUnaryOpString(t *testing.T) {
	tests := []struct {
		op   UnaryOp
		want string
	}{
		{Onegint, "negint"},
		{Onegf, "negf"},
		{Onegl, "negl"},
		{Onotint, "notint"},
		{Onotbool, "notbool"},
	}
	for _, tt := range tests {
		if got := tt.op.String(); got != tt.want {
			t.Errorf("UnaryOp(%d).String() = %q, want %q", tt.op, got, tt.want)
		}
	}
}

func TestBinaryOpString(t *testing.T) {
	tests := []struct {
		op   BinaryOp
		want string
	}{
		{Oadd, "add"},
		{Osub, "sub"},
		{Omul, "mul"},
		{Odiv, "div"},
		{Oaddf, "addf"},
		{Oaddl, "addl"},
		{Ocmp, "cmp"},
	}
	for _, tt := range tests {
		if got := tt.op.String(); got != tt.want {
			t.Errorf("BinaryOp(%d).String() = %q, want %q", tt.op, got, tt.want)
		}
	}
}

func TestComparisonString(t *testing.T) {
	tests := []struct {
		cmp  Comparison
		want string
	}{
		{Ceq, "=="},
		{Cne, "!="},
		{Clt, "<"},
		{Cle, "<="},
		{Cgt, ">"},
		{Cge, ">="},
	}
	for _, tt := range tests {
		if got := tt.cmp.String(); got != tt.want {
			t.Errorf("Comparison(%d).String() = %q, want %q", tt.cmp, got, tt.want)
		}
	}
}

func TestConstantTypes(t *testing.T) {
	tests := []struct {
		name string
		c    Constant
	}{
		{"int", Ointconst{Value: 42}},
		{"float", Ofloatconst{Value: 3.14}},
		{"long", Olongconst{Value: 1000000000000}},
		{"single", Osingleconst{Value: 2.5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify the constant implements both interfaces
			var _ Node = tt.c
			var _ Constant = tt.c
		})
	}
}

func TestExpressionTypes(t *testing.T) {
	tests := []struct {
		name string
		e    Expr
	}{
		{"Evar", Evar{Name: "x"}},
		{"Econst", Econst{Const: Ointconst{Value: 1}}},
		{"Eunop", Eunop{Op: Onegint, Arg: Evar{Name: "x"}}},
		{"Ebinop", Ebinop{Op: Oadd, Left: Evar{Name: "x"}, Right: Econst{Const: Ointconst{Value: 1}}}},
		{"Ecmp", Ecmp{Op: Ocmp, Cmp: Ceq, Left: Evar{Name: "x"}, Right: Econst{Const: Ointconst{Value: 0}}}},
		{"Eload", Eload{Chunk: Mint32, Addr: Evar{Name: "p"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify the expression implements both interfaces
			var _ Node = tt.e
			var _ Expr = tt.e
		})
	}
}

func TestStatementTypes(t *testing.T) {
	resultVar := "result"
	tests := []struct {
		name string
		s    Stmt
	}{
		{"Sskip", Sskip{}},
		{"Sassign", Sassign{Name: "x", RHS: Econst{Const: Ointconst{Value: 1}}}},
		{"Sstore", Sstore{Chunk: Mint32, Addr: Evar{Name: "p"}, Value: Econst{Const: Ointconst{Value: 42}}}},
		{"Scall", Scall{Result: &resultVar, Func: Evar{Name: "f"}, Args: nil}},
		{"Stailcall", Stailcall{Func: Evar{Name: "f"}, Args: nil}},
		{"Sseq", Sseq{First: Sskip{}, Second: Sskip{}}},
		{"Sifthenelse", Sifthenelse{Cond: Evar{Name: "c"}, Then: Sskip{}, Else: Sskip{}}},
		{"Sloop", Sloop{Body: Sskip{}}},
		{"Sblock", Sblock{Body: Sskip{}}},
		{"Sexit", Sexit{N: 1}},
		{"Sswitch", Sswitch{Expr: Evar{Name: "x"}, Cases: nil, Default: Sskip{}}},
		{"Sreturn_void", Sreturn{Value: nil}},
		{"Sreturn_value", Sreturn{Value: Evar{Name: "x"}}},
		{"Slabel", Slabel{Label: "L1", Body: Sskip{}}},
		{"Sgoto", Sgoto{Label: "L1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify the statement implements both interfaces
			var _ Node = tt.s
			var _ Stmt = tt.s
		})
	}
}

func TestSeqFlattensSkip(t *testing.T) {
	// Empty seq should be Sskip
	result := Seq()
	if _, ok := result.(Sskip); !ok {
		t.Errorf("Seq() = %T, want Sskip", result)
	}

	// Single statement should be returned directly
	assign := Sassign{Name: "x", RHS: Econst{Const: Ointconst{Value: 1}}}
	result = Seq(assign)
	if _, ok := result.(Sassign); !ok {
		t.Errorf("Seq(Sassign) = %T, want Sassign", result)
	}

	// Sskip should be filtered out
	result = Seq(Sskip{}, assign)
	if _, ok := result.(Sassign); !ok {
		t.Errorf("Seq(Sskip, Sassign) = %T, want Sassign", result)
	}

	// Two non-skip statements should create Sseq
	assign2 := Sassign{Name: "y", RHS: Econst{Const: Ointconst{Value: 2}}}
	result = Seq(assign, assign2)
	if seq, ok := result.(Sseq); !ok {
		t.Errorf("Seq(Sassign, Sassign) = %T, want Sseq", result)
	} else {
		if _, ok := seq.First.(Sassign); !ok {
			t.Errorf("Sseq.First = %T, want Sassign", seq.First)
		}
		if _, ok := seq.Second.(Sassign); !ok {
			t.Errorf("Sseq.Second = %T, want Sassign", seq.Second)
		}
	}
}

func TestASTConstruction(t *testing.T) {
	// Build a simple function: int f(int x) { return x + 1; }
	body := Sreturn{
		Value: Ebinop{
			Op:    Oadd,
			Left:  Evar{Name: "x"},
			Right: Econst{Const: Ointconst{Value: 1}},
		},
	}

	fn := Function{
		Name:       "f",
		Sig:        Sig{Args: []string{"int"}, Return: "int"},
		Params:     []string{"x"},
		Vars:       nil,
		Stackspace: 0,
		Body:       body,
	}

	if fn.Name != "f" {
		t.Errorf("Function.Name = %q, want %q", fn.Name, "f")
	}
	if len(fn.Params) != 1 || fn.Params[0] != "x" {
		t.Errorf("Function.Params = %v, want [x]", fn.Params)
	}
}

func TestProgramConstruction(t *testing.T) {
	fn := Function{
		Name:       "main",
		Sig:        Sig{Return: "int"},
		Params:     nil,
		Vars:       nil,
		Stackspace: 0,
		Body:       Sreturn{Value: Econst{Const: Ointconst{Value: 0}}},
	}

	glob := GlobVar{
		Name: "g",
		Size: 4,
		Init: nil,
	}

	prog := Program{
		Globals:   []GlobVar{glob},
		Functions: []Function{fn},
	}

	if len(prog.Functions) != 1 {
		t.Errorf("Program has %d functions, want 1", len(prog.Functions))
	}
	if len(prog.Globals) != 1 {
		t.Errorf("Program has %d globals, want 1", len(prog.Globals))
	}
}

func TestBlockAndExit(t *testing.T) {
	// Test block with exit
	block := Sblock{
		Body: Seq(
			Sassign{Name: "x", RHS: Econst{Const: Ointconst{Value: 1}}},
			Sexit{N: 1},
		),
	}

	if _, ok := block.Body.(Sseq); !ok {
		t.Errorf("Block body should be Sseq, got %T", block.Body)
	}
}

func TestTypedOperations(t *testing.T) {
	// Integer operations
	intAdd := Ebinop{Op: Oadd, Left: Evar{Name: "a"}, Right: Evar{Name: "b"}}
	if intAdd.Op != Oadd {
		t.Errorf("Expected Oadd, got %v", intAdd.Op)
	}

	// Long operations
	longMul := Ebinop{Op: Omull, Left: Evar{Name: "a"}, Right: Evar{Name: "b"}}
	if longMul.Op != Omull {
		t.Errorf("Expected Omull, got %v", longMul.Op)
	}

	// Float operations
	floatDiv := Ebinop{Op: Odivf, Left: Evar{Name: "a"}, Right: Evar{Name: "b"}}
	if floatDiv.Op != Odivf {
		t.Errorf("Expected Odivf, got %v", floatDiv.Op)
	}

	// Comparison
	cmp := Ecmp{Op: Ocmp, Cmp: Clt, Left: Evar{Name: "a"}, Right: Evar{Name: "b"}}
	if cmp.Cmp != Clt {
		t.Errorf("Expected Clt, got %v", cmp.Cmp)
	}
}

func TestMemoryOperations(t *testing.T) {
	// Load
	load := Eload{Chunk: Mint32, Addr: Evar{Name: "p"}}
	if load.Chunk != Mint32 {
		t.Errorf("Expected Mint32, got %v", load.Chunk)
	}

	// Store
	store := Sstore{Chunk: Mint64, Addr: Evar{Name: "p"}, Value: Econst{Const: Olongconst{Value: 100}}}
	if store.Chunk != Mint64 {
		t.Errorf("Expected Mint64, got %v", store.Chunk)
	}
}

func TestFunctionWithStackSpace(t *testing.T) {
	// Function with stack-allocated locals
	fn := Function{
		Name:       "f",
		Sig:        Sig{Return: "void"},
		Params:     nil,
		Vars:       []string{"buf"},
		Stackspace: 128, // e.g., for a local array
		Body:       Sskip{},
	}

	if fn.Stackspace != 128 {
		t.Errorf("Function.Stackspace = %d, want 128", fn.Stackspace)
	}
	if len(fn.Vars) != 1 || fn.Vars[0] != "buf" {
		t.Errorf("Function.Vars = %v, want [buf]", fn.Vars)
	}
}

func TestSwitchConstruction(t *testing.T) {
	sw := Sswitch{
		IsLong: false,
		Expr:   Evar{Name: "x"},
		Cases: []SwitchCase{
			{Value: 0, Body: Sreturn{Value: Econst{Const: Ointconst{Value: 10}}}},
			{Value: 1, Body: Sreturn{Value: Econst{Const: Ointconst{Value: 20}}}},
		},
		Default: Sreturn{Value: Econst{Const: Ointconst{Value: -1}}},
	}

	if len(sw.Cases) != 2 {
		t.Errorf("Switch has %d cases, want 2", len(sw.Cases))
	}
	if sw.Cases[0].Value != 0 {
		t.Errorf("First case value = %d, want 0", sw.Cases[0].Value)
	}
}

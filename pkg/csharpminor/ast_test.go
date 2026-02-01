package csharpminor

import (
	"testing"

	"github.com/raymyers/ralph-cc/pkg/ctypes"
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
			t.Errorf("%d.String() = %q, want %q", tt.chunk, got, tt.want)
		}
	}
}

func TestUnaryOpString(t *testing.T) {
	tests := []struct {
		op   UnaryOp
		want string
	}{
		{Ocast8signed, "cast8signed"},
		{Onegint, "negint"},
		{Onegf, "negf"},
		{Onotint, "notint"},
		{Onotbool, "notbool"},
		{Osingleoffloat, "singleoffloat"},
		{Ofloatofsingle, "floatofsingle"},
		{Ointoffloat, "intoffloat"},
		{Olongofint, "longofint"},
	}
	for _, tt := range tests {
		if got := tt.op.String(); got != tt.want {
			t.Errorf("%d.String() = %q, want %q", tt.op, got, tt.want)
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
		{Odivu, "divu"},
		{Oaddf, "addf"},
		{Osubf, "subf"},
		{Oaddl, "addl"},
		{Oand, "and"},
		{Oor, "or"},
		{Oxor, "xor"},
		{Oshl, "shl"},
		{Oshr, "shr"},
		{Ocmp, "cmp"},
		{Ocmpu, "cmpu"},
		{Ocmpf, "cmpf"},
		{Ocmpl, "cmpl"},
	}
	for _, tt := range tests {
		if got := tt.op.String(); got != tt.want {
			t.Errorf("%d.String() = %q, want %q", tt.op, got, tt.want)
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
			t.Errorf("%d.String() = %q, want %q", tt.cmp, got, tt.want)
		}
	}
}

func TestConstantTypes(t *testing.T) {
	// Test that all constant types implement the Constant interface
	var _ Constant = Ointconst{Value: 42}
	var _ Constant = Ofloatconst{Value: 3.14}
	var _ Constant = Olongconst{Value: 123456789}
	var _ Constant = Osingleconst{Value: 2.5}
}

func TestExpressionTypes(t *testing.T) {
	// Test that all expression types implement the Expr interface
	var _ Expr = Evar{Name: "x"}
	var _ Expr = Etempvar{ID: 1}
	var _ Expr = Eaddrof{Name: "f"}
	var _ Expr = Econst{Const: Ointconst{Value: 42}}
	var _ Expr = Eunop{Op: Onegint, Arg: Etempvar{ID: 1}}
	var _ Expr = Ebinop{Op: Oadd, Left: Etempvar{ID: 1}, Right: Etempvar{ID: 2}}
	var _ Expr = Ecmp{Op: Ocmp, Cmp: Ceq, Left: Etempvar{ID: 1}, Right: Etempvar{ID: 2}}
	var _ Expr = Eload{Chunk: Mint32, Addr: Evar{Name: "x"}}
}

func TestStatementTypes(t *testing.T) {
	// Test that all statement types implement the Stmt interface
	var _ Stmt = Sskip{}
	var _ Stmt = Sset{TempID: 1, RHS: Econst{Const: Ointconst{Value: 42}}}
	var _ Stmt = Sstore{Chunk: Mint32, Addr: Evar{Name: "x"}, Value: Etempvar{ID: 1}}
	var _ Stmt = Scall{Result: nil, Func: Eaddrof{Name: "f"}, Args: nil}
	var _ Stmt = Stailcall{Func: Eaddrof{Name: "f"}, Args: nil}
	var _ Stmt = Sbuiltin{Result: nil, Builtin: "memcpy", Args: nil}
	var _ Stmt = Sseq{First: Sskip{}, Second: Sskip{}}
	var _ Stmt = Sifthenelse{Cond: Etempvar{ID: 1}, Then: Sskip{}, Else: Sskip{}}
	var _ Stmt = Sloop{Body: Sskip{}}
	var _ Stmt = Sblock{Body: Sskip{}}
	var _ Stmt = Sexit{N: 1}
	var _ Stmt = Sswitch{Expr: Etempvar{ID: 1}, Cases: nil, Default: Sskip{}}
	var _ Stmt = Sreturn{Value: nil}
	var _ Stmt = Slabel{Label: "L1", Body: Sskip{}}
	var _ Stmt = Sgoto{Label: "L1"}
}

func TestSeqFlattensSkip(t *testing.T) {
	// Single non-skip statement
	s1 := Sreturn{Value: Econst{Const: Ointconst{Value: 0}}}
	result := Seq(s1)
	if _, ok := result.(Sreturn); !ok {
		t.Errorf("Seq(single) should return the statement directly")
	}

	// Skip statements should be ignored
	result = Seq(Sskip{}, s1)
	if _, ok := result.(Sreturn); !ok {
		t.Errorf("Seq(skip, stmt) should return stmt")
	}

	result = Seq(s1, Sskip{})
	if _, ok := result.(Sreturn); !ok {
		t.Errorf("Seq(stmt, skip) should return stmt")
	}

	// Two non-skip statements become sequence
	s2 := Sset{TempID: 1, RHS: Econst{Const: Ointconst{Value: 42}}}
	result = Seq(s2, s1)
	seq, ok := result.(Sseq)
	if !ok {
		t.Errorf("Seq(stmt, stmt) should return Sseq")
	}
	if _, ok := seq.First.(Sset); !ok {
		t.Errorf("First should be Sset")
	}
	if _, ok := seq.Second.(Sreturn); !ok {
		t.Errorf("Second should be Sreturn")
	}
}

func TestChunkForType(t *testing.T) {
	tests := []struct {
		name string
		typ  ctypes.Type
		want Chunk
	}{
		{"int8 signed", ctypes.Tint{Size: ctypes.I8, Sign: ctypes.Signed}, Mint8signed},
		{"int8 unsigned", ctypes.Tint{Size: ctypes.I8, Sign: ctypes.Unsigned}, Mint8unsigned},
		{"int16 signed", ctypes.Tint{Size: ctypes.I16, Sign: ctypes.Signed}, Mint16signed},
		{"int16 unsigned", ctypes.Tint{Size: ctypes.I16, Sign: ctypes.Unsigned}, Mint16unsigned},
		{"int32 signed", ctypes.Tint{Size: ctypes.I32, Sign: ctypes.Signed}, Mint32},
		{"int32 unsigned", ctypes.Tint{Size: ctypes.I32, Sign: ctypes.Unsigned}, Mint32},
		{"bool", ctypes.Tint{Size: ctypes.IBool, Sign: ctypes.Signed}, Mint32},
		{"long signed", ctypes.Tlong{Sign: ctypes.Signed}, Mint64},
		{"long unsigned", ctypes.Tlong{Sign: ctypes.Unsigned}, Mint64},
		{"float32", ctypes.Tfloat{Size: ctypes.F32}, Mfloat32},
		{"float64", ctypes.Tfloat{Size: ctypes.F64}, Mfloat64},
		{"pointer", ctypes.Tpointer{Elem: ctypes.Int()}, Mint64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ChunkForType(tt.typ)
			if got != tt.want {
				t.Errorf("ChunkForType(%v) = %v, want %v", tt.typ, got, tt.want)
			}
		})
	}
}

func TestASTConstruction(t *testing.T) {
	// Build a simple function: int f() { return 42; }
	fn := Function{
		Name: "f",
		Sig: Sig{
			Args:   nil,
			Return: ctypes.Int(),
		},
		Params: nil,
		Locals: nil,
		Temps:  nil,
		Body:   Sreturn{Value: Econst{Const: Ointconst{Value: 42}}},
	}

	if fn.Name != "f" {
		t.Errorf("function name = %q, want %q", fn.Name, "f")
	}
	if len(fn.Params) != 0 {
		t.Errorf("params count = %d, want 0", len(fn.Params))
	}

	ret, ok := fn.Body.(Sreturn)
	if !ok {
		t.Fatalf("body should be Sreturn, got %T", fn.Body)
	}
	constExpr, ok := ret.Value.(Econst)
	if !ok {
		t.Fatalf("return value should be Econst, got %T", ret.Value)
	}
	intConst, ok := constExpr.Const.(Ointconst)
	if !ok {
		t.Fatalf("constant should be Ointconst, got %T", constExpr.Const)
	}
	if intConst.Value != 42 {
		t.Errorf("constant value = %d, want 42", intConst.Value)
	}
}

func TestProgramConstruction(t *testing.T) {
	prog := Program{
		Globals: []VarDecl{
			{Name: "count", Size: 4},
		},
		Functions: []Function{
			{
				Name: "main",
				Sig: Sig{
					Args:   nil,
					Return: ctypes.Int(),
				},
				Body: Sreturn{Value: Econst{Const: Ointconst{Value: 0}}},
			},
		},
	}

	if len(prog.Globals) != 1 {
		t.Errorf("globals count = %d, want 1", len(prog.Globals))
	}
	if prog.Globals[0].Name != "count" {
		t.Errorf("global name = %q, want %q", prog.Globals[0].Name, "count")
	}
	if prog.Globals[0].Size != 4 {
		t.Errorf("global size = %d, want 4", prog.Globals[0].Size)
	}
	if len(prog.Functions) != 1 {
		t.Errorf("functions count = %d, want 1", len(prog.Functions))
	}
}

func TestBlockAndExit(t *testing.T) {
	// Test Sblock and Sexit for break/continue
	// while (x > 0) x-- becomes:
	// block { loop { block { if (x > 0) { x-- } else exit 0 }; exit 1 } }
	// (simplified)
	outerBlock := Sblock{
		Body: Sloop{
			Body: Sblock{
				Body: Sifthenelse{
					Cond: Ecmp{
						Op:    Ocmp,
						Cmp:   Cgt,
						Left:  Etempvar{ID: 1},
						Right: Econst{Const: Ointconst{Value: 0}},
					},
					Then: Sstore{
						Chunk: Mint32,
						Addr:  Evar{Name: "x"},
						Value: Ebinop{
							Op:    Osub,
							Left:  Etempvar{ID: 1},
							Right: Econst{Const: Ointconst{Value: 1}},
						},
					},
					Else: Sexit{N: 1}, // break: exit the loop block
				},
			},
		},
	}

	// Verify structure
	innerLoop, ok := outerBlock.Body.(Sloop)
	if !ok {
		t.Fatalf("inner should be Sloop, got %T", outerBlock.Body)
	}
	innerBlock, ok := innerLoop.Body.(Sblock)
	if !ok {
		t.Fatalf("loop body should be Sblock, got %T", innerLoop.Body)
	}
	ite, ok := innerBlock.Body.(Sifthenelse)
	if !ok {
		t.Fatalf("block body should be Sifthenelse, got %T", innerBlock.Body)
	}
	exit, ok := ite.Else.(Sexit)
	if !ok {
		t.Fatalf("else branch should be Sexit, got %T", ite.Else)
	}
	if exit.N != 1 {
		t.Errorf("exit depth = %d, want 1", exit.N)
	}
}

func TestTypedOperations(t *testing.T) {
	// Test that we can express typed operations correctly

	// Integer addition
	intAdd := Ebinop{
		Op:    Oadd,
		Left:  Etempvar{ID: 1},
		Right: Econst{Const: Ointconst{Value: 1}},
	}
	if intAdd.Op != Oadd {
		t.Errorf("int add op = %v, want %v", intAdd.Op, Oadd)
	}

	// Float addition
	floatAdd := Ebinop{
		Op:    Oaddf,
		Left:  Etempvar{ID: 2},
		Right: Econst{Const: Ofloatconst{Value: 1.0}},
	}
	if floatAdd.Op != Oaddf {
		t.Errorf("float add op = %v, want %v", floatAdd.Op, Oaddf)
	}

	// Long addition
	longAdd := Ebinop{
		Op:    Oaddl,
		Left:  Etempvar{ID: 3},
		Right: Econst{Const: Olongconst{Value: 1}},
	}
	if longAdd.Op != Oaddl {
		t.Errorf("long add op = %v, want %v", longAdd.Op, Oaddl)
	}

	// Integer comparison
	intCmp := Ecmp{
		Op:    Ocmp,
		Cmp:   Clt,
		Left:  Etempvar{ID: 1},
		Right: Econst{Const: Ointconst{Value: 10}},
	}
	if intCmp.Op != Ocmp || intCmp.Cmp != Clt {
		t.Errorf("int cmp = %v/%v, want %v/%v", intCmp.Op, intCmp.Cmp, Ocmp, Clt)
	}
}

func TestMemoryOperations(t *testing.T) {
	// Test Eload and Sstore with chunks

	// Load int32 from address
	load := Eload{
		Chunk: Mint32,
		Addr:  Evar{Name: "x"},
	}
	if load.Chunk != Mint32 {
		t.Errorf("load chunk = %v, want %v", load.Chunk, Mint32)
	}

	// Store int32 to address
	store := Sstore{
		Chunk: Mint32,
		Addr:  Evar{Name: "x"},
		Value: Econst{Const: Ointconst{Value: 42}},
	}
	if store.Chunk != Mint32 {
		t.Errorf("store chunk = %v, want %v", store.Chunk, Mint32)
	}

	// Load float64 from address
	loadF := Eload{
		Chunk: Mfloat64,
		Addr:  Evar{Name: "y"},
	}
	if loadF.Chunk != Mfloat64 {
		t.Errorf("load float chunk = %v, want %v", loadF.Chunk, Mfloat64)
	}
}

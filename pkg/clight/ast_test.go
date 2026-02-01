package clight

import (
	"testing"

	"github.com/raymyers/ralph-cc/pkg/ctypes"
)

func TestExpressionTypes(t *testing.T) {
	tests := []struct {
		name string
		expr Expr
		want ctypes.Type
	}{
		{
			"Econst_int",
			Econst_int{Value: 42, Typ: ctypes.Int()},
			ctypes.Int(),
		},
		{
			"Econst_float",
			Econst_float{Value: 3.14, Typ: ctypes.Double()},
			ctypes.Double(),
		},
		{
			"Evar",
			Evar{Name: "x", Typ: ctypes.Int()},
			ctypes.Int(),
		},
		{
			"Etempvar",
			Etempvar{ID: 1, Typ: ctypes.Int()},
			ctypes.Int(),
		},
		{
			"Ederef",
			Ederef{Ptr: Evar{Name: "p", Typ: ctypes.Pointer(ctypes.Int())}, Typ: ctypes.Int()},
			ctypes.Int(),
		},
		{
			"Eaddrof",
			Eaddrof{Arg: Evar{Name: "x", Typ: ctypes.Int()}, Typ: ctypes.Pointer(ctypes.Int())},
			ctypes.Pointer(ctypes.Int()),
		},
		{
			"Eunop",
			Eunop{Op: Oneg, Arg: Econst_int{Value: 1, Typ: ctypes.Int()}, Typ: ctypes.Int()},
			ctypes.Int(),
		},
		{
			"Ebinop",
			Ebinop{Op: Oadd, Left: Econst_int{Value: 1, Typ: ctypes.Int()}, Right: Econst_int{Value: 2, Typ: ctypes.Int()}, Typ: ctypes.Int()},
			ctypes.Int(),
		},
		{
			"Ecast",
			Ecast{Arg: Econst_int{Value: 1, Typ: ctypes.Int()}, Typ: ctypes.Long()},
			ctypes.Long(),
		},
		{
			"Esizeof",
			Esizeof{ArgType: ctypes.Int(), Typ: ctypes.UInt()},
			ctypes.UInt(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.expr.ExprType()
			if !ctypes.Equal(got, tt.want) {
				t.Errorf("ExprType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSeqFlattensSkip(t *testing.T) {
	// Single non-skip statement
	s1 := Sreturn{Value: Econst_int{Value: 0, Typ: ctypes.Int()}}
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
	s2 := Sassign{LHS: Evar{Name: "x", Typ: ctypes.Int()}, RHS: Econst_int{Value: 1, Typ: ctypes.Int()}}
	result = Seq(s2, s1)
	seq, ok := result.(Ssequence)
	if !ok {
		t.Errorf("Seq(stmt, stmt) should return Ssequence")
	}
	if _, ok := seq.First.(Sassign); !ok {
		t.Errorf("First should be Sassign")
	}
	if _, ok := seq.Second.(Sreturn); !ok {
		t.Errorf("Second should be Sreturn")
	}
}

func TestUnaryOpString(t *testing.T) {
	tests := []struct {
		op   UnaryOp
		want string
	}{
		{Onotbool, "!"},
		{Onotint, "~"},
		{Oneg, "-"},
		{Oabsfloat, "abs"},
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
		{Oadd, "+"},
		{Osub, "-"},
		{Omul, "*"},
		{Odiv, "/"},
		{Omod, "%"},
		{Oeq, "=="},
		{One, "!="},
		{Olt, "<"},
		{Ogt, ">"},
		{Ole, "<="},
		{Oge, ">="},
		{Oand, "&"},
		{Oor, "|"},
		{Oxor, "^"},
		{Oshl, "<<"},
		{Oshr, ">>"},
	}
	for _, tt := range tests {
		if got := tt.op.String(); got != tt.want {
			t.Errorf("%d.String() = %q, want %q", tt.op, got, tt.want)
		}
	}
}

func TestASTConstruction(t *testing.T) {
	// Build a simple function: int f() { int x; x = 1; return x; }
	intTyp := ctypes.Int()
	
	fn := Function{
		Name:   "f",
		Return: intTyp,
		Params: nil,
		Locals: []VarDecl{{Name: "x", Type: intTyp}},
		Temps:  nil,
		Body: Seq(
			Sassign{
				LHS: Evar{Name: "x", Typ: intTyp},
				RHS: Econst_int{Value: 1, Typ: intTyp},
			},
			Sreturn{
				Value: Evar{Name: "x", Typ: intTyp},
			},
		),
	}

	if fn.Name != "f" {
		t.Errorf("function name = %q, want %q", fn.Name, "f")
	}
	if len(fn.Locals) != 1 {
		t.Errorf("locals count = %d, want 1", len(fn.Locals))
	}

	// Verify body structure
	seq, ok := fn.Body.(Ssequence)
	if !ok {
		t.Fatalf("body should be Ssequence, got %T", fn.Body)
	}
	if _, ok := seq.First.(Sassign); !ok {
		t.Errorf("first statement should be Sassign")
	}
	if _, ok := seq.Second.(Sreturn); !ok {
		t.Errorf("second statement should be Sreturn")
	}
}

func TestProgramConstruction(t *testing.T) {
	prog := Program{
		Structs: []ctypes.Tstruct{
			{Name: "Point", Fields: []ctypes.Field{
				{Name: "x", Type: ctypes.Int()},
				{Name: "y", Type: ctypes.Int()},
			}},
		},
		Globals: []VarDecl{
			{Name: "count", Type: ctypes.Int()},
		},
		Functions: []Function{
			{Name: "main", Return: ctypes.Int(), Body: Sreturn{Value: Econst_int{Value: 0, Typ: ctypes.Int()}}},
		},
	}

	if len(prog.Structs) != 1 {
		t.Errorf("structs count = %d, want 1", len(prog.Structs))
	}
	if len(prog.Globals) != 1 {
		t.Errorf("globals count = %d, want 1", len(prog.Globals))
	}
	if len(prog.Functions) != 1 {
		t.Errorf("functions count = %d, want 1", len(prog.Functions))
	}
}

func TestEfieldAccess(t *testing.T) {
	// Test field access: p->x where p is struct Point*
	pointType := ctypes.Tstruct{Name: "Point", Fields: []ctypes.Field{
		{Name: "x", Type: ctypes.Int()},
		{Name: "y", Type: ctypes.Int()},
	}}
	ptrType := ctypes.Pointer(pointType)
	
	// p->x is *(p).x in Clight, i.e., Efield(Ederef(p), "x")
	expr := Efield{
		Arg: Ederef{
			Ptr: Evar{Name: "p", Typ: ptrType},
			Typ: pointType,
		},
		FieldName: "x",
		Typ:       ctypes.Int(),
	}

	if expr.FieldName != "x" {
		t.Errorf("field name = %q, want %q", expr.FieldName, "x")
	}
	if !ctypes.Equal(expr.ExprType(), ctypes.Int()) {
		t.Errorf("field type = %v, want int", expr.ExprType())
	}
}

func TestLoopConstruction(t *testing.T) {
	// while (x > 0) { x--; } becomes:
	// Sloop(Sifthenelse(x > 0, x--, Sbreak), Sskip)
	intTyp := ctypes.Int()
	
	loop := Sloop{
		Body: Sifthenelse{
			Cond: Ebinop{
				Op:    Ogt,
				Left:  Evar{Name: "x", Typ: intTyp},
				Right: Econst_int{Value: 0, Typ: intTyp},
				Typ:   ctypes.Tint{Size: ctypes.IBool, Sign: ctypes.Signed},
			},
			Then: Sassign{
				LHS: Evar{Name: "x", Typ: intTyp},
				RHS: Ebinop{
					Op:    Osub,
					Left:  Evar{Name: "x", Typ: intTyp},
					Right: Econst_int{Value: 1, Typ: intTyp},
					Typ:   intTyp,
				},
			},
			Else: Sbreak{},
		},
		Continue: Sskip{},
	}

	// Verify loop structure
	ite, ok := loop.Body.(Sifthenelse)
	if !ok {
		t.Fatalf("loop body should be Sifthenelse, got %T", loop.Body)
	}
	if _, ok := ite.Cond.(Ebinop); !ok {
		t.Errorf("condition should be Ebinop")
	}
	if _, ok := ite.Else.(Sbreak); !ok {
		t.Errorf("else branch should be Sbreak")
	}
}

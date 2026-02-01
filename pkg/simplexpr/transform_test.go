package simplexpr

import (
	"testing"

	"github.com/raymyers/ralph-cc/pkg/cabs"
	"github.com/raymyers/ralph-cc/pkg/clight"
	"github.com/raymyers/ralph-cc/pkg/ctypes"
)

func TestHasSideEffects(t *testing.T) {
	tests := []struct {
		name     string
		expr     cabs.Expr
		expected bool
	}{
		{"constant", cabs.Constant{Value: 42}, false},
		{"variable", cabs.Variable{Name: "x"}, false},
		{"paren", cabs.Paren{Expr: cabs.Constant{Value: 1}}, false},
		{"negation", cabs.Unary{Op: cabs.OpNeg, Expr: cabs.Variable{Name: "x"}}, false},
		{"pre-increment", cabs.Unary{Op: cabs.OpPreInc, Expr: cabs.Variable{Name: "x"}}, true},
		{"post-increment", cabs.Unary{Op: cabs.OpPostInc, Expr: cabs.Variable{Name: "x"}}, true},
		{"pre-decrement", cabs.Unary{Op: cabs.OpPreDec, Expr: cabs.Variable{Name: "x"}}, true},
		{"post-decrement", cabs.Unary{Op: cabs.OpPostDec, Expr: cabs.Variable{Name: "x"}}, true},
		{"addition", cabs.Binary{Op: cabs.OpAdd, Left: cabs.Variable{Name: "x"}, Right: cabs.Constant{Value: 1}}, false},
		{"assignment", cabs.Binary{Op: cabs.OpAssign, Left: cabs.Variable{Name: "x"}, Right: cabs.Constant{Value: 1}}, true},
		{"add-assign", cabs.Binary{Op: cabs.OpAddAssign, Left: cabs.Variable{Name: "x"}, Right: cabs.Constant{Value: 1}}, true},
		{"function call", cabs.Call{Func: cabs.Variable{Name: "f"}, Args: nil}, true},
		{"comma", cabs.Binary{Op: cabs.OpComma, Left: cabs.Constant{Value: 1}, Right: cabs.Constant{Value: 2}}, true},
		{"nested side-effect", cabs.Binary{Op: cabs.OpAdd, Left: cabs.Unary{Op: cabs.OpPreInc, Expr: cabs.Variable{Name: "x"}}, Right: cabs.Constant{Value: 1}}, true},
		{"sizeof expr", cabs.SizeofExpr{Expr: cabs.Variable{Name: "x"}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasSideEffects(tt.expr)
			if got != tt.expected {
				t.Errorf("HasSideEffects() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTransformExpr_Constant(t *testing.T) {
	tr := New()
	result := tr.TransformExpr(cabs.Constant{Value: 42})

	if len(result.Stmts) != 0 {
		t.Errorf("expected no statements, got %d", len(result.Stmts))
	}

	constExpr, ok := result.Expr.(clight.Econst_int)
	if !ok {
		t.Fatalf("expected Econst_int, got %T", result.Expr)
	}
	if constExpr.Value != 42 {
		t.Errorf("expected value 42, got %d", constExpr.Value)
	}
}

func TestTransformExpr_Variable(t *testing.T) {
	tr := New()
	tr.SetType("x", ctypes.Int())
	result := tr.TransformExpr(cabs.Variable{Name: "x"})

	if len(result.Stmts) != 0 {
		t.Errorf("expected no statements, got %d", len(result.Stmts))
	}

	varExpr, ok := result.Expr.(clight.Evar)
	if !ok {
		t.Fatalf("expected Evar, got %T", result.Expr)
	}
	if varExpr.Name != "x" {
		t.Errorf("expected name 'x', got %s", varExpr.Name)
	}
}

func TestTransformExpr_BinaryOp(t *testing.T) {
	tr := New()
	tr.SetType("x", ctypes.Int())
	tr.SetType("y", ctypes.Int())

	// x + y
	result := tr.TransformExpr(cabs.Binary{
		Op:    cabs.OpAdd,
		Left:  cabs.Variable{Name: "x"},
		Right: cabs.Variable{Name: "y"},
	})

	if len(result.Stmts) != 0 {
		t.Errorf("expected no statements for pure binary, got %d", len(result.Stmts))
	}

	binExpr, ok := result.Expr.(clight.Ebinop)
	if !ok {
		t.Fatalf("expected Ebinop, got %T", result.Expr)
	}
	if binExpr.Op != clight.Oadd {
		t.Errorf("expected Oadd, got %v", binExpr.Op)
	}
}

func TestTransformExpr_Assignment(t *testing.T) {
	tr := New()
	tr.SetType("x", ctypes.Int())

	// x = 5
	result := tr.TransformExpr(cabs.Binary{
		Op:    cabs.OpAssign,
		Left:  cabs.Variable{Name: "x"},
		Right: cabs.Constant{Value: 5},
	})

	// Assignment should produce statements
	if len(result.Stmts) == 0 {
		t.Error("expected statements for assignment")
	}

	// Result should be a temporary holding the assigned value
	tempExpr, ok := result.Expr.(clight.Etempvar)
	if !ok {
		t.Fatalf("expected Etempvar, got %T", result.Expr)
	}

	// Check that we have a Sset and Sassign
	hasSet := false
	hasAssign := false
	for _, stmt := range result.Stmts {
		switch stmt.(type) {
		case clight.Sset:
			hasSet = true
		case clight.Sassign:
			hasAssign = true
		}
	}
	if !hasSet || !hasAssign {
		t.Errorf("expected Sset and Sassign, got %v", result.Stmts)
	}
	_ = tempExpr // avoid unused warning
}

func TestTransformExpr_PreIncrement(t *testing.T) {
	tr := New()
	tr.SetType("x", ctypes.Int())

	// ++x
	result := tr.TransformExpr(cabs.Unary{
		Op:   cabs.OpPreInc,
		Expr: cabs.Variable{Name: "x"},
	})

	// Should produce statements
	if len(result.Stmts) < 2 {
		t.Errorf("expected at least 2 statements for ++x, got %d", len(result.Stmts))
	}

	// Result should be a temporary
	_, ok := result.Expr.(clight.Etempvar)
	if !ok {
		t.Fatalf("expected Etempvar, got %T", result.Expr)
	}
}

func TestTransformExpr_PostIncrement(t *testing.T) {
	tr := New()
	tr.SetType("x", ctypes.Int())

	// x++
	result := tr.TransformExpr(cabs.Unary{
		Op:   cabs.OpPostInc,
		Expr: cabs.Variable{Name: "x"},
	})

	// Should produce statements
	if len(result.Stmts) < 2 {
		t.Errorf("expected at least 2 statements for x++, got %d", len(result.Stmts))
	}

	// Result should be a temporary holding the OLD value
	_, ok := result.Expr.(clight.Etempvar)
	if !ok {
		t.Fatalf("expected Etempvar, got %T", result.Expr)
	}
}

func TestTransformExpr_CompoundAssign(t *testing.T) {
	tr := New()
	tr.SetType("x", ctypes.Int())

	// x += 10
	result := tr.TransformExpr(cabs.Binary{
		Op:    cabs.OpAddAssign,
		Left:  cabs.Variable{Name: "x"},
		Right: cabs.Constant{Value: 10},
	})

	if len(result.Stmts) < 2 {
		t.Errorf("expected at least 2 statements for x += 10, got %d", len(result.Stmts))
	}

	_, ok := result.Expr.(clight.Etempvar)
	if !ok {
		t.Fatalf("expected Etempvar, got %T", result.Expr)
	}
}

func TestTransformExpr_FunctionCall(t *testing.T) {
	tr := New()

	// f(1, 2)
	result := tr.TransformExpr(cabs.Call{
		Func: cabs.Variable{Name: "f"},
		Args: []cabs.Expr{
			cabs.Constant{Value: 1},
			cabs.Constant{Value: 2},
		},
	})

	// Should have a Scall statement
	hasCall := false
	for _, stmt := range result.Stmts {
		if _, ok := stmt.(clight.Scall); ok {
			hasCall = true
			break
		}
	}
	if !hasCall {
		t.Error("expected Scall statement for function call")
	}

	// Result should be a temporary
	_, ok := result.Expr.(clight.Etempvar)
	if !ok {
		t.Fatalf("expected Etempvar for function call result, got %T", result.Expr)
	}
}

func TestTransformExpr_Comma(t *testing.T) {
	tr := New()
	tr.SetType("x", ctypes.Int())

	// (x, 5) - result should be 5
	result := tr.TransformExpr(cabs.Binary{
		Op:    cabs.OpComma,
		Left:  cabs.Variable{Name: "x"},
		Right: cabs.Constant{Value: 5},
	})

	// Result should be the right expression
	constExpr, ok := result.Expr.(clight.Econst_int)
	if !ok {
		t.Fatalf("expected Econst_int, got %T", result.Expr)
	}
	if constExpr.Value != 5 {
		t.Errorf("expected 5, got %d", constExpr.Value)
	}
}

func TestTransformExpr_Conditional(t *testing.T) {
	tr := New()
	tr.SetType("x", ctypes.Int())

	// x ? 1 : 2
	result := tr.TransformExpr(cabs.Conditional{
		Cond: cabs.Variable{Name: "x"},
		Then: cabs.Constant{Value: 1},
		Else: cabs.Constant{Value: 2},
	})

	// Should have an if-then-else statement
	hasIf := false
	for _, stmt := range result.Stmts {
		if _, ok := stmt.(clight.Sifthenelse); ok {
			hasIf = true
			break
		}
	}
	if !hasIf {
		t.Error("expected Sifthenelse for ternary operator")
	}

	// Result should be a temporary
	_, ok := result.Expr.(clight.Etempvar)
	if !ok {
		t.Fatalf("expected Etempvar, got %T", result.Expr)
	}
}

func TestTransformExpr_NestedSideEffects(t *testing.T) {
	tr := New()
	tr.SetType("x", ctypes.Int())

	// (++x) + 1
	result := tr.TransformExpr(cabs.Binary{
		Op: cabs.OpAdd,
		Left: cabs.Unary{
			Op:   cabs.OpPreInc,
			Expr: cabs.Variable{Name: "x"},
		},
		Right: cabs.Constant{Value: 1},
	})

	// The ++x side-effect should be extracted
	if len(result.Stmts) == 0 {
		t.Error("expected statements for nested side-effect")
	}

	// Result should be a binary expression
	_, ok := result.Expr.(clight.Ebinop)
	if !ok {
		t.Fatalf("expected Ebinop, got %T", result.Expr)
	}
}

func TestTempTypes(t *testing.T) {
	tr := New()
	tr.SetType("x", ctypes.Int())
	tr.SetType("y", ctypes.Long())

	// x = 5 (creates temp)
	tr.TransformExpr(cabs.Binary{
		Op:    cabs.OpAssign,
		Left:  cabs.Variable{Name: "x"},
		Right: cabs.Constant{Value: 5},
	})

	// ++x (creates temp)
	tr.TransformExpr(cabs.Unary{
		Op:   cabs.OpPreInc,
		Expr: cabs.Variable{Name: "x"},
	})

	temps := tr.TempTypes()
	if len(temps) < 2 {
		t.Errorf("expected at least 2 temps, got %d", len(temps))
	}
}

func TestTransformExpr_UnaryOps(t *testing.T) {
	tests := []struct {
		name    string
		op      cabs.UnaryOp
		clightOp clight.UnaryOp
	}{
		{"negation", cabs.OpNeg, clight.Oneg},
		{"not", cabs.OpNot, clight.Onotbool},
		{"bitnot", cabs.OpBitNot, clight.Onotint},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := New()
			tr.SetType("x", ctypes.Int())

			result := tr.TransformExpr(cabs.Unary{
				Op:   tt.op,
				Expr: cabs.Variable{Name: "x"},
			})

			if len(result.Stmts) != 0 {
				t.Errorf("expected no statements, got %d", len(result.Stmts))
			}

			unop, ok := result.Expr.(clight.Eunop)
			if !ok {
				t.Fatalf("expected Eunop, got %T", result.Expr)
			}
			if unop.Op != tt.clightOp {
				t.Errorf("expected %v, got %v", tt.clightOp, unop.Op)
			}
		})
	}
}

func TestTransformExpr_AddressOf(t *testing.T) {
	tr := New()
	tr.SetType("x", ctypes.Int())

	// &x
	result := tr.TransformExpr(cabs.Unary{
		Op:   cabs.OpAddrOf,
		Expr: cabs.Variable{Name: "x"},
	})

	addr, ok := result.Expr.(clight.Eaddrof)
	if !ok {
		t.Fatalf("expected Eaddrof, got %T", result.Expr)
	}

	// Type should be pointer to int
	ptrTyp, ok := addr.Typ.(ctypes.Tpointer)
	if !ok {
		t.Fatalf("expected Tpointer, got %T", addr.Typ)
	}
	if !ctypes.Equal(ptrTyp.Elem, ctypes.Int()) {
		t.Errorf("expected pointer to int, got pointer to %v", ptrTyp.Elem)
	}
}

func TestTransformExpr_Deref(t *testing.T) {
	tr := New()
	tr.SetType("p", ctypes.Pointer(ctypes.Int()))

	// *p
	result := tr.TransformExpr(cabs.Unary{
		Op:   cabs.OpDeref,
		Expr: cabs.Variable{Name: "p"},
	})

	deref, ok := result.Expr.(clight.Ederef)
	if !ok {
		t.Fatalf("expected Ederef, got %T", result.Expr)
	}

	// Type should be int (dereferenced)
	if !ctypes.Equal(deref.Typ, ctypes.Int()) {
		t.Errorf("expected int type, got %v", deref.Typ)
	}
}

func TestReset(t *testing.T) {
	tr := New()

	// Generate some temps
	tr.TransformExpr(cabs.Unary{
		Op:   cabs.OpPreInc,
		Expr: cabs.Variable{Name: "x"},
	})

	if len(tr.TempTypes()) == 0 {
		t.Fatal("expected temps to be generated")
	}

	// Reset
	tr.Reset()

	if len(tr.TempTypes()) != 0 {
		t.Error("expected temps to be cleared after reset")
	}
}

package simpllocals

import (
	"testing"

	"github.com/raymyers/ralph-cc/pkg/cabs"
	"github.com/raymyers/ralph-cc/pkg/clight"
	"github.com/raymyers/ralph-cc/pkg/ctypes"
)

func TestIsScalarType(t *testing.T) {
	tests := []struct {
		name     string
		typ      ctypes.Type
		expected bool
	}{
		{"int", ctypes.Int(), true},
		{"unsigned int", ctypes.UInt(), true},
		{"char", ctypes.Char(), true},
		{"long", ctypes.Long(), true},
		{"float", ctypes.Float(), true},
		{"double", ctypes.Double(), true},
		{"pointer", ctypes.Pointer(ctypes.Int()), true},
		{"void", ctypes.Void(), false},
		{"array", ctypes.Array(ctypes.Int(), 10), false},
		{"struct", ctypes.Tstruct{Name: "Point"}, false},
		{"union", ctypes.Tunion{Name: "Data"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsScalarType(tt.typ)
			if got != tt.expected {
				t.Errorf("IsScalarType(%v) = %v, want %v", tt.typ, got, tt.expected)
			}
		})
	}
}

func TestAnalyzeAddressTaken(t *testing.T) {
	tr := New()

	// &x - x is address-taken
	tr.AnalyzeAddressTaken(cabs.Unary{
		Op:   cabs.OpAddrOf,
		Expr: cabs.Variable{Name: "x"},
	})

	if !tr.IsAddressTaken("x") {
		t.Error("expected x to be address-taken")
	}
	if tr.IsAddressTaken("y") {
		t.Error("expected y to not be address-taken")
	}
}

func TestAnalyzeAddressTakenNested(t *testing.T) {
	tr := New()

	// f(&x) - x is address-taken even when nested in a call
	tr.AnalyzeAddressTaken(cabs.Call{
		Func: cabs.Variable{Name: "f"},
		Args: []cabs.Expr{
			cabs.Unary{Op: cabs.OpAddrOf, Expr: cabs.Variable{Name: "x"}},
		},
	})

	if !tr.IsAddressTaken("x") {
		t.Error("expected x to be address-taken")
	}
}

func TestAnalyzeAddressTakenBinary(t *testing.T) {
	tr := New()

	// &x + 1 - x is address-taken
	tr.AnalyzeAddressTaken(cabs.Binary{
		Op:   cabs.OpAdd,
		Left: cabs.Unary{Op: cabs.OpAddrOf, Expr: cabs.Variable{Name: "x"}},
		Right: cabs.Constant{Value: 1},
	})

	if !tr.IsAddressTaken("x") {
		t.Error("expected x to be address-taken")
	}
}

func TestCanPromoteToTemp(t *testing.T) {
	tr := New()
	tr.addressTaken["taken"] = true

	tests := []struct {
		name     string
		varName  string
		typ      ctypes.Type
		expected bool
	}{
		{"int not taken", "x", ctypes.Int(), true},
		{"int taken", "taken", ctypes.Int(), false},
		{"pointer not taken", "p", ctypes.Pointer(ctypes.Int()), true},
		{"array not taken", "arr", ctypes.Array(ctypes.Int(), 10), false},
		{"struct not taken", "s", ctypes.Tstruct{Name: "S"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tr.CanPromoteToTemp(tt.varName, tt.typ)
			if got != tt.expected {
				t.Errorf("CanPromoteToTemp(%s, %v) = %v, want %v", tt.varName, tt.typ, got, tt.expected)
			}
		})
	}
}

func TestPromoteLocal(t *testing.T) {
	tr := New()

	// Promote x
	id1 := tr.PromoteLocal("x", ctypes.Int())
	if id1 < 1 {
		t.Errorf("expected valid temp ID, got %d", id1)
	}

	// Promote y - should get different ID
	id2 := tr.PromoteLocal("y", ctypes.Long())
	if id2 <= id1 {
		t.Errorf("expected id2 > id1, got id2=%d, id1=%d", id2, id1)
	}

	// Promote x again - should return same ID
	id3 := tr.PromoteLocal("x", ctypes.Int())
	if id3 != id1 {
		t.Errorf("expected same ID for x, got %d vs %d", id3, id1)
	}

	// Check temps
	temps := tr.TempTypes()
	if len(temps) != 2 {
		t.Errorf("expected 2 temps, got %d", len(temps))
	}
}

func TestPromoteLocalAddressTaken(t *testing.T) {
	tr := New()
	tr.addressTaken["x"] = true

	id := tr.PromoteLocal("x", ctypes.Int())
	if id != -1 {
		t.Errorf("expected -1 for address-taken var, got %d", id)
	}
}

func TestPromoteLocalNonScalar(t *testing.T) {
	tr := New()

	id := tr.PromoteLocal("arr", ctypes.Array(ctypes.Int(), 10))
	if id != -1 {
		t.Errorf("expected -1 for array, got %d", id)
	}
}

func TestTransformExpr_Evar(t *testing.T) {
	tr := New()
	tr.PromoteLocal("x", ctypes.Int())

	// Transform Evar for promoted local
	expr := clight.Evar{Name: "x", Typ: ctypes.Int()}
	result := tr.TransformExpr(expr)

	tv, ok := result.(clight.Etempvar)
	if !ok {
		t.Fatalf("expected Etempvar, got %T", result)
	}
	if tv.ID != 1 {
		t.Errorf("expected temp ID 1, got %d", tv.ID)
	}
}

func TestTransformExpr_EvarNotPromoted(t *testing.T) {
	tr := New()
	tr.addressTaken["x"] = true

	// Transform Evar for non-promoted local
	expr := clight.Evar{Name: "x", Typ: ctypes.Int()}
	result := tr.TransformExpr(expr)

	_, ok := result.(clight.Evar)
	if !ok {
		t.Fatalf("expected Evar (unchanged), got %T", result)
	}
}

func TestTransformExpr_Ebinop(t *testing.T) {
	tr := New()
	tr.PromoteLocal("x", ctypes.Int())

	// x + 1
	expr := clight.Ebinop{
		Op:    clight.Oadd,
		Left:  clight.Evar{Name: "x", Typ: ctypes.Int()},
		Right: clight.Econst_int{Value: 1, Typ: ctypes.Int()},
		Typ:   ctypes.Int(),
	}
	result := tr.TransformExpr(expr)

	binop, ok := result.(clight.Ebinop)
	if !ok {
		t.Fatalf("expected Ebinop, got %T", result)
	}

	// Left should be Etempvar now
	_, ok = binop.Left.(clight.Etempvar)
	if !ok {
		t.Fatalf("expected left to be Etempvar, got %T", binop.Left)
	}
}

func TestTransformStmt_Sassign(t *testing.T) {
	tr := New()
	tr.PromoteLocal("x", ctypes.Int())

	// x = 5 should become Sset when x is promoted
	stmt := clight.Sassign{
		LHS: clight.Evar{Name: "x", Typ: ctypes.Int()},
		RHS: clight.Econst_int{Value: 5, Typ: ctypes.Int()},
	}
	result := tr.TransformStmt(stmt)

	sset, ok := result.(clight.Sset)
	if !ok {
		t.Fatalf("expected Sset, got %T", result)
	}
	if sset.TempID != 1 {
		t.Errorf("expected temp ID 1, got %d", sset.TempID)
	}
}

func TestTransformStmt_SassignNotPromoted(t *testing.T) {
	tr := New()
	tr.addressTaken["x"] = true

	// x = 5 should stay Sassign when x is not promoted
	stmt := clight.Sassign{
		LHS: clight.Evar{Name: "x", Typ: ctypes.Int()},
		RHS: clight.Econst_int{Value: 5, Typ: ctypes.Int()},
	}
	result := tr.TransformStmt(stmt)

	_, ok := result.(clight.Sassign)
	if !ok {
		t.Fatalf("expected Sassign, got %T", result)
	}
}

func TestTransformStmt_Sreturn(t *testing.T) {
	tr := New()
	tr.PromoteLocal("x", ctypes.Int())

	stmt := clight.Sreturn{
		Value: clight.Evar{Name: "x", Typ: ctypes.Int()},
	}
	result := tr.TransformStmt(stmt)

	ret, ok := result.(clight.Sreturn)
	if !ok {
		t.Fatalf("expected Sreturn, got %T", result)
	}

	_, ok = ret.Value.(clight.Etempvar)
	if !ok {
		t.Fatalf("expected return value to be Etempvar, got %T", ret.Value)
	}
}

func TestTransformStmt_Scall(t *testing.T) {
	tr := New()
	tr.PromoteLocal("x", ctypes.Int())

	// f(x) - x arg should be transformed
	stmt := clight.Scall{
		Func: clight.Evar{Name: "f", Typ: ctypes.Int()},
		Args: []clight.Expr{
			clight.Evar{Name: "x", Typ: ctypes.Int()},
		},
	}
	result := tr.TransformStmt(stmt)

	call, ok := result.(clight.Scall)
	if !ok {
		t.Fatalf("expected Scall, got %T", result)
	}

	if len(call.Args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(call.Args))
	}

	_, ok = call.Args[0].(clight.Etempvar)
	if !ok {
		t.Fatalf("expected arg to be Etempvar, got %T", call.Args[0])
	}
}

func TestAnalyzeLocals(t *testing.T) {
	tr := New()
	tr.addressTaken["y"] = true

	decls := []clight.VarDecl{
		{Name: "x", Type: ctypes.Int()},
		{Name: "y", Type: ctypes.Int()},
		{Name: "arr", Type: ctypes.Array(ctypes.Int(), 10)},
	}

	infos := tr.AnalyzeLocals(decls)

	if len(infos) != 3 {
		t.Fatalf("expected 3 infos, got %d", len(infos))
	}

	// x should be promoted
	if !infos[0].Promoted {
		t.Error("expected x to be promoted")
	}
	if infos[0].TempID < 1 {
		t.Error("expected x to have valid temp ID")
	}

	// y should not be promoted (address taken)
	if infos[1].Promoted {
		t.Error("expected y to not be promoted (address taken)")
	}

	// arr should not be promoted (not scalar)
	if infos[2].Promoted {
		t.Error("expected arr to not be promoted (not scalar)")
	}
}

func TestFilterUnpromotedLocals(t *testing.T) {
	infos := []LocalInfo{
		{Name: "x", Type: ctypes.Int(), Promoted: true, TempID: 1},
		{Name: "y", Type: ctypes.Int(), Promoted: false, TempID: -1},
		{Name: "arr", Type: ctypes.Array(ctypes.Int(), 10), Promoted: false, TempID: -1},
	}

	result := FilterUnpromotedLocals(infos)

	if len(result) != 2 {
		t.Fatalf("expected 2 unpromoted locals, got %d", len(result))
	}

	if result[0].Name != "y" || result[1].Name != "arr" {
		t.Errorf("unexpected locals: %v", result)
	}
}

func TestAnalyzeStmtReturn(t *testing.T) {
	tr := New()

	tr.AnalyzeStmt(cabs.Return{
		Expr: cabs.Unary{Op: cabs.OpAddrOf, Expr: cabs.Variable{Name: "x"}},
	})

	if !tr.IsAddressTaken("x") {
		t.Error("expected x to be address-taken from return statement")
	}
}

func TestAnalyzeStmtIf(t *testing.T) {
	tr := New()

	tr.AnalyzeStmt(cabs.If{
		Cond: cabs.Variable{Name: "cond"},
		Then: cabs.Computation{
			Expr: cabs.Unary{Op: cabs.OpAddrOf, Expr: cabs.Variable{Name: "x"}},
		},
		Else: cabs.Computation{
			Expr: cabs.Unary{Op: cabs.OpAddrOf, Expr: cabs.Variable{Name: "y"}},
		},
	})

	if !tr.IsAddressTaken("x") {
		t.Error("expected x to be address-taken from then branch")
	}
	if !tr.IsAddressTaken("y") {
		t.Error("expected y to be address-taken from else branch")
	}
}

func TestAnalyzeStmtWhile(t *testing.T) {
	tr := New()

	tr.AnalyzeStmt(cabs.While{
		Cond: cabs.Variable{Name: "cond"},
		Body: cabs.Computation{
			Expr: cabs.Unary{Op: cabs.OpAddrOf, Expr: cabs.Variable{Name: "x"}},
		},
	})

	if !tr.IsAddressTaken("x") {
		t.Error("expected x to be address-taken from while body")
	}
}

func TestAnalyzeStmtFor(t *testing.T) {
	tr := New()

	tr.AnalyzeStmt(cabs.For{
		Init: cabs.Unary{Op: cabs.OpAddrOf, Expr: cabs.Variable{Name: "a"}},
		Cond: cabs.Unary{Op: cabs.OpAddrOf, Expr: cabs.Variable{Name: "b"}},
		Step: cabs.Unary{Op: cabs.OpAddrOf, Expr: cabs.Variable{Name: "c"}},
		Body: cabs.Computation{
			Expr: cabs.Unary{Op: cabs.OpAddrOf, Expr: cabs.Variable{Name: "d"}},
		},
	})

	for _, name := range []string{"a", "b", "c", "d"} {
		if !tr.IsAddressTaken(name) {
			t.Errorf("expected %s to be address-taken from for loop", name)
		}
	}
}

func TestReset(t *testing.T) {
	tr := New()

	// Set up some state
	tr.addressTaken["x"] = true
	tr.PromoteLocal("y", ctypes.Int())

	if len(tr.TempTypes()) == 0 {
		t.Fatal("expected temps before reset")
	}

	tr.Reset()

	if len(tr.TempTypes()) != 0 {
		t.Error("expected no temps after reset")
	}
	if tr.IsAddressTaken("x") {
		t.Error("expected address-taken to be cleared after reset")
	}
}

func TestSetNextTempID(t *testing.T) {
	tr := New()
	tr.SetNextTempID(100)

	id := tr.PromoteLocal("x", ctypes.Int())
	if id != 100 {
		t.Errorf("expected temp ID 100, got %d", id)
	}
}

package cshmgen

import (
	"testing"

	"github.com/raymyers/ralph-cc/pkg/clight"
	"github.com/raymyers/ralph-cc/pkg/csharpminor"
	"github.com/raymyers/ralph-cc/pkg/ctypes"
)

func TestTranslateConstInt(t *testing.T) {
	tr := NewExprTranslator(nil)
	expr := clight.Econst_int{Value: 42, Typ: ctypes.Int()}
	result := tr.TranslateExpr(expr)

	econst, ok := result.(csharpminor.Econst)
	if !ok {
		t.Fatalf("expected Econst, got %T", result)
	}
	intConst, ok := econst.Const.(csharpminor.Ointconst)
	if !ok {
		t.Fatalf("expected Ointconst, got %T", econst.Const)
	}
	if intConst.Value != 42 {
		t.Errorf("expected 42, got %d", intConst.Value)
	}
}

func TestTranslateConstLong(t *testing.T) {
	tr := NewExprTranslator(nil)
	expr := clight.Econst_long{Value: 1234567890123, Typ: ctypes.Long()}
	result := tr.TranslateExpr(expr)

	econst, ok := result.(csharpminor.Econst)
	if !ok {
		t.Fatalf("expected Econst, got %T", result)
	}
	longConst, ok := econst.Const.(csharpminor.Olongconst)
	if !ok {
		t.Fatalf("expected Olongconst, got %T", econst.Const)
	}
	if longConst.Value != 1234567890123 {
		t.Errorf("expected 1234567890123, got %d", longConst.Value)
	}
}

func TestTranslateConstFloat(t *testing.T) {
	tr := NewExprTranslator(nil)
	expr := clight.Econst_float{Value: 3.14, Typ: ctypes.Double()}
	result := tr.TranslateExpr(expr)

	econst, ok := result.(csharpminor.Econst)
	if !ok {
		t.Fatalf("expected Econst, got %T", result)
	}
	floatConst, ok := econst.Const.(csharpminor.Ofloatconst)
	if !ok {
		t.Fatalf("expected Ofloatconst, got %T", econst.Const)
	}
	if floatConst.Value != 3.14 {
		t.Errorf("expected 3.14, got %f", floatConst.Value)
	}
}

func TestTranslateConstSingle(t *testing.T) {
	tr := NewExprTranslator(nil)
	expr := clight.Econst_single{Value: 2.5, Typ: ctypes.Float()}
	result := tr.TranslateExpr(expr)

	econst, ok := result.(csharpminor.Econst)
	if !ok {
		t.Fatalf("expected Econst, got %T", result)
	}
	singleConst, ok := econst.Const.(csharpminor.Osingleconst)
	if !ok {
		t.Fatalf("expected Osingleconst, got %T", econst.Const)
	}
	if singleConst.Value != 2.5 {
		t.Errorf("expected 2.5, got %f", singleConst.Value)
	}
}

func TestTranslateVar(t *testing.T) {
	tr := NewExprTranslator(nil)
	expr := clight.Evar{Name: "global_x", Typ: ctypes.Int()}
	result := tr.TranslateExpr(expr)

	evar, ok := result.(csharpminor.Evar)
	if !ok {
		t.Fatalf("expected Evar, got %T", result)
	}
	if evar.Name != "global_x" {
		t.Errorf("expected global_x, got %s", evar.Name)
	}
}

func TestTranslateTempvar(t *testing.T) {
	tr := NewExprTranslator(nil)
	expr := clight.Etempvar{ID: 5, Typ: ctypes.Int()}
	result := tr.TranslateExpr(expr)

	tempvar, ok := result.(csharpminor.Etempvar)
	if !ok {
		t.Fatalf("expected Etempvar, got %T", result)
	}
	if tempvar.ID != 5 {
		t.Errorf("expected ID 5, got %d", tempvar.ID)
	}
}

func TestTranslateUnop(t *testing.T) {
	tests := []struct {
		name     string
		op       clight.UnaryOp
		argType  ctypes.Type
		wantOp   csharpminor.UnaryOp
	}{
		{"neg int", clight.Oneg, ctypes.Int(), csharpminor.Onegint},
		{"neg long", clight.Oneg, ctypes.Long(), csharpminor.Onegl},
		{"neg float", clight.Oneg, ctypes.Double(), csharpminor.Onegf},
		{"neg single", clight.Oneg, ctypes.Float(), csharpminor.Onegs},
		{"not int", clight.Onotint, ctypes.Int(), csharpminor.Onotint},
		{"not long", clight.Onotint, ctypes.Long(), csharpminor.Onotl},
		{"notbool", clight.Onotbool, ctypes.Int(), csharpminor.Onotbool},
	}

	tr := NewExprTranslator(nil)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			arg := clight.Econst_int{Value: 1, Typ: tc.argType}
			expr := clight.Eunop{Op: tc.op, Arg: arg, Typ: tc.argType}
			result := tr.TranslateExpr(expr)

			eunop, ok := result.(csharpminor.Eunop)
			if !ok {
				t.Fatalf("expected Eunop, got %T", result)
			}
			if eunop.Op != tc.wantOp {
				t.Errorf("expected op %v, got %v", tc.wantOp, eunop.Op)
			}
		})
	}
}

func TestTranslateBinop(t *testing.T) {
	tests := []struct {
		name   string
		op     clight.BinaryOp
		typ    ctypes.Type
		wantOp csharpminor.BinaryOp
	}{
		{"add int", clight.Oadd, ctypes.Int(), csharpminor.Oadd},
		{"add long", clight.Oadd, ctypes.Long(), csharpminor.Oaddl},
		{"add float", clight.Oadd, ctypes.Double(), csharpminor.Oaddf},
		{"sub int", clight.Osub, ctypes.Int(), csharpminor.Osub},
		{"mul int", clight.Omul, ctypes.Int(), csharpminor.Omul},
		{"div int signed", clight.Odiv, ctypes.Int(), csharpminor.Odiv},
		{"div int unsigned", clight.Odiv, ctypes.UInt(), csharpminor.Odivu},
		{"and int", clight.Oand, ctypes.Int(), csharpminor.Oand},
		{"or int", clight.Oor, ctypes.Int(), csharpminor.Oor},
		{"xor int", clight.Oxor, ctypes.Int(), csharpminor.Oxor},
		{"shl int", clight.Oshl, ctypes.Int(), csharpminor.Oshl},
		{"shr int signed", clight.Oshr, ctypes.Int(), csharpminor.Oshr},
		{"shr int unsigned", clight.Oshr, ctypes.UInt(), csharpminor.Oshru},
	}

	tr := NewExprTranslator(nil)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			left := clight.Econst_int{Value: 1, Typ: tc.typ}
			right := clight.Econst_int{Value: 2, Typ: tc.typ}
			expr := clight.Ebinop{Op: tc.op, Left: left, Right: right, Typ: tc.typ}
			result := tr.TranslateExpr(expr)

			ebinop, ok := result.(csharpminor.Ebinop)
			if !ok {
				t.Fatalf("expected Ebinop, got %T", result)
			}
			if ebinop.Op != tc.wantOp {
				t.Errorf("expected op %v, got %v", tc.wantOp, ebinop.Op)
			}
		})
	}
}

func TestTranslateComparison(t *testing.T) {
	tests := []struct {
		name    string
		op      clight.BinaryOp
		typ     ctypes.Type
		wantOp  csharpminor.BinaryOp
		wantCmp csharpminor.Comparison
	}{
		{"eq int", clight.Oeq, ctypes.Int(), csharpminor.Ocmp, csharpminor.Ceq},
		{"ne int", clight.One, ctypes.Int(), csharpminor.Ocmp, csharpminor.Cne},
		{"lt int", clight.Olt, ctypes.Int(), csharpminor.Ocmp, csharpminor.Clt},
		{"le int", clight.Ole, ctypes.Int(), csharpminor.Ocmp, csharpminor.Cle},
		{"gt int", clight.Ogt, ctypes.Int(), csharpminor.Ocmp, csharpminor.Cgt},
		{"ge int", clight.Oge, ctypes.Int(), csharpminor.Ocmp, csharpminor.Cge},
		{"eq uint", clight.Oeq, ctypes.UInt(), csharpminor.Ocmpu, csharpminor.Ceq},
		{"lt long", clight.Olt, ctypes.Long(), csharpminor.Ocmpl, csharpminor.Clt},
		{"lt float", clight.Olt, ctypes.Double(), csharpminor.Ocmpf, csharpminor.Clt},
	}

	tr := NewExprTranslator(nil)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			left := clight.Econst_int{Value: 1, Typ: tc.typ}
			right := clight.Econst_int{Value: 2, Typ: tc.typ}
			// Comparison result type is int
			expr := clight.Ebinop{Op: tc.op, Left: left, Right: right, Typ: ctypes.Int()}
			result := tr.TranslateExpr(expr)

			ecmp, ok := result.(csharpminor.Ecmp)
			if !ok {
				t.Fatalf("expected Ecmp, got %T", result)
			}
			if ecmp.Op != tc.wantOp {
				t.Errorf("expected op %v, got %v", tc.wantOp, ecmp.Op)
			}
			if ecmp.Cmp != tc.wantCmp {
				t.Errorf("expected cmp %v, got %v", tc.wantCmp, ecmp.Cmp)
			}
		})
	}
}

func TestTranslateCast(t *testing.T) {
	tests := []struct {
		name     string
		from     ctypes.Type
		to       ctypes.Type
		wantOp   csharpminor.UnaryOp
		needCast bool
	}{
		{"same type", ctypes.Int(), ctypes.Int(), 0, false},
		{"int to char", ctypes.Int(), ctypes.Char(), csharpminor.Ocast8signed, true},
		{"int to uchar", ctypes.Int(), ctypes.UChar(), csharpminor.Ocast8unsigned, true},
		{"int to long", ctypes.Int(), ctypes.Long(), csharpminor.Olongofint, true},
		{"long to int", ctypes.Long(), ctypes.Int(), csharpminor.Ointoflong, true},
		{"float to double", ctypes.Float(), ctypes.Double(), csharpminor.Ofloatofsingle, true},
		{"double to float", ctypes.Double(), ctypes.Float(), csharpminor.Osingleoffloat, true},
		{"int to double", ctypes.Int(), ctypes.Double(), csharpminor.Ofloatofint, true},
		{"double to int", ctypes.Double(), ctypes.Int(), csharpminor.Ointoffloat, true},
	}

	tr := NewExprTranslator(nil)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			arg := clight.Econst_int{Value: 1, Typ: tc.from}
			expr := clight.Ecast{Arg: arg, Typ: tc.to}
			result := tr.TranslateExpr(expr)

			if !tc.needCast {
				// Should return the arg unchanged
				_, ok := result.(csharpminor.Econst)
				if !ok {
					t.Fatalf("expected no cast, got %T", result)
				}
				return
			}

			eunop, ok := result.(csharpminor.Eunop)
			if !ok {
				t.Fatalf("expected Eunop for cast, got %T", result)
			}
			if eunop.Op != tc.wantOp {
				t.Errorf("expected op %v, got %v", tc.wantOp, eunop.Op)
			}
		})
	}
}

func TestTranslateDeref(t *testing.T) {
	tr := NewExprTranslator(nil)
	// *p where p is int*
	ptr := clight.Etempvar{ID: 1, Typ: ctypes.Pointer(ctypes.Int())}
	expr := clight.Ederef{Ptr: ptr, Typ: ctypes.Int()}
	result := tr.TranslateExpr(expr)

	eload, ok := result.(csharpminor.Eload)
	if !ok {
		t.Fatalf("expected Eload, got %T", result)
	}
	if eload.Chunk != csharpminor.Mint32 {
		t.Errorf("expected Mint32 chunk, got %v", eload.Chunk)
	}
}

func TestTranslateAddrof(t *testing.T) {
	tr := NewExprTranslator(nil)
	// &x where x is a global
	inner := clight.Evar{Name: "x", Typ: ctypes.Int()}
	expr := clight.Eaddrof{Arg: inner, Typ: ctypes.Pointer(ctypes.Int())}
	result := tr.TranslateExpr(expr)

	eaddrof, ok := result.(csharpminor.Eaddrof)
	if !ok {
		t.Fatalf("expected Eaddrof, got %T", result)
	}
	if eaddrof.Name != "x" {
		t.Errorf("expected name x, got %s", eaddrof.Name)
	}
}

func TestTranslateSizeof(t *testing.T) {
	tests := []struct {
		name string
		typ  ctypes.Type
		want int32
	}{
		{"char", ctypes.Char(), 1},
		{"short", ctypes.Short(), 2},
		{"int", ctypes.Int(), 4},
		{"long", ctypes.Long(), 8},
		{"float", ctypes.Float(), 4},
		{"double", ctypes.Double(), 8},
		{"pointer", ctypes.Pointer(ctypes.Int()), 8},
		{"array", ctypes.Array(ctypes.Int(), 10), 40},
	}

	tr := NewExprTranslator(nil)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expr := clight.Esizeof{ArgType: tc.typ, Typ: ctypes.UInt()}
			result := tr.TranslateExpr(expr)

			econst, ok := result.(csharpminor.Econst)
			if !ok {
				t.Fatalf("expected Econst, got %T", result)
			}
			intConst, ok := econst.Const.(csharpminor.Ointconst)
			if !ok {
				t.Fatalf("expected Ointconst, got %T", econst.Const)
			}
			if intConst.Value != tc.want {
				t.Errorf("sizeof(%s) = %d, want %d", tc.name, intConst.Value, tc.want)
			}
		})
	}
}

func TestTranslateAlignof(t *testing.T) {
	tests := []struct {
		name string
		typ  ctypes.Type
		want int32
	}{
		{"char", ctypes.Char(), 1},
		{"short", ctypes.Short(), 2},
		{"int", ctypes.Int(), 4},
		{"long", ctypes.Long(), 8},
		{"pointer", ctypes.Pointer(ctypes.Int()), 8},
	}

	tr := NewExprTranslator(nil)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expr := clight.Ealignof{ArgType: tc.typ, Typ: ctypes.UInt()}
			result := tr.TranslateExpr(expr)

			econst, ok := result.(csharpminor.Econst)
			if !ok {
				t.Fatalf("expected Econst, got %T", result)
			}
			intConst, ok := econst.Const.(csharpminor.Ointconst)
			if !ok {
				t.Fatalf("expected Ointconst, got %T", econst.Const)
			}
			if intConst.Value != tc.want {
				t.Errorf("alignof(%s) = %d, want %d", tc.name, intConst.Value, tc.want)
			}
		})
	}
}

func TestTranslateFieldAccess(t *testing.T) {
	// Struct: struct point { int x; int y; }
	pointType := ctypes.Tstruct{
		Name: "point",
		Fields: []ctypes.Field{
			{Name: "x", Type: ctypes.Int()},
			{Name: "y", Type: ctypes.Int()},
		},
	}

	tr := NewExprTranslator(nil)

	// p.x where p is a struct point
	p := clight.Evar{Name: "p", Typ: pointType}
	expr := clight.Efield{Arg: p, FieldName: "x", Typ: ctypes.Int()}
	result := tr.TranslateExpr(expr)

	// Should become Eload(Mint32, &p + 0)
	eload, ok := result.(csharpminor.Eload)
	if !ok {
		t.Fatalf("expected Eload, got %T", result)
	}
	if eload.Chunk != csharpminor.Mint32 {
		t.Errorf("expected Mint32 chunk, got %v", eload.Chunk)
	}

	// p.y where p is a struct point - offset should be 4
	expr2 := clight.Efield{Arg: p, FieldName: "y", Typ: ctypes.Int()}
	result2 := tr.TranslateExpr(expr2)

	eload2, ok := result2.(csharpminor.Eload)
	if !ok {
		t.Fatalf("expected Eload, got %T", result2)
	}
	// The address should be a binop (base + offset)
	_, ok = eload2.Addr.(csharpminor.Ebinop)
	if !ok {
		t.Fatalf("expected Ebinop for field y address, got %T", eload2.Addr)
	}
}

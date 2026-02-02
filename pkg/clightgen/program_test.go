package clightgen

import (
	"testing"

	"github.com/raymyers/ralph-cc/pkg/cabs"
	"github.com/raymyers/ralph-cc/pkg/clight"
	"github.com/raymyers/ralph-cc/pkg/ctypes"
)

func TestTranslateProgram_Empty(t *testing.T) {
	prog := &cabs.Program{}
	result := TranslateProgram(prog)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Functions) != 0 {
		t.Errorf("expected 0 functions, got %d", len(result.Functions))
	}
}

func TestTranslateProgram_SingleFunction(t *testing.T) {
	prog := &cabs.Program{
		Definitions: []cabs.Definition{
			cabs.FunDef{
				Name:       "main",
				ReturnType: "int",
				Params:     nil,
				Body:       &cabs.Block{Items: []cabs.Stmt{}},
			},
		},
	}
	result := TranslateProgram(prog)

	if len(result.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(result.Functions))
	}
	if result.Functions[0].Name != "main" {
		t.Errorf("expected function name 'main', got %q", result.Functions[0].Name)
	}
}

func TestTranslateProgram_SkipsFunctionDeclarations(t *testing.T) {
	// Function declarations (prototypes with nil Body) should be skipped
	prog := &cabs.Program{
		Definitions: []cabs.Definition{
			// Declaration (prototype) - should be skipped
			cabs.FunDef{
				Name:       "printf",
				ReturnType: "int",
				Params: []cabs.Param{
					{Name: "format", TypeSpec: "char*"},
				},
				Variadic: true,
				Body:     nil, // nil Body indicates declaration, not definition
			},
			// Definition - should be included
			cabs.FunDef{
				Name:       "main",
				ReturnType: "int",
				Params:     nil,
				Body:       &cabs.Block{Items: []cabs.Stmt{}},
			},
			// Another declaration - should be skipped
			cabs.FunDef{
				Name:       "puts",
				ReturnType: "int",
				Params: []cabs.Param{
					{Name: "s", TypeSpec: "char*"},
				},
				Body: nil,
			},
		},
	}
	result := TranslateProgram(prog)

	// Only the definition (main) should be translated
	if len(result.Functions) != 1 {
		t.Fatalf("expected 1 function (definition only), got %d", len(result.Functions))
	}
	if result.Functions[0].Name != "main" {
		t.Errorf("expected function name 'main', got %q", result.Functions[0].Name)
	}
}

func TestTranslateProgram_FunctionWithParams(t *testing.T) {
	prog := &cabs.Program{
		Definitions: []cabs.Definition{
			cabs.FunDef{
				Name:       "add",
				ReturnType: "int",
				Params: []cabs.Param{
					{Name: "a", TypeSpec: "int"},
					{Name: "b", TypeSpec: "int"},
				},
				Body: &cabs.Block{Items: []cabs.Stmt{}},
			},
		},
	}
	result := TranslateProgram(prog)

	if len(result.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(result.Functions))
	}
	fn := result.Functions[0]
	if len(fn.Params) != 2 {
		t.Errorf("expected 2 params, got %d", len(fn.Params))
	}
}

func TestTranslateProgram_FunctionWithReturn(t *testing.T) {
	prog := &cabs.Program{
		Definitions: []cabs.Definition{
			cabs.FunDef{
				Name:       "f",
				ReturnType: "int",
				Body: &cabs.Block{
					Items: []cabs.Stmt{
						cabs.Return{Expr: cabs.Constant{Value: 42}},
					},
				},
			},
		},
	}
	result := TranslateProgram(prog)

	if len(result.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(result.Functions))
	}
}

func TestTranslateProgram_FunctionWithLocalVar(t *testing.T) {
	prog := &cabs.Program{
		Definitions: []cabs.Definition{
			cabs.FunDef{
				Name:       "f",
				ReturnType: "int",
				Body: &cabs.Block{
					Items: []cabs.Stmt{
						cabs.DeclStmt{
							Decls: []cabs.Decl{
								{Name: "x", TypeSpec: "int", Initializer: cabs.Constant{Value: 5}},
							},
						},
						cabs.Return{Expr: cabs.Variable{Name: "x"}},
					},
				},
			},
		},
	}
	result := TranslateProgram(prog)

	if len(result.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(result.Functions))
	}
}

func TestTranslateProgram_StructDef(t *testing.T) {
	prog := &cabs.Program{
		Definitions: []cabs.Definition{
			cabs.StructDef{
				Name: "Point",
				Fields: []cabs.StructField{
					{Name: "x", TypeSpec: "int"},
					{Name: "y", TypeSpec: "int"},
				},
			},
		},
	}
	result := TranslateProgram(prog)

	if len(result.Structs) != 1 {
		t.Fatalf("expected 1 struct, got %d", len(result.Structs))
	}
	if result.Structs[0].Name != "Point" {
		t.Errorf("expected struct name 'Point', got %q", result.Structs[0].Name)
	}
	if len(result.Structs[0].Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(result.Structs[0].Fields))
	}
}

func TestTranslateProgram_UnionDef(t *testing.T) {
	prog := &cabs.Program{
		Definitions: []cabs.Definition{
			cabs.UnionDef{
				Name: "Value",
				Fields: []cabs.StructField{
					{Name: "i", TypeSpec: "int"},
					{Name: "f", TypeSpec: "float"},
				},
			},
		},
	}
	result := TranslateProgram(prog)

	if len(result.Unions) != 1 {
		t.Fatalf("expected 1 union, got %d", len(result.Unions))
	}
	if result.Unions[0].Name != "Value" {
		t.Errorf("expected union name 'Value', got %q", result.Unions[0].Name)
	}
}

func TestTypeFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected ctypes.Type
	}{
		{"void", ctypes.Void()},
		{"char", ctypes.Char()},
		{"unsigned char", ctypes.UChar()},
		{"short", ctypes.Short()},
		{"int", ctypes.Int()},
		{"unsigned int", ctypes.UInt()},
		{"unsigned", ctypes.UInt()},
		{"long", ctypes.Long()},
		{"unsigned long", ctypes.Tlong{Sign: ctypes.Unsigned}},
		{"float", ctypes.Float()},
		{"double", ctypes.Double()},
		{"int*", ctypes.Pointer(ctypes.Int())},
		{"char*", ctypes.Pointer(ctypes.Char())},
		{"struct Point", ctypes.Tstruct{Name: "Point"}},
		{"union Value", ctypes.Tunion{Name: "Value"}},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := TypeFromString(tc.input)
			if !typesEqual(result, tc.expected) {
				t.Errorf("TypeFromString(%q) = %T, expected %T", tc.input, result, tc.expected)
			}
		})
	}
}

func typesEqual(a, b ctypes.Type) bool {
	switch at := a.(type) {
	case ctypes.Tvoid:
		_, ok := b.(ctypes.Tvoid)
		return ok
	case ctypes.Tint:
		bt, ok := b.(ctypes.Tint)
		return ok && at.Size == bt.Size && at.Sign == bt.Sign
	case ctypes.Tlong:
		bt, ok := b.(ctypes.Tlong)
		return ok && at.Sign == bt.Sign
	case ctypes.Tfloat:
		bt, ok := b.(ctypes.Tfloat)
		return ok && at.Size == bt.Size
	case ctypes.Tpointer:
		bt, ok := b.(ctypes.Tpointer)
		return ok && typesEqual(at.Elem, bt.Elem)
	case ctypes.Tstruct:
		bt, ok := b.(ctypes.Tstruct)
		return ok && at.Name == bt.Name
	case ctypes.Tunion:
		bt, ok := b.(ctypes.Tunion)
		return ok && at.Name == bt.Name
	default:
		return false
	}
}

func TestTransformStmt_Return(t *testing.T) {
	prog := &cabs.Program{
		Definitions: []cabs.Definition{
			cabs.FunDef{
				Name:       "f",
				ReturnType: "void",
				Body: &cabs.Block{
					Items: []cabs.Stmt{
						cabs.Return{Expr: nil}, // void return
					},
				},
			},
		},
	}
	result := TranslateProgram(prog)
	if len(result.Functions) != 1 {
		t.Fatalf("expected 1 function")
	}
}

func TestTransformStmt_If(t *testing.T) {
	prog := &cabs.Program{
		Definitions: []cabs.Definition{
			cabs.FunDef{
				Name:       "f",
				ReturnType: "int",
				Params:     []cabs.Param{{Name: "x", TypeSpec: "int"}},
				Body: &cabs.Block{
					Items: []cabs.Stmt{
						cabs.If{
							Cond: cabs.Variable{Name: "x"},
							Then: cabs.Return{Expr: cabs.Constant{Value: 1}},
							Else: nil,
						},
						cabs.Return{Expr: cabs.Constant{Value: 0}},
					},
				},
			},
		},
	}
	result := TranslateProgram(prog)
	if len(result.Functions) != 1 {
		t.Fatalf("expected 1 function")
	}
}

func TestTransformStmt_IfElse(t *testing.T) {
	prog := &cabs.Program{
		Definitions: []cabs.Definition{
			cabs.FunDef{
				Name:       "f",
				ReturnType: "int",
				Params:     []cabs.Param{{Name: "x", TypeSpec: "int"}},
				Body: &cabs.Block{
					Items: []cabs.Stmt{
						cabs.If{
							Cond: cabs.Variable{Name: "x"},
							Then: cabs.Return{Expr: cabs.Constant{Value: 1}},
							Else: cabs.Return{Expr: cabs.Constant{Value: 0}},
						},
					},
				},
			},
		},
	}
	result := TranslateProgram(prog)
	if len(result.Functions) != 1 {
		t.Fatalf("expected 1 function")
	}
}

func TestTransformStmt_While(t *testing.T) {
	prog := &cabs.Program{
		Definitions: []cabs.Definition{
			cabs.FunDef{
				Name:       "f",
				ReturnType: "void",
				Params:     []cabs.Param{{Name: "n", TypeSpec: "int"}},
				Body: &cabs.Block{
					Items: []cabs.Stmt{
						cabs.While{
							Cond: cabs.Variable{Name: "n"},
							Body: cabs.Computation{Expr: cabs.Unary{Op: cabs.OpPreDec, Expr: cabs.Variable{Name: "n"}}},
						},
					},
				},
			},
		},
	}
	result := TranslateProgram(prog)
	if len(result.Functions) != 1 {
		t.Fatalf("expected 1 function")
	}
	// Check that body contains a loop
	fn := result.Functions[0]
	if !containsLoop(fn.Body) {
		t.Error("expected body to contain a loop")
	}
}

func TestTransformStmt_DoWhile(t *testing.T) {
	prog := &cabs.Program{
		Definitions: []cabs.Definition{
			cabs.FunDef{
				Name:       "f",
				ReturnType: "void",
				Params:     []cabs.Param{{Name: "n", TypeSpec: "int"}},
				Body: &cabs.Block{
					Items: []cabs.Stmt{
						cabs.DoWhile{
							Body: cabs.Computation{Expr: cabs.Unary{Op: cabs.OpPreDec, Expr: cabs.Variable{Name: "n"}}},
							Cond: cabs.Variable{Name: "n"},
						},
					},
				},
			},
		},
	}
	result := TranslateProgram(prog)
	if len(result.Functions) != 1 {
		t.Fatalf("expected 1 function")
	}
}

func TestTransformStmt_For(t *testing.T) {
	prog := &cabs.Program{
		Definitions: []cabs.Definition{
			cabs.FunDef{
				Name:       "f",
				ReturnType: "void",
				Params:     []cabs.Param{{Name: "n", TypeSpec: "int"}},
				Body: &cabs.Block{
					Items: []cabs.Stmt{
						cabs.For{
							Init: cabs.Binary{
								Op:    cabs.OpAssign,
								Left:  cabs.Variable{Name: "n"},
								Right: cabs.Constant{Value: 0},
							},
							Cond: cabs.Binary{
								Op:    cabs.OpLt,
								Left:  cabs.Variable{Name: "n"},
								Right: cabs.Constant{Value: 10},
							},
							Step: cabs.Unary{Op: cabs.OpPostInc, Expr: cabs.Variable{Name: "n"}},
							Body: cabs.Break{},
						},
					},
				},
			},
		},
	}
	result := TranslateProgram(prog)
	if len(result.Functions) != 1 {
		t.Fatalf("expected 1 function")
	}
}

func TestTransformStmt_ForNoCond(t *testing.T) {
	// for (;;) style infinite loop
	prog := &cabs.Program{
		Definitions: []cabs.Definition{
			cabs.FunDef{
				Name:       "f",
				ReturnType: "void",
				Body: &cabs.Block{
					Items: []cabs.Stmt{
						cabs.For{
							Body: cabs.Break{},
						},
					},
				},
			},
		},
	}
	result := TranslateProgram(prog)
	if len(result.Functions) != 1 {
		t.Fatalf("expected 1 function")
	}
}

func TestTransformStmt_Break(t *testing.T) {
	prog := &cabs.Program{
		Definitions: []cabs.Definition{
			cabs.FunDef{
				Name:       "f",
				ReturnType: "void",
				Body: &cabs.Block{
					Items: []cabs.Stmt{
						cabs.While{
							Cond: cabs.Constant{Value: 1},
							Body: cabs.Break{},
						},
					},
				},
			},
		},
	}
	result := TranslateProgram(prog)
	if len(result.Functions) != 1 {
		t.Fatalf("expected 1 function")
	}
}

func TestTransformStmt_Continue(t *testing.T) {
	prog := &cabs.Program{
		Definitions: []cabs.Definition{
			cabs.FunDef{
				Name:       "f",
				ReturnType: "void",
				Body: &cabs.Block{
					Items: []cabs.Stmt{
						cabs.While{
							Cond: cabs.Constant{Value: 1},
							Body: cabs.Continue{},
						},
					},
				},
			},
		},
	}
	result := TranslateProgram(prog)
	if len(result.Functions) != 1 {
		t.Fatalf("expected 1 function")
	}
}

func TestTransformStmt_Switch(t *testing.T) {
	prog := &cabs.Program{
		Definitions: []cabs.Definition{
			cabs.FunDef{
				Name:       "f",
				ReturnType: "int",
				Params:     []cabs.Param{{Name: "x", TypeSpec: "int"}},
				Body: &cabs.Block{
					Items: []cabs.Stmt{
						cabs.Switch{
							Expr: cabs.Variable{Name: "x"},
							Cases: []cabs.SwitchCase{
								{Expr: cabs.Constant{Value: 1}, Stmts: []cabs.Stmt{cabs.Return{Expr: cabs.Constant{Value: 1}}}},
								{Expr: cabs.Constant{Value: 2}, Stmts: []cabs.Stmt{cabs.Return{Expr: cabs.Constant{Value: 2}}}},
								{Expr: nil, Stmts: []cabs.Stmt{cabs.Return{Expr: cabs.Constant{Value: 0}}}}, // default
							},
						},
					},
				},
			},
		},
	}
	result := TranslateProgram(prog)
	if len(result.Functions) != 1 {
		t.Fatalf("expected 1 function")
	}
}

func TestTransformStmt_GotoLabel(t *testing.T) {
	prog := &cabs.Program{
		Definitions: []cabs.Definition{
			cabs.FunDef{
				Name:       "f",
				ReturnType: "int",
				Body: &cabs.Block{
					Items: []cabs.Stmt{
						cabs.Goto{Label: "done"},
						cabs.Label{Name: "done", Stmt: cabs.Return{Expr: cabs.Constant{Value: 0}}},
					},
				},
			},
		},
	}
	result := TranslateProgram(prog)
	if len(result.Functions) != 1 {
		t.Fatalf("expected 1 function")
	}
}

func TestTransformStmt_NestedBlock(t *testing.T) {
	prog := &cabs.Program{
		Definitions: []cabs.Definition{
			cabs.FunDef{
				Name:       "f",
				ReturnType: "int",
				Body: &cabs.Block{
					Items: []cabs.Stmt{
						cabs.Block{
							Items: []cabs.Stmt{
								cabs.DeclStmt{Decls: []cabs.Decl{{Name: "x", TypeSpec: "int"}}},
							},
						},
						cabs.Return{Expr: cabs.Constant{Value: 0}},
					},
				},
			},
		},
	}
	result := TranslateProgram(prog)
	if len(result.Functions) != 1 {
		t.Fatalf("expected 1 function")
	}
}

func TestTransformStmt_Computation(t *testing.T) {
	prog := &cabs.Program{
		Definitions: []cabs.Definition{
			cabs.FunDef{
				Name:       "f",
				ReturnType: "void",
				Params:     []cabs.Param{{Name: "x", TypeSpec: "int"}},
				Body: &cabs.Block{
					Items: []cabs.Stmt{
						cabs.Computation{Expr: cabs.Unary{Op: cabs.OpPostInc, Expr: cabs.Variable{Name: "x"}}},
					},
				},
			},
		},
	}
	result := TranslateProgram(prog)
	if len(result.Functions) != 1 {
		t.Fatalf("expected 1 function")
	}
}

func containsLoop(stmt clight.Stmt) bool {
	switch s := stmt.(type) {
	case clight.Sloop:
		return true
	case clight.Ssequence:
		return containsLoop(s.First) || containsLoop(s.Second)
	default:
		return false
	}
}

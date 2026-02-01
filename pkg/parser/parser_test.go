package parser

import (
	"fmt"
	"os"
	"testing"

	"github.com/raymyers/ralph-cc/pkg/cabs"
	"github.com/raymyers/ralph-cc/pkg/lexer"
	"gopkg.in/yaml.v3"
)

// TestSpec represents a test case from parse.yaml
type TestSpec struct {
	Name  string  `yaml:"name"`
	Input string  `yaml:"input"`
	AST   ASTSpec `yaml:"ast"`
}

// ASTSpec represents the expected AST structure
type ASTSpec struct {
	Kind       string    `yaml:"kind"`
	Name       string    `yaml:"name,omitempty"`
	ReturnType string    `yaml:"return_type,omitempty"`
	Body       *ASTSpec  `yaml:"body,omitempty"`
	Items      []ASTSpec `yaml:"items,omitempty"`
	Expr       *ASTSpec  `yaml:"expr,omitempty"`
	Left       *ASTSpec  `yaml:"left,omitempty"`
	Right      *ASTSpec  `yaml:"right,omitempty"`
	Op         string    `yaml:"op,omitempty"`
	Value      *int64    `yaml:"value,omitempty"`
}

// TestFile represents the parse.yaml file structure
type TestFile struct {
	Tests []TestSpec `yaml:"tests"`
}

func TestParseYAML(t *testing.T) {
	data, err := os.ReadFile("../../testdata/parse.yaml")
	if err != nil {
		t.Fatalf("failed to read parse.yaml: %v", err)
	}

	var testFile TestFile
	if err := yaml.Unmarshal(data, &testFile); err != nil {
		t.Fatalf("failed to parse parse.yaml: %v", err)
	}

	for _, tc := range testFile.Tests {
		t.Run(tc.Name, func(t *testing.T) {
			l := lexer.New(tc.Input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			if def == nil {
				t.Fatal("ParseDefinition returned nil")
			}

			verifyAST(t, def, tc.AST)
		})
	}
}

func verifyAST(t *testing.T, node cabs.Node, spec ASTSpec) {
	t.Helper()

	switch spec.Kind {
	case "FunDef":
		funDef, ok := node.(cabs.FunDef)
		if !ok {
			t.Fatalf("expected FunDef, got %T", node)
		}
		if spec.Name != "" && funDef.Name != spec.Name {
			t.Errorf("FunDef.Name: expected %q, got %q", spec.Name, funDef.Name)
		}
		if spec.ReturnType != "" && funDef.ReturnType != spec.ReturnType {
			t.Errorf("FunDef.ReturnType: expected %q, got %q", spec.ReturnType, funDef.ReturnType)
		}
		if spec.Body != nil {
			verifyAST(t, *funDef.Body, *spec.Body)
		}

	case "Block":
		block, ok := node.(cabs.Block)
		if !ok {
			t.Fatalf("expected Block, got %T", node)
		}
		if len(spec.Items) != len(block.Items) {
			t.Fatalf("Block.Items: expected %d items, got %d", len(spec.Items), len(block.Items))
		}
		for i, itemSpec := range spec.Items {
			verifyAST(t, block.Items[i], itemSpec)
		}

	case "Return":
		ret, ok := node.(cabs.Return)
		if !ok {
			t.Fatalf("expected Return, got %T", node)
		}
		if spec.Expr != nil {
			if ret.Expr == nil {
				t.Fatal("Return.Expr: expected expression, got nil")
			}
			verifyAST(t, ret.Expr, *spec.Expr)
		}

	case "Constant":
		constant, ok := node.(cabs.Constant)
		if !ok {
			t.Fatalf("expected Constant, got %T", node)
		}
		if spec.Value != nil && constant.Value != *spec.Value {
			t.Errorf("Constant.Value: expected %d, got %d", *spec.Value, constant.Value)
		}

	case "Variable":
		variable, ok := node.(cabs.Variable)
		if !ok {
			t.Fatalf("expected Variable, got %T", node)
		}
		if spec.Name != "" && variable.Name != spec.Name {
			t.Errorf("Variable.Name: expected %q, got %q", spec.Name, variable.Name)
		}

	case "Binary":
		binary, ok := node.(cabs.Binary)
		if !ok {
			t.Fatalf("expected Binary, got %T", node)
		}
		if spec.Op != "" && binary.Op.String() != spec.Op {
			t.Errorf("Binary.Op: expected %q, got %q", spec.Op, binary.Op.String())
		}
		if spec.Left != nil {
			verifyAST(t, binary.Left, *spec.Left)
		}
		if spec.Right != nil {
			verifyAST(t, binary.Right, *spec.Right)
		}

	case "Unary":
		unary, ok := node.(cabs.Unary)
		if !ok {
			t.Fatalf("expected Unary, got %T", node)
		}
		if spec.Op != "" && unary.Op.String() != spec.Op {
			t.Errorf("Unary.Op: expected %q, got %q", spec.Op, unary.Op.String())
		}
		if spec.Expr != nil {
			verifyAST(t, unary.Expr, *spec.Expr)
		}

	case "Paren":
		paren, ok := node.(cabs.Paren)
		if !ok {
			t.Fatalf("expected Paren, got %T", node)
		}
		if spec.Expr != nil {
			verifyAST(t, paren.Expr, *spec.Expr)
		}

	case "Conditional":
		cond, ok := node.(cabs.Conditional)
		if !ok {
			t.Fatalf("expected Conditional, got %T", node)
		}
		// We'd need Cond, Then, Else fields in ASTSpec to fully verify
		_ = cond

	default:
		t.Fatalf("unknown AST kind: %s", spec.Kind)
	}
}

func TestEmptyFunction(t *testing.T) {
	input := `int main() {}`

	l := lexer.New(input)
	p := New(l)
	def := p.ParseDefinition()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	funDef, ok := def.(cabs.FunDef)
	if !ok {
		t.Fatalf("expected FunDef, got %T", def)
	}

	if funDef.Name != "main" {
		t.Errorf("expected name 'main', got %q", funDef.Name)
	}
	if funDef.ReturnType != "int" {
		t.Errorf("expected return type 'int', got %q", funDef.ReturnType)
	}
	if len(funDef.Body.Items) != 0 {
		t.Errorf("expected empty body, got %d items", len(funDef.Body.Items))
	}
}

func TestReturnStatement(t *testing.T) {
	input := `int f() { return 42; }`

	l := lexer.New(input)
	p := New(l)
	def := p.ParseDefinition()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	funDef, ok := def.(cabs.FunDef)
	if !ok {
		t.Fatalf("expected FunDef, got %T", def)
	}

	if len(funDef.Body.Items) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(funDef.Body.Items))
	}

	ret, ok := funDef.Body.Items[0].(cabs.Return)
	if !ok {
		t.Fatalf("expected Return, got %T", funDef.Body.Items[0])
	}

	constant, ok := ret.Expr.(cabs.Constant)
	if !ok {
		t.Fatalf("expected Constant, got %T", ret.Expr)
	}

	if constant.Value != 42 {
		t.Errorf("expected value 42, got %d", constant.Value)
	}
}

func TestBinaryExpressions(t *testing.T) {
	tests := []struct {
		input    string
		leftVal  int64
		op       cabs.BinaryOp
		rightVal int64
	}{
		{"int f() { return 1 + 2; }", 1, cabs.OpAdd, 2},
		{"int f() { return 5 - 3; }", 5, cabs.OpSub, 3},
		{"int f() { return 2 * 3; }", 2, cabs.OpMul, 3},
		{"int f() { return 6 / 2; }", 6, cabs.OpDiv, 2},
		{"int f() { return 7 % 3; }", 7, cabs.OpMod, 3},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			funDef := def.(cabs.FunDef)
			ret := funDef.Body.Items[0].(cabs.Return)
			binary, ok := ret.Expr.(cabs.Binary)
			if !ok {
				t.Fatalf("expected Binary, got %T", ret.Expr)
			}

			if binary.Op != tt.op {
				t.Errorf("wrong op: expected %v, got %v", tt.op, binary.Op)
			}

			left := binary.Left.(cabs.Constant)
			if left.Value != tt.leftVal {
				t.Errorf("wrong left value: expected %d, got %d", tt.leftVal, left.Value)
			}

			right := binary.Right.(cabs.Constant)
			if right.Value != tt.rightVal {
				t.Errorf("wrong right value: expected %d, got %d", tt.rightVal, right.Value)
			}
		})
	}
}

func TestOperatorPrecedence(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Multiplicative before additive
		{"int f() { return 1 + 2 * 3; }", "(1 + (2 * 3))"},
		{"int f() { return 2 * 3 + 4; }", "((2 * 3) + 4)"},
		// Parentheses override precedence
		{"int f() { return (1 + 2) * 3; }", "((1 + 2) * 3)"},
		// Left associativity
		{"int f() { return 1 - 2 - 3; }", "((1 - 2) - 3)"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			funDef := def.(cabs.FunDef)
			ret := funDef.Body.Items[0].(cabs.Return)
			actual := exprString(ret.Expr)

			if actual != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, actual)
			}
		})
	}
}

func TestUnaryExpressions(t *testing.T) {
	tests := []struct {
		input    string
		op       cabs.UnaryOp
		innerVal int64
	}{
		{"int f() { return -5; }", cabs.OpNeg, 5},
		{"int f() { return !0; }", cabs.OpNot, 0},
		{"int f() { return ~1; }", cabs.OpBitNot, 1},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			funDef := def.(cabs.FunDef)
			ret := funDef.Body.Items[0].(cabs.Return)
			unary, ok := ret.Expr.(cabs.Unary)
			if !ok {
				t.Fatalf("expected Unary, got %T", ret.Expr)
			}

			if unary.Op != tt.op {
				t.Errorf("wrong op: expected %v, got %v", tt.op, unary.Op)
			}

			constant := unary.Expr.(cabs.Constant)
			if constant.Value != tt.innerVal {
				t.Errorf("wrong inner value: expected %d, got %d", tt.innerVal, constant.Value)
			}
		})
	}
}

func TestVariableExpressions(t *testing.T) {
	input := `int f() { return x; }`

	l := lexer.New(input)
	p := New(l)
	def := p.ParseDefinition()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	funDef := def.(cabs.FunDef)
	ret := funDef.Body.Items[0].(cabs.Return)
	variable, ok := ret.Expr.(cabs.Variable)
	if !ok {
		t.Fatalf("expected Variable, got %T", ret.Expr)
	}

	if variable.Name != "x" {
		t.Errorf("expected name 'x', got %q", variable.Name)
	}
}

func TestParenthesizedExpressions(t *testing.T) {
	input := `int f() { return (42); }`

	l := lexer.New(input)
	p := New(l)
	def := p.ParseDefinition()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	funDef := def.(cabs.FunDef)
	ret := funDef.Body.Items[0].(cabs.Return)
	paren, ok := ret.Expr.(cabs.Paren)
	if !ok {
		t.Fatalf("expected Paren, got %T", ret.Expr)
	}

	constant := paren.Expr.(cabs.Constant)
	if constant.Value != 42 {
		t.Errorf("expected value 42, got %d", constant.Value)
	}
}

func TestComparisonAndLogicalOperators(t *testing.T) {
	tests := []struct {
		input string
		op    cabs.BinaryOp
	}{
		{"int f() { return 1 < 2; }", cabs.OpLt},
		{"int f() { return 1 <= 2; }", cabs.OpLe},
		{"int f() { return 1 > 2; }", cabs.OpGt},
		{"int f() { return 1 >= 2; }", cabs.OpGe},
		{"int f() { return 1 == 2; }", cabs.OpEq},
		{"int f() { return 1 != 2; }", cabs.OpNe},
		{"int f() { return 1 && 2; }", cabs.OpAnd},
		{"int f() { return 1 || 2; }", cabs.OpOr},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			funDef := def.(cabs.FunDef)
			ret := funDef.Body.Items[0].(cabs.Return)
			binary, ok := ret.Expr.(cabs.Binary)
			if !ok {
				t.Fatalf("expected Binary, got %T", ret.Expr)
			}

			if binary.Op != tt.op {
				t.Errorf("wrong op: expected %v, got %v", tt.op, binary.Op)
			}
		})
	}
}

func TestBitwiseOperators(t *testing.T) {
	tests := []struct {
		input string
		op    cabs.BinaryOp
	}{
		{"int f() { return 1 & 2; }", cabs.OpBitAnd},
		{"int f() { return 1 | 2; }", cabs.OpBitOr},
		{"int f() { return 1 ^ 2; }", cabs.OpBitXor},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			funDef := def.(cabs.FunDef)
			ret := funDef.Body.Items[0].(cabs.Return)
			binary, ok := ret.Expr.(cabs.Binary)
			if !ok {
				t.Fatalf("expected Binary, got %T", ret.Expr)
			}

			if binary.Op != tt.op {
				t.Errorf("wrong op: expected %v, got %v", tt.op, binary.Op)
			}
		})
	}
}

func TestShiftOperators(t *testing.T) {
	tests := []struct {
		input string
		op    cabs.BinaryOp
	}{
		{"int f() { return 1 << 2; }", cabs.OpShl},
		{"int f() { return 8 >> 2; }", cabs.OpShr},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			funDef := def.(cabs.FunDef)
			ret := funDef.Body.Items[0].(cabs.Return)
			binary, ok := ret.Expr.(cabs.Binary)
			if !ok {
				t.Fatalf("expected Binary, got %T", ret.Expr)
			}

			if binary.Op != tt.op {
				t.Errorf("wrong op: expected %v, got %v", tt.op, binary.Op)
			}
		})
	}
}

func TestTernaryOperator(t *testing.T) {
	input := `int f() { return 1 ? 2 : 3; }`

	l := lexer.New(input)
	p := New(l)
	def := p.ParseDefinition()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	funDef := def.(cabs.FunDef)
	ret := funDef.Body.Items[0].(cabs.Return)
	cond, ok := ret.Expr.(cabs.Conditional)
	if !ok {
		t.Fatalf("expected Conditional, got %T", ret.Expr)
	}

	condVal := cond.Cond.(cabs.Constant)
	if condVal.Value != 1 {
		t.Errorf("expected cond value 1, got %d", condVal.Value)
	}

	thenVal := cond.Then.(cabs.Constant)
	if thenVal.Value != 2 {
		t.Errorf("expected then value 2, got %d", thenVal.Value)
	}

	elseVal := cond.Else.(cabs.Constant)
	if elseVal.Value != 3 {
		t.Errorf("expected else value 3, got %d", elseVal.Value)
	}
}

func TestAssignmentOperator(t *testing.T) {
	input := `int f() { return x = 1; }`

	l := lexer.New(input)
	p := New(l)
	def := p.ParseDefinition()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	funDef := def.(cabs.FunDef)
	ret := funDef.Body.Items[0].(cabs.Return)
	binary, ok := ret.Expr.(cabs.Binary)
	if !ok {
		t.Fatalf("expected Binary, got %T", ret.Expr)
	}

	if binary.Op != cabs.OpAssign {
		t.Errorf("wrong op: expected OpAssign, got %v", binary.Op)
	}

	left := binary.Left.(cabs.Variable)
	if left.Name != "x" {
		t.Errorf("expected left to be variable 'x', got %q", left.Name)
	}

	right := binary.Right.(cabs.Constant)
	if right.Value != 1 {
		t.Errorf("expected right to be 1, got %d", right.Value)
	}
}

func TestFunctionCall(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		funcName string
		argCount int
	}{
		{"no args", "int f() { return foo(); }", "foo", 0},
		{"one arg", "int f() { return bar(1); }", "bar", 1},
		{"two args", "int f() { return baz(1, 2); }", "baz", 2},
		{"three args", "int f() { return qux(1, 2, 3); }", "qux", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			funDef := def.(cabs.FunDef)
			ret := funDef.Body.Items[0].(cabs.Return)
			call, ok := ret.Expr.(cabs.Call)
			if !ok {
				t.Fatalf("expected Call, got %T", ret.Expr)
			}

			fn := call.Func.(cabs.Variable)
			if fn.Name != tt.funcName {
				t.Errorf("expected function name %q, got %q", tt.funcName, fn.Name)
			}

			if len(call.Args) != tt.argCount {
				t.Errorf("expected %d args, got %d", tt.argCount, len(call.Args))
			}
		})
	}
}

func TestArraySubscript(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		arrayName string
		indexVal  int64
	}{
		{"simple", "int f() { return a[0]; }", "a", 0},
		{"with index", "int f() { return arr[5]; }", "arr", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			funDef := def.(cabs.FunDef)
			ret := funDef.Body.Items[0].(cabs.Return)
			idx, ok := ret.Expr.(cabs.Index)
			if !ok {
				t.Fatalf("expected Index, got %T", ret.Expr)
			}

			arr := idx.Array.(cabs.Variable)
			if arr.Name != tt.arrayName {
				t.Errorf("expected array name %q, got %q", tt.arrayName, arr.Name)
			}

			index := idx.Index.(cabs.Constant)
			if index.Value != tt.indexVal {
				t.Errorf("expected index %d, got %d", tt.indexVal, index.Value)
			}
		})
	}
}

func TestCompoundAssignment(t *testing.T) {
	tests := []struct {
		input string
		op    cabs.BinaryOp
	}{
		{"int f() { return x += 1; }", cabs.OpAddAssign},
		{"int f() { return x -= 1; }", cabs.OpSubAssign},
		{"int f() { return x *= 2; }", cabs.OpMulAssign},
		{"int f() { return x /= 2; }", cabs.OpDivAssign},
		{"int f() { return x %= 3; }", cabs.OpModAssign},
		{"int f() { return x &= 1; }", cabs.OpAndAssign},
		{"int f() { return x |= 1; }", cabs.OpOrAssign},
		{"int f() { return x ^= 1; }", cabs.OpXorAssign},
		{"int f() { return x <<= 1; }", cabs.OpShlAssign},
		{"int f() { return x >>= 1; }", cabs.OpShrAssign},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			funDef := def.(cabs.FunDef)
			ret := funDef.Body.Items[0].(cabs.Return)
			binary, ok := ret.Expr.(cabs.Binary)
			if !ok {
				t.Fatalf("expected Binary, got %T", ret.Expr)
			}

			if binary.Op != tt.op {
				t.Errorf("wrong op: expected %v, got %v", tt.op, binary.Op)
			}

			left := binary.Left.(cabs.Variable)
			if left.Name != "x" {
				t.Errorf("expected left to be variable 'x', got %q", left.Name)
			}
		})
	}
}

func TestPrefixIncDec(t *testing.T) {
	tests := []struct {
		input string
		op    cabs.UnaryOp
	}{
		{"int f() { return ++x; }", cabs.OpPreInc},
		{"int f() { return --x; }", cabs.OpPreDec},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			funDef := def.(cabs.FunDef)
			ret := funDef.Body.Items[0].(cabs.Return)
			unary, ok := ret.Expr.(cabs.Unary)
			if !ok {
				t.Fatalf("expected Unary, got %T", ret.Expr)
			}

			if unary.Op != tt.op {
				t.Errorf("wrong op: expected %v, got %v", tt.op, unary.Op)
			}

			inner := unary.Expr.(cabs.Variable)
			if inner.Name != "x" {
				t.Errorf("expected inner to be variable 'x', got %q", inner.Name)
			}
		})
	}
}

func TestPostfixIncDec(t *testing.T) {
	tests := []struct {
		input string
		op    cabs.UnaryOp
	}{
		{"int f() { return x++; }", cabs.OpPostInc},
		{"int f() { return x--; }", cabs.OpPostDec},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			funDef := def.(cabs.FunDef)
			ret := funDef.Body.Items[0].(cabs.Return)
			unary, ok := ret.Expr.(cabs.Unary)
			if !ok {
				t.Fatalf("expected Unary, got %T", ret.Expr)
			}

			if unary.Op != tt.op {
				t.Errorf("wrong op: expected %v, got %v", tt.op, unary.Op)
			}

			inner := unary.Expr.(cabs.Variable)
			if inner.Name != "x" {
				t.Errorf("expected inner to be variable 'x', got %q", inner.Name)
			}
		})
	}
}

func TestMemberAccess(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		structName string
		memberName string
		isArrow    bool
	}{
		{"dot", "int f() { return s.x; }", "s", "x", false},
		{"arrow", "int f() { return p->y; }", "p", "y", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			funDef := def.(cabs.FunDef)
			ret := funDef.Body.Items[0].(cabs.Return)
			member, ok := ret.Expr.(cabs.Member)
			if !ok {
				t.Fatalf("expected Member, got %T", ret.Expr)
			}

			varExpr := member.Expr.(cabs.Variable)
			if varExpr.Name != tt.structName {
				t.Errorf("expected struct name %q, got %q", tt.structName, varExpr.Name)
			}

			if member.Name != tt.memberName {
				t.Errorf("expected member name %q, got %q", tt.memberName, member.Name)
			}

			if member.IsArrow != tt.isArrow {
				t.Errorf("expected isArrow=%v, got %v", tt.isArrow, member.IsArrow)
			}
		})
	}
}

func TestAddressAndDereference(t *testing.T) {
	tests := []struct {
		input string
		op    cabs.UnaryOp
	}{
		{"int f() { return &x; }", cabs.OpAddrOf},
		{"int f() { return *p; }", cabs.OpDeref},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			funDef := def.(cabs.FunDef)
			ret := funDef.Body.Items[0].(cabs.Return)
			unary, ok := ret.Expr.(cabs.Unary)
			if !ok {
				t.Fatalf("expected Unary, got %T", ret.Expr)
			}

			if unary.Op != tt.op {
				t.Errorf("wrong op: expected %v, got %v", tt.op, unary.Op)
			}
		})
	}
}

func TestCommaOperator(t *testing.T) {
	input := `int f() { return 1, 2; }`

	l := lexer.New(input)
	p := New(l)
	def := p.ParseDefinition()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	funDef := def.(cabs.FunDef)
	ret := funDef.Body.Items[0].(cabs.Return)
	binary, ok := ret.Expr.(cabs.Binary)
	if !ok {
		t.Fatalf("expected Binary, got %T", ret.Expr)
	}

	if binary.Op != cabs.OpComma {
		t.Errorf("wrong op: expected OpComma, got %v", binary.Op)
	}
}

// exprString returns a string representation of an expression for testing
func exprString(e cabs.Expr) string {
	switch expr := e.(type) {
	case cabs.Constant:
		return fmt.Sprintf("%d", expr.Value)
	case cabs.Variable:
		return expr.Name
	case cabs.Binary:
		return fmt.Sprintf("(%s %s %s)", exprString(expr.Left), expr.Op.String(), exprString(expr.Right))
	case cabs.Unary:
		return fmt.Sprintf("(%s%s)", expr.Op.String(), exprString(expr.Expr))
	case cabs.Paren:
		return exprString(expr.Expr)
	case cabs.Conditional:
		return fmt.Sprintf("(%s ? %s : %s)", exprString(expr.Cond), exprString(expr.Then), exprString(expr.Else))
	default:
		return "?"
	}
}

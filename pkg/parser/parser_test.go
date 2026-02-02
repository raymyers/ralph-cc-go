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

func TestSizeofExpr(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"sizeof variable", "int f() { return sizeof x; }"},
		{"sizeof parenthesized expr", "int f() { return sizeof(x); }"},
		{"sizeof unary expr", "int f() { return sizeof *p; }"},
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
			_, ok := ret.Expr.(cabs.SizeofExpr)
			if !ok {
				t.Fatalf("expected SizeofExpr, got %T", ret.Expr)
			}
		})
	}
}

func TestSizeofType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		typeName string
	}{
		{"sizeof int", "int f() { return sizeof(int); }", "int"},
		{"sizeof void", "int f() { return sizeof(void); }", "void"},
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
			sizeofT, ok := ret.Expr.(cabs.SizeofType)
			if !ok {
				t.Fatalf("expected SizeofType, got %T", ret.Expr)
			}

			if sizeofT.TypeName != tt.typeName {
				t.Errorf("expected type name %q, got %q", tt.typeName, sizeofT.TypeName)
			}
		})
	}
}

func TestCastExpression(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		typeName string
	}{
		{"cast int", "int f() { return (int)x; }", "int"},
		{"cast void", "int f() { return (void)x; }", "void"},
		{"cast with literal", "int f() { return (int)42; }", "int"},
		{"cast with expression", "int f() { return (int)(a + b); }", "int"},
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
			cast, ok := ret.Expr.(cabs.Cast)
			if !ok {
				t.Fatalf("expected Cast, got %T", ret.Expr)
			}

			if cast.TypeName != tt.typeName {
				t.Errorf("expected type name %q, got %q", tt.typeName, cast.TypeName)
			}
		})
	}
}

func TestExpressionStatement(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"function call", "int f() { foo(); }"},
		{"assignment", "int f() { x = 1; }"},
		{"compound assignment", "int f() { x += 2; }"},
		{"increment", "int f() { i++; }"},
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
			if len(funDef.Body.Items) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(funDef.Body.Items))
			}

			_, ok := funDef.Body.Items[0].(cabs.Computation)
			if !ok {
				t.Fatalf("expected Computation, got %T", funDef.Body.Items[0])
			}
		})
	}
}

func TestIfStatement(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		hasElse bool
	}{
		{"if only", "int f() { if (x) return 1; }", false},
		{"if with else", "int f() { if (x) return 1; else return 0; }", true},
		{"if with block", "int f() { if (x) { return 1; } }", false},
		{"if-else with blocks", "int f() { if (x) { return 1; } else { return 0; } }", true},
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
			if len(funDef.Body.Items) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(funDef.Body.Items))
			}

			ifStmt, ok := funDef.Body.Items[0].(cabs.If)
			if !ok {
				t.Fatalf("expected If, got %T", funDef.Body.Items[0])
			}

			if tt.hasElse && ifStmt.Else == nil {
				t.Error("expected else branch, got nil")
			}
			if !tt.hasElse && ifStmt.Else != nil {
				t.Error("expected no else branch")
			}
		})
	}
}

func TestWhileStatement(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"simple while", "int f() { while (x) x = x - 1; }"},
		{"while with block", "int f() { while (x > 0) { x = x - 1; } }"},
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
			if len(funDef.Body.Items) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(funDef.Body.Items))
			}

			_, ok := funDef.Body.Items[0].(cabs.While)
			if !ok {
				t.Fatalf("expected While, got %T", funDef.Body.Items[0])
			}
		})
	}
}

func TestForStatement(t *testing.T) {
	tests := []struct {
		name string
		input string
	}{
		{"complete for", "int f() { for (i = 0; i < 10; i = i + 1) x = x + 1; }"},
		{"for with block", "int f() { for (i = 0; i < 10; i++) { x++; } }"},
		{"infinite loop", "int f() { for (;;) x++; }"},
		{"no init", "int f() { for (; i < 10; i++) x++; }"},
		{"no step", "int f() { for (i = 0; i < 10;) i++; }"},
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
			if len(funDef.Body.Items) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(funDef.Body.Items))
			}

			_, ok := funDef.Body.Items[0].(cabs.For)
			if !ok {
				t.Fatalf("expected For, got %T", funDef.Body.Items[0])
			}
		})
	}
}

func TestBreakStatement(t *testing.T) {
	input := `int f() { while (1) { break; } }`

	l := lexer.New(input)
	p := New(l)
	def := p.ParseDefinition()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	funDef := def.(cabs.FunDef)
	whileStmt := funDef.Body.Items[0].(cabs.While)
	block := whileStmt.Body.(*cabs.Block)

	_, ok := block.Items[0].(cabs.Break)
	if !ok {
		t.Fatalf("expected Break, got %T", block.Items[0])
	}
}

func TestContinueStatement(t *testing.T) {
	input := `int f() { while (1) { continue; } }`

	l := lexer.New(input)
	p := New(l)
	def := p.ParseDefinition()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	funDef := def.(cabs.FunDef)
	whileStmt := funDef.Body.Items[0].(cabs.While)
	block := whileStmt.Body.(*cabs.Block)

	_, ok := block.Items[0].(cabs.Continue)
	if !ok {
		t.Fatalf("expected Continue, got %T", block.Items[0])
	}
}

func TestBreakContinueInFor(t *testing.T) {
	input := `int f() { for (;;) { if (x) break; continue; } }`

	l := lexer.New(input)
	p := New(l)
	def := p.ParseDefinition()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	funDef := def.(cabs.FunDef)
	forStmt := funDef.Body.Items[0].(cabs.For)
	block := forStmt.Body.(*cabs.Block)

	if len(block.Items) != 2 {
		t.Fatalf("expected 2 statements in for body, got %d", len(block.Items))
	}

	_, ok := block.Items[0].(cabs.If)
	if !ok {
		t.Errorf("expected If, got %T", block.Items[0])
	}

	_, ok = block.Items[1].(cabs.Continue)
	if !ok {
		t.Errorf("expected Continue, got %T", block.Items[1])
	}
}

func TestForStatementOptionalParts(t *testing.T) {
	input := "int f() { for (;;) return 1; }"

	l := lexer.New(input)
	p := New(l)
	def := p.ParseDefinition()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	funDef := def.(cabs.FunDef)
	forStmt := funDef.Body.Items[0].(cabs.For)

	if forStmt.Init != nil {
		t.Error("expected nil Init")
	}
	if forStmt.Cond != nil {
		t.Error("expected nil Cond")
	}
	if forStmt.Step != nil {
		t.Error("expected nil Step")
	}
}

func TestDoWhileStatement(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"simple do-while", "int f() { do x = x - 1; while (x); }"},
		{"do-while with block", "int f() { do { x = x - 1; } while (x > 0); }"},
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
			if len(funDef.Body.Items) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(funDef.Body.Items))
			}

			_, ok := funDef.Body.Items[0].(cabs.DoWhile)
			if !ok {
				t.Fatalf("expected DoWhile, got %T", funDef.Body.Items[0])
			}
		})
	}
}

func TestDanglingElse(t *testing.T) {
	// The dangling else should bind to the nearest if
	input := `int f() { if (a) if (b) return 1; else return 2; }`

	l := lexer.New(input)
	p := New(l)
	def := p.ParseDefinition()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	funDef := def.(cabs.FunDef)
	outerIf := funDef.Body.Items[0].(cabs.If)

	// Outer if should NOT have else (else binds to inner if)
	if outerIf.Else != nil {
		t.Error("outer if should not have else branch")
	}

	// Then should be an if statement with an else
	innerIf, ok := outerIf.Then.(cabs.If)
	if !ok {
		t.Fatalf("expected inner If, got %T", outerIf.Then)
	}

	if innerIf.Else == nil {
		t.Error("inner if should have else branch")
	}
}

func TestMultipleStatements(t *testing.T) {
	input := `int f() { x = 1; y = 2; return x + y; }`

	l := lexer.New(input)
	p := New(l)
	def := p.ParseDefinition()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	funDef := def.(cabs.FunDef)
	if len(funDef.Body.Items) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(funDef.Body.Items))
	}

	// First two are expression statements
	_, ok := funDef.Body.Items[0].(cabs.Computation)
	if !ok {
		t.Errorf("statement 0: expected Computation, got %T", funDef.Body.Items[0])
	}
	_, ok = funDef.Body.Items[1].(cabs.Computation)
	if !ok {
		t.Errorf("statement 1: expected Computation, got %T", funDef.Body.Items[1])
	}

	// Third is return
	_, ok = funDef.Body.Items[2].(cabs.Return)
	if !ok {
		t.Errorf("statement 2: expected Return, got %T", funDef.Body.Items[2])
	}
}

func TestCastPrecedence(t *testing.T) {
	// Cast should have higher precedence than binary operators
	input := "int f() { return (int)a + b; }"

	l := lexer.New(input)
	p := New(l)
	def := p.ParseDefinition()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	funDef := def.(cabs.FunDef)
	ret := funDef.Body.Items[0].(cabs.Return)

	// Should be parsed as ((int)a) + b, not (int)(a + b)
	binary, ok := ret.Expr.(cabs.Binary)
	if !ok {
		t.Fatalf("expected Binary at top level, got %T", ret.Expr)
	}

	if binary.Op != cabs.OpAdd {
		t.Errorf("expected + operator, got %v", binary.Op)
	}

	_, ok = binary.Left.(cabs.Cast)
	if !ok {
		t.Errorf("expected Cast as left operand, got %T", binary.Left)
	}
}

func TestSwitchStatement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		numCases int
	}{
		{"simple switch", "int f() { switch (x) { case 1: return 1; } }", 1},
		{"switch with default", "int f() { switch (x) { case 1: return 1; default: return 0; } }", 2},
		{"multiple cases", "int f() { switch (x) { case 1: return 1; case 2: return 2; default: return 0; } }", 3},
		{"fallthrough", "int f() { switch (x) { case 1: case 2: return 2; } }", 2},
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
			if len(funDef.Body.Items) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(funDef.Body.Items))
			}

			switchStmt, ok := funDef.Body.Items[0].(cabs.Switch)
			if !ok {
				t.Fatalf("expected Switch, got %T", funDef.Body.Items[0])
			}

			if len(switchStmt.Cases) != tt.numCases {
				t.Errorf("expected %d cases, got %d", tt.numCases, len(switchStmt.Cases))
			}
		})
	}
}

func TestSwitchWithBreak(t *testing.T) {
	input := `int f() { switch (x) { case 1: x = 1; break; case 2: x = 2; break; default: x = 0; } }`

	l := lexer.New(input)
	p := New(l)
	def := p.ParseDefinition()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	funDef := def.(cabs.FunDef)
	switchStmt := funDef.Body.Items[0].(cabs.Switch)

	if len(switchStmt.Cases) != 3 {
		t.Fatalf("expected 3 cases, got %d", len(switchStmt.Cases))
	}

	// First case should have 2 statements (assignment and break)
	if len(switchStmt.Cases[0].Stmts) != 2 {
		t.Errorf("case 0: expected 2 statements, got %d", len(switchStmt.Cases[0].Stmts))
	}

	// Verify first case expression is 1
	c0Expr := switchStmt.Cases[0].Expr.(cabs.Constant)
	if c0Expr.Value != 1 {
		t.Errorf("case 0 expr: expected 1, got %d", c0Expr.Value)
	}

	// Default case (last) should have Expr == nil
	if switchStmt.Cases[2].Expr != nil {
		t.Error("default case should have nil Expr")
	}
}

func TestGotoStatement(t *testing.T) {
	input := `int f() { goto done; x = 1; done: return 0; }`

	l := lexer.New(input)
	p := New(l)
	def := p.ParseDefinition()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	funDef := def.(cabs.FunDef)
	if len(funDef.Body.Items) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(funDef.Body.Items))
	}

	// First statement is goto
	gotoStmt, ok := funDef.Body.Items[0].(cabs.Goto)
	if !ok {
		t.Fatalf("expected Goto, got %T", funDef.Body.Items[0])
	}
	if gotoStmt.Label != "done" {
		t.Errorf("expected label 'done', got %q", gotoStmt.Label)
	}

	// Third statement is a label
	labelStmt, ok := funDef.Body.Items[2].(cabs.Label)
	if !ok {
		t.Fatalf("expected Label, got %T", funDef.Body.Items[2])
	}
	if labelStmt.Name != "done" {
		t.Errorf("expected label name 'done', got %q", labelStmt.Name)
	}

	// Label should wrap a return statement
	_, ok = labelStmt.Stmt.(cabs.Return)
	if !ok {
		t.Fatalf("expected Return inside label, got %T", labelStmt.Stmt)
	}
}

func TestLabelStatement(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		labelName string
	}{
		{"simple label", "int f() { loop: x++; }", "loop"},
		{"label with return", "int f() { end: return 0; }", "end"},
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
			labelStmt, ok := funDef.Body.Items[0].(cabs.Label)
			if !ok {
				t.Fatalf("expected Label, got %T", funDef.Body.Items[0])
			}

			if labelStmt.Name != tt.labelName {
				t.Errorf("expected label name %q, got %q", tt.labelName, labelStmt.Name)
			}
		})
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

func TestVariableDeclaration(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		declCount int
		typeName  string
		varName   string
		hasInit   bool
	}{
		{
			name:      "simple int declaration",
			input:     `int f() { int x; return 0; }`,
			declCount: 1,
			typeName:  "int",
			varName:   "x",
			hasInit:   false,
		},
		{
			name:      "declaration with initializer",
			input:     `int f() { int x = 1; return 0; }`,
			declCount: 1,
			typeName:  "int",
			varName:   "x",
			hasInit:   true,
		},
		{
			name:      "multiple declarations",
			input:     `int f() { int x, y; return 0; }`,
			declCount: 2,
			typeName:  "int",
			varName:   "x", // first decl
			hasInit:   false,
		},
		{
			name:      "pointer declaration",
			input:     `int f() { int *p; return 0; }`,
			declCount: 1,
			typeName:  "int*",
			varName:   "p",
			hasInit:   false,
		},
		{
			name:      "char declaration",
			input:     `int f() { char c; return 0; }`,
			declCount: 1,
			typeName:  "char",
			varName:   "c",
			hasInit:   false,
		},
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
			declStmt, ok := funDef.Body.Items[0].(cabs.DeclStmt)
			if !ok {
				t.Fatalf("expected DeclStmt, got %T", funDef.Body.Items[0])
			}

			if len(declStmt.Decls) != tt.declCount {
				t.Errorf("expected %d declarations, got %d", tt.declCount, len(declStmt.Decls))
			}

			if declStmt.Decls[0].TypeSpec != tt.typeName {
				t.Errorf("expected type %q, got %q", tt.typeName, declStmt.Decls[0].TypeSpec)
			}

			if declStmt.Decls[0].Name != tt.varName {
				t.Errorf("expected name %q, got %q", tt.varName, declStmt.Decls[0].Name)
			}

			if tt.hasInit && declStmt.Decls[0].Initializer == nil {
				t.Error("expected initializer, got nil")
			}
			if !tt.hasInit && declStmt.Decls[0].Initializer != nil {
				t.Error("expected no initializer")
			}
		})
	}
}

func TestFunctionParameters(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		paramCount int
		params     []struct {
			typeSpec string
			name     string
		}
	}{
		{
			name:       "no parameters",
			input:      `int f() { return 0; }`,
			paramCount: 0,
			params:     nil,
		},
		{
			name:       "void parameter",
			input:      `int f(void) { return 0; }`,
			paramCount: 0,
			params:     nil,
		},
		{
			name:       "one parameter",
			input:      `int f(int x) { return 0; }`,
			paramCount: 1,
			params: []struct {
				typeSpec string
				name     string
			}{
				{"int", "x"},
			},
		},
		{
			name:       "two parameters",
			input:      `int add(int a, int b) { return 0; }`,
			paramCount: 2,
			params: []struct {
				typeSpec string
				name     string
			}{
				{"int", "a"},
				{"int", "b"},
			},
		},
		{
			name:       "pointer parameter",
			input:      `int f(int *p) { return 0; }`,
			paramCount: 1,
			params: []struct {
				typeSpec string
				name     string
			}{
				{"int*", "p"},
			},
		},
		{
			name:       "mixed parameters",
			input:      `int f(int x, char *s) { return 0; }`,
			paramCount: 2,
			params: []struct {
				typeSpec string
				name     string
			}{
				{"int", "x"},
				{"char*", "s"},
			},
		},
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
			if len(funDef.Params) != tt.paramCount {
				t.Fatalf("expected %d parameters, got %d", tt.paramCount, len(funDef.Params))
			}

			for i, expected := range tt.params {
				if funDef.Params[i].TypeSpec != expected.typeSpec {
					t.Errorf("param %d type: expected %q, got %q", i, expected.typeSpec, funDef.Params[i].TypeSpec)
				}
				if funDef.Params[i].Name != expected.name {
					t.Errorf("param %d name: expected %q, got %q", i, expected.name, funDef.Params[i].Name)
				}
			}
		})
	}
}

func TestTypeQualifiersInDeclaration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		typeName string
		varName  string
	}{
		{
			name:     "const declaration",
			input:    `int f() { const int x = 1; return 0; }`,
			typeName: "int",
			varName:  "x",
		},
		{
			name:     "volatile declaration",
			input:    `int f() { volatile int x; return 0; }`,
			typeName: "int",
			varName:  "x",
		},
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
			declStmt, ok := funDef.Body.Items[0].(cabs.DeclStmt)
			if !ok {
				t.Fatalf("expected DeclStmt, got %T", funDef.Body.Items[0])
			}

			if declStmt.Decls[0].TypeSpec != tt.typeName {
				t.Errorf("expected type %q, got %q", tt.typeName, declStmt.Decls[0].TypeSpec)
			}

			if declStmt.Decls[0].Name != tt.varName {
				t.Errorf("expected name %q, got %q", tt.varName, declStmt.Decls[0].Name)
			}
		})
	}
}

func TestStorageClassSpecifiers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		typeName string
		varName  string
	}{
		{
			name:     "static declaration",
			input:    `int f() { static int x; return 0; }`,
			typeName: "int",
			varName:  "x",
		},
		{
			name:     "auto declaration",
			input:    `int f() { auto int x; return 0; }`,
			typeName: "int",
			varName:  "x",
		},
		{
			name:     "register declaration",
			input:    `int f() { register int x; return 0; }`,
			typeName: "int",
			varName:  "x",
		},
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
			declStmt, ok := funDef.Body.Items[0].(cabs.DeclStmt)
			if !ok {
				t.Fatalf("expected DeclStmt, got %T", funDef.Body.Items[0])
			}

			if declStmt.Decls[0].TypeSpec != tt.typeName {
				t.Errorf("expected type %q, got %q", tt.typeName, declStmt.Decls[0].TypeSpec)
			}

			if declStmt.Decls[0].Name != tt.varName {
				t.Errorf("expected name %q, got %q", tt.varName, declStmt.Decls[0].Name)
			}
		})
	}
}

func TestArrayDeclaration(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		typeName  string
		varName   string
		numDims   int
		dimValues []int64 // expected dimension values (constants)
	}{
		{
			name:      "simple array",
			input:     `int f() { int arr[10]; return 0; }`,
			typeName:  "int",
			varName:   "arr",
			numDims:   1,
			dimValues: []int64{10},
		},
		{
			name:      "char array",
			input:     `int f() { char buf[256]; return 0; }`,
			typeName:  "char",
			varName:   "buf",
			numDims:   1,
			dimValues: []int64{256},
		},
		{
			name:      "multi-dimensional array",
			input:     `int f() { int matrix[3][4]; return 0; }`,
			typeName:  "int",
			varName:   "matrix",
			numDims:   2,
			dimValues: []int64{3, 4},
		},
		{
			name:      "3d array",
			input:     `int f() { int cube[2][3][4]; return 0; }`,
			typeName:  "int",
			varName:   "cube",
			numDims:   3,
			dimValues: []int64{2, 3, 4},
		},
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
			declStmt, ok := funDef.Body.Items[0].(cabs.DeclStmt)
			if !ok {
				t.Fatalf("expected DeclStmt, got %T", funDef.Body.Items[0])
			}

			decl := declStmt.Decls[0]
			if decl.TypeSpec != tt.typeName {
				t.Errorf("expected type %q, got %q", tt.typeName, decl.TypeSpec)
			}

			if decl.Name != tt.varName {
				t.Errorf("expected name %q, got %q", tt.varName, decl.Name)
			}

			if len(decl.ArrayDims) != tt.numDims {
				t.Errorf("expected %d array dimensions, got %d", tt.numDims, len(decl.ArrayDims))
			}

			for i, expectedVal := range tt.dimValues {
				if i >= len(decl.ArrayDims) {
					break
				}
				dim := decl.ArrayDims[i]
				if dim == nil {
					t.Errorf("dimension %d: expected constant %d, got nil", i, expectedVal)
					continue
				}
				constant, ok := dim.(cabs.Constant)
				if !ok {
					t.Errorf("dimension %d: expected Constant, got %T", i, dim)
					continue
				}
				if constant.Value != expectedVal {
					t.Errorf("dimension %d: expected %d, got %d", i, expectedVal, constant.Value)
				}
			}
		})
	}
}

func TestVariableLengthArray(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		typeName string
		varName  string
		numDims  int
	}{
		{
			name:     "VLA with variable size",
			input:    `int f(int n) { int arr[n]; return 0; }`,
			typeName: "int",
			varName:  "arr",
			numDims:  1,
		},
		{
			name:     "VLA with expression size",
			input:    `int f(int n) { int arr[n + 1]; return 0; }`,
			typeName: "int",
			varName:  "arr",
			numDims:  1,
		},
		{
			name:     "VLA with multiplication",
			input:    `int f(int n, int m) { int arr[n * m]; return 0; }`,
			typeName: "int",
			varName:  "arr",
			numDims:  1,
		},
		{
			name:     "2D VLA",
			input:    `int f(int n, int m) { int matrix[n][m]; return 0; }`,
			typeName: "int",
			varName:  "matrix",
			numDims:  2,
		},
		{
			name:     "mixed VLA and constant",
			input:    `int f(int n) { int arr[n][10]; return 0; }`,
			typeName: "int",
			varName:  "arr",
			numDims:  2,
		},
		{
			name:     "empty array dimension",
			input:    `int f() { int arr[]; return 0; }`,
			typeName: "int",
			varName:  "arr",
			numDims:  1,
		},
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
			// Declarations may be at different positions depending on params
			var declStmt cabs.DeclStmt
			found := false
			for _, item := range funDef.Body.Items {
				if ds, ok := item.(cabs.DeclStmt); ok {
					declStmt = ds
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("no DeclStmt found in function body")
			}

			decl := declStmt.Decls[0]
			if decl.TypeSpec != tt.typeName {
				t.Errorf("expected type %q, got %q", tt.typeName, decl.TypeSpec)
			}

			if decl.Name != tt.varName {
				t.Errorf("expected name %q, got %q", tt.varName, decl.Name)
			}

			if len(decl.ArrayDims) != tt.numDims {
				t.Errorf("expected %d array dimensions, got %d", tt.numDims, len(decl.ArrayDims))
			}
		})
	}
}

func TestVLADimensionExpressions(t *testing.T) {
	// Test that VLA dimensions are correctly parsed as expressions
	input := `int f(int n) { int arr[n + 1]; return 0; }`

	l := lexer.New(input)
	p := New(l)
	def := p.ParseDefinition()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	funDef := def.(cabs.FunDef)
	declStmt, ok := funDef.Body.Items[0].(cabs.DeclStmt)
	if !ok {
		t.Fatalf("expected DeclStmt, got %T", funDef.Body.Items[0])
	}

	decl := declStmt.Decls[0]
	if len(decl.ArrayDims) != 1 {
		t.Fatalf("expected 1 dimension, got %d", len(decl.ArrayDims))
	}

	// The dimension should be a Binary expression (n + 1)
	binary, ok := decl.ArrayDims[0].(cabs.Binary)
	if !ok {
		t.Fatalf("expected Binary expression for VLA dimension, got %T", decl.ArrayDims[0])
	}

	if binary.Op != cabs.OpAdd {
		t.Errorf("expected OpAdd, got %v", binary.Op)
	}

	// Left side should be variable 'n'
	if v, ok := binary.Left.(cabs.Variable); !ok || v.Name != "n" {
		t.Errorf("expected Variable 'n' on left, got %T", binary.Left)
	}

	// Right side should be constant 1
	if c, ok := binary.Right.(cabs.Constant); !ok || c.Value != 1 {
		t.Errorf("expected Constant 1 on right, got %T", binary.Right)
	}
}

func TestPointerDeclaration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		typeName string
		varName  string
	}{
		{
			name:     "simple pointer",
			input:    `int f() { int *p; return 0; }`,
			typeName: "int*",
			varName:  "p",
		},
		{
			name:     "double pointer",
			input:    `int f() { int **pp; return 0; }`,
			typeName: "int**",
			varName:  "pp",
		},
		{
			name:     "void pointer",
			input:    `int f() { void *vp; return 0; }`,
			typeName: "void*",
			varName:  "vp",
		},
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
			declStmt, ok := funDef.Body.Items[0].(cabs.DeclStmt)
			if !ok {
				t.Fatalf("expected DeclStmt, got %T", funDef.Body.Items[0])
			}

			if declStmt.Decls[0].TypeSpec != tt.typeName {
				t.Errorf("expected type %q, got %q", tt.typeName, declStmt.Decls[0].TypeSpec)
			}

			if declStmt.Decls[0].Name != tt.varName {
				t.Errorf("expected name %q, got %q", tt.varName, declStmt.Decls[0].Name)
			}
		})
	}
}

func TestTypedefDeclaration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		typeName string
		defName  string
	}{
		{
			name:     "simple typedef",
			input:    `typedef int myint; int f() { return 0; }`,
			typeName: "int",
			defName:  "myint",
		},
		{
			name:     "pointer typedef",
			input:    `typedef int* intptr; int f() { return 0; }`,
			typeName: "int*",
			defName:  "intptr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			typedefDef, ok := def.(cabs.TypedefDef)
			if !ok {
				t.Fatalf("expected TypedefDef, got %T", def)
			}

			if typedefDef.TypeSpec != tt.typeName {
				t.Errorf("expected type %q, got %q", tt.typeName, typedefDef.TypeSpec)
			}

			if typedefDef.Name != tt.defName {
				t.Errorf("expected name %q, got %q", tt.defName, typedefDef.Name)
			}
		})
	}
}

func TestTypedefUse(t *testing.T) {
	// Test that typedef names are recognized as types in subsequent parsing
	input := `typedef int myint; myint f() { return 0; }`

	l := lexer.New(input)
	p := New(l)

	// First parse typedef
	def1 := p.ParseDefinition()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors on typedef: %v", p.Errors())
	}

	_, ok := def1.(cabs.TypedefDef)
	if !ok {
		t.Fatalf("first def should be TypedefDef, got %T", def1)
	}

	// Parse function using typedef
	def2 := p.ParseDefinition()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors on function: %v", p.Errors())
	}

	funDef, ok := def2.(cabs.FunDef)
	if !ok {
		t.Fatalf("second def should be FunDef, got %T", def2)
	}

	if funDef.ReturnType != "myint" {
		t.Errorf("expected return type 'myint', got %q", funDef.ReturnType)
	}
}

func TestTypedefInlineStructUnion(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		defName    string
		isUnion    bool
		fieldCount int
		fields     []struct {
			typeSpec string
			name     string
		}
	}{
		{
			name:       "anonymous inline struct typedef",
			input:      `typedef struct { int x; int y; } Point;`,
			defName:    "Point",
			isUnion:    false,
			fieldCount: 2,
			fields: []struct {
				typeSpec string
				name     string
			}{
				{"int", "x"},
				{"int", "y"},
			},
		},
		{
			name:       "anonymous inline union typedef",
			input:      `typedef union { char bytes[4]; int value; } IntBytes;`,
			defName:    "IntBytes",
			isUnion:    true,
			fieldCount: 2,
			fields: []struct {
				typeSpec string
				name     string
			}{
				{"char[]", "bytes"},
				{"int", "value"},
			},
		},
		{
			name:       "mbstate_t-like union typedef",
			input:      `typedef union { char __mbstate8[128]; long long _mbstateL; } __mbstate_t;`,
			defName:    "__mbstate_t",
			isUnion:    true,
			fieldCount: 2,
			fields: []struct {
				typeSpec string
				name     string
			}{
				{"char[]", "__mbstate8"},
				{"long long", "_mbstateL"},
			},
		},
		{
			name:       "typedef struct with tag name",
			input:      `typedef struct _Point { int x; int y; } Point;`,
			defName:    "Point",
			isUnion:    false,
			fieldCount: 2,
			fields: []struct {
				typeSpec string
				name     string
			}{
				{"int", "x"},
				{"int", "y"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			typedefDef, ok := def.(cabs.TypedefDef)
			if !ok {
				t.Fatalf("expected TypedefDef, got %T", def)
			}

			if typedefDef.Name != tt.defName {
				t.Errorf("expected name %q, got %q", tt.defName, typedefDef.Name)
			}

			if typedefDef.InlineType == nil {
				t.Fatal("expected InlineType to be set, got nil")
			}

			if tt.isUnion {
				unionDef, ok := typedefDef.InlineType.(cabs.UnionDef)
				if !ok {
					t.Fatalf("expected UnionDef inline, got %T", typedefDef.InlineType)
				}

				if len(unionDef.Fields) != tt.fieldCount {
					t.Fatalf("expected %d fields, got %d", tt.fieldCount, len(unionDef.Fields))
				}

				for i, expected := range tt.fields {
					if unionDef.Fields[i].TypeSpec != expected.typeSpec {
						t.Errorf("field %d: expected type %q, got %q", i, expected.typeSpec, unionDef.Fields[i].TypeSpec)
					}
					if unionDef.Fields[i].Name != expected.name {
						t.Errorf("field %d: expected name %q, got %q", i, expected.name, unionDef.Fields[i].Name)
					}
				}
			} else {
				structDef, ok := typedefDef.InlineType.(cabs.StructDef)
				if !ok {
					t.Fatalf("expected StructDef inline, got %T", typedefDef.InlineType)
				}

				if len(structDef.Fields) != tt.fieldCount {
					t.Fatalf("expected %d fields, got %d", tt.fieldCount, len(structDef.Fields))
				}

				for i, expected := range tt.fields {
					if structDef.Fields[i].TypeSpec != expected.typeSpec {
						t.Errorf("field %d: expected type %q, got %q", i, expected.typeSpec, structDef.Fields[i].TypeSpec)
					}
					if structDef.Fields[i].Name != expected.name {
						t.Errorf("field %d: expected name %q, got %q", i, expected.name, structDef.Fields[i].Name)
					}
				}
			}
		})
	}
}

func TestTypedefInlineEnum(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		defName    string
		valueCount int
		values     []string
	}{
		{
			name:       "anonymous inline enum typedef",
			input:      `typedef enum { RED, GREEN, BLUE } Color;`,
			defName:    "Color",
			valueCount: 3,
			values:     []string{"RED", "GREEN", "BLUE"},
		},
		{
			name:       "inline enum typedef with values",
			input:      `typedef enum { NONE = 0, SOME = 1, ALL = 2 } Level;`,
			defName:    "Level",
			valueCount: 3,
			values:     []string{"NONE", "SOME", "ALL"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			typedefDef, ok := def.(cabs.TypedefDef)
			if !ok {
				t.Fatalf("expected TypedefDef, got %T", def)
			}

			if typedefDef.Name != tt.defName {
				t.Errorf("expected name %q, got %q", tt.defName, typedefDef.Name)
			}

			if typedefDef.InlineType == nil {
				t.Fatal("expected InlineType to be set, got nil")
			}

			enumDef, ok := typedefDef.InlineType.(cabs.EnumDef)
			if !ok {
				t.Fatalf("expected EnumDef inline, got %T", typedefDef.InlineType)
			}

			if len(enumDef.Values) != tt.valueCount {
				t.Fatalf("expected %d values, got %d", tt.valueCount, len(enumDef.Values))
			}

			for i, expected := range tt.values {
				if enumDef.Values[i].Name != expected {
					t.Errorf("value %d: expected %q, got %q", i, expected, enumDef.Values[i].Name)
				}
			}
		})
	}
}

func TestStructDefinition(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		structName string
		fieldCount int
		fields     []struct {
			typeSpec string
			name     string
		}
	}{
		{
			name:       "simple struct",
			input:      `struct Point { int x; int y; };`,
			structName: "Point",
			fieldCount: 2,
			fields: []struct {
				typeSpec string
				name     string
			}{
				{"int", "x"},
				{"int", "y"},
			},
		},
		{
			name:       "struct with pointer field",
			input:      `struct Node { int value; int *next; };`,
			structName: "Node",
			fieldCount: 2,
			fields: []struct {
				typeSpec string
				name     string
			}{
				{"int", "value"},
				{"int*", "next"},
			},
		},
		{
			name:       "anonymous struct",
			input:      `struct { int x; };`,
			structName: "",
			fieldCount: 1,
			fields: []struct {
				typeSpec string
				name     string
			}{
				{"int", "x"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			structDef, ok := def.(cabs.StructDef)
			if !ok {
				t.Fatalf("expected StructDef, got %T", def)
			}

			if structDef.Name != tt.structName {
				t.Errorf("expected name %q, got %q", tt.structName, structDef.Name)
			}

			if len(structDef.Fields) != tt.fieldCount {
				t.Fatalf("expected %d fields, got %d", tt.fieldCount, len(structDef.Fields))
			}

			for i, expected := range tt.fields {
				if structDef.Fields[i].TypeSpec != expected.typeSpec {
					t.Errorf("field %d type: expected %q, got %q", i, expected.typeSpec, structDef.Fields[i].TypeSpec)
				}
				if structDef.Fields[i].Name != expected.name {
					t.Errorf("field %d name: expected %q, got %q", i, expected.name, structDef.Fields[i].Name)
				}
			}
		})
	}
}

func TestUnionDefinition(t *testing.T) {
	input := `union Value { int i; float f; };`

	l := lexer.New(input)
	p := New(l)
	def := p.ParseDefinition()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	unionDef, ok := def.(cabs.UnionDef)
	if !ok {
		t.Fatalf("expected UnionDef, got %T", def)
	}

	if unionDef.Name != "Value" {
		t.Errorf("expected name 'Value', got %q", unionDef.Name)
	}

	if len(unionDef.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(unionDef.Fields))
	}

	if unionDef.Fields[0].TypeSpec != "int" || unionDef.Fields[0].Name != "i" {
		t.Errorf("unexpected first field: %v", unionDef.Fields[0])
	}

	if unionDef.Fields[1].TypeSpec != "float" || unionDef.Fields[1].Name != "f" {
		t.Errorf("unexpected second field: %v", unionDef.Fields[1])
	}
}

func TestEnumDefinition(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		enumName   string
		valueCount int
	}{
		{
			name:       "simple enum",
			input:      `enum Color { RED, GREEN, BLUE };`,
			enumName:   "Color",
			valueCount: 3,
		},
		{
			name:       "enum with explicit values",
			input:      `enum Status { OK = 0, ERROR = 1 };`,
			enumName:   "Status",
			valueCount: 2,
		},
		{
			name:       "anonymous enum",
			input:      `enum { A, B, C };`,
			enumName:   "",
			valueCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			enumDef, ok := def.(cabs.EnumDef)
			if !ok {
				t.Fatalf("expected EnumDef, got %T", def)
			}

			if enumDef.Name != tt.enumName {
				t.Errorf("expected name %q, got %q", tt.enumName, enumDef.Name)
			}

			if len(enumDef.Values) != tt.valueCount {
				t.Errorf("expected %d values, got %d", tt.valueCount, len(enumDef.Values))
			}
		})
	}
}

func TestParseProgram(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedDefs  int
		expectedTypes []string // types of definitions: "FunDef", "TypedefDef", "StructDef", etc.
	}{
		{
			name:          "single function",
			input:         `int main() { return 0; }`,
			expectedDefs:  1,
			expectedTypes: []string{"FunDef"},
		},
		{
			name: "two functions",
			input: `int add(int a, int b) { return a + b; }
                    int main() { return 0; }`,
			expectedDefs:  2,
			expectedTypes: []string{"FunDef", "FunDef"},
		},
		{
			name: "typedef and function",
			input: `typedef int myint;
                    myint f() { return 0; }`,
			expectedDefs:  2,
			expectedTypes: []string{"TypedefDef", "FunDef"},
		},
		{
			name: "struct and function",
			input: `struct Point { int x; int y; };
                    int main() { return 0; }`,
			expectedDefs:  2,
			expectedTypes: []string{"StructDef", "FunDef"},
		},
		{
			name: "multiple definitions",
			input: `typedef int myint;
                    struct Point { int x; int y; };
                    enum Color { RED, GREEN, BLUE };
                    union Value { int i; float f; };
                    int helper() { return 1; }
                    int main() { return 0; }`,
			expectedDefs:  6,
			expectedTypes: []string{"TypedefDef", "StructDef", "EnumDef", "UnionDef", "FunDef", "FunDef"},
		},
		{
			name:          "empty program",
			input:         ``,
			expectedDefs:  0,
			expectedTypes: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			if len(program.Definitions) != tt.expectedDefs {
				t.Fatalf("expected %d definitions, got %d", tt.expectedDefs, len(program.Definitions))
			}

			for i, expectedType := range tt.expectedTypes {
				actualType := defTypeName(program.Definitions[i])
				if actualType != expectedType {
					t.Errorf("definition %d: expected %s, got %s", i, expectedType, actualType)
				}
			}
		})
	}
}

func defTypeName(def cabs.Definition) string {
	switch def.(type) {
	case cabs.FunDef:
		return "FunDef"
	case cabs.TypedefDef:
		return "TypedefDef"
	case cabs.StructDef:
		return "StructDef"
	case cabs.UnionDef:
		return "UnionDef"
	case cabs.EnumDef:
		return "EnumDef"
	default:
		return fmt.Sprintf("unknown(%T)", def)
	}
}

func TestFunctionPointerDeclaration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		typeName string
		varName  string
	}{
		{
			name:     "simple function pointer",
			input:    `int f() { int (*fp)(int, int); return 0; }`,
			typeName: "int(*)(int,int)",
			varName:  "fp",
		},
		{
			name:     "void function pointer",
			input:    `int f() { void (*handler)(void); return 0; }`,
			typeName: "void(*)(void)",
			varName:  "handler",
		},
		{
			name:     "no args function pointer",
			input:    `int f() { int (*getter)(); return 0; }`,
			typeName: "int(*)()",
			varName:  "getter",
		},
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
			declStmt, ok := funDef.Body.Items[0].(cabs.DeclStmt)
			if !ok {
				t.Fatalf("expected DeclStmt, got %T", funDef.Body.Items[0])
			}

			if declStmt.Decls[0].TypeSpec != tt.typeName {
				t.Errorf("expected type %q, got %q", tt.typeName, declStmt.Decls[0].TypeSpec)
			}

			if declStmt.Decls[0].Name != tt.varName {
				t.Errorf("expected name %q, got %q", tt.varName, declStmt.Decls[0].Name)
			}
		})
	}
}


// Error Recovery Tests

func TestErrorRecoveryInBlock(t *testing.T) {
	// Test that parser continues after an error in a statement
	input := `int f() {
		int x = 1;
		@ invalid syntax;
		int y = 2;
		return y;
	}`

	l := lexer.New(input)
	p := New(l)
	def := p.ParseDefinition()

	// Should have errors
	if len(p.Errors()) == 0 {
		t.Fatal("expected parser errors, got none")
	}

	// But should still parse the function
	if def == nil {
		t.Fatal("expected function definition despite errors")
	}

	funDef, ok := def.(cabs.FunDef)
	if !ok {
		t.Fatalf("expected FunDef, got %T", def)
	}

	// Should have recovered and parsed valid statements after the error
	// We expect at least the first declaration and possibly the last return
	if len(funDef.Body.Items) == 0 {
		t.Error("expected at least one statement to be parsed")
	}
}

func TestErrorRecoveryMultipleErrors(t *testing.T) {
	// Test that multiple errors are reported
	input := `int f() {
		@ error1;
		$ error2;
		return 1;
	}`

	l := lexer.New(input)
	p := New(l)
	_ = p.ParseDefinition()

	// Should report multiple errors
	if len(p.Errors()) < 2 {
		t.Errorf("expected at least 2 errors, got %d: %v", len(p.Errors()), p.Errors())
	}
}

func TestErrorRecoveryParseSecondFunction(t *testing.T) {
	// Test that second function is parsed even if first has errors
	input := `int broken() {
		@ invalid;
		return 1;
	}
	int valid() {
		return 42;
	}`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()

	// Should have errors from first function
	if len(p.Errors()) == 0 {
		t.Fatal("expected parser errors from first function")
	}

	// Should still parse both functions
	if len(program.Definitions) != 2 {
		t.Errorf("expected 2 definitions, got %d", len(program.Definitions))
	}

	// Second function should be valid
	if len(program.Definitions) >= 2 {
		funDef, ok := program.Definitions[1].(cabs.FunDef)
		if !ok {
			t.Fatalf("expected second definition to be FunDef, got %T", program.Definitions[1])
		}
		if funDef.Name != "valid" {
			t.Errorf("expected second function name 'valid', got %q", funDef.Name)
		}
	}
}

func TestErrorRecoveryContinuesAfterMissingSemicolon(t *testing.T) {
	// Missing semicolon should not stop parsing subsequent statements
	input := `int f() {
		int x = 1
		int y = 2;
		return y;
	}`

	l := lexer.New(input)
	p := New(l)
	def := p.ParseDefinition()

	// Should have error about missing semicolon
	if len(p.Errors()) == 0 {
		t.Fatal("expected parser error for missing semicolon")
	}

	if def == nil {
		t.Fatal("expected function definition despite error")
	}

	funDef, ok := def.(cabs.FunDef)
	if !ok {
		t.Fatalf("expected FunDef, got %T", def)
	}

	// Should have parsed something
	if len(funDef.Body.Items) == 0 {
		t.Error("expected at least one statement to be parsed")
	}
}

func TestErrorRecoverySyncToBlockEnd(t *testing.T) {
	// Test recovery with nested blocks
	input := `int f() {
		if (1) {
			@ invalid in nested block;
			return 1;
		}
		return 2;
	}`

	l := lexer.New(input)
	p := New(l)
	def := p.ParseDefinition()

	// Should have errors
	if len(p.Errors()) == 0 {
		t.Fatal("expected parser errors")
	}

	// But should still produce a function
	if def == nil {
		t.Fatal("expected function definition despite errors")
	}
}

func TestStructParameterType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"struct parameter",
			"int getx(struct Point p) { return p.x; }",
			"struct Point",
		},
		{
			"struct pointer parameter",
			"int getx(struct Point *p) { return p->x; }",
			"struct Point*",
		},
		{
			"union parameter",
			"int getu(union Data d) { return d.i; }",
			"union Data",
		},
		{
			"enum parameter",
			"int gete(enum Color c) { return c; }",
			"enum Color",
		},
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
			if len(funDef.Params) != 1 {
				t.Fatalf("expected 1 parameter, got %d", len(funDef.Params))
			}

			if funDef.Params[0].TypeSpec != tt.expected {
				t.Errorf("expected type %q, got %q", tt.expected, funDef.Params[0].TypeSpec)
			}
		})
	}
}

func TestStructReturnType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"struct pointer return",
			"struct Point *getp(int x) { return 0; }",
			"struct Point*",
		},
		{
			"struct return",
			"struct Point getp(int x) { struct Point p; return p; }",
			"struct Point",
		},
		{
			"union pointer return",
			"union Data *getd(int x) { return 0; }",
			"union Data*",
		},
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
			if funDef.ReturnType != tt.expected {
				t.Errorf("expected return type %q, got %q", tt.expected, funDef.ReturnType)
			}
		})
	}
}

func TestC99ForLoopDeclaration(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		numDecls  int
		declName  string
		declType  string
		hasInit   bool
	}{
		{
			"simple c99 for",
			"int f() { for (int i = 0; i < 10; i++) x++; }",
			1, "i", "int", true,
		},
		{
			"c99 for without init",
			"int f() { for (int i; i < 10; i++) x++; }",
			1, "i", "int", false,
		},
		{
			"c99 for with pointer",
			"int f() { for (int *p = arr; p < end; p++) x++; }",
			1, "p", "int*", true,
		},
		{
			"c99 for multiple vars",
			"int f() { for (int i = 0, j = 10; i < j; i++, j--) x++; }",
			2, "i", "int", true,
		},
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
			forStmt := funDef.Body.Items[0].(cabs.For)

			// Should have declarations, not expression init
			if len(forStmt.InitDecl) != tt.numDecls {
				t.Fatalf("expected %d declarations, got %d", tt.numDecls, len(forStmt.InitDecl))
			}

			if forStmt.Init != nil {
				t.Errorf("expected Init to be nil for C99 for-loop")
			}

			// Check first declaration
			if forStmt.InitDecl[0].Name != tt.declName {
				t.Errorf("expected decl name %q, got %q", tt.declName, forStmt.InitDecl[0].Name)
			}
			if forStmt.InitDecl[0].TypeSpec != tt.declType {
				t.Errorf("expected decl type %q, got %q", tt.declType, forStmt.InitDecl[0].TypeSpec)
			}
			if tt.hasInit && forStmt.InitDecl[0].Initializer == nil {
				t.Errorf("expected initializer")
			}
			if !tt.hasInit && forStmt.InitDecl[0].Initializer != nil {
				t.Errorf("did not expect initializer")
			}
		})
	}
}

func TestStructDefinitionVsReturnType(t *testing.T) {
	// Test that struct definitions are still parsed correctly
	tests := []struct {
		name    string
		input   string
		isStruct bool
	}{
		{
			"struct definition",
			"struct Point { int x; int y; };",
			true,
		},
		{
			"struct forward declaration",
			"struct Point;",
			true,
		},
		{
			"function returning struct pointer",
			"struct Point *make_point(int x) { return 0; }",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			_, isStruct := def.(cabs.StructDef)
			if isStruct != tt.isStruct {
				if tt.isStruct {
					t.Errorf("expected StructDef, got %T", def)
				} else {
					t.Errorf("expected FunDef, got %T", def)
				}
			}
		})
	}
}

func TestFunctionPointerInStructField(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		fieldName  string
		fieldType  string
		numFields  int
	}{
		{
			"simple function pointer field",
			"struct S { int (*f)(void); };",
			"f",
			"int(*)(void)",
			1,
		},
		{
			"function pointer with params",
			"struct S { int (*_read)(void *, char *, int); };",
			"_read",
			"int(*)(void*, char*, int)",
			1,
		},
		{
			"function pointer returning void",
			"struct S { void (*callback)(int); };",
			"callback",
			"void(*)(int)",
			1,
		},
		{
			"function pointer with regular fields",
			"struct FILE { int x; int (*_read)(void *, char *, int); int y; };",
			"_read",
			"int(*)(void*, char*, int)",
			3,
		},
		{
			"function pointer returning pointer",
			"struct S { char *(*getstr)(void); };",
			"getstr",
			"char*(*)(void)",
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			structDef, ok := def.(cabs.StructDef)
			if !ok {
				t.Fatalf("expected StructDef, got %T", def)
			}

			if len(structDef.Fields) != tt.numFields {
				t.Fatalf("expected %d fields, got %d", tt.numFields, len(structDef.Fields))
			}

			// Find the function pointer field
			var found bool
			for _, field := range structDef.Fields {
				if field.Name == tt.fieldName {
					found = true
					if field.TypeSpec != tt.fieldType {
						t.Errorf("expected field type %q, got %q", tt.fieldType, field.TypeSpec)
					}
					break
				}
			}
			if !found {
				t.Errorf("field %q not found in struct", tt.fieldName)
			}
		})
	}
}

func TestVariadicFunctionDeclaration(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		numParams  int
		isVariadic bool
	}{
		{
			"printf-like variadic function",
			"int printf(const char *fmt, ...) { return 0; }",
			1,
			true,
		},
		{
			"variadic with multiple params",
			"int foo(int a, int b, ...) { return 0; }",
			2,
			true,
		},
		{
			"non-variadic function",
			"int bar(int x, int y) { return x + y; }",
			2,
			false,
		},
		{
			"variadic with single param",
			"void log(const char *msg, ...) {}",
			1,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			funDef, ok := def.(cabs.FunDef)
			if !ok {
				t.Fatalf("expected FunDef, got %T", def)
			}

			if len(funDef.Params) != tt.numParams {
				t.Errorf("expected %d params, got %d", tt.numParams, len(funDef.Params))
			}

			if funDef.Variadic != tt.isVariadic {
				t.Errorf("expected Variadic=%v, got %v", tt.isVariadic, funDef.Variadic)
			}
		})
	}
}

func TestInlineKeyword(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		funcName   string
		returnType string
		hasBody    bool
	}{
		{
			"inline function declaration",
			"inline int foo(void);",
			"foo",
			"int",
			false,
		},
		{
			"__inline function declaration",
			"__inline int bar(void);",
			"bar",
			"int",
			false,
		},
		{
			"__inline__ function declaration",
			"__inline__ int baz(void);",
			"baz",
			"int",
			false,
		},
		{
			"inline function with body",
			"inline int square(int x) { return x * x; }",
			"square",
			"int",
			true,
		},
		{
			"extern inline function",
			"extern inline int helper(void);",
			"helper",
			"int",
			false,
		},
		{
			"static inline function",
			"static inline int internal(void) { return 1; }",
			"internal",
			"int",
			true,
		},
		{
			"inline with __attribute__",
			"extern __inline __attribute__((always_inline)) int fast_func(int a);",
			"fast_func",
			"int",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			funDef, ok := def.(cabs.FunDef)
			if !ok {
				t.Fatalf("expected FunDef, got %T", def)
			}

			if funDef.Name != tt.funcName {
				t.Errorf("expected function name %q, got %q", tt.funcName, funDef.Name)
			}

			if funDef.ReturnType != tt.returnType {
				t.Errorf("expected return type %q, got %q", tt.returnType, funDef.ReturnType)
			}

			if tt.hasBody && funDef.Body == nil {
				t.Error("expected function to have body, but Body is nil")
			}
			if !tt.hasBody && funDef.Body != nil {
				t.Error("expected function declaration (no body), but Body is not nil")
			}
		})
	}
}

func TestAttributeSkipping(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		funcName   string
		hasBody    bool
		isVariadic bool
	}{
		{
			"function declaration with __attribute__",
			"int printf(const char *, ...) __attribute__((__format__ (__printf__, 1, 2)));",
			"printf",
			false,
			true,
		},
		{
			"function declaration with __asm",
			`int fopen(const char *, const char *) __asm("_fopen");`,
			"fopen",
			false,
			false,
		},
		{
			"function with __asm and __attribute__",
			`int popen(const char *, const char *) __asm("_popen") __attribute__((cold));`,
			"popen",
			false,
			false,
		},
		{
			"leading __attribute__ on function",
			"__attribute__((deprecated)) int old_func(void);",
			"old_func",
			false,
			false,
		},
		{
			"leading __attribute__ on function with body",
			"__attribute__((cold)) int rare_func(void) { return 0; }",
			"rare_func",
			true,
			false,
		},
		{
			"complex nested __attribute__",
			`int getenv_s(int *, char *, int, const char *) __attribute__((availability(macosx,introduced=10.10)));`,
			"getenv_s",
			false,
			false,
		},
		{
			"__asm__ variant",
			`int foo(void) __asm__("_foo");`,
			"foo",
			false,
			false,
		},
		{
			"multiple __attribute__ blocks",
			`int bar(void) __attribute__((cold)) __attribute__((noinline));`,
			"bar",
			false,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			funDef, ok := def.(cabs.FunDef)
			if !ok {
				t.Fatalf("expected FunDef, got %T", def)
			}

			if funDef.Name != tt.funcName {
				t.Errorf("expected function name %q, got %q", tt.funcName, funDef.Name)
			}

			if tt.hasBody && funDef.Body == nil {
				t.Error("expected function to have body, but Body is nil")
			}
			if !tt.hasBody && funDef.Body != nil {
				t.Error("expected function declaration (no body), but Body is not nil")
			}

			if funDef.Variadic != tt.isVariadic {
				t.Errorf("expected Variadic=%v, got %v", tt.isVariadic, funDef.Variadic)
			}
		})
	}
}


func TestGlobalVariableDeclaration(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		storageClass string
		typeSpec     string
		varName      string
		hasArray     bool
		hasInit      bool
	}{
		{
			name:         "extern int variable",
			input:        "extern int sys_nerr;",
			storageClass: "extern",
			typeSpec:     "int",
			varName:      "sys_nerr",
			hasArray:     false,
			hasInit:      false,
		},
		{
			name:         "extern const int variable",
			input:        "extern const int sys_nerr;",
			storageClass: "extern",
			typeSpec:     "int",
			varName:      "sys_nerr",
			hasArray:     false,
			hasInit:      false,
		},
		{
			name:         "extern pointer variable",
			input:        "extern const char *const sys_errlist[];",
			storageClass: "extern",
			typeSpec:     "char*",
			varName:      "sys_errlist",
			hasArray:     true,
			hasInit:      false,
		},
		{
			name:         "static variable with initializer",
			input:        "static int global_count = 0;",
			storageClass: "static",
			typeSpec:     "int",
			varName:      "global_count",
			hasArray:     false,
			hasInit:      true,
		},
		{
			name:         "simple global variable",
			input:        "int counter;",
			storageClass: "",
			typeSpec:     "int",
			varName:      "counter",
			hasArray:     false,
			hasInit:      false,
		},
		{
			name:         "global array with size",
			input:        "int arr[10];",
			storageClass: "",
			typeSpec:     "int",
			varName:      "arr",
			hasArray:     true,
			hasInit:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			varDef, ok := def.(cabs.VarDef)
			if !ok {
				t.Fatalf("expected VarDef, got %T", def)
			}

			if varDef.StorageClass != tt.storageClass {
				t.Errorf("StorageClass: expected %q, got %q", tt.storageClass, varDef.StorageClass)
			}

			if varDef.TypeSpec != tt.typeSpec {
				t.Errorf("TypeSpec: expected %q, got %q", tt.typeSpec, varDef.TypeSpec)
			}

			if varDef.Name != tt.varName {
				t.Errorf("Name: expected %q, got %q", tt.varName, varDef.Name)
			}

			hasArray := len(varDef.ArrayDims) > 0
			if hasArray != tt.hasArray {
				t.Errorf("hasArray: expected %v, got %v", tt.hasArray, hasArray)
			}

			hasInit := varDef.Initializer != nil
			if hasInit != tt.hasInit {
				t.Errorf("hasInit: expected %v, got %v", tt.hasInit, hasInit)
			}
		})
	}
}

func TestTypedefWithArrayDimension(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		typeName string
		typeSpec string
	}{
		{
			name:     "fixed size array typedef",
			input:    "typedef unsigned char uuid_t[16];",
			typeName: "uuid_t",
			typeSpec: "unsigned char[16]",
		},
		{
			name:     "char array typedef",
			input:    "typedef char uuid_string_t[37];",
			typeName: "uuid_string_t",
			typeSpec: "char[37]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			def := p.ParseDefinition()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			typedefDef, ok := def.(cabs.TypedefDef)
			if !ok {
				t.Fatalf("expected TypedefDef, got %T", def)
			}

			if typedefDef.Name != tt.typeName {
				t.Errorf("Name: expected %q, got %q", tt.typeName, typedefDef.Name)
			}

			if typedefDef.TypeSpec != tt.typeSpec {
				t.Errorf("TypeSpec: expected %q, got %q", tt.typeSpec, typedefDef.TypeSpec)
			}
		})
	}
}

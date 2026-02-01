package parser

import (
	"os"
	"testing"

	"github.com/raymyers/ralph-cc/pkg/cabs"
	"github.com/raymyers/ralph-cc/pkg/lexer"
	"gopkg.in/yaml.v3"
)

// TestSpec represents a test case from parse.yaml
type TestSpec struct {
	Name  string     `yaml:"name"`
	Input string     `yaml:"input"`
	AST   ASTSpec    `yaml:"ast"`
}

// ASTSpec represents the expected AST structure
type ASTSpec struct {
	Kind       string    `yaml:"kind"`
	Name       string    `yaml:"name,omitempty"`
	ReturnType string    `yaml:"return_type,omitempty"`
	Body       *ASTSpec  `yaml:"body,omitempty"`
	Items      []ASTSpec `yaml:"items,omitempty"`
	Expr       *ASTSpec  `yaml:"expr,omitempty"`
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

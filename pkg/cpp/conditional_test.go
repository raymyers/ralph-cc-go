package cpp

import (
	"strings"
	"testing"
)

func TestConditionalIfdef(t *testing.T) {
	tests := []struct {
		name     string
		defined  []string
		testName string
		expect   bool
	}{
		{"defined macro", []string{"FOO"}, "FOO", true},
		{"undefined macro", []string{}, "FOO", false},
		{"one of many", []string{"BAR", "FOO", "BAZ"}, "FOO", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := NewMacroTable()
			for _, name := range tt.defined {
				mt.DefineSimple(name, "1", SourceLoc{})
			}

			cp := NewConditionalProcessor(mt)
			if err := cp.ProcessIfdef(tt.testName); err != nil {
				t.Fatalf("ProcessIfdef error: %v", err)
			}

			if cp.IsActive() != tt.expect {
				t.Errorf("IsActive() = %v, want %v", cp.IsActive(), tt.expect)
			}

			if err := cp.ProcessEndif(); err != nil {
				t.Fatalf("ProcessEndif error: %v", err)
			}
		})
	}
}

func TestConditionalIfndef(t *testing.T) {
	tests := []struct {
		name     string
		defined  []string
		testName string
		expect   bool
	}{
		{"undefined macro", []string{}, "FOO", true},
		{"defined macro", []string{"FOO"}, "FOO", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := NewMacroTable()
			for _, name := range tt.defined {
				mt.DefineSimple(name, "1", SourceLoc{})
			}

			cp := NewConditionalProcessor(mt)
			if err := cp.ProcessIfndef(tt.testName); err != nil {
				t.Fatalf("ProcessIfndef error: %v", err)
			}

			if cp.IsActive() != tt.expect {
				t.Errorf("IsActive() = %v, want %v", cp.IsActive(), tt.expect)
			}

			if err := cp.ProcessEndif(); err != nil {
				t.Fatalf("ProcessEndif error: %v", err)
			}
		})
	}
}

func TestConditionalIf(t *testing.T) {
	tests := []struct {
		name    string
		defines map[string]string
		expr    string
		expect  bool
	}{
		{"simple true", nil, "1", true},
		{"simple false", nil, "0", false},
		{"comparison", nil, "1 > 0", true},
		{"defined macro value", map[string]string{"X": "42"}, "X > 0", true},
		{"undefined evaluates to 0", nil, "UNDEFINED", false},
		{"defined operator", map[string]string{"FOO": "1"}, "defined(FOO)", true},
		{"defined operator not", nil, "defined(FOO)", false},
		{"logical and", nil, "1 && 1", true},
		{"logical or", nil, "0 || 1", true},
		{"complex", map[string]string{"X": "5"}, "X >= 5 && X < 10", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := NewMacroTable()
			for name, val := range tt.defines {
				mt.DefineSimple(name, val, SourceLoc{})
			}

			cp := NewConditionalProcessor(mt)
			tokens := tokenize(tt.expr)
			if err := cp.ProcessIf(tokens); err != nil {
				t.Fatalf("ProcessIf error: %v", err)
			}

			if cp.IsActive() != tt.expect {
				t.Errorf("IsActive() = %v, want %v", cp.IsActive(), tt.expect)
			}

			if err := cp.ProcessEndif(); err != nil {
				t.Fatalf("ProcessEndif error: %v", err)
			}
		})
	}
}

func TestConditionalElse(t *testing.T) {
	mt := NewMacroTable()
	cp := NewConditionalProcessor(mt)

	// #ifdef UNDEFINED
	if err := cp.ProcessIfdef("UNDEFINED"); err != nil {
		t.Fatalf("ProcessIfdef error: %v", err)
	}
	if cp.IsActive() {
		t.Error("should be inactive in false branch")
	}

	// #else
	if err := cp.ProcessElse(); err != nil {
		t.Fatalf("ProcessElse error: %v", err)
	}
	if !cp.IsActive() {
		t.Error("should be active in else branch")
	}

	// #endif
	if err := cp.ProcessEndif(); err != nil {
		t.Fatalf("ProcessEndif error: %v", err)
	}
}

func TestConditionalElif(t *testing.T) {
	mt := NewMacroTable()
	mt.DefineSimple("X", "2", SourceLoc{})
	cp := NewConditionalProcessor(mt)

	// #if X == 1
	tokens := tokenize("X == 1")
	if err := cp.ProcessIf(tokens); err != nil {
		t.Fatalf("ProcessIf error: %v", err)
	}
	if cp.IsActive() {
		t.Error("first branch should be inactive")
	}

	// #elif X == 2
	tokens = tokenize("X == 2")
	if err := cp.ProcessElif(tokens); err != nil {
		t.Fatalf("ProcessElif error: %v", err)
	}
	if !cp.IsActive() {
		t.Error("elif branch should be active")
	}

	// #else
	if err := cp.ProcessElse(); err != nil {
		t.Fatalf("ProcessElse error: %v", err)
	}
	if cp.IsActive() {
		t.Error("else branch should be inactive (elif was taken)")
	}

	// #endif
	if err := cp.ProcessEndif(); err != nil {
		t.Fatalf("ProcessEndif error: %v", err)
	}
}

func TestConditionalNested(t *testing.T) {
	mt := NewMacroTable()
	mt.DefineSimple("OUTER", "1", SourceLoc{})
	cp := NewConditionalProcessor(mt)

	// #ifdef OUTER
	if err := cp.ProcessIfdef("OUTER"); err != nil {
		t.Fatalf("ProcessIfdef error: %v", err)
	}
	if !cp.IsActive() {
		t.Error("outer should be active")
	}

	// #ifdef INNER (undefined)
	if err := cp.ProcessIfdef("INNER"); err != nil {
		t.Fatalf("ProcessIfdef error: %v", err)
	}
	if cp.IsActive() {
		t.Error("inner should be inactive")
	}

	// #endif (inner)
	if err := cp.ProcessEndif(); err != nil {
		t.Fatalf("ProcessEndif error: %v", err)
	}
	if !cp.IsActive() {
		t.Error("should be back to active outer")
	}

	// #endif (outer)
	if err := cp.ProcessEndif(); err != nil {
		t.Fatalf("ProcessEndif error: %v", err)
	}
}

func TestConditionalNestedInactive(t *testing.T) {
	mt := NewMacroTable()
	cp := NewConditionalProcessor(mt)

	// #ifdef UNDEFINED
	if err := cp.ProcessIfdef("UNDEFINED"); err != nil {
		t.Fatalf("ProcessIfdef error: %v", err)
	}
	// Inactive

	// #ifdef ANYTHING (should not evaluate, just track nesting)
	if err := cp.ProcessIfdef("ANYTHING"); err != nil {
		t.Fatalf("ProcessIfdef error: %v", err)
	}
	if cp.Depth() != 2 {
		t.Errorf("depth = %d, want 2", cp.Depth())
	}

	// #endif
	if err := cp.ProcessEndif(); err != nil {
		t.Fatalf("ProcessEndif error: %v", err)
	}
	if cp.Depth() != 1 {
		t.Errorf("depth = %d, want 1", cp.Depth())
	}

	// #endif
	if err := cp.ProcessEndif(); err != nil {
		t.Fatalf("ProcessEndif error: %v", err)
	}
	if cp.Depth() != 0 {
		t.Errorf("depth = %d, want 0", cp.Depth())
	}
}

func TestConditionalErrors(t *testing.T) {
	tests := []struct {
		name   string
		action func(cp *ConditionalProcessor) error
		errMsg string
	}{
		{
			name: "else without if",
			action: func(cp *ConditionalProcessor) error {
				return cp.ProcessElse()
			},
			errMsg: "without matching #if",
		},
		{
			name: "endif without if",
			action: func(cp *ConditionalProcessor) error {
				return cp.ProcessEndif()
			},
			errMsg: "without matching #if",
		},
		{
			name: "elif without if",
			action: func(cp *ConditionalProcessor) error {
				return cp.ProcessElif(tokenize("1"))
			},
			errMsg: "without matching #if",
		},
		{
			name: "duplicate else",
			action: func(cp *ConditionalProcessor) error {
				cp.ProcessIfdef("X")
				cp.ProcessElse()
				return cp.ProcessElse()
			},
			errMsg: "duplicate #else",
		},
		{
			name: "elif after else",
			action: func(cp *ConditionalProcessor) error {
				cp.ProcessIfdef("X")
				cp.ProcessElse()
				return cp.ProcessElif(tokenize("1"))
			},
			errMsg: "after #else",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := NewMacroTable()
			cp := NewConditionalProcessor(mt)
			err := tt.action(cp)
			if err == nil {
				t.Fatalf("expected error containing %q", tt.errMsg)
			}
			if !containsStr(err.Error(), tt.errMsg) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestConditionalCheckBalanced(t *testing.T) {
	mt := NewMacroTable()
	cp := NewConditionalProcessor(mt)

	// Start nested
	cp.ProcessIfdef("X")
	cp.ProcessIfdef("Y")

	err := cp.CheckBalanced()
	if err == nil {
		t.Fatal("expected error for unbalanced conditionals")
	}

	// Balance it
	cp.ProcessEndif()
	cp.ProcessEndif()

	err = cp.CheckBalanced()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExpressionEvaluation(t *testing.T) {
	tests := []struct {
		expr   string
		expect int64
	}{
		{"42", 42},
		{"0x2A", 42},
		{"052", 42},
		{"-5", -5},
		{"+5", 5},
		{"!0", 1},
		{"!1", 0},
		{"~0", -1},
		{"2 + 3", 5},
		{"10 - 3", 7},
		{"3 * 4", 12},
		{"15 / 3", 5},
		{"17 % 5", 2},
		{"1 << 4", 16},
		{"16 >> 2", 4},
		{"5 < 10", 1},
		{"5 > 10", 0},
		{"5 <= 5", 1},
		{"5 >= 6", 0},
		{"5 == 5", 1},
		{"5 != 5", 0},
		{"0xFF & 0x0F", 15},
		{"0xF0 | 0x0F", 255},
		{"0xFF ^ 0x0F", 240},
		{"1 && 1", 1},
		{"1 && 0", 0},
		{"0 || 1", 1},
		{"0 || 0", 0},
		{"1 ? 2 : 3", 2},
		{"0 ? 2 : 3", 3},
		{"(2 + 3) * 4", 20},
		{"'a'", 97},
		{"'\\n'", 10},
		{"'\\0'", 0},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			mt := NewMacroTable()
			cp := NewConditionalProcessor(mt)
			tokens := tokenize(tt.expr)

			// Evaluate directly through the internal method
			result, err := cp.evaluateCondition(tokens)
			if err != nil {
				t.Fatalf("evaluateCondition error: %v", err)
			}

			expectBool := tt.expect != 0
			if result != expectBool {
				t.Errorf("result = %v, want %v (expr value %d)", result, expectBool, tt.expect)
			}
		})
	}
}

func TestDefinedOperator(t *testing.T) {
	tests := []struct {
		name    string
		defined []string
		expr    string
		expect  bool
	}{
		{"defined(X) true", []string{"X"}, "defined(X)", true},
		{"defined(X) false", []string{}, "defined(X)", false},
		{"defined X true", []string{"X"}, "defined X", true},
		{"defined X false", []string{}, "defined X", false},
		{"!defined(X)", []string{}, "!defined(X)", true},
		{"defined(X) && defined(Y)", []string{"X", "Y"}, "defined(X) && defined(Y)", true},
		{"defined(X) || defined(Y)", []string{"X"}, "defined(X) || defined(Y)", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := NewMacroTable()
			for _, name := range tt.defined {
				mt.DefineSimple(name, "1", SourceLoc{})
			}

			cp := NewConditionalProcessor(mt)
			tokens := tokenize(tt.expr)

			result, err := cp.evaluateCondition(tokens)
			if err != nil {
				t.Fatalf("evaluateCondition error: %v", err)
			}

			if result != tt.expect {
				t.Errorf("result = %v, want %v", result, tt.expect)
			}
		})
	}
}

func containsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}

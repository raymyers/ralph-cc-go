package cpp

import (
	"strings"
	"testing"
)

func TestExpandObjectMacro(t *testing.T) {
	tests := []struct {
		name     string
		defines  map[string]string
		input    string
		expected string
	}{
		{
			name:     "simple replacement",
			defines:  map[string]string{"X": "42"},
			input:    "int a = X;",
			expected: "int a = 42;",
		},
		{
			name:     "multiple replacements",
			defines:  map[string]string{"X": "1", "Y": "2"},
			input:    "int a = X + Y;",
			expected: "int a = 1 + 2;",
		},
		{
			name:     "no replacement if not defined",
			defines:  map[string]string{"X": "42"},
			input:    "int a = Y;",
			expected: "int a = Y;",
		},
		{
			name:     "chained macro expansion",
			defines:  map[string]string{"X": "Y", "Y": "42"},
			input:    "int a = X;",
			expected: "int a = 42;",
		},
		{
			name:     "empty replacement",
			defines:  map[string]string{"EMPTY": ""},
			input:    "a EMPTY b",
			expected: "a b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := NewMacroTable()
			for name, value := range tt.defines {
				if err := mt.DefineSimple(name, value, SourceLoc{File: "test", Line: 1}); err != nil {
					t.Fatalf("DefineSimple(%s, %s) error: %v", name, value, err)
				}
			}

			e := NewExpander(mt)
			result, err := e.ExpandString(tt.input)
			if err != nil {
				t.Fatalf("ExpandString error: %v", err)
			}

			// Normalize whitespace for comparison
			result = normalizeWhitespace(result)
			expected := normalizeWhitespace(tt.expected)
			if result != expected {
				t.Errorf("got %q, want %q", result, expected)
			}
		})
	}
}

func TestExpandFunctionMacro(t *testing.T) {
	tests := []struct {
		name     string
		macros   []macroSpec
		input    string
		expected string
	}{
		{
			name: "simple function macro",
			macros: []macroSpec{
				{name: "ADD", params: []string{"a", "b"}, body: "((a)+(b))"},
			},
			input:    "int x = ADD(1, 2);",
			expected: "int x = ((1)+(2));",
		},
		{
			name: "nested parentheses in argument",
			macros: []macroSpec{
				{name: "F", params: []string{"x"}, body: "x"},
			},
			input:    "F((1+2))",
			expected: "(1+2)",
		},
		{
			name: "commas in nested parens",
			macros: []macroSpec{
				{name: "F", params: []string{"x"}, body: "x"},
			},
			input:    "F((a,b))",
			expected: "(a,b)",
		},
		{
			name: "macro not invoked without parens",
			macros: []macroSpec{
				{name: "F", params: []string{"x"}, body: "x"},
			},
			input:    "F",
			expected: "F",
		},
		{
			name: "whitespace between name and parens",
			macros: []macroSpec{
				{name: "F", params: []string{"x"}, body: "x"},
			},
			input:    "F (42)",
			expected: "42",
		},
		{
			name: "nested macro calls",
			macros: []macroSpec{
				{name: "ADD", params: []string{"a", "b"}, body: "((a)+(b))"},
				{name: "MUL", params: []string{"a", "b"}, body: "((a)*(b))"},
			},
			input:    "ADD(MUL(1,2), 3)",
			expected: "((((1)*(2)))+(3))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := NewMacroTable()
			for _, m := range tt.macros {
				bodyTokens := tokenize(m.body)
				if err := mt.DefineFunction(m.name, m.params, m.variadic, bodyTokens, SourceLoc{File: "test", Line: 1}); err != nil {
					t.Fatalf("DefineFunction error: %v", err)
				}
			}

			e := NewExpander(mt)
			result, err := e.ExpandString(tt.input)
			if err != nil {
				t.Fatalf("ExpandString error: %v", err)
			}

			result = normalizeWhitespace(result)
			expected := normalizeWhitespace(tt.expected)
			if result != expected {
				t.Errorf("got %q, want %q", result, expected)
			}
		})
	}
}

func TestStringification(t *testing.T) {
	tests := []struct {
		name     string
		macros   []macroSpec
		input    string
		expected string
	}{
		{
			name: "simple stringification",
			macros: []macroSpec{
				{name: "STR", params: []string{"x"}, body: "#x"},
			},
			input:    `STR(hello)`,
			expected: `"hello"`,
		},
		{
			name: "stringification with multiple tokens",
			macros: []macroSpec{
				{name: "STR", params: []string{"x"}, body: "#x"},
			},
			input:    `STR(a + b)`,
			expected: `"a + b"`,
		},
		{
			name: "stringification escapes quotes",
			macros: []macroSpec{
				{name: "STR", params: []string{"x"}, body: "#x"},
			},
			input:    `STR("hello")`,
			expected: `"\"hello\""`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := NewMacroTable()
			for _, m := range tt.macros {
				bodyTokens := tokenize(m.body)
				if err := mt.DefineFunction(m.name, m.params, m.variadic, bodyTokens, SourceLoc{File: "test", Line: 1}); err != nil {
					t.Fatalf("DefineFunction error: %v", err)
				}
			}

			e := NewExpander(mt)
			result, err := e.ExpandString(tt.input)
			if err != nil {
				t.Fatalf("ExpandString error: %v", err)
			}

			result = normalizeWhitespace(result)
			expected := normalizeWhitespace(tt.expected)
			if result != expected {
				t.Errorf("got %q, want %q", result, expected)
			}
		})
	}
}

func TestTokenPasting(t *testing.T) {
	tests := []struct {
		name     string
		macros   []macroSpec
		input    string
		expected string
	}{
		{
			name: "simple pasting",
			macros: []macroSpec{
				{name: "PASTE", params: []string{"a", "b"}, body: "a##b"},
			},
			input:    "PASTE(foo, bar)",
			expected: "foobar",
		},
		{
			name: "pasting numbers",
			macros: []macroSpec{
				{name: "CONCAT", params: []string{"a", "b"}, body: "a##b"},
			},
			input:    "CONCAT(x, 123)",
			expected: "x123",
		},
		{
			name: "object-like macro with paste",
			macros: []macroSpec{
				{name: "V", params: nil, body: "1"},
				{name: "MAKE", params: []string{"x"}, body: "v##x"},
			},
			input:    "MAKE(V)",
			expected: "vV",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := NewMacroTable()
			for _, m := range tt.macros {
				bodyTokens := tokenize(m.body)
				if m.params == nil {
					// Object-like
					if err := mt.DefineObject(m.name, bodyTokens, SourceLoc{File: "test", Line: 1}); err != nil {
						t.Fatalf("DefineObject error: %v", err)
					}
				} else {
					if err := mt.DefineFunction(m.name, m.params, m.variadic, bodyTokens, SourceLoc{File: "test", Line: 1}); err != nil {
						t.Fatalf("DefineFunction error: %v", err)
					}
				}
			}

			e := NewExpander(mt)
			result, err := e.ExpandString(tt.input)
			if err != nil {
				t.Fatalf("ExpandString error: %v", err)
			}

			result = normalizeWhitespace(result)
			expected := normalizeWhitespace(tt.expected)
			if result != expected {
				t.Errorf("got %q, want %q", result, expected)
			}
		})
	}
}

func TestVariadicMacros(t *testing.T) {
	tests := []struct {
		name     string
		macros   []macroSpec
		input    string
		expected string
	}{
		{
			name: "simple variadic",
			macros: []macroSpec{
				{name: "PRINT", params: []string{"fmt"}, variadic: true, body: "printf(fmt, __VA_ARGS__)"},
			},
			input:    `PRINT("x=%d", x)`,
			expected: `printf("x=%d", x)`,
		},
		{
			name: "variadic with multiple args",
			macros: []macroSpec{
				{name: "DEBUG", params: []string{}, variadic: true, body: "printf(__VA_ARGS__)"},
			},
			input:    `DEBUG("a=%d b=%d", a, b)`,
			expected: `printf("a=%d b=%d", a, b)`,
		},
		{
			name: "variadic with no extra args",
			macros: []macroSpec{
				{name: "LOG", params: []string{"msg"}, variadic: true, body: "log(msg)"},
			},
			input:    `LOG("hello")`,
			expected: `log("hello")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := NewMacroTable()
			for _, m := range tt.macros {
				bodyTokens := tokenize(m.body)
				if err := mt.DefineFunction(m.name, m.params, m.variadic, bodyTokens, SourceLoc{File: "test", Line: 1}); err != nil {
					t.Fatalf("DefineFunction error: %v", err)
				}
			}

			e := NewExpander(mt)
			result, err := e.ExpandString(tt.input)
			if err != nil {
				t.Fatalf("ExpandString error: %v", err)
			}

			result = normalizeWhitespace(result)
			expected := normalizeWhitespace(tt.expected)
			if result != expected {
				t.Errorf("got %q, want %q", result, expected)
			}
		})
	}
}

func TestRecursiveExpansionPrevention(t *testing.T) {
	tests := []struct {
		name     string
		defines  map[string]string
		input    string
		expected string
	}{
		{
			name:     "direct self-reference",
			defines:  map[string]string{"X": "X + 1"},
			input:    "X",
			expected: "X+1", // whitespace is stripped by DefineSimple tokenization
		},
		{
			name:     "indirect self-reference",
			defines:  map[string]string{"A": "B", "B": "A"},
			input:    "A",
			expected: "A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := NewMacroTable()
			for name, value := range tt.defines {
				if err := mt.DefineSimple(name, value, SourceLoc{File: "test", Line: 1}); err != nil {
					t.Fatalf("DefineSimple error: %v", err)
				}
			}

			e := NewExpander(mt)
			result, err := e.ExpandString(tt.input)
			if err != nil {
				t.Fatalf("ExpandString error: %v", err)
			}

			result = normalizeWhitespace(result)
			expected := normalizeWhitespace(tt.expected)
			if result != expected {
				t.Errorf("got %q, want %q", result, expected)
			}
		})
	}
}

func TestBuiltinMacros(t *testing.T) {
	mt := NewMacroTable()
	e := NewExpander(mt)
	e.loc = SourceLoc{File: "test.c", Line: 42, Column: 1}

	tests := []struct {
		input    string
		contains string
	}{
		{"__FILE__", `"test.c"`},
		{"__LINE__", "42"},
		{"__STDC__", "1"},
		{"__STDC_VERSION__", "201112L"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := e.ExpandString(tt.input)
			if err != nil {
				t.Fatalf("ExpandString error: %v", err)
			}
			if !strings.Contains(result, tt.contains) {
				t.Errorf("%s expansion %q does not contain %q", tt.input, result, tt.contains)
			}
		})
	}
}

func TestExpanderErrors(t *testing.T) {
	tests := []struct {
		name   string
		macros []macroSpec
		input  string
		errMsg string
	}{
		{
			name: "wrong number of arguments",
			macros: []macroSpec{
				{name: "F", params: []string{"a", "b"}, body: "a+b"},
			},
			input:  "F(1)",
			errMsg: "requires 2 arguments",
		},
		{
			name: "unterminated argument list",
			macros: []macroSpec{
				{name: "F", params: []string{"x"}, body: "x"},
			},
			input:  "F(1",
			errMsg: "unterminated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := NewMacroTable()
			for _, m := range tt.macros {
				bodyTokens := tokenize(m.body)
				if err := mt.DefineFunction(m.name, m.params, m.variadic, bodyTokens, SourceLoc{File: "test", Line: 1}); err != nil {
					t.Fatalf("DefineFunction error: %v", err)
				}
			}

			e := NewExpander(mt)
			_, err := e.ExpandString(tt.input)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errMsg)
			}
			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errMsg)
			}
		})
	}
}

// Helper types and functions

type macroSpec struct {
	name     string
	params   []string
	variadic bool
	body     string
}

func tokenize(s string) []Token {
	lex := NewLexer(s, "test")
	var tokens []Token
	for {
		tok := lex.NextToken()
		if tok.Type == PP_EOF || tok.Type == PP_NEWLINE {
			break
		}
		tokens = append(tokens, tok)
	}
	return tokens
}

func normalizeWhitespace(s string) string {
	// Replace sequences of whitespace with single space
	var sb strings.Builder
	lastWasSpace := true
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !lastWasSpace {
				sb.WriteByte(' ')
				lastWasSpace = true
			}
		} else {
			sb.WriteRune(r)
			lastWasSpace = false
		}
	}
	return strings.TrimSpace(sb.String())
}

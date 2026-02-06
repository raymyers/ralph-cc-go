package lexer

import "testing"

func TestNextToken(t *testing.T) {
	input := `int main() { return 42; }`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenInt_, "int"},
		{TokenIdent, "main"},
		{TokenLParen, "("},
		{TokenRParen, ")"},
		{TokenLBrace, "{"},
		{TokenReturn, "return"},
		{TokenInt, "42"},
		{TokenSemicolon, ";"},
		{TokenRBrace, "}"},
		{TokenEOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestOperators(t *testing.T) {
	input := `+ - * / % = == != < <= > >= && || ! & | ^ ~`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenPlus, "+"},
		{TokenMinus, "-"},
		{TokenStar, "*"},
		{TokenSlash, "/"},
		{TokenPercent, "%"},
		{TokenAssign, "="},
		{TokenEq, "=="},
		{TokenNe, "!="},
		{TokenLt, "<"},
		{TokenLe, "<="},
		{TokenGt, ">"},
		{TokenGe, ">="},
		{TokenAnd, "&&"},
		{TokenOr, "||"},
		{TokenNot, "!"},
		{TokenAmpersand, "&"},
		{TokenPipe, "|"},
		{TokenCaret, "^"},
		{TokenTilde, "~"},
		{TokenEOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestComments(t *testing.T) {
	input := `int // comment
main /* block
comment */ ()`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenInt_, "int"},
		{TokenIdent, "main"},
		{TokenLParen, "("},
		{TokenRParen, ")"},
		{TokenEOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLineDirective(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedLine int
		expectedFile string
	}{
		{
			name:         "simple #line",
			input:        "#line 42\nint",
			expectedLine: 42,
			expectedFile: "",
		},
		{
			name:         "GCC style # number",
			input:        "# 100\nint",
			expectedLine: 100,
			expectedFile: "",
		},
		{
			name:         "#line with filename",
			input:        "#line 50 \"test.c\"\nint",
			expectedLine: 50,
			expectedFile: "test.c",
		},
		{
			name:         "GCC style with filename",
			input:        "# 75 \"foo.c\"\nint",
			expectedLine: 75,
			expectedFile: "foo.c",
		},
		{
			name:         "GCC style with flags",
			input:        "# 10 \"bar.c\" 1 2\nint",
			expectedLine: 10,
			expectedFile: "bar.c",
		},
		{
			name:         "multiple #line directives",
			input:        "#line 5 \"a.c\"\n# 20 \"b.c\"\nint",
			expectedLine: 20,
			expectedFile: "b.c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(tt.input)
			tok := l.NextToken()

			if tok.Type != TokenInt_ {
				t.Fatalf("expected TokenInt_, got %s", tok.Type)
			}
			if tok.Line != tt.expectedLine {
				t.Errorf("line wrong. expected=%d, got=%d", tt.expectedLine, tok.Line)
			}
			if l.Filename() != tt.expectedFile {
				t.Errorf("filename wrong. expected=%q, got=%q", tt.expectedFile, l.Filename())
			}
		})
	}
}

func TestLineDirectiveDoesNotBreakCode(t *testing.T) {
	// Ensure normal code with # in comments works
	input := `int // # not a directive
main()`

	l := New(input)

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenInt_, "int"},
		{TokenIdent, "main"},
		{TokenLParen, "("},
		{TokenRParen, ")"},
		{TokenEOF, ""},
	}

	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestEllipsis(t *testing.T) {
	input := `int printf(const char *fmt, ...)`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenInt_, "int"},
		{TokenIdent, "printf"},
		{TokenLParen, "("},
		{TokenConst, "const"},
		{TokenChar, "char"},
		{TokenStar, "*"},
		{TokenIdent, "fmt"},
		{TokenComma, ","},
		{TokenEllipsis, "..."},
		{TokenRParen, ")"},
		{TokenEOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestEllipsisVsDot(t *testing.T) {
	// Test that single dots are still recognized correctly alongside ellipsis
	input := `s.x ...`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenIdent, "s"},
		{TokenDot, "."},
		{TokenIdent, "x"},
		{TokenEllipsis, "..."},
		{TokenEOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}


func TestAttributeTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"__attribute__", TokenAttribute},
		{"__asm", TokenAsm},
		{"__asm__", TokenAsm},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := New(tt.input)
			tok := l.NextToken()
			if tok.Type != tt.expected {
				t.Errorf("expected %s for %q, got %s", tt.expected, tt.input, tok.Type)
			}
		})
	}
}

func TestInlineTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"inline", TokenInline},
		{"__inline", TokenInline},
		{"__inline__", TokenInline},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := New(tt.input)
			tok := l.NextToken()
			if tok.Type != tt.expected {
				t.Errorf("expected %s for %q, got %s", tt.expected, tt.input, tok.Type)
			}
		})
	}
}

func TestAttributeInContext(t *testing.T) {
	input := `int foo(void) __attribute__((cold)) __asm("_foo");`

	expected := []struct {
		Type    TokenType
		Literal string
	}{
		{TokenInt_, "int"},
		{TokenIdent, "foo"},
		{TokenLParen, "("},
		{TokenVoid, "void"},
		{TokenRParen, ")"},
		{TokenAttribute, "__attribute__"},
		{TokenLParen, "("},
		{TokenLParen, "("},
		{TokenIdent, "cold"},
		{TokenRParen, ")"},
		{TokenRParen, ")"},
		{TokenAsm, "__asm"},
		{TokenLParen, "("},
		{TokenString, "_foo"},
		{TokenRParen, ")"},
		{TokenSemicolon, ";"},
		{TokenEOF, ""},
	}

	l := New(input)
	for i, exp := range expected {
		tok := l.NextToken()
		if tok.Type != exp.Type {
			t.Fatalf("token[%d]: expected type %s, got %s (literal: %q)", i, exp.Type, tok.Type, tok.Literal)
		}
		if tok.Literal != exp.Literal {
			t.Fatalf("token[%d]: expected literal %q, got %q", i, exp.Literal, tok.Literal)
		}
	}
}

func TestCharLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple char", "'a'", "a"},
		{"newline escape", `'\n'`, `\n`},
		{"tab escape", `'\t'`, `\t`},
		{"backslash escape", `'\\'`, `\\`},
		{"null escape", `'\0'`, `\0`},
		{"single quote escape", `'\''`, `\'`},
		{"digit char", "'0'", "0"},
		{"space char", "' '", " "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(tt.input)
			tok := l.NextToken()
			if tok.Type != TokenCharLit {
				t.Errorf("expected TokenCharLit, got %s", tok.Type)
			}
			if tok.Literal != tt.expected {
				t.Errorf("expected literal %q, got %q", tt.expected, tok.Literal)
			}
		})
	}
}

func TestCharLiteralInContext(t *testing.T) {
	input := `if (c == '\n') { x = 'x'; }`

	expected := []struct {
		Type    TokenType
		Literal string
	}{
		{TokenIf, "if"},
		{TokenLParen, "("},
		{TokenIdent, "c"},
		{TokenEq, "=="},
		{TokenCharLit, `\n`},
		{TokenRParen, ")"},
		{TokenLBrace, "{"},
		{TokenIdent, "x"},
		{TokenAssign, "="},
		{TokenCharLit, "x"},
		{TokenSemicolon, ";"},
		{TokenRBrace, "}"},
		{TokenEOF, ""},
	}

	l := New(input)
	for i, exp := range expected {
		tok := l.NextToken()
		if tok.Type != exp.Type {
			t.Fatalf("token[%d]: expected type %s, got %s (literal: %q)", i, exp.Type, tok.Type, tok.Literal)
		}
		if tok.Literal != exp.Literal {
			t.Fatalf("token[%d]: expected literal %q, got %q", i, exp.Literal, tok.Literal)
		}
	}
}


func TestHexAndOctalLiterals(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Hex literals
		{"0x0", "0x0"},
		{"0xFF", "0xFF"},
		{"0xABCD", "0xABCD"},
		{"0X10", "0X10"},
		{"0xDeadBeef", "0xDeadBeef"},
		// Hex with suffixes
		{"0xFFu", "0xFFu"},
		{"0xFFUL", "0xFFUL"},
		{"0x1ll", "0x1ll"},
		// Octal literals
		{"0", "0"},
		{"07", "07"},
		{"0777", "0777"},
		{"0123", "0123"},
		// Octal with suffixes
		{"0777L", "0777L"},
		{"0123ul", "0123ul"},
		// Decimal for comparison
		{"123", "123"},
		{"42u", "42u"},
		{"1000L", "1000L"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := New(tt.input)
			tok := l.NextToken()
			if tok.Type != TokenInt {
				t.Errorf("expected INT token for %q, got %s", tt.input, tok.Type)
			}
			if tok.Literal != tt.expected {
				t.Errorf("expected literal %q, got %q", tt.expected, tok.Literal)
			}
		})
	}
}

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

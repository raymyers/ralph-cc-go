package cpp

import (
	"testing"
)

func TestTokenTypeString(t *testing.T) {
	tests := []struct {
		tt   TokenType
		want string
	}{
		{PP_EOF, "EOF"},
		{PP_IDENTIFIER, "IDENTIFIER"},
		{PP_NUMBER, "NUMBER"},
		{PP_CHAR_CONST, "CHAR_CONST"},
		{PP_STRING, "STRING"},
		{PP_PUNCTUATOR, "PUNCTUATOR"},
		{PP_HASH, "HASH"},
		{PP_HASHHASH, "HASHHASH"},
		{PP_NEWLINE, "NEWLINE"},
		{PP_WHITESPACE, "WHITESPACE"},
		{PP_HEADER_NAME, "HEADER_NAME"},
		{PP_PLACEHOLDER, "PLACEHOLDER"},
		{TokenType(999), "UNKNOWN"},
	}
	for _, tc := range tests {
		if got := tc.tt.String(); got != tc.want {
			t.Errorf("TokenType(%d).String() = %q, want %q", tc.tt, got, tc.want)
		}
	}
}

func TestLexerIdentifier(t *testing.T) {
	l := NewLexer("foo _bar123 __MACRO", "test.c")
	tok := l.NextToken()
	if tok.Type != PP_IDENTIFIER || tok.Text != "foo" {
		t.Errorf("got %v %q, want IDENTIFIER foo", tok.Type, tok.Text)
	}
	tok = l.NextToken() // whitespace
	if tok.Type != PP_WHITESPACE {
		t.Errorf("got %v, want WHITESPACE", tok.Type)
	}
	tok = l.NextToken()
	if tok.Type != PP_IDENTIFIER || tok.Text != "_bar123" {
		t.Errorf("got %v %q, want IDENTIFIER _bar123", tok.Type, tok.Text)
	}
	l.NextToken() // whitespace
	tok = l.NextToken()
	if tok.Type != PP_IDENTIFIER || tok.Text != "__MACRO" {
		t.Errorf("got %v %q, want IDENTIFIER __MACRO", tok.Type, tok.Text)
	}
}

func TestLexerNumber(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"42", "42"},
		{"3.14", "3.14"},
		{".5", ".5"},
		{"0x1F", "0x1F"},
		{"1e10", "1e10"},
		{"1E-5", "1E-5"},
		{"0xAp+3", "0xAp+3"},
		{"123ULL", "123ULL"},
		{"1.5f", "1.5f"},
	}
	for _, tc := range tests {
		l := NewLexer(tc.input, "test.c")
		tok := l.NextToken()
		if tok.Type != PP_NUMBER || tok.Text != tc.want {
			t.Errorf("input %q: got %v %q, want NUMBER %q", tc.input, tok.Type, tok.Text, tc.want)
		}
	}
}

func TestLexerString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`"hello"`, `"hello"`},
		{`"with\nescape"`, `"with\nescape"`},
		{`"with\"quote"`, `"with\"quote"`},
		{`""`, `""`},
	}
	for _, tc := range tests {
		l := NewLexer(tc.input, "test.c")
		tok := l.NextToken()
		if tok.Type != PP_STRING || tok.Text != tc.want {
			t.Errorf("input %q: got %v %q, want STRING %q", tc.input, tok.Type, tok.Text, tc.want)
		}
	}
}

func TestLexerCharConst(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`'a'`, `'a'`},
		{`'\n'`, `'\n'`},
		{`'\''`, `'\''`},
		{`'0'`, `'0'`},
	}
	for _, tc := range tests {
		l := NewLexer(tc.input, "test.c")
		tok := l.NextToken()
		if tok.Type != PP_CHAR_CONST || tok.Text != tc.want {
			t.Errorf("input %q: got %v %q, want CHAR_CONST %q", tc.input, tok.Type, tok.Text, tc.want)
		}
	}
}

func TestLexerPunctuator(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"+", "+"},
		{"++", "++"},
		{"->", "->"},
		{"<<=", "<<="},
		{">>=", ">>="},
		{"...", "..."},
		{"==", "=="},
		{"!=", "!="},
		{"&&", "&&"},
		{"||", "||"},
		{"[", "["},
		{"]", "]"},
		{"{", "{"},
		{"}", "}"},
		{"(", "("},
		{")", ")"},
	}
	for _, tc := range tests {
		l := NewLexer(tc.input, "test.c")
		tok := l.NextToken()
		if tok.Type != PP_PUNCTUATOR || tok.Text != tc.want {
			t.Errorf("input %q: got %v %q, want PUNCTUATOR %q", tc.input, tok.Type, tok.Text, tc.want)
		}
	}
}

func TestLexerHash(t *testing.T) {
	// # at beginning of line is PP_HASH
	l := NewLexer("#define", "test.c")
	tok := l.NextToken()
	if tok.Type != PP_HASH || tok.Text != "#" {
		t.Errorf("got %v %q, want HASH #", tok.Type, tok.Text)
	}

	// # not at beginning of line is PP_PUNCTUATOR
	l = NewLexer("a #", "test.c")
	l.NextToken() // a
	l.NextToken() // whitespace
	tok = l.NextToken()
	if tok.Type != PP_PUNCTUATOR || tok.Text != "#" {
		t.Errorf("got %v %q, want PUNCTUATOR #", tok.Type, tok.Text)
	}
}

func TestLexerHashHash(t *testing.T) {
	l := NewLexer("a ## b", "test.c")
	l.NextToken() // a
	l.NextToken() // whitespace
	tok := l.NextToken()
	if tok.Type != PP_HASHHASH || tok.Text != "##" {
		t.Errorf("got %v %q, want HASHHASH ##", tok.Type, tok.Text)
	}
}

func TestLexerNewline(t *testing.T) {
	l := NewLexer("a\nb", "test.c")
	tok := l.NextToken() // a
	if tok.Type != PP_IDENTIFIER {
		t.Errorf("got %v, want IDENTIFIER", tok.Type)
	}
	tok = l.NextToken()
	if tok.Type != PP_NEWLINE {
		t.Errorf("got %v, want NEWLINE", tok.Type)
	}
	tok = l.NextToken() // b
	if tok.Type != PP_IDENTIFIER {
		t.Errorf("got %v, want IDENTIFIER", tok.Type)
	}
}

func TestLexerLineContinuation(t *testing.T) {
	l := NewLexer("abc\\\ndef", "test.c")
	tok := l.NextToken()
	if tok.Type != PP_IDENTIFIER || tok.Text != "abcdef" {
		t.Errorf("got %v %q, want IDENTIFIER abcdef", tok.Type, tok.Text)
	}
}

func TestLexerLineComment(t *testing.T) {
	l := NewLexer("a // comment\nb", "test.c")
	tok := l.NextToken() // a
	if tok.Type != PP_IDENTIFIER {
		t.Errorf("got %v, want IDENTIFIER", tok.Type)
	}
	tok = l.NextToken() // space before comment
	if tok.Type != PP_WHITESPACE {
		t.Errorf("got %v, want WHITESPACE (space before comment)", tok.Type)
	}
	tok = l.NextToken() // comment (replaced with space)
	if tok.Type != PP_WHITESPACE {
		t.Errorf("got %v, want WHITESPACE (comment replaced)", tok.Type)
	}
	tok = l.NextToken() // newline
	if tok.Type != PP_NEWLINE {
		t.Errorf("got %v, want NEWLINE", tok.Type)
	}
	tok = l.NextToken() // b
	if tok.Type != PP_IDENTIFIER || tok.Text != "b" {
		t.Errorf("got %v %q, want IDENTIFIER b", tok.Type, tok.Text)
	}
}

func TestLexerBlockComment(t *testing.T) {
	l := NewLexer("a /* comment */ b", "test.c")
	tok := l.NextToken() // a
	if tok.Type != PP_IDENTIFIER {
		t.Errorf("got %v, want IDENTIFIER", tok.Type)
	}
	tok = l.NextToken() // whitespace
	if tok.Type != PP_WHITESPACE {
		t.Errorf("got %v, want WHITESPACE", tok.Type)
	}
	tok = l.NextToken() // comment (replaced with space)
	if tok.Type != PP_WHITESPACE {
		t.Errorf("got %v, want WHITESPACE (comment replaced)", tok.Type)
	}
	tok = l.NextToken() // whitespace
	if tok.Type != PP_WHITESPACE {
		t.Errorf("got %v, want WHITESPACE", tok.Type)
	}
	tok = l.NextToken() // b
	if tok.Type != PP_IDENTIFIER || tok.Text != "b" {
		t.Errorf("got %v %q, want IDENTIFIER b", tok.Type, tok.Text)
	}
}

func TestLexerSourceLocation(t *testing.T) {
	l := NewLexer("ab\ncd", "test.c")

	tok := l.NextToken() // ab
	if tok.Loc.Line != 1 || tok.Loc.Column != 1 {
		t.Errorf("got line=%d col=%d, want line=1 col=1", tok.Loc.Line, tok.Loc.Column)
	}
	if tok.Loc.File != "test.c" {
		t.Errorf("got file=%q, want test.c", tok.Loc.File)
	}

	l.NextToken() // newline

	tok = l.NextToken() // cd
	if tok.Loc.Line != 2 || tok.Loc.Column != 1 {
		t.Errorf("got line=%d col=%d, want line=2 col=1", tok.Loc.Line, tok.Loc.Column)
	}
}

func TestLexerAllTokens(t *testing.T) {
	l := NewLexer("a b", "test.c")
	tokens := l.AllTokens()

	if len(tokens) != 4 { // a, whitespace, b, EOF
		t.Errorf("got %d tokens, want 4", len(tokens))
	}
	if tokens[0].Type != PP_IDENTIFIER {
		t.Errorf("token 0: got %v, want IDENTIFIER", tokens[0].Type)
	}
	if tokens[1].Type != PP_WHITESPACE {
		t.Errorf("token 1: got %v, want WHITESPACE", tokens[1].Type)
	}
	if tokens[2].Type != PP_IDENTIFIER {
		t.Errorf("token 2: got %v, want IDENTIFIER", tokens[2].Type)
	}
	if tokens[3].Type != PP_EOF {
		t.Errorf("token 3: got %v, want EOF", tokens[3].Type)
	}
}

func TestScanHeaderName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`<stdio.h>`, `<stdio.h>`},
		{`"myfile.h"`, `"myfile.h"`},
		{`<sys/types.h>`, `<sys/types.h>`},
	}
	for _, tc := range tests {
		l := NewLexer(tc.input, "test.c")
		l.atBOL = false // pretend we're past the #include
		tok := l.ScanHeaderName()
		if tok.Type != PP_HEADER_NAME || tok.Text != tc.want {
			t.Errorf("input %q: got %v %q, want HEADER_NAME %q", tc.input, tok.Type, tok.Text, tc.want)
		}
	}
}

func TestTokensToString(t *testing.T) {
	tokens := []Token{
		{Type: PP_IDENTIFIER, Text: "foo"},
		{Type: PP_WHITESPACE, Text: " "},
		{Type: PP_PUNCTUATOR, Text: "="},
		{Type: PP_WHITESPACE, Text: " "},
		{Type: PP_NUMBER, Text: "42"},
	}
	got := TokensToString(tokens)
	want := "foo = 42"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIsIdentifier(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"foo", true},
		{"_bar", true},
		{"foo123", true},
		{"__FILE__", true},
		{"123abc", false},
		{"foo-bar", false},
		{"", false},
	}
	for _, tc := range tests {
		if got := IsIdentifier(tc.input); got != tc.want {
			t.Errorf("IsIdentifier(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestLexerDirective(t *testing.T) {
	l := NewLexer("#define FOO 42", "test.c")

	tok := l.NextToken()
	if tok.Type != PP_HASH {
		t.Errorf("got %v, want HASH", tok.Type)
	}

	tok = l.NextToken()
	if tok.Type != PP_IDENTIFIER || tok.Text != "define" {
		t.Errorf("got %v %q, want IDENTIFIER define", tok.Type, tok.Text)
	}

	tok = l.NextToken() // whitespace
	tok = l.NextToken()
	if tok.Type != PP_IDENTIFIER || tok.Text != "FOO" {
		t.Errorf("got %v %q, want IDENTIFIER FOO", tok.Type, tok.Text)
	}

	tok = l.NextToken() // whitespace
	tok = l.NextToken()
	if tok.Type != PP_NUMBER || tok.Text != "42" {
		t.Errorf("got %v %q, want NUMBER 42", tok.Type, tok.Text)
	}
}

func TestLexerHashAtBOLAfterNewline(t *testing.T) {
	l := NewLexer("a\n#define", "test.c")

	l.NextToken() // a
	l.NextToken() // newline

	tok := l.NextToken()
	if tok.Type != PP_HASH {
		t.Errorf("got %v, want HASH (# at BOL after newline)", tok.Type)
	}
}

func TestLexerEmptyInput(t *testing.T) {
	l := NewLexer("", "test.c")
	tok := l.NextToken()
	if tok.Type != PP_EOF {
		t.Errorf("got %v, want EOF", tok.Type)
	}
}

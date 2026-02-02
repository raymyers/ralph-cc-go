// Package cpp implements a standalone C preprocessor.
package cpp

import (
	"strings"
	"unicode"
)

// TokenType represents the type of a preprocessing token.
type TokenType int

const (
	PP_EOF TokenType = iota
	PP_IDENTIFIER
	PP_NUMBER
	PP_CHAR_CONST
	PP_STRING
	PP_PUNCTUATOR
	PP_HASH         // # at line start (directive marker)
	PP_HASHHASH     // ## (token pasting)
	PP_NEWLINE      // significant for directive boundaries
	PP_WHITESPACE   // preserved for macro spacing
	PP_HEADER_NAME  // <file> or "file" after #include
	PP_PLACEHOLDER  // placeholder during macro expansion
)

func (t TokenType) String() string {
	switch t {
	case PP_EOF:
		return "EOF"
	case PP_IDENTIFIER:
		return "IDENTIFIER"
	case PP_NUMBER:
		return "NUMBER"
	case PP_CHAR_CONST:
		return "CHAR_CONST"
	case PP_STRING:
		return "STRING"
	case PP_PUNCTUATOR:
		return "PUNCTUATOR"
	case PP_HASH:
		return "HASH"
	case PP_HASHHASH:
		return "HASHHASH"
	case PP_NEWLINE:
		return "NEWLINE"
	case PP_WHITESPACE:
		return "WHITESPACE"
	case PP_HEADER_NAME:
		return "HEADER_NAME"
	case PP_PLACEHOLDER:
		return "PLACEHOLDER"
	default:
		return "UNKNOWN"
	}
}

// SourceLoc represents a position in the source file.
type SourceLoc struct {
	File   string
	Line   int
	Column int
}

// Token represents a preprocessing token.
type Token struct {
	Type TokenType
	Text string
	Loc  SourceLoc
}

// Lexer tokenizes C source code into preprocessing tokens.
type Lexer struct {
	input    string
	pos      int
	line     int
	column   int
	filename string
	atBOL    bool // at beginning of line (for # detection)
}

// NewLexer creates a new preprocessor lexer.
func NewLexer(input, filename string) *Lexer {
	return &Lexer{
		input:    input,
		pos:      0,
		line:     1,
		column:   1,
		filename: filename,
		atBOL:    true,
	}
}

// NextToken returns the next preprocessing token.
func (l *Lexer) NextToken() Token {
	// Handle line continuation first (backslash-newline)
	l.handleLineContinuation()

	if l.pos >= len(l.input) {
		return Token{Type: PP_EOF, Text: "", Loc: l.loc()}
	}

	// Check for newline (significant for directive boundaries)
	if l.peek() == '\n' {
		tok := Token{Type: PP_NEWLINE, Text: "\n", Loc: l.loc()}
		l.advance()
		l.atBOL = true
		return tok
	}

	// Handle whitespace (preserved for macro spacing)
	if l.isWhitespace(l.peek()) {
		return l.scanWhitespace()
	}

	// Handle comments (replace with single space per C spec)
	if l.peek() == '/' && l.pos+1 < len(l.input) {
		if l.input[l.pos+1] == '/' {
			return l.scanLineComment()
		}
		if l.input[l.pos+1] == '*' {
			return l.scanBlockComment()
		}
	}

	// Check for # at beginning of line (directive marker)
	if l.peek() == '#' && l.atBOL {
		return l.scanHash()
	}

	l.atBOL = false

	// Check for ## (token pasting)
	if l.peek() == '#' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '#' {
		tok := Token{Type: PP_HASHHASH, Text: "##", Loc: l.loc()}
		l.advance()
		l.advance()
		return tok
	}

	// Check for # (stringification operator in macros)
	if l.peek() == '#' {
		tok := Token{Type: PP_PUNCTUATOR, Text: "#", Loc: l.loc()}
		l.advance()
		return tok
	}

	// Handle string literals
	if l.peek() == '"' {
		return l.scanString()
	}

	// Handle character constants
	if l.peek() == '\'' {
		return l.scanCharConst()
	}

	// Handle preprocessing numbers (broader than C numbers)
	if l.isDigit(l.peek()) || (l.peek() == '.' && l.pos+1 < len(l.input) && l.isDigit(l.input[l.pos+1])) {
		return l.scanNumber()
	}

	// Handle identifiers and keywords
	if l.isIdentStart(l.peek()) {
		return l.scanIdentifier()
	}

	// Handle punctuators
	return l.scanPunctuator()
}

// AllTokens returns all tokens from the input.
func (l *Lexer) AllTokens() []Token {
	var tokens []Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == PP_EOF {
			break
		}
	}
	return tokens
}

func (l *Lexer) handleLineContinuation() {
	for l.pos < len(l.input)-1 && l.input[l.pos] == '\\' && l.input[l.pos+1] == '\n' {
		l.pos += 2
		l.line++
		l.column = 1
	}
}

// skipLineContinuation checks for and skips a line continuation at the current position.
// Returns true if a continuation was skipped.
func (l *Lexer) skipLineContinuation() bool {
	if l.pos < len(l.input)-1 && l.input[l.pos] == '\\' && l.input[l.pos+1] == '\n' {
		l.pos += 2
		l.line++
		l.column = 1
		return true
	}
	return false
}

func (l *Lexer) loc() SourceLoc {
	return SourceLoc{File: l.filename, Line: l.line, Column: l.column}
}

func (l *Lexer) peek() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) peekAt(offset int) byte {
	if l.pos+offset >= len(l.input) {
		return 0
	}
	return l.input[l.pos+offset]
}

func (l *Lexer) advance() {
	if l.pos < len(l.input) {
		if l.input[l.pos] == '\n' {
			l.line++
			l.column = 1
		} else {
			l.column++
		}
		l.pos++
	}
}

func (l *Lexer) isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\f' || c == '\v'
}

func (l *Lexer) isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func (l *Lexer) isHexDigit(c byte) bool {
	return l.isDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func (l *Lexer) isIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func (l *Lexer) isIdentContinue(c byte) bool {
	return l.isIdentStart(c) || l.isDigit(c)
}

func (l *Lexer) scanWhitespace() Token {
	loc := l.loc()
	start := l.pos
	for l.pos < len(l.input) && l.isWhitespace(l.peek()) {
		l.advance()
	}
	return Token{Type: PP_WHITESPACE, Text: l.input[start:l.pos], Loc: loc}
}

func (l *Lexer) scanLineComment() Token {
	loc := l.loc()
	// Skip //
	l.advance()
	l.advance()
	for l.pos < len(l.input) && l.peek() != '\n' {
		l.advance()
	}
	// Per C spec, comments are replaced with a single space
	return Token{Type: PP_WHITESPACE, Text: " ", Loc: loc}
}

func (l *Lexer) scanBlockComment() Token {
	loc := l.loc()
	// Skip /*
	l.advance()
	l.advance()
	for l.pos < len(l.input) {
		if l.peek() == '*' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '/' {
			l.advance()
			l.advance()
			break
		}
		l.advance()
	}
	// Per C spec, comments are replaced with a single space
	return Token{Type: PP_WHITESPACE, Text: " ", Loc: loc}
}

func (l *Lexer) scanHash() Token {
	loc := l.loc()
	l.advance() // consume #
	l.atBOL = false

	// Check for ## at start of line
	if l.peek() == '#' {
		l.advance()
		return Token{Type: PP_HASHHASH, Text: "##", Loc: loc}
	}

	return Token{Type: PP_HASH, Text: "#", Loc: loc}
}

func (l *Lexer) scanString() Token {
	loc := l.loc()
	start := l.pos
	l.advance() // consume opening "
	for l.pos < len(l.input) {
		if l.peek() == '"' {
			l.advance()
			break
		}
		if l.peek() == '\\' && l.pos+1 < len(l.input) {
			l.advance() // skip backslash
			l.advance() // skip escaped char
			continue
		}
		if l.peek() == '\n' {
			// Unterminated string literal
			break
		}
		l.advance()
	}
	return Token{Type: PP_STRING, Text: l.input[start:l.pos], Loc: loc}
}

func (l *Lexer) scanCharConst() Token {
	loc := l.loc()
	start := l.pos
	l.advance() // consume opening '
	for l.pos < len(l.input) {
		if l.peek() == '\'' {
			l.advance()
			break
		}
		if l.peek() == '\\' && l.pos+1 < len(l.input) {
			l.advance() // skip backslash
			l.advance() // skip escaped char
			continue
		}
		if l.peek() == '\n' {
			// Unterminated char constant
			break
		}
		l.advance()
	}
	return Token{Type: PP_CHAR_CONST, Text: l.input[start:l.pos], Loc: loc}
}

func (l *Lexer) scanNumber() Token {
	// Preprocessing numbers are broader than C numbers:
	// pp-number: digit | . digit | pp-number digit | pp-number identifier-nondigit
	//          | pp-number e sign | pp-number E sign | pp-number p sign | pp-number P sign
	//          | pp-number .
	loc := l.loc()
	start := l.pos

	for l.pos < len(l.input) {
		c := l.peek()
		if l.isDigit(c) || l.isIdentContinue(c) || c == '.' {
			// Check for exponent sign
			if (c == 'e' || c == 'E' || c == 'p' || c == 'P') && l.pos+1 < len(l.input) {
				next := l.input[l.pos+1]
				if next == '+' || next == '-' {
					l.advance()
					l.advance()
					continue
				}
			}
			l.advance()
		} else {
			break
		}
	}
	return Token{Type: PP_NUMBER, Text: l.input[start:l.pos], Loc: loc}
}

func (l *Lexer) scanIdentifier() Token {
	loc := l.loc()
	var text strings.Builder
	for {
		// Skip any line continuations
		for l.skipLineContinuation() {
		}
		if l.pos >= len(l.input) || !l.isIdentContinue(l.peek()) {
			break
		}
		text.WriteByte(l.peek())
		l.advance()
	}
	return Token{Type: PP_IDENTIFIER, Text: text.String(), Loc: loc}
}

func (l *Lexer) scanPunctuator() Token {
	loc := l.loc()
	start := l.pos

	// Try to match multi-character punctuators first
	remaining := l.input[l.pos:]

	// Three-character punctuators
	if len(remaining) >= 3 {
		three := remaining[:3]
		if three == "<<=" || three == ">>=" || three == "..." {
			l.advance()
			l.advance()
			l.advance()
			return Token{Type: PP_PUNCTUATOR, Text: three, Loc: loc}
		}
	}

	// Two-character punctuators
	if len(remaining) >= 2 {
		two := remaining[:2]
		switch two {
		case "->", "++", "--", "<<", ">>", "<=", ">=", "==", "!=",
			"&&", "||", "*=", "/=", "%=", "+=", "-=", "&=", "^=", "|=":
			l.advance()
			l.advance()
			return Token{Type: PP_PUNCTUATOR, Text: two, Loc: loc}
		}
	}

	// Single-character punctuators
	l.advance()
	return Token{Type: PP_PUNCTUATOR, Text: l.input[start:l.pos], Loc: loc}
}

// ScanHeaderName scans a header name after #include directive.
// This is called by the directive parser when it sees #include.
func (l *Lexer) ScanHeaderName() Token {
	// Skip whitespace first
	for l.pos < len(l.input) && l.isWhitespace(l.peek()) {
		l.advance()
	}

	if l.pos >= len(l.input) {
		return Token{Type: PP_EOF, Text: "", Loc: l.loc()}
	}

	loc := l.loc()
	start := l.pos

	if l.peek() == '<' {
		// Angle bracket form: <file>
		l.advance()
		for l.pos < len(l.input) && l.peek() != '>' && l.peek() != '\n' {
			l.advance()
		}
		if l.peek() == '>' {
			l.advance()
		}
		return Token{Type: PP_HEADER_NAME, Text: l.input[start:l.pos], Loc: loc}
	}

	if l.peek() == '"' {
		// Quoted form: "file"
		l.advance()
		for l.pos < len(l.input) && l.peek() != '"' && l.peek() != '\n' {
			l.advance()
		}
		if l.peek() == '"' {
			l.advance()
		}
		return Token{Type: PP_HEADER_NAME, Text: l.input[start:l.pos], Loc: loc}
	}

	// Could be a macro that expands to a header name
	return l.NextToken()
}

// TokensToString converts a slice of tokens back to source text.
func TokensToString(tokens []Token) string {
	var sb strings.Builder
	for _, tok := range tokens {
		sb.WriteString(tok.Text)
	}
	return sb.String()
}

// IsIdentifier checks if a string is a valid C identifier.
func IsIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	r := rune(s[0])
	if !unicode.IsLetter(r) && r != '_' {
		return false
	}
	for _, r := range s[1:] {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}

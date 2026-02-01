package lexer

import (
	"unicode"
)

// Lexer tokenizes C source code
type Lexer struct {
	input    string
	pos      int    // current position in input
	readPos  int    // next reading position
	ch       byte   // current character
	line     int
	column   int
	filename string // current filename from #line directive
}

// New creates a new Lexer for the given input
func New(input string) *Lexer {
	l := &Lexer{input: input, line: 1, column: 0}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0 // EOF
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++
	l.column++

	if l.ch == '\n' {
		l.line++
		l.column = 0
	}
}

func (l *Lexer) peekChar() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

func (l *Lexer) peekCharN(n int) byte {
	pos := l.readPos + n - 1
	if pos >= len(l.input) {
		return 0
	}
	return l.input[pos]
}

// NextToken returns the next token from the input
func (l *Lexer) NextToken() Token {
	l.skipWhitespace()
	l.skipComments()
	l.skipWhitespace()

	tok := Token{Line: l.line, Column: l.column}

	switch l.ch {
	case 0:
		tok.Type = TokenEOF
		tok.Literal = ""
	case '+':
		if l.peekChar() == '=' {
			tok.Type = TokenPlusAssign
			tok.Literal = "+="
			l.readChar()
		} else if l.peekChar() == '+' {
			tok.Type = TokenIncrement
			tok.Literal = "++"
			l.readChar()
		} else {
			tok = l.newToken(TokenPlus, l.ch)
		}
	case '-':
		if l.peekChar() == '>' {
			tok.Type = TokenArrow
			tok.Literal = "->"
			l.readChar()
		} else if l.peekChar() == '=' {
			tok.Type = TokenMinusAssign
			tok.Literal = "-="
			l.readChar()
		} else if l.peekChar() == '-' {
			tok.Type = TokenDecrement
			tok.Literal = "--"
			l.readChar()
		} else {
			tok = l.newToken(TokenMinus, l.ch)
		}
	case '*':
		if l.peekChar() == '=' {
			tok.Type = TokenStarAssign
			tok.Literal = "*="
			l.readChar()
		} else {
			tok = l.newToken(TokenStar, l.ch)
		}
	case '/':
		if l.peekChar() == '=' {
			tok.Type = TokenSlashAssign
			tok.Literal = "/="
			l.readChar()
		} else {
			tok = l.newToken(TokenSlash, l.ch)
		}
	case '%':
		if l.peekChar() == '=' {
			tok.Type = TokenPercentAssign
			tok.Literal = "%="
			l.readChar()
		} else {
			tok = l.newToken(TokenPercent, l.ch)
		}
	case '=':
		if l.peekChar() == '=' {
			tok.Type = TokenEq
			tok.Literal = "=="
			l.readChar()
		} else {
			tok = l.newToken(TokenAssign, l.ch)
		}
	case '!':
		if l.peekChar() == '=' {
			tok.Type = TokenNe
			tok.Literal = "!="
			l.readChar()
		} else {
			tok = l.newToken(TokenNot, l.ch)
		}
	case '<':
		if l.peekChar() == '<' && l.peekCharN(2) == '=' {
			tok.Type = TokenShlAssign
			tok.Literal = "<<="
			l.readChar()
			l.readChar()
		} else if l.peekChar() == '<' {
			tok.Type = TokenShl
			tok.Literal = "<<"
			l.readChar()
		} else if l.peekChar() == '=' {
			tok.Type = TokenLe
			tok.Literal = "<="
			l.readChar()
		} else {
			tok = l.newToken(TokenLt, l.ch)
		}
	case '>':
		if l.peekChar() == '>' && l.peekCharN(2) == '=' {
			tok.Type = TokenShrAssign
			tok.Literal = ">>="
			l.readChar()
			l.readChar()
		} else if l.peekChar() == '>' {
			tok.Type = TokenShr
			tok.Literal = ">>"
			l.readChar()
		} else if l.peekChar() == '=' {
			tok.Type = TokenGe
			tok.Literal = ">="
			l.readChar()
		} else {
			tok = l.newToken(TokenGt, l.ch)
		}
	case '?':
		tok = l.newToken(TokenQuestion, l.ch)
	case ':':
		tok = l.newToken(TokenColon, l.ch)
	case '&':
		if l.peekChar() == '&' {
			tok.Type = TokenAnd
			tok.Literal = "&&"
			l.readChar()
		} else if l.peekChar() == '=' {
			tok.Type = TokenAndAssign
			tok.Literal = "&="
			l.readChar()
		} else {
			tok = l.newToken(TokenAmpersand, l.ch)
		}
	case '|':
		if l.peekChar() == '|' {
			tok.Type = TokenOr
			tok.Literal = "||"
			l.readChar()
		} else if l.peekChar() == '=' {
			tok.Type = TokenOrAssign
			tok.Literal = "|="
			l.readChar()
		} else {
			tok = l.newToken(TokenPipe, l.ch)
		}
	case '^':
		if l.peekChar() == '=' {
			tok.Type = TokenXorAssign
			tok.Literal = "^="
			l.readChar()
		} else {
			tok = l.newToken(TokenCaret, l.ch)
		}
	case '~':
		tok = l.newToken(TokenTilde, l.ch)
	case '(':
		tok = l.newToken(TokenLParen, l.ch)
	case ')':
		tok = l.newToken(TokenRParen, l.ch)
	case '{':
		tok = l.newToken(TokenLBrace, l.ch)
	case '}':
		tok = l.newToken(TokenRBrace, l.ch)
	case '[':
		tok = l.newToken(TokenLBracket, l.ch)
	case ']':
		tok = l.newToken(TokenRBracket, l.ch)
	case ';':
		tok = l.newToken(TokenSemicolon, l.ch)
	case ',':
		tok = l.newToken(TokenComma, l.ch)
	case '.':
		tok = l.newToken(TokenDot, l.ch)
	case '"':
		tok.Type = TokenString
		tok.Literal = l.readString()
		return tok
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = LookupIdent(tok.Literal)
			return tok
		} else if isDigit(l.ch) {
			tok.Type = TokenInt
			tok.Literal = l.readNumber()
			return tok
		} else {
			tok = l.newToken(TokenIllegal, l.ch)
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) newToken(tokenType TokenType, ch byte) Token {
	return Token{Type: tokenType, Literal: string(ch), Line: l.line, Column: l.column}
}

func (l *Lexer) skipWhitespace() {
	for {
		for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
			l.readChar()
		}
		// Handle #line directives (preprocessor output)
		if l.ch == '#' {
			if l.skipLineDirective() {
				continue
			}
		}
		break
	}
}

func (l *Lexer) skipComments() {
	for l.ch == '/' {
		if l.peekChar() == '/' {
			// Single-line comment
			for l.ch != '\n' && l.ch != 0 {
				l.readChar()
			}
			l.skipWhitespace()
		} else if l.peekChar() == '*' {
			// Multi-line comment
			l.readChar() // consume /
			l.readChar() // consume *
			for {
				if l.ch == 0 {
					break
				}
				if l.ch == '*' && l.peekChar() == '/' {
					l.readChar() // consume *
					l.readChar() // consume /
					break
				}
				l.readChar()
			}
			l.skipWhitespace()
		} else {
			break
		}
	}
}

// skipLineDirective handles #line directives from preprocessor output
// Format: #line <number> ["<filename>"]
// Also handles: # <number> ["<filename>"] (GCC style)
func (l *Lexer) skipLineDirective() bool {
	if l.ch != '#' {
		return false
	}

	// Save position in case this isn't a #line directive
	startPos := l.pos
	startReadPos := l.readPos
	startLine := l.line
	startColumn := l.column
	startCh := l.ch

	l.readChar() // consume '#'

	// Skip whitespace after #
	for l.ch == ' ' || l.ch == '\t' {
		l.readChar()
	}

	// Check for "line" keyword (optional in GCC output)
	if l.ch == 'l' {
		// Try to read "line"
		if l.readPos+3 <= len(l.input) && l.input[l.pos:l.pos+4] == "line" {
			l.readChar() // l
			l.readChar() // i
			l.readChar() // n
			l.readChar() // e
			// Skip whitespace after "line"
			for l.ch == ' ' || l.ch == '\t' {
				l.readChar()
			}
		}
	}

	// Read line number
	if !isDigit(l.ch) {
		// Not a valid #line directive, restore position
		l.pos = startPos
		l.readPos = startReadPos
		l.line = startLine
		l.column = startColumn
		l.ch = startCh
		return false
	}

	lineNum := 0
	for isDigit(l.ch) {
		lineNum = lineNum*10 + int(l.ch-'0')
		l.readChar()
	}

	// Skip whitespace
	for l.ch == ' ' || l.ch == '\t' {
		l.readChar()
	}

	// Optional filename
	if l.ch == '"' {
		l.readChar() // consume opening quote
		filenameStart := l.pos
		for l.ch != '"' && l.ch != '\n' && l.ch != 0 {
			if l.ch == '\\' {
				l.readChar() // skip escape char
			}
			l.readChar()
		}
		l.filename = l.input[filenameStart:l.pos]
		if l.ch == '"' {
			l.readChar() // consume closing quote
		}
	}

	// Skip to end of line (there may be flags like 1 2 3 4 after filename)
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}

	// At this point, l.ch == '\n' (we stopped before consuming the newline).
	// The #line directive specifies what line number the NEXT line should be.
	// When skipWhitespace consumes this newline via readChar(), the line WON'T
	// be incremented because readChar() increments when the NEW char is '\n'.
	// So we set l.line = lineNum directly, and that will be the line number
	// for the first token on the next line.
	l.line = lineNum

	return true
}

// Filename returns the current filename from #line directives
func (l *Lexer) Filename() string {
	return l.filename
}

func (l *Lexer) readIdentifier() string {
	pos := l.pos
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[pos:l.pos]
}

func (l *Lexer) readNumber() string {
	pos := l.pos
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[pos:l.pos]
}

func (l *Lexer) readString() string {
	l.readChar() // consume opening quote
	pos := l.pos
	for l.ch != '"' && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar() // skip escape char
		}
		l.readChar()
	}
	str := l.input[pos:l.pos]
	l.readChar() // consume closing quote
	return str
}

func isLetter(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

package lexer

import (
	"unicode"
)

// Lexer tokenizes C source code
type Lexer struct {
	input   string
	pos     int  // current position in input
	readPos int  // next reading position
	ch      byte // current character
	line    int
	column  int
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
		tok = l.newToken(TokenPlus, l.ch)
	case '-':
		if l.peekChar() == '>' {
			tok.Type = TokenArrow
			tok.Literal = "->"
			l.readChar()
		} else {
			tok = l.newToken(TokenMinus, l.ch)
		}
	case '*':
		tok = l.newToken(TokenStar, l.ch)
	case '/':
		tok = l.newToken(TokenSlash, l.ch)
	case '%':
		tok = l.newToken(TokenPercent, l.ch)
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
		if l.peekChar() == '=' {
			tok.Type = TokenLe
			tok.Literal = "<="
			l.readChar()
		} else {
			tok = l.newToken(TokenLt, l.ch)
		}
	case '>':
		if l.peekChar() == '=' {
			tok.Type = TokenGe
			tok.Literal = ">="
			l.readChar()
		} else {
			tok = l.newToken(TokenGt, l.ch)
		}
	case '&':
		if l.peekChar() == '&' {
			tok.Type = TokenAnd
			tok.Literal = "&&"
			l.readChar()
		} else {
			tok = l.newToken(TokenAmpersand, l.ch)
		}
	case '|':
		if l.peekChar() == '|' {
			tok.Type = TokenOr
			tok.Literal = "||"
			l.readChar()
		} else {
			tok = l.newToken(TokenPipe, l.ch)
		}
	case '^':
		tok = l.newToken(TokenCaret, l.ch)
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
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
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

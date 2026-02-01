package lexer

// TokenType represents the type of a token
type TokenType int

const (
	// Special tokens
	TokenEOF TokenType = iota
	TokenIllegal

	// Literals
	TokenIdent  // main, foo, x
	TokenInt    // 42
	TokenString // "hello"

	// Keywords
	TokenInt_    // int
	TokenVoid    // void
	TokenReturn  // return
	TokenIf      // if
	TokenElse    // else
	TokenWhile   // while
	TokenDo      // do
	TokenFor     // for
	TokenTypedef // typedef
	TokenStruct  // struct
	TokenSizeof  // sizeof

	// Operators
	TokenPlus      // +
	TokenMinus     // -
	TokenStar      // *
	TokenSlash     // /
	TokenPercent   // %
	TokenAssign    // =
	TokenEq        // ==
	TokenNe        // !=
	TokenLt        // <
	TokenLe        // <=
	TokenGt        // >
	TokenGe        // >=
	TokenAnd       // &&
	TokenOr        // ||
	TokenNot       // !
	TokenAmpersand // &
	TokenPipe      // |
	TokenCaret     // ^
	TokenTilde     // ~
	TokenShl       // <<
	TokenShr       // >>
	TokenQuestion  // ?
	TokenColon     // :

	// Compound assignment operators
	TokenPlusAssign    // +=
	TokenMinusAssign   // -=
	TokenStarAssign    // *=
	TokenSlashAssign   // /=
	TokenPercentAssign // %=
	TokenAndAssign     // &=
	TokenOrAssign      // |=
	TokenXorAssign     // ^=
	TokenShlAssign     // <<=
	TokenShrAssign     // >>=

	// Increment/decrement
	TokenIncrement // ++
	TokenDecrement // --

	// Delimiters
	TokenLParen    // (
	TokenRParen    // )
	TokenLBrace    // {
	TokenRBrace    // }
	TokenLBracket  // [
	TokenRBracket  // ]
	TokenSemicolon // ;
	TokenComma     // ,
	TokenDot       // .
	TokenArrow     // ->
)

var tokenNames = map[TokenType]string{
	TokenEOF:           "EOF",
	TokenIllegal:       "ILLEGAL",
	TokenIdent:         "IDENT",
	TokenInt:           "INT",
	TokenString:        "STRING",
	TokenInt_:          "int",
	TokenVoid:          "void",
	TokenReturn:        "return",
	TokenIf:            "if",
	TokenElse:          "else",
	TokenWhile:         "while",
	TokenDo:            "do",
	TokenFor:           "for",
	TokenTypedef:       "typedef",
	TokenStruct:        "struct",
	TokenSizeof:        "sizeof",
	TokenPlus:          "+",
	TokenMinus:         "-",
	TokenStar:          "*",
	TokenSlash:         "/",
	TokenPercent:       "%",
	TokenAssign:        "=",
	TokenEq:            "==",
	TokenNe:            "!=",
	TokenLt:            "<",
	TokenLe:            "<=",
	TokenGt:            ">",
	TokenGe:            ">=",
	TokenAnd:           "&&",
	TokenOr:            "||",
	TokenNot:           "!",
	TokenAmpersand:     "&",
	TokenPipe:          "|",
	TokenCaret:         "^",
	TokenTilde:         "~",
	TokenShl:           "<<",
	TokenShr:           ">>",
	TokenQuestion:      "?",
	TokenColon:         ":",
	TokenPlusAssign:    "+=",
	TokenMinusAssign:   "-=",
	TokenStarAssign:    "*=",
	TokenSlashAssign:   "/=",
	TokenPercentAssign: "%=",
	TokenAndAssign:     "&=",
	TokenOrAssign:      "|=",
	TokenXorAssign:     "^=",
	TokenShlAssign:     "<<=",
	TokenShrAssign:     ">>=",
	TokenIncrement:     "++",
	TokenDecrement:     "--",
	TokenLParen:        "(",
	TokenRParen:        ")",
	TokenLBrace:        "{",
	TokenRBrace:        "}",
	TokenLBracket:      "[",
	TokenRBracket:      "]",
	TokenSemicolon:     ";",
	TokenComma:         ",",
	TokenDot:           ".",
	TokenArrow:         "->",
}

func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return "UNKNOWN"
}

// Token represents a lexical token
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

// keywords maps keyword strings to token types
var keywords = map[string]TokenType{
	"int":     TokenInt_,
	"void":    TokenVoid,
	"return":  TokenReturn,
	"if":      TokenIf,
	"else":    TokenElse,
	"while":   TokenWhile,
	"do":      TokenDo,
	"for":     TokenFor,
	"typedef": TokenTypedef,
	"struct":  TokenStruct,
	"sizeof":  TokenSizeof,
}

// LookupIdent returns the token type for an identifier (keyword or IDENT)
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return TokenIdent
}

// Package parser implements a recursive descent parser for C
package parser

import (
	"fmt"

	"github.com/raymyers/ralph-cc/pkg/cabs"
	"github.com/raymyers/ralph-cc/pkg/lexer"
)

// Precedence levels for Pratt parsing (lowest to highest)
const (
	precLowest     = 0
	precComma      = 1  // ,
	precAssign     = 2  // =, +=, -=, etc.
	precTernary    = 3  // ?:
	precOr         = 4  // ||
	precAnd        = 5  // &&
	precBitOr      = 6  // |
	precBitXor     = 7  // ^
	precBitAnd     = 8  // &
	precEquality   = 9  // ==, !=
	precRelational = 10 // <, <=, >, >=
	precShift      = 11 // <<, >>
	precAdditive   = 12 // +, -
	precMulti      = 13 // *, /, %
	precUnary      = 14 // -, !, ~, ++x, --x, &x, *x
	precPostfix    = 15 // function call, array subscript, member access, x++, x--
)

// Parser parses C source code into a Cabs AST
type Parser struct {
	l         *lexer.Lexer
	curToken  lexer.Token
	peekToken lexer.Token
	errors    []string
	typedefs  map[string]bool // typedef names in scope
}

// New creates a new Parser for the given lexer
func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:        l,
		typedefs: make(map[string]bool),
	}
	// Read two tokens to initialize curToken and peekToken
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

// Errors returns the list of parsing errors
func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) addError(msg string) {
	p.errors = append(p.errors, fmt.Sprintf("line %d, col %d: %s",
		p.curToken.Line, p.curToken.Column, msg))
}

func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.addError(fmt.Sprintf("expected %s, got %s", t, p.peekToken.Type))
	return false
}

func (p *Parser) expect(t lexer.TokenType) bool {
	if p.curTokenIs(t) {
		p.nextToken()
		return true
	}
	p.addError(fmt.Sprintf("expected %s, got %s", t, p.curToken.Type))
	return false
}

// syncToStmtEnd synchronizes to the end of a statement (';' or '}')
// Used for panic-mode error recovery within blocks
func (p *Parser) syncToStmtEnd() {
	for !p.curTokenIs(lexer.TokenEOF) {
		// Stop at semicolon (end of statement)
		if p.curTokenIs(lexer.TokenSemicolon) {
			p.nextToken() // consume ';'
			return
		}
		// Stop at closing brace (end of block)
		if p.curTokenIs(lexer.TokenRBrace) {
			return
		}
		// Stop at opening brace (start of new block) - don't consume
		if p.curTokenIs(lexer.TokenLBrace) {
			return
		}
		p.nextToken()
	}
}

// syncToBlockEnd synchronizes to matching closing brace
// Handles nested braces correctly
func (p *Parser) syncToBlockEnd() {
	depth := 1
	for !p.curTokenIs(lexer.TokenEOF) && depth > 0 {
		if p.curTokenIs(lexer.TokenLBrace) {
			depth++
		} else if p.curTokenIs(lexer.TokenRBrace) {
			depth--
		}
		if depth > 0 {
			p.nextToken()
		}
	}
}

// isStatementStart returns true if the current token can start a statement
func (p *Parser) isStatementStart() bool {
	switch p.curToken.Type {
	case lexer.TokenReturn, lexer.TokenIf, lexer.TokenWhile, lexer.TokenDo,
		lexer.TokenFor, lexer.TokenSwitch, lexer.TokenBreak, lexer.TokenContinue,
		lexer.TokenGoto, lexer.TokenLBrace, lexer.TokenSemicolon:
		return true
	}
	// Type specifiers, storage class specifiers, type qualifiers
	if p.isStorageClassSpecifier() || p.isTypeQualifier() || p.isTypeSpecifierKeyword() {
		return true
	}
	// Identifiers (could be expression or typedef name)
	if p.curTokenIs(lexer.TokenIdent) {
		return true
	}
	// Expression statement starters (literals, unary ops, etc.)
	switch p.curToken.Type {
	case lexer.TokenInt, lexer.TokenLParen, lexer.TokenStar, lexer.TokenAmpersand,
		lexer.TokenMinus, lexer.TokenNot, lexer.TokenTilde, lexer.TokenIncrement,
		lexer.TokenDecrement, lexer.TokenSizeof:
		return true
	}
	return false
}

// ParseDefinition parses a top-level definition (function, typedef, struct, union, or enum)
func (p *Parser) ParseDefinition() cabs.Definition {
	// Check for typedef
	if p.curTokenIs(lexer.TokenTypedef) {
		return p.parseTypedef()
	}

	// Check for struct definition
	if p.curTokenIs(lexer.TokenStruct) {
		return p.parseStructOrUnion(false)
	}

	// Check for union definition
	if p.curTokenIs(lexer.TokenUnion) {
		return p.parseStructOrUnion(true)
	}

	// Check for enum definition
	if p.curTokenIs(lexer.TokenEnum) {
		return p.parseEnumDef()
	}

	// Skip storage class specifiers for now
	for p.isStorageClassSpecifier() {
		p.nextToken()
	}

	// Skip type qualifiers
	for p.isTypeQualifier() {
		p.nextToken()
	}

	if !p.isTypeSpecifier() {
		p.addError(fmt.Sprintf("expected type specifier, got %s", p.curToken.Type))
		return nil
	}

	returnType := p.curToken.Literal
	p.nextToken()

	// Handle struct/union after seeing the keyword but followed by a definition body
	// (e.g., "struct Point { int x; int y; };")
	if (returnType == "struct" || returnType == "union") && p.curTokenIs(lexer.TokenIdent) && p.peekTokenIs(lexer.TokenLBrace) {
		name := p.curToken.Literal
		p.nextToken() // consume name
		return p.parseStructBody(name, returnType == "union")
	}

	// Handle pointer in return type
	for p.curTokenIs(lexer.TokenStar) {
		returnType = returnType + "*"
		p.nextToken()
	}

	if !p.curTokenIs(lexer.TokenIdent) {
		p.addError(fmt.Sprintf("expected function name, got %s", p.curToken.Type))
		return nil
	}
	name := p.curToken.Literal
	p.nextToken()

	// Parameter list
	if !p.expect(lexer.TokenLParen) {
		return nil
	}

	params := p.parseParameterList()

	if !p.curTokenIs(lexer.TokenRParen) {
		p.addError(fmt.Sprintf("expected ')' after parameters, got %s", p.curToken.Type))
		return nil
	}
	p.nextToken() // consume ')'

	// Function body
	if !p.curTokenIs(lexer.TokenLBrace) {
		p.addError(fmt.Sprintf("expected '{', got %s", p.curToken.Type))
		return nil
	}
	body := p.parseBlock()

	return cabs.FunDef{
		ReturnType: returnType,
		Name:       name,
		Params:     params,
		Body:       body,
	}
}

// parseStructOrUnion parses a struct or union definition
func (p *Parser) parseStructOrUnion(isUnion bool) cabs.Definition {
	p.nextToken() // consume 'struct' or 'union'

	// Check for struct name
	name := ""
	if p.curTokenIs(lexer.TokenIdent) {
		name = p.curToken.Literal
		p.nextToken()
	}

	// If no body, this is just a forward declaration or use of existing type
	if !p.curTokenIs(lexer.TokenLBrace) {
		// Forward declaration: struct Name;
		if p.curTokenIs(lexer.TokenSemicolon) {
			p.nextToken()
			if isUnion {
				return cabs.UnionDef{Name: name, Fields: nil}
			}
			return cabs.StructDef{Name: name, Fields: nil}
		}
		p.addError(fmt.Sprintf("expected '{' or ';' after struct/union name, got %s", p.curToken.Type))
		return nil
	}

	return p.parseStructBody(name, isUnion)
}

// parseStructBody parses the body of a struct or union definition
func (p *Parser) parseStructBody(name string, isUnion bool) cabs.Definition {
	p.nextToken() // consume '{'

	var fields []cabs.StructField

	for !p.curTokenIs(lexer.TokenRBrace) && !p.curTokenIs(lexer.TokenEOF) {
		// Parse field: type name;
		if !p.isTypeSpecifier() && !p.isTypeQualifier() {
			p.addError(fmt.Sprintf("expected type specifier in struct field, got %s", p.curToken.Type))
			p.nextToken()
			continue
		}

		// Skip type qualifiers
		for p.isTypeQualifier() {
			p.nextToken()
		}

		typeSpec := p.curToken.Literal
		p.nextToken()

		// Handle pointer types
		for p.curTokenIs(lexer.TokenStar) {
			typeSpec = typeSpec + "*"
			p.nextToken()
		}

		// Field name
		if !p.curTokenIs(lexer.TokenIdent) {
			p.addError(fmt.Sprintf("expected field name, got %s", p.curToken.Type))
			continue
		}
		fieldName := p.curToken.Literal
		p.nextToken()

		// Handle array fields
		for p.curTokenIs(lexer.TokenLBracket) {
			p.nextToken() // consume '['
			for !p.curTokenIs(lexer.TokenRBracket) && !p.curTokenIs(lexer.TokenEOF) {
				p.nextToken()
			}
			if p.curTokenIs(lexer.TokenRBracket) {
				p.nextToken()
			}
			typeSpec = typeSpec + "[]"
		}

		fields = append(fields, cabs.StructField{TypeSpec: typeSpec, Name: fieldName})

		// Expect semicolon
		if !p.expect(lexer.TokenSemicolon) {
			continue
		}
	}

	if !p.curTokenIs(lexer.TokenRBrace) {
		p.addError(fmt.Sprintf("expected '}' at end of struct body, got %s", p.curToken.Type))
		return nil
	}
	p.nextToken() // consume '}'

	// Optional trailing semicolon for struct definition
	if p.curTokenIs(lexer.TokenSemicolon) {
		p.nextToken()
	}

	if isUnion {
		return cabs.UnionDef{Name: name, Fields: fields}
	}
	return cabs.StructDef{Name: name, Fields: fields}
}

// parseEnumDef parses an enum definition
func (p *Parser) parseEnumDef() cabs.Definition {
	p.nextToken() // consume 'enum'

	name := ""
	if p.curTokenIs(lexer.TokenIdent) {
		name = p.curToken.Literal
		p.nextToken()
	}

	if !p.curTokenIs(lexer.TokenLBrace) {
		// Forward declaration
		if p.curTokenIs(lexer.TokenSemicolon) {
			p.nextToken()
			return cabs.EnumDef{Name: name, Values: nil}
		}
		p.addError(fmt.Sprintf("expected '{' or ';' after enum name, got %s", p.curToken.Type))
		return nil
	}
	p.nextToken() // consume '{'

	var values []cabs.EnumVal

	for !p.curTokenIs(lexer.TokenRBrace) && !p.curTokenIs(lexer.TokenEOF) {
		if !p.curTokenIs(lexer.TokenIdent) {
			p.addError(fmt.Sprintf("expected enumerator name, got %s", p.curToken.Type))
			break
		}

		enumName := p.curToken.Literal
		p.nextToken()

		var value cabs.Expr
		if p.curTokenIs(lexer.TokenAssign) {
			p.nextToken() // consume '='
			value = p.parseExprPrec(precAssign)
		}

		values = append(values, cabs.EnumVal{Name: enumName, Value: value})

		if p.curTokenIs(lexer.TokenComma) {
			p.nextToken()
		} else {
			break
		}
	}

	if !p.curTokenIs(lexer.TokenRBrace) {
		p.addError(fmt.Sprintf("expected '}' at end of enum, got %s", p.curToken.Type))
		return nil
	}
	p.nextToken() // consume '}'

	// Optional trailing semicolon
	if p.curTokenIs(lexer.TokenSemicolon) {
		p.nextToken()
	}

	return cabs.EnumDef{Name: name, Values: values}
}

// parseParameterList parses function parameters: (type name, type name, ...)
func (p *Parser) parseParameterList() []cabs.Param {
	var params []cabs.Param

	// Empty parameter list or void
	if p.curTokenIs(lexer.TokenRParen) {
		return params
	}
	if p.curTokenIs(lexer.TokenVoid) && p.peekTokenIs(lexer.TokenRParen) {
		p.nextToken() // consume 'void'
		return params
	}

	// Parse first parameter
	param := p.parseParameter()
	if param != nil {
		params = append(params, *param)
	}

	// Parse remaining parameters
	for p.curTokenIs(lexer.TokenComma) {
		p.nextToken() // consume ','
		param := p.parseParameter()
		if param != nil {
			params = append(params, *param)
		}
	}

	return params
}

// parseParameter parses a single function parameter: type name
func (p *Parser) parseParameter() *cabs.Param {
	// Skip type qualifiers
	for p.isTypeQualifier() {
		p.nextToken()
	}

	if !p.isTypeSpecifier() {
		p.addError(fmt.Sprintf("expected type specifier in parameter, got %s", p.curToken.Type))
		return nil
	}

	typeSpec := p.curToken.Literal
	p.nextToken()

	// Handle pointer types
	for p.curTokenIs(lexer.TokenStar) {
		typeSpec = typeSpec + "*"
		p.nextToken()
	}

	// Parameter name is optional in declarations, but we require it for now
	name := ""
	if p.curTokenIs(lexer.TokenIdent) {
		name = p.curToken.Literal
		p.nextToken()
	}

	// Handle array parameters like int arr[]
	for p.curTokenIs(lexer.TokenLBracket) {
		p.nextToken() // consume '['
		// Skip array size if present
		for !p.curTokenIs(lexer.TokenRBracket) && !p.curTokenIs(lexer.TokenEOF) {
			p.nextToken()
		}
		if p.curTokenIs(lexer.TokenRBracket) {
			p.nextToken() // consume ']'
		}
		typeSpec = typeSpec + "[]"
	}

	return &cabs.Param{TypeSpec: typeSpec, Name: name}
}

// parseTypedef parses a typedef declaration
func (p *Parser) parseTypedef() cabs.Definition {
	p.nextToken() // consume 'typedef'

	if !p.isTypeSpecifier() {
		p.addError(fmt.Sprintf("expected type specifier in typedef, got %s", p.curToken.Type))
		return nil
	}

	typeSpec := p.curToken.Literal
	p.nextToken()

	// Handle pointer types
	for p.curTokenIs(lexer.TokenStar) {
		typeSpec = typeSpec + "*"
		p.nextToken()
	}

	if !p.curTokenIs(lexer.TokenIdent) {
		p.addError(fmt.Sprintf("expected typedef name, got %s", p.curToken.Type))
		return nil
	}

	name := p.curToken.Literal
	p.nextToken()

	if !p.expect(lexer.TokenSemicolon) {
		return nil
	}

	// Register the typedef name
	p.typedefs[name] = true

	return cabs.TypedefDef{TypeSpec: typeSpec, Name: name}
}

func (p *Parser) isTypeSpecifier() bool {
	switch p.curToken.Type {
	case lexer.TokenInt_, lexer.TokenVoid, lexer.TokenChar, lexer.TokenShort,
		lexer.TokenLong, lexer.TokenFloat, lexer.TokenDouble,
		lexer.TokenSigned, lexer.TokenUnsigned, lexer.TokenStruct,
		lexer.TokenUnion, lexer.TokenEnum:
		return true
	case lexer.TokenIdent:
		// Check if it's a typedef name
		return p.typedefs[p.curToken.Literal]
	}
	return false
}

func (p *Parser) isStorageClassSpecifier() bool {
	switch p.curToken.Type {
	case lexer.TokenStatic, lexer.TokenExtern, lexer.TokenAuto, lexer.TokenRegister:
		return true
	}
	return false
}

func (p *Parser) isTypeQualifier() bool {
	switch p.curToken.Type {
	case lexer.TokenConst, lexer.TokenVolatile, lexer.TokenRestrict:
		return true
	}
	return false
}

// isDeclarationStart checks if current token starts a declaration
func (p *Parser) isDeclarationStart() bool {
	return p.isStorageClassSpecifier() || p.isTypeQualifier() || p.isTypeSpecifier()
}

func (p *Parser) parseBlock() *cabs.Block {
	block := &cabs.Block{Items: []cabs.Stmt{}}

	p.nextToken() // consume '{'

	for !p.curTokenIs(lexer.TokenRBrace) && !p.curTokenIs(lexer.TokenEOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Items = append(block.Items, stmt)
		} else {
			// Error recovery: sync to end of statement and continue
			p.syncToStmtEnd()
		}
	}

	p.nextToken() // consume '}'

	return block
}

func (p *Parser) parseStatement() cabs.Stmt {
	// Check for declarations first (they can start with storage class, type qualifier, or type specifier)
	if p.isStorageClassSpecifier() || p.isTypeQualifier() || p.isTypeSpecifierKeyword() {
		return p.parseDeclarationStatement()
	}

	switch p.curToken.Type {
	case lexer.TokenReturn:
		return p.parseReturnStatement()
	case lexer.TokenIf:
		return p.parseIfStatement()
	case lexer.TokenWhile:
		return p.parseWhileStatement()
	case lexer.TokenDo:
		return p.parseDoWhileStatement()
	case lexer.TokenFor:
		return p.parseForStatement()
	case lexer.TokenSwitch:
		return p.parseSwitchStatement()
	case lexer.TokenBreak:
		return p.parseBreakStatement()
	case lexer.TokenContinue:
		return p.parseContinueStatement()
	case lexer.TokenGoto:
		return p.parseGotoStatement()
	case lexer.TokenLBrace:
		return p.parseBlock()
	case lexer.TokenIdent:
		// Check for label: identifier followed by ':'
		if p.peekTokenIs(lexer.TokenColon) {
			return p.parseLabelStatement()
		}
		// Check if it's a typedef name (declaration)
		if p.typedefs[p.curToken.Literal] {
			return p.parseDeclarationStatement()
		}
		// Expression statement
		return p.parseExpressionStatement()
	default:
		// Expression statement: expr;
		return p.parseExpressionStatement()
	}
}

// isTypeSpecifierKeyword checks if current token is a type specifier keyword (not typedef name)
func (p *Parser) isTypeSpecifierKeyword() bool {
	switch p.curToken.Type {
	case lexer.TokenInt_, lexer.TokenVoid, lexer.TokenChar, lexer.TokenShort,
		lexer.TokenLong, lexer.TokenFloat, lexer.TokenDouble,
		lexer.TokenSigned, lexer.TokenUnsigned, lexer.TokenStruct,
		lexer.TokenUnion, lexer.TokenEnum:
		return true
	}
	return false
}

// parseDeclarationStatement parses a variable declaration: type name [= initializer], ...;
func (p *Parser) parseDeclarationStatement() cabs.Stmt {
	// Collect storage class specifiers (skip for now, just consume)
	for p.isStorageClassSpecifier() {
		p.nextToken()
	}

	// Collect type qualifiers (skip for now, just consume)
	for p.isTypeQualifier() {
		p.nextToken()
	}

	// Parse base type
	if !p.isTypeSpecifier() {
		p.addError(fmt.Sprintf("expected type specifier, got %s", p.curToken.Type))
		return nil
	}

	baseType := p.curToken.Literal
	p.nextToken()

	var decls []cabs.Decl

	// Parse declarators
	for {
		typeSpec := baseType

		// Check for function pointer: type (*name)(params)
		if p.curTokenIs(lexer.TokenLParen) && p.peekTokenIs(lexer.TokenStar) {
			p.nextToken() // consume '('
			p.nextToken() // consume '*'

			// Collect pointer modifiers
			ptrDepth := 1
			for p.curTokenIs(lexer.TokenStar) {
				ptrDepth++
				p.nextToken()
			}

			// Get the name
			if !p.curTokenIs(lexer.TokenIdent) {
				p.addError(fmt.Sprintf("expected identifier in function pointer, got %s", p.curToken.Type))
				return nil
			}
			name := p.curToken.Literal
			p.nextToken()

			if !p.expect(lexer.TokenRParen) {
				return nil
			}

			// Parse the function parameter list
			if !p.curTokenIs(lexer.TokenLParen) {
				p.addError(fmt.Sprintf("expected '(' for function pointer parameters, got %s", p.curToken.Type))
				return nil
			}
			paramList := p.parseFunctionPointerParams()

			// Build the type spec: returnType(*)(params)
			ptrStr := "(*)"
			for i := 1; i < ptrDepth; i++ {
				ptrStr = "(*" + ptrStr + ")"
			}
			typeSpec = typeSpec + ptrStr + "(" + paramList + ")"

			var init cabs.Expr
			// Check for initializer
			if p.curTokenIs(lexer.TokenAssign) {
				p.nextToken() // consume '='
				init = p.parseExprPrec(precAssign)
				if init == nil {
					return nil
				}
			}

			decls = append(decls, cabs.Decl{
				TypeSpec:    typeSpec,
				Name:        name,
				Initializer: init,
			})
		} else {
			// Regular declarator: pointer and/or identifier
			// Skip pointer declarators (*)
			for p.curTokenIs(lexer.TokenStar) {
				typeSpec = typeSpec + "*"
				p.nextToken()
			}

			// Expect identifier
			if !p.curTokenIs(lexer.TokenIdent) {
				p.addError(fmt.Sprintf("expected identifier in declaration, got %s", p.curToken.Type))
				return nil
			}
			name := p.curToken.Literal
			p.nextToken()

			// Check for array declarator
			var arrayDims []cabs.Expr
			for p.curTokenIs(lexer.TokenLBracket) {
				p.nextToken() // consume '['
				if p.curTokenIs(lexer.TokenRBracket) {
					// Empty array dimension: int arr[]
					arrayDims = append(arrayDims, nil)
				} else {
					// Parse size expression (supports VLAs and constant expressions)
					sizeExpr := p.parseExprPrec(precAssign)
					if sizeExpr == nil {
						return nil
					}
					arrayDims = append(arrayDims, sizeExpr)
				}
				if !p.expect(lexer.TokenRBracket) {
					return nil
				}
			}

			var init cabs.Expr
			// Check for initializer
			if p.curTokenIs(lexer.TokenAssign) {
				p.nextToken() // consume '='
				init = p.parseExprPrec(precAssign) // Use assignment precedence to stop at comma
				if init == nil {
					return nil
				}
			}

			decls = append(decls, cabs.Decl{
				TypeSpec:    typeSpec,
				Name:        name,
				ArrayDims:   arrayDims,
				Initializer: init,
			})
		}

		// Check for more declarators
		if !p.curTokenIs(lexer.TokenComma) {
			break
		}
		p.nextToken() // consume ','
	}

	if !p.expect(lexer.TokenSemicolon) {
		return nil
	}

	return cabs.DeclStmt{Decls: decls}
}

// parseFunctionPointerParams parses the parameter list in a function pointer type
// Returns a string representation like "int,int" or "void"
func (p *Parser) parseFunctionPointerParams() string {
	p.nextToken() // consume '('

	var params []string

	if p.curTokenIs(lexer.TokenRParen) {
		p.nextToken() // consume ')'
		return ""
	}

	// Parse first parameter type
	for !p.curTokenIs(lexer.TokenRParen) && !p.curTokenIs(lexer.TokenEOF) {
		var paramType string
		// Collect type qualifiers
		for p.isTypeQualifier() {
			p.nextToken()
		}
		// Get the type specifier
		if p.isTypeSpecifier() {
			paramType = p.curToken.Literal
			p.nextToken()
		}
		// Handle pointers
		for p.curTokenIs(lexer.TokenStar) {
			paramType = paramType + "*"
			p.nextToken()
		}
		// Skip parameter name if present
		if p.curTokenIs(lexer.TokenIdent) {
			p.nextToken()
		}
		params = append(params, paramType)

		if p.curTokenIs(lexer.TokenComma) {
			p.nextToken() // consume ','
		} else {
			break
		}
	}

	if !p.expect(lexer.TokenRParen) {
		return ""
	}

	result := ""
	for i, param := range params {
		if i > 0 {
			result += ","
		}
		result += param
	}
	return result
}

func (p *Parser) parseExpressionStatement() cabs.Stmt {
	expr := p.parseExpression()
	if expr == nil {
		return nil
	}

	if !p.expect(lexer.TokenSemicolon) {
		return nil
	}

	return cabs.Computation{Expr: expr}
}

func (p *Parser) parseIfStatement() cabs.Stmt {
	p.nextToken() // consume 'if'

	if !p.expect(lexer.TokenLParen) {
		return nil
	}

	cond := p.parseExpression()
	if cond == nil {
		return nil
	}

	if !p.expect(lexer.TokenRParen) {
		return nil
	}

	then := p.parseStatement()
	if then == nil {
		return nil
	}

	var els cabs.Stmt
	if p.curTokenIs(lexer.TokenElse) {
		p.nextToken() // consume 'else'
		els = p.parseStatement()
	}

	return cabs.If{Cond: cond, Then: then, Else: els}
}

func (p *Parser) parseWhileStatement() cabs.Stmt {
	p.nextToken() // consume 'while'

	if !p.expect(lexer.TokenLParen) {
		return nil
	}

	cond := p.parseExpression()
	if cond == nil {
		return nil
	}

	if !p.expect(lexer.TokenRParen) {
		return nil
	}

	body := p.parseStatement()
	if body == nil {
		return nil
	}

	return cabs.While{Cond: cond, Body: body}
}

func (p *Parser) parseDoWhileStatement() cabs.Stmt {
	p.nextToken() // consume 'do'

	body := p.parseStatement()
	if body == nil {
		return nil
	}

	if !p.curTokenIs(lexer.TokenWhile) {
		p.addError(fmt.Sprintf("expected 'while' after do body, got %s", p.curToken.Type))
		return nil
	}
	p.nextToken() // consume 'while'

	if !p.expect(lexer.TokenLParen) {
		return nil
	}

	cond := p.parseExpression()
	if cond == nil {
		return nil
	}

	if !p.expect(lexer.TokenRParen) {
		return nil
	}

	if !p.expect(lexer.TokenSemicolon) {
		return nil
	}

	return cabs.DoWhile{Body: body, Cond: cond}
}

func (p *Parser) parseForStatement() cabs.Stmt {
	p.nextToken() // consume 'for'

	if !p.expect(lexer.TokenLParen) {
		return nil
	}

	// Parse init expression (optional)
	var init cabs.Expr
	if !p.curTokenIs(lexer.TokenSemicolon) {
		init = p.parseExpression()
	}

	if !p.expect(lexer.TokenSemicolon) {
		return nil
	}

	// Parse condition expression (optional)
	var cond cabs.Expr
	if !p.curTokenIs(lexer.TokenSemicolon) {
		cond = p.parseExpression()
	}

	if !p.expect(lexer.TokenSemicolon) {
		return nil
	}

	// Parse step expression (optional)
	var step cabs.Expr
	if !p.curTokenIs(lexer.TokenRParen) {
		step = p.parseExpression()
	}

	if !p.expect(lexer.TokenRParen) {
		return nil
	}

	body := p.parseStatement()
	if body == nil {
		return nil
	}

	return cabs.For{Init: init, Cond: cond, Step: step, Body: body}
}

func (p *Parser) parseBreakStatement() cabs.Stmt {
	p.nextToken() // consume 'break'

	if !p.expect(lexer.TokenSemicolon) {
		return nil
	}

	return cabs.Break{}
}

func (p *Parser) parseContinueStatement() cabs.Stmt {
	p.nextToken() // consume 'continue'

	if !p.expect(lexer.TokenSemicolon) {
		return nil
	}

	return cabs.Continue{}
}

func (p *Parser) parseSwitchStatement() cabs.Stmt {
	p.nextToken() // consume 'switch'

	if !p.expect(lexer.TokenLParen) {
		return nil
	}

	expr := p.parseExpression()
	if expr == nil {
		return nil
	}

	if !p.expect(lexer.TokenRParen) {
		return nil
	}

	if !p.curTokenIs(lexer.TokenLBrace) {
		p.addError(fmt.Sprintf("expected '{' after switch condition, got %s", p.curToken.Type))
		return nil
	}
	p.nextToken() // consume '{'

	var cases []cabs.SwitchCase
	for !p.curTokenIs(lexer.TokenRBrace) && !p.curTokenIs(lexer.TokenEOF) {
		c := p.parseSwitchCase()
		if c != nil {
			cases = append(cases, *c)
		}
	}

	p.nextToken() // consume '}'

	return cabs.Switch{Expr: expr, Cases: cases}
}

func (p *Parser) parseSwitchCase() *cabs.SwitchCase {
	var caseExpr cabs.Expr

	if p.curTokenIs(lexer.TokenCase) {
		p.nextToken() // consume 'case'
		caseExpr = p.parseExpression()
		if caseExpr == nil {
			return nil
		}
	} else if p.curTokenIs(lexer.TokenDefault) {
		p.nextToken() // consume 'default'
		// caseExpr remains nil for default
	} else {
		p.addError(fmt.Sprintf("expected 'case' or 'default' in switch, got %s", p.curToken.Type))
		return nil
	}

	if !p.expect(lexer.TokenColon) {
		return nil
	}

	// Parse statements until we hit case, default, or }
	var stmts []cabs.Stmt
	for !p.curTokenIs(lexer.TokenCase) && !p.curTokenIs(lexer.TokenDefault) &&
		!p.curTokenIs(lexer.TokenRBrace) && !p.curTokenIs(lexer.TokenEOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
	}

	return &cabs.SwitchCase{Expr: caseExpr, Stmts: stmts}
}

func (p *Parser) parseGotoStatement() cabs.Stmt {
	p.nextToken() // consume 'goto'

	if !p.curTokenIs(lexer.TokenIdent) {
		p.addError(fmt.Sprintf("expected label name after goto, got %s", p.curToken.Type))
		return nil
	}
	label := p.curToken.Literal
	p.nextToken() // consume label

	if !p.expect(lexer.TokenSemicolon) {
		return nil
	}

	return cabs.Goto{Label: label}
}

func (p *Parser) parseLabelStatement() cabs.Stmt {
	label := p.curToken.Literal
	p.nextToken() // consume label name
	p.nextToken() // consume ':'

	stmt := p.parseStatement()
	if stmt == nil {
		return nil
	}

	return cabs.Label{Name: label, Stmt: stmt}
}

func (p *Parser) parseReturnStatement() cabs.Stmt {
	p.nextToken() // consume 'return'

	var expr cabs.Expr
	if !p.curTokenIs(lexer.TokenSemicolon) {
		expr = p.parseExpression()
	}

	if !p.expect(lexer.TokenSemicolon) {
		return nil
	}

	return cabs.Return{Expr: expr}
}

// parseExpression is the entry point for expression parsing
func (p *Parser) parseExpression() cabs.Expr {
	return p.parseExprPrec(precLowest)
}

// parseExprPrec implements Pratt parsing with the given precedence level
// After calling parsePrefix, curToken is positioned on the token AFTER the prefix expression
func (p *Parser) parseExprPrec(prec int) cabs.Expr {
	left := p.parsePrefix()
	if left == nil {
		return nil
	}

	// curToken is now positioned on the potential infix operator
	for prec < p.curPrecedence() {
		left = p.parseInfix(left)
		if left == nil {
			return nil
		}
	}

	return left
}

// parsePrefix parses prefix expressions: literals, identifiers, unary ops, parentheses
// After parsing, curToken is positioned on the token AFTER the prefix expression
func (p *Parser) parsePrefix() cabs.Expr {
	switch p.curToken.Type {
	case lexer.TokenInt:
		return p.parseIntegerLiteral()
	case lexer.TokenIdent:
		return p.parseIdentifier()
	case lexer.TokenLParen:
		return p.parseGroupedExpression()
	case lexer.TokenMinus:
		return p.parsePrefixUnary(cabs.OpNeg)
	case lexer.TokenNot:
		return p.parsePrefixUnary(cabs.OpNot)
	case lexer.TokenTilde:
		return p.parsePrefixUnary(cabs.OpBitNot)
	case lexer.TokenIncrement:
		return p.parsePrefixUnary(cabs.OpPreInc)
	case lexer.TokenDecrement:
		return p.parsePrefixUnary(cabs.OpPreDec)
	case lexer.TokenAmpersand:
		return p.parsePrefixUnary(cabs.OpAddrOf)
	case lexer.TokenStar:
		return p.parsePrefixUnary(cabs.OpDeref)
	case lexer.TokenSizeof:
		return p.parseSizeof()
	default:
		p.addError(fmt.Sprintf("expected expression, got %s", p.curToken.Type))
		return nil
	}
}

func (p *Parser) parseIntegerLiteral() cabs.Expr {
	lit := p.curToken.Literal
	var value int64
	fmt.Sscanf(lit, "%d", &value)
	p.nextToken() // move past the literal
	return cabs.Constant{Value: value}
}

func (p *Parser) parseIdentifier() cabs.Expr {
	name := p.curToken.Literal
	p.nextToken() // move past the identifier
	return cabs.Variable{Name: name}
}

func (p *Parser) parseGroupedExpression() cabs.Expr {
	// Disambiguate: (type)expr vs (expr)
	// If we see '(' followed by a type specifier followed by ')', it's a cast
	if p.isTypeSpecifierPeek() {
		return p.parseCast()
	}

	p.nextToken() // consume '('

	expr := p.parseExpression()
	if expr == nil {
		return nil
	}

	if !p.curTokenIs(lexer.TokenRParen) {
		p.addError(fmt.Sprintf("expected ')', got %s", p.curToken.Type))
		return nil
	}
	p.nextToken() // consume ')'

	return cabs.Paren{Expr: expr}
}

// parseCast parses a cast expression: (type)expr
func (p *Parser) parseCast() cabs.Expr {
	p.nextToken() // consume '('
	typeName := p.curToken.Literal
	p.nextToken() // consume type name

	if !p.curTokenIs(lexer.TokenRParen) {
		p.addError(fmt.Sprintf("expected ')' after type in cast, got %s", p.curToken.Type))
		return nil
	}
	p.nextToken() // consume ')'

	// Cast has same precedence as unary operators
	expr := p.parseExprPrec(precUnary)
	if expr == nil {
		return nil
	}

	return cabs.Cast{TypeName: typeName, Expr: expr}
}

func (p *Parser) parsePrefixUnary(op cabs.UnaryOp) cabs.Expr {
	p.nextToken() // consume operator

	expr := p.parseExprPrec(precUnary)
	if expr == nil {
		return nil
	}

	return cabs.Unary{Op: op, Expr: expr}
}

// parseSizeof parses sizeof expressions: sizeof expr or sizeof(type) or sizeof(expr)
func (p *Parser) parseSizeof() cabs.Expr {
	p.nextToken() // consume 'sizeof'

	// Check if followed by '('
	if p.curTokenIs(lexer.TokenLParen) {
		// Could be sizeof(type) or sizeof(expr)
		// For now, check if the token after '(' is a type specifier
		if p.isTypeSpecifierPeek() {
			// sizeof(type)
			p.nextToken() // consume '('
			typeName := p.curToken.Literal
			p.nextToken() // consume type name
			if !p.curTokenIs(lexer.TokenRParen) {
				p.addError(fmt.Sprintf("expected ')' after type in sizeof, got %s", p.curToken.Type))
				return nil
			}
			p.nextToken() // consume ')'
			return cabs.SizeofType{TypeName: typeName}
		}
		// sizeof(expr) - parse as expression, the parentheses will be part of the expression
	}

	// sizeof expr (unary prefix)
	expr := p.parseExprPrec(precUnary)
	if expr == nil {
		return nil
	}
	return cabs.SizeofExpr{Expr: expr}
}

// isTypeSpecifierPeek checks if the peek token is a type specifier (for sizeof disambiguation)
func (p *Parser) isTypeSpecifierPeek() bool {
	switch p.peekToken.Type {
	case lexer.TokenInt_, lexer.TokenVoid, lexer.TokenChar, lexer.TokenShort,
		lexer.TokenLong, lexer.TokenFloat, lexer.TokenDouble,
		lexer.TokenSigned, lexer.TokenUnsigned, lexer.TokenStruct,
		lexer.TokenUnion, lexer.TokenEnum:
		return true
	case lexer.TokenIdent:
		return p.typedefs[p.peekToken.Literal]
	}
	return false
}

// parseInfix parses infix (binary) expressions
// curToken is on the operator when called
// After parsing, curToken is positioned on the token AFTER the expression
func (p *Parser) parseInfix(left cabs.Expr) cabs.Expr {
	// Special case for ternary operator
	if p.curTokenIs(lexer.TokenQuestion) {
		return p.parseTernary(left)
	}

	// Special case for function call
	if p.curTokenIs(lexer.TokenLParen) {
		return p.parseCall(left)
	}

	// Special case for array subscript
	if p.curTokenIs(lexer.TokenLBracket) {
		return p.parseIndex(left)
	}

	// Special case for member access
	if p.curTokenIs(lexer.TokenDot) || p.curTokenIs(lexer.TokenArrow) {
		return p.parseMember(left)
	}

	// Special case for postfix increment/decrement
	if p.curTokenIs(lexer.TokenIncrement) {
		p.nextToken()
		return cabs.Unary{Op: cabs.OpPostInc, Expr: left}
	}
	if p.curTokenIs(lexer.TokenDecrement) {
		p.nextToken()
		return cabs.Unary{Op: cabs.OpPostDec, Expr: left}
	}

	op, ok := p.tokenToBinaryOp()
	if !ok {
		p.addError(fmt.Sprintf("unexpected infix operator: %s", p.curToken.Type))
		return nil
	}

	prec := p.curPrecedence()
	p.nextToken() // consume operator

	// Right-associative for all assignment operators
	if isAssignOp(op) {
		prec--
	}

	right := p.parseExprPrec(prec)
	if right == nil {
		return nil
	}

	return cabs.Binary{Op: op, Left: left, Right: right}
}

// isAssignOp returns true if the operator is an assignment operator
func isAssignOp(op cabs.BinaryOp) bool {
	switch op {
	case cabs.OpAssign, cabs.OpAddAssign, cabs.OpSubAssign, cabs.OpMulAssign,
		cabs.OpDivAssign, cabs.OpModAssign, cabs.OpAndAssign, cabs.OpOrAssign,
		cabs.OpXorAssign, cabs.OpShlAssign, cabs.OpShrAssign:
		return true
	}
	return false
}

// parseMember parses member access: s.x or p->y
func (p *Parser) parseMember(expr cabs.Expr) cabs.Expr {
	isArrow := p.curTokenIs(lexer.TokenArrow)
	p.nextToken() // consume '.' or '->'

	if !p.curTokenIs(lexer.TokenIdent) {
		p.addError(fmt.Sprintf("expected member name, got %s", p.curToken.Type))
		return nil
	}
	name := p.curToken.Literal
	p.nextToken() // consume member name

	return cabs.Member{Expr: expr, Name: name, IsArrow: isArrow}
}

// parseTernary parses the ternary operator: cond ? then : else
func (p *Parser) parseTernary(cond cabs.Expr) cabs.Expr {
	p.nextToken() // consume '?'

	// Parse the 'then' expression (can include any operator, even comma)
	then := p.parseExpression()
	if then == nil {
		return nil
	}

	if !p.curTokenIs(lexer.TokenColon) {
		p.addError(fmt.Sprintf("expected ':' in ternary, got %s", p.curToken.Type))
		return nil
	}
	p.nextToken() // consume ':'

	// Parse the 'else' expression with ternary precedence (right-associative)
	els := p.parseExprPrec(precTernary - 1)
	if els == nil {
		return nil
	}

	return cabs.Conditional{Cond: cond, Then: then, Else: els}
}

// parseCall parses a function call: f() or f(a, b, c)
func (p *Parser) parseCall(fn cabs.Expr) cabs.Expr {
	p.nextToken() // consume '('

	var args []cabs.Expr

	if !p.curTokenIs(lexer.TokenRParen) {
		// Parse first argument
		arg := p.parseExprPrec(precAssign) // Use assignment precedence to avoid comma confusion
		if arg == nil {
			return nil
		}
		args = append(args, arg)

		// Parse remaining arguments
		for p.curTokenIs(lexer.TokenComma) {
			p.nextToken() // consume ','
			arg := p.parseExprPrec(precAssign)
			if arg == nil {
				return nil
			}
			args = append(args, arg)
		}
	}

	if !p.curTokenIs(lexer.TokenRParen) {
		p.addError(fmt.Sprintf("expected ')' in call, got %s", p.curToken.Type))
		return nil
	}
	p.nextToken() // consume ')'

	return cabs.Call{Func: fn, Args: args}
}

// parseIndex parses array subscript: arr[idx]
func (p *Parser) parseIndex(arr cabs.Expr) cabs.Expr {
	p.nextToken() // consume '['

	idx := p.parseExpression()
	if idx == nil {
		return nil
	}

	if !p.curTokenIs(lexer.TokenRBracket) {
		p.addError(fmt.Sprintf("expected ']' in subscript, got %s", p.curToken.Type))
		return nil
	}
	p.nextToken() // consume ']'

	return cabs.Index{Array: arr, Index: idx}
}

// tokenToBinaryOp converts the current token to a binary operator
func (p *Parser) tokenToBinaryOp() (cabs.BinaryOp, bool) {
	switch p.curToken.Type {
	case lexer.TokenPlus:
		return cabs.OpAdd, true
	case lexer.TokenMinus:
		return cabs.OpSub, true
	case lexer.TokenStar:
		return cabs.OpMul, true
	case lexer.TokenSlash:
		return cabs.OpDiv, true
	case lexer.TokenPercent:
		return cabs.OpMod, true
	case lexer.TokenLt:
		return cabs.OpLt, true
	case lexer.TokenLe:
		return cabs.OpLe, true
	case lexer.TokenGt:
		return cabs.OpGt, true
	case lexer.TokenGe:
		return cabs.OpGe, true
	case lexer.TokenEq:
		return cabs.OpEq, true
	case lexer.TokenNe:
		return cabs.OpNe, true
	case lexer.TokenAnd:
		return cabs.OpAnd, true
	case lexer.TokenOr:
		return cabs.OpOr, true
	case lexer.TokenAmpersand:
		return cabs.OpBitAnd, true
	case lexer.TokenPipe:
		return cabs.OpBitOr, true
	case lexer.TokenCaret:
		return cabs.OpBitXor, true
	case lexer.TokenShl:
		return cabs.OpShl, true
	case lexer.TokenShr:
		return cabs.OpShr, true
	case lexer.TokenAssign:
		return cabs.OpAssign, true
	case lexer.TokenPlusAssign:
		return cabs.OpAddAssign, true
	case lexer.TokenMinusAssign:
		return cabs.OpSubAssign, true
	case lexer.TokenStarAssign:
		return cabs.OpMulAssign, true
	case lexer.TokenSlashAssign:
		return cabs.OpDivAssign, true
	case lexer.TokenPercentAssign:
		return cabs.OpModAssign, true
	case lexer.TokenAndAssign:
		return cabs.OpAndAssign, true
	case lexer.TokenOrAssign:
		return cabs.OpOrAssign, true
	case lexer.TokenXorAssign:
		return cabs.OpXorAssign, true
	case lexer.TokenShlAssign:
		return cabs.OpShlAssign, true
	case lexer.TokenShrAssign:
		return cabs.OpShrAssign, true
	case lexer.TokenComma:
		return cabs.OpComma, true
	default:
		return 0, false
	}
}

// precedences maps token types to their precedence levels
func tokenPrecedence(t lexer.TokenType) int {
	switch t {
	case lexer.TokenComma:
		return precComma
	case lexer.TokenAssign, lexer.TokenPlusAssign, lexer.TokenMinusAssign,
		lexer.TokenStarAssign, lexer.TokenSlashAssign, lexer.TokenPercentAssign,
		lexer.TokenAndAssign, lexer.TokenOrAssign, lexer.TokenXorAssign,
		lexer.TokenShlAssign, lexer.TokenShrAssign:
		return precAssign
	case lexer.TokenQuestion:
		return precTernary
	case lexer.TokenOr:
		return precOr
	case lexer.TokenAnd:
		return precAnd
	case lexer.TokenPipe:
		return precBitOr
	case lexer.TokenCaret:
		return precBitXor
	case lexer.TokenAmpersand:
		return precBitAnd
	case lexer.TokenEq, lexer.TokenNe:
		return precEquality
	case lexer.TokenLt, lexer.TokenLe, lexer.TokenGt, lexer.TokenGe:
		return precRelational
	case lexer.TokenShl, lexer.TokenShr:
		return precShift
	case lexer.TokenPlus, lexer.TokenMinus:
		return precAdditive
	case lexer.TokenStar, lexer.TokenSlash, lexer.TokenPercent:
		return precMulti
	case lexer.TokenLParen, lexer.TokenLBracket, lexer.TokenDot, lexer.TokenArrow,
		lexer.TokenIncrement, lexer.TokenDecrement:
		return precPostfix
	default:
		return precLowest
	}
}

func (p *Parser) curPrecedence() int {
	return tokenPrecedence(p.curToken.Type)
}

func (p *Parser) peekPrecedence() int {
	return tokenPrecedence(p.peekToken.Type)
}

// ParseProgram parses a complete translation unit (file) containing multiple definitions
func (p *Parser) ParseProgram() *cabs.Program {
	program := &cabs.Program{
		Definitions: []cabs.Definition{},
	}

	for !p.curTokenIs(lexer.TokenEOF) {
		def := p.ParseDefinition()
		if def != nil {
			program.Definitions = append(program.Definitions, def)
		} else {
			// Skip to next definition on error
			p.skipToNextDefinition()
		}
	}

	return program
}

// skipToNextDefinition skips tokens until we find a likely start of a new definition
func (p *Parser) skipToNextDefinition() {
	for !p.curTokenIs(lexer.TokenEOF) {
		// Stop at tokens that typically start definitions
		if p.isTypeSpecifier() || p.curTokenIs(lexer.TokenTypedef) ||
			p.curTokenIs(lexer.TokenStruct) || p.curTokenIs(lexer.TokenUnion) ||
			p.curTokenIs(lexer.TokenEnum) || p.isStorageClassSpecifier() {
			return
		}
		p.nextToken()
	}
}

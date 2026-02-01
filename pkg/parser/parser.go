// Package parser implements a recursive descent parser for C
package parser

import (
	"fmt"

	"github.com/raymyers/ralph-cc/pkg/cabs"
	"github.com/raymyers/ralph-cc/pkg/lexer"
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

// ParseDefinition parses a top-level definition (function)
func (p *Parser) ParseDefinition() cabs.Definition {
	// For now, only parse function definitions: type name() { body }
	if !p.isTypeSpecifier() {
		p.addError(fmt.Sprintf("expected type specifier, got %s", p.curToken.Type))
		return nil
	}

	returnType := p.curToken.Literal
	p.nextToken()

	if !p.curTokenIs(lexer.TokenIdent) {
		p.addError(fmt.Sprintf("expected function name, got %s", p.curToken.Type))
		return nil
	}
	name := p.curToken.Literal
	p.nextToken()

	// Parameter list (empty for now)
	if !p.expect(lexer.TokenLParen) {
		return nil
	}
	if !p.expect(lexer.TokenRParen) {
		return nil
	}

	// Function body
	if !p.curTokenIs(lexer.TokenLBrace) {
		p.addError(fmt.Sprintf("expected '{', got %s", p.curToken.Type))
		return nil
	}
	body := p.parseBlock()

	return cabs.FunDef{
		ReturnType: returnType,
		Name:       name,
		Body:       body,
	}
}

func (p *Parser) isTypeSpecifier() bool {
	switch p.curToken.Type {
	case lexer.TokenInt_, lexer.TokenVoid:
		return true
	case lexer.TokenIdent:
		// Check if it's a typedef name
		return p.typedefs[p.curToken.Literal]
	}
	return false
}

func (p *Parser) parseBlock() *cabs.Block {
	block := &cabs.Block{Items: []cabs.Stmt{}}

	p.nextToken() // consume '{'

	for !p.curTokenIs(lexer.TokenRBrace) && !p.curTokenIs(lexer.TokenEOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Items = append(block.Items, stmt)
		}
	}

	p.nextToken() // consume '}'

	return block
}

func (p *Parser) parseStatement() cabs.Stmt {
	switch p.curToken.Type {
	case lexer.TokenReturn:
		return p.parseReturnStatement()
	default:
		p.addError(fmt.Sprintf("unexpected token in statement: %s", p.curToken.Type))
		p.nextToken()
		return nil
	}
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

func (p *Parser) parseExpression() cabs.Expr {
	// For now, only parse integer literals
	if p.curTokenIs(lexer.TokenInt) {
		lit := p.curToken.Literal
		p.nextToken()
		var value int64
		fmt.Sscanf(lit, "%d", &value)
		return cabs.Constant{Value: value}
	}

	p.addError(fmt.Sprintf("expected expression, got %s", p.curToken.Type))
	return nil
}

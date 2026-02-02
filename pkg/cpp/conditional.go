// conditional.go implements conditional compilation (#if, #ifdef, etc.)
package cpp

import (
	"fmt"
	"strconv"
	"strings"
)

// ConditionState tracks the state of nested conditional compilation.
type ConditionState struct {
	active    bool // true if current branch is active (included)
	seenElse  bool // true if #else has been seen for this level
	anyActive bool // true if any branch at this level was active
}

// ConditionalProcessor handles conditional compilation directives.
type ConditionalProcessor struct {
	macros   *MacroTable
	expander *Expander
	stack    []ConditionState // stack of nested conditions
}

// NewConditionalProcessor creates a new conditional processor.
func NewConditionalProcessor(macros *MacroTable) *ConditionalProcessor {
	return &ConditionalProcessor{
		macros:   macros,
		expander: NewExpander(macros),
		stack:    []ConditionState{},
	}
}

// IsActive returns true if the current location is active (should be included).
func (cp *ConditionalProcessor) IsActive() bool {
	// If stack is empty, we're at top level and active
	if len(cp.stack) == 0 {
		return true
	}
	// Check all levels - all must be active
	for _, state := range cp.stack {
		if !state.active {
			return false
		}
	}
	return true
}

// ProcessIf handles #if directive.
func (cp *ConditionalProcessor) ProcessIf(expr []Token) error {
	// If we're in an inactive branch, just push inactive state
	if !cp.IsActive() {
		cp.stack = append(cp.stack, ConditionState{active: false, anyActive: false})
		return nil
	}

	// Evaluate the condition
	result, err := cp.evaluateCondition(expr)
	if err != nil {
		return fmt.Errorf("#if: %w", err)
	}

	cp.stack = append(cp.stack, ConditionState{active: result, anyActive: result})
	return nil
}

// ProcessIfdef handles #ifdef directive.
func (cp *ConditionalProcessor) ProcessIfdef(name string) error {
	if !cp.IsActive() {
		cp.stack = append(cp.stack, ConditionState{active: false, anyActive: false})
		return nil
	}

	defined := cp.macros.IsDefined(name)
	cp.stack = append(cp.stack, ConditionState{active: defined, anyActive: defined})
	return nil
}

// ProcessIfndef handles #ifndef directive.
func (cp *ConditionalProcessor) ProcessIfndef(name string) error {
	if !cp.IsActive() {
		cp.stack = append(cp.stack, ConditionState{active: false, anyActive: false})
		return nil
	}

	notDefined := !cp.macros.IsDefined(name)
	cp.stack = append(cp.stack, ConditionState{active: notDefined, anyActive: notDefined})
	return nil
}

// ProcessElif handles #elif directive.
func (cp *ConditionalProcessor) ProcessElif(expr []Token) error {
	if len(cp.stack) == 0 {
		return fmt.Errorf("#elif without matching #if")
	}

	state := &cp.stack[len(cp.stack)-1]
	if state.seenElse {
		return fmt.Errorf("#elif after #else")
	}

	// If any previous branch was active, this branch is inactive
	if state.anyActive {
		state.active = false
		return nil
	}

	// Check parent levels - if any parent is inactive, we're inactive
	parentActive := true
	for i := 0; i < len(cp.stack)-1; i++ {
		if !cp.stack[i].active {
			parentActive = false
			break
		}
	}

	if !parentActive {
		state.active = false
		return nil
	}

	// Evaluate condition
	result, err := cp.evaluateCondition(expr)
	if err != nil {
		return fmt.Errorf("#elif: %w", err)
	}

	state.active = result
	if result {
		state.anyActive = true
	}
	return nil
}

// ProcessElse handles #else directive.
func (cp *ConditionalProcessor) ProcessElse() error {
	if len(cp.stack) == 0 {
		return fmt.Errorf("#else without matching #if")
	}

	state := &cp.stack[len(cp.stack)-1]
	if state.seenElse {
		return fmt.Errorf("duplicate #else")
	}
	state.seenElse = true

	// Check parent levels
	parentActive := true
	for i := 0; i < len(cp.stack)-1; i++ {
		if !cp.stack[i].active {
			parentActive = false
			break
		}
	}

	// #else is active only if parent is active and no previous branch was active
	state.active = parentActive && !state.anyActive
	if state.active {
		state.anyActive = true
	}
	return nil
}

// ProcessEndif handles #endif directive.
func (cp *ConditionalProcessor) ProcessEndif() error {
	if len(cp.stack) == 0 {
		return fmt.Errorf("#endif without matching #if")
	}
	cp.stack = cp.stack[:len(cp.stack)-1]
	return nil
}

// Depth returns the nesting depth of conditionals.
func (cp *ConditionalProcessor) Depth() int {
	return len(cp.stack)
}

// CheckBalanced returns an error if there are unclosed conditionals.
func (cp *ConditionalProcessor) CheckBalanced() error {
	if len(cp.stack) > 0 {
		return fmt.Errorf("unterminated conditional directive, %d level(s) unclosed", len(cp.stack))
	}
	return nil
}

// evaluateCondition evaluates a preprocessor constant expression.
func (cp *ConditionalProcessor) evaluateCondition(tokens []Token) (bool, error) {
	// First, handle 'defined' operator and expand macros
	processed, err := cp.processDefinedAndExpand(tokens)
	if err != nil {
		return false, err
	}

	// Parse and evaluate the expression
	result, err := cp.evaluateExpr(processed)
	if err != nil {
		return false, err
	}

	return result != 0, nil
}

// processDefinedAndExpand handles the 'defined' operator and expands macros.
func (cp *ConditionalProcessor) processDefinedAndExpand(tokens []Token) ([]Token, error) {
	var result []Token
	i := 0

	for i < len(tokens) {
		tok := tokens[i]

		// Skip whitespace
		if tok.Type == PP_WHITESPACE {
			i++
			continue
		}

		// Handle 'defined' operator
		if tok.Type == PP_IDENTIFIER && tok.Text == "defined" {
			i++

			// Skip whitespace
			for i < len(tokens) && tokens[i].Type == PP_WHITESPACE {
				i++
			}

			if i >= len(tokens) {
				return nil, fmt.Errorf("defined operator requires an identifier")
			}

			var name string
			if tokens[i].Type == PP_PUNCTUATOR && tokens[i].Text == "(" {
				// defined(NAME) form
				i++
				for i < len(tokens) && tokens[i].Type == PP_WHITESPACE {
					i++
				}
				if i >= len(tokens) || tokens[i].Type != PP_IDENTIFIER {
					return nil, fmt.Errorf("defined() requires an identifier")
				}
				name = tokens[i].Text
				i++
				for i < len(tokens) && tokens[i].Type == PP_WHITESPACE {
					i++
				}
				if i >= len(tokens) || tokens[i].Type != PP_PUNCTUATOR || tokens[i].Text != ")" {
					return nil, fmt.Errorf("missing ) in defined()")
				}
				i++
			} else if tokens[i].Type == PP_IDENTIFIER {
				// defined NAME form
				name = tokens[i].Text
				i++
			} else {
				return nil, fmt.Errorf("defined operator requires an identifier")
			}

			// Replace with 1 or 0
			value := "0"
			if cp.macros.IsDefined(name) {
				value = "1"
			}
			result = append(result, Token{Type: PP_NUMBER, Text: value, Loc: tok.Loc})
			continue
		}

		result = append(result, tok)
		i++
	}

	// Now expand any remaining macros
	expanded, err := cp.expander.Expand(result)
	if err != nil {
		return nil, err
	}

	// Replace any remaining identifiers with 0 (undefined macros evaluate to 0)
	var final []Token
	for _, tok := range expanded {
		if tok.Type == PP_IDENTIFIER {
			final = append(final, Token{Type: PP_NUMBER, Text: "0", Loc: tok.Loc})
		} else {
			final = append(final, tok)
		}
	}

	return final, nil
}

// evaluateExpr evaluates a constant expression from tokens.
func (cp *ConditionalProcessor) evaluateExpr(tokens []Token) (int64, error) {
	// Filter out whitespace
	var filtered []Token
	for _, tok := range tokens {
		if tok.Type != PP_WHITESPACE && tok.Type != PP_NEWLINE {
			filtered = append(filtered, tok)
		}
	}

	if len(filtered) == 0 {
		return 0, fmt.Errorf("empty expression")
	}

	p := &exprParser{tokens: filtered, pos: 0}
	result, err := p.parseConditional()
	if err != nil {
		return 0, err
	}

	if p.pos < len(p.tokens) {
		return 0, fmt.Errorf("unexpected token after expression: %s", p.tokens[p.pos].Text)
	}

	return result, nil
}

// exprParser parses and evaluates preprocessor constant expressions.
type exprParser struct {
	tokens []Token
	pos    int
}

func (p *exprParser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: PP_EOF}
	}
	return p.tokens[p.pos]
}

func (p *exprParser) advance() Token {
	tok := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *exprParser) match(text string) bool {
	if p.peek().Type == PP_PUNCTUATOR && p.peek().Text == text {
		p.advance()
		return true
	}
	return false
}

// Precedence: conditional -> logicalOr -> logicalAnd -> bitwiseOr -> bitwiseXor -> bitwiseAnd
//             -> equality -> relational -> shift -> additive -> multiplicative -> unary -> primary

func (p *exprParser) parseConditional() (int64, error) {
	cond, err := p.parseLogicalOr()
	if err != nil {
		return 0, err
	}

	if p.match("?") {
		thenVal, err := p.parseConditional()
		if err != nil {
			return 0, err
		}
		if !p.match(":") {
			return 0, fmt.Errorf("expected ':' in conditional expression")
		}
		elseVal, err := p.parseConditional()
		if err != nil {
			return 0, err
		}
		if cond != 0 {
			return thenVal, nil
		}
		return elseVal, nil
	}

	return cond, nil
}

func (p *exprParser) parseLogicalOr() (int64, error) {
	left, err := p.parseLogicalAnd()
	if err != nil {
		return 0, err
	}

	for p.match("||") {
		right, err := p.parseLogicalAnd()
		if err != nil {
			return 0, err
		}
		if left != 0 || right != 0 {
			left = 1
		} else {
			left = 0
		}
	}

	return left, nil
}

func (p *exprParser) parseLogicalAnd() (int64, error) {
	left, err := p.parseBitwiseOr()
	if err != nil {
		return 0, err
	}

	for p.match("&&") {
		right, err := p.parseBitwiseOr()
		if err != nil {
			return 0, err
		}
		if left != 0 && right != 0 {
			left = 1
		} else {
			left = 0
		}
	}

	return left, nil
}

func (p *exprParser) parseBitwiseOr() (int64, error) {
	left, err := p.parseBitwiseXor()
	if err != nil {
		return 0, err
	}

	for p.peek().Type == PP_PUNCTUATOR && p.peek().Text == "|" {
		p.advance()
		right, err := p.parseBitwiseXor()
		if err != nil {
			return 0, err
		}
		left = left | right
	}

	return left, nil
}

func (p *exprParser) parseBitwiseXor() (int64, error) {
	left, err := p.parseBitwiseAnd()
	if err != nil {
		return 0, err
	}

	for p.peek().Type == PP_PUNCTUATOR && p.peek().Text == "^" {
		p.advance()
		right, err := p.parseBitwiseAnd()
		if err != nil {
			return 0, err
		}
		left = left ^ right
	}

	return left, nil
}

func (p *exprParser) parseBitwiseAnd() (int64, error) {
	left, err := p.parseEquality()
	if err != nil {
		return 0, err
	}

	for p.peek().Type == PP_PUNCTUATOR && p.peek().Text == "&" {
		p.advance()
		right, err := p.parseEquality()
		if err != nil {
			return 0, err
		}
		left = left & right
	}

	return left, nil
}

func (p *exprParser) parseEquality() (int64, error) {
	left, err := p.parseRelational()
	if err != nil {
		return 0, err
	}

	for {
		if p.match("==") {
			right, err := p.parseRelational()
			if err != nil {
				return 0, err
			}
			if left == right {
				left = 1
			} else {
				left = 0
			}
		} else if p.match("!=") {
			right, err := p.parseRelational()
			if err != nil {
				return 0, err
			}
			if left != right {
				left = 1
			} else {
				left = 0
			}
		} else {
			break
		}
	}

	return left, nil
}

func (p *exprParser) parseRelational() (int64, error) {
	left, err := p.parseShift()
	if err != nil {
		return 0, err
	}

	for {
		if p.match("<=") {
			right, err := p.parseShift()
			if err != nil {
				return 0, err
			}
			if left <= right {
				left = 1
			} else {
				left = 0
			}
		} else if p.match(">=") {
			right, err := p.parseShift()
			if err != nil {
				return 0, err
			}
			if left >= right {
				left = 1
			} else {
				left = 0
			}
		} else if p.peek().Type == PP_PUNCTUATOR && p.peek().Text == "<" {
			p.advance()
			right, err := p.parseShift()
			if err != nil {
				return 0, err
			}
			if left < right {
				left = 1
			} else {
				left = 0
			}
		} else if p.peek().Type == PP_PUNCTUATOR && p.peek().Text == ">" {
			p.advance()
			right, err := p.parseShift()
			if err != nil {
				return 0, err
			}
			if left > right {
				left = 1
			} else {
				left = 0
			}
		} else {
			break
		}
	}

	return left, nil
}

func (p *exprParser) parseShift() (int64, error) {
	left, err := p.parseAdditive()
	if err != nil {
		return 0, err
	}

	for {
		if p.match("<<") {
			right, err := p.parseAdditive()
			if err != nil {
				return 0, err
			}
			left = left << uint(right)
		} else if p.match(">>") {
			right, err := p.parseAdditive()
			if err != nil {
				return 0, err
			}
			left = left >> uint(right)
		} else {
			break
		}
	}

	return left, nil
}

func (p *exprParser) parseAdditive() (int64, error) {
	left, err := p.parseMultiplicative()
	if err != nil {
		return 0, err
	}

	for {
		if p.peek().Type == PP_PUNCTUATOR && p.peek().Text == "+" {
			p.advance()
			right, err := p.parseMultiplicative()
			if err != nil {
				return 0, err
			}
			left = left + right
		} else if p.peek().Type == PP_PUNCTUATOR && p.peek().Text == "-" {
			p.advance()
			right, err := p.parseMultiplicative()
			if err != nil {
				return 0, err
			}
			left = left - right
		} else {
			break
		}
	}

	return left, nil
}

func (p *exprParser) parseMultiplicative() (int64, error) {
	left, err := p.parseUnary()
	if err != nil {
		return 0, err
	}

	for {
		if p.peek().Type == PP_PUNCTUATOR && p.peek().Text == "*" {
			p.advance()
			right, err := p.parseUnary()
			if err != nil {
				return 0, err
			}
			left = left * right
		} else if p.peek().Type == PP_PUNCTUATOR && p.peek().Text == "/" {
			p.advance()
			right, err := p.parseUnary()
			if err != nil {
				return 0, err
			}
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			left = left / right
		} else if p.peek().Type == PP_PUNCTUATOR && p.peek().Text == "%" {
			p.advance()
			right, err := p.parseUnary()
			if err != nil {
				return 0, err
			}
			if right == 0 {
				return 0, fmt.Errorf("modulo by zero")
			}
			left = left % right
		} else {
			break
		}
	}

	return left, nil
}

func (p *exprParser) parseUnary() (int64, error) {
	if p.peek().Type == PP_PUNCTUATOR {
		switch p.peek().Text {
		case "!":
			p.advance()
			val, err := p.parseUnary()
			if err != nil {
				return 0, err
			}
			if val == 0 {
				return 1, nil
			}
			return 0, nil
		case "-":
			p.advance()
			val, err := p.parseUnary()
			if err != nil {
				return 0, err
			}
			return -val, nil
		case "+":
			p.advance()
			return p.parseUnary()
		case "~":
			p.advance()
			val, err := p.parseUnary()
			if err != nil {
				return 0, err
			}
			return ^val, nil
		}
	}

	return p.parsePrimary()
}

func (p *exprParser) parsePrimary() (int64, error) {
	tok := p.peek()

	// Parenthesized expression
	if tok.Type == PP_PUNCTUATOR && tok.Text == "(" {
		p.advance()
		val, err := p.parseConditional()
		if err != nil {
			return 0, err
		}
		if !p.match(")") {
			return 0, fmt.Errorf("expected ')'")
		}
		return val, nil
	}

	// Number
	if tok.Type == PP_NUMBER {
		p.advance()
		return parseNumber(tok.Text)
	}

	// Character constant
	if tok.Type == PP_CHAR_CONST {
		p.advance()
		return parseCharConst(tok.Text)
	}

	return 0, fmt.Errorf("unexpected token in expression: %s (%v)", tok.Text, tok.Type)
}

// parseNumber parses an integer constant from a string.
func parseNumber(s string) (int64, error) {
	// Remove any suffix (L, U, LL, etc.)
	s = strings.TrimRight(s, "lLuU")

	// Handle hex, octal, binary
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		val, err := strconv.ParseInt(s[2:], 16, 64)
		if err != nil {
			return 0, err
		}
		return val, nil
	}
	if strings.HasPrefix(s, "0b") || strings.HasPrefix(s, "0B") {
		val, err := strconv.ParseInt(s[2:], 2, 64)
		if err != nil {
			return 0, err
		}
		return val, nil
	}
	if strings.HasPrefix(s, "0") && len(s) > 1 && s[1] >= '0' && s[1] <= '7' {
		val, err := strconv.ParseInt(s[1:], 8, 64)
		if err != nil {
			return 0, err
		}
		return val, nil
	}

	// Decimal
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return val, nil
}

// parseCharConst parses a character constant like 'a' or '\n'.
func parseCharConst(s string) (int64, error) {
	// Remove quotes
	if len(s) < 2 || s[0] != '\'' || s[len(s)-1] != '\'' {
		return 0, fmt.Errorf("invalid character constant: %s", s)
	}
	inner := s[1 : len(s)-1]

	if len(inner) == 0 {
		return 0, fmt.Errorf("empty character constant")
	}

	if inner[0] == '\\' {
		// Escape sequence
		if len(inner) < 2 {
			return 0, fmt.Errorf("invalid escape sequence")
		}
		switch inner[1] {
		case 'n':
			return '\n', nil
		case 't':
			return '\t', nil
		case 'r':
			return '\r', nil
		case '\\':
			return '\\', nil
		case '\'':
			return '\'', nil
		case '"':
			return '"', nil
		case '0':
			return 0, nil
		case 'a':
			return '\a', nil
		case 'b':
			return '\b', nil
		case 'f':
			return '\f', nil
		case 'v':
			return '\v', nil
		case 'x':
			// Hex escape
			if len(inner) < 3 {
				return 0, fmt.Errorf("invalid hex escape")
			}
			val, err := strconv.ParseInt(inner[2:], 16, 64)
			if err != nil {
				return 0, err
			}
			return val, nil
		default:
			// Octal escape
			if inner[1] >= '0' && inner[1] <= '7' {
				val, err := strconv.ParseInt(inner[1:], 8, 64)
				if err != nil {
					return 0, err
				}
				return val, nil
			}
			return 0, fmt.Errorf("unknown escape sequence: %s", inner)
		}
	}

	// Simple character
	return int64(inner[0]), nil
}

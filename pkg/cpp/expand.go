// expand.go implements macro expansion including argument substitution,
// stringification, and token pasting.
package cpp

import (
	"fmt"
	"strings"
)

// Expander handles macro expansion.
type Expander struct {
	macros   *MacroTable
	hideset  map[string]bool // macros currently being expanded (blue paint)
	loc      SourceLoc       // current expansion location for __FILE__/__LINE__
}

// NewExpander creates a new macro expander.
func NewExpander(macros *MacroTable) *Expander {
	return &Expander{
		macros:  macros,
		hideset: make(map[string]bool),
	}
}

// Expand expands all macros in the token stream.
func (e *Expander) Expand(tokens []Token) ([]Token, error) {
	return e.expandTokens(tokens, nil)
}

// ExpandWithLoc expands tokens, using the given location for __FILE__/__LINE__.
func (e *Expander) ExpandWithLoc(tokens []Token, loc SourceLoc) ([]Token, error) {
	e.loc = loc
	return e.expandTokens(tokens, nil)
}

// expandTokens expands macros in a token stream.
// parentHideset is inherited hideset for nested expansion.
func (e *Expander) expandTokens(tokens []Token, parentHideset map[string]bool) ([]Token, error) {
	var result []Token
	i := 0

	for i < len(tokens) {
		tok := tokens[i]

		// Only identifiers can be macros
		if tok.Type != PP_IDENTIFIER {
			result = append(result, tok)
			i++
			continue
		}

		// Check if the identifier is a macro
		macro := e.macros.Lookup(tok.Text)
		if macro == nil {
			result = append(result, tok)
			i++
			continue
		}

		// Check hideset (blue paint) - prevent recursive expansion
		inHideset := e.hideset[tok.Text]
		if parentHideset != nil && parentHideset[tok.Text] {
			inHideset = true
		}
		if inHideset {
			result = append(result, tok)
			i++
			continue
		}

		// Handle built-in macros
		if macro.Kind == MacroBuiltin {
			expanded, err := e.expandBuiltin(macro, tok.Loc)
			if err != nil {
				return nil, err
			}
			result = append(result, expanded...)
			i++
			continue
		}

		// Handle function-like macro
		if macro.Kind == MacroFunction {
			// Look for opening paren (may have whitespace before it)
			parenIdx := i + 1
			for parenIdx < len(tokens) && tokens[parenIdx].Type == PP_WHITESPACE {
				parenIdx++
			}

			if parenIdx >= len(tokens) || tokens[parenIdx].Type != PP_PUNCTUATOR || tokens[parenIdx].Text != "(" {
				// No '(' follows - not a macro invocation
				result = append(result, tok)
				i++
				continue
			}

			// Parse arguments
			args, endIdx, err := e.parseArguments(tokens, parenIdx, macro)
			if err != nil {
				return nil, fmt.Errorf("%s:%d: %w", tok.Loc.File, tok.Loc.Line, err)
			}

			// Expand the macro
			expanded, err := e.expandFunctionMacro(macro, args, tok.Loc)
			if err != nil {
				return nil, err
			}

			result = append(result, expanded...)
			i = endIdx + 1
			continue
		}

		// Handle object-like macro
		expanded, err := e.expandObjectMacro(macro, tok.Loc)
		if err != nil {
			return nil, err
		}
		result = append(result, expanded...)
		i++
	}

	return result, nil
}

// expandBuiltin expands a built-in macro.
func (e *Expander) expandBuiltin(macro *Macro, loc SourceLoc) ([]Token, error) {
	// Use the current location context
	useLoc := loc
	if e.loc.File != "" {
		useLoc = e.loc
	}

	switch macro.Name {
	case "__FILE__":
		return e.macros.GetFileToken(useLoc), nil
	case "__LINE__":
		return e.macros.GetLineToken(useLoc), nil
	default:
		if macro.BuiltinFunc != nil {
			return macro.BuiltinFunc(useLoc), nil
		}
		return nil, fmt.Errorf("built-in macro %s has no implementation", macro.Name)
	}
}

// expandObjectMacro expands an object-like macro.
func (e *Expander) expandObjectMacro(macro *Macro, loc SourceLoc) ([]Token, error) {
	// Add to hideset
	e.hideset[macro.Name] = true
	defer delete(e.hideset, macro.Name)

	// Copy replacement tokens with new location
	replacement := make([]Token, len(macro.Replacement))
	for i, tok := range macro.Replacement {
		replacement[i] = tok
		replacement[i].Loc = loc
	}

	// Handle token pasting
	replacement, err := e.handleTokenPasting(replacement)
	if err != nil {
		return nil, err
	}

	// Recursively expand the result
	return e.expandTokens(replacement, e.hideset)
}

// expandFunctionMacro expands a function-like macro with given arguments.
func (e *Expander) expandFunctionMacro(macro *Macro, args [][]Token, loc SourceLoc) ([]Token, error) {
	// Add to hideset
	e.hideset[macro.Name] = true
	defer delete(e.hideset, macro.Name)

	// Build parameter map
	paramMap := make(map[string][]Token)
	for i, param := range macro.Params {
		if i < len(args) {
			paramMap[param] = args[i]
		} else {
			paramMap[param] = nil
		}
	}

	// Handle variadic __VA_ARGS__
	if macro.IsVariadic {
		vaArgs := e.buildVAArgs(args, len(macro.Params))
		paramMap["__VA_ARGS__"] = vaArgs
	}

	// Substitute parameters in replacement list
	var result []Token
	i := 0
	replacement := macro.Replacement

	for i < len(replacement) {
		tok := replacement[i]

		// Handle stringification: # followed by parameter
		if (tok.Type == PP_PUNCTUATOR && tok.Text == "#") || tok.Type == PP_HASH {
			// Skip whitespace after #
			nextIdx := i + 1
			for nextIdx < len(replacement) && replacement[nextIdx].Type == PP_WHITESPACE {
				nextIdx++
			}
			if nextIdx < len(replacement) && replacement[nextIdx].Type == PP_IDENTIFIER {
				paramName := replacement[nextIdx].Text
				if paramTokens, ok := paramMap[paramName]; ok {
					stringified := e.stringify(paramTokens, loc)
					result = append(result, stringified)
					i = nextIdx + 1
					continue
				}
			}
		}

		// Handle parameter substitution
		if tok.Type == PP_IDENTIFIER {
			if paramTokens, ok := paramMap[tok.Text]; ok {
				// Check if adjacent to ## - don't expand if so
				beforePaste := i > 0 && isPasteOp(replacement[i-1])
				afterPaste := i+1 < len(replacement) && isPasteOp(replacement[i+1])

				if beforePaste || afterPaste {
					// Don't expand, just substitute
					for _, pt := range paramTokens {
						pt.Loc = loc
						result = append(result, pt)
					}
				} else {
					// Expand arguments before substitution
					expanded, err := e.expandTokens(paramTokens, e.hideset)
					if err != nil {
						return nil, err
					}
					for _, pt := range expanded {
						pt.Loc = loc
						result = append(result, pt)
					}
				}
				i++
				continue
			}
		}

		// Copy token as-is
		newTok := tok
		newTok.Loc = loc
		result = append(result, newTok)
		i++
	}

	// Handle token pasting
	result, err := e.handleTokenPasting(result)
	if err != nil {
		return nil, err
	}

	// Recursively expand the result
	return e.expandTokens(result, e.hideset)
}

// parseArguments parses the arguments to a function-like macro invocation.
// Returns the list of argument token lists and the index of the closing paren.
func (e *Expander) parseArguments(tokens []Token, startIdx int, macro *Macro) ([][]Token, int, error) {
	// startIdx points to '('
	i := startIdx + 1
	var args [][]Token
	var currentArg []Token
	parenDepth := 1

	for i < len(tokens) {
		tok := tokens[i]

		if tok.Type == PP_PUNCTUATOR {
			switch tok.Text {
			case "(":
				parenDepth++
				currentArg = append(currentArg, tok)
			case ")":
				parenDepth--
				if parenDepth == 0 {
					// End of arguments
					// Only add argument if we have content or we had a comma
					if len(currentArg) > 0 || len(args) > 0 {
						args = append(args, trimWhitespace(currentArg))
					}
					// Validate argument count
					if err := e.validateArgCount(macro, args); err != nil {
						return nil, 0, err
					}
					return args, i, nil
				}
				currentArg = append(currentArg, tok)
			case ",":
				if parenDepth == 1 {
					// Argument separator
					args = append(args, trimWhitespace(currentArg))
					currentArg = nil
				} else {
					currentArg = append(currentArg, tok)
				}
			default:
				currentArg = append(currentArg, tok)
			}
		} else {
			currentArg = append(currentArg, tok)
		}
		i++
	}

	return nil, 0, fmt.Errorf("unterminated macro argument list")
}

// validateArgCount checks if the number of arguments is valid for the macro.
func (e *Expander) validateArgCount(macro *Macro, args [][]Token) error {
	expected := len(macro.Params)

	if macro.IsVariadic {
		// Variadic: at least (params - 1) args required
		if len(args) < expected {
			return fmt.Errorf("macro %s requires at least %d arguments, got %d",
				macro.Name, expected, len(args))
		}
	} else {
		// Fixed: exact match required
		if len(args) != expected {
			return fmt.Errorf("macro %s requires %d arguments, got %d",
				macro.Name, expected, len(args))
		}
	}
	return nil
}

// buildVAArgs builds the __VA_ARGS__ replacement from extra arguments.
func (e *Expander) buildVAArgs(args [][]Token, numParams int) []Token {
	if len(args) <= numParams {
		return nil
	}

	var result []Token
	extraArgs := args[numParams:]
	for i, arg := range extraArgs {
		if i > 0 {
			result = append(result, Token{Type: PP_PUNCTUATOR, Text: ","})
			result = append(result, Token{Type: PP_WHITESPACE, Text: " "})
		}
		result = append(result, arg...)
	}
	return result
}

// stringify converts tokens to a string literal (the # operator).
func (e *Expander) stringify(tokens []Token, loc SourceLoc) Token {
	var sb strings.Builder
	sb.WriteByte('"')

	// Normalize whitespace: sequences of whitespace become single space
	lastWasSpace := true // Start true to skip leading space
	for _, tok := range tokens {
		if tok.Type == PP_WHITESPACE || tok.Type == PP_NEWLINE {
			if !lastWasSpace {
				sb.WriteByte(' ')
				lastWasSpace = true
			}
			continue
		}
		lastWasSpace = false

		// Escape special characters in strings and char constants
		if tok.Type == PP_STRING || tok.Type == PP_CHAR_CONST {
			for _, c := range tok.Text {
				if c == '"' || c == '\\' {
					sb.WriteByte('\\')
				}
				sb.WriteRune(c)
			}
		} else {
			sb.WriteString(tok.Text)
		}
	}

	// Trim trailing space
	str := sb.String()
	if strings.HasSuffix(str, " ") {
		str = str[:len(str)-1]
	}
	str += "\""

	return Token{Type: PP_STRING, Text: str, Loc: loc}
}

// handleTokenPasting handles the ## operator.
func (e *Expander) handleTokenPasting(tokens []Token) ([]Token, error) {
	var result []Token
	i := 0

	for i < len(tokens) {
		tok := tokens[i]

		// Look for ##
		if tok.Type == PP_HASHHASH {
			// Paste previous token with next token
			if len(result) == 0 {
				return nil, fmt.Errorf("## cannot appear at start of replacement list")
			}
			if i+1 >= len(tokens) {
				return nil, fmt.Errorf("## cannot appear at end of replacement list")
			}

			// Skip whitespace after ##
			nextIdx := i + 1
			for nextIdx < len(tokens) && tokens[nextIdx].Type == PP_WHITESPACE {
				nextIdx++
			}
			if nextIdx >= len(tokens) {
				return nil, fmt.Errorf("## cannot appear at end of replacement list")
			}

			// Get the tokens to paste
			leftTok := result[len(result)-1]
			rightTok := tokens[nextIdx]

			// Remove left token from result (will be replaced with pasted)
			result = result[:len(result)-1]

			// Handle placeholder tokens (empty)
			if leftTok.Type == PP_PLACEHOLDER {
				result = append(result, rightTok)
				i = nextIdx + 1
				continue
			}
			if rightTok.Type == PP_PLACEHOLDER {
				result = append(result, leftTok)
				i = nextIdx + 1
				continue
			}

			// Concatenate the token texts
			pastedText := leftTok.Text + rightTok.Text

			// Re-tokenize the result
			pastedTokens := retokenize(pastedText, leftTok.Loc)
			if len(pastedTokens) == 0 {
				// Empty result is a placeholder
				result = append(result, Token{Type: PP_PLACEHOLDER, Text: "", Loc: leftTok.Loc})
			} else {
				result = append(result, pastedTokens...)
			}

			i = nextIdx + 1
			continue
		}

		result = append(result, tok)
		i++
	}

	// Filter out placeholders and whitespace tokens adjacent to ##
	var filtered []Token
	for _, tok := range result {
		if tok.Type != PP_PLACEHOLDER {
			filtered = append(filtered, tok)
		}
	}

	return filtered, nil
}

// retokenize tokenizes a pasted string.
func retokenize(text string, loc SourceLoc) []Token {
	if text == "" {
		return nil
	}

	lex := NewLexer(text, loc.File)
	var tokens []Token
	for {
		tok := lex.NextToken()
		if tok.Type == PP_EOF || tok.Type == PP_NEWLINE {
			break
		}
		if tok.Type != PP_WHITESPACE {
			tok.Loc = loc
			tokens = append(tokens, tok)
		}
	}
	return tokens
}

// isPasteOp checks if a token is the ## operator or whitespace before/after ##.
func isPasteOp(tok Token) bool {
	return tok.Type == PP_HASHHASH
}

// trimWhitespace removes leading and trailing whitespace from a token slice.
func trimWhitespace(tokens []Token) []Token {
	// Trim leading
	start := 0
	for start < len(tokens) && tokens[start].Type == PP_WHITESPACE {
		start++
	}
	// Trim trailing
	end := len(tokens)
	for end > start && tokens[end-1].Type == PP_WHITESPACE {
		end--
	}
	if start >= end {
		return nil
	}
	return tokens[start:end]
}

// ExpandString is a convenience function to expand macros in a string.
func (e *Expander) ExpandString(input string) (string, error) {
	lex := NewLexer(input, "<string>")
	tokens := lex.AllTokens()

	// Remove EOF
	if len(tokens) > 0 && tokens[len(tokens)-1].Type == PP_EOF {
		tokens = tokens[:len(tokens)-1]
	}

	expanded, err := e.Expand(tokens)
	if err != nil {
		return "", err
	}

	return TokensToString(expanded), nil
}
